#!/bin/bash
# Bump the patch version after publishing all three release tags.
set -euo pipefail

VERSION=${1:?usage: post-release.sh X.Y.Z}
export VERSION # For make
[[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || {
	echo "expected version X.Y.Z: $VERSION" >&2
	exit 1
}
[ -z "$(git status --porcelain)" ] || {
	echo "working directory not clean" >&2
	exit 1
}
[ "$(git log -1 --format=%s)" = "Release $VERSION" ] || {
	echo "HEAD is not Release $VERSION" >&2
	exit 1
}
for tag in "v$VERSION" "pkg/api/v$VERSION" "pkg/mcp/v$VERSION"; do
	[ "$(git rev-parse -q --verify "refs/tags/$tag^{}")" = "$(git rev-parse HEAD)" ] || {
		echo "missing release tag at HEAD: $tag" >&2
		exit 1
	}
done

IFS=. read -r major minor patch <<<"$VERSION"
next="$major.$minor.$((10#$patch + 1))"
sed -i "s/^VERSION?=.*/VERSION?=$next/" Makefile
make all
git add -u
git commit -m "Update version to $next"
git push origin HEAD
