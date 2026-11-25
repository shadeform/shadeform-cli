#!/usr/bin/env bash
#
# Tests push-release-tag.sh against throwaway git repositories: a bare one
# that plays GitHub and a checkout that plays the workflow run. Nothing here
# touches the real repository or the network.
#
# Usage: .github/scripts/push-release-tag_test.sh

set -euo pipefail

script="$(cd "$(dirname "$0")" && pwd)/push-release-tag.sh"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

# Keep the developer's and the runner's git configuration out of the tests.
export GIT_CONFIG_GLOBAL=/dev/null
export GIT_CONFIG_NOSYSTEM=1
export GIT_AUTHOR_NAME=test GIT_AUTHOR_EMAIL=test@example.com
export GIT_COMMITTER_NAME=test GIT_COMMITTER_EMAIL=test@example.com

failures=0
count=0
remote=""      # the bare repository of the current test
hook_before="" # see before_git

# A stand-in for git that runs a command once, just before one chosen git
# subcommand, and then hands over to the real git. It is how a test lets
# "another run" act between two steps of the script. It is only on the PATH
# for a run that asked for it with before_git.
real_git="$(command -v git)"
real_path="$PATH"
mkdir "$work/shim"
cat >"$work/shim/git" <<'EOF'
#!/usr/bin/env bash
if [ "${1:-}" = "$SHIM_BEFORE" ] && [ -f "$SHIM_HOOK" ]; then
  mv "$SHIM_HOOK" "$SHIM_HOOK.ran"
  PATH="$SHIM_REAL_PATH" bash "$SHIM_HOOK.ran" >&2
fi
exec "$SHIM_REAL_GIT" "$@"
EOF
chmod +x "$work/shim/git"

# before_git SUBCOMMAND COMMAND...: the next run of the script runs COMMAND
# just before its first `git SUBCOMMAND`.
before_git() {
  hook_before="$1"
  shift
  printf '%q ' "$@" >"$work/hook"
}

# new_repo NAME GENERATED: a bare remote and a checkout of it, with one
# commit whose version.go holds GENERATED. Leaves the shell in the checkout.
new_repo() {
  cd "$work"
  remote="$work/$1.git"
  git init -q --bare -b main "$remote"
  git init -q -b main "$1"
  cd "$1"
  git remote add origin "$remote"
  mkdir -p internal/cli
  printf 'package cli\n\nvar Version = "%s"\n' "$2" >internal/cli/version.go
  commit "initial"
}

# commit MESSAGE: commits and pushes to main, like a merge.
commit() {
  echo "$1" >>changes.txt
  git add -A
  git commit -q -m "$1"
  git push -q origin HEAD:main
}

# release TAG: a release that both the remote and the checkout know about.
release() {
  git tag -a "$1" -m "shade $1"
  git push -q origin "refs/tags/$1"
}

# other_commit MESSAGE: someone else merges to main. The checkout does not
# have the commit until it fetches. Prints the commit.
other_commit() {
  local dir="$remote.other"
  git clone -q "$remote" "$dir"
  (
    cd "$dir"
    echo "$1" >>changes.txt
    git add -A
    git commit -q -m "$1"
    git push -q origin HEAD:main
    git rev-parse HEAD
  )
}

# run_script [ARGS...]: runs the script, setting got, status and hook_missed.
run_script() {
  status=0
  hook_missed=""
  if [ -n "$hook_before" ]; then
    got=$(SHIM_BEFORE="$hook_before" SHIM_HOOK="$work/hook" SHIM_REAL_GIT="$real_git" SHIM_REAL_PATH="$real_path" \
      PATH="$work/shim:$real_path" "$script" "$@" 2>"$work/stderr") || status=$?
    if [ -e "$work/hook.ran" ]; then
      rm -f "$work/hook.ran"
    else
      hook_missed="the script never ran git $hook_before"
      rm -f "$work/hook"
    fi
    hook_before=""
  else
    got=$("$script" "$@" 2>"$work/stderr") || status=$?
  fi
}

pass() { echo "ok    $1"; }
fail() {
  failures=$((failures + 1))
  echo "FAIL  $1: $2"
  if [ -s "$work/stderr" ]; then
    sed 's/^/        /' "$work/stderr"
  fi
}

