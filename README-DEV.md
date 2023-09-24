# Developer information

Browsable Go and REST API documentation (generated from source):
- [REST API documentation](pkg/api/docs/swagger.md)
- [Go API documentation](https://pkg.go.dev/github.com/korrel8r/korrel8r/pkg/korrel8r)

If you are interested in helping to develop Korrel8r:
- clone this repository
- `make help` will list make targets with brief explanation.
- `make run` will run korrel8r directly from source using the checked-in configuration

## Building Images ##

By default, the Makefile uses `quay.io/korrel8r/korrel8r` as its image repository.
Set IMG to use a different repository:

- `make image IMG=quay.io/myaccount/mykorrel8r` build and push an image to your image repository.
- `make deploy IMG=quay.io/myaccount/mykorrel8r` deploy your image to your cluster.

**NOTE**: you need a _public_ image repository on a public service like `quay.io` or `docker.io`.
On some services (including `quay.io`) new repositories are _private_ by default.
You may need to log in and manually set the visibility of your new korrel8r repository _public_.

## Tags and Releases ##

The image tag is set from `git describe`, so there's always a simple relationship between git and image tags.
Git describe computes a descriptive name based on the nearest tag and git hash - like `v0.1.0-9-g92f8e41`

- `make tag TAG=vX.Y.Z` creates a vX.Y.Z release tag on the current branch.
  Build and push the corresponding image with `make image`. 
- `make release` pushes the current image tag and creates a `:latest` alias.
