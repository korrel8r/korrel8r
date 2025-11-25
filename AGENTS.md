# Developer Guide

**Development and contribution guide for korrel8r - for humans and AI agents**

> **New to korrel8r?**
> - [README.md](README.md) - Project overview and key information
> - [User Guide](https://korrel8r.github.io/korrel8r) - Complete user documentation (essential for understanding user workflows)

This guide covers the korrel8r codebase, development setup, and contribution workflows.

## Architecture Overview

> **Understanding User Workflows**: Before diving into the code, read the [User Guide](https://korrel8r.github.io/korrel8r/#how_korrel8r_works) to understand how users interact with domains, classes, queries, and rules.

### Domains (`pkg/domains/`)
A `Domain` implements one type of observability data and clients for the associated stores.

Domains must implement these four core interfaces:
- `Domain`: Collection of classes, factory for queries, stores and objects.
- `Store`: Client connection to a data store (Prometheus, Loki, K8s API, etc.)
- `Query`: Domain-specific query representation
- `Object`: Individual data items from the domain

### Rules (`etc/korrel8r/rules/`)

A rule links a (set of) start classes to a (set of) goal classes.
Each rule contains a Go templates that define how to correlate data.
- Start and goal classes may be in the same or different domains.
- The template is applied to an `Object` of a start class, and generates a `Query` to return objects of a goal class.
- Rules are defined in YAML files, new can be added without rebuilding korrel8r.
- See [User Guide Configuration](https://korrel8r.github.io/korrel8r/#_configuration) for rule syntax and examples.

### Engine (`pkg/engine/`)

The `Engine` is the heart of Korrel8r.
- Loads domains, stores, and a rule graph from configuration files
- Creates  a *rule graph* with class nodes and rule edges.
- Traverses the rule graph to create a live *correlation graph* including queries and results.
  1. Reduce the total rule graph according to search parameters, to restrict the search space.
  1. Apply rules to objects, which generates queries.
  1. Call stores to evaluate queries, which generates more objects.
  1. Repeat until search criteria are met.
- Goal search: find paths from a start object to a specific class of related data.
- Neighborhood search:  find all data reachable in <= N rules from the start object.

## Development Setup
### Prerequisites

- **Go 1.21+** - [Installation guide](https://golang.org/doc/install)
- **Make** - Standard build tool
- **Container runtime** - Docker or Podman for image operations
- **OpenShift/Kubernetes cluster** - Required for cluster tests

Cluster Setup:
- Log into cluster as `kubeadmin` or admin user
- Deploy observability collectors and stores
  - Install from OperatorHub OR
  - Use [korrel8r/config](https://github.com/korrel8r/config) scripts (development only)

### Clone and Build

```bash
git clone https://github.com/korrel8r/korrel8r.git
cd korrel8r
make install        # Install korrel8r to $GOPATH/bin
```

## Development Workflow

### Quick Development Loop

To see descriptions of the main make targets and variables:
``` bash
make help
```

1. Make changes to code
1. Run tests
   ```
   make test
   make test-no-cluster # no cluster available
   ```
1. Do full lint & test before commit
   ```
   make all
   ```

### Running locally

Korrel8r can run outside of the cluster for development. See the [User Guide](https://korrel8r.github.io/korrel8r/#_running_outside_the_cluster) for complete setup instructions.

``` bash
# Set default configuration.
export KORREL8R_CONFIG="$PWD/etc/korrel8r/openshift-route.yaml"

# Execute a single neighborhood query
korrel8r neighbors --query 'k8s:Deployment:{namespace: korrel8r}'

# Run as an out-of-cluster server
korrel8r web --http :8080
```


### Deploying to a cluster

To deploy korrel8r in a cluster you will need to create a container image.
The image includes configuration `etc/korrel8r/openshift-svc.yaml` to run in-cluster
and access stores via internal service addresses.

> **Important**: Use a _public_ image repository.
> Some registry services create _private_ repositories by default.
> You may need to take additional steps to make new repositories _public_.

```bash
# Set your public image repository, example:
export REGISTRY_BASE=quay.io/YOUR_ACCOUNT_HERE

# Build image, deploy to cluster.
make image deploy

# Call the /domains REST endpoint using URL of korrel8r route and login token for cluster.
KORREL8R_URL=$(oc get route/korrel8r -n korrel8r -o template='https://{{.spec.host}}')
TOKEN=$(oc whoami -t)
curl --oauth2-bearer $TOKEN $KORREL8R_URL/api/v1alpha1/domains
```

### Advanced Development Workflows

**Hot-Reload Development with Devspace**

For rapid development cycles, use devspace to sync local changes directly to a cluster pod:

1. Install devspace: https://www.devspace.sh/docs/getting-started/installation
1. Set target namespace
   ```
   devspace use namespace korrel8r-dev
   ```
1. Create development image with sync capabilities
   ```
   export REGISTRY_BASE=quay.io/youraccount  # Must be public repository
   make devspace-image
   ```
1. Start hot-reload development
   ```
   devspace dev
   ```

Now local code changes automatically restart korrel8r in cluster!

## Testing

- **Package tests**: Standard Go tests in every `pkg/` sub-directory.
- **Cluster tests**: "Cluster" in the name of a test (e.g., `TestClusterConnection`) means the test requires a cluster.
- **Rule tests**: Tests in `etc/korrel8r/rules/*_test.go` test rules define in YAML configuration.

You can run any subset of the tests directly using `go test`, or use these make targets:

Run all tests:
```bash
make test
```

Run all tests that do not require a cluster
``` bash
make test-no-cluster
```

### Test Requirements

**Cluster Tests:**
- `kubectl`/`oc` session with cluster access
- Observability stores deployed (Prometheus, Loki, etc.)
- Logged in as `kubeadmin` or other user with sufficient RBAC permissions.
- Tests use current cluster context and credentials, ensure you're logged in:
  ```
  oc whoami  # or kubectl config current-context
  ```

### Coverage Analysis

```bash
make cover                  # Run tests with coverage
go tool cover -html=cover.out  # View coverage report
```

## Debugging
### Common Development Issues

**Authentication Problems**
```bash
# Check cluster connection
oc whoami  # Should return username
oc auth can-i get pods  # Should return "yes"

# Check bearer token
export TOKEN=$(oc whoami -t)
curl -H "Authorization: Bearer $TOKEN" $API_SERVER/api/v1/pods
```

### Logging and Observability

**Enable Debug Logging**
```bash
korrel8r -v3 web  # Verbose logging (levels 1-9)
```

**Profile Performance**
```bash
go test -cpuprofile cpu.prof -memprofile mem.prof ./...
go tool pprof cpu.prof
```

## Contributing

**Contribution Workflow**
1. Fork repository and create feature branch
2. Implement changes following existing patterns
3. Add tests covering new functionality
4. Ensure `make all` passes
5. Submit pull request with clear description

## AI Agent Tips

### Effective Development Patterns

**Understanding the Codebase**
1. Start with `pkg/korrel8r/korrel8r.go` - understand core abstractions
2. Examine existing domains in `pkg/domains/` for implementation patterns
3. Review correlation rules in `etc/korrel8r/rules/` for relationship logic
4. Study REST API in `pkg/rest/` for external interface patterns

**Code Generation and Templates**
- Rule templates use Go template syntax - familiar patterns in `text/template`
- Configuration uses standard YAML/JSON unmarshaling with struct tags
- Domain registration follows Go init() patterns

**Common Gotchas**
- Bearer token authentication - tokens expire, need refresh logic
- Domain interface evolution - check for interface compatibility
- Rule template syntax - Go templates have specific escaping requirements

