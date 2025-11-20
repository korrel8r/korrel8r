# Korrel8r

[![Go Reference](https://pkg.go.dev/badge/github.com/korrel8r/korrel8r.svg)](https://pkg.go.dev/github.com/korrel8r/korrel8r)
[![Build](https://github.com/korrel8r/korrel8r/actions/workflows/build.yml/badge.svg)](https://github.com/korrel8r/korrel8r/actions/workflows/build.yml)
[![Publish](https://github.com/korrel8r/korrel8r/actions/workflows/publish.yml/badge.svg)](https://github.com/korrel8r/korrel8r/actions/workflows/publish.yml)

**Navigate relationships between cluster resources and observability signals**

> This is an overview of the korrel8r project.
> See the [User Guide](https://korrel8r.github.io/korrel8r) for more on installing and using Korrel8r

## What is Korrel8r?

**The Problem:** Kubernetes clusters generate observability data across diverse systems - logs in Loki, metrics in Prometheus, traces in tempo, events in the API server, alerts in Alertmanager and so on. Each system has different data models, query languages, and APIs. When troubleshooting issues, you need to manually piece together information from multiple sources, which is time-consuming and error-prone.

**The Solution:** Korrel8r is a rule-based correlation engine that automatically discovers and graphs relationships between cluster resources and observability signals. It generates relationship graphs that span multiple data stores, enabling tools to provide unified troubleshooting experiences.

**Key Capabilities:**
- **Universal correlation**: Connects data across multiple formats, stores, and query languages (OTEL, Prometheus/PromQL, Loki/LogQL, etc.)
- **Multiples stores**: Relationships can span data in Prometheus, Loki, Alertmanager, Kube API server, and more
- **Extensible Rules**: Configurable YAML rules define how different data types relate to each other
- **Extensible Domains**: Add new domains to handle new signal types, query languages, and data stores
- **Flexible use**: Deploy as a REST service or use from the command line.
  Model Context Protocol (MCP) support for AI integration.

**Who uses Korrel8r:** 
- Cluster administrators, SREs, and developers who need to troubleshoot issues across complex Kubernetes environments. 
- Observability tool builders who want to display and manipulate correlation graphs.\
  The [Red Hat OpenShift troubleshooting panel](https://docs.openshift.com/container-platform/latest/observability/monitoring/accessing-third-party-monitoring-apis.html) displays clickable korrel8r graphs for navigation.

## Documentation & Resources

- **[User Guide](https://korrel8r.github.io/korrel8r)** - How-to and reference guide
- **[Developer Guide](AGENTS.md)** - Development and testing tips for humans and AI agents
- **[REST API Reference](https://korrel8r.github.io/korrel8r/#_rest_api)** - Complete API documentation
- **[Configuration Guide](https://korrel8r.github.io/korrel8r/#_configuration)** - Store configuration and rules

Getting Help
- **[GitHub Issues](https://github.com/korrel8r/korrel8r/issues)** - Report bugs or request features
- **[GitHub Discussions](https://github.com/korrel8r/korrel8r/discussions)** - Community support and questions

## Contributing

We welcome contributions from the community!

- **[Issues](https://github.com/korrel8r/korrel8r/issues)** - Report bugs or request features
- **[Pull Requests](https://github.com/korrel8r/korrel8r/pulls)** - Submit code contributions
- **[Project Board](https://github.com/orgs/korrel8r/projects/3)** - View planned work and project status
- **[Developer Guide](AGENTS.md)** - Development setup and testing guidelines

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.
