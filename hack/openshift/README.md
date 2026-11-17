# Set up an OpenShift cluster for korrel8r tests

The setup scripts and resources live in <https://github.com/korrel8r/config>, in the
`openshift` directory. They are not copied into this repository; `make` fetches them
on demand into the git-ignored directory `_test-setup`.

Log in to an OpenShift cluster as `cluster-admin`, then:

```bash
make test-setup         # Install observability signals (logs, metrics, traces, network)
make test               # Run the full suite, including _cluster tests
make test-setup-clean   # Remove the installed observability components
```

Variables (see `make help`):

- `TEST_SETUP_REPO`: repository to fetch, defaults to the public HTTPS URL.
  Use `git@github.com:korrel8r/config.git` if you prefer SSH.
- `TEST_SETUP_REF`: branch, tag or commit to use, defaults to `main`.
  Pin it to a known-good commit for reproducible setup.
- `TEST_SETUP_DIR`: checkout location, defaults to `_test-setup`.
  Point it at an existing clone to work from that instead:

  ```bash
  make test-setup TEST_SETUP_DIR=$HOME/src/korrel8r/config
  ```

Individual components can be installed separately, see the `openshift/Makefile`
in the test-setup repository.
