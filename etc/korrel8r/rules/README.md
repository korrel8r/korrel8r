# Korrel8r Rules

This directory contains correlation rules that define relationships between observability domains.

## Rule Structure

Rules are defined in YAML files, organized by domain:
- `k8s.yaml` - Kubernetes resource correlations
- `log.yaml` - Log correlations
- `metric.yaml` - Metric correlations (pending)
- `alert.yaml` - Alert correlations
- `trace.yaml` - Trace correlations
- `netflow.yaml` - Network flow correlations
- `incident.yaml` - Incident management correlations
- `all.yaml` - Cross-domain correlations

Each rule file follows this structure:
```yaml
rules:
  - name: RuleName
    start:
      domain: source-domain
      classes: [SourceClass]  # Optional: specific classes
    goal:
      domain: target-domain
      classes: [TargetClass]  # Optional: specific classes
    result:
      query: |-
        <query-template>
```

## Creating New Rules

### Using the `/generate-rule` Command (Recommended)

If you're using Claude Code, use the interactive rule generator:
```
/generate-rule
```

This command will guide you through creating a new rule with the correct syntax and structure.

### Manual Creation

1. Choose the appropriate YAML file or create a new one
2. Add your rule following the structure above
3. Use Go template syntax in the query field
4. Test your rule with `make test`

See the [User Guide - Configuration](https://korrel8r.github.io/korrel8r/#_configuration) for detailed rule syntax and examples.

## Testing Rules

Each rule file has a corresponding `*_test.go` file. After creating a rule:

1. Add test cases to the appropriate test file
2. Run tests:
   ```bash
   make test                # All tests
   make test-no-cluster    # Without cluster
   go test ./etc/korrel8r/rules/  # Just rule tests
   ```

## Common Template Functions

Available in rule queries:
- `{{.metadata.namespace}}`, `{{.metadata.name}}` - K8s object fields
- `{{mustToJson .}}` - Convert to JSON
- `{{k8sClass .apiVersion .kind}}` - Generate K8s class name
- `{{lower .kind}}` - Lowercase string
- `{{logTypeForNamespace .metadata.namespace}}` - Get log type for namespace
- Standard Go template functions: `{{with}}`, `{{range}}`, etc.

## More Information

- [User Guide - Configuration](https://korrel8r.github.io/korrel8r/#_configuration) - Full rule documentation
- [AGENTS.md](../../../AGENTS.md) - Developer guide
- [.claude/commands/](../../../.claude/commands/) - Custom Claude Code commands for rule generation
