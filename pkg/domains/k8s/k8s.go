// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package k8s implements Kubernetes resources stored in a Kube API server.
//
// # Class
//
// A k8s class corresponds to a kind of Kubernetes resource, the class name is `KIND.VERSION.GROUP`
// VERSION can be omitted if there is no ambiguity.
// Example class names: `k8s:Pod`, `k8s:Pod.v1`, `k8s:Deployment.v1.apps`, `k8s:Deployment.apps`
//
// # Object
//
// Objects are represented by the standard Go types used by `k8s.io/client-go/api`, and by Kube-generated CRD struct types.
// Rules starting from the k8s domain should use the capitalized Go field names rather than the lowercase JSON field names.
//
// # Query
//
// Queries are the JSON-serialized form of this struct: [Query]
//
// For example:
//
//	k8s:Pod.v1.:{"namespace":"openshift-cluster-version","name":"cluster-version-operator-8d86bcb65-btlgn"}
//
// # Store
//
// k8s stores connects to the current logged-in Kubernetes cluster, no other configuration is needed than:
//
//	domain: k8s
package k8s

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"reflect"
	"strings"

	"github.com/korrel8r/korrel8r/internal/pkg/must"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/korrel8r/impl"
	"github.com/korrel8r/korrel8r/pkg/openshift/console"
	"golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

// Domain for Kubernetes resources stored in a Kube API server.
var Domain = domain{}

// Class represents a kind of kubernetes resource.
type Class schema.GroupVersionKind

// Object is a Go struct type representing a serialized Kubernetes resource.
type Object client.Object

// Query represents a Kubernetes resource query.
type Query struct {
	// Namespace restricts the search to a namespace.
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
	// Labels restricts the search to objects with matching label values (optional)
	Labels client.MatchingLabels `json:"labels,omitempty"`
	// Fields restricts the search to objects with matching field values (optional)
	Fields client.MatchingFields `json:"fields,omitempty"`

	class Class
}

// Store implements a korrel8r.Store using the kubernetes API server.
type Store struct {
	c      client.Client
	base   *url.URL
	groups []schema.GroupVersion
}

// Validate interfaces
var (
	_ korrel8r.Domain   = Domain
	_ korrel8r.Class    = Class{}
	_ korrel8r.Object   = Object(nil)
	_ korrel8r.Query    = &Query{}
	_ console.Converter = &Store{}
)

// domain implementation
type domain struct{}

func (d domain) Name() string        { return "k8s" }
func (d domain) String() string      { return d.Name() }
func (d domain) Description() string { return "Resource objects in a Kubernetes API server" }
func (d domain) Store(sc korrel8r.StoreConfig) (s korrel8r.Store, err error) {
	client, cfg, err := NewClient()
	if err != nil {
		return nil, err
	}
	return NewStore(client, cfg)
}

func (d domain) Class(name string) korrel8r.Class {
	var gvk schema.GroupVersionKind
	s := ""
	ok := false
	if gvk.Kind, s, ok = strings.Cut(name, "."); !ok { // Just Kind
		return classForGK(gvk.GroupKind())
	}
	if gvk.Version, gvk.Group = s, ""; Scheme.Recognizes(gvk) { // Kind.Version
		return Class(gvk)
	}
	if gvk.Version, gvk.Group, ok = strings.Cut(s, "."); ok && Scheme.Recognizes(gvk) { // s == Kind.Version.Group
		return Class(gvk)
	}
	gvk.Version, gvk.Group = "", s // s == Kind.Group
	return classForGK(gvk.GroupKind())
}

func classForGK(gk schema.GroupKind) korrel8r.Class {
	if versions := Scheme.VersionsForGroupKind(gk); len(versions) > 0 {
		return Class(gk.WithVersion(versions[0].Version))
	}
	return nil
}

func (d domain) Classes() (classes []korrel8r.Class) {
	for gvk := range Scheme.AllKnownTypes() {
		classes = append(classes, Class(gvk))
	}
	return classes
}

func (d domain) Query(s string) (korrel8r.Query, error) {
	var q Query
	c, err := impl.UnmarshalQueryString(d, s, &q)
	if err != nil {
		return nil, err
	}
	q.class = c.(Class)
	return &q, nil
}

// ClassOf returns the Class of o, which must be a pointer to a typed API resource struct.
func ClassOf(o client.Object) Class { return Class(GroupVersionKind(o)) }

