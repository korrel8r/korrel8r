// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rules_test

import (
	"testing"

	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/domains/log"
)

func TestK8sRules(t *testing.T) {
	for _, x := range []ruleTest{
		{
			rule:  "LogToPod",
			start: log.Object{"kubernetes_namespace_name": "foo", "kubernetes_pod_name": "bar", "message": "hello"},
			want:  []string{`k8s:Pod.v1:{"namespace":"foo","name":"bar"}`},
		},
		{
			rule:  "LogToPod",
			start: log.Object{"kubernetes_namespace_name": "default", "kubernetes_pod_name": "baz", "message": "bye"},
			want:  []string{`k8s:Pod.v1:{"namespace":"default","name":"baz"}`},
		},
		{
			rule: "SelectorToPods",
			start: k8s.Object{
				"kind": "Deployment", "apiVersion": "v1",
				"metadata": k8s.Object{"name": "x", "namespace": "ns"},
				"spec": k8s.Object{
					"selector": k8s.Object{"matchLabels": k8s.Object{"test": "testme"}},
					"template": k8s.Object{"metadata": k8s.Object{"name": "x", "namespace": "ns"}}},
			},
			want: []string{`k8s:Pod.v1:{"namespace":"ns","labels":{"test":"testme"}}`},
		},
		{
			rule: "ServiceToPods",
			start: newK8s("Service", "ns", "my-svc", k8s.Object{
				"spec": k8s.Object{
					"selector": k8s.Object{"app": "web"},
				},
			}),
			want: []string{`k8s:Pod.v1:{"namespace":"ns","labels":{"app":"web"}}`},
		},
		{
			rule:  "EventToAll",
			start: k8sEvent(newK8s("Pod", "aNamespace", "foo", nil), "a"),
			want:  []string{`k8s:Pod.v1:{"namespace":"aNamespace","name":"foo"}`},
		},
		{
			rule:  "Event2ToAll",
			start: k8sEvent2(newK8s("Pod", "aNamespace", "foo", nil), "a"),
			want:  []string{`k8s:Pod.v1:{"namespace":"aNamespace","name":"foo"}`},
		},
		{
			rule:  "AllToEvent",
			start: newK8s("Pod", "aNamespace", "foo", nil),
			want:  []string{`k8s:Event.v1:{"namespace":"aNamespace","fields":{"involvedObject.apiVersion":"v1","involvedObject.kind":"Pod","involvedObject.name":"foo","involvedObject.namespace":"aNamespace"}}`},
		},
		{
			rule:  "PodToAlert",
			start: newK8s("Pod", "aNamespace", "foo", nil),
			want:  []string{`alert:alert:{"namespace":"aNamespace","pod":"foo"}`},
		},
		{
			rule:  "PodToLokiAlert",
			start: newK8s("Pod", "aNamespace", "foo", nil),
			want:  []string{`alert:alert:{"namespace":"aNamespace","kubernetes_pod_name":"foo"}`},
		},
		{
			rule:  "DeploymentToAlert",
			start: newK8s("Deployment.apps", "aNamespace", "foo", nil),
			want:  []string{`alert:alert:{"namespace":"aNamespace","deployment":"foo"}`},
		},
		{
			rule:  "DaemonSetToAlert",
			start: newK8s("DaemonSet.apps", "aNamespace", "foo", nil),
			want:  []string{`alert:alert:{"namespace":"aNamespace","daemonset":"foo"}`},
		},
		{
			rule:  "StatefulSetToAlert",
			start: newK8s("StatefulSet.apps", "aNamespace", "foo", nil),
			want:  []string{`alert:alert:{"namespace":"aNamespace","statefulset":"foo"}`},
		},
		{
			rule:  "PodDisruptionBudgetToAlert",
			start: newK8s("PodDisruptionBudget.policy", "aNamespace", "foo", nil),
			want:  []string{`alert:alert:{"namespace":"aNamespace","poddisruptionbudget":"foo"}`},
		},
		{
			rule: "DependentToOwner",
			start: newK8s("Pod", "aNamespace", "foo", k8s.Object{
				"metadata": k8s.Object{
					"ownerReferences": []k8s.Object{{
						"name":       "owner",
						"kind":       "Deployment", // Namespace scoped owner
						"apiVersion": "apps/v1",
					}}},
			}),
			want: []string{`k8s:Deployment.v1.apps:{"namespace":"aNamespace","name":"owner"}`},
		},
		{
			rule: "DependentToOwner",
			start: newK8s("Pod", "aNamespace", "foo", k8s.Object{
				"metadata": k8s.Object{
					"ownerReferences": []k8s.Object{{
						"name":       "owner",
						"kind":       "PersistentVolume", // Cluster scoped owner
						"apiVersion": "v1",
					}}},
			}),
			want: []string{`k8s:PersistentVolume.v1:{"name":"owner"}`},
		},
		{
			rule: "PVCToPV",
			start: newK8s("PersistentVolumeClaim", "ns", "pvc-1", k8s.Object{
				"spec": k8s.Object{
					"volumeName": "pv-123",
				},
			}),
			want: []string{`k8s:PersistentVolume.v1:{"name":"pv-123"}`},
		},
		{
			rule: "PVToStorageClass",
			start: newK8s("PersistentVolume", "", "pv-123", k8s.Object{
				"spec": k8s.Object{
					"storageClassName": "sc-1",
				},
			}),
			want: []string{`k8s:StorageClass.v1.storage.k8s.io:{"name":"sc-1"}`},
		},
		{
			rule: "PVCToStorageClass",
			start: newK8s("PersistentVolumeClaim", "ns", "pvc-1", k8s.Object{
				"spec": k8s.Object{
					"storageClassName": "sc-1",
				},
			}),
			want: []string{`k8s:StorageClass.v1.storage.k8s.io:{"name":"sc-1"}`},
		},
		{
			rule: "PodToPVC",
			start: newK8s("Pod", "ns", "my-pod", k8s.Object{
				"spec": k8s.Object{
					"volumes": []k8s.Object{
						{"name": "data", "persistentVolumeClaim": k8s.Object{"claimName": "my-pvc"}},
						{"name": "config", "configMap": k8s.Object{"name": "my-cm"}},
					},
				},
			}),
			want: []string{`k8s:PersistentVolumeClaim.v1:{"namespace":"ns","name":"my-pvc"}`},
		},
		{
			rule: "PodToConfigMap",
			start: newK8s("Pod", "ns", "my-pod", k8s.Object{
				"spec": k8s.Object{
					"volumes": []k8s.Object{
						{"name": "config", "configMap": k8s.Object{"name": "app-config"}},
						{"name": "data", "emptyDir": k8s.Object{}},
					},
				},
			}),
			want: []string{`k8s:ConfigMap.v1:{"namespace":"ns","name":"app-config"}`},
		},
		{
			rule: "PodToSecret",
			start: newK8s("Pod", "ns", "my-pod", k8s.Object{
				"spec": k8s.Object{
					"volumes": []k8s.Object{
						{"name": "certs", "secret": k8s.Object{"secretName": "tls-secret"}},
						{"name": "keys", "secret": k8s.Object{"secretName": "api-keys"}},
					},
				},
			}),
			want: []string{
				`k8s:Secret.v1:{"namespace":"ns","name":"tls-secret"}`,
				`k8s:Secret.v1:{"namespace":"ns","name":"api-keys"}`,
			},
		},
		{
			rule: "PodToServiceAccount",
			start: newK8s("Pod", "ns", "my-pod", k8s.Object{
				"spec": k8s.Object{
					"serviceAccountName": "my-sa",
				},
			}),
			want: []string{`k8s:ServiceAccount.v1:{"namespace":"ns","name":"my-sa"}`},
		},
		{
			rule: "PVToPVC",
			start: newK8s("PersistentVolume", "", "pv-123", k8s.Object{
				"spec": k8s.Object{
					"claimRef": k8s.Object{
						"namespace": "ns",
						"name":      "my-pvc",
					},
				},
			}),
			want: []string{`k8s:PersistentVolumeClaim.v1:{"namespace":"ns","name":"my-pvc"}`},
		},
		{
			rule: "IngressToService",
			start: newK8s("Ingress.networking.k8s.io", "ns", "my-ingress", k8s.Object{
				"spec": k8s.Object{
					"rules": []k8s.Object{
						{"http": k8s.Object{
							"paths": []k8s.Object{
								{"backend": k8s.Object{"service": k8s.Object{"name": "web-svc"}}},
								{"backend": k8s.Object{"service": k8s.Object{"name": "api-svc"}}},
							},
						}},
					},
				},
			}),
			want: []string{
				`k8s:Service.v1:{"namespace":"ns","name":"web-svc"}`,
				`k8s:Service.v1:{"namespace":"ns","name":"api-svc"}`,
			},
		},
		{
			rule: "HPAToTarget",
			start: newK8s("HorizontalPodAutoscaler.autoscaling", "ns", "my-hpa", k8s.Object{
				"spec": k8s.Object{
					"scaleTargetRef": k8s.Object{
						"apiVersion": "apps/v1",
						"kind":       "Deployment",
						"name":       "my-deploy",
					},
				},
			}),
			want: []string{`k8s:Deployment.v1.apps:{"namespace":"ns","name":"my-deploy"}`},
		},
		{
			rule:  "ServiceToEndpointSlice",
			start: newK8s("Service", "ns", "my-svc", nil),
			want:  []string{`k8s:EndpointSlice.v1.discovery.k8s.io:{"namespace":"ns","labels":{"kubernetes.io/service-name":"my-svc"}}`},
		},
		{
			rule:  "ServiceToEndpoints",
			start: newK8s("Service", "ns", "my-svc", nil),
			want:  []string{`k8s:Endpoints.v1:{"namespace":"ns","name":"my-svc"}`},
		},
		{
			rule: "EndpointSliceToService",
			start: newK8s("EndpointSlice.discovery.k8s.io", "ns", "my-svc-abc12", k8s.Object{
				"metadata": k8s.Object{
					"labels": k8s.Object{
						"kubernetes.io/service-name": "my-svc",
					},
				},
			}),
			want: []string{`k8s:Service.v1:{"namespace":"ns","name":"my-svc"}`},
		},
		{
			rule: "CSVToCRD",
			start: newK8s("ClusterServiceVersion.operators.coreos.com", "operators", "test-operator.v0.1.0", k8s.Object{
				"spec": k8s.Object{
					"customresourcedefinitions": k8s.Object{
						"owned": []k8s.Object{
							{"name": "things.test.io"},
							{"name": "items.test.io"},
						},
					},
				},
			}),
			want: []string{
				`k8s:CustomResourceDefinition.v1.apiextensions.k8s.io:{"name":"things.test.io"}`,
				`k8s:CustomResourceDefinition.v1.apiextensions.k8s.io:{"name":"items.test.io"}`,
			},
		},
		{
			rule: "CRDToInstances",
			start: newK8s("CustomResourceDefinition.apiextensions.k8s.io", "", "nodes.config.openshift.io", k8s.Object{
				"spec": k8s.Object{
					"group":    "config.openshift.io",
					"names":    k8s.Object{"kind": "Node"},
					"versions": []k8s.Object{{"name": "v1"}},
				},
			}),
			want: []string{`k8s:Node.v1.config.openshift.io:{}`},
		},
		{
			rule: "SubscriptionToCSV",
			start: newK8s("Subscription.v1alpha1.operators.coreos.com", "foo", "bar", k8s.Object{
				"spec": k8s.Object{
					"channel":         "stable-6.3",
					"name":            "cluster-logging",
					"source":          "redhat-operators",
					"sourceNamespace": "openshift-marketplace",
				},
				"status": k8s.Object{
					"currentCSV": "blah",
				},
			}),
			want: []string{`k8s:ClusterServiceVersion.v1alpha1.operators.coreos.com:{"name":"blah"}`},
		},
	} {
		x.Run(t)
	}
}

