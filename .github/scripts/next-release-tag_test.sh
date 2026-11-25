#!/usr/bin/env bash
#
# Tests next-release-tag.sh against throwaway git repositories. Nothing here
# touches the real repository or the network.
#
# Usage: .github/scripts/next-release-tag_test.sh

set -euo pipefail

script="$(cd "$(dirname "$0")" && pwd)/next-release-tag.sh"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

# Keep the developer's and the runner's git configuration out of the tests.
export GIT_CONFIG_GLOBAL=/dev/null
export GIT_CONFIG_NOSYSTEM=1
export GIT_AUTHOR_NAME=test GIT_AUTHOR_EMAIL=test@example.com
export GIT_COMMITTER_NAME=test GIT_COMMITTER_EMAIL=test@example.com

failures=0
count=0

# new_repo NAME GENERATED: a repository with one commit whose version.go
# holds GENERATED. Leaves the shell inside it.
new_repo() {
  cd "$work"
  rm -rf "$1"
  git init -q -b main "$1"
  cd "$1"
  mkdir -p internal/cli
  set_generated "$2"
  commit "initial"
}

set_generated() {
  printf 'package cli\n\nvar Version = "%s"\n' "$1" >internal/cli/version.go
}

commit() {
  echo "$1" >>changes.txt
  git add -A
  git commit -q -m "$1"
}

# expect_tag NAME WANT [ARGS...]: the script succeeds and prints exactly WANT.
# An empty WANT means "no release".
expect_tag() {
  local name="$1" want="$2" got status=0
  shift 2
  count=$((count + 1))
  got=$("$script" "$@" 2>"$work/stderr") || status=$?
  if [ "$status" -ne 0 ] || [ "$got" != "$want" ]; then
    failures=$((failures + 1))
    echo "FAIL  $name: want '$want', got '$got' (exit $status)"
    sed 's/^/        /' "$work/stderr"
  else
    echo "ok    $name"
  fi
}

# expect_fail NAME PATTERN [ARGS...]: the script fails, prints no tag, and
# explains itself with a message matching PATTERN.
expect_fail() {
  local name="$1" pattern="$2" got status=0
  shift 2
  count=$((count + 1))
  got=$("$script" "$@" 2>"$work/stderr") || status=$?
  if [ "$status" -eq 0 ] || [ -n "$got" ] || ! grep -q -- "$pattern" "$work/stderr"; then
    failures=$((failures + 1))
    echo "FAIL  $name: want a failure matching '$pattern', got '$got' (exit $status)"
    sed 's/^/        /' "$work/stderr"
  else
    echo "ok    $name"
  fi
}

# --- Automatic versions ----------------------------------------------------

new_repo first 0.1.0
expect_tag "no release yet: the generated version" v0.1.0

new_repo patch 0.3.1
git tag v0.3.1
commit "hand-written change"
expect_tag "hand-written change: next patch" v0.3.2

new_repo regen 0.3.1
git tag v0.3.1
set_generated 0.4.0
commit "regenerate"
expect_tag "regeneration: the generated version" v0.4.0

new_repo regen-major 0.3.1
git tag v0.3.1
set_generated 1.0.0
commit "regenerate"
expect_tag "regeneration to a new major" v1.0.0

new_repo tagged 0.3.1
git tag v0.3.1
expect_tag "commit already released: nothing" ""

new_repo tagged-annotated 0.3.1
git tag -a v0.3.1 -m "shade v0.3.1"
expect_tag "commit already released with an annotated tag: nothing" ""
commit "hand-written change"
expect_tag "annotated tags count as releases" v0.3.2

new_repo behind 0.3.1
git tag v0.3.1
commit "one"
git tag v0.3.2
commit "two"
git tag v0.3.3
commit "three"
expect_tag "generated version behind the latest release: next patch" v0.3.4

new_repo numeric 0.3.1
git tag v0.3.9
commit "one"
git tag v0.3.10
commit "two"
expect_tag "tags are ordered as numbers, not text" v0.3.11

new_repo minor-order 0.3.1
git tag v0.9.0
commit "one"
git tag v0.10.0
commit "two"
expect_tag "minor versions are ordered as numbers" v0.10.1

new_repo prerelease 0.3.1
git tag v0.3.1
commit "one"
git tag v0.4.0-rc.1
expect_tag "a prerelease tag does not mark a commit as released" v0.3.2
commit "two"
expect_tag "a prerelease tag is not the latest release" v0.3.2

new_repo other-tags 0.3.1
git tag v0.3.1
commit "one"
git tag vnext
git tag v1
git tag v01.2.3
git tag nightly
expect_tag "tags that are not vX.Y.Z are ignored" v0.3.2

# --- Requested versions ----------------------------------------------------

new_repo requested 0.3.1
git tag v0.3.1
commit "one"
expect_tag "requested version" v0.4.0 0.4.0
expect_tag "requested version with a leading v" v0.4.0 v0.4.0
expect_tag "requested version equal to the next patch" v0.3.2 0.3.2
expect_fail "requested version that is the latest release" "already exists" 0.3.1
expect_fail "requested version older than the latest release" "not newer than the latest release" 0.2.0
expect_fail "requested version with two parts" "not a release version" 1.2
expect_fail "requested prerelease version" "not a release version" 0.4.0-rc.1
expect_fail "requested version with a leading zero" "not a release version" 0.04.0
expect_fail "requested version that is not a version" "not a release version" latest

new_repo requested-existing 0.3.1
git tag v0.3.1
commit "one"
git tag v0.4.0
commit "two"
expect_fail "requested version whose tag exists on an earlier commit" "already exists" 0.4.0
expect_fail "requested version between two releases" "not newer than the latest release" 0.3.5

new_repo requested-generated 0.5.0
git tag v0.3.1
commit "one"
expect_fail "requested version older than the generated version" "older than the generated version" 0.4.0
expect_tag "requested version equal to the generated version" v0.5.0 0.5.0

new_repo requested-released 0.3.1
git tag v0.3.1
expect_fail "requested version for a commit that is already released" "already released as v0.3.1" 0.4.0

new_repo requested-first 0.1.0
expect_tag "requested version with no release yet" v0.2.0 0.2.0

# --- Refusals --------------------------------------------------------------

new_repo old-commit 0.3.1
git tag v0.3.1
commit "one"
old=$(git rev-parse HEAD)
commit "two"
git tag v0.3.2
git checkout -q "$old"
expect_fail "older commit than the latest release" "not in this commit's history"
expect_fail "older commit than the latest release, requested version" "not in this commit's history" 0.4.0

new_repo no-version 0.3.1
git tag v0.3.1
printf 'package cli\n' >internal/cli/version.go
commit "break version.go"
expect_fail "version.go without a version" "could not read"

new_repo bad-version 0.3.1
git tag v0.3.1
set_generated "0.3"
commit "break version.go"
expect_fail "version.go with a malformed version" "could not read"

new_repo missing-file 0.3.1
git tag v0.3.1
git rm -q internal/cli/version.go
git commit -q -m "remove version.go"
expect_fail "version.go missing" "could not read"

echo
if [ "$failures" -ne 0 ]; then
  echo "$failures of $count checks failed"
  exit 1
fi
echo "all $count checks passed"
