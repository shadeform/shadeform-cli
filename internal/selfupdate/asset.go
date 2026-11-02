package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
)

// ChecksumsFile is the GoReleaser checksum manifest attached to every release.
const ChecksumsFile = "checksums.txt"

// BinaryName returns the executable's file name inside an archive for goos.
func BinaryName(goos string) string {
	if goos == "windows" {
		return "shade.exe"
	}
	return "shade"
}

// AssetName returns the release archive for a platform, following the
// name_template in .goreleaser.yaml: shade_<Os>_<arch>.tar.gz, where Os is
// title-cased, amd64 is spelled x86_64, and Windows ships a zip.
func AssetName(goos, goarch string) (string, error) {
	var os string
	switch goos {
	case "darwin":
		os = "Darwin"
	case "linux":
		os = "Linux"
	case "windows":
		os = "Windows"
	default:
		return "", fmt.Errorf("no release archive is published for %s/%s", goos, goarch)
	}
	var arch string
	switch goarch {
	case "amd64":
		arch = "x86_64"
	case "arm64":
		arch = "arm64"
	default:
		return "", fmt.Errorf("no release archive is published for %s/%s", goos, goarch)
	}
	ext := ".tar.gz"
	if goos == "windows" {
		ext = ".zip"
	}
	return "shade_" + os + "_" + arch + ext, nil
}

// ParseChecksums reads a GoReleaser checksums.txt ("<sha256>  <file>" per
// line) into a file name to hex digest map.
func ParseChecksums(data []byte) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := strings.TrimPrefix(fields[len(fields)-1], "*")
		out[name] = strings.ToLower(fields[0])
	}
	return out
}

// ErrChecksumMismatch is wrapped by VerifySHA256 when the digest differs.
var ErrChecksumMismatch = errors.New("checksum mismatch")

// VerifySHA256 compares data against a hex digest.
func VerifySHA256(data []byte, wantHex string) error {
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if !strings.EqualFold(got, strings.TrimSpace(wantHex)) {
		return fmt.Errorf("%w: expected %s, downloaded file is %s", ErrChecksumMismatch, wantHex, got)
	}
	return nil
}

// ExtractBinary pulls the named file out of a .tar.gz or .zip archive.
func ExtractBinary(archiveName string, archive []byte, binary string) ([]byte, error) {
	switch {
	case strings.HasSuffix(archiveName, ".zip"):
		return extractZip(archive, binary)
	case strings.HasSuffix(archiveName, ".tar.gz"), strings.HasSuffix(archiveName, ".tgz"):
		return extractTarGz(archive, binary)
	}
	return nil, fmt.Errorf("unsupported archive %q", archiveName)
}

func extractTarGz(archive []byte, binary string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, fmt.Errorf("reading archive: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading archive: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg || path.Base(hdr.Name) != binary {
			continue
		}
		data, err := io.ReadAll(io.LimitReader(tr, maxDownload+1))
		if err != nil {
			return nil, fmt.Errorf("reading %s from archive: %w", binary, err)
		}
		if len(data) > maxDownload {
			return nil, fmt.Errorf("%s in archive exceeds %d bytes", binary, maxDownload)
		}
		return data, nil
	}
	return nil, fmt.Errorf("archive does not contain %s", binary)
}

func extractZip(archive []byte, binary string) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, fmt.Errorf("reading archive: %w", err)
	}
	for _, f := range zr.File {
		if f.FileInfo().IsDir() || path.Base(f.Name) != binary {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("reading %s from archive: %w", binary, err)
		}
		defer rc.Close()
		data, err := io.ReadAll(io.LimitReader(rc, maxDownload+1))
		if err != nil {
			return nil, fmt.Errorf("reading %s from archive: %w", binary, err)
		}
		if len(data) > maxDownload {
			return nil, fmt.Errorf("%s in archive exceeds %d bytes", binary, maxDownload)
		}
		return data, nil
	}
	return nil, fmt.Errorf("archive does not contain %s", binary)
}
