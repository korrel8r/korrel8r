// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package session

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/korrel8r/korrel8r/pkg/api"
)

var ErrNoConsole = errors.New("no console is connected")

// consoleEvents implements the MCP-to-REST and REST-to-MCP pathways through a session.
type consoleEvents struct {
	fromConsole atomic.Pointer[api.Console] // State from console, REST to MCP.
	fromAgent   chan *api.Console           // Unbuffered: blocks MCP sender till console receives.

	mu          sync.Mutex
	cancel      context.CancelFunc // Cancel the current console listener.
	listenerCtx context.Context    // Non-nil while a console listener is active.
}

func newConsoleEvents() *consoleEvents {
	return &consoleEvents{
		fromAgent: make(chan *api.Console),
		cancel:    func() {},
	}
}

// ShowInConsole sends an update to the console via the unbuffered fromAgent channel.
// Blocks until the console receives the update or ctx is canceled.
// The blocking is deliberate, it delays returning until we know the update was at least sent.
// This tells the caller it was likely sent and processed, although it's not guaranteed.
func (c *consoleEvents) ShowInConsole(ctx context.Context, update *api.Console) error {
	for {
		c.mu.Lock()
		listenerCtx := c.listenerCtx
		c.mu.Unlock()

		if listenerCtx == nil {
			return ErrNoConsole
		}

		select {
		case c.fromAgent <- update:
			return nil
		case <-listenerCtx.Done():
			// Listener may have been replaced — retry with fresh state.
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// ConsoleEvents loops sending updates and keepalives until ctx is canceled or a send fails.
// ctx should be the HTTP request context, which cancels on client disconnect.
// Avoid passing a context with a short timeout — this is a long-lived SSE subscription.
//
// Only one listener is active at a time. A new caller evicts the previous listener
// by canceling its context, so a stale connection cannot lock out new consoles.
func (c *consoleEvents) ConsoleEvents(
	ctx context.Context,
	send func(*api.Console) error,
	tick func() error,
	ticker *time.Ticker) error {

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.cancel()
		c.cancel = cancel
		c.listenerCtx = ctx
	}()
	defer func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.listenerCtx == ctx {
			c.fromConsole.Store(nil)
			c.listenerCtx = nil
		}
	}()
	for {
		select {
		case update := <-c.fromAgent:
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
