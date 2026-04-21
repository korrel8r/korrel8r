// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package session

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func TestTokenReviewCluster_SessionID(t *testing.T) {
	cfg, err := config.GetConfig()
	require.NoError(t, err)
	require.NotEmpty(t, cfg.BearerToken, "cluster config must have a bearer token")

	m := NewPool(0, testFactory)
	defer m.Close()

	// Use the real bearer token to get a session.
	ctx := tokenCtx(cfg.BearerToken)
	key := m.Key(ctx)
	s, err := m.Get(key)
	require.NoError(t, err)

	// Session ID should be the username, not a hash.
	assert.NotEqual(t, hashToken(cfg.BearerToken), s.Key,
		"with TokenReview available, session key should be username, not hashed token")
	t.Logf("Session key (username): %s, ID: %d", s.Key, s.ID)

	// Same token should return the same session.
	s2, err := m.Get(key)
	require.NoError(t, err)
	assert.Same(t, s, s2, "same token should return same session")
}
