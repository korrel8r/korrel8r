Kubernetes resources.

### Classes

Each Kind of kubernetes resource is a class. Class names have the format:

```
k8s:KIND[.VERSION][.GROUP]
```

Missing VERSION implies "v1", if present VERSION must follow the [Kubernetes version patterns](<https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/#version-priority>). Missing GROUP implies the core group.

Examples:

```
k8s:Pod
k8s:Pod.v1
k8s:Deployment.apps
k8s:Deployment.v1.apps
k8s:Route.v1.route.openshift.io
```

### Object

A map of JSON kubernetes field names and Go values. Rule templates should use the JSON \(lowerCase\) field names, not the UpperCase Go field names.

### Query

JSON selector with the following fields:

- namespace: namespace containing the resource
- name: name of resource
- labels: label selector object for metadata labels \- \{ "label": "value", ... \}
- fields: [field selector object](<https://kubernetes.io/docs/concepts/overview/working-with-objects/field-selectors/>) \- \{ "field": "value", ... \}

Examples:

```
k8s:Pod.v1:{"namespace":"some-namespace", "name":"some-name"}
k8s:Deployment.v1:{"labels":{"app":"my-application"}, "namespace":"some-namespace" }
```

### Store

The k8s domain automatically connects to the currently logged\-in kubectl cluster. No additional configuration is needed.

```
stores:
    domain: k8s
```

### Field Selectors

Kubernetes defines [field selectors](<https://kubernetes.io/docs/concepts/overview/working-with-objects/field-selectors/>), similar to label selectors but acting on resource field values.

Not all fields are supported, the following are allowed as field\-selectors in a query.

All resources support the field metadata.name, the resource name. All namespaced resources also support metadata.namespace, the resource namespace.

Core Resources \(v1\):

- pods: metadata.name, metadata.namespace, spec.nodeName, spec.restartPolicy, spec.schedulerName, spec.serviceAccountName, spec.hostNetwork, status.phase, status.podIP, status.nominatedNodeName
- events: metadata.name, metadata.namespace, involvedObject.kind, involvedObject.namespace, involvedObject.name, involvedObject.uid, involvedObject.apiVersion, involvedObject.resourceVersion, involvedObject.fieldPath, reason, reportingComponent, source, type
- namespaces: metadata.name, status.phase
- nodes: metadata.name, spec.unschedulable
- secrets: metadata.name, metadata.namespace, type
- services: metadata.name, metadata.namespace, spec.clusterIP
- replicationcontrollers: metadata.name, metadata.namespace, status.replicas

Other Built\-in Resources:

- events.events.k8s.io: metadata.name, metadata.namespace, reason, reportingController, regarding.kind, regarding.namespace, regarding.name, regarding.uid, regarding.apiVersion, regarding.resourceVersion, regarding.fieldPath, type
- jobs.batch: metadata.name, metadata.namespace, status.successful
- certificatesigningrequests.certificates.k8s.io: metadata.name, spec.signerName
- resourceslices.resource.k8s.io: metadata.name, spec.nodeName

Since K8s 1.30\+, CRDs can define custom selectableFields. None of the OpenShift observability resources do this.

### Template Functions

The following template functions are available to rules.

```
k8sClass
	Takes string arguments (apiVersion, kind).
	Returns the korrel8r.Class implied by the arguments, or an error.

k8sIsNamespaced
	Takes a k8s Class argument, returns true if the class is a namespace-scoped resource.

k8sHealthStatus
	Takes a k8s Object, evaluates its health using the kube-health library.
	Returns "Error", "Warning", or "" for healthy/unknown objects.
	Analyzes observed generation and standard Kubernetes conditions.
```

