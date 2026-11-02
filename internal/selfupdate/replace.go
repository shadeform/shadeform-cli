package selfupdate

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

// PermissionError means the install directory is not writable by this user.
// The command turns it into advice (sudo, or a different install dir) rather
// than a bare failure, and nothing has been touched when it is returned.
type PermissionError struct {
	Path string
	Err  error
}

func (e *PermissionError) Error() string {
	return fmt.Sprintf("%s is not writable: %v", e.Path, e.Err)
}

func (e *PermissionError) Unwrap() error { return e.Err }

// ExecutablePath returns the running binary with symlinks resolved, so the
// file that is replaced is the real one, not a link in a bin directory.
func ExecutablePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return exe, nil
}

// CheckWritable reports a PermissionError up front when the target's
// directory cannot be written, so the command can refuse before it spends
// time and bandwidth downloading something it cannot install.
func CheckWritable(target string) error {
	dir := filepath.Dir(target)
	tmp, err := os.CreateTemp(dir, ".shade-update-probe-*")
	if err != nil {
		if isPermission(err) {
			return &PermissionError{Path: dir, Err: err}
		}
		return fmt.Errorf("creating temporary file in %s: %w", dir, err)
	}
	name := tmp.Name()
	_ = tmp.Close()
	_ = os.Remove(name)
	return nil
}

// Replace installs binary at target atomically: the new file is written next
// to the target and renamed over it, so a crash or a failed write leaves the
// old binary intact. On Windows a running executable cannot be overwritten,
// so the old one is first renamed aside; RemoveLeftover tidies that up on a
// later run.
func Replace(target string, binary []byte) error {
	dir := filepath.Dir(target)
	tmp, err := os.CreateTemp(dir, ".shade-update-*")
	if err != nil {
		if isPermission(err) {
			return &PermissionError{Path: dir, Err: err}
		}
		return fmt.Errorf("creating temporary file in %s: %w", dir, err)
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }

	if _, err := tmp.Write(binary); err != nil {
		tmp.Close()
		cleanup()
		return fmt.Errorf("writing %s: %w", tmpName, err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		cleanup()
		return fmt.Errorf("writing %s: %w", tmpName, err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("writing %s: %w", tmpName, err)
	}
	mode := fs.FileMode(0o755)
	if info, err := os.Stat(target); err == nil {
		mode = info.Mode().Perm() | 0o111
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		cleanup()
		return fmt.Errorf("setting permissions on %s: %w", tmpName, err)
	}

	if runtime.GOOS == "windows" {
		old := leftoverName(target)
		_ = os.Remove(old)
		if err := os.Rename(target, old); err != nil && !errors.Is(err, fs.ErrNotExist) {
			cleanup()
			if isPermission(err) {
				return &PermissionError{Path: target, Err: err}
			}
			return fmt.Errorf("moving the current binary aside: %w", err)
		}
	}
	if err := os.Rename(tmpName, target); err != nil {
		cleanup()
		if isPermission(err) {
			return &PermissionError{Path: target, Err: err}
		}
		return fmt.Errorf("installing %s: %w", target, err)
	}
	return nil
}

// RemoveLeftover deletes the previous binary a Windows update moved aside.
// It is a no-op elsewhere and silent on failure.
func RemoveLeftover(target string) {
	if runtime.GOOS != "windows" || target == "" {
		return
	}
	_ = os.Remove(leftoverName(target))
}

func leftoverName(target string) string { return target + ".old" }

func isPermission(err error) bool {
	return errors.Is(err, fs.ErrPermission)
}
