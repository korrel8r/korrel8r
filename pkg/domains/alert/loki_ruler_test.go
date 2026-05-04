// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package alert

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetLokiRulesForTenant tests querying Loki Ruler for a specific tenant
func TestGetLokiRulesForTenant(t *testing.T) {
	// Create a mock Loki Ruler server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request path
		expectedPath := "/api/logs/v1/application/loki/api/v1/rules"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		// Return mock Loki rules response as raw JSON
		// Use raw JSON to avoid encoding issues with prometheus types
		responseJSON := `{
			"status": "success",
			"data": {
				"groups": [{
					"name": "devAppAlert",
					"file": "dev-workload-alerts",
					"rules": [{
						"type": "alerting",
						"name": "DevAppLogVolumeIsHigh",
						"query": "count_over_time({kubernetes_namespace_name=\"test-namespace\"}[2m]) > 10",
						"state": "firing",
						"alerts": [{
							"labels": {
								"alertname": "DevAppLogVolumeIsHigh",
								"namespace": "test-namespace",
								"severity": "info"
							},
							"annotations": {
								"description": "My application has high amount of logs."
							},
							"state": "firing",
							"activeAt": "2024-01-01T00:00:00Z",
							"value": "150"
						}]
					}]
				}]
			}
		}`

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responseJSON))
	}))
	defer server.Close()

	// Create store with mock server URL
	lokiRulerURL, _ := url.Parse(server.URL)
	store := &Store{
		lokiRulerURL: lokiRulerURL,
		httpClient:   server.Client(),
	}

	// Test querying application tenant
	result, err := store.getLokiRulesForTenant(context.Background(), "application", nil)
	require.NoError(t, err)
	assert.Len(t, result.Groups, 1, "Should have one rule group")
	assert.Equal(t, "devAppAlert", result.Groups[0].Name)

	// Verify alert details
	require.Len(t, result.Groups[0].Rules, 1, "Should have one rule")
	alertingRule, ok := result.Groups[0].Rules[0].(v1.AlertingRule)
	require.True(t, ok, "Rule should be an AlertingRule")
	assert.Equal(t, "DevAppLogVolumeIsHigh", alertingRule.Name)
	assert.Len(t, alertingRule.Alerts, 1)
	assert.Equal(t, model.LabelValue("DevAppLogVolumeIsHigh"), alertingRule.Alerts[0].Labels["alertname"])
}

// TestGetLokiRulesForTenant_WithNamespaceFilter tests namespace filtering
func TestGetLokiRulesForTenant_WithNamespaceFilter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify namespace query parameters
		namespaces := r.URL.Query()["namespace"]
		assert.Contains(t, namespaces, "test-ns-1")
		assert.Contains(t, namespaces, "test-ns-2")

		responseJSON := `{"status": "success", "data": {"groups": []}}`
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responseJSON))
	}))
	defer server.Close()

	lokiRulerURL, _ := url.Parse(server.URL)
	store := &Store{
		lokiRulerURL: lokiRulerURL,
		httpClient:   server.Client(),
	}

	namespaces := map[string]bool{"test-ns-1": true, "test-ns-2": true}
	_, err := store.getLokiRulesForTenant(context.Background(), "application", namespaces)
	require.NoError(t, err)
}

// TestGetLokiRulesForTenant_Error tests error handling
func TestGetLokiRulesForTenant_Error(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		response       interface{}
		expectedErrMsg string
	}{
		{
			name:           "non-OK status code",
			statusCode:     http.StatusNotFound,
			response:       nil,
			expectedErrMsg: "loki ruler request returned status 404",
		},
		{
			name:       "non-success status in response",
			statusCode: http.StatusOK,
			response: struct {
				Status string `json:"status"`
			}{
				Status: "error",
			},
			expectedErrMsg: "loki ruler API returned status: error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				if tt.response != nil {
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(tt.response)
				}
			}))
			defer server.Close()

			lokiRulerURL, _ := url.Parse(server.URL)
			store := &Store{
				lokiRulerURL: lokiRulerURL,
				httpClient:   server.Client(),
			}

			_, err := store.getLokiRulesForTenant(context.Background(), "application", nil)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErrMsg)
		})
	}
}

