# Change Log

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased] - 2024-05-28

**NOTE**: _Add upgrade notes here when these changes are released_

### Removed

- Korrel8r web API server has been removed from the `korrel8r` command.
  The new `korrel8rcli` command provides a similar feature.

### Deprecated

- `korrel8r web` command line flags:
  - `korrel8r web --html` - use the new `korrel8rcli` command instead.
  - `korrel8r web --rest` - no longer required, REST server is always activated.

### Added

- New `korrel8rcli` command at ./client/cmd/korrel8rcli
  - REST client, command line access to a remote korrel8r server. See `korrel8rcli --help`
  - Web browser API using data from remote korrel8r server, see `korrel8rcli web --help`
  - Client packages for 3rd party use, see ./client/pkg/swagger
-  [New rules: ConsolePlugin, PodDisruptionBudget](https://github.com/korrel8r/korrel8r/commit/98f449b8a764e213dfb0c5c8ae37763bb6b88907)

### Fixed 

- [Fix in-cluster service accounts and certs.](https://github.com/korrel8r/korrel8r/issues/116)
- [OSC-8 Korrel8r does not deply on Openshift 4.15 due to security profile restrictions bug](https://github.com/korrel8r/korrel8r/issues/105)
- [Add Changelog for Korrel8r releases](https://github.com/korrel8r/korrel8r/issues/102)


## [0.6.3] - 2024-05-14

No change log was kept up to this release, use `git log` for git commit history.