func ClassOfAPIVersionKind(apiVersion, kind string) Class {
	return Class(schema.FromAPIVersionAndKind(apiVersion, kind))
}

// GroupVersionKind returns the GVK of o, which must be a pointer to a typed API resource struct.
// Returns empty if o is not a known resource type.
func GroupVersionKind(o client.Object) schema.GroupVersionKind {
	if gvks, _, err := Scheme.ObjectKinds(o); err == nil {
		return gvks[0]
	}
	return schema.GroupVersionKind{}
}

func (c Class) ID(o korrel8r.Object) any {
	if o, _ := o.(client.Object); o != nil {
		return client.ObjectKeyFromObject(o)
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

func (c Class) Domain() korrel8r.Domain { return Domain }
func (c Class) New() korrel8r.Object {
	if o, err := Scheme.New(schema.GroupVersionKind(c)); err == nil {
		return o
	}
	return nil
}
func (c Class) Name() string   { return fmt.Sprintf("%v.%v.%v", c.Kind, c.Version, c.Group) }
func (c Class) String() string { return korrel8r.ClassName(c) }

func (c Class) Description() string {
	// k8s objects have SwaggerDoc() method that is not declared on the Object interface.
	if o, _ := c.New().(interface{ SwaggerDoc() map[string]string }); o != nil {
		// Result is a map of property decriptions, where "" is mapped to the overall type description.
		return o.SwaggerDoc()[""]
	}
	return ""
}

func (c Class) GVK() schema.GroupVersionKind { return schema.GroupVersionKind(c) }

func NewQuery(c Class, namespace, name string, labels, fields map[string]string) *Query {
	return &Query{
		class:     c,
		Namespace: namespace,
		Name:      name,
		Labels:    labels,
		Fields:    fields,
	}
}

func (q Query) Class() korrel8r.Class { return q.class }
func (q Query) Query() string         { return korrel8r.JSONString(q) }
func (q Query) String() string        { return korrel8r.QueryName(q) }

// NewStore creates a new k8s store.
func NewStore(c client.Client, cfg *rest.Config) (korrel8r.Store, error) {
	host := cfg.Host
	if host == "" {
		host = "localhost"
	}
	base, _, err := rest.DefaultServerURL(host, cfg.APIPath, schema.GroupVersion{}, true)

	// TODO should be using discovery client?
	groups := Scheme.PreferredVersionAllGroups()
	slices.SortFunc(groups, func(a, b schema.GroupVersion) int { // Move core and openshift to front.
		if a.Group == "" || (strings.Contains(a.Group, ".openshift.io/") && b.Group != "") {
			return -1
		}
		return 0
	})

	return &Store{c: c, base: base, groups: groups}, err
}

func (s Store) Domain() korrel8r.Domain { return Domain }
func (s Store) Client() client.Client   { return s.c }

func (s *Store) Get(ctx context.Context, query korrel8r.Query, result korrel8r.Appender) (err error) {
	q, err := impl.TypeAssert[*Query](query)
	if err != nil {
		return err
	}
	if q.Name != "" { // Request for single object.
		return s.getObject(ctx, q, result)
	} else {
		return s.getList(ctx, q, result)
	}
}

func setMeta(o Object) Object {
	gvk := must.Must1(apiutil.GVKForObject(o, Scheme))
	o.GetObjectKind().SetGroupVersionKind(gvk)
	return o
}

func (s *Store) getObject(ctx context.Context, q *Query, result korrel8r.Appender) error {
	o, err := Scheme.New(q.class.GVK())
	if err != nil {
		return err
	}
	co, _ := o.(client.Object)
	if co == nil {
		return fmt.Errorf("invalid client.Object: %T", o)
	}
	err = s.c.Get(ctx, NamespacedName(q.Namespace, q.Name), co)
	if err != nil {
		return err
	}
	result.Append(setMeta(co))
	return nil
}

func (s *Store) getList(ctx context.Context, q *Query, result korrel8r.Appender) error {
	gvk := q.class.GVK()
	gvk.Kind = gvk.Kind + "List"
	o, err := Scheme.New(gvk)
	if err != nil {
		return err
	}
	list, _ := o.(client.ObjectList)
	if list == nil {
		return fmt.Errorf("invalid list object %T", o)
	}
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
	if err := s.c.List(ctx, list, opts...); err != nil {
		return err
	}
	defer func() { // Handle reflect panics.
		if r := recover(); r != nil && err == nil {
			err = fmt.Errorf("invalid list object: %T", list)
		}
	}()
	items := reflect.ValueOf(list).Elem().FieldByName("Items")
	for i := 0; i < items.Len(); i++ {
		result.Append(setMeta(items.Index(i).Addr().Interface().(client.Object)))
	}
	return nil
}

func (s *Store) resource(gvk schema.GroupVersionKind) (schema.GroupVersionResource, error) {
	rm, err := s.c.RESTMapper().RESTMappings(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return schema.GroupVersionResource{}, err
	}
	if len(rm) == 0 {
		return schema.GroupVersionResource{}, fmt.Errorf("no resource mapping found for: %v", gvk)
	}
	return rm[0].Resource, nil
}

func (s *Store) QueryToConsoleURL(query korrel8r.Query) (*url.URL, error) {
	q, err := impl.TypeAssert[*Query](query)
	if err != nil {
		return nil, err
	}
	gvr, err := s.resource(q.class.GVK())
	if err != nil {
		return nil, err
	}
	var u url.URL
	switch {
	case q.class.GVK() == eventGVK && len(q.Fields) != 0:
		return s.eventQueryToConsoleURL(q) // Special case
	case len(q.Labels) > 0: // Label search
		// Search using label selector
		u.Path = path.Join("search", "ns", q.Namespace) // TODO non-namespaced searches?
		v := url.Values{}
		gvk := q.class.GVK()
		v.Add("kind", fmt.Sprintf("%v~%v~%v", gvk.Group, gvk.Version, gvk.Kind))
		v.Add("q", selectorString(q.Labels))
		u.RawQuery = v.Encode()
	default: // Named resource
		if q.Namespace != "" { // Namespaced resource
			u.Path = path.Join("k8s", "ns", q.Namespace, gvr.Resource, q.Name)
		} else { // Cluster resource
			u.Path = path.Join("k8s", "cluster", gvr.Resource, q.Name)
		}
	}
	return &u, nil
}

const (
	// Event.involvedObject field names
	iKind       = "involvedObject.kind"
	iName       = "involvedObject.name"
	iNamespace  = "involvedObject.namespace"
	iAPIVersion = "involvedObject.apiVersion"
)

func (s *Store) eventQueryToConsoleURL(q *Query) (*url.URL, error) {
	gv, err := schema.ParseGroupVersion(q.Fields[iAPIVersion])
	if err != nil {
		return nil, err
	}
	u, err := s.QueryToConsoleURL(&Query{ // URL for involved object
		class:     Class(gv.WithKind(q.Fields[iKind])),
		Namespace: q.Fields[iNamespace],
		Name:      q.Fields[iName],
	})
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(u.Path, "events")
	return u, nil
}

var eventGVK = schema.GroupVersionKind{Version: "v1", Kind: "Event"}

func (s *Store) ConsoleURLToQuery(u *url.URL) (korrel8r.Query, error) {
	namespace, resource, name, events, err := parsePath(u)
	if err != nil {
		return nil, err
	}
	if resource == "projects" { // Openshift alias for namespace
		resource = "namespaces"
	}

	uq := u.Query()
	var gvk schema.GroupVersionKind
	switch {
	case strings.Contains(resource, "~"):
		gvk = parseGVK(resource)
	case resource != "":
		gvks, err := s.c.RESTMapper().KindsFor(schema.GroupVersionResource{Resource: resource})
		if err != nil {
			return nil, err
		}
		gvk = gvks[0]
	default:
		gvk = parseGVK(uq.Get("kind"))
	}
	if gvk.Version == "" { // Fill in a partial GVK
		rm, err := s.c.RESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return nil, err
		}
		gvk = rm.GroupVersionKind
	}
	if events { // Query events involving the object, not the object itself
		q := &Query{
			class: Class(eventGVK),
			Fields: map[string]string{
				iNamespace:  namespace,
				iName:       name,
				iAPIVersion: gvk.GroupVersion().String(),
				iKind:       gvk.Kind,
			}}
		return q, nil
	} else {
		q := Query{Namespace: namespace, Name: name, class: Class(gvk)}
		if labels := uq.Get("q"); labels != "" {
			if q.Labels, err = parseSelector(labels); err != nil {
				return nil, err
			}
		}
		return &q, nil
	}
}

func NamespacedName(namespace, name string) types.NamespacedName {
	return types.NamespacedName{Namespace: namespace, Name: name}
}
