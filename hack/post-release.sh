#!/bin/bash
# Bump to the next -dev version after publishing a release.
set -euo pipefail

previous_version=${1:?usage: post-release.sh X.Y.Z}
export previous_version # For make
[[ "$previous_version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || {
  echo "expected version X.Y.Z: $previous_version" >&2
  exit 1
}
[ -z "$(git status --porcelain)" ] || {
  echo "working directory not clean" >&2
  exit 1
}

IFS=. read -r major minor patch <<<"$previous_version"
next_version="$major.$minor.$((10#$patch + 1))-dev"
sed -i "s/^VERSION?=.*/VERSION?=$next_version/" Makefile
make all
git add -u
git commit -m "Update version to $next_version"
echo "Version update commit $next_version is ready to push." >&2
