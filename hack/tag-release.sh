#!/bin/bash
# Apply and push a release tag for the main module and submodules.
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

# Update submodule versions in go.mod files to the release version.
# The replace directives override these locally; the versions here
# are what external consumers see via the Go module proxy.
sed -i -E "s|pkg/api v[0-9]+\.[0-9]+\.[0-9]+|pkg/api v$VERSION|g" go.mod pkg/mcp/go.mod
sed -i -E "s|pkg/mcp v[0-9]+\.[0-9]+\.[0-9]+|pkg/mcp v$VERSION|g" go.mod
if [ -n "$(git diff go.mod pkg/mcp/go.mod)" ]; then
	git add go.mod pkg/mcp/go.mod
	git commit --amend --no-edit
fi

set -x
git tag "v$VERSION" -a -m "Release $VERSION"
git tag "pkg/api/v$VERSION" -a -m "Release pkg/api $VERSION"
git tag "pkg/mcp/v$VERSION" -a -m "Release pkg/mcp $VERSION"
git push origin "v$VERSION" "pkg/api/v$VERSION" "pkg/mcp/v$VERSION"
