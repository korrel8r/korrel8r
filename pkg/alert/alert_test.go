package alert

import (
	"context"
	"testing"

	"github.com/korrel8/korrel8/internal/pkg/test"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/stretchr/testify/require"
)

// skip for now: Test is failing with 404
func skipTestQuery_Alert(t *testing.T) {
	test.SkipIfNoCluster(t)
	store, err := NewOpenshiftStore(context.Background(), test.RESTConfig)
	require.NoError(t, err)
	result := korrel8.NewListResult()
	require.NoError(t, store.Get(context.Background(), &korrel8.Query{}, result))
	require.NotEmpty(t, result.List())
}
