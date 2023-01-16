package alert

import (
	"context"
	"testing"

	"github.com/korrel8/korrel8/internal/pkg/test"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/korrel8/korrel8/pkg/uri"
	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	// Dubious test, assumes there is an alert on the cluster.
	test.SkipIfNoCluster(t)
	store, err := NewOpenshiftAlertManagerStore(context.Background(), test.RESTConfig)
	require.NoError(t, err)
	result := korrel8.NewListResult()
	require.NoError(t, store.Get(context.Background(), uri.Reference{}, result))
	require.NotEmpty(t, result.List())
}
