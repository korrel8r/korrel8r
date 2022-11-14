// package rules is a test-only package to verify the rules.yaml files give expected results.
package rules

import (
	"context"
	"net/url"
	"os"
	"testing"

	"github.com/korrel8/korrel8/internal/pkg/decoder"
	"github.com/korrel8/korrel8/pkg/engine"
	"github.com/korrel8/korrel8/pkg/k8s"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/korrel8/korrel8/pkg/templaterule"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta/testrestmapper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func setup(t *testing.T) (client.Client, *engine.Engine) {
	t.Helper()
	e := engine.New("test")
	c := fake.NewClientBuilder().WithRESTMapper(testrestmapper.TestOnlyStaticRESTMapper(k8s.Scheme)).Build()
	s, err := k8s.NewStore(c)
	require.NoError(t, err)
	e.AddDomain(k8s.Domain, s)
	return c, e
}

func TestEventRules(t *testing.T) {
	c, e := setup(t)
	f, err := os.Open("k8s.yaml")
	require.NoError(t, err)
	defer f.Close()
	templaterule.AddRules(decoder.New(f), e)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: "default",
		}}
	event := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "some-event",
			Namespace: "default",
		},
		InvolvedObject: corev1.ObjectReference{
			Kind:       "Pod",
			Namespace:  "default",
			Name:       "foo",
			APIVersion: "v1",
		}}
	require.NoError(t, c.Create(ctx, pod))
	require.NoError(t, c.Create(ctx, event))

	t.Run("PodToEvent", func(t *testing.T) {
		paths, err := e.Graph().ShortestPaths(k8s.ClassOf(pod), k8s.ClassOf(event))
		require.NoError(t, err)
		queries, err := e.FollowAll(ctx, []korrel8.Object{pod}, nil, paths)
		require.NoError(t, err)
		want := []korrel8.Query{{
			Path:     "/api/v1/events",
			RawQuery: url.Values{"fieldSelector": []string{"involvedObject.name=foo,involvedObject.namespace=default"}}.Encode()}}
		require.NotEmpty(t, queries)
		assert.Equal(t, want, queries, "%v != %v", &want[0], &queries[0])
	})

	t.Run("EventToPod", func(t *testing.T) {
		paths, err := e.Graph().ShortestPaths(k8s.ClassOf(event), k8s.ClassOf(pod))
		require.NoError(t, err)
		queries, err := e.FollowAll(ctx, []korrel8.Object{event}, nil, paths)
		require.NoError(t, err)
		want := []korrel8.Query{{Path: "/api/v1/namespaces/default/pods/foo"}}
		require.NotEmpty(t, queries)
		assert.Equal(t, want, queries, "%v != %v", &want[0], &queries[0])
	})
}

var ctx = context.Background()
