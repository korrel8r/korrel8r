// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package log

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/loki"
	"github.com/korrel8r/korrel8r/internal/pkg/text"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/stretchr/testify/assert"
)

func TestDomain(t *testing.T) {
	t.Run("Domain name", func(t *testing.T) {
		assert.Equal(t, "log", Domain.Name())
	})

	t.Run("Domain description", func(t *testing.T) {
		description := Domain.Description()
		want := "application, infrastructure, and audit logs."
		assert.Equal(t, want, text.Summary(description))
		assert.Contains(t, description, want)
	})

	t.Run("Domain classes", func(t *testing.T) {
		assert.ElementsMatch(t, []Class{Application, Infrastructure, Audit}, Domain.Classes())
		for _, c := range Domain.Classes() {
			assert.Equal(t, c, Domain.Class(c.Name()))
		}
	})
}

func TestDomainQuery(t *testing.T) {
	d := &domain{Domain.Domain}

	t.Run("Valid query", func(t *testing.T) {
		query, err := d.Query("log:application:{}")
		assert.NoError(t, err)
		assert.NotNil(t, query)
	})

	t.Run("Invalid query", func(t *testing.T) {
		query, err := d.Query("invalid")
		assert.Error(t, err)
		assert.Nil(t, query)
	})
}

func TestClass(t *testing.T) {
	tests := []struct {
		name     string
		class    Class
		expected string
	}{
		{"Application", Application, Application.Name()},
		{"Infrastructure", Infrastructure, Infrastructure.Name()},
		{"Audit", Audit, Audit.Name()},
		{"Custom", Class("custom"), "custom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Run("Domain", func(t *testing.T) {
				assert.Equal(t, Domain, tt.class.Domain())
			})

			t.Run("Name", func(t *testing.T) {
				assert.Equal(t, tt.expected, tt.class.Name())
			})

			t.Run("String", func(t *testing.T) {
				expected := "log:" + tt.expected
				assert.Equal(t, expected, tt.class.String())
			})
		})
	}
}

func TestClassUnmarshal(t *testing.T) {
	class := Application

	t.Run("Valid JSON", func(t *testing.T) {
		data := `{"body": "test message", "timestamp": "2023-01-01T00:00:00Z"}`
		obj, err := class.Unmarshal([]byte(data))
		assert.NoError(t, err)
		assert.NotNil(t, obj)

		logObj, ok := obj.(Object)
		assert.True(t, ok)
		assert.Equal(t, "test message", logObj["body"])
		assert.Equal(t, "2023-01-01T00:00:00Z", logObj["timestamp"])
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		data := `invalid json`
		obj, err := class.Unmarshal([]byte(data))
		assert.Error(t, err)
		assert.Nil(t, obj)
	})
}

func TestClassPreview(t *testing.T) {
	class := Application

	t.Run("Valid Object", func(t *testing.T) {
		obj := Object{
			AttrBody: "test log message",
		}
		preview := class.Preview(obj)
		assert.Equal(t, "test log message", preview)
	})

	t.Run("Object without body", func(t *testing.T) {
		obj := Object{
			"timestamp": "2023-01-01T00:00:00Z",
		}
		preview := class.Preview(obj)
		assert.Equal(t, "", preview)
	})

	t.Run("Non-Object", func(t *testing.T) {
		preview := class.Preview("not an object")
		assert.Equal(t, "", preview)
	})

	t.Run("Nil object", func(t *testing.T) {
		preview := class.Preview(nil)
		assert.Equal(t, "", preview)
	})
}

func TestNewObject(t *testing.T) {
	testTime := time.Now()

	t.Run("Complete loki log", func(t *testing.T) {
		lokiLog := &loki.Log{
			Time: testTime,
			Body: "test log body",
			Labels: map[string]string{
				"app":       "test-app",
				"namespace": "default",
			},
			Metadata: map[string]string{
				"source": "container",
				"level":  "info",
			},
		}

		obj := newObject(lokiLog)

		assert.Equal(t, "test log body", obj[AttrBody])
		assert.Equal(t, "test-app", obj["app"])
		assert.Equal(t, "default", obj["namespace"])
		assert.Equal(t, "container", obj["source"])
		assert.Equal(t, "info", obj["level"])
	})

	t.Run("With _timestamp field", func(t *testing.T) {
		lokiLog := &loki.Log{
			Time: testTime,
			Body: "test log body",
			Metadata: map[string]string{
				Attr_Timestamp: "2023-01-01T00:00:00Z",
			},
		}

		obj := newObject(lokiLog)

		assert.Equal(t, "2023-01-01T00:00:00Z", obj[AttrTimestamp])
		assert.Equal(t, "2023-01-01T00:00:00Z", obj[Attr_Timestamp])
	})
}

