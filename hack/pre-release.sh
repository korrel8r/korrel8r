#!/bin/bash
# Prepare, validate, and commit a release. Does not publish tags or images.
set -euo pipefail

VERSION=${1:?usage: pre-release.sh X.Y.Z[-suffix]}
export VERSION # For make
[[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+([-+][0-9A-Za-z.+-]+)?$ ]] || {
	echo "expected version X.Y.Z[-prerelease][+build]: $VERSION" >&2
	exit 1
}
case $(git branch --show-current) in
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
if ! grep -q "^## \[$VERSION\] - " CHANGELOG.md; then
	grep -q '^## \[Unreleased\]$' CHANGELOG.md || {
		echo "missing Unreleased changelog section" >&2
		exit 1
	}
fi
for tag in "v$VERSION" "pkg/api/v$VERSION" "pkg/mcp/v$VERSION"; do
	if git rev-parse -q --verify "refs/tags/$tag" >/dev/null; then
		echo "tag already exists: $tag" >&2
		exit 1
	fi
done

sed -i "s/^VERSION?=.*/VERSION?=$VERSION/" Makefile
if ! grep -q "^## \[$VERSION\] - " CHANGELOG.md; then
	sed -i "/^## \[Unreleased\]$/a\\
\\
## [$VERSION] - $(date +%F)" CHANGELOG.md
fi

set -x
go mod edit -require="github.com/korrel8r/korrel8r/pkg/api@v$VERSION" -require="github.com/korrel8r/korrel8r/pkg/mcp@v$VERSION"
go -C pkg/mcp mod edit -require="github.com/korrel8r/korrel8r/pkg/api@v$VERSION"

make clean
make all kustomize-edit

git add -A
git commit -m "Release $VERSION"
[ -z "$(git status --porcelain)" ] || {
	echo "release commit left uncommitted changes" >&2
	exit 1
}
echo "Release $VERSION prepared. Review the commit, then run make push-release."
