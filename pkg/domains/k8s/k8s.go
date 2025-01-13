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
	"fmt"
	"net/url"
	"reflect"
	"strings"

	"github.com/korrel8r/korrel8r/internal/pkg/must"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/korrel8r/impl"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

// Domain for Kubernetes resources stored in a Kube API server.
var Domain = domain{}

// Class represents a kind of kubernetes resource.
//
// The format of a class name is: "k8s:KIND.VERSION.GROUP".
// VERSION and GROUP are optional if there is no ambiguity.
//
// Examples: `k8s:Pod.v1`, `ks8:Pod`, `k8s:Deployment.v1.apps`, `k8s:Deployment.apps`, `k8s:Deployment`
type Class schema.GroupVersionKind

// Object is a struct type representing a Kubernetes resource.
//
// Object can be one of the of the standard k8s types from [k8s.io/api/core],
// or a generated custom resource type.
//
// Rules templates should use capitalized Go field names rather than the lowercase JSON field names.
type Object client.Object

// Query struct for a Kubernetes query.
//
// Example:
//
//	k8s:Pod.v1.:{"namespace":"openshift-cluster-version","name":"cluster-version-operator-8d86bcb65-btlgn"}
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

// Stores presents the Kubernetes API server as a korrel8r.Store.
//
// The k8s domain automatically connects to the current cluster (as determined by kubectl),
// no additional configuration is needed.
//
//	 stores:
//		  domain: k8s
type Store struct {
	c    client.Client
	base *url.URL
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

func (d domain) Name() string        { return "k8s" }
func (d domain) String() string      { return d.Name() }
func (d domain) Description() string { return "Resource objects in a Kubernetes API server" }
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

func (d domain) Class(name string) korrel8r.Class {
	var gvk schema.GroupVersionKind
	s := ""
	ok := false
	if gvk.Kind, s, ok = strings.Cut(name, "."); !ok { // Just Kind
		return classForKind(gvk.Kind)
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

func classForKind(kind string) korrel8r.Class {
	for _, gv := range Scheme.PrioritizedVersionsAllGroups() {
		gvk := gv.WithKind(kind)
		if Scheme.Recognizes(gvk) {
			return Class(gvk)
		}
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
	class, query, err := impl.UnmarshalQueryString[Query](d, s)
	if err != nil {
		return nil, err
	}
	query.class = class.(Class)
	return &query, nil
}

// ClassOf returns the Class of o, which must be a pointer to a typed API resource struct.
func ClassOf(o client.Object) Class { return Class(GroupVersionKind(o)) }

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
func (c Class) Unmarshal(b []byte) (korrel8r.Object, error) {
	if o, err := Scheme.New(schema.GroupVersionKind(c)); err == nil {
		err := json.Unmarshal(b, &o)
		return o, err
	}
	return nil, fmt.Errorf("unknown k8s type: %v", c)
}
func (c Class) Name() string                 { return fmt.Sprintf("%v.%v.%v", c.Kind, c.Version, c.Group) }
func (c Class) String() string               { return impl.ClassString(c) }
func (c Class) GVK() schema.GroupVersionKind { return schema.GroupVersionKind(c) }

func NewQuery(c Class, namespace, name string, labels, fields map[string]string) *Query {
	return &Query{
		Namespace: namespace,
		Name:      name,
		Labels:    labels,
		Fields:    fields,
		class:     c,
	}
}

func (q Query) Class() korrel8r.Class        { return q.class }
func (q Query) Data() string                 { b, _ := json.Marshal(q); return string(b) }
func (q Query) String() string               { return impl.QueryString(q) }
func (q Query) GVK() schema.GroupVersionKind { return q.class.GVK() }

// NewStore creates a new k8s store.
func NewStore(c client.Client, cfg *rest.Config) (korrel8r.Store, error) {
	host := cfg.Host
	if host == "" {
		host = "localhost"
	}
	base, _, err := rest.DefaultServerURL(host, cfg.APIPath, schema.GroupVersion{}, true)
	return &Store{c: c, base: base}, err
}

func (s Store) Domain() korrel8r.Domain { return Domain }
func (s Store) Client() client.Client   { return s.c }

func (s *Store) Get(ctx context.Context, query korrel8r.Query, c *korrel8r.Constraint, result korrel8r.Appender) (err error) {
	defer func() {
		if errors.IsNotFound(err) {
			err = nil // Finding nothing is not an error.
		}
	}()

	q, err := impl.TypeAssert[*Query](query)
	if err != nil {
		return err
	}
	appender := korrel8r.AppenderFunc(func(o korrel8r.Object) {
		// Include only objects created before or during the constraint interval.
		if c.CompareTime(o.(Object).GetCreationTimestamp().Time) <= 0 {
			result.Append(o)
		}
	})
	if q.Name != "" { // Request for single object.
		return s.getObject(ctx, q, appender)
	} else {
		return s.getList(ctx, q, appender, c)
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

func (s *Store) getList(ctx context.Context, q *Query, result korrel8r.Appender, c *korrel8r.Constraint) error {
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
	items := reflect.ValueOf(list).Elem().FieldByName("Items")
	for i := 0; i < items.Len(); i++ {
		result.Append(setMeta(items.Index(i).Addr().Interface().(client.Object)))
	}
	return nil
}

func NamespacedName(namespace, name string) types.NamespacedName {
	return types.NamespacedName{Namespace: namespace, Name: name}
}
