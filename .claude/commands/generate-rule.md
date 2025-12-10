---
description: Generate a new Korrel8r correlation rule
argument-hint: [optional-domain]
---

Help me generate a new Korrel8r correlation rule file.

## Context
Korrel8r rules define correlations between different observability domains (k8s resources, logs, metrics, alerts, traces, etc.). Rules are stored as YAML files in `etc/korrel8r/rules/`.

## Rule File Structure
```yaml
aliases:  # Optional: Define reusable class groups
  - name: <alias-name>
    domain: <domain>
    classes: [<class1>, <class2>, ...]

rules:
  - name: <RuleName>
    start:
      domain: <source-domain>
      classes: [<source-classes>]  # Optional: specific classes
    goal:
      domain: <target-domain>
      classes: [<target-classes>]  # Optional: specific classes
    result:
      query: |-
        <query-template>
```

## Common Domains
- `k8s`: Kubernetes resources (Pod, Deployment.apps, Node, Event.v1, etc.)
- `log`: Application logs (log types vary by namespace)
- `metric`: Prometheus metrics
- `alert`: Alertmanager alerts
- `trace`: Distributed traces
- `netflow`: Network flow data
- `incident`: Incident management

## Query Template Functions
Available Go template functions:
- `{{.metadata.namespace}}`, `{{.metadata.name}}` - Access K8s object fields
- `{{mustToJson .}}` - Convert to JSON
- `{{k8sClass .apiVersion .kind}}` - Generate K8s class name
- `{{k8sIsNamespaced $class}}` - Check if class is namespaced
- `{{lower .kind}}` - Lowercase string
- `{{with .field}}...{{end}}` - Conditional rendering
- `{{range .array}}...{{end}}` - Iterate arrays
- `{{logTypeForNamespace .metadata.namespace}}` - Get log type for namespace

## Example Rules
**K8s Pod to Logs:**
```yaml
- name: PodToLogs
  start:
    domain: k8s
    classes: [Pod]
  goal:
    domain: log
  result:
    query: |-
      log:{{logTypeForNamespace .metadata.namespace}}:{"namespace":"{{.metadata.namespace}}","name":"{{.metadata.name}}"}
```

**Alert to K8s Deployment:**
```yaml
- name: AlertToDeployment
  start:
    domain: alert
  goal:
    domain: k8s
    classes: [Deployment.apps]
  result:
    query: |-
      k8s:Deployment.v1.apps:{namespace: "{{.Labels.namespace}}", name: "{{.Labels.deployment}}"}
```

**K8s Resource to Metrics:**
```yaml
- name: AllToMetric
  start:
    domain: k8s
  goal:
    domain: metric
  result:
    query: |-
      metric:metric:{namespace="{{.metadata.namespace}}",{{lower .kind}}="{{.metadata.name}}"}
```

## Interactive Generation
Please ask me:
1. What is the **source domain** and **source classes** (if specific)?
2. What is the **target domain** and **target classes** (if specific)?
3. What **fields or labels** from the source should map to the target query?
4. What should we **name this rule**?
5. Should this go in an **existing rule file** (k8s.yaml, log.yaml, alert.yaml, etc.) or a **new file**?

Then generate the complete rule definition and offer to create/update the appropriate YAML file in `etc/korrel8r/rules/`.
