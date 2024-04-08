#!/bin/bash
# Apply and push a release tag.
set -e
VERSION=$1
[[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || {
	echo "not a semantic version X.Y.Z: $VERSION"
	exit 1
}
[ "$(git status -sb)" = "## main...origin/main" ] || {
	echo "working directory not clean"
	echo
	git status
	exit 1
}
set -x
git tag "v$VERSION" -a -m "Release $VERSION"
git push origin "v$VERSION"
