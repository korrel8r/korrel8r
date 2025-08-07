// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/korrel8r/korrel8r/pkg/ptr"
	"github.com/korrel8r/korrel8r/pkg/unique"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewRESTMapper returns a simple rest mapper for a static list of resources.
func NewRESTMapper(gvks ...schema.GroupVersionKind) *meta.DefaultRESTMapper {
	groups := unique.Set[schema.GroupVersion]{}
	for _, gvk := range gvks {
		groups.Add(gvk.GroupVersion())
	}
	rm := meta.NewDefaultRESTMapper(groups.List())
	for _, gvk := range gvks {
		rm.Add(gvk, meta.RESTScopeNamespace)
	}
	return rm
}

func WaitForPodReady(t *testing.T, c client.Client, namespace, name string) {
	err := wait.PollUntilContextTimeout(t.Context(), 5*time.Second, time.Minute, true,
		func(ctx context.Context) (ok bool, err error) {
			defer func() {
				if !ok || err != nil {
					t.Logf("waiting for pod %v/%v, err: %v", namespace, name, err)
				}
			}()
			pod := &corev1.Pod{}
			err = c.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, pod)
			switch {
			case err == nil:
				return IsPodReady(pod), nil
			case errors.IsNotFound(err):
				return false, nil
			default:
				return false, err
			}
		})
	require.NoError(t, err)
}

func IsPodReady(pod *corev1.Pod) bool {
	return slices.IndexFunc(pod.Status.Conditions, func(c corev1.PodCondition) bool {
		return c.Type == corev1.PodReady && c.Status == corev1.ConditionTrue
	}) >= 0
}

var TestLabels = map[string]string{"test": "korrel8r"}

func TempNamespace(t *testing.T, c client.Client, prefix string) *corev1.Namespace {
	t.Helper()
	namespace := prefix + RandomName(8)
	t.Logf("temporary namespace: %v", namespace)
	ns := &corev1.Namespace{ObjectMeta: v1.ObjectMeta{Name: namespace, Labels: TestLabels}}
	require.NoError(t, c.Create(t.Context(), ns))
	t.Cleanup(func() { _ = c.Delete(context.Background(), ns) })
	return ns
}

// DefaultSecurityContext required in openshift clusters.
var DefaultSecurityContext = &corev1.SecurityContext{
	SeccompProfile: &corev1.SeccompProfile{
		Type: corev1.SeccompProfileTypeRuntimeDefault,
	},
	AllowPrivilegeEscalation: ptr.To(false),
	Capabilities: &corev1.Capabilities{
		Drop: []corev1.Capability{"ALL"},
	},
}
