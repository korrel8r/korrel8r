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
	fromConsole atomic.Pointer[api.Console] // Latest state from console (REST→MCP).
	toConsole   chan *api.Console           // Latest-value channel for console updates (MCP→REST).
	connected   atomic.Bool                 // True while an SSE listener is connected.
}

func newConsoleEvents() *consoleEvents {
	return &consoleEvents{
		toConsole: make(chan *api.Console),
	}
}

// ShowInConsole sends an update to the console SSE listener.
// Returns ErrNoConsole if no listener is connected.
// Drops the update if the listener is not ready to receive.
func (c *consoleEvents) ShowInConsole(update *api.Console) error {
	select {
	case c.toConsole <- update:
		return nil
	default:
		return ErrNoConsole
	}
}

// ConsoleEvents loops sending updates and keepalives until ctx is canceled or a send fails.
// Returns ErrConsoleBusy if another listener is already connected.
// ready (if non-nil) is called after acquiring the connection slot but before entering the loop.
// ctx should be the HTTP request context, which cancels on client disconnect.
// Avoid passing a context with a short timeout — this is a long-lived SSE subscription.
func (c *consoleEvents) ConsoleEvents(
	ctx context.Context,
	ready func() error,
	send func(*api.Console) error,
	tick func() error,
	tickInterval time.Duration) error {

	// Only allow one connection at a time.
	if !c.connected.CompareAndSwap(false, true) {
		return ErrConsoleBusy
	}
	defer func() {
		c.connected.Store(false)
		c.fromConsole.Store(nil)
	}()

	if ready != nil {
		if err := ready(); err != nil {
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
