// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestK8sHealthStatus(t *testing.T) {
	for _, tt := range []struct {
		name string
		obj  Object
		want string
	}{
		{
			name: "no status field returns empty",
			obj: Object{
				"apiVersion": "v1",
				"kind":       "ConfigMap",
				"metadata":   Object{"name": "test", "namespace": "default"},
			},
			want: "",
		},
		{
			name: "ready condition true returns empty",
			obj: Object{
				"apiVersion": "v1",
				"kind":       "Pod",
				"metadata":   Object{"name": "healthy-pod", "namespace": "default"},
				"status": Object{
					"conditions": []any{
						Object{
							"type":   "Ready",
							"status": "True",
						},
					},
				},
			},
			want: "",
		},
		{
			name: "ready condition false returns Error",
			obj: Object{
				"apiVersion": "v1",
				"kind":       "Pod",
				"metadata":   Object{"name": "unhealthy-pod", "namespace": "default"},
				"status": Object{
					"conditions": []any{
						Object{
							"type":   "Ready",
							"status": "False",
						},
					},
				},
			},
			want: "Error",
		},
		{
			name: "memory pressure returns Warning",
			obj: Object{
				"apiVersion": "v1",
				"kind":       "Node",
				"metadata":   Object{"name": "bad-node"},
				"status": Object{
					"conditions": []any{
						Object{
							"type":   "Ready",
							"status": "True",
						},
						Object{
							"type":   "MemoryPressure",
							"status": "True",
						},
					},
				},
			},
			want: "Warning",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := k8sHealthStatus(tt.obj)
			assert.Equal(t, tt.want, got)
		})
	}
}
