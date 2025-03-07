# Korrel8r

[![Go Reference](https://pkg.go.dev/badge/github.com/korrel8r/korrel8r.svg)](https://pkg.go.dev/github.com/korrel8r/korrel8r)
[![Build](https://github.com/korrel8r/korrel8r/actions/workflows/build.yml/badge.svg)](https://github.com/korrel8r/korrel8r/actions/workflows/build.yml)
[![Publish](https://github.com/korrel8r/korrel8r/actions/workflows/publish.yml/badge.svg)](https://github.com/korrel8r/korrel8r/actions/workflows/publish.yml)

Korrel8r is an observability tool for correlating observable signals and resources in a kubernetes cluster.

It is a _rule based correlation engine_, with an extensible rule set, that can navigate
- many types of signal and resource data
- serialized in diverse formats, with diverse schema, data models and naming conventions
- queried using diverse query languages
- stored in multiple stores with diverse query APIs

The [Korrel8r Documentation](https://korrel8r.github.io/korrel8r) explains the ideas behind korrel8r,
how to use the built-in rules, and how to write new rules.

These Go packages include:
- [Core abstractions and interfaces](https://pkg.go.dev/github.com/korrel8r/korrel8r/pkg/korrel8r)
  used to implement new _domains_. See the [user guide](https://korrel8r.github.io/korrel8r)

- [Domain packages](https://pkg.go.dev/github.com/korrel8r/korrel8r/pkg/domains) for k8s resources, alerts, logs, metris and more

## Contributing

- [Raising issues](https://github.com/korrel8r/korrel8r/issues)
- [Project plans](https://github.com/orgs/korrel8r/projects/3/views/2)
- [Hackers guide](./doc/HACKING.adoc)

⚠️ **NOTE**: _Early development, no compatibility guarantees_ ⚠️

<!-- ❗NOTE❗ All documentation on this site uses asciidoc, exccept for this README. -->
<!-- This README is markdown to display properly on pkg.go.dev, for Go package documentation. -->
