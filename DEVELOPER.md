# Developer Guide

Instructions for coding agents and contributors working in Korrel8r. Keep changes focused, follow existing patterns, and validate the smallest relevant scope before running repository-wide checks.

## Start Here

- Read [README.md](README.md) for product context and [CONTRIBUTING.md](CONTRIBUTING.md) for contributor requirements.
- See [developer notes and documentation](doc/dev/README.md) for technical procedures and measurement workflows.
- User documentation under `doc/content` is published at https://korrel8r.github.io/korrel8r and can be previewed using `make preview`
- Run `make help` for the authoritative list of build targets and variables.
- Before editing, inspect nearby implementation and tests. Do not assume conventions from another domain or package apply unchanged.
- Preserve unrelated working-tree changes. Never discard or rewrite files outside the requested scope.

## Repository Map

| Path | Purpose |
| --- | --- |
| `cmd/korrel8r/` | CLI and server entry point |
| `pkg/korrel8r/korrel8r.go` | Core `Domain`, `Class`, `Store`, `Query`, `Object`, and `Rule` contracts |
| `pkg/domains/` | Alert, incident, Kubernetes, log, metric, netflow, and trace domain implementations |
| `pkg/engine/` | Rule-graph construction and correlation searches |
| `pkg/graph/`, `pkg/result/` | Graph and result data structures |
| `pkg/rules/` | Runtime template-rule implementation |
| `pkg/rules/quickrules/` | Compiled quicktemplate rules |
| `pkg/config/` | Configuration loading and rule metadata |
| `pkg/rest/`, `pkg/api/` | REST implementation and generated OpenAPI types |
| `pkg/mcp/` | MCP interface |
| `etc/korrel8r/` | Runtime configuration and configuration rules |
| `korrel8r-openapi.yaml` | REST API source specification |
| `doc/` | Documentation site sources and generators |

## Architecture

A domain represents one observability data type and its backing store. Domain implementations center on:

- `Domain`: classes and factories for queries and stores.
- `Class`: schema/type metadata and object decoding.
- `Store`: executes a domain query; a not-found condition returns an empty result, not an error.
- `Query`: comparable, domain-specific selection with a fully qualified string form.
- `Object`: the underlying JSON-compatible signal value.

A rule connects start classes to goal classes. Applying a rule to a start object produces goal queries. The engine reduces the total rule graph for a search, applies rules, executes generated queries, and repeats as it builds the result graph.

- **Goal search:** find paths from a start object to a requested class.
- **Neighborhood search:** find everything reachable within a maximum number of rules.

When adding a domain, use the closest existing package under `pkg/domains/` as a model and add package-local tests/testdata. Keep interface behavior consistent with `pkg/korrel8r/korrel8r.go`.

## Rules

There are two rule forms:

### Compiled quickrules

- Sources: `pkg/rules/quickrules/*.qtpl`
- Tests: `pkg/rules/quickrules/*_test.go`
- Detailed format: `pkg/rules/quickrules/doc.go`
- Edit `.qtpl` sources, then run `make generate`.
- Do **not** hand-edit generated `.qtpl.go` files or `pkg/rules/quickrules/applyfuncs.go`.

Prefer quickrules for built-in, type-safe, performance-sensitive relationships. Add table-driven cases for generated queries and non-applicable inputs.

### Configuration rules

- Runtime rules: `etc/korrel8r/rules/`
- Syntax guide: [Writing Rules](https://korrel8r.github.io/korrel8r/docs/writing-rules/)
- YAML rules use Go `text/template` and require no binary rebuild.
- Include new runtime rule files from `etc/korrel8r/rules/all.yaml` when they should load by default.

Use configuration rules for user-installable or rapidly iterated rules. Preserve valid YAML and ensure generated query strings parse in the goal domain.

## Generated Files

`make generate` is the supported generation path. Generated artifacts include:

- `pkg/api/gen-openapi.go`
- `pkg/rest/gen-openapi.go`
- `pkg/rules/quickrules/*.qtpl.go`
- `pkg/rules/quickrules/applyfuncs.go`
- `pkg/domains/*/doc.md`
- `internal/pkg/build/version.txt`

Edit their source specifications/templates instead. Commit regenerated outputs when the source change requires them. After generation or linting, inspect `git diff` because these targets may update tracked files.

## Coding Conventions

- Use standard Go style and package-local conventions.
- Keep public API comments accurate and add tests for behavioral changes.
- Wrap errors with useful operation/context information; preserve errors where callers need `errors.Is`/`errors.As`.
- Pass `context.Context` through I/O and request paths; do not replace caller context with a background context.
- Avoid broad refactors in bug fixes. Do not add dependencies unless necessary.
- For shell changes, follow existing scripts; lint runs `shfmt` and `shellcheck`.
- `make lint` is mutating: it runs generation, `go mod tidy`, `golangci-lint --fix`, and `shfmt -w`.

### Logging Levels

Use the project verbosity levels consistently:

- **0:** startup, fatal failures, or events requiring human action.
- **1:** low-volume operator information/warnings; avoid code-internal language.
- **2:** low-volume setup or state-change debugging.
- **3:** per-request debugging.
- **4:** per-rule-evaluation debugging.
- **5:** per-query-execution debugging.

## Validation

Start narrow, then expand according to the change:

```bash
# Fast package checks
go test ./pkg/path/to/changed/package

go test ./pkg/rules/quickrules/   # Quickrule changes
make generate                     # Generated-source changes
make lint                         # Full formatting/lint; modifies files
make test NO_CLUSTER=1            # Repository tests without cluster cases
make test                         # Full suite; requires cluster access
make all                          # Pre-commit build/test/image/doc validation
```

Tests whose names end in `_cluster` (including `_cluster-fm`) require a configured Kubernetes/OpenShift environment. Do not claim full validation if those tests were skipped or infrastructure was unavailable. Report the exact commands run and any failures.

For OpenAPI changes, edit `korrel8r-openapi.yaml`, regenerate, and test both `pkg/api` and `pkg/rest`. For documentation changes, use `make doc` and, when relevant, `make check-links`.

## Local and Cluster Workflows

Run locally with an appropriate configuration:

```bash
export KORREL8R_CONFIG="$PWD/etc/korrel8r/openshift-route.yaml"
korrel8r neighbors --query 'k8s:Deployment:{namespace: korrel8r}'
korrel8r web --http :8080
```

Cluster tests and deployment need a valid `kubectl`/`oc` session and sufficient RBAC. Deployment images must use a public repository:

```bash
export REGISTRY_BASE=quay.io/YOUR_ACCOUNT
make image deploy
```

Useful authentication checks:

```bash
oc whoami
oc auth can-i get pods
```

For rapid in-cluster iteration, see the Devspace workflow in the project documentation and the `devspace-image` Make target.

## Completion Checklist

Before finishing:

1. Review `git diff` and `git status`; verify only intended changes are present.
2. Confirm generated files match their sources and no generated file was edited directly.
3. Run focused tests plus the broadest feasible lint/test command.
4. Add or update tests and documentation for changed behavior.
5. Summarize changed files, validation performed, and any skipped cluster-dependent checks.
