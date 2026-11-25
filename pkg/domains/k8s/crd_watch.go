// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package k8s

import (
	"context"
	"sync/atomic"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	watchRetryMin = 5 * time.Second // Delay before re-starting a CRD watch, doubled after each failure.
	watchRetryMax = 5 * time.Minute // Maximum delay between failed CRD watch attempts.
	refreshDelay  = time.Second     // Collect CRD events for this long before refreshing.
)

// crdWatcher is a running CRD watch loop, see [domain.watchClasses].
type crdWatcher struct {
	ctx     context.Context
	cancel  context.CancelFunc
	done    chan struct{} // Closed when the watch loop exits.
	failing atomic.Bool   // True if the most recent attempt to start a watch failed.
}

// startWatch starts a CRD watcher using c and di, unless a healthy watcher is already running.
//
// The watcher reconnects by itself when a watch ends, so a working watcher is kept when
// stores are re-created. A failing watcher is replaced, in case the new client works better.
func (d *domain) startWatch(c client.WithWatch, di discovery.DiscoveryInterface) {
	d.watchMu.Lock()
	defer d.watchMu.Unlock()
	if d.watcher != nil {
		if !d.watcher.failing.Load() {
			return
		}
		d.watcher.cancel()
	}
	w := &crdWatcher{done: make(chan struct{})}
	w.ctx, w.cancel = context.WithCancel(context.Background())
	d.watcher = w
	go func() {
		defer close(w.done)
		d.watchClasses(w, c, di)
	}()
}

// watchClasses refreshes classes when CRDs change, until the watcher is cancelled.
// Watches that end are restarted, with exponential back-off for watches that fail to start.
func (d *domain) watchClasses(w *crdWatcher, c client.WithWatch, di discovery.DiscoveryInterface) {
	delay, failures := watchRetryMin, 0
	for w.ctx.Err() == nil {
		if err := d.watchCRDs(w, c, di); err != nil {
			if w.ctx.Err() != nil {
				return
			}
			w.failing.Store(true)
			failures++
			if failures == 1 {
				log.Error(err, "k8s CRD watch failed, new resource types will not be detected until it succeeds")
			} else {
				log.V(1).Info("k8s CRD watch still failing", "error", err, "failures", failures, "retry", delay)
			}
		} else {
			delay, failures = watchRetryMin, 0
		}
		select {
		case <-w.ctx.Done():
			return
		case <-time.After(delay):
		}
		if failures > 0 {
			delay = min(2*delay, watchRetryMax)
		}
	}
}

// watchCRDs watches CRDs until the watch ends, refreshing classes after events.
// Returns an error if the watch could not be started, nil when a started watch ends.
// Refresh after starting each watch to catch changes missed between watches: a watch
// does not replay events for CRDs that already exist.
func (d *domain) watchCRDs(cw *crdWatcher, c client.WithWatch, di discovery.DiscoveryInterface) error {
	ctx := cw.ctx
	crds := &unstructured.UnstructuredList{}
	crds.SetGroupVersionKind(schema.GroupVersionKind{Group: "apiextensions.k8s.io", Version: "v1", Kind: "CustomResourceDefinitionList"})
	w, err := c.Watch(ctx, crds)
	if err != nil {
		return err
	}
	defer w.Stop()
	if cw.failing.Swap(false) {
		log.Info("k8s CRD watch started after failures")
	}
	d.refresh(di)
	var refresh <-chan time.Time // Non-nil if a refresh is pending.
	for {
		select {
		case <-ctx.Done():
			return nil
		case _, ok := <-w.ResultChan():
			if !ok {
				return nil
			}
			if refresh == nil {
				refresh = time.After(refreshDelay)
			}
		case <-refresh:
			refresh = nil
			d.refresh(di)
		}
	}
}
