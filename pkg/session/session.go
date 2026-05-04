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
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/pkg/api"
	"github.com/korrel8r/korrel8r/pkg/auth"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/waitable"
)

var log = logging.Log()

// Console state, passes values between REST and MCP operations.
type Console struct {
}

// Session holds per-user state including engine, console state, and configuration.
type Session struct {
	ID             string // Session ID - a username or hashed authorization token.
	Engine         *engine.Engine
	ConsoleState   *waitable.Value[*api.Console] // Actual console state
	ConsoleRequest *waitable.Value[*api.Console] // Requested by agent.

	lastUsed atomic.Int64 // UnixNano timestamp for expiration, atomic for lock-free access.
}

func (s *Session) String() string { return s.ID }

// FromContext returns the session from ctx. See [WithSession].
func FromContext(ctx context.Context) *Session {
	s, _ := ctx.Value(sessionKey{}).(*Session)
	return s
}

// WithSession returns a context with a session. See [FromContext].
func WithSession(ctx context.Context, s *Session) context.Context {
	return context.WithValue(ctx, sessionKey{}, s)
}

// Manager returns sessions by key.
type Manager interface {
	// Get the session for a context.
	Get(ctx context.Context) (*Session, error)
}

// singleManager always returns the same session, ignoring the context.
type singleManager struct {
	session *Session
}

func newSession(e *engine.Engine, id string) *Session {
	return &Session{
		ID:             id,
		Engine:         e,
		ConsoleState:   waitable.NewValue[*api.Console](nil),
		ConsoleRequest: waitable.NewValue[*api.Console](nil),
	}
}

// NewSingle returns a Manager that always returns the same session.
func NewSingle(e *engine.Engine) Manager {
	return &singleManager{session: newSession(e, "")}
}

func (m *singleManager) Get(ctx context.Context) (*Session, error) {
	return m.session, nil
}
func (m *singleManager) Close() {}

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
	factory     func() (*engine.Engine, error)
	timeout     time.Duration
	lastCleanup atomic.Int64
}

// NewPool creates a Manager that creates a new Session per key using factory
// and expires sessions after timeout of inactivity.
// timeout == 0 means never time out.
// Automatically attempts to use k8s TokenReview for bearer token resolution.
func NewPool(timeout time.Duration, factory func() (*engine.Engine, error)) Manager {
	tokenReview, err := auth.NewTokenReview()
	if err != nil {
		log.V(1).Info("TokenReview not available, using hashed token as session ID", "error", err)
	}
	return NewPoolWithTokenReview(timeout, factory, tokenReview)
}

// NewPoolWithTokenReview is like NewPool but uses the provided TokenReview
// instead of creating one automatically. Pass nil to disable token resolution.
func NewPoolWithTokenReview(timeout time.Duration, factory func() (*engine.Engine, error), tr *auth.TokenReview) Manager {
	return &poolManager{
		timeout:     timeout,
		tokenReview: tr,
		factory:     factory,
	}
}

// id returns session ID for context
func (m *poolManager) id(ctx context.Context) string {
	token := auth.ContextToken(ctx)
	if token == "" {
		log.V(4).Info("No bearer token found, using generic session")
		return ""
	}
	if m.tokenReview != nil {
		userName, err := m.tokenReview.User(token)
		if err == nil {
			return userName
		}
		log.V(4).Info("Token review error", "error", err, "FIXME", token)
	}
	log.V(4).Info("Cannot determine username, session ID is hashed token")
	return hashToken(token)
}

// Get returns the Session for the given context, creating a new session if needed.
func (m *poolManager) Get(ctx context.Context) (*Session, error) {
	id := m.id(ctx)
	v, _ := m.sessions.LoadOrStore(id, &entry{})
	e := v.(*entry)
	e.once.Do(func() {
		var eng *engine.Engine
		eng, e.err = m.factory()
		if e.err == nil {
			e.session = newSession(eng, id)
			log.V(1).Info("Session created", "session", id)
		}
	})
	if e.err != nil {
		log.Error(e.err, "Session create failed")
		m.sessions.CompareAndSwap(id, e, &entry{}) // Allow retry with a fresh entry.
		return nil, e.err
	}
	now := time.Now().UnixNano()
	e.session.lastUsed.Store(now)
	m.maybeCleanup(now)
	return e.session, nil
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
			log.V(2).Info("Session expired", "session", ent.session.ID)
			m.sessions.Delete(key)
		}
		return true
	})
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

type sessionKey struct{}

// UpdateRequest adds session and timeout to request context.
// Returns the request and cancel function for the timeout
func UpdateRequest(req *http.Request, sessions Manager) (*http.Request, func(), error) {
	ctx := req.Context()
	ctx = auth.WithToken(ctx, auth.HeaderToken(req.Header))
	ss, err := sessions.Get(ctx)
	if err != nil {
		return nil, func() {}, err
	}
	ctx = WithSession(ctx, ss)
	ctx, cancel := ss.Engine.WithTimeout(ctx, 0)
	req = req.WithContext(ctx)
	return req, cancel, nil
}

// Middleware to enable auth, session and timeout.
func Middleware(sessions Manager) func(*gin.Context) {
	return func(c *gin.Context) {
		c.Request = auth.UpdateRequest(c.Request)
		req, cancel, err := UpdateRequest(c.Request, sessions)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, c.Error(err).JSON())
			return
		}
		defer cancel()
		c.Request = req
		c.Next()
	}
}
