// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package session

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/auth"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testFactory() (*engine.Engine, error) {
	return engine.Build().Domains(mock.NewDomain("mock")).Engine()
}

func tokenCtx(token string) context.Context {
	return auth.WithToken(context.Background(), token)
}

func getSession(t *testing.T, m Manager, token string) *Session {
	t.Helper()
	s, err := m.Get(tokenCtx(token))
	require.NoError(t, err)
	return s
}

func TestGet_SameKey(t *testing.T) {
	m := NewPool(time.Hour, testFactory)
	s1 := getSession(t, m, "key-a")
	s2 := getSession(t, m, "key-a")
	assert.Same(t, s1, s2, "same key should return same session")
}

func TestGet_DifferentKeys(t *testing.T) {
	m := NewPool(time.Hour, testFactory)
	s1 := getSession(t, m, "key-a")
	s2 := getSession(t, m, "key-b")
	assert.NotSame(t, s1, s2, "different keys should return different sessions")
}

func TestGet_HashedKey(t *testing.T) {
	m := NewPool(time.Hour, testFactory)
	s := getSession(t, m, "some-token")
	assert.Equal(t, hashToken("some-token"), s.ID, "should use hashed token as key")
}

func TestConcurrent(t *testing.T) {
	m := NewPool(time.Hour, testFactory)
	var wg sync.WaitGroup
	var count atomic.Int32
	for range 100 {
		wg.Go(func() {
			s, err := m.Get(tokenCtx("shared-token"))
			if err == nil && s != nil {
				count.Add(1)
			}
		})
	}
	wg.Wait()
	assert.Equal(t, int32(100), count.Load(), "all goroutines should get an engine")
}

func TestConcurrent_NewKey(t *testing.T) {
	m := NewPool(time.Hour, testFactory)

	// All goroutines race to create the same new key.
	// They should all get the same session.
	sessions := make([]*Session, 100)
	errs := make([]error, 100)
	var wg sync.WaitGroup
	for i := range 100 {
		wg.Go(func() {
			sessions[i], errs[i] = m.Get(tokenCtx("new-key"))
		})
	}
	wg.Wait()
	for i, err := range errs {
		require.NoError(t, err, "goroutine %d", i)
	}
	// All should return the same session instance.
	for i := 1; i < len(sessions); i++ {
		assert.Same(t, sessions[0], sessions[i], "goroutine %d got a different session", i)
	}
}

func TestCleanup_OneExpiredOneActive(t *testing.T) {
	timeout := 50 * time.Millisecond
	m := NewPool(timeout, testFactory)

	sOld := getSession(t, m, "old-token")

	// Wait long enough for old-token to expire.
	time.Sleep(timeout * 3)

	// Create a new session that should still be active.
	sNew := getSession(t, m, "new-token")

	// old-token should have expired and return a fresh session.
	sOldAgain := getSession(t, m, "old-token")
	assert.NotSame(t, sOld, sOldAgain, "expired session should be replaced")

	// new-token should still return the same session (not expired).
	sNewAgain := getSession(t, m, "new-token")
	assert.Same(t, sNew, sNewAgain, "active session should be retained")
}
