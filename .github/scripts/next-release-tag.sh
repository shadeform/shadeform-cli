#!/usr/bin/env bash
#
# Prints the release tag the checked-out commit should get, or nothing when
# the commit should not be released. tag_release.yaml runs it on every push to
# main that changes shipped code.
#
# Usage: next-release-tag.sh [version]
#
# Without an argument, the first rule that applies wins:
#
#   1. The commit already has a release tag: print nothing.
#   2. The generated version in internal/cli/version.go is newer than the
#      latest release tag: that version. This is a Speakeasy regeneration,
#      which is how a minor or major bump arrives.
#   3. Otherwise: the latest release tag plus one patch.
#
# With a version argument (0.4.0 or v0.4.0), that version is used instead. It
# must be newer than the latest release and not older than the generated
# version, and the commit must not be released already.
#
# A release tag is vX.Y.Z. Prerelease tags such as v1.0.0-rc.1 are ignored:
# they neither count as the latest release nor mark a commit as released.
#
# The latest release must be in the commit's history. Without that check,
# re-running an old workflow run would publish old code as the newest version.
#
# The tag goes to stdout and the reasoning to stderr. Needs the full history
# and all tags (actions/checkout with fetch-depth: 0).

set -euo pipefail

version_file="${VERSION_FILE:-internal/cli/version.go}"
requested="${1:-}"

number='(0|[1-9][0-9]*)'
version_re="^${number}\\.${number}\\.${number}$"

note() { echo "$*" >&2; }
die() {
  echo "error: $*" >&2
  exit 1
}

# newer A B: A is a strictly newer X.Y.Z than B.
newer() {
  local a b
  local -a x y
  IFS=. read -r -a x <<<"$1"
  IFS=. read -r -a y <<<"$2"
  for i in 0 1 2; do
    a="${x[$i]}"
    b="${y[$i]}"
    if [ "$a" -ne "$b" ]; then
      [ "$a" -gt "$b" ]
      return
    fi
  done
  return 1
}

generated=$(sed -n 's/^var Version = "\(.*\)"$/\1/p' "$version_file" 2>/dev/null || true)
if ! [[ "$generated" =~ $version_re ]]; then
  die "could not read an X.Y.Z version from $version_file (got '${generated}')"
fi

# The newest release tag in the repository, without its leading v.
latest=""
while IFS= read -r tag; do
  candidate="${tag#v}"
  [[ "$candidate" =~ $version_re ]] || continue
  if [ -z "$latest" ] || newer "$candidate" "$latest"; then
    latest="$candidate"
  fi
done < <(git tag --list 'v*')

# The release tag on this commit, if it has one.
released=""
while IFS= read -r tag; do
  if [[ "${tag#v}" =~ $version_re ]]; then
    released="$tag"
    break
  fi
done < <(git tag --points-at HEAD --list 'v*')

require_latest_in_history() {
  [ -n "$latest" ] || return 0
  if ! git merge-base --is-ancestor "v${latest}^{commit}" HEAD; then
    die "v$latest is the latest release and is not in this commit's history; refusing to release an older commit as a newer version"
  fi
}

if [ -n "$requested" ]; then
  wanted="${requested#v}"
  if ! [[ "$wanted" =~ $version_re ]]; then
    die "'$requested' is not a release version; use X.Y.Z, for example 0.4.0"
  fi
  if [ -n "$released" ]; then
    die "this commit is already released as $released"
  fi
  if git rev-parse -q --verify "refs/tags/v$wanted" >/dev/null; then
    die "v$wanted already exists"
  fi
  if [ -n "$latest" ] && ! newer "$wanted" "$latest"; then
    die "v$wanted is not newer than the latest release v$latest"
  fi
  if newer "$generated" "$wanted"; then
    die "v$wanted is older than the generated version $generated in $version_file"
  fi
  require_latest_in_history
  note "Using the requested version v$wanted (latest release: ${latest:+v}${latest:-none})."
  echo "v$wanted"
  exit 0
fi

if [ -n "$released" ]; then
  note "This commit is already released as $released; nothing to do."
  exit 0
fi

require_latest_in_history

if [ -z "$latest" ]; then
  note "No release yet; using the generated version $generated."
  echo "v$generated"
elif newer "$generated" "$latest"; then
  note "The generated version $generated is newer than the latest release v$latest; using it."
  echo "v$generated"
else
  IFS=. read -r major minor patch <<<"$latest"
  next="$major.$minor.$((patch + 1))"
  note "The latest release is v$latest and the generated version is $generated; using the next patch, v$next."
  echo "v$next"
fi
