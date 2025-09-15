// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package log

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/must"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/korrel8r/impl"
	"github.com/korrel8r/korrel8r/pkg/result"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var podClass = must.Must1(k8s.ParseClass("Pod.v1"))

type directStore struct {
	*impl.Store
	K8sStore  *k8s.Store
	Clientset kubernetes.Interface // Access to extended pod API with logs.
}

func newDirectStore(k8sStore *k8s.Store) (*directStore, error) {
	clientset, err := kubernetes.NewForConfig(k8sStore.Config())
	if err != nil {
		return nil, err
	}
	return &directStore{Store: impl.NewStore(Domain), K8sStore: k8sStore, Clientset: clientset}, nil
}

func (s *directStore) Get(ctx context.Context, query korrel8r.Query, constraint *korrel8r.Constraint, logResult korrel8r.Appender) (err error) {
	q, err := impl.TypeAssert[*Query](query)
	if err != nil {
		return err
	}
	if q.direct == nil {
		return fmt.Errorf("direct log store cannot execute Loki query: %v", query)
	}

	var cancel func()
	if timeout := constraint.GetTimeout(); timeout > 0 {
		ctx, cancel = context.WithDeadline(ctx, time.Now().Add(timeout))
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	defer cancel()
	group, ctx := errgroup.WithContext(ctx)

	// Get pods for the query
	podQuery := k8s.NewQuery(podClass, q.direct.Selector)
	pods := result.NewList()
	if err := s.K8sStore.Get(ctx, podQuery, constraint, pods); err != nil {
		return err
	}

	// Collect logs
	count := &atomic.Int64{}
	out := make(chan Object)
	done := make(chan struct{})

	go func() {
		for log := range out {
			logResult.Append(log)
		}
		close(done)
	}()

	// Read log streams for each container in each pod, push log records to channel.
	for _, o := range pods.List() {
		ko, _ := o.(k8s.Object)
		pod, err := k8s.AsStructured[corev1.Pod](ko)
		if err != nil {
			return err
		}
		for _, c := range pod.Spec.Containers {
			if !q.direct.IsContainerSelected(c.Name) {
				continue
			}
			opts := &corev1.PodLogOptions{
				Container:  c.Name,
				Timestamps: true,
			}
			if start := constraint.GetStart(); !start.IsZero() {
				opts.SinceTime = &metav1.Time{Time: *constraint.Start}
			}
			group.Go(func() error {
				stream, err := s.Clientset.CoreV1().Pods(pod.GetNamespace()).GetLogs(pod.GetName(), opts).Stream(ctx)
				if err != nil {
					return err
				}
				attrs := Object{ // Common attributes
					AttrK8sPodName:              pod.GetName(),
					AttrK8sNamespaceName:        pod.GetNamespace(),
					AttrK8sContainerName:        c.Name,
					AttrKubernetesPodName:       pod.GetName(),
					AttrKubernetesNamespaceName: pod.GetNamespace(),
					AttrKubernetesContainerName: c.Name,
				}
				return s.readPodLogs(ctx, stream, out, attrs, constraint, count)
			})
		}
	}
	err = group.Wait()
	close(out)
	<-done
	return err
}

func (s *directStore) readPodLogs(ctx context.Context, stream io.ReadCloser, out chan<- Object, attrs Object, constraint *korrel8r.Constraint, count *atomic.Int64) (err error) {
	// Arrange to close the stream when the context is done.
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			_ = stream.Close()
		case <-done:
		}
	}()

	// Scan the log lines, create log.Object
	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		if scanner.Err() != nil {
			if scanner.Err() == io.EOF {
				return ctx.Err()
			}
			return err
		}
		line := scanner.Text()
		o := maps.Clone(attrs)
		o[AttrBody] = line

		// Check timestamp
		ts, msg, _ := strings.Cut(line, " ")
		if msg != "" {
			if timestamp, terr := time.Parse(time.RFC3339Nano, ts); terr == nil {
				o[AttrBody] = msg
				o[AttrObservedTimestamp] = ts // Already in RFC3999 format
				n := constraint.CompareTime(timestamp)
				if n < 0 { // Before time range, ignore this line
					continue
				} else if n > 0 { // After time range, stop now.
					return nil
				}
			}
		}
		// Check limit
		limit := int64(constraint.GetLimit())
		if limit > 0 && count.Add(1) > limit {
			return
		}
		out <- o
	}
	return nil
}

type ContainerSelector struct {
	k8s.Selector
	// Containers is a list of container names to be included in the result.
	// Empty or missing means all containers are included.
	Containers []string `json:"containers,omitempty"`
}

func (s ContainerSelector) IsContainerSelected(container string) bool {
	return len(s.Containers) == 0 || slices.Index(s.Containers, container) >= 0
}

// LogQL returns a log QL query that is equivalent to the podSelector.
func (p *ContainerSelector) LogQL() string {
	w := &strings.Builder{}
	add := func(k, v string) {
		if v != "" {
			if w.String() == "" {
				w.WriteString("{")
			} else {
				w.WriteString(",")
			}
			fmt.Fprintf(w, "%v%q", k, v)
		}
	}
	// FIXME OTEL vs. Viaq queries - need to handle OTEL after migration.
	// Defer this till store, detect otel/viaq content using a loki labels query.
	add("kubernetes_namespace_name=", p.Namespace)
	add("kubernetes_pod_name=", p.Name)
	add("kubernetes_container_name=~", strings.Join(p.Containers, "|"))
	w.WriteString("}|json")
	for k, v := range p.Labels {
		fmt.Fprintf(w, "|kubernetes_labels_%v=%q", SafeLabel(k), v)
	}
	return w.String()
}
