// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package session manages per-user sessions, each with its own Engine.
// Sessions are identified by a string key (typically the Authorization header value)
// and expire after a configurable timeout of inactivity.
package session

import (
	"context"
	"sync"
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
	// OnExpire registers a callback that is called when a session expires.
	OnExpire(f func(key string))
	// Close releases resources associated with the manager.
	Close()
}

// FromContext get the session key from the auth token in a context.
func FromContext(ctx context.Context, mgr Manager) (*Session, error) {
	token, _ := auth.Token(ctx)
	return mgr.Get(token)
}

// Session holds per-user state including engine, console, and configuration.
type Session struct {
	Engine  *engine.Engine
	Configs config.Configs
	Console *ConsoleState // nil if no console is connected.

	lastUsed time.Time
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
func (m *singleManager) OnExpire(func(string))        {}
func (m *singleManager) Close()                       {}

// poolManager maps session keys to Sessions, with timeout-based cleanup.
type poolManager struct {
	mu       sync.Mutex
	sessions map[string]*Session
	timeout  time.Duration
	factory  func() (*engine.Engine, error)
	done     chan struct{}
	onExpire []func(key string) // called when a session expires
}

// NewPool creates a Manager that creates a new Session per key using factory
// and expires sessions after timeout of inactivity.
func NewPool(timeout time.Duration, factory func() (*engine.Engine, error)) Manager {
	m := &poolManager{
		sessions: make(map[string]*Session),
		timeout:  timeout,
		factory:  factory,
		done:     make(chan struct{}),
	}
	tick := min(timeout/2, time.Minute)
	go m.cleanupLoop(tick)
	return m
}

// Get returns the Session for the given session key, creating a new session if needed.
func (m *poolManager) Get(key string) (*Session, error) {
	// Already in cache?
	if s := m.get(key); s != nil {
		return s, nil
	}
	// Create engine outside of the lock.
	e, err := m.factory()
	if err != nil {
		log.Error(err, "Session create failed", "key", key)
		return nil, err
	}
	// Lock to write a new session.
	m.mu.Lock()
	defer m.mu.Unlock()
	// Double check, session created by another goroutine.
	if existing := m.getLH(key); existing != nil {
		// Forget the engine we created, garbage collector will clean it up.
		return existing, nil
	}
	s := &Session{Engine: e, Console: NewConsoleState(), lastUsed: time.Now()}
	m.sessions[key] = s
	log.V(1).Info("Session created", "key", key)
	return s, nil
}

// OnExpire registers a callback that is called when a session expires.
// The callback receives the session key. It is called without the Manager lock held.
func (m *poolManager) OnExpire(f func(key string)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onExpire = append(m.onExpire, f)
}

// get returns session for key or nil. Takes lock.
func (m *poolManager) get(key string) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.getLH(key)
}

// getLH returns session for key or nil. Must be called with the lock held.
func (m *poolManager) getLH(key string) *Session {
	if s := m.sessions[key]; s != nil {
		s.lastUsed = time.Now()
		return s
	}
	return nil
}

// Close stops the cleanup goroutine.
func (m *poolManager) Close() {
	close(m.done)
}

func (m *poolManager) cleanupLoop(tick time.Duration) {
	ticker := time.NewTicker(tick)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			m.cleanup()
		case <-m.done:
			return
		}
	}
}

func (m *poolManager) cleanup() {
	m.mu.Lock()
	var expired []string
	now := time.Now()
	for key, s := range m.sessions {
		if now.Sub(s.lastUsed) > m.timeout {
			delete(m.sessions, key)
			expired = append(expired, key)
		}
	}
	callbacks := m.onExpire
	m.mu.Unlock()
	for _, key := range expired {
		for _, f := range callbacks {
			f(key)
		}
		log.V(1).Info("Session expired", "key", key)
	}
}
