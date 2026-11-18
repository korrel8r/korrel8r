#!/bin/bash
# Publish a prepared release, then resume development on main.
set -euo pipefail

VERSION=${1:?usage: push-release.sh X.Y.Z[-suffix]}
export VERSION # For make
[[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+([-+][0-9A-Za-z.+-]+)?$ ]] || {
	echo "expected version X.Y.Z[-prerelease][+build]: $VERSION" >&2
	exit 1
}
[ "$(git log -1 --format=%s)" = "Prepare release $VERSION" ] || {
	echo "run VERSION=$VERSION make pre-release first" >&2
	exit 1
}
grep -Fxq "VERSION?=$VERSION" Makefile || {
	echo "Makefile version is not $VERSION" >&2
	exit 1
}
grep -q "^## \[$VERSION\] - " CHANGELOG.md || {
	echo "missing changelog entry for $VERSION" >&2
	exit 1
}
[ -z "$(git status --porcelain)" ] || {
	echo "working directory not clean" >&2
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
# Check before creating tags: an atomic push cannot advance a diverged branch.
git fetch origin "$branch"
git merge-base --is-ancestor FETCH_HEAD HEAD || {
	echo "origin/$branch advanced since pre-release; integrate upstream, revalidate, and retag before publishing" >&2
	exit 1
}
for tag in "v$VERSION" "pkg/api/v$VERSION" "pkg/mcp/v$VERSION"; do
	if git rev-parse -q --verify "refs/tags/$tag" >/dev/null || git ls-remote --exit-code --tags origin "refs/tags/$tag" >/dev/null; then
		echo "tag already exists: $tag" >&2
		exit 1
	fi
done

# Publish API at the validated commit before changing any of its source.
git tag -a "pkg/api/v$VERSION" -m "Release API $VERSION"
git push --atomic origin "HEAD:refs/heads/$branch" "refs/tags/pkg/api/v$VERSION"

# MCP now requires the published API version, which can be tidied and tested.
go -C pkg/mcp mod edit -require="github.com/korrel8r/korrel8r/pkg/api@v$VERSION"
GOPROXY=direct go -C pkg/mcp mod tidy
go -C pkg/mcp test ./...
git diff --exit-code "pkg/api/v$VERSION" -- pkg/api
git add -A pkg/mcp
git commit -m "Prepare MCP $VERSION"
git tag -a "pkg/mcp/v$VERSION" -m "Release MCP $VERSION"
git push --atomic origin "HEAD:refs/heads/$branch" "refs/tags/pkg/mcp/v$VERSION"

# Resolve and validate the final root dependency versions before publishing
# the root release. All checks that depend on new tags must run at this stage.
go mod edit -require="github.com/korrel8r/korrel8r/pkg/api@v$VERSION" -require="github.com/korrel8r/korrel8r/pkg/mcp@v$VERSION"
GOPROXY=direct go mod download "github.com/korrel8r/korrel8r/pkg/api@v$VERSION" "github.com/korrel8r/korrel8r/pkg/mcp@v$VERSION"
make all
git diff --exit-code "pkg/api/v$VERSION" -- pkg/api
git diff --exit-code "pkg/mcp/v$VERSION" -- pkg/mcp
git add -A
git commit -m "Release $VERSION"
[ -z "$(git status --porcelain)" ] || {
	echo "release commit left uncommitted changes" >&2
	exit 1
}
# Push the image before the root tag so a failed image push leaves the root
# version unpublished. The module tags have already been published.
git push origin "HEAD:refs/heads/$branch"
make image-latest NOLINT=1
[ -z "$(git status --porcelain)" ] || {
	echo "image build changed the working tree; not tagging" >&2
	exit 1
}
git tag -a "v$VERSION" -m "Release $VERSION"
git push origin "refs/tags/v$VERSION"
echo "Released $VERSION. Run 'make post-release' to bump the patch version."
