// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package session

import (
	"testing"
	"time"

	"github.com/korrel8r/korrel8r/pkg/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func TestTokenReviewCluster_SessionID(t *testing.T) {
	cfg, err := config.GetConfig()
	require.NoError(t, err)
	require.NotEmpty(t, cfg.BearerToken, "cluster config must have a bearer token")

	tr, err := auth.NewTokenReview()
	require.NoError(t, err)
	m := NewTokenReviewManager(tr, time.Hour, testFactory)

	// Use the real bearer token to get a session.
	ctx := tokenCtx(cfg.BearerToken)
	s, err := m.Get(ctx)
	require.NoError(t, err)

	assert.NotEmpty(t, s.ID, "session ID should be a username")
	t.Logf("Session ID (username): %s", s.ID)

	// Same token should return the same session.
	s2, err := m.Get(tokenCtx(cfg.BearerToken))
	require.NoError(t, err)
	assert.Same(t, s, s2, "same token should return same session")
}
