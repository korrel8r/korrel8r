// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package podlog provides direct access to Kubernetes Pod logs via the Kube API-server.
//
// Logs are returned in OTEL format.
//
// # Store
//
// The store is the Kube API server itself, providing direct access to live pod log files.
// Logs are not guaranteed to be persisted after a pod is destroyed.
// No parameters are required, a podlog store automatically connects using Kube configuration.
//
//	domain: podlog
package podlog

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/korrel8r/impl"
	"github.com/korrel8r/korrel8r/pkg/otel"
	"github.com/korrel8r/korrel8r/pkg/unique"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// FIXME documentation

var (
	// Verify implementing interfaces.
	_ korrel8r.Domain    = Domain
	_ korrel8r.Store     = &Store{}
	_ korrel8r.Query     = &Query{}
	_ korrel8r.Class     = Class{}
	_ korrel8r.Previewer = Class{}

	podClass = k8s.Domain.Class("Pod").(k8s.Class)
)

var Domain = &domain{
	impl.NewDomain("podlog", "Live container logs via the Kube API server.", Class{}),
}

type Class struct{}

type Object otel.Log

// Query has the same fields as a [k8s.Query] with and additional 'containers' field.
type Query struct {
	k8s.Selector
	// Containers is a list of container names to be included in the result.
	// Empty or missing means all containers are included.
	Containers []string `json:"containers,omitempty"`
}

type Store struct {
	K8sStore  *k8s.Store
	Clientset kubernetes.Interface
}

type domain struct{ *impl.Domain }

func (d *domain) Query(s string) (korrel8r.Query, error) {
	_, q, err := impl.UnmarshalQueryString[Query](d, s)
	return &q, err
}

func (*domain) Store(_ any) (korrel8r.Store, error) { return NewStore(nil, nil) }

// NewStore creates a store using the given client and config.
// See [k8s.NewClient]
func NewStore(c client.WithWatch, cfg *rest.Config) (*Store, error) {
	k8sStore, err := k8s.NewStore(c, cfg)
	if err != nil {
		return nil, err
	}
	cs, err := kubernetes.NewForConfig(k8sStore.Config())
	if err != nil {
		return nil, err
	}
	return &Store{K8sStore: k8sStore, Clientset: cs}, nil
}

func (c Class) Domain() korrel8r.Domain                     { return Domain }
func (c Class) Name() string                                { return "log" }
func (c Class) String() string                              { return impl.ClassString(c) }
func (c Class) Unmarshal(b []byte) (korrel8r.Object, error) { return impl.UnmarshalAs[Object](b) }
func (c Class) Preview(o korrel8r.Object) string            { return Preview(o) }

// Preview returns the log body as a string.
func Preview(o korrel8r.Object) (line string) {
	if log, ok := o.(Object); ok {
		return fmt.Sprintf("%v", log.Body)
	}
	return ""
}

func (q *Query) Class() korrel8r.Class { return Class{} }
func (q *Query) String() string        { return impl.QueryString(q) }
func (q *Query) Data() string          { b, _ := json.Marshal(q); return string(b) }

func (s *Store) Domain() korrel8r.Domain                 { return Domain }
func (s *Store) StoreClasses() ([]korrel8r.Class, error) { return []korrel8r.Class{Class{}}, nil }

func (s *Store) Get(ctx context.Context, query korrel8r.Query, constraint *korrel8r.Constraint, logResult korrel8r.Appender) error {
	q, err := impl.TypeAssert[*Query](query)
	if err != nil {
		return err
	}
	if timeout := constraint.GetTimeout(); timeout > 0 {
		var cancel func()
		ctx, cancel = context.WithDeadline(ctx, time.Now().Add(timeout))
		defer cancel()
	}
	getter := s.newGetter(q, constraint)
	podQuery := k8s.NewQuery(podClass, q.Selector)
	podResult := korrel8r.AppenderFunc(func(o korrel8r.Object) {
		pod, err := k8s.AsStructured[corev1.Pod](o.(k8s.Object))
		if getter.Errs.Add(err) {
			return
		}
		getter.startPod(ctx, pod)
	})
	if err := s.K8sStore.Get(ctx, podQuery, constraint, podResult); err != nil {
		return err
	}
	for {
		select {
		case log, ok := <-getter.LogChan:
			if ok {
				logResult.Append(log)
			} else {
				return getter.Errs.Err()
			}
		case <-ctx.Done():
			return getter.Errs.Err()
		}
	}
}

type getter struct {
	Store      *Store
	Containers []string
	Constraint *korrel8r.Constraint
	Opts       *corev1.PodLogOptions
	Errs       unique.Errors
	LogChan    chan Object
	Workers    atomic.Int64
}

func (s *Store) newGetter(q *Query, constraint *korrel8r.Constraint) *getter {
	g := getter{
		Store:      s,
		Constraint: constraint,
		Containers: q.Containers,
		LogChan:    make(chan Object),
		Opts:       &corev1.PodLogOptions{Timestamps: true},
	}
	if start := constraint.GetStart(); !start.IsZero() {
		g.Opts.SinceTime = &metav1.Time{Time: *constraint.Start}
	}
	if n := int64(constraint.GetLimit()); n > 0 {
		g.Opts.TailLines = &n
	}
	return &g
}

func (g *getter) startPod(ctx context.Context, pod *corev1.Pod) {
	// Start collecting logs for each matching container
	for _, c := range pod.Spec.Containers {
		if len(g.Containers) == 0 || slices.Index(g.Containers, c.Name) >= 0 {
			opts := *g.Opts // Independent copy
			opts.Container = c.Name
			g.Workers.Add(1)
			go func() {
				defer func() {
					if g.Workers.Add(-1) == 0 {
						close(g.LogChan)
					}
				}()
				req := g.Store.Clientset.CoreV1().Pods(pod.GetNamespace()).GetLogs(pod.GetName(), &opts)
				stream, err := req.Stream(ctx)
				if g.Errs.Add(err) {
					return
				}
				g.readStream(ctx, stream, pod, &c)
			}()
		}
	}
}

func (g *getter) readStream(ctx context.Context, stream io.ReadCloser, pod *corev1.Pod, container *corev1.Container) {
	// Cancel reading if context is canceled
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = stream.Close() // Interrupt stream.Read()
		case <-done:
		}
	}()

	// Clean up on exit
	defer func() {
		close(done) // Don't leak the goroutine
		_ = stream.Close()
		g.Errs.Add(ctx.Err())
	}()

	// Scan the log lines
	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		var log Object
		log.Body = scanner.Text()
		// Check for timestamp in text
		if head, tail, ok := strings.Cut(scanner.Text(), " "); ok {
			if timestamp, err := time.Parse(time.RFC3339Nano, head); err == nil {
				log.Body = tail
				log.Timestamp = timestamp
			} else {
				log.Timestamp = time.Now()
			}
		}
		if g.Constraint.CompareTime(log.Timestamp) != 0 {
			continue // Skip timestamp out of range
		}
		log.Attributes = map[string]any{
			"k8s.pod.name":           pod.GetName(),
			"k8s.pod.namespace.name": pod.GetNamespace(),
			"k8s.container":          container.Name,
			// FIXME review OTEL attribute list.
		}
		g.LogChan <- log
	}
}
