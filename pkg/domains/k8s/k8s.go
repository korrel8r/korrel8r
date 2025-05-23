// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package k8s implements [Kubernetes] resources stored in a Kube API server.
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
package k8s

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/korrel8r/impl"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Domain for Kubernetes resources stored in a Kube API server.
var Domain = domain{}

// Class represents a kind of kubernetes resource.
// The format of a class name is: "k8s:KIND[.VERSION][.GROUP]".
//
// Missing VERSION implies "v1", if present VERSION must follow the [Kubernetes version patterns].
// Missing GROUP implies the core group.
//
// Examples: `k8s:Pod`, `ks8:Pod.v1`, `k8s:Deployment.apps`, `k8s:Deployment.v1.apps`, `k8s:Route.v1.route.openshift.io`
//
// [Kubernetes version patterns]: https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definition-versioning/#version-priority
type Class schema.GroupVersionKind

// Object represents a kubernetes resource as a map, map keys are serialized field names.
// Rule templates should use the JSON (lowerCase) field names, NOT the UpperCase Go struct field names.
type Object = map[string]any

// Query struct for a Kubernetes query.
//
// Example:
//
//	k8s:Pod.v1:{"namespace":"openshift-cluster-version","name":"cluster-version-operator-8d86bcb65-btlgn"}
type Query struct {
	// Namespace restricts the search to a namespace.
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
	// Labels restricts the search to objects with matching label values (optional)
	Labels client.MatchingLabels `json:"labels,omitempty"`
	// Fields restricts the search to objects with matching field values (optional)
	Fields client.MatchingFields `json:"fields,omitempty"`

	class Class // class is the underlying k8s.Class object. Implied by query name prefix.
}

// Store presents the Kubernetes API server as a korrel8r.Store.
//
// Uses the default kube config to connect to the cluster, no additional configuration is needed.
//
//	 stores:
//		  domain: k8s
type Store struct {
	cfg      *rest.Config
	c        client.Client
	base     *url.URL
	discover discovery.DiscoveryInterface
}

// Validate interfaces
var (
	_ korrel8r.Domain = Domain
	_ korrel8r.Class  = Class{}
	_ korrel8r.Object = Object(nil)
	_ korrel8r.Query  = &Query{}
)

// domain implementation
type domain struct{}

func (d domain) Name() string              { return "k8s" }
func (d domain) String() string            { return d.Name() }
func (d domain) Description() string       { return "Resource objects in a Kubernetes API server" }
func (d domain) Classes() []korrel8r.Class { return nil }

// Store connects to the kube config default cluster. The config parameter is ignored.
func (d domain) Store(_ any) (s korrel8r.Store, err error) {
	cfg, err := GetConfig()
	if err != nil {
		return nil, err
	}
	c, err := NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return NewStore(c, cfg)
}

var (
	// classRE regexp matching for KIND[.VERSION][.GROUP]
	classRE    = regexp.MustCompile(`^([^./]+)(?:\.(v[0-9]+(?:(?:alpha|beta)[0-9]*)?))?(?:\.([^/]*))?$`)
	emptyClass = Class(schema.GroupVersionKind{})
)

func ClassNamed(name string) Class {
	if m := classRE.FindStringSubmatch(name); m != nil {
		gvk := schema.GroupVersionKind{Kind: m[1], Version: m[2], Group: m[3]}
		if gvk.Version == "" {
			gvk.Version = "v1"
		}
		return Class(gvk)
	}
	return emptyClass
}

func (d domain) Class(name string) korrel8r.Class {
	c := ClassNamed(name)
	if c == emptyClass {
		return nil
	}
	return c
}

func (d domain) Query(s string) (korrel8r.Query, error) {
	class, query, err := impl.UnmarshalQueryString[Query](d, s)
	if err != nil {
		return nil, err
	}
	query.class = class.(Class)
	return &query, nil
}

func (c Class) ID(o korrel8r.Object) any {
	if o, _ := o.(Object); o != nil {
		return client.ObjectKeyFromObject(Wrap(o))
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
	return Unwrap(u)
}

func NewQuery(c Class, namespace, name string, labels, fields map[string]string) *Query {
	return &Query{
		Namespace: namespace,
		Name:      name,
		Labels:    labels,
		Fields:    fields,
		class:     c,
	}
}

func (q Query) Class() korrel8r.Class { return q.class }
func (q Query) Data() string          { b, _ := json.Marshal(q); return string(b) }
func (q Query) String() string        { return impl.QueryString(q) }

// NewStore creates a new k8s store.
func NewStore(c client.Client, cfg *rest.Config) (*Store, error) {
	dc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return NewStoreWithDiscovery(c, cfg, dc)
}

// NewStoreWithDiscovery creates a store with the specified discovery interface.
// Intended for tests with a fake client and discovery.
func NewStoreWithDiscovery(c client.Client, cfg *rest.Config, di discovery.DiscoveryInterface) (*Store, error) {
	host := cfg.Host
	if host == "" {
		host = "localhost"
	}
	base, _, err := rest.DefaultServerURL(host, cfg.APIPath, schema.GroupVersion{}, true)
	return &Store{cfg: cfg, c: c, base: base, discover: di}, err
}

func (s Store) Domain() korrel8r.Domain { return Domain }
func (s Store) Client() client.Client   { return s.c }

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
		if c.CompareTime(Wrap(o.(Object)).GetCreationTimestamp().Time) <= 0 {
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

func (s *Store) StoreClasses() (classes []korrel8r.Class, err error) {
	// discovery client can return a partial resource list with an error, do the same.
	clusterScoped, err1 := s.discover.ServerPreferredResources()
	namespaceScoped, err2 := s.discover.ServerPreferredNamespacedResources()
	err = errors.Join(err1, err2)
	for _, lists := range [][]*metav1.APIResourceList{clusterScoped, namespaceScoped} {
		for _, l := range lists {
			for _, r := range l.APIResources {
				gv, _ := schema.ParseGroupVersion(l.GroupVersion)
				g := r.Group
				if g == "" {
					g = gv.Group
				}
				v := r.Version
				if v == "" {
					v = gv.Version
				}
				gvk := schema.GroupVersionKind{Group: g, Version: v, Kind: r.Kind}
				classes = append(classes, Class(gvk))
			}
		}
	}
	return classes, err
}

func (s *Store) getObject(ctx context.Context, q *Query, result korrel8r.Appender) error {
	u := Wrap(q.class.New())
	if err := s.c.Get(ctx, types.NamespacedName{Namespace: q.Namespace, Name: q.Name}, u); err != nil {
		return err
	}
	result.Append(Unwrap(u))
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
		result.Append(Unwrap(&list.Items[i]))
	}
	return nil
}

func Wrap(o Object) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: o}
}

func Unwrap(u *unstructured.Unstructured) Object {
	return Object(u.Object)
}
