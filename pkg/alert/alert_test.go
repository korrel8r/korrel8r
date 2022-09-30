package alert

import (
	"context"
	"testing"

	"github.com/alanconway/korrel8/internal/pkg/test"
	"github.com/stretchr/testify/require"
)

func TestQuery_Alert(t *testing.T) {
	// Dubious test, assumes there is an alert on the cluster.
	test.SkipIfNoCluster(t)
	store, err := OpenshiftManagerStore(test.RESTConfig)
	require.NoError(t, err)
	alerts, err := store.Query(context.Background(), "")
	require.NoError(t, err)
	require.NotEmpty(t, alerts)
}
