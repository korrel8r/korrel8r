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
	t.Skip("temporary skip: failing with 404")
	test.SkipIfNoCluster(t)
	store, err := NewOpenshiftStore(context.Background(), test.RESTConfig)
	require.NoError(t, err)
	result := korrel8.NewListResult()
	require.NoError(t, store.Get(context.Background(), uri.Reference{}, result))
	require.NotEmpty(t, result.List())
}
