// package k8s is a Kubernetes implementation of the korrel8 interfaces.
package k8s

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/alanconway/korrel8/pkg/korrel8"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const Domain = "k8s"

// Class implements korrel8.Class as a k8s GroupVersionKind
type Class schema.GroupVersionKind

var _ korrel8.Class = Class{}

func (c Class) String() string { return schema.GroupVersionKind(c).String() }

func (c Class) Contains(x any) bool {
	o, _ := x.(runtime.Object)
	gvks, _, _ := scheme.Scheme.ObjectKinds(o)
	for _, gvk := range gvks {
		if gvk == schema.GroupVersionKind(c) {
			return true
		}
	}
	return false
}

func ClassOf(o runtime.Object) (Class, error) {
	gvks, _, err := scheme.Scheme.ObjectKinds(o)
	if err != nil || len(gvks) == 0 {
		return Class{}, fmt.Errorf("not a k8s object: %T(%#v)", o, o)
	}
	return Class(gvks[0]), nil
}

func Decode(data []byte) (runtime.Object, error) {
	o, _, err := scheme.Codecs.UniversalDeserializer().Decode(data, nil, nil)
	return o, err
}

func Encode(o runtime.Object) ([]byte, error) {
	class, err := ClassOf(o)
	if err != nil {
		return nil, err
	}
	gvk := schema.GroupVersionKind(class)
	cf := scheme.Codecs
	info, _ := runtime.SerializerInfoForMediaType(cf.SupportedMediaTypes(), runtime.ContentTypeJSON)
	encoder := info.Serializer
	codec := cf.CodecForVersions(encoder, nil, schema.GroupVersion{Group: gvk.Group, Version: gvk.Version}, nil)
	return runtime.Encode(codec, o)
}

type Store struct{ kc client.Client }

func NewStore(kc client.Client) Store { return Store{kc: kc} }

// Execute accepts several types of query string.
// FIXME initially just takes Kind.
func (s Store) Execute(query korrel8.Query) (result []any, err error) {
	list := listOf(findKind(string(query)))
	if list == nil {
		return nil, fmt.Errorf("invalid %v query: %v", Domain, query)
	}
	if err := s.kc.List(context.Background(), list); err != nil {
		return nil, err
	}
	defer func() { // Handle reflect panics.
		if r := recover(); r != nil && err == nil {
			err = fmt.Errorf("cannot execute query %v: %v", query, r)
		}
	}()
	items := reflect.ValueOf(list).Elem().FieldByName("Items")
	result = []any{}
	for i := 0; i < items.Len(); i++ {
		result = append(result, items.Index(i).Interface())
	}
	return result, nil
}

func listOf(gvk schema.GroupVersionKind) client.ObjectList {
	gvk.Kind = gvk.Kind + "List"
	listType := scheme.Scheme.KnownTypes(gvk.GroupVersion())[gvk.Kind]
	list, _ := reflect.New(listType).Interface().(client.ObjectList)
	return list
}

func findKind(kind string) schema.GroupVersionKind {
	for gvk := range scheme.Scheme.AllKnownTypes() {
		if strings.EqualFold(gvk.Kind, kind) {
			return gvk
		}
	}
	return schema.GroupVersionKind{}
}
