package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/meta/testrestmapper"
)

func TestKindToResource(t *testing.T) {
	rm := testrestmapper.TestOnlyStaticRESTMapper(Scheme)
	for _, tc := range [][]string{
		{"pods", "Pod", "v1"},
		{"pods", "Pod", ""},
		{"deployments", "Deployment", "apps/v1"},
		{"events", "Event", "events.k8s.io/v1"},
		{"events", "Event", "v1"},
	} {
		resource, err := kindToResource(rm, tc[1], tc[2])
		if assert.NoError(t, err) {
			assert.Equal(t, tc[0], resource)
		}
	}
}
