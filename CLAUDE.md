# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Korrel8r is a rule-based correlation engine for observability data in Kubernetes clusters. It correlates signals and resources across diverse formats, schemas, and query languages stored in multiple observability stores.

## Architecture

The project follows a domain-driven design pattern:

- **Domains**: Located in `pkg/domains/`, each domain handles a specific type of observability data (k8s resources, logs, metrics, traces, alerts, incidents, etc.)
- **Engine**: Core correlation logic in `pkg/engine/` that executes correlation rules and traverses relationships
- **Rules**: YAML configuration files in `etc/korrel8r/rules/` define how different data types correlate
- **REST API**: Web service implementation in `pkg/rest/` provides HTTP endpoints for correlation queries
- **MCP Support**: Model Context Protocol implementation in `pkg/mcp/` for AI tool integration

Key interfaces in `pkg/korrel8r/`:
- `Domain`: Defines a type of observability data
- `Store`: Provides access to data repositories
- `Query`: Represents queries in domain-specific languages
- `Object`: Represents individual data items

## Development Commands

### Building and Testing
```bash
make build          # Build korrel8r executable to _bin/korrel8r
make install        # Build and install with go install
make test           # Run all tests (requires OpenShift cluster)
make test-no-cluster # Run tests that don't need a cluster
make lint           # Run linter and fix code style issues
make all            # Build, lint, test everything locally
```

### Running Locally
```bash
make run            # Run korrel8r web server for debugging
make run-mcp        # Run korrel8r MCP server for debugging
make runw           # Run with auto-rebuild on source changes
```

### Testing Configuration
- Default config: `etc/korrel8r/openshift-route.yaml`
- Override with: `make run CONFIG=path/to/config.yaml`
- Environment variable: `KORREL8R_CONFIG`

### Coverage and Benchmarks
```bash
make cover          # Run tests with coverage analysis
make bench          # Run benchmarks
```

### Container Operations
```bash
make image-build    # Build container image locally
make image          # Build and push image (requires REGISTRY_BASE)
make deploy         # Deploy to current Kubernetes cluster
```

### Documentation
```bash
make _site          # Generate documentation website
make tools          # Install development tools
```

## Testing

Tests are organized by domain with shared test utilities:
- Domain tests: `pkg/domains/*/testdata/domain_test.yaml`
- Rule tests: `etc/korrel8r/rules/*_test.go`
- Integration tests require an OpenShift cluster with observability data

Use `make test-no-cluster` for local development without cluster dependencies.

## Configuration

Korrel8r uses YAML configuration files that specify:
- Store connections (Prometheus, Loki, Jaeger, etc.)
- Domain-specific settings
- Authentication and networking options

The main configuration structure is defined in `pkg/config/types.go`.

## Rule System

Correlation rules are YAML files in `etc/korrel8r/rules/` that define relationships between different observability domains. The `all.yaml` file includes all rule sets.

## Domain Implementation

To add a new domain:
1. Create package in `pkg/domains/`
2. Implement Domain, Store, Query, and Object interfaces
3. Add correlation rules in `etc/korrel8r/rules/`
4. Register domain in `pkg/domains/all.go`

## Key Environment Variables

- `KORREL8R_CONFIG`: Path to configuration file
- `GOCOVERDIR`: Directory for coverage data collection
- `REGISTRY_BASE`: Container registry for image operations

## Development Notes

- Uses Go 1.24.0 (check go.mod for current version)
- Supports both podman and docker for container operations
- Uses bingo for versioned tool management
- Code generation required before building (`make generate`)
- Copyright headers enforced by `hack/copyright.sh`