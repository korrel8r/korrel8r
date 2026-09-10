---
title: Korrel8r
layout: hextra-home
---

<div class="hx:mt-6 hx:mb-6">
{{< hextra/hero-headline >}}
  Correlation engine&nbsp;for Kubernetes
{{< /hextra/hero-headline >}}
</div>

<div class="hx:mb-12">
{{< hextra/hero-subtitle >}}
  Connect observability signals &mdash; logs, metrics, alerts, traces, network flows &mdash;&nbsp;
  into a searchable graph of related data across multiple back-ends.
{{< /hextra/hero-subtitle >}}
</div>

<div class="hx:mb-6">
{{< hextra/hero-button text="Introduction" link="docs/introduction/" >}}
{{< hextra/hero-button text="Get Started" link="docs/getting-started/" style="background: transparent; color: inherit; border: 1px solid #e5e7eb;" >}}
</div>

{{< hextra/feature-grid >}}
  {{< hextra/feature-card
    title="Correlation Searches"
    subtitle="Neighborhood search finds everything reachable within N steps. Goal search finds paths to a specific type of data &mdash; connecting an alert to its deployments, pods, and logs."
    icon="search-circle"
    link="docs/introduction/#search-strategies"
  >}}
  {{< hextra/feature-card
    title="AI Agent Ready"
    subtitle="MCP interface lets AI agents use korrel8r's rule engine to find correlated data instead of spending tokens discovering connections."
    icon="sparkles"
    link="docs/ai-agents/"
  >}}
  {{< hextra/feature-card
    title="Multi-Store"
    subtitle="Query Prometheus, Loki, Tempo, the Kubernetes API, and more through a single correlation engine."
    icon="database"
    link="docs/reference/domains/"
  >}}
  {{< hextra/feature-card
    title="Extensible Rules"
    subtitle="Write correlation rules in YAML with Go templates. No code changes needed to add new relationships."
    icon="puzzle"
    link="docs/writing-rules/"
  >}}
  {{< hextra/feature-card
    title="OpenShift Console"
    subtitle="The troubleshooting panel displays interactive correlation graphs. Click nodes to jump directly to related resources."
    icon="template"
    link="docs/troubleshooting-panel/"
  >}}
  {{< hextra/feature-card
    title="Command Line & API"
    subtitle="Run correlation searches from the korrel8r CLI, or drive the engine over its REST API from your own tools and scripts."
    icon="terminal"
    link="docs/command/"
  >}}
{{< /hextra/feature-grid >}}
