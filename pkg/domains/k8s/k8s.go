// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package k8s implements [Kubernetes] resources stored in a Kube API server.
//
// # Class
//
// Class represents a kind of kubernetes resource.
// The format of a class name is: "k8s:KIND[.VERSION][.GROUP]".
//
// Missing VERSION implies "v1", if present VERSION must follow the [Kubernetes version patterns].
// Missing GROUP implies the core group.
//
// Examples: `k8s:Pod`, `ks8:Pod/v1`, `k8s:Deployment.apps`, `k8s:Deployment.apps/v1`, `k8s:Route.route.openshift.io/v1`
//
// # Object
//
// Object represents a kubernetes resource as a map, map keys are serialized field names.
// Rule templates should use the JSON (lowerCase) field names, NOT the UpperCase Go struct field names.
//
// # Query
//
// Example:
//
//	k8s:Pod.v1:{"namespace":"openshift-cluster-version","name":"cluster-version-operator-8d86bcb65-btlgn"}
//
// # Store
//
// The k8s domain automatically connects to the current cluster (as determined by kubectl),
// no additional configuration is needed.
//
//	 stores:
//		  domain: k8s
//
// [Kubernetes]: https://kubernetes.io/docs/concepts/overview/
// [Kubernetes version patterns]: https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/#version-priority
package k8s

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/korrel8r/impl"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	log    = logging.Log()
	Domain = &domain{classScopes: map[korrel8r.Class]bool{}}
)

func init() {
	Domain.addClasses(defaultResources)
}

type Class schema.GroupVersionKind

type Object = map[string]any

type Query struct {
	Selector
	class Class // class is the underlying k8s.Class object. Implied by query name prefix.
}

// Selector for k8s queries.
type Selector struct {
	// Namespace restricts the search to a namespace (optional).
	Namespace string `json:"namespace,omitempty"`
	// Name of the object (optional).
	Name string `json:"name,omitempty"`
	// Labels restricts the search to objects with matching label values (optional)
	Labels client.MatchingLabels `json:"labels,omitempty"`
	// Fields restricts the search to objects with matching field values (optional)
	Fields client.MatchingFields `json:"fields,omitempty"`
}

// Store presents the Kubernetes API server as a korrel8r.Store.
//
// Uses the default kube config to connect to the cluster, no additional configuration is needed.
//
//	 stores:
//		  domain: k8s
type Store struct {
	cfg      *rest.Config
	c        client.WithWatch
	base     *url.URL
	discover discovery.DiscoveryInterface
}

// Validate interfaces
var (
	_ korrel8r.Domain = Domain
	_ korrel8r.Class  = Class{}
	_ korrel8r.Object = Object(nil)
	_ korrel8r.Query  = &Query{}
	_ korrel8r.Store  = &Store{}
)

// domain implementation
type domain struct {
	m           sync.Mutex
	classScopes map[korrel8r.Class]bool
	classes     []korrel8r.Class
}

func (d *domain) Name() string              { return "k8s" }
func (d *domain) String() string            { return d.Name() }
func (d *domain) Description() string       { return "Resource objects in a Kubernetes API server" }
func (d *domain) Classes() []korrel8r.Class { d.m.Lock(); defer d.m.Unlock(); return d.classes }

func (d *domain) addClasses(list []*metav1.APIResourceList) {
	for _, l := range list {
		gv, err := schema.ParseGroupVersion(l.GroupVersion)
		if err != nil {
			continue
		}
		for _, r := range l.APIResources {
			g := r.Group
			if g == "" {
				g = gv.Group
			}
			v := r.Version
			if v == "" {
				v = gv.Version
			}
			c := Class(schema.GroupVersionKind{Group: g, Version: v, Kind: r.Kind})
			d.classScopes[c] = r.Namespaced
		}
	}
	// Update the sorted class list.
	d.classes = slices.Collect(maps.Keys(d.classScopes))
	slices.SortFunc(d.classes, func(a, b korrel8r.Class) int { return cmp.Compare(a.String(), b.String()) })
}

// Store connects to the kube config default cluster. The config parameter is ignored.
func (d *domain) Store(config any) (s korrel8r.Store, err error) {
	return d.NewStore(nil, nil)
}

// classRE regexp matching for KIND[.VERSION][.GROUP]
var classRE = regexp.MustCompile(`^([^./]+)(?:\.(v[0-9]+(?:(?:alpha|beta)[0-9]*)?))?(?:\.([^/]*))?$`)

// Class returns a named class.
// Non-nil return does not mean the resource is available on the current cluster.
// See Class.IsKnown().
func (d *domain) Class(name string) korrel8r.Class {
	if c, err := ParseClass(name); err == nil {
		return c
	}
	return nil
}

