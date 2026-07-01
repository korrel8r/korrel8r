// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package session

import (
	"context"
	"errors"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/korrel8r/korrel8r/pkg/api"
)

var ErrNoConsole = errors.New("no console is connected")

type consoleListener struct {
	ch chan *api.Console
}

// consoleEvents implements the MCP-to-REST and REST-to-MCP pathways through a session.
// Multiple console SSE listeners can be active simultaneously; ShowInConsole fans out to all.
type consoleEvents struct {
	fromConsole  atomic.Pointer[api.Console] // State from console, REST to MCP.
	mcpConnected bool                        // True after an MCP client has connected. Protected by mu.

	mu        sync.Mutex
	listeners []*consoleListener
}

func newConsoleEvents() *consoleEvents {
	return &consoleEvents{}
}

// addListener registers a new SSE listener.
// Returns true if an MCP client is already connected.
func (c *consoleEvents) addListener() (*consoleListener, bool) {
	l := &consoleListener{ch: make(chan *api.Console, 1)}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.listeners = append(c.listeners, l)
	return l, c.mcpConnected
}

func (c *consoleEvents) removeListener(l *consoleListener) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.listeners = slices.DeleteFunc(c.listeners, func(x *consoleListener) bool { return x == l })
	if len(c.listeners) == 0 {
		c.fromConsole.Store(nil)
	}
}

// ShowInConsole sends an update to all connected console SSE listeners.
// Any previously unsent value is replaced with the new one.
// Records that an MCP client is connected, so late-joining listeners
// receive an initial notification.
// Returns ErrNoConsole if no listeners are connected.
func (c *consoleEvents) ShowInConsole(update *api.Console) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.mcpConnected = true
	if len(c.listeners) == 0 {
		return ErrNoConsole
	}
	for _, l := range c.listeners {
		select {
		case <-l.ch:
		default:
		}
		l.ch <- update
	}
	return nil
}

// ConsoleEvents loops sending updates and keepalives until ctx is canceled or a send fails.
// ctx should be the HTTP request context, which cancels on client disconnect.
// Avoid passing a context with a short timeout — this is a long-lived SSE subscription.
//
// Multiple listeners can be active simultaneously; each receives all updates.
func (c *consoleEvents) ConsoleEvents(
	ctx context.Context,
	send func(*api.Console) error,
	tick func() error,
	ticker *time.Ticker) error {

	l, connected := c.addListener()
	defer c.removeListener(l)

	if connected {
		if err := send(&api.Console{}); err != nil {
			return err
		}
	}

	for {
		select {
		case update := <-l.ch:
			if err := send(update); err != nil {
				return err
			}
		case <-ticker.C:
			if err := tick(); err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (c *consoleEvents) SetConsoleState(state *api.Console) { c.fromConsole.Store(state) }
func (c *consoleEvents) ConsoleState() *api.Console         { return c.fromConsole.Load() }
