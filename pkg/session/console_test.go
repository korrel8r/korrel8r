// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package session

import (
	"context"
	"testing"
	"time"

	"github.com/korrel8r/korrel8r/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func console(view string) *api.Console { return &api.Console{View: api.Query(view)} }

func noop(*api.Console) error { return nil }

func TestShowInConsole_ErrNoConsole(t *testing.T) {
	c := newConsoleEvents()
	assert.ErrorIs(t, c.ShowInConsole(console("v1")), ErrNoConsole)
}

func TestConsoleEvents_ErrConsoleBusy(t *testing.T) {
	c := newConsoleEvents()

	// Two competing calls, one will return ErrConsoleBusy
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 2)
	go func() { errCh <- c.ConsoleEvents(ctx, nil, noop, func() error { return nil }, time.Hour) }()
	go func() { errCh <- c.ConsoleEvents(ctx, nil, noop, func() error { return nil }, time.Hour) }()
	assert.ErrorIs(t, <-errCh, ErrConsoleBusy)
	cancel()
	assert.ErrorIs(t, <-errCh, context.Canceled)

	// Next call should be OK, previous calls are over.
	ctx, cancel = context.WithCancel(context.Background())
	go func() { errCh <- c.ConsoleEvents(ctx, nil, noop, func() error { return nil }, time.Hour) }()
	select {
	case err := <-errCh:
		t.Fatalf("unexpected error: %v", err)
	default:
	}
	cancel()
	assert.ErrorIs(t, <-errCh, context.Canceled)
}

func TestConsoleEvents_SendsUpdates(t *testing.T) {
	c := newConsoleEvents()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var received []*api.Console
	send := func(u *api.Console) error {
		received = append(received, u)
		if len(received) == 2 {
			cancel()
		}
		return nil
	}

	errCh := make(chan error)
	go func() { errCh <- c.ConsoleEvents(ctx, nil, send, func() error { return nil }, time.Hour) }()
	time.Sleep(10 * time.Millisecond)

	require.NoError(t, c.ShowInConsole(console("v1")))
	time.Sleep(10 * time.Millisecond)
	require.NoError(t, c.ShowInConsole(console("v2")))

	assert.ErrorIs(t, <-errCh, context.Canceled)
	assert.Equal(t, []*api.Console{console("v1"), console("v2")}, received)
}

func TestConsoleEvents_Tick(t *testing.T) {
	c := newConsoleEvents()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tickCount := 0
	tick := func() error {
		tickCount++
		if tickCount >= 2 {
			cancel()
		}
		return nil
	}

	err := c.ConsoleEvents(ctx, nil, noop, tick, 10*time.Millisecond)
	assert.ErrorIs(t, err, context.Canceled)
	assert.GreaterOrEqual(t, tickCount, 2)
}

func TestConsoleState(t *testing.T) {
	c := newConsoleEvents()
	assert.Nil(t, c.ConsoleState())
	c.SetConsoleState(console("state1"))
	assert.Equal(t, console("state1"), c.ConsoleState())
}

func TestConsoleEvents_ResetsConsoleState(t *testing.T) {
	c := newConsoleEvents()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		c.SetConsoleState(console("state1"))
		cancel()
	}()

	_ = c.ConsoleEvents(ctx, nil, noop, func() error { return nil }, time.Hour)
	assert.Nil(t, c.ConsoleState())
}
