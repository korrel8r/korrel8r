#!/bin/bash
# Validate and prepare a release locally; do not publish any commits or tags.
set -euo pipefail

VERSION=${1:?usage: pre-release.sh X.Y.Z[-suffix]}
export VERSION # For make
[[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+([-+][0-9A-Za-z.+-]+)?$ ]] || {
	echo "expected version X.Y.Z[-prerelease][+build]: $VERSION" >&2
	exit 1
}
branch=$(git branch --show-current)
case $branch in
main | v[0-9]*.[0-9]*) ;;
*)
	echo "releases must be from main or a vX.Y branch" >&2
	exit 1
	;;
esac
[ -z "$(git status --porcelain)" ] || {
	echo "working tree must be clean before preparing a release" >&2
	exit 1
}
# Recheck during push-release: the remote branch may advance while validating.
git fetch origin "$branch"
git merge-base --is-ancestor FETCH_HEAD HEAD || {
	echo "local $branch is not based on origin/$branch; integrate upstream changes before preparing the release" >&2
	exit 1
}
if ! grep -q "^## \[$VERSION\] - " CHANGELOG.md; then
	grep -q '^## \[Unreleased\]$' CHANGELOG.md || {
		echo "missing Unreleased changelog section" >&2
		exit 1
	}
fi
for tag in "v$VERSION" "pkg/api/v$VERSION" "pkg/mcp/v$VERSION"; do
	if git rev-parse -q --verify "refs/tags/$tag" >/dev/null || git ls-remote --exit-code --tags origin "refs/tags/$tag" >/dev/null; then
		echo "tag already exists: $tag" >&2
		exit 1
	fi
done

# Generate and validate against the last published module versions. The API
# source (including generated OpenAPI types) must be final before its tag is
# published. This stage builds an image locally, but does not push it.
sed -i "s/^VERSION?=.*/VERSION?=$VERSION/" Makefile
if ! grep -q "^## \[$VERSION\] - " CHANGELOG.md; then
	sed -i "/^## \[Unreleased\]$/a\\
\\
## [$VERSION] - $(date +%F)" CHANGELOG.md
fi
make clean
make all
git add -A
git commit -m "chore(release): pkg/api $VERSION"
[ -z "$(git status --porcelain)" ] || {
	echo "pre-release left uncommitted changes" >&2
	exit 1
}
echo "Release $VERSION validated locally. Review the commit, then run make push-release to publish it."