# expect_push NAME WANT [ARGS...]: the script succeeds and prints exactly
# WANT. A non-empty WANT must then be an annotated tag on the remote that
# points at the checked-out commit. An empty WANT means "nothing to do".
expect_push() {
  local name="$1" want="$2" at kind
  shift 2
  count=$((count + 1))
  run_script "$@"
  if [ -n "$hook_missed" ]; then
    fail "$name" "$hook_missed"
  elif [ "$status" -ne 0 ] || [ "$got" != "$want" ]; then
    fail "$name" "want '$want', got '$got' (exit $status)"
  elif [ -n "$want" ]; then
    at=$(git -C "$remote" rev-parse -q --verify "refs/tags/$want^{commit}" || true)
    kind=$(git -C "$remote" cat-file -t "refs/tags/$want" 2>/dev/null || true)
    if [ "$at" != "$(git rev-parse HEAD)" ]; then
      fail "$name" "$want on the remote is at '$at', not at the checked-out commit"
    elif [ "$kind" != "tag" ]; then
      fail "$name" "$want on the remote is not an annotated tag"
    else
      pass "$name"
    fi
  else
    pass "$name"
  fi
}

# expect_fail NAME PATTERN [ARGS...]: the script fails, prints no tag, and
# explains itself with a message matching PATTERN.
expect_fail() {
  local name="$1" pattern="$2"
  shift 2
  count=$((count + 1))
  run_script "$@"
  if [ -n "$hook_missed" ]; then
    fail "$name" "$hook_missed"
  elif [ "$status" -eq 0 ] || [ -n "$got" ] || ! grep -q -- "$pattern" "$work/stderr"; then
    fail "$name" "want a failure matching '$pattern', got '$got' (exit $status)"
  else
    pass "$name"
  fi
}

# expect_remote_tags NAME "TAG TAG...": exactly these tags are on the remote.
expect_remote_tags() {
  local have
  count=$((count + 1))
  : >"$work/stderr"
  have=$(git -C "$remote" tag --list --sort=v:refname | paste -sd' ' -)
  if [ "$have" != "$2" ]; then
    fail "$1" "want the remote tags '$2', got '$have'"
  else
    pass "$1"
  fi
}

# expect_remote_tag_at NAME TAG COMMIT: TAG on the remote points at COMMIT.
expect_remote_tag_at() {
  local at
  count=$((count + 1))
  : >"$work/stderr"
  at=$(git -C "$remote" rev-parse -q --verify "refs/tags/$2^{commit}" || true)
  if [ "$at" != "$3" ]; then
    fail "$1" "want $2 on the remote at '$3', got '$at'"
  else
    pass "$1"
  fi
}

# expect_no_local_tag NAME TAG: the checkout was not left with TAG.
expect_no_local_tag() {
  count=$((count + 1))
  : >"$work/stderr"
  if git rev-parse -q --verify "refs/tags/$2" >/dev/null; then
    fail "$1" "the checkout still has $2"
  else
    pass "$1"
  fi
}

# --- Nothing in the way ----------------------------------------------------

new_repo normal 0.3.1
release v0.3.1
commit "hand-written change"
expect_push "pushes the next patch" v0.3.2
expect_remote_tags "the remote has the new tag" "v0.3.1 v0.3.2"
expect_push "a second run on the same commit does nothing" ""
expect_remote_tags "a second run adds no tag" "v0.3.1 v0.3.2"

new_repo first 0.1.0
expect_push "pushes the first release" v0.1.0

new_repo requested 0.3.1
release v0.3.1
commit "one"
expect_push "pushes a requested version" v0.4.0 0.4.0

new_repo other-remote 0.3.1
release v0.3.1
commit "one"
git remote rename origin upstream
RELEASE_REMOTE=upstream expect_push "pushes to the remote named in RELEASE_REMOTE" v0.3.2

# --- Another run released something while this one was testing --------------
# Only the remote has the other run's tag. The checkout learns about it from
# the fetch.

new_repo same-commit 0.3.1
release v0.3.1
commit "one"
git -C "$remote" tag -a -m "shade v0.3.2" v0.3.2 "$(git rev-parse HEAD)"
expect_push "this commit was released meanwhile: nothing" ""
expect_remote_tags "this commit was released meanwhile: no second tag" "v0.3.1 v0.3.2"

new_repo older-commit 0.3.1
release v0.3.1
commit "one"
one=$(git rev-parse HEAD)
commit "two"
git -C "$remote" tag -a -m "shade v0.3.2" v0.3.2 "$one"
expect_push "an older commit was released meanwhile: the version after it" v0.3.3

new_repo newer-commit 0.3.1
release v0.3.1
commit "one"
other=$(other_commit "two")
git -C "$remote" tag -a -m "shade v0.3.2" v0.3.2 "$other"
expect_fail "a newer commit was released meanwhile" "not in this commit's history"
expect_remote_tags "a newer commit was released meanwhile: nothing pushed" "v0.3.1 v0.3.2"
expect_remote_tag_at "a newer commit was released meanwhile: its tag is untouched" v0.3.2 "$other"

