// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package session

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/korrel8r/korrel8r/pkg/api"
)

var (
	ErrNoConsole   = errors.New("no console is connected")
	ErrConsoleBusy = errors.New("another console is already connected to this session")
)

// consoleEvents implements the MCP-to-REST and REST-to-MCP pathways through a session.
// Only one console SSE listener can be active per session.
type consoleEvents struct {
	session      string
	fromConsole  atomic.Pointer[api.Console] // Latest state from console (REST→MCP).
	toConsole    chan *api.Console           // Latest-value channel for console updates (MCP→REST).
	connected    atomic.Bool                 // True while an SSE listener is connected.
	mcpConnected atomic.Bool                 // True once ShowInConsole has been called; survives listener reconnects.
}

func newConsoleEvents(session string) *consoleEvents {
	return &consoleEvents{
		session:   session,
		toConsole: make(chan *api.Console, 1),
	}
}

// Listen registers the SSE listener for this session.
// Returns ErrConsoleBusy if a listener is already connected.
// The caller must call Close when done.
func (c *consoleEvents) Listen() error {
	if !c.connected.CompareAndSwap(false, true) {
		return ErrConsoleBusy
	}
	return nil
}

// Close removes the active SSE listener and drains pending updates.
func (c *consoleEvents) Close() {
	c.connected.Store(false)
	c.fromConsole.Store(nil)
	select {
	case <-c.toConsole:
	default:
	}
}

// ShowInConsole enqueues an update for the console SSE listener.
// Returns ErrNoConsole if no listener is connected.
// The latest value wins: any stale pending update is replaced.
func (c *consoleEvents) ShowInConsole(update *api.Console) error {
	if !c.connected.Load() {
		return ErrNoConsole
	}
	c.mcpConnected.Store(true)
	c.Send(update)
	return nil
}

// Send enqueues an update without requiring a connected listener.
// Use this for signals (like MCP connection) that may arrive before the SSE listener starts.
func (c *consoleEvents) Send(update *api.Console) {
	// Drain old value
	select {
	case <-c.toConsole:
	default:
	}
	// Post new value
	select {
	case c.toConsole <- update:
	default:
	}
}

// ConsoleEvents loops sending updates and keepalives until ctx is canceled or a send fails.
// ctx should be the HTTP request context, which cancels on client disconnect.
// Avoid passing a context with a short timeout — this is a long-lived SSE subscription.
//
// The caller must call SetListener before this and ClearListener after.
func (c *consoleEvents) ConsoleEvents(
	ctx context.Context,
	send func(*api.Console) error,
	tick func() error,
	tickInterval time.Duration) error {

	// Send empty "connected" notification directly (not via channel, to preserve pending updates).
	if c.mcpConnected.Load() {
		if err := send(&api.Console{}); err != nil {
			return err
		}
	}
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()
	for {
		select {
		case update := <-c.toConsole:
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
