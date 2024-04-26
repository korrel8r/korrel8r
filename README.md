# Signal correlation for Kubernetes

[![Build](https://github.com/korrel8r/korrel8r/actions/workflows/build.yml/badge.svg)](https://github.com/korrel8r/korrel8r/actions/workflows/build.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/korrel8r/korrel8r.svg)](https://pkg.go.dev/github.com/korrel8r/korrel8r)

## Overview

There are many tools that collect observability signals from a Kubernetes clusters.
Each tool may use different labelling conventions, data stores, and query languages for the data it collects. 

Korrel8r calls each distinct set of tools and conventions a _domain_, for example.
- Container logs stored in Loki.
- Metrics and alerts stored in Prometheus.
- Kubernetes resources stored in thea API server.

Korrel8r uses _rules_ to define relationships between data in different domains.
Rules form a _graph_. Walking the graph can correlate data via indirect relationships that span multiple domains.

## Documentation

- [Korrel8r user guide and reference](https://korrel8r.github.io/korrel8r)
- [Hackers guide](./doc/HACKING.adoc) for experimenting and contributing to the project.
- Important Go packages:
  - [Core abstractions and interfaces](https://pkg.go.dev/github.com/korrel8r/korrel8r/pkg/korrel8r)
  - [Domain packages](https://pkg.go.dev/github.com/korrel8r/korrel8r/pkg/domains)

⚠️ **NOTE**: _Early development, no compatibility guarantees_ ⚠️

<!-- NOTE: All documentation on this site uses asciidoc, exccept for this README -->
<!--       This README is in markdown due to limitations of pkg.dev.go -->
