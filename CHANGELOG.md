# Change Log

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased]

_Note unreleased changes on main here pending the next release_

### Changed
- BREAKING CHANGE: combined otellog and podlog domains into log domain.
  - Both OTEL AND Viaq log objects are now flat maps with "_" separated keys.
  - New k8s query syntax for all logs: {name: namespace: labels: fields: containers:}
  - Still supports LogQL queries for stored logs.
  - Default config: store first, fall back to direct API server access.

### Changed
- Replaced Swagger 2 specification with OpenAPI 3, updated REST API code.
- Added authorization present/absent in debugging logs (no auth material included)
- Added doc/lab/model-context-protocol.adoc with MCP experiments.

### Removed
- Dropped Built-in swagger UI.
  Was only useful for trivial experiments, can use korrrelrcli as client instead.

### Fixed
- Fixed bug causing panic if invalid class was passed to the REST API in a goal search.

## [0.8.1] - 2025-05-23

### Added
- Support for logs stored in the OTEL format.
- Log output from k8s/client-go package included in logs.

## [0.8.0] - 2025-02-12

### Changed
- BREAKING CHANGE: k8s field names are now "lowerCase" serialized JSON style, not "UpperCase" Go style.
  Affects k8s rule templates, which are now compatible with `kubectl -o template`.

### Fixed
- Wildcard rules (empty start and goal) were not being processed.

### Added
- k8s can handle any resource type defined in the connected API server, not just a compiled-in set.
  Uses the runtime API server and unstructured map objects and the runtime API server.
- incidents domain: Incidents group alerts into higher-level groups. 
- Added DependentToOwner rule to follow k8s ownerReferences
- Improved diagnostic logging: de-duplicate errors, better log messages.
- Added --rules flag to korrel8r goals, neighbours
- File-based mock stores for internal testing.

## [0.7.6] - 2024-12-19

### Fixed
- Clean up command line, logging levels.
- Apply constraint timeout to total operation length.

### Added
- feat(#182): Faster concurrent graph traversal
- Profiling options
- Commands: goals, neighbours

## [0.7.5] - 2024-11-22

### Fixed
- fix(COO-557): Remove spurious '|json' in log queries.
- fix(COO-563): Korrel8r does not interrupt over-long queries.
- fix(COO-558): Fix parsing of complex promql queries.

## [0.7.4] - 2024-11-12

### Added
- Adjust log verbose level via API at runtime.
- Added "ndjson" output type

### Fixed
- Trace domain fixes: OTEL rules, tempo select clauses and constraints, handle missing attributes. 

## [0.7.3] - 2024-11-01

### Fixed
- General overhaul of project documentation.
- Always include start node in neighbours search
- Fix nil pointer errors in REST error handling.
- Re-organized trace domain for better OTEL compliance.
- Trim trailing/leading whitespace from query strings.

## [0.7.2] - 2024-08-23

### Fixed
- #156: Enable errors for missing values in templates

## [0.7.1] - 2024-08-22

### Removed

- Removed deprecated `korrel8r web` command line flags `--http` and `--rest`
- Removed `korrel8r --profile=<type>` flag, use `korrel8r web --profile`

### Added
-  `korrel8r web --profile` to enable/disable http profiles.

## [0.7.0] - 2024-08-22

### Fixed

- [Delegate HTTP header credentials from REST API to stores](https://github.com/korrel8r/korrel8r/issues/120)
  Korrel8r now impersonates the Authorization header to act on behalf-of its callers.
- [Authentication and Authorization for restricted access](https://github.com/korrel8r/korrel8r/issues/73)
- [Use strict parsing to catch query errors.](https://github.com/korrel8r/korrel8r/issues/107)
- REST API fix invalid JSON in responses, return [] instead of null for empty lists.
- Bugs in forwarding REST authorization tokens to stores.
- Various other bug fixes.

### Added
- Trace domain to support traces in a Grafana Tempo store.
- Mock store for unit tests, moved cluster tests to test/openshift.
- Default constraints and timeouts to avoid slow response times caused by excessively large responses.
  - Defaults can be over-ridden by REST and command-line options.
  - DefaultDuration queries for only an hour of of data.
  - DefaultLimit restricts queries to at most 1000 result objects.
  - DefaultTimeout limits query latency to 5 seconds.

## [0.6.6] - 2024-06-04

### Added
- "objects" operation on REST APIs

### Fixed
- Error messages for REST API.
- Swagger page to use host from URL for "try it out"
- Switch to ubi8 base image, Makefile cleanup.

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
- `korrel8r web --spec` dumps the OpenAPI specification for korrel8r to stdout or a file.

### Fixed

- [Fix in-cluster service accounts and certs.](https://github.com/korrel8r/korrel8r/issues/116)
- [OSC-8 Korrel8r does not deply on Openshift 4.15 due to security profile restrictions bug](https://github.com/korrel8r/korrel8r/issues/105)
- [Add Changelog for Korrel8r releases](https://github.com/korrel8r/korrel8r/issues/102)


## [0.6.3] - 2024-05-14

No change log was kept up to this release, use `git log` for git commit history.
