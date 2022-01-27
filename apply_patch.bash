#!/usr/bin/env bash

# NOTE: This script creates a fresh version of a Encore patched Go runtime, however we are aiming to create
#       reproducible builds, as such all copies and file writes are given fixed timestamps (rather than the system time
#       when thie script was run). This allows the Git commit at the end of the process to have a deterministic hash,
#       thus when we build the Go binary it will produce an identical binary.

die () {
    echo >&2 "$@"
    exit 1
}

_(){ eval "$@" 2>&1 | sed "s/^/    /" ; return "${PIPESTATUS[0]}" ;}

[ "$#" -eq 1 ] || die "Usage: $0 [go release]

Patches the Go runtime with Encore tracing code.

Examples:
========
A Go release: $0 1.17
Nightly release: $0 master"

set -e

# Parameters
GO_VERSION="$1"
RELEASE_BRANCH="$1"
if [ "$GO_VERSION" != "master" ]; then
  RELEASE_BRANCH="release-branch.go$GO_VERSION"
fi

# Start working in the Go submodule directory
pushd "$(dirname -- "$0")/go" > /dev/null

# Checkout an updated clean copy of the Go runtime from the $RELEASE_BRANCH
echo "♻️  Checking out a clean copy of the Go runtime from $RELEASE_BRANCH..."
  _ git fetch origin "$RELEASE_BRANCH"

  _ git checkout -f "$RELEASE_BRANCH"         # Checkout the branch to track it
  _ git reset --hard origin/"$RELEASE_BRANCH" # Reset against the current head of that branch (ignoring any local commits)
echo

LAST_COMMIT_TIME=$(git log -1 --format=%cd)

# Copy our overlay in and then apply the patches
echo "🏗️  Applying Encore changes to the Go runtime..."
  if [ -f ./VERSION ]; then
    rm ./VERSION
  fi

  _ git apply --3way ../patches/*.diff
  _ cp -p -P -v -R ../overlay/* ./
echo

echo "🤖  Committing runtime changes..."
  _ git add .

  # Note; we set all the details of the commit to git hash deterministic
  GIT_COMMITTER_NAME='Encore Patcher' \
  GIT_COMMITTER_EMAIL='noreply@encore.dev' \
  GIT_COMMITTER_DATE="$LAST_COMMIT_TIME" \
  git commit --allow-empty --date="$LAST_COMMIT_TIME" \
      --author='Encore Patcher <noreply@encore.dev>' \
      -m 'Applied Encore.dev instrumentation changes to the Go runtime' 2>&1 | sed "s/^/    /"
echo

# Restore the working directory back
popd > /dev/null

echo "✅  Done"
