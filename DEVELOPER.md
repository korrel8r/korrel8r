# Developer Guide

**Architecture and development reference for korrel8r**

> **First time?**
> - [README.md](README.md) - Project overview
> - [CONTRIBUTING.md](CONTRIBUTING.md) - How to contribute, build, and submit changes
> - [User Guide](https://korrel8r.github.io/korrel8r) - How users interact with korrel8r (essential context)

## Architecture Overview

### Domains (`pkg/domains/`)

A `Domain` implements one type of observability data and clients for the associated stores.

Domains must implement these four core interfaces:
- `Domain`: Collection of classes, factory for queries, stores and objects.
- `Store`: Client connection to a data store (Prometheus, Loki, K8s API, etc.)
- `Query`: Domain-specific query representation
- `Object`: Individual data items from the domain

### Rules (`etc/korrel8r/rules/` and `pkg/rules/`)

A rule links a (set of) start classes to a (set of) goal classes.
Each rule contains Go templates that define how to correlate data.
- Start and goal classes may be in the same or different domains.
- The template is applied to an `Object` of a start class, and generates a `Query` to return objects of a goal class.
- See [Configuration](https://korrel8r.github.io/korrel8r/docs/configuration/) for rule syntax and examples.

There are two ways to define rules:

- **Configuration rules** (`etc/korrel8r/rules/`): plain YAML files with the rule metadata and Go template.
  New rules can be added without rebuilding korrel8r. Include new files in `all.yaml` to get them picked up by default configurations.
  See [etc/korrel8r/rules/README.md](etc/korrel8r/rules/README.md).
- **Compiled rules** (`pkg/rules/quickrules/`): quicktemplate `{% func %}` rules compiled into the binary.
  Faster and type-safe, but require rebuilding when a rule changes. Edit only `*.qtpl`, then run `make generate`
  (`qtc` compiles templates, `hack/gen-applyfuncs.sh` regenerates `applyfuncs.go`).
  See [pkg/rules/quickrules/README.md](pkg/rules/quickrules/README.md).

### Engine (`pkg/engine/`)

The `Engine` is the heart of korrel8r.
- Loads domains, stores, and a rule graph from configuration files.
- Creates a *rule graph* with class nodes and rule edges.
- Traverses the rule graph to create a live *correlation graph* including queries and results:
  1. Reduce the total rule graph according to search parameters, to restrict the search space.
  2. Apply rules to objects, which generates queries.
  3. Call stores to evaluate queries, which generates more objects.
  4. Repeat until search criteria are met.
- **Goal search**: find paths from a start object to a specific class of related data.
- **Neighborhood search**: find all data reachable in <= N rules from the start object.

## Coding Guidelines

Follow standard Go formatting rules, automatically enforced by `make lint`.

### Logging Levels

Log at the correct level to make logs readable for operators and useful for debugging:

- **0**: Always visible. Service startup, fatal errors, events requiring human intervention.
- **1**: Low-volume info/warnings for service operators. Don't assume the reader understands the code.
- **2**: Low-volume debugging (setup, occasional state changes).
- **3**: Per-request debugging.
- **4**: Per-rule-evaluation debugging (many per request).
- **5**: Per-query-execution debugging (many per rule evaluation).

## Development Workflows

### Running Locally

Korrel8r can run outside the cluster for development. See [Getting Started](https://korrel8r.github.io/korrel8r/docs/getting-started/#command-line) for setup.

```bash
export KORREL8R_CONFIG="$PWD/etc/korrel8r/openshift-route.yaml"

korrel8r neighbors --query 'k8s:Deployment:{namespace: korrel8r}'
korrel8r web --http :8080
```

### Deploying to a Cluster

Build a container image and deploy. The image includes `etc/korrel8r/openshift-svc.yaml` for in-cluster configuration.

> **Important**: Use a _public_ image repository.

```bash
export REGISTRY_BASE=quay.io/YOUR_ACCOUNT_HERE

make image deploy

KORREL8R_URL=$(oc get route/korrel8r -n openshift-cluster-observability-operator -o template='https://{{.spec.host}}')
TOKEN=$(oc whoami -t)
curl --oauth2-bearer $TOKEN $KORREL8R_URL/api/v1alpha1/domains
```

### Hot-Reload with Devspace

For rapid development cycles, [devspace](https://www.devspace.sh/docs/getting-started/installation) syncs local changes directly to a cluster pod:

```bash
devspace use namespace korrel8r-dev
export REGISTRY_BASE=quay.io/youraccount  # Must be public
make devspace-image
devspace dev
```

## Testing

See [CONTRIBUTING.md](CONTRIBUTING.md) for test commands. Test organization:

- **Package tests**: Standard Go tests in every `pkg/` sub-directory.
- **Cluster tests**: "Cluster" in the test name (e.g., `TestClusterConnection`) means the test requires a cluster.
- **Rule tests**: Tests in `etc/korrel8r/rules/*_test.go` test rules defined in YAML configuration.

## Debugging

### Authentication

```bash
oc whoami
oc auth can-i get pods
export TOKEN=$(oc whoami -t)
curl -H "Authorization: Bearer $TOKEN" $API_SERVER/api/v1/pods
```

### Logging and Profiling

```bash
korrel8r -v3 web                                          # Verbose logging
go test -cpuprofile cpu.prof -memprofile mem.prof ./...    # Profile
go tool pprof cpu.prof
```

## AI Agent Tips

- Core abstractions: `pkg/korrel8r/korrel8r.go`
- Follow existing domains in `pkg/domains/` as patterns for new domains.
- Correlation rules: `etc/korrel8r/rules/` - see [Writing Rules](https://korrel8r.github.io/korrel8r/docs/writing-rules/).
- REST API: `pkg/rest/`
- Use `/generate-rule` to interactively create new correlation rules.
