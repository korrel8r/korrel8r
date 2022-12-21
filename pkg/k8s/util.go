package k8s

import (
	"context"

	"github.com/korrel8/korrel8/internal/pkg/test"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

// clientObjectStruct is a constraint for types T where *T implements client.Object, e.g. corev1.Pod
type clientObjectStruct[T any] interface {
	client.Object
	*T
}

func New[T any, PT clientObjectStruct[T]](namespace, name string) PT {
	var (
		t T
		o PT = &t
	)
	gvk := test.Must(apiutil.GVKForObject(o, Scheme))
	o.GetObjectKind().SetGroupVersionKind(gvk)
	o.SetNamespace(namespace)
	o.SetName(name)
	return o
}

func EventFor(o client.Object, name string) *corev1.Event {
	gvk := o.GetObjectKind().GroupVersionKind()
	e := New[corev1.Event](name, o.GetNamespace())
	e.InvolvedObject = corev1.ObjectReference{
		Kind:       gvk.Kind,
		Namespace:  o.GetNamespace(),
		Name:       o.GetName(),
		APIVersion: gvk.GroupVersion().String(),
	}
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
