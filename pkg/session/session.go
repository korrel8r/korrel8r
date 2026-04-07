// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package session manages per-user sessions, each with its own Engine.
// Sessions are identified by a string key (typically the Authorization header value)
// and expire after a configurable timeout of inactivity.
package session

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"sync/atomic"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/rest/auth"
)

var log = logging.Log()

// Manager returns sessions by key.
type Manager interface {
	// Get returns the Session for the given key, creating a new session if needed.
	Get(key string) (*Session, error)
	// Close releases resources associated with the manager.
	Close()
}

// FromContext gets the session for the auth token in a context.
//
// The token is hashed before use as a session key to avoid storing raw credentials in memory.
// If no auth token is present, all unauthenticated requests share a single session keyed by "".
func FromContext(ctx context.Context, mgr Manager) (*Session, error) {
	token, _ := auth.Token(ctx)
	return mgr.Get(hashKey(token))
}

// hashKey returns a hex-encoded SHA-256 hash of the key, or "" for an empty key.
// Used to avoid storing security-sensitive tokens in memory or logs.
func hashKey(key string) string {
	if key == "" {
		return ""
	}
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

// Session holds per-user state including engine, console state, and configuration.
type Session struct {
	Engine  *engine.Engine
	Configs config.Configs
	Console *Console // nil if no console is connected.

	lastUsed atomic.Int64 // UnixNano timestamp, atomic for lock-free access.
}

// singleManager always returns the same session, ignoring the key.
type singleManager struct {
	session *Session
}

// NewSingle returns a Manager that always returns the same session.
func NewSingle(e *engine.Engine, configs config.Configs) Manager {
	return &singleManager{&Session{Engine: e, Configs: configs, Console: NewConsoleState()}}
}

func (m *singleManager) Get(string) (*Session, error) { return m.session, nil }
func (m *singleManager) Close()                       { m.session.Console.Close() }

// entry holds a session that is initialized exactly once.
// Concurrent callers block on once.Do until the session is ready.
type entry struct {
	once    sync.Once
	session *Session
	err     error
}

// poolManager maps session keys to Sessions, with timeout-based cleanup.
type poolManager struct {
	sessions    sync.Map // map[string]*entry
	factory     func() (*engine.Engine, config.Configs, error)
	timeout     time.Duration
	lastCleanup atomic.Int64
}

// NewPool creates a Manager that creates a new Session per key using factory
// and expires sessions after timeout of inactivity.
// timeout == 0 means never time out.
func NewPool(timeout time.Duration, factory func() (*engine.Engine, config.Configs, error)) Manager {
	return &poolManager{
		timeout: timeout,
		factory: factory,
	}
}

// Get returns the Session for the given session key, creating a new session if needed.
func (m *poolManager) Get(key string) (*Session, error) {
	v, _ := m.sessions.LoadOrStore(key, &entry{})
	e := v.(*entry)
	e.once.Do(func() {
		var eng *engine.Engine
		var configs config.Configs
		eng, configs, e.err = m.factory()
		if e.err == nil {
			e.session = &Session{Engine: eng, Configs: configs, Console: NewConsoleState()}
			log.V(1).Info("Session created", "key", key)
		}
	})
	if e.err != nil {
		log.Error(e.err, "Session create failed", "key", key)
		m.sessions.CompareAndSwap(key, e, &entry{}) // Allow retry with a fresh entry.
		return nil, e.err
	}
	now := time.Now().UnixNano()
	e.session.lastUsed.Store(now)
	m.maybeCleanup(now)
	return e.session, nil
}

// Close closes all sessions.
func (m *poolManager) Close() {
	m.sessions.Range(func(key, value any) bool {
		if ent := value.(*entry); ent.session != nil {
			ent.session.Console.Close()
		}
		m.sessions.Delete(key)
		return true
	})
}

// maybeCleanup runs cleanup if timeout is enabled and enough time has passed since the last cleanup.
//
// Note this allows sessions to hang around longer if there is no activity, but in that case the
// server is not under pressure and the number of sessions is not growing.
func (m *poolManager) maybeCleanup(now int64) {
	if m.timeout <= 0 {
		return
	}
	last := m.lastCleanup.Load()
	if now-last < int64(m.timeout) {
		return
	}
	if !m.lastCleanup.CompareAndSwap(last, now) {
		return // Another goroutine is already cleaning up.
	}
	m.sessions.Range(func(key, value any) bool {
		ent := value.(*entry)
		if ent.session != nil && now-ent.session.lastUsed.Load() > int64(m.timeout) {
			log.V(1).Info("Session expired", "key", key)
			m.sessions.Delete(key)
			ent.session.Console.Close()
		}
		return true
	})
}
