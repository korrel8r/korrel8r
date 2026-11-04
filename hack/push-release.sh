#!/bin/bash
# Publish a prepared release, then resume development on main.
set -euo pipefail

VERSION=${1:?usage: push-release.sh X.Y.Z[-suffix]}
export VERSION # For make
[[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+([-+][0-9A-Za-z.+-]+)?$ ]] || {
	echo "expected version X.Y.Z[-prerelease][+build]: $VERSION" >&2
	exit 1
}
[ "$(git log -1 --format=%s)" = "Release $VERSION" ] || {
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
# A dependency's replace directive is ignored by Go consumers. Verify the
# required versions before publishing the matching module tags.
for module in api mcp; do
	grep -Eq "^[[:space:]]*(require[[:space:]]+)?github.com/korrel8r/korrel8r/pkg/$module v$VERSION([[:space:]]|$)" go.mod || {
		echo "go.mod must require pkg/$module v$VERSION" >&2
		exit 1
	}
done
grep -Eq "^[[:space:]]*(require[[:space:]]+)?github.com/korrel8r/korrel8r/pkg/api v$VERSION([[:space:]]|$)" pkg/mcp/go.mod || {
	echo "pkg/mcp/go.mod must require pkg/api v$VERSION" >&2
	exit 1
}

# pre-release already built and tested this commit.
# No need to lint or test here, just push the commit and publish the versioned and latest images.
git push origin "HEAD:refs/heads/$branch"
make image-latest NOLINT=1
[ -z "$(git status --porcelain)" ] || {
	echo "image build changed the working tree; not tagging" >&2
	exit 1
}
tags=("v$VERSION" "pkg/api/v$VERSION" "pkg/mcp/v$VERSION")
refs=()
for tag in "${tags[@]}"; do
	git tag -a "$tag" -m "Release $VERSION"
	refs+=("refs/tags/$tag")
done
git push --atomic origin "${refs[@]}"
echo "Released $VERSION. Run make post-release VERSION=$VERSION to bump the patch version."
