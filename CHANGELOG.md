# Change log for project Korrel8r

This is the project's commit log. It is placeholder until a more user-readable change log is available.

This project uses semantic versioning (https://semver.org)

## Version v0.5.2

- Log domain fixes to enable rules with start=log
- Release v0.5.1
- Use sigs.k8s.io/yaml consistently for YAML parsing.
- Add controller-gen to make config code usable in a k8s API.
- Release v0.5.0
- Fix github workflows to generate all code & docs.
- Added domain doc with class and query examples.
- Enable profiling and /debug/pprof endpoint.
- Change query format to DOMAIN:CLASS:QUERY.
- Fix Makefile: deploy should not depend on image.
- Minor cleanup, simplified log.Object.
- Use lokistak "1x.demo" size for hack/openshift setup.
- Change full class name from CLASS.DOMAIN to DOMAIN:CLASS

## Version v0.4.0

- Release v0.4.0
- Release v0.4.0
- Release v0.4.0
- Fix #65: HTTPs flags to korrel8r server.
- Upgrade dependencies for dependabot alert.
- Asciidoc generation from domain doc comments and REST API
- Call podman with --quiet flag from Makefile

## Version v0.3.2

- Release v0.3.2
- Fix bug: Engine.Class parsing k8s classes for core kinds.
- Publish docs to github pages.
- Fix normalization in api_test.go
- Fix ordering issue in tests mock.Query
- Generate HTML and PDF documentation.
- Create jekyll-gh-pages.yml
- Generate HTML and PDF documentation.
- Documentation improvements.
- Better documentation, easier deployment.
- Engine "template" command mode for doc generation
- Add String() methods to Domain types for readable messages.
- Deploy minio with "make all" in hack/openshift

## Version v0.3.1

- Release v0.3.1
- Improved pkg/korrel8r documentation for developers of new domains.

## Version v0.3.0

- Release v0.3.0
- Present queries as JSON objects in REST API request and response
- Rename JSON field "query" to "queries" in api.Start struct.
- Add CHANGELOG.md and hack/changelog.sh script to generate it.

## Version v0.2.0

- Release v0.2.0
- Update release tagging process.

## Version v0.1.2

- Get class names and descriptions from the rest API and CLI.
- Added class and domain descriptions for documentation.
- Simplify graph.QueryCount to match api.Queries
- Use quay.io image for loki tests.
- Fix main_test configuration.
- Rename name methods String() -> Name()
- Use dotted class names of the form 'classname.domainname'
- Speed up cmd/korrel8r tests
- Fix github build workflow.
- Change verison.go to version.txt
- Update github workflows

## Version v0.1.1

- Use quay.io/korrel8r account, create "latest" version.
- Update README and hack/openshift docs.
- Use central k8s client with dynamic rest mapper.
- Save error information with store config on engine.
- Fix #60: Building and deploying a korrel8r image.
- Add instructions for openshift-local (CRC) cluster setup.
- Fix #61: Warn but don't exit if a store cannot be configured
- Updated all dependencies.
- Improved REST API with swagger model and doc.
- Update engine to accept CLASS.DOMAIN dotted format.

## Version v0.1.0

- changes for v0.1.0
- changes for v0.1.0
- REST API #17
- Fix query serialization in QueryCount.
- Don't backtrack on neighbourhood search
- Fix problem with stores loading from config.
- Refactor Query as string, not struct.
- Rename domain logs->log, consistent capitalization.
- Configuration file definitions and parsing #55
- Make korrel8r.Domain a factory for Stores.
- Register domains with korrel8r.Domains
- Simplified mock tests, fixed bugs uncovered.
- Added copyright line to each file, hack/copyright.sh for new files.
- Refactor: break rules dependency on engine.
- Refactor: simplif main, move internal packages, better main tests.
- Refactor web ui code and introduce placeholder for REST API.
- Demo video: using korrel8r web UI with openshift console.
- Update README.md
- Fix bogus .gitignore entry, remove empty file.
- Don't backtrack in neighbourhood search
- Enable /debug/pprof profile endpoint in UI HTTP server
- Deploy Kubernetes dashboard
- Fix Prometheus <-> Alertmanager alert matching
- Fix QueryToConsoleURL() for alerts
- Add hack/openshift scripts beside hack/kind
- Fixes to the Loki manifests
- Use unpriviledged ports for kind setup
- Fix store URLs in the correlate web page
- Display domains when listing rules
- README.md: link to the GitHub template
- Fix #42 - panic on nil k8s store.
- Add AlertToMetric rule
- New Namespace and Logging rules.
- Tooltips show object previews.
- Minor fixes: logging, HTML, missing schema types.
- Update issue templates
- Add script for deploying a local setup
- Add PodToAlert and NamespaceToAlert rules
- Fix QueryToConsoleURL()
- Retrieve alerts from the Prometheus API
- Demo script: using korrel8r with openshift console in a browser.
- Rules for logs to metrics, logs to pods.
- Refactor graph traversal code
- Added JSONMap to deduplicate JSON-encodable objects.
- Drop comparable requirement for korrel8r.Rule
- Add command-line flags to pass URLs
- Enable dependabot
- Makefile: add build target
- Refactor graph code, fix flaky tests
- Bump golang.org/x/net from 0.5.0 to 0.7.0
- UI hover docs, clearer labels, more self-describing.
- Improved graph traversal logic, avoid repeat store operations.
- Move query unmarshaling to the Domain.
- Web UI takes Query as starting point as well as console URL
- WebUI demo: alerts, events, better UI
- Add metric domain, misc bug fixes
- Move openshift to public pkg
- Remove Class.New, fix ambiguous k8s class names
- Add template rules class groups
- Handle console URLs with label search
- Remove result class from template rules, now part of query
- Gather domain implementations to domain directory
- Fix label selectors
- Switch to struct Query types rather than string or URI reference
- Renamed korrel8 to korrel8r
- Improved web-ui display and interativity
- Reference conversions from console to store reference formats.
- Added korrel8.Store.Resolve
- Update alert domain code.
- Improvements to webUI code.
- Add uri.Reference.RelativeTo
- Rename RefConverter, minor fixes
- Make de-duplication optional for korrel8.Class
- Fix map ordering comparison bug in templaterule tests
- Fix table syntax in README.md
- Console rewriting in web UI.
- Full console URL rewriting for k8s domain.
- WIP: Cleanup web UI, stopped pre vaca
- WIP: Clean up CLI and webUI
- URL rewriting cleanup
- Rename uri.Reference.Values -> Query, like URL
- Remove unused Store.Resolve
- Update README
- Rename korrel8.Query as uri.Reference.
- New rules and tests
- Test fixes for pod selector rules.
- Update README
- Refactor korrel8.Query as stand-alone class
- Get rid of go-multierror, use multierr
- Improved mock store for testing
- Minor refactoring and clean-up
- Refactor rules to new format.
- Fix lint errors.
- Add simple web UI
- Update alert logic.
- WIP: Added URLRewriter for console rules.
- Template rule fixes, k8s event rules.
- Template matching rules - generate multiple rules from one template.
- Switch to gonum graph package, more flexible.
- Replace query string with relative URI
- Support for LokiStack API, improved tests & command line args.
- Multi-function queries - REST or browser.
- Separated graph logic from engine, use shortest path.
- Improved logging and error reporting
- Add golangci-lint check to repo.
- Update import paths for move to gihtub korrel8 org.
- Use Prometheus alerts endpoint instead of Alertmanager (#2)
- Merge pull request #1 from simonpasquier/load-from-yaml-or-json
- cmd/korrel8: skip rule files other than YAML and JSON
- Create go.yml
- Updated README, request for input.
- Replaced Class.NewDeduplicator() with class.Key()
- Changed Rule.Apply to return a single query.
- Load rules from file
- Simplify object, no need for wrapper around native type.
- Add Result type to gather and de-duplicate results.
- Listing k8s classes in the cmd
- Provide constraint as well as object for rule templates.
- Improve testing of main.
- Add Engine to bring it all together.
- First demo, many shortcuts - see FIXME
- Added metrics, moved rules to separte package.
- Tidy up: Refactor and Renaming.
- Added Constraint for time ranges and other filters
- Re-organized alert domain code and internal test packages.
- Following rules and paths, de-duplicating results.
- Initial Loki log store implementation.
- Result.Get() with deduplication
- Identifiable Objects, multi-query Result.
- Fill out k8s code, renaming and doc comments.
- Correlation, first sketch: Types and backwards chaining logic.
