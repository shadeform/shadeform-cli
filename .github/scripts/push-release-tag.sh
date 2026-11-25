#!/usr/bin/env bash
#
# Tags the checked-out commit for release and pushes the tag. tag_release.yaml
# runs it once go vet and go test have passed.
#
# Usage: push-release-tag.sh [version]
#
# The version is picked by next-release-tag.sh, which takes the same optional
# argument. It is picked here, again, after fetching the tags, because another
# run of the workflow may have released something while this one was testing:
#
#   - This commit was released meanwhile: nothing to do.
#   - A newer commit was released meanwhile: next-release-tag.sh refuses, and
#     nothing is pushed. This commit's changes are part of that release.
#   - An older commit was released meanwhile: this commit gets the version
#     after it.
#
# A tag with the picked name can still appear on the remote between the fetch
# and the push. If it points at this commit there is nothing to do. If it
# points anywhere else this fails, because this commit was not released.
#
# Prints the tag it pushed on stdout, or nothing when there was nothing to do.
# The reasoning goes to stderr. The remote is origin unless RELEASE_REMOTE is
# set. The tag is annotated, so git needs user.name and user.email.

set -euo pipefail

here="$(cd "$(dirname "$0")" && pwd)"
remote="${RELEASE_REMOTE:-origin}"
requested="${1:-}"

note() { echo "$*" >&2; }
die() {
  echo "error: $*" >&2
  exit 1
}

# remote_tag_commit TAG: the commit TAG points at on the remote, or nothing
# when the remote has no such tag. An annotated tag is listed twice, once as
# the tag object and once, with ^{} after the name, as the commit behind it.
remote_tag_commit() {
  local listing
  listing=$(git ls-remote --tags "$remote" "refs/tags/$1*") || return 1
  awk -v ref="refs/tags/$1" '
    $2 == ref "^{}" { peeled = $1 }
    $2 == ref { direct = $1 }
    END {
      if (peeled != "") print peeled
      else if (direct != "") print direct
    }' <<<"$listing"
}

git fetch --quiet --tags "$remote" >&2 || die "could not fetch the tags from $remote"

tag=$("$here/next-release-tag.sh" "$requested")
if [ -z "$tag" ]; then
  # next-release-tag.sh has already said why.
  exit 0
fi

head=$(git rev-parse HEAD)

existing=$(remote_tag_commit "$tag") || die "could not read the tags on $remote"
if [ "$existing" = "$head" ]; then
  note "$tag already exists on $remote for this commit; nothing to do."
  exit 0
fi
if [ -n "$existing" ]; then
  die "$tag already exists on $remote at ${existing:0:12}, not at this commit (${head:0:12}); this commit was not released"
fi

git tag -a "$tag" -m "shade $tag"
if ! git push --quiet "$remote" "refs/tags/$tag" >&2; then
  git tag -d "$tag" >/dev/null
  # The usual reason is that the tag appeared on the remote a moment ago.
  existing=$(remote_tag_commit "$tag") || existing=""
  if [ "$existing" = "$head" ]; then
    note "$tag was pushed for this commit by another run; nothing to do."
    exit 0
  fi
  if [ -n "$existing" ]; then
    die "$tag already exists on $remote at ${existing:0:12}, not at this commit (${head:0:12}); this commit was not released"
  fi
  die "could not push $tag to $remote; this commit was not released"
fi

note "Pushed $tag for ${head:0:12}."
echo "$tag"
