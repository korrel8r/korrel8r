// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package session

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/korrel8r/korrel8r/pkg/api"
	"github.com/korrel8r/korrel8r/internal/pkg/logging"
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
	update       chan *api.Console           // Latest-value channel for console updates (MCP→REST).
	connected    atomic.Bool                 // True while an SSE listener is connected.
	mcpConnected atomic.Bool                 // True once any MCP client has connected; survives listener reconnects.
}

func newConsoleEvents(session string) *consoleEvents {
	return &consoleEvents{
		session: session,
		update:  make(chan *api.Console, 1),
	}
}

// SetListener registers the SSE listener for this session.
// Returns ErrConsoleBusy if a listener is already connected.
// The caller must call ClearListener when done.
func (c *consoleEvents) SetListener() error {
	if !c.connected.CompareAndSwap(false, true) {
		return ErrConsoleBusy
	}
	return nil
}

// ClearListener removes the active SSE listener and drains pending updates.
func (c *consoleEvents) ClearListener() {
	c.connected.Store(false)
	c.fromConsole.Store(nil)
	select {
	case <-c.update:
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
	c.EnqueueConsoleUpdate(update)
	return nil
}

// EnqueueConsoleUpdate enqueues an update without requiring a connected listener.
// Use this for signals (like MCP connection) that may arrive before the SSE listener starts.
func (c *consoleEvents) EnqueueConsoleUpdate(update *api.Console) {
	c.mcpConnected.Store(true)
	select {
	case <-c.update:
	default:
	}
	c.update <- update
	log.V(3).Info("Console: enqueue", "session", c.session, "update", logging.JSON(update))
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
	ticker *time.Ticker) error {

	consoleSend := func(update *api.Console) error {
		log.V(3).Info("Console: send", "session", c.session, "update", logging.JSON(update))
		return send(update)
	}

	if c.mcpConnected.Load() {
		if err := consoleSend(&api.Console{}); err != nil {
			return err
		}
	}
	for {
		select {
		case update := <-c.update:
			if err := consoleSend(update); err != nil {
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
