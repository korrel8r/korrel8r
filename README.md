# Signal correlation for Kubernetes

[![Build](https://github.com/korrel8r/korrel8r/actions/workflows/build.yml/badge.svg)](https://github.com/korrel8r/korrel8r/actions/workflows/build.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/korrel8r/korrel8r.svg)](https://pkg.go.dev/github.com/korrel8r/korrel8r)

## Overview

There are many tools that collect observability signals from a Kubernetes clusters.
Each tool may use different labelling conventions, data stores, and query languages for the data it collects. 

Korrel8r uses an extendable set of _rules_ to follow relationships between different types of signal data,
even when they use incompatible schema and query languages.

## Documentation

- [Korrel8r user guide and reference](https://korrel8r.github.io/korrel8r)
- [Hackers guide](./doc/HACKING.adoc) for experimenting and contributing to the project.
- Important Go packages:
  - [Core abstractions and interfaces](https://pkg.go.dev/github.com/korrel8r/korrel8r/pkg/korrel8r)
  - [Domain packages](https://pkg.go.dev/github.com/korrel8r/korrel8r/pkg/domains)

⚠️ **NOTE**: _Early development, no compatibility guarantees_ ⚠️

<!-- ❗NOTE❗ All documentation on this site uses asciidoc, exccept for this README. -->
<!-- This README is markdown to display properly on pkg.go.dev, for Go package documentation. -->