func TestQuery(t *testing.T) {
	t.Run("LogQL query", func(t *testing.T) {
		class := Application
		logQL := `{app="test"}`

		query := &Query{
			class: class,
			logQL: logQL,
		}

		assert.Equal(t, class, query.Class())
		assert.Equal(t, logQL, query.Data())
		assert.Contains(t, query.String(), "log:application")
	})

	t.Run("Direct query with logQL set", func(t *testing.T) {
		class := Infrastructure
		containerSelector := &ContainerSelector{
			Selector: k8s.Selector{
				Name:      "test-pod",
				Namespace: "default",
			},
			Containers: []string{"app", "sidecar"},
		}

		query := &Query{
			class:  class,
			direct: containerSelector,
			logQL:  containerSelector.ViaqLogQL(),
		}

		assert.Equal(t, class, query.Class())
		assert.Equal(t, containerSelector.ViaqLogQL(), query.Data())
	})

	t.Run("Direct query without logQL falls back to JSON", func(t *testing.T) {
		class := Infrastructure
		containerSelector := &ContainerSelector{
			Selector: k8s.Selector{
				Name:      "test-pod",
				Namespace: "default",
			},
			Containers: []string{"app", "sidecar"},
		}

		query := &Query{
			class:  class,
			direct: containerSelector,
		}

		assert.Equal(t, class, query.Class())
		data := query.Data()
		var selector ContainerSelector
		err := json.Unmarshal([]byte(data), &selector)
		assert.NoError(t, err)
		assert.Equal(t, "test-pod", selector.Name)
		assert.Equal(t, "default", selector.Namespace)
		assert.ElementsMatch(t, []string{"app", "sidecar"}, selector.Containers)
	})
}

func TestNewQuery(t *testing.T) {
	t.Run("Valid LogQL query", func(t *testing.T) {
		queryStr := "log:application:{app=\"test\"}"

		query, err := NewQuery(queryStr)
		assert.NoError(t, err)
		assert.NotNil(t, query)
		assert.Equal(t, Application, query.class)
		assert.Equal(t, `{app="test"}`, query.logQL)
		assert.Nil(t, query.direct)
	})

	t.Run("Valid direct query with JSON selector", func(t *testing.T) {
		selector := ContainerSelector{
			Selector: k8s.Selector{
				Name:      "test-pod",
				Namespace: "default",
			},
			Containers: []string{"app"},
		}
		selectorJSON, _ := json.Marshal(selector)
		queryStr := "log:infrastructure:" + string(selectorJSON)

		query, err := NewQuery(queryStr)
		assert.NoError(t, err)
		assert.NotNil(t, query)
		assert.Equal(t, Infrastructure, query.class)
		assert.NotNil(t, query.direct)
		assert.Equal(t, "test-pod", query.direct.Name)
		assert.Equal(t, "default", query.direct.Namespace)
		assert.ElementsMatch(t, []string{"app"}, query.direct.Containers)
	})

	t.Run("Invalid query format", func(t *testing.T) {
		queryStr := "invalid:format"

		query, err := NewQuery(queryStr)
		assert.Error(t, err)
		assert.Nil(t, query)
	})

	t.Run("Invalid class", func(t *testing.T) {
		queryStr := "log:invalid:{}"

		query, err := NewQuery(queryStr)
		assert.Error(t, err)
		assert.Nil(t, query)
	})
}

