#!/bin/bash
# Apply and push a release tag for the main module.
set -e
VERSION=$1
[[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[^[:space:]]+)?$ ]] || {
	echo "not a semantic version X.Y.Z: $VERSION"
	exit 1
}
BRANCH=$(git rev-parse --abbrev-ref HEAD)
[[ "$BRANCH" == "main" || "$BRANCH" =~ ^v[0-9]+\.[0-9]+$ ]] || {
	echo "releases must be from 'main' or a 'vX.Y' branch, not '$BRANCH'"
	exit 1
}
[ "$(git status -s)" = "" ] || {
	git status
	echo "working directory not clean"
	exit 1
}

set -x
git tag "v$VERSION" -a -m "Release $VERSION"
git push origin "v$VERSION"