// TestGetLokiRules tests querying all Loki tenants
func TestGetLokiRules(t *testing.T) {
	// Track which tenants were queried
	queriedTenants := make(map[string]int)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract tenant from path: /api/logs/v1/{tenant}/loki/api/v1/rules
		path := r.URL.Path
		var tenant string
		switch path {
		case "/api/logs/v1/application/loki/api/v1/rules":
			tenant = "application"
		case "/api/logs/v1/infrastructure/loki/api/v1/rules":
			tenant = "infrastructure"
		case "/api/logs/v1/audit/loki/api/v1/rules":
			tenant = "audit"
		default:
			t.Errorf("Unexpected path: %s", path)
			return
		}

		queriedTenants[tenant]++

		// Return different alerts for each tenant using raw JSON
		var responseJSON string
		switch tenant {
		case "application":
			responseJSON = `{
				"status": "success",
				"data": {
					"groups": [{
						"name": "app-alerts",
						"rules": [{
							"type": "alerting",
							"name": "AppAlert",
							"alerts": [{
								"labels": {"alertname": "AppAlert", "tenant": "application"},
								"state": "firing"
							}]
						}]
					}]
				}
			}`
		case "infrastructure":
			responseJSON = `{
				"status": "success",
				"data": {
					"groups": [{
						"name": "infra-alerts",
						"rules": [{
							"type": "alerting",
							"name": "InfraAlert",
							"alerts": [{
								"labels": {"alertname": "InfraAlert", "tenant": "infrastructure"},
								"state": "firing"
							}]
						}]
					}]
				}
			}`
		default:
			// audit tenant returns empty
			responseJSON = `{"status": "success", "data": {"groups": []}}`
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responseJSON))
	}))
	defer server.Close()

	lokiRulerURL, _ := url.Parse(server.URL)
	store := &Store{
		lokiRulerURL: lokiRulerURL,
		httpClient:   server.Client(),
	}

	// Query all tenants
	result, err := store.getLokiRules(context.Background(), nil)
	require.NoError(t, err)

	// Verify all three tenants were queried
	assert.Equal(t, 1, queriedTenants["application"])
	assert.Equal(t, 1, queriedTenants["infrastructure"])
	assert.Equal(t, 1, queriedTenants["audit"])

	// Verify results were merged (should have 2 groups: app-alerts and infra-alerts)
	assert.Len(t, result.Groups, 2)

	groupNames := make(map[string]bool)
	for _, group := range result.Groups {
		groupNames[group.Name] = true
	}
	assert.True(t, groupNames["app-alerts"])
	assert.True(t, groupNames["infra-alerts"])
}

// TestGetLokiRules_NoURL tests handling when Loki Ruler URL is not configured
func TestGetLokiRules_NoURL(t *testing.T) {
	tests := []struct {
		name         string
		lokiRulerURL *url.URL
	}{
		{
			name:         "nil URL",
			lokiRulerURL: nil,
		},
		{
			name:         "empty URL",
			lokiRulerURL: &url.URL{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &Store{
				lokiRulerURL: tt.lokiRulerURL,
			}

			result, err := store.getLokiRules(context.Background(), nil)
			require.NoError(t, err)
			assert.Empty(t, result.Groups)
		})
	}
}

// TestGetLokiRules_PartialFailure tests that failures on some tenants don't fail entire query
func TestGetLokiRules_PartialFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Fail infrastructure tenant, succeed others
		if path == "/api/logs/v1/infrastructure/loki/api/v1/rules" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Return success for other tenants
		responseJSON := `{"status": "success", "data": {"groups": [{"name": "test-group", "rules": []}]}}`
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responseJSON))
	}))
	defer server.Close()

	lokiRulerURL, _ := url.Parse(server.URL)
	store := &Store{
		lokiRulerURL: lokiRulerURL,
		httpClient:   server.Client(),
	}

	// Should succeed despite infrastructure tenant failing
	result, err := store.getLokiRules(context.Background(), nil)
	require.NoError(t, err)

	// Should have 2 groups (application and audit succeeded)
	assert.Len(t, result.Groups, 2)
}

// TestStoreGet_WithLokiRuler tests integration of Loki Ruler into Store.Get
// This is a minimal test since full integration would require mocking Prometheus/Alertmanager too
func TestLokiRulerIntegration(t *testing.T) {
	// Create a mock Loki Ruler server
	lokiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		responseJSON := `{
			"status": "success",
			"data": {
				"groups": [{
					"name": "loki-alerts",
					"rules": [{
						"type": "alerting",
						"name": "LokiAlert",
						"query": "count_over_time({job=\"test\"}[5m]) > 100",
						"state": "firing",
						"alerts": [{
							"labels": {"alertname": "LokiAlert", "namespace": "test-ns"},
							"state": "firing",
							"activeAt": "2024-01-01T00:00:00Z",
							"value": "200"
						}]
					}]
				}]
			}
		}`

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responseJSON))
	}))
	defer lokiServer.Close()

	// Verify the store can query Loki Ruler
	lokiRulerURL, _ := url.Parse(lokiServer.URL)
	store := &Store{
		lokiRulerURL: lokiRulerURL,
		httpClient:   lokiServer.Client(),
	}

	result, err := store.getLokiRules(context.Background(), nil)
	require.NoError(t, err)
	assert.Len(t, result.Groups, 3) // 3 tenants

	// Verify we got alerts from Loki Ruler
	foundLokiAlert := false
	for _, group := range result.Groups {
		if group.Name == "loki-alerts" {
			foundLokiAlert = true
			break
		}
	}
	assert.True(t, foundLokiAlert, "Should find Loki alerts in merged results")
}