func TestK8sStatusRules(t *testing.T) {
	for _, x := range []statusRuleTest{
		{
			rule:  "HasFinalizer",
			class: "Pod",
			start: newK8s("Pod", "ns", "pod1", k8s.Object{
				"metadata": k8s.Object{
					"finalizers": []any{"kubernetes.io/pv-protection"},
				},
			}),
			want: []string{"Finalizer"},
		},
		{
			rule:  "HasFinalizer",
			class: "Pod",
			start: newK8s("Pod", "ns", "pod2", nil),
			want:  nil,
		},
		{
			rule:  "EventType",
			class: "Event.v1",
			start: newK8s("Event.v1", "ns", "evt1", k8s.Object{
				"type": "Warning",
			}),
			want: []string{"Warning"},
		},
		{
			rule:  "EventType",
			class: "Event.v1",
			start: newK8s("Event.v1", "ns", "evt2", k8s.Object{
				"type": "Normal",
			}),
			want: nil,
		},
		{
			rule:  "EventType",
			class: "Event.v1",
			start: newK8s("Event.v1", "ns", "evt3", nil),
			want:  nil,
		},
		{
			rule:  "HealthStatus",
			class: "Pod",
			start: newK8s("Pod", "ns", "unhealthy-pod", k8s.Object{
				"status": k8s.Object{
					"conditions": []any{
						k8s.Object{"type": "Ready", "status": "False"},
					},
				},
			}),
			want: []string{"Error"},
		},
		{
			rule:  "HealthStatus",
			class: "Node",
			start: newK8s("Node", "", "bad-node", k8s.Object{
				"status": k8s.Object{
					"conditions": []any{
						k8s.Object{"type": "Ready", "status": "True"},
						k8s.Object{"type": "MemoryPressure", "status": "True"},
					},
				},
			}),
			want: []string{"Warning"},
		},
		{
			rule:  "HealthStatus",
			class: "Pod",
			start: newK8s("Pod", "ns", "healthy-pod", k8s.Object{
				"status": k8s.Object{
					"conditions": []any{
						k8s.Object{"type": "Ready", "status": "True"},
					},
				},
			}),
			want: nil,
		},
		{
			rule:  "HealthStatus",
			class: "Pod",
			start: newK8s("Pod", "ns", "no-status", nil),
			want:  nil,
		},
	} {
		x.Run(t)
	}
}
