// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package log

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/loki"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/stretchr/testify/assert"
)

func TestDomain(t *testing.T) {
	t.Run("Domain name", func(t *testing.T) {
		assert.Equal(t, "log", Domain.Name())
	})

	t.Run("Domain description", func(t *testing.T) {
		assert.Equal(t, "Records from container and node logs.", Domain.Description())
	})

	t.Run("Domain classes", func(t *testing.T) {
		classes := Domain.Classes()
		assert.Len(t, classes, 3)

		classNames := make([]string, len(classes))
		for i, c := range classes {
			classNames[i] = c.Name()
		}
		assert.ElementsMatch(t, []string{Application, Infrastructure, Audit}, classNames)
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
		{"Application", Class(Application), Application},
		{"Infrastructure", Class(Infrastructure), Infrastructure},
		{"Audit", Class(Audit), Audit},
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
	class := Class(Application)

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
	class := Class(Application)

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
}

func TestPreview(t *testing.T) {
	t.Run("Valid Object with body", func(t *testing.T) {
		obj := Object{
			AttrBody: "test log message",
			"other":  "field",
		}
		preview := Preview(obj)
		assert.Equal(t, "test log message", preview)
	})

	t.Run("Object without body", func(t *testing.T) {
		obj := Object{
			"timestamp": "2023-01-01T00:00:00Z",
		}
		preview := Preview(obj)
		assert.Equal(t, "", preview)
	})

	t.Run("Non-Object", func(t *testing.T) {
		preview := Preview("not an object")
		assert.Equal(t, "", preview)
	})

	t.Run("Nil object", func(t *testing.T) {
		preview := Preview(nil)
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
		class := Class(Application)
		logQL := `{app="test"}`

		query := &Query{
			class: class,
			logQL: logQL,
		}

		assert.Equal(t, class, query.Class())
		assert.Equal(t, logQL, query.Data())
		assert.Contains(t, query.String(), "log:application")
	})

	t.Run("Direct query", func(t *testing.T) {
		class := Class(Infrastructure)
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
			logQL:  containerSelector.LogQL(),
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
		assert.Equal(t, Class(Application), query.class)
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
		assert.Equal(t, Class(Infrastructure), query.class)
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

func TestConstants(t *testing.T) {
	t.Run("Class constants", func(t *testing.T) {
		assert.Equal(t, "application", Application)
		assert.Equal(t, "infrastructure", Infrastructure)
		assert.Equal(t, "audit", Audit)
	})

	t.Run("Store key constants", func(t *testing.T) {
		assert.Equal(t, "loki", StoreKeyLoki)
		assert.Equal(t, "lokiStack", StoreKeyLokiStack)
		assert.Equal(t, "direct", StoreKeyDirect)
	})

	t.Run("Attribute constants", func(t *testing.T) {
		assert.Equal(t, "timestamp", AttrTimestamp)
		assert.Equal(t, "_timestamp", Attr_Timestamp)
		assert.Equal(t, "body", AttrBody)
	})
}

func TestObject(t *testing.T) {
	t.Run("Object creation and access", func(t *testing.T) {
		obj := Object{
			AttrBody:       "test message",
			AttrTimestamp:  "2023-01-01T00:00:00Z",
			"custom_field": "custom_value",
		}

		assert.Equal(t, "test message", obj[AttrBody])
		assert.Equal(t, "2023-01-01T00:00:00Z", obj[AttrTimestamp])
		assert.Equal(t, "custom_value", obj["custom_field"])
	})

	t.Run("Empty object", func(t *testing.T) {
		obj := Object{}
		assert.Equal(t, "", obj[AttrBody])
		assert.Equal(t, "", obj["nonexistent"])
	})
}

// Additional tests for improved coverage

func TestDomainQueryMethod(t *testing.T) {
	d := &domain{Domain.Domain}

	t.Run("Query method delegates to NewQuery", func(t *testing.T) {
		query, err := d.Query("log:application:{}")
		assert.NoError(t, err)
		assert.NotNil(t, query)
		assert.Equal(t, Class(Application), query.Class())
	})

	t.Run("Query method returns error for invalid input", func(t *testing.T) {
		query, err := d.Query("invalid-query-format")
		assert.Error(t, err)
		assert.Nil(t, query)
	})
}

func TestClassUnmarshalAndPreview(t *testing.T) {
	class := Class(Application)

	t.Run("Unmarshal calls through to implementation", func(t *testing.T) {
		data := `{"body": "test", "level": "info"}`
		obj, err := class.Unmarshal([]byte(data))
		assert.NoError(t, err)
		assert.NotNil(t, obj)

		logObj, ok := obj.(Object)
		assert.True(t, ok)
		assert.Equal(t, "test", logObj["body"])
		assert.Equal(t, "info", logObj["level"])
	})

	t.Run("Preview calls Preview function", func(t *testing.T) {
		obj := Object{AttrBody: "test message"}
		preview := class.Preview(obj)
		assert.Equal(t, "test message", preview)
	})
}

func TestSafeLabel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Valid label", "valid_label_123", "valid_label_123"},
		{"Label with colon", "app:version", "app:version"},
		{"Label starting with number", "123invalid", "_23invalid"},
		{"Label with special chars", "app-name.service", "app_name_service"},
		{"Label with spaces", "my app", "my_app"},
		{"Empty string", "", ""},
		{"Only invalid chars", "-.@", "___"},
		{"Mixed valid/invalid", "app-1.2.3", "app_1_2_3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SafeLabel(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLogTypeForNamespace(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		expected  string
	}{
		{"Default namespace", "default", Infrastructure},
		{"Openshift namespace", "openshift", Infrastructure},
		{"Openshift with suffix", "openshift-cluster-version", Infrastructure},
		{"Kube namespace", "kube", Infrastructure},
		{"Kube with suffix", "kube-system", Infrastructure},
		{"Application namespace", "my-app", Application},
		{"User namespace", "user-namespace", Application},
		{"Empty namespace", "", Application},
		{"Random namespace", "random", Application},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := logTypeForNamespace(tt.namespace)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContainerSelector_LogQL(t *testing.T) {
	tests := []struct {
		name               string
		selector           ContainerSelector
		expected           string
		expectedContains   []string // For cases where label order is non-deterministic
		useContainsCheck   bool
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
			name: "Pod name only",
			selector: ContainerSelector{
				Selector: k8s.Selector{
					Name: "test-pod",
				},
			},
			expected: `{kubernetes_pod_name="test-pod"}|json`,
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
		{
			name: "Empty containers list",
			selector: ContainerSelector{
				Selector: k8s.Selector{
					Namespace: "test",
				},
				Containers: []string{},
			},
			expected: `{kubernetes_namespace_name="test"}|json`,
		},
		{
			name: "Container with regex special characters",
			selector: ContainerSelector{
				Containers: []string{"app-1.2", "sidecar[prod]", "init+container"},
			},
			expected: `{kubernetes_container_name=~"app-1.2|sidecar[prod]|init+container"}|json`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.selector.LogQL()
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
		{
			name: "Exact match required",
			selector: ContainerSelector{
				Containers: []string{"app"},
			},
			container: "app-extra",
			expected:  false,
		},
		{
			name: "Case sensitive",
			selector: ContainerSelector{
				Containers: []string{"App"},
			},
			container: "app",
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
