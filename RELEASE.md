# Releasing a New Version

Release version `X.Y.Z` from `main`, or a patch from a `vX.Y` branch.
The root, `pkg/api`, and `pkg/mcp` modules share a version but are tagged on
successive commits: API, MCP, then root.

## Prepare

Prepare to release - local only, does not publish anything.
Update and commit the `Unreleased` section of `CHANGELOG.md` on `main` or a `vX.Y` branch.
With a clean tree and cluster access, run:

```sh
unset REGISTRY_BASE; make pre-release VERSION=X.Y.Z
```

This runs `make all` against the last published module versions and creates a
local `chore(release): pkg/api X.Y.Z` commit. It pushes nothing. Review the commit.
Do not set `NO_CLUSTER=1`, a release should pass all tests.

## Publish tags and images

With cluster and image registry access, run:

```sh
unset REGISTRY_BASE; make push-release
```

This publishes the API tag, then updates and tidies MCP against it and publishes
the MCP tag. Finally it updates and validates the root against both tags,
publishes the image, and publishes the root tag. `go mod tidy` cannot resolve
unpublished versions, so final-version checks must run at this stage.

Both scripts check that the branch contains the latest upstream commit. If
`push-release` fails, do not move or delete published tags: finish the remaining
stage or use a new version if tagged source must change. If the first atomic
push fails, its API tag is *only local*; integrate upstream, revalidate, and
remove that unpublished local tag before retrying. Never force-push.

## Post-release

On `main` or `vX.Y`, run:

```sh
unset REGISTRY_BASE; make post-release
```

This bumps `VERSION?=` to `X.Y.(Z+1)` (no `-dev`),
runs `make all`, and commits and pushes the version bump. Leave Go module
requirements at the last released version until preparing the next release.

## Patch an Older Version

Create branch `vX.Y` from tag `vX.Y.0` if it does not already exist. Follow the
same preparation and publication steps on that branch.