func ParseClass(name string) (Class, error) {
	if m := classRE.FindStringSubmatch(name); m != nil {
		gvk := schema.GroupVersionKind{Kind: m[1], Version: m[2], Group: m[3]}
		if gvk.Version == "" {
			gvk.Version = "v1"
		}
		return Class(gvk), nil
	}
	return Class{}, korrel8r.ClassNotFoundError(impl.NameJoin(Domain.Name(), name))
}

func (d *domain) Query(s string) (korrel8r.Query, error) {
	class, query, err := impl.UnmarshalQueryString[Query](d, s)
	if err != nil {
		return nil, err
	}
	query.class = class.(Class)
	return &query, nil
}

func (c Class) ID(o korrel8r.Object) any {
	if o, _ := o.(Object); o != nil {
		return client.ObjectKeyFromObject(ToUnstructured(o))
	}
	return nil
}

func (c Class) Preview(o korrel8r.Object) string {
	switch o := o.(type) {
	case *corev1.Event:
		return o.Message
	default:
		return fmt.Sprintf("%v", c.ID(o))
	}
}

func (c Class) Domain() korrel8r.Domain      { return Domain }
func (c Class) String() string               { return impl.ClassString(c) }
func (c Class) GVK() schema.GroupVersionKind { return schema.GroupVersionKind(c) }

func (c Class) Name() string {
	w := &strings.Builder{}
	fmt.Fprintf(w, "%v.%v", c.Kind, c.Version)
	if c.Group != "" {
		fmt.Fprintf(w, ".%v", c.Group)
	}
	return w.String()
}

func (c Class) Unmarshal(b []byte) (korrel8r.Object, error) {
	o := c.New()
	err := json.Unmarshal(b, &o)
	return o, err
}

func (c Class) New() Object {
	u := &unstructured.Unstructured{}
	u.GetObjectKind().SetGroupVersionKind(c.GVK())
	return FromUnstructured(u)
}

// IsNamespaced returns true if the k8s resource represented by this class is namespaced
func (c Class) IsNamespaced() bool {
	Domain.m.Lock()
	defer Domain.m.Unlock()
	ns, ok := Domain.classScopes[c]
	return ns || !ok
}

// IsKnown returns true if the k8s resource represented by this class is known to the API server.
func (c Class) IsKnown() bool {
	Domain.m.Lock()
	defer Domain.m.Unlock()
	_, ok := Domain.classScopes[c]
	return ok
}

func NewQuery(c Class, s Selector) *Query { return &Query{class: c, Selector: s} }

func (q Query) Class() korrel8r.Class { return q.class }
func (q Query) Data() string          { b, _ := json.Marshal(q); return string(b) }
func (q Query) String() string        { return impl.QueryString(q) }

// NewStore creates a new k8s store.
// Called with nil, nil uses default kube config values.
func (d *domain) NewStore(c client.WithWatch, cfg *rest.Config) (*Store, error) {
	var err error
	if cfg == nil {
		cfg, err = GetConfig()
		if err != nil {
			return nil, err
		}
	}
	if c == nil {
		c, err = NewClient(cfg)
		if err != nil {
			return nil, err
		}
	}
	dc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return d.NewStoreWithDiscovery(c, cfg, dc)
}

// NewStoreWithDiscovery creates a store with the specified discovery interface.
// Intended for tests with a fake client and discovery.
func (d *domain) NewStoreWithDiscovery(c client.WithWatch, cfg *rest.Config, di discovery.DiscoveryInterface) (*Store, error) {
	host := cfg.Host
	if host == "" {
		host = "localhost"
	}
	base, _, err := rest.DefaultServerURL(host, cfg.APIPath, schema.GroupVersion{}, true)
	if err != nil {
		return nil, err
	}
	_, resources, err := di.ServerGroupsAndResources()
	if err != nil {
		log.Info("k8s discovery error, continuing", "error", err) // Log but continue.
	}
	d.addClasses(resources)
	return &Store{cfg: cfg, c: c, base: base, discover: di}, nil
}

func (s *Store) Domain() korrel8r.Domain  { return Domain }
func (s *Store) Client() client.WithWatch { return s.c }
func (s *Store) Config() *rest.Config     { return s.cfg }

func (s *Store) Get(ctx context.Context, query korrel8r.Query, c *korrel8r.Constraint, result korrel8r.Appender) (err error) {
	// Skip the call if the class is not known
	class, err := impl.TypeAssert[Class](query.Class())
	if err != nil {
		return err
	}
	gvk := class.GVK()
	if _, err := s.c.RESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version); err != nil {
		return err
	}
	q, err := impl.TypeAssert[*Query](query)
	if err != nil {
		return err
	}
	appender := korrel8r.AppenderFunc(func(o korrel8r.Object) {
		// Include only objects created before or during the constraint interval.
		if c.CompareTime(ToUnstructured(o.(Object)).GetCreationTimestamp().Time) <= 0 {
			result.Append(o)
		}
	})
	if q.Name != "" { // Request for single object.
		err = s.getObject(ctx, q, appender)
	} else {
		err = s.getList(ctx, q, appender, c)
	}
	if apierrors.IsNotFound(err) {
		err = nil // Finding nothing is not an error.
	}
	return err
}

