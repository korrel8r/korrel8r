// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func TestTokenReviewCluster(t *testing.T) {
	cfg, err := config.GetConfig()
	require.NoError(t, err)
	require.NotEmpty(t, cfg.BearerToken, "cluster config must have a bearer token")

	tr, err := NewTokenReview()
	require.NoError(t, err)

	username, err := tr.User(cfg.BearerToken)
	require.NoError(t, err)
	assert.NotEmpty(t, username, "should resolve bearer token to a username")
	t.Logf("TokenReview resolved to username: %s", username)
}

func TestTokenReviewCluster_InvalidToken(t *testing.T) {
	_, err := config.GetConfig()
	require.NoError(t, err)

	tr, err := NewTokenReview()
	require.NoError(t, err)

	_, err = tr.User("invalid-token-that-does-not-exist")
	assert.Error(t, err, "invalid token should fail")
}