new_repo requested-same-commit 0.3.1
release v0.3.1
commit "one"
git -C "$remote" tag -a -m "shade v0.3.2" v0.3.2 "$(git rev-parse HEAD)"
expect_fail "requested version, but a merge run released this commit meanwhile" "already released as v0.3.2" 0.4.0
expect_remote_tags "requested version, commit released meanwhile: nothing pushed" "v0.3.1 v0.3.2"

new_repo requested-taken 0.3.1
release v0.3.1
commit "one"
one=$(git rev-parse HEAD)
commit "two"
git -C "$remote" tag -a -m "shade v0.4.0" v0.4.0 "$one"
expect_fail "requested version that was taken meanwhile" "already exists" 0.4.0

new_repo requested-newer-commit 0.3.1
release v0.3.1
commit "one"
other=$(other_commit "two")
git -C "$remote" tag -a -m "shade v0.3.2" v0.3.2 "$other"
expect_fail "requested version, but a newer commit was released meanwhile" "not in this commit's history" 0.4.0

# --- A tag appears between the fetch and the push ---------------------------
# The fetch did not see it, so the version was picked without it.

new_repo late-other-commit 0.3.1
release v0.3.1
commit "one"
other=$(other_commit "two")
before_git ls-remote git -C "$remote" tag v0.3.2 "$other"
expect_fail "the name is taken by another commit after the fetch" "not at this commit"
expect_remote_tag_at "taken after the fetch: the other tag is untouched" v0.3.2 "$other"
expect_no_local_tag "taken after the fetch: no tag left in the checkout" v0.3.2

new_repo late-other-commit-annotated 0.3.1
release v0.3.1
commit "one"
other=$(other_commit "two")
before_git ls-remote git -C "$remote" tag -a -m "shade v0.3.2" v0.3.2 "$other"
expect_fail "the name is taken by another commit after the fetch, annotated" "not at this commit"
expect_remote_tag_at "taken after the fetch, annotated: the other tag is untouched" v0.3.2 "$other"

new_repo late-same-commit 0.3.1
release v0.3.1
commit "one"
before_git ls-remote git -C "$remote" tag v0.3.2 "$(git rev-parse HEAD)"
expect_push "this commit is tagged after the fetch: nothing" ""
expect_no_local_tag "tagged after the fetch: no tag left in the checkout" v0.3.2

new_repo late-same-commit-annotated 0.3.1
release v0.3.1
commit "one"
before_git ls-remote git -C "$remote" tag -a -m "shade v0.3.2" v0.3.2 "$(git rev-parse HEAD)"
expect_push "this commit is tagged after the fetch, annotated: nothing" ""

new_repo late-longer-name 0.3.1
release v0.3.1
commit "one"
other=$(other_commit "two")
before_git ls-remote git -C "$remote" tag v0.3.20 "$other"
expect_push "a tag whose name only starts the same is not in the way" v0.3.2

new_repo push-other-commit 0.3.1
release v0.3.1
commit "one"
other=$(other_commit "two")
before_git push git -C "$remote" tag v0.3.2 "$other"
expect_fail "the name is taken by another commit just before the push" "not at this commit"
expect_remote_tag_at "taken just before the push: the other tag is untouched" v0.3.2 "$other"
expect_no_local_tag "taken just before the push: no tag left in the checkout" v0.3.2

new_repo push-same-commit 0.3.1
release v0.3.1
commit "one"
# The message differs so that the other run's tag is a different tag object,
# as it would be with another run's timestamp, and the push is rejected.
before_git push git -C "$remote" tag -a -m "shade v0.3.2, from another run" v0.3.2 "$(git rev-parse HEAD)"
expect_push "this commit is tagged just before the push: nothing" ""
expect_no_local_tag "tagged just before the push: no tag left in the checkout" v0.3.2

# --- The remote says no -----------------------------------------------------

new_repo rejected 0.3.1
release v0.3.1
commit "one"
printf '#!/bin/sh\necho "tags are admin-only" >&2\nexit 1\n' >"$remote/hooks/pre-receive"
chmod +x "$remote/hooks/pre-receive"
expect_fail "the remote rejects the tag" "could not push v0.3.2"
expect_remote_tags "the remote rejects the tag: nothing pushed" "v0.3.1"
expect_no_local_tag "the remote rejects the tag: no tag left in the checkout" v0.3.2

new_repo unreachable 0.3.1
release v0.3.1
commit "one"
git remote set-url origin "$work/missing.git"
expect_fail "the remote cannot be reached" "could not fetch the tags"
expect_no_local_tag "the remote cannot be reached: no tag left in the checkout" v0.3.2

echo
if [ "$failures" -ne 0 ]; then
  echo "$failures of $count checks failed"
  exit 1
fi
echo "all $count checks passed"
