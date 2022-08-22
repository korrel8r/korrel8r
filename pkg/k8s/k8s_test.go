package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestClassOf(t *testing.T) {
	c, err := ClassOf(&corev1.Pod{})
	assert.NoError(t, err)
	assert.Equal(t, c, Class{Group: "", Version: "v1", Kind: "Pod"})
	assert.True(t, c.Contains(&corev1.Pod{TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"}}))
	assert.False(t, c.Contains(&corev1.Service{}))
}

func TestEncodeDecode(t *testing.T) {
	want := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "where"},
		// No TypeMeta
	}
	data, err := Encode(want)
	assert.NoError(t, err)
	got, err := Decode(data)
	assert.NoError(t, err)
	// Encode fills in the TypeMeta, so fill it in for the "want" value
	want.SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"})
	assert.Equal(t, want, got)
}
