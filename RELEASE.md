# Releasing a New Version

Release version `X.Y.Z` from `main`, or a patch from a `vX.Y` branch.
The root module, `pkg/api`, and `pkg/mcp` modules share the same version and commit.

## Prepare

1. On branch `main` or `vX.Y`, update the `Unreleased` section of `CHANGELOG.md`.
   Commit and push, you must have a clean tree to run the next step.

2. With test cluster and image registry access, unset any local `REGISTRY_BASE`
   setting so the release uses the default `quay.io/korrel8r`:

   ```sh
   unset REGISTRY_BASE; make pre-release VERSION=X.Y.Z 
   ```

   - Sets the Makefile and module versions to `X.Y.Z` and dates the changelog.
   - Tidies modules, regenerates sources, and runs the full tests.
   - Builds the image, updates manifests
   - Creates a release commit `Release X.Y.Z`, but does not push it.

   **Note:**
   - If this fails, review and clear any partial changes before rerunning; the script requires a clean tree.
   - Do not use `NO_CLUSTER=1` a release must pass all tests.

3. Review the release commit; it should contain only release-related changes:

## Publish tags and images

In a clean tree, after a successful `make pre-release`, run:

```sh
unset REGISTRY_BASE; make push-release
```

This requires a `Release X.Y.Z` commit at HEAD and matching module versions and
`CHANGELOG` entry. `hack/push-release.sh` pushes the commit if needed, publishes
versioned and `latest` images, then atomically pushes three tags:

- `vX.Y.Z` (root module)
- `pkg/api/vX.Y.Z`
- `pkg/mcp/vX.Y.Z`

Go ignores local `replace` directives in published modules, so required module
versions must match the tags.

## Post-release

On `main` or `vX.Y`, run:

```sh
make post-release
```

This checks the release tags, bumps `VERSION?=` to `X.Y.(Z+1)` (no `-dev`),
runs `make all`, and commits and pushes the version bump. Leave Go module
requirements at the last released version until preparing the next release.

## Patch an Older Version

Create branch `vX.Y` from tag `vX.Y.0` if it does not already exist. Follow the
same preparation and publication steps on that branch.
