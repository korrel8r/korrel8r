// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package session

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testFactory() (*engine.Engine, config.Configs, error) {
	e, err := engine.Build().Domains(mock.NewDomain("mock")).Engine()
	return e, nil, err
}

func TestGet_SameKey(t *testing.T) {
	m := NewPool(time.Hour, testFactory)
	defer m.Close()
	s1, err := m.Get("key-a")
	require.NoError(t, err)
	s2, err := m.Get("key-a")
	require.NoError(t, err)
	assert.Same(t, s1, s2, "same key should return same session")
}

func TestGet_DifferentKeys(t *testing.T) {
	m := NewPool(time.Hour, testFactory)
	defer m.Close()
	s1, err := m.Get("key-a")
	require.NoError(t, err)
	s2, err := m.Get("key-b")
	require.NoError(t, err)
	assert.NotSame(t, s1, s2, "different keys should return different sessions")
}

func TestConcurrent(t *testing.T) {
	m := NewPool(time.Hour, testFactory)
	defer m.Close()

	var wg sync.WaitGroup
	var count atomic.Int32
	for range 100 {
		wg.Go(func() {
			e, err := m.Get("shared-token")
			if err == nil && e != nil {
				count.Add(1)
			}
		})
	}
	wg.Wait()
	assert.Equal(t, int32(100), count.Load(), "all goroutines should get an engine")
}

func TestConcurrent_NewKey(t *testing.T) {
	m := NewPool(time.Hour, testFactory)
	defer m.Close()

	// All goroutines race to create the same new key.
	// They should all get the same session.
	sessions := make([]*Session, 100)
	errs := make([]error, 100)
	var wg sync.WaitGroup
	for i := range 100 {
		wg.Go(func() {
			sessions[i], errs[i] = m.Get("new-key")
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
	defer m.Close()

	sOld, err := m.Get("old-token")
	require.NoError(t, err)

	// Wait long enough for old-token to expire.
	time.Sleep(timeout * 3)

	// Create a new session that should still be active.
	sNew, err := m.Get("new-token")
	require.NoError(t, err)

	// old-token should have expired and return a fresh session.
	sOldAgain, err := m.Get("old-token")
	require.NoError(t, err)
	assert.NotSame(t, sOld, sOldAgain, "expired session should be replaced")

	// new-token should still return the same session (not expired).
	sNewAgain, err := m.Get("new-token")
	require.NoError(t, err)
	assert.Same(t, sNew, sNewAgain, "active session should be retained")
}