func (s *Store) getObject(ctx context.Context, q *Query, result korrel8r.Appender) error {
	u := ToUnstructured(q.class.New())
	if err := s.c.Get(ctx, types.NamespacedName{Namespace: q.Namespace, Name: q.Name}, u); err != nil {
		return err
	}
	result.Append(FromUnstructured(u))
	return nil
}

func (s *Store) getList(ctx context.Context, q *Query, result korrel8r.Appender, c *korrel8r.Constraint) (err error) {
	gvk := q.class.GVK()
	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(gvk.GroupVersion().WithKind(gvk.Kind + "List"))
	var opts []client.ListOption
	if q.Namespace != "" {
		opts = append(opts, client.InNamespace(q.Namespace))
	}
	if len(q.Labels) > 0 {
		opts = append(opts, q.Labels)
	}
	if len(q.Fields) > 0 {
		opts = append(opts, q.Fields)
	}
	if limit := c.GetLimit(); limit > 0 {
		opts = append(opts, client.Limit(int64(limit)))
	}
	if err := s.c.List(ctx, list, opts...); err != nil {
		return err
	}
	defer func() { // Handle reflect panics.
		if r := recover(); r != nil && err == nil {
			err = fmt.Errorf("invalid list object: %T", list)
		}
	}()
	for i := range list.Items {
		result.Append(FromUnstructured(&list.Items[i]))
	}
	return nil
}

func ToUnstructured(o Object) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: o}
}

func FromUnstructured(u *unstructured.Unstructured) Object {
	return Object(u.Object)
}

func ToStructured(o Object, target any) error {
	return runtime.DefaultUnstructuredConverter.FromUnstructured(o, target)
}

func AsStructured[T any](o Object) (*T, error) {
	var target T
	err := ToStructured(o, &target)
	return &target, err
}

func FromStructured(source any) (Object, error) {
	return runtime.DefaultUnstructuredConverter.ToUnstructured(source)
}

// defaultResources are always known as k8s classes.
// Additional resources can be loaded on creation of a k8s.Store.
var defaultResources = []*metav1.APIResourceList{
	{GroupVersion: "v1", APIResources: []metav1.APIResource{
		{Namespaced: true, Kind: "Binding"},
		{Namespaced: false, Kind: "ComponentStatus"},
		{Namespaced: true, Kind: "ConfigMap"},
		{Namespaced: true, Kind: "Endpoints"},
		{Namespaced: true, Kind: "Event"},
		{Namespaced: true, Kind: "LimitRange"},
		{Namespaced: false, Kind: "Namespace"},
		{Namespaced: false, Kind: "Node"},
		{Namespaced: true, Kind: "PersistentVolumeClaim"},
		{Namespaced: false, Kind: "PersistentVolume"},
		{Namespaced: true, Kind: "Pod"},
		{Namespaced: true, Kind: "PodTemplate"},
		{Namespaced: true, Kind: "ReplicationController"},
		{Namespaced: true, Kind: "ResourceQuota"},
		{Namespaced: true, Kind: "Secret"},
		{Namespaced: true, Kind: "ServiceAccount"},
		{Namespaced: true, Kind: "Service"},
	}},
	{GroupVersion: "apps/v1", APIResources: []metav1.APIResource{
		{Namespaced: true, Kind: "ControllerRevision"},
		{Namespaced: true, Kind: "DaemonSet"},
		{Namespaced: true, Kind: "Deployment"},
		{Namespaced: true, Kind: "ReplicaSet"},
		{Namespaced: true, Kind: "StatefulSet"},
	}},
	{GroupVersion: "batch/v1", APIResources: []metav1.APIResource{
		{Namespaced: true, Kind: "CronJob"},
		{Namespaced: true, Kind: "Job"},
	}},
	{GroupVersion: "policy/v1", APIResources: []metav1.APIResource{
		{Namespaced: true, Kind: "PodDisruptionBudget"},
	}},
	{GroupVersion: "storage.k8s.io", APIResources: []metav1.APIResource{
		{Namespaced: false, Kind: "VolumeSnapshotClass"},
		{Namespaced: false, Kind: "VolumeSnapshotContent"},
		{Namespaced: true, Kind: "VolumeSnapshot"},
		{Namespaced: false, Kind: "CSIDriver"},
		{Namespaced: false, Kind: "CSINode"},
		{Namespaced: true, Kind: "CSIStorageCapacity"},
		{Namespaced: false, Kind: "StorageClass"},
		{Namespaced: false, Kind: "VolumeAttachment"},
	}},
}
