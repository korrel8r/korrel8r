# Korrel8r

[![Go Reference](https://pkg.go.dev/badge/github.com/korrel8r/korrel8r.svg)](https://pkg.go.dev/github.com/korrel8r/korrel8r)
[![Build](https://github.com/korrel8r/korrel8r/actions/workflows/build.yaml/badge.svg)](https://github.com/korrel8r/korrel8r/actions/workflows/build.yaml)
[![Publish](https://github.com/korrel8r/korrel8r/actions/workflows/publish.yaml/badge.svg)](https://github.com/korrel8r/korrel8r/actions/workflows/publish.yaml)
[![Daily Image](https://github.com/korrel8r/korrel8r/actions/workflows/daily-image.yaml/badge.svg)](https://github.com/korrel8r/korrel8r/actions/workflows/daily-image.yaml)

**Navigate relationships between cluster resources and observability signals**

## What is Korrel8r?

Kubernetes clusters generate observability data across diverse systems - logs, metrics, traces, alerts, and events - each with different data models, query languages, and APIs. Troubleshooting requires manually piecing together information from multiple sources, which is time-consuming and error-prone.

**Korrel8r is a rule-based correlation engine** that automatically discovers and graphs relationships between cluster resources and observability signals across multiple data stores, enabling unified troubleshooting experiences.

### Key Features

- **Universal correlation** across multiple formats, stores, and query languages
- **Multi-store relationships** spanning Prometheus, Loki, Alertmanager, Kubernetes API, and more
- **Extensible rules** defined in configurable YAML files
- **Extensible domains** for new signal types, query languages, and data stores
- **Flexible deployment** as a REST service or command-line tool

### Use Cases

- **Cluster operators** troubleshooting issues across complex Kubernetes environments
- **Tool builders** creating observability and troubleshooting interfaces
- **Integrated solutions** like the [OpenShift troubleshooting panel](https://docs.redhat.com/en/documentation/red_hat_openshift_cluster_observability_operator/1-latest/html/ui_plugins_for_red_hat_openshift_cluster_observability_operator/troubleshooting-ui-plugin)

> **Get Started**: See the [User Guide](https://korrel8r.github.io/korrel8r) for installation, usage, and configuration details.

## Documentation

- **[User Guide](https://korrel8r.github.io/korrel8r)** - Complete installation, usage, and reference guide
- **[Developer Guide](AGENTS.md)** - Contributing and development setup
- **[GitHub Issues](https://github.com/korrel8r/korrel8r/issues)** - Report bugs or request features
- **[GitHub Discussions](https://github.com/korrel8r/korrel8r/discussions)** - Community support

## Contributing

We welcome contributions! See the [Developer Guide](AGENTS.md) for setup instructions, development workflows, and contribution guidelines.

- **[Project Board](https://github.com/orgs/korrel8r/projects/3)** - Current work and project status
- **[Pull Requests](https://github.com/korrel8r/korrel8r/pulls)** - Submit contributions

### AI-Assisted Development

Korrel8r includes custom commands for Claude Code to streamline development:
- `/generate-rule` - Interactive assistant for creating correlation rules
- See [.claude/commands/](/.claude/commands/) for all available commands

The [Developer Guide](AGENTS.md) includes specific tips for AI agents working with the codebase.

## Artifacts

``` bash
# Latest release
go install github.com/korrel8r/korrel8r/cmd/korrel8r@latest # Executable
docker pull quay.io/korrel8r/korrel8r:latest                # Image

# Daily images from main for development, not tested for release.
docker pull quay.io/korrel8r/korrel8r:dev-latest            # Latest
docker pull quay.io/korrel8r/korrel8r:dev-v0.8.4-9-g7304c37 # 9 commits since release 0.8.4 
```

See all available images at: https://quay.io/repository/korrel8r/korrel8r?tab=tags

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.
