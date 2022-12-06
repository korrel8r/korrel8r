// package k8s is a Kubernetes implementation of the korrel8 interfaces
package k8s

/// FIXME move this back to the domains as optional store functions?

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"regexp"

	"github.com/korrel8/korrel8/pkg/korrel8"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type domain struct{}

func (d domain) String() string { return "k8s" }

// Class name in one of the forms: Kind,  Kind.Group,  Kind.Version.Group.
func (d domain) Class(name string) korrel8.Class {
	tryGVK, tryGK := schema.ParseKindArg(name)
	switch {
	case tryGVK != nil && Scheme.Recognizes(*tryGVK): // Direct hit
		return Class(*tryGVK)
	case tryGK.Group != "": // GroupKind, must find version
		for _, gv := range Scheme.VersionsForGroupKind(tryGK) {
			if gvk := tryGK.WithVersion(gv.Version); Scheme.Recognizes(gvk) {
				return Class(gvk)
			}
		}
	default: // Only have a Kind, search for group and version.
		for _, gv := range Scheme.PreferredVersionAllGroups() {
			if gvk := gv.WithKind(tryGK.Kind); Scheme.Recognizes(gvk) {
				return Class(gvk)
			}
		}
	}
	return nil
}

func (d domain) Classes() (classes []korrel8.Class) {
	for gvk := range Scheme.AllKnownTypes() {
		classes = append(classes, Class(gvk))
	}
	return classes
}

var Domain korrel8.Domain = domain{} // Implements interface

// TODO the Class implementation assumes all objects are pointers to the generated API struct.
// We could use scheme & GVK comparisons to generalize to untyped representations as well.

// Class is a k8s GroupVersionKind.
type Class schema.GroupVersionKind

// ClassOf returns the Class of o, which must be a pointer to a typed API resource struct.
func ClassOf(o client.Object) korrel8.Class {
	if gvks, _, err := Scheme.ObjectKinds(o); err == nil {
		return Class(gvks[0])
	}
	return nil
}

func (c Class) Key(o korrel8.Object) any { return c }
func (c Class) Domain() korrel8.Domain   { return Domain }
func (c Class) New() korrel8.Object {
	if o, err := Scheme.New(schema.GroupVersionKind(c)); err == nil {
		return o
	}
	return nil
}

func (c Class) String() string { return fmt.Sprintf("%v.%v.%v", c.Kind, c.Version, c.Group) }

type Object client.Object

// Store implements the korrel8.Store interface as a k8s API client.
type Store struct {
	c   client.Client
	cfg *rest.Config
}

// NewStore creates a new store
func NewStore(c client.Client, cfg *rest.Config) (*Store, error) { return &Store{c: c, cfg: cfg}, nil }

func (s *Store) Get(ctx context.Context, query *korrel8.Query, result korrel8.Result) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("%v query error: %w: query= %v", Domain, err, query)
		}
	}()
	gvk, nsName, err := s.ParseQuery(query.Path)
	if err != nil {
		return err
	}
	if nsName.Name != "" { // Request for single object.
		return s.getObject(ctx, gvk, nsName, result)
	} else {
		return s.getList(ctx, gvk, nsName.Namespace, query.Query(), result)
	}
}

func (s Store) URL(q *korrel8.Query) *url.URL {
	u := url.URL{Scheme: "https", Host: s.cfg.Host}
	return u.ResolveReference(q)
}

// parsing a REST URI into components then using client.Client to recreate the REST query.
//
// TODO revisit: this is weirdly indirect - parse an API path to make a Client call which re-creates the API path.
// Should be able to use a REST client directly, but client.Client does REST client creation & caching
// and manages schema and RESTMapper stuff which I'm not sure I understand yet.
func (s *Store) ParseQuery(path string) (gvk schema.GroupVersionKind, nsName types.NamespacedName, err error) {
	parts := k8sPathRegex.FindStringSubmatch(path)
	if len(parts) != pCount {
		return gvk, nsName, fmt.Errorf("invalid k8s REST path: %v", path)
	}
	nsName.Namespace, nsName.Name = parts[pNamespace], parts[pName]
	gvr := schema.GroupVersionResource{Group: parts[pGroup], Version: parts[pVersion], Resource: parts[pResource]}
	gvk, err = s.c.RESTMapper().KindFor(gvr)
	return gvk, nsName, err
}

func (s *Store) ClassFor(resource string) korrel8.Class {
	gvks, err := s.c.RESTMapper().KindsFor(schema.GroupVersionResource{Resource: resource})
	if err != nil || len(gvks) == 0 {
		return nil
	}
	return Class(gvks[0])
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

// ToConsole converts a k8s query to a console query
func ToConsole(q *korrel8.Query) (*url.URL, error) {
	parts := k8sPathRegex.FindStringSubmatch(q.Path)
	if len(parts) != pCount {
		return nil, fmt.Errorf("invalid k8s query: %v", q)
	}
	var ns string
	if parts[pNamespace] != "" {
		ns = "/ns/" + parts[pNamespace]
	}
	return &url.URL{Path: fmt.Sprintf("/k8s/%v/%v/%v", ns, parts[pResource], parts[pName])}, nil
}
