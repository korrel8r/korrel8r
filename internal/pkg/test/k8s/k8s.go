// package k8s contains tools for building k8s objects in memory, useful for tests.
package k8s

import (
	"context"
	"fmt"

	"github.com/korrel8/korrel8/internal/pkg/test"
	"github.com/korrel8/korrel8/pkg/k8s"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

type Builder[T client.Object] struct{ o T }

func Build[T client.Object](o T) Builder[T] {
	gvk := test.Must(apiutil.GVKForObject(o, k8s.Scheme))
	o.GetObjectKind().SetGroupVersionKind(gvk)
	return Builder[T]{o: o}
}

func (b Builder[T]) Object() T { return b.o }

func (b Builder[T]) NSName(namespace, name string) Builder[T] {
	b.o.SetNamespace(namespace)
	b.o.SetName(name)
	return b
}

var eventID int

func EventFor(o client.Object) *corev1.Event {
	gvk := o.GetObjectKind().GroupVersionKind()
	id := fmt.Sprint(eventID)
	eventID++
	e := &corev1.Event{
		InvolvedObject: corev1.ObjectReference{
			Kind:       gvk.Kind,
			Namespace:  o.GetNamespace(),
			Name:       o.GetName(),
			APIVersion: gvk.GroupVersion().String(),
		}}
	_ = Build(e).NSName(id, o.GetNamespace())
	return e
}

func Create(c client.Client, objs ...client.Object) error {
	for _, o := range objs {
		if err := c.Create(context.Background(), o); err != nil {
			return err
		}
	}
	return nil
}