func TestContainerSelector_LogQL(t *testing.T) {
	tests := []struct {
		name             string
		selector         ContainerSelector
		expected         string
		expectedContains []string
		useContainsCheck bool
	}{
		{
			name:     "Empty selector",
			selector: ContainerSelector{},
			expected: "}|json",
		},
		{
			name: "Namespace only",
			selector: ContainerSelector{
				Selector: k8s.Selector{
					Namespace: "default",
				},
			},
			expected: `{kubernetes_namespace_name="default"}|json`,
		},
		{
			name: "Single container",
			selector: ContainerSelector{
				Containers: []string{"app"},
			},
			expected: `{kubernetes_container_name=~"app"}|json`,
		},
		{
			name: "Multiple containers",
			selector: ContainerSelector{
				Containers: []string{"app", "sidecar", "init"},
			},
			expected: `{kubernetes_container_name=~"app|sidecar|init"}|json`,
		},
		{
			name: "Namespace and pod name",
			selector: ContainerSelector{
				Selector: k8s.Selector{
					Namespace: "production",
					Name:      "web-server",
				},
			},
			expected: `{kubernetes_namespace_name="production",kubernetes_pod_name="web-server"}|json`,
		},
		{
			name: "Namespace and containers",
			selector: ContainerSelector{
				Selector: k8s.Selector{
					Namespace: "staging",
				},
				Containers: []string{"web", "db"},
			},
			expected: `{kubernetes_namespace_name="staging",kubernetes_container_name=~"web|db"}|json`,
		},
		{
			name: "Single label",
			selector: ContainerSelector{
				Selector: k8s.Selector{
					Labels: map[string]string{
						"app": "nginx",
					},
				},
			},
			expected: `}|json|kubernetes_labels_app="nginx"`,
		},
		{
			name: "Multiple labels",
			selector: ContainerSelector{
				Selector: k8s.Selector{
					Labels: map[string]string{
						"app":     "nginx",
						"version": "1.20",
						"env":     "prod",
					},
				},
			},
			useContainsCheck: true,
			expectedContains: []string{
				"}|json",
				`kubernetes_labels_app="nginx"`,
				`kubernetes_labels_env="prod"`,
				`kubernetes_labels_version="1.20"`,
			},
		},
		{
			name: "Labels with special characters",
			selector: ContainerSelector{
				Selector: k8s.Selector{
					Labels: map[string]string{
						"app.kubernetes.io/name":    "nginx",
						"app.kubernetes.io/version": "1.20.1",
						"deployment-type":           "canary",
					},
				},
			},
			useContainsCheck: true,
			expectedContains: []string{
				"}|json",
				`kubernetes_labels_app_kubernetes_io_name="nginx"`,
				`kubernetes_labels_app_kubernetes_io_version="1.20.1"`,
				`kubernetes_labels_deployment_type="canary"`,
			},
		},
		{
			name: "All fields populated",
			selector: ContainerSelector{
				Selector: k8s.Selector{
					Namespace: "production",
					Name:      "web-app-12345",
					Labels: map[string]string{
						"app":        "web",
						"component":  "frontend",
						"tier":       "web",
						"release.id": "v1.2.3",
					},
				},
				Containers: []string{"web", "logging-agent"},
			},
			useContainsCheck: true,
			expectedContains: []string{
				`{kubernetes_namespace_name="production",kubernetes_pod_name="web-app-12345",kubernetes_container_name=~"web|logging-agent"}|json`,
				`kubernetes_labels_app="web"`,
				`kubernetes_labels_component="frontend"`,
				`kubernetes_labels_release_id="v1.2.3"`,
				`kubernetes_labels_tier="web"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.selector.ViaqLogQL()
			if tt.useContainsCheck {
				for _, expected := range tt.expectedContains {
					assert.Contains(t, result, expected)
				}
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestContainerSelector_OtelLogQL(t *testing.T) {
	tests := []struct {
		name     string
		selector ContainerSelector
		expected string
	}{
		{
			name:     "Empty selector",
			selector: ContainerSelector{},
			expected: "}",
		},
		{
			name: "Namespace only",
			selector: ContainerSelector{
				Selector: k8s.Selector{
					Namespace: "default",
				},
			},
			expected: `{k8s_namespace_name="default"}`,
		},
		{
			name: "Namespace and pod",
			selector: ContainerSelector{
				Selector: k8s.Selector{
					Namespace: "production",
					Name:      "web-server",
				},
			},
			expected: `{k8s_namespace_name="production",k8s_pod_name="web-server"}`,
		},
		{
			name: "Namespace with containers",
			selector: ContainerSelector{
				Selector: k8s.Selector{
					Namespace: "staging",
				},
				Containers: []string{"web", "db"},
			},
			expected: `{k8s_namespace_name="staging",k8s_container_name=~"web|db"}`,
		},
		{
			name: "Labels are ignored in OTEL LogQL",
			selector: ContainerSelector{
				Selector: k8s.Selector{
					Namespace: "ns",
					Labels:    map[string]string{"app": "web"},
				},
			},
			expected: `{k8s_namespace_name="ns"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.selector.OtelLogQL()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestQueryExpand(t *testing.T) {
	t.Run("Direct query expands to Viaq and OTEL", func(t *testing.T) {
		q := &Query{
			class: Application,
			direct: &ContainerSelector{
				Selector: k8s.Selector{
					Namespace: "myapp",
					Name:      "pod-1",
				},
			},
		}
		expanded := q.Expand()
		assert.Len(t, expanded, 2)

		viaq := expanded[0].(*Query)
		assert.Equal(t, Application, viaq.class)
		assert.Contains(t, viaq.logQL, "kubernetes_namespace_name")
		assert.NotNil(t, viaq.direct, "Viaq query should retain direct selector")

		otel := expanded[1].(*Query)
		assert.Equal(t, Application, otel.class)
		assert.Contains(t, otel.logQL, "k8s_namespace_name")
		assert.Nil(t, otel.direct, "OTEL query should have nil direct selector")
	})

	t.Run("LogQL-only query does not expand", func(t *testing.T) {
		q := &Query{
			class: Application,
			logQL: `{app="test"}`,
		}
		expanded := q.Expand()
		assert.Nil(t, expanded)
	})

	t.Run("Label-only selector expands to Viaq only", func(t *testing.T) {
		q := &Query{
			class: Application,
			direct: &ContainerSelector{
				Selector: k8s.Selector{
					Namespace: "ns",
					Labels:    map[string]string{"app": "web"},
				},
			},
		}
		expanded := q.Expand()
		assert.Len(t, expanded, 1, "label-only selector should not produce OTEL variant")

		viaqData := expanded[0].Data()
		assert.Contains(t, viaqData, `kubernetes_namespace_name="ns"`)
		assert.Contains(t, viaqData, `kubernetes_labels_app="web"`)
	})

	t.Run("Named selector expands to Viaq and OTEL", func(t *testing.T) {
		q := &Query{
			class: Application,
			direct: &ContainerSelector{
				Selector: k8s.Selector{
					Namespace: "ns",
					Name:      "my-pod",
				},
			},
		}
		expanded := q.Expand()
		assert.Len(t, expanded, 2, "named selector should produce both variants")

		viaqData := expanded[0].Data()
		assert.Contains(t, viaqData, `kubernetes_namespace_name="ns"`)
		assert.Contains(t, viaqData, `kubernetes_pod_name="my-pod"`)

		otelData := expanded[1].Data()
		assert.Contains(t, otelData, `k8s_namespace_name="ns"`)
		assert.Contains(t, otelData, `k8s_pod_name="my-pod"`)
	})
}

func TestContainerSelector_IsContainerSelected(t *testing.T) {
	tests := []struct {
		name      string
		selector  ContainerSelector
		container string
		expected  bool
	}{
		{
			name:      "Empty containers list - any container selected",
			selector:  ContainerSelector{},
			container: "any-container",
			expected:  true,
		},
		{
			name: "Container in list",
			selector: ContainerSelector{
				Containers: []string{"app", "sidecar"},
			},
			container: "app",
			expected:  true,
		},
		{
			name: "Container not in list",
			selector: ContainerSelector{
				Containers: []string{"app", "sidecar"},
			},
			container: "unknown",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.selector.IsContainerSelected(tt.container)
			assert.Equal(t, tt.expected, result)
		})
	}
}
