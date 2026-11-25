// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package k8s

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/watch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// readyWatch signals when the watch loop starts receiving events, after its initial refresh.
type readyWatch struct {
	watch.Interface
	ready chan struct{}
	once  *sync.Once
}

func (w *readyWatch) ResultChan() <-chan watch.Event {
	w.once.Do(func() { close(w.ready) })
	return w.Interface.ResultChan()
}

type readyWatchClient struct {
	client.WithWatch
	ready chan struct{}
	once  sync.Once
}

func (c *readyWatchClient) Watch(ctx context.Context, list client.ObjectList, opts ...client.ListOption) (watch.Interface, error) {
	w, err := c.WithWatch.Watch(ctx, list, opts...)
	if err != nil {
		return nil, err
	}
	return &readyWatch{Interface: w, ready: c.ready, once: &c.once}, nil
}

func TestDiscoverNewClass_cluster(t *testing.T) {
	cfg, err := GetConfig()
	require.NoError(t, err)
	base, err := NewClient(cfg)
	require.NoError(t, err)
	c := &readyWatchClient{WithWatch: base, ready: make(chan struct{})}
	d := newDomain()
	_, err = d.NewStore(c, cfg)
	require.NoError(t, err)
	currentWatcher(t, d) // Stop the background watch when the test ends.

	// Create the CRD only after the watch has started and its initial discovery has finished.
	select {
	case <-c.ready:
	case <-time.After(30 * time.Second):
		t.Fatal("CRD watch did not start (check cluster-scoped watch permission)")
	}

	group := fmt.Sprintf("korrel8r-e2e-%d.example.com", time.Now().UnixNano())
	crd := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "apiextensions.k8s.io/v1",
		"kind":       "CustomResourceDefinition",
		"metadata":   map[string]any{"name": "widgets." + group},
		"spec": map[string]any{
			"group": group,
			"names": map[string]any{"kind": "E2eWidget", "plural": "widgets", "singular": "widget"},
			"scope": "Namespaced",
			"versions": []any{map[string]any{
				"name": "v1", "served": true, "storage": true,
				"schema": map[string]any{"openAPIV3Schema": map[string]any{"type": "object"}},
			}},
		},
	}}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	t.Cleanup(cancel)
	require.NoError(t, c.Create(ctx, crd))
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := c.Delete(cleanupCtx, crd); err != nil {
			t.Errorf("delete test CRD %s: %v", crd.GetName(), err)
		}
	})

	require.Eventually(t, func() bool {
		found := &unstructured.Unstructured{}
		found.SetGroupVersionKind(crd.GroupVersionKind())
		if err := c.Get(ctx, client.ObjectKeyFromObject(crd), found); err != nil {
			return false
		}
		conditions, _, _ := unstructured.NestedSlice(found.Object, "status", "conditions")
		for _, condition := range conditions {
			c, ok := condition.(map[string]any)
			if ok && c["type"] == "Established" && c["status"] == "True" {
				return true
			}
		}
		return false
	}, time.Minute, time.Second, "test CRD was not established")

	require.Eventually(t, func() bool {
		return d.Class("E2eWidget.v1."+group) != nil
	}, time.Minute, time.Second, "new CRD class was not discovered")
}
