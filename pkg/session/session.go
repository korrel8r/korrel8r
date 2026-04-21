// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package session manages per-user sessions, each with its own Engine.
// Each session has a numeric ID for logging and a string Key for map lookup.
// Sessions expire after a configurable timeout of inactivity.
//
// Session key resolution follows this priority:
//  1. Bearer token resolved via TokenReview to username
//  2. Hashed Authorization header
package session

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"sync"
	"sync/atomic"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/pkg/auth"
	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/engine"
)

var log = logging.Log()

// Manager returns sessions by key.
type Manager interface {
	// Key returns the session key for a context.
	Key(ctx context.Context) string
	// Get returns the Session for a key, creating a new session if needed.
	Get(key string) (*Session, error)
	// Close releases resources associated with the manager.
	Close()
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// Session holds per-user state including engine, console state, and configuration.
type Session struct {
	Engine  *engine.Engine
	Configs config.Configs
	Console *Console // nil if no console is connected.
	ID      int      // Numeric session ID for logging.
	Key     string   // Session key for map lookup (username or hashed auth header).

	lastUsed atomic.Int64 // UnixNano timestamp, atomic for lock-free access.
}

// singleManager always returns the same session, ignoring the context.
type singleManager struct {
	session *Session
}

// NewSingle returns a Manager that always returns the same session.
func NewSingle(e *engine.Engine, configs config.Configs) Manager {
	return &singleManager{&Session{Engine: e, Configs: configs, Console: NewConsoleState()}}
}

func (m *singleManager) Key(context.Context) string   { return "" }
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
	tokenReview *auth.TokenReview
	factory     func() (*engine.Engine, config.Configs, error)
	timeout     time.Duration
	lastCleanup atomic.Int64
	nextID      atomic.Int64
}

// NewPool creates a Manager that creates a new Session per key using factory
// and expires sessions after timeout of inactivity.
// timeout == 0 means never time out.
// Automatically attempts to use k8s TokenReview for bearer token resolution.
func NewPool(timeout time.Duration, factory func() (*engine.Engine, config.Configs, error)) Manager {
	tokenReview, err := auth.NewTokenReview()
	if err != nil {
		log.V(1).Info("TokenReview not available, using hashed token as session ID", "error", err)
	}
	return NewPoolWithTokenReview(timeout, factory, tokenReview)
}

// NewPoolWithTokenReview is like NewPool but uses the provided TokenReview
// instead of creating one automatically. Pass nil to disable token resolution.
func NewPoolWithTokenReview(timeout time.Duration, factory func() (*engine.Engine, config.Configs, error), tr *auth.TokenReview) Manager {
	return &poolManager{
		timeout:     timeout,
		tokenReview: tr,
		factory:     factory,
	}
}

// Key determines the session key from request context.
func (m *poolManager) Key(ctx context.Context) string {
	if token, _ := auth.Token(ctx); token != "" {
		if m.tokenReview != nil {
			if userName, err := m.tokenReview.User(token); err == nil {
				return userName
			}
		}
		return hashToken(token)
	}
	return ""
}

// Get returns the Session for the given context, creating a new session if needed.
func (m *poolManager) Get(key string) (*Session, error) {
	v, _ := m.sessions.LoadOrStore(key, &entry{})
	e := v.(*entry)
	e.once.Do(func() {
		var eng *engine.Engine
		var configs config.Configs
		eng, configs, e.err = m.factory()
		if e.err == nil {
			id := int(m.nextID.Add(1))
			e.session = &Session{Engine: eng, Configs: configs, Console: NewConsoleState(), ID: id, Key: key}
			log.V(1).Info("Session created", "id", id, "key", key)
		}
	})
	if e.err != nil {
		log.Error(e.err, "Session create failed")
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
			log.V(2).Info("Session expired", "id", ent.session.ID)
			m.sessions.Delete(key)
			ent.session.Console.Close()
		}
		return true
	})
}
