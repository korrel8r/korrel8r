// package k8s is a Kubernetes implementation of the korrel8r interfaces
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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"

	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

var (
	_ korrel8r.Domain   = Domain
	_ korrel8r.Class    = Class{}
	_ korrel8r.Query    = &Query{}
	_ console.Converter = &Store{}
)

// Domain is a korrel8r.Domain.
var Domain = domain{}

type domain struct{}

func (d domain) String() string { return "k8s" }

// Class name in one of the forms: Kind,  Kind.Group,  Kind.Version.Group.
// Group must be included, missing group implies core group.
func (d domain) Class(name string) korrel8r.Class {
	gvk, gk := schema.ParseKindArg(name)
	if gvk != nil && Scheme.Recognizes(*gvk) { // Direct hit
		return Class(*gvk)
	} else {
		if vs := Scheme.VersionsForGroupKind(gk); len(vs) > 0 {
			return Class(gk.WithVersion(vs[0].Version))
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

func (domain) UnmarshalQuery(r []byte) (korrel8r.Query, error) {
	return impl.UnmarshalQuery(r, &Query{})
}

// Class implements korrel8r.Class
type Class schema.GroupVersionKind

// ClassOf returns the Class of o, which must be a pointer to a typed API resource struct.
func ClassOf(o client.Object) Class {
	if gvks, _, err := Scheme.ObjectKinds(o); err == nil {
		return Class(gvks[0])
	}
	return Class{}
}

func (c Class) ID(o korrel8r.Object) any {
	if o, _ := o.(client.Object); o != nil {
		return client.ObjectKeyFromObject(o)
	}
	return nil
}

func (c Class) Domain() korrel8r.Domain { return Domain }
func (c Class) New() korrel8r.Object {
	if o, err := Scheme.New(schema.GroupVersionKind(c)); err == nil {
		return o
	}
	return nil
}
func (c Class) String() string               { return fmt.Sprintf("%v.%v.%v", c.Kind, c.Version, c.Group) }
func (c Class) ShortString() string          { return c.Kind }
func (c Class) GVK() schema.GroupVersionKind { return schema.GroupVersionKind(c) }

type Object client.Object

type Query struct {
	schema.GroupVersionKind                       // `json:",omitempty"`
	types.NamespacedName                          // `json:",omitempty"`
	Labels                  client.MatchingLabels // `json:",omitempty"`
	Fields                  client.MatchingFields // `json:",omitempty"`
}

func NewQuery(c Class, namespace, name string, labels, fields map[string]string) *Query {
	return &Query{
		GroupVersionKind: c.GVK(),
		NamespacedName:   types.NamespacedName{Namespace: namespace, Name: name},
		Labels:           labels,
		Fields:           fields,
	}
}

func (q *Query) Class() korrel8r.Class { return Class(q.GroupVersionKind) }

// Store implements the korrel8r.Store interface as a k8s API client.
type Store struct {
	c      client.Client
	base   *url.URL
	groups []schema.GroupVersion
}

// NewStore creates a new store
func NewStore(c client.Client, cfg *rest.Config) (*Store, error) {
	host := cfg.Host
	if host == "" {
		host = "localhost"
	}
	base, _, err := rest.DefaultServerURL(host, cfg.APIPath, schema.GroupVersion{}, true)

	// TODO should be using discovery client?
	groups := Scheme.PreferredVersionAllGroups()
	slices.SortFunc(groups, func(a, b schema.GroupVersion) bool { // Move core and openshift to front.
		return a.Group == "" || (strings.Contains(a.Group, ".openshift.io/") && b.Group != "")
	})

	return &Store{c: c, base: base, groups: groups}, err
}

func (Store) Domain() korrel8r.Domain { return Domain }

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
	gvk := must.Must1(apiutil.GVKForObject(o, scheme.Scheme))
	o.GetObjectKind().SetGroupVersionKind(gvk)
	return o
}

func (s *Store) getObject(ctx context.Context, q *Query, result korrel8r.Appender) error {
	scheme := s.c.Scheme()
	o, err := scheme.New(q.GroupVersionKind)
	if err != nil {
		return err
	}
	co, _ := o.(client.Object)
	if co == nil {
		return fmt.Errorf("invalid client.Object: %T", o)
	}
	err = s.c.Get(ctx, q.NamespacedName, co)
	if err != nil {
		return err
	}
	result.Append(setMeta(co))
	return nil
}

func (s *Store) getList(ctx context.Context, q *Query, result korrel8r.Appender) error {
	gvk := q.GroupVersionKind
	gvk.Kind = gvk.Kind + "List"
	o, err := s.c.Scheme().New(gvk)
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
	gvr, err := s.resource(q.GroupVersionKind)
	if err != nil {
		return nil, err
	}
	var u url.URL
	switch {
	case q.GroupVersionKind == eventGVK && len(q.Fields) != 0:
		return s.eventQueryToConsoleURL(q) // Special case
	case len(q.Labels) > 0: // Label search
		// Search using label selector
		u.Path = path.Join("search", "ns", q.Namespace) // TODO non-namespaced searches?
		v := url.Values{}
		v.Add("kind", fmt.Sprintf("%v~%v~%v", q.Group, q.Version, q.Kind))
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
		GroupVersionKind: gv.WithKind(q.Fields[iKind]),
		NamespacedName:   NamespacedName(q.Fields[iNamespace], q.Fields[iName]),
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
			GroupVersionKind: eventGVK,
			Fields: map[string]string{
				iNamespace:  namespace,
				iName:       name,
				iAPIVersion: gvk.GroupVersion().String(),
				iKind:       gvk.Kind,
			}}
		return q, nil
	} else {
		q := Query{NamespacedName: NamespacedName(namespace, name), GroupVersionKind: gvk}
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
