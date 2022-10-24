// package k8s is a Kubernetes implementation of the korrel8 interfaces
package k8s

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"reflect"
	"regexp"
	"strings"

	"github.com/korrel8/korrel8/pkg/korrel8"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type domain struct{}

func (d domain) String() string { return "k8s" }

var Domain = domain{}

// parseGVK parse name of the form: kind[.version[.group]]
// TODO is this in the k8s libs?
func parseGVK(name string) (gvk schema.GroupVersionKind) {
	parts := strings.SplitN(name, ".", 3)
	for i, f := range []*string{&gvk.Kind, &gvk.Version, &gvk.Group} {
		if len(parts) > i {
			*f = parts[i]
		}
	}
	return gvk
}

func (d domain) Class(name string) korrel8.Class {
	gvk := parseGVK(name)
	var t reflect.Type
	if gvk.Group != "" { // Fully qualified, direct lookup
		return Class{Scheme.AllKnownTypes()[gvk]}
	} else { // No version
		gvs := Scheme.PreferredVersionAllGroups()
		for _, gv := range gvs {
			if t = Scheme.KnownTypes(gv)[gvk.Kind]; t != nil {
				return Class{t}
			}
		}
	}
	return nil
}

func (d domain) KnownClasses() (classes []korrel8.Class) {
	for _, t := range Scheme.AllKnownTypes() {
		classes = append(classes, Class{t})
	}
	return classes
}
func (d domain) NewQuery() korrel8.Query { q := Query(""); return &q }

var _ korrel8.Domain = Domain // Implements interface

// TODO the Class implementation assumes all objects are pointers to the generated API struct.
// We could use scheme & GVK comparisons to generalize to untyped representations as well.

// Class ses the Go API struct type to identify a kind of resource.
type Class struct{ reflect.Type }

// ClassOf returns the Class of o, which must be a pointer to a typed API resource struct.
func ClassOf(o client.Object) Class { return Class{reflect.TypeOf(o).Elem()} }

func (c Class) Contains(o korrel8.Object) bool { return reflect.TypeOf(o) == c.Type }
func (c Class) Key(o korrel8.Object) any {
	co, _ := o.(client.Object)
	return client.ObjectKeyFromObject(co)
}
func (c Class) Domain() korrel8.Domain { return Domain }
func (c Class) New() korrel8.Object    { return reflect.New(c.Type).Interface() }

func (c Class) String() string {
	// TODO there must be an easier way...
	for gvk, t := range Scheme.AllKnownTypes() {
		if t == c.Type {
			if gvk.Group == "" { // Core group has empty group name
				return fmt.Sprintf("%v.%v", gvk.Kind, gvk.Version)
			}
			return fmt.Sprintf("%v.%v.%v", gvk.Kind, gvk.Version, gvk.Group)
		}
	}
	return c.Type.String()
}

// Query is a REST URI
type Query string

func (q *Query) REST(base *url.URL) *url.URL {
	u := *base
	u.Path = path.Join(u.Path, q.String())
	return &u
}

func (q *Query) String() string { return string(*q) }

func (q *Query) Browser(base *url.URL) *url.URL { panic("FIXME") }

type Object client.Object

// Store implements the korrel8.Store interface over a k8s API client.
type Store struct{ c client.Client }

// NewStore creates a new store
func NewStore(c client.Client) (*Store, error) { return &Store{c: c}, nil }

func (s *Store) Get(ctx context.Context, q korrel8.Query, result korrel8.Result) (err error) {
	query, ok := q.(*Query)
	if !ok {
		// FIXME extract common implementation stuff.
		return fmt.Errorf("%v store expects %T but got %T", Domain, query, q)
	}
	defer func() {
		if err != nil {
			err = fmt.Errorf("%w: executing %v query %q", err, Domain, query)
		}
	}()
	u, err := url.Parse(string(*query))
	if err != nil {
		return err
	}
	gvk, nsName, err := s.parseAPIPath(u.Path)
	if err != nil {
		return err
	}
	if nsName.Name != "" { // Request for single object.
		return s.getObject(ctx, gvk, nsName, result)
	} else {
		return s.getList(ctx, gvk, nsName.Namespace, u.Query(), result)
	}
}

// parsing a REST URI into components then using client.Client to recreate the REST query.
//
// TODO revisit: this is weirdly indirect - parse an API path to make a Client call which re-creates the API path.
// Should be able to use a REST client directly, but client.Client does REST client creation & caching
// and manages schema and RESTMapper stuff which I'm not sure I understand yet.
func (s *Store) parseAPIPath(path string) (gvk schema.GroupVersionKind, nsName types.NamespacedName, err error) {
	parts := k8sPathRegex.FindStringSubmatch(path)
	if len(parts) != pCount {
		return gvk, nsName, fmt.Errorf("invalid URI")
	}
	nsName.Namespace, nsName.Name = parts[pNamespace], parts[pName]
	gvr := schema.GroupVersionResource{Group: parts[pGroup], Version: parts[pVersion], Resource: parts[pResource]}
	gvk, err = s.c.RESTMapper().KindFor(gvr)
	return gvk, nsName, err
}

func (s *Store) getObject(ctx context.Context, gvk schema.GroupVersionKind, nsName types.NamespacedName, result korrel8.Result) error {
	scheme := s.c.Scheme()
	o, err := scheme.New(gvk)
	if err != nil {
		return err
	}
	co, _ := o.(client.Object)
	if co == nil {
		return fmt.Errorf("invalid client.Object: %T", o)
	}
	err = s.c.Get(ctx, nsName, co)
	if err != nil {
		return err
	}
	result.Append(co)
	return nil
}

func (s *Store) parseAPIQuery(q url.Values) (opts []client.ListOption, err error) {
	if s := q.Get("labelSelector"); s != "" {
		selector, err := labels.Parse(s)
		if err != nil {
			return nil, err
		}
		opts = append(opts, client.MatchingLabelsSelector{Selector: selector})
	}
	if s := q.Get("fieldSelector"); s != "" {
		selector, err := fields.ParseSelector(s)
		if err != nil {
			return nil, err
		}
		opts = append(opts, client.MatchingFieldsSelector{Selector: selector})
	}
	return opts, nil
}

func (s *Store) getList(ctx context.Context, gvk schema.GroupVersionKind, namespace string, query url.Values, result korrel8.Result) error {
	gvk.Kind = gvk.Kind + "List"
	o, err := s.c.Scheme().New(gvk)
	if err != nil {
		return err
	}
	list, _ := o.(client.ObjectList)
	if list == nil {
		return fmt.Errorf("invalid list object %T", o)
	}
	opts, err := s.parseAPIQuery(query)
	if err != nil {
		return err
	}
	if namespace != "" {
		opts = append(opts, client.InNamespace(namespace))
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
		result.Append(items.Index(i).Addr().Interface().(client.Object))
	}
	return nil
}

// Parse a K8s API path into: group, version, namespace, resourcetype, name.
// See: https://kubernetes.io/docs/reference/using-api/api-concepts/
var k8sPathRegex = regexp.MustCompile(`^(?:(?:/apis/([^/]+)/)|(?:/api/))([^/]+)(?:/namespaces/([^/]+))?/([^/]+)(?:/([^/]+))?`)

// Indices for match results from k8sPathRegex
const (
	pGroup = iota + 1
	pVersion
	pNamespace
	pResource
	pName
	pCount
)
