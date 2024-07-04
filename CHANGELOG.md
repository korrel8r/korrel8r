# Change Log

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased]

### Fixed

- [Delegate HTTP header credentials from REST API to stores](https://github.com/korrel8r/korrel8r/issues/120)
  Korrel8r now impersonates the Authorization header to act on behalf-of its callers.
- [Use strict parsing to catch query errors.](https://github.com/korrel8r/korrel8r/issues/107)

## [0.6.4] - 2024-05-29

### Removed

- Korrel8r web API server has been removed from the `korrel8r` command.
  Replaced by http://github.com/korrel8r/client, a REST client and visualization tool.

### Deprecated

- `korrel8r web` command line flags:
  - `korrel8r web --html` - use the new `korrel8rcli` command instead.
  - `korrel8r web --rest` - no longer required, REST server is always activated.

### Added

-  [New rules: ConsolePlugin, PodDisruptionBudget](https://github.com/korrel8r/korrel8r/commit/98f449b8a764e213dfb0c5c8ae37763bb6b88907)
- `korrel8r web --spec` dumps the swagger specification for korrel8r to stdout or a file.

### Fixed

- [Fix in-cluster service accounts and certs.](https://github.com/korrel8r/korrel8r/issues/116)
- [OSC-8 Korrel8r does not deply on Openshift 4.15 due to security profile restrictions bug](https://github.com/korrel8r/korrel8r/issues/105)
- [Add Changelog for Korrel8r releases](https://github.com/korrel8r/korrel8r/issues/102)


## [0.6.3] - 2024-05-14

No change log was kept up to this release, use `git log` for git commit history.
