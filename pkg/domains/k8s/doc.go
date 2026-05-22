// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package k8s is a korrel8r domain for Kubernetes resources.
//
// # Classes
//
// Each Kind of kubernetes resource is a class. Class names have the format:
//
//	k8s:KIND[.VERSION][.GROUP]
//
// Missing VERSION implies "v1", if present VERSION must follow the
// [Kubernetes version patterns].
// Missing GROUP implies the core group.
//
// Examples:
//
//	k8s:Pod
//	k8s:Pod.v1
//	k8s:Deployment.apps
//	k8s:Deployment.v1.apps
//	k8s:Route.v1.route.openshift.io
//
// # Object
//
// A map of JSON kubernetes field names and Go values.
// Rule templates should use the JSON (lowerCase) field names, not the UpperCase Go field names.
//
// # Query
//
// JSON selector with the following fields:
//   - namespace: namespace containing the resource
//   - name: name of resource
//   - labels: label selector object for metadata labels - { "label": "value", ... }
//   - fields: [field selector object] - { "field": "value", ... }
//
// Examples:
//
//	k8s:Pod.v1:{"namespace":"some-namespace", "name":"some-name"}
//	k8s:Deployment.v1:{"labels":{"app":"my-application"}, "namespace":"some-namespace" }
//
// # Store
//
// The k8s domain automatically connects to the currently logged-in kubectl cluster.
// No additional configuration is needed.
//
//	stores:
//	    domain: k8s
//
// [Kubernetes version patterns]: https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/#version-priority
// [field selector object]: https://kubernetes.io/docs/concepts/overview/working-with-objects/field-selectors/
package k8s
