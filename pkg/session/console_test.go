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

func TestSetListener_Busy(t *testing.T) {
	c := newConsoleEvents("test")
	require.NoError(t, c.SetListener())
	defer c.ClearListener()
	assert.ErrorIs(t, c.SetListener(), ErrConsoleBusy)
}

func TestSetListener_Reuse(t *testing.T) {
	c := newConsoleEvents("test")
	require.NoError(t, c.SetListener())
	c.ClearListener()
	require.NoError(t, c.SetListener())
	c.ClearListener()
}

func TestShowInConsole_NoListener(t *testing.T) {
	c := newConsoleEvents("test")
	assert.ErrorIs(t, c.ShowInConsole(console("v1")), ErrNoConsole)
}

func TestShowInConsole_Send(t *testing.T) {
	c := newConsoleEvents("test")
	require.NoError(t, c.SetListener())
	defer c.ClearListener()
	require.NoError(t, c.ShowInConsole(console("v1")))
	select {
	case got := <-c.update:
		assert.Equal(t, console("v1"), got)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for update")
	}
}

func TestShowInConsole_LatestValueWins(t *testing.T) {
	c := newConsoleEvents("test")
	require.NoError(t, c.SetListener())
	defer c.ClearListener()
	require.NoError(t, c.ShowInConsole(console("v1")))
	require.NoError(t, c.ShowInConsole(console("v2")))
	select {
	case got := <-c.update:
		assert.Equal(t, console("v2"), got)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for update")
	}
}

func TestClearListener_DrainsPending(t *testing.T) {
	c := newConsoleEvents("test")
	require.NoError(t, c.SetListener())
	require.NoError(t, c.ShowInConsole(console("v1")))
	c.ClearListener()
	select {
	case <-c.update:
		t.Fatal("expected channel to be drained")
	default:
	}
}

func TestConsoleState(t *testing.T) {
	c := newConsoleEvents("test")
	assert.Nil(t, c.ConsoleState())
	c.SetConsoleState(console("state1"))
	assert.Equal(t, console("state1"), c.ConsoleState())
}

func TestClearListener_ResetsConsoleState(t *testing.T) {
	c := newConsoleEvents("test")
	require.NoError(t, c.SetListener())
	c.SetConsoleState(console("state1"))
	c.ClearListener()
	assert.Nil(t, c.ConsoleState())
}

func TestConsoleEvents_SendsUpdates(t *testing.T) {
	c := newConsoleEvents("test")
	require.NoError(t, c.SetListener())
	defer c.ClearListener()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var received []*api.Console
	send := func(u *api.Console) error {
		received = append(received, u)
		if len(received) == 3 {
			cancel()
		}
		return nil
	}
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	// ShowInConsole sets mcpConnected, so pre-loop check sends empty first.
	require.NoError(t, c.ShowInConsole(console("v1")))

	go func() {
		time.Sleep(10 * time.Millisecond)
		require.NoError(t, c.ShowInConsole(console("v2")))
	}()

	err := c.ConsoleEvents(ctx, send, func() error { return nil }, ticker)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, []*api.Console{{}, console("v1"), console("v2")}, received)
}

func TestConsoleEvents_MCPConnectsFirst(t *testing.T) {
	c := newConsoleEvents("test")

	// MCP connects before the SSE listener — empty event waits in the channel.
	c.EnqueueConsoleUpdate(&api.Console{})

	require.NoError(t, c.SetListener())
	defer c.ClearListener()

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
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	err := c.ConsoleEvents(ctx, send, func() error { return nil }, ticker)
	assert.ErrorIs(t, err, context.Canceled)
	// Pre-loop empty (mcpConnected flag), then queued empty from EnqueueConsoleUpdate.
	assert.Equal(t, []*api.Console{{}, {}}, received)
}

func TestConsoleEvents_MCPConnectsLater(t *testing.T) {
	c := newConsoleEvents("test")
	require.NoError(t, c.SetListener())
	defer c.ClearListener()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var received []*api.Console
	send := func(u *api.Console) error {
		received = append(received, u)
		cancel()
		return nil
	}
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	go func() {
		time.Sleep(10 * time.Millisecond)
		require.NoError(t, c.ShowInConsole(console("v1")))
	}()

	err := c.ConsoleEvents(ctx, send, func() error { return nil }, ticker)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, []*api.Console{console("v1")}, received)
}

func TestConsoleEvents_MCPEmptyThenUpdate(t *testing.T) {
	c := newConsoleEvents("test")
	require.NoError(t, c.SetListener())
	defer c.ClearListener()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var received []*api.Console
	send := func(u *api.Console) error {
		received = append(received, u)
		if len(received) == 3 {
			cancel()
		}
		return nil
	}
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	// Simulate MCP initialized then tool call.
	require.NoError(t, c.ShowInConsole(&api.Console{}))

	go func() {
		time.Sleep(10 * time.Millisecond)
		require.NoError(t, c.ShowInConsole(console("v1")))
	}()

	err := c.ConsoleEvents(ctx, send, func() error { return nil }, ticker)
	assert.ErrorIs(t, err, context.Canceled)
	// Pre-loop empty (mcpConnected), then queued empty, then v1.
	assert.Equal(t, []*api.Console{{}, {}, console("v1")}, received)
}

func TestConsoleEvents_ReconnectGetsMCPSignal(t *testing.T) {
	c := newConsoleEvents("test")
	require.NoError(t, c.SetListener())

	// MCP connects, event is consumed by the loop.
	c.EnqueueConsoleUpdate(&api.Console{})
	ctx1, cancel1 := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel1()
	}()
	_ = c.ConsoleEvents(ctx1, func(*api.Console) error { return nil }, func() error { return nil }, time.NewTicker(time.Hour))
	c.ClearListener()

	// New listener reconnects — should get empty signal from mcpConnected flag.
	require.NoError(t, c.SetListener())
	defer c.ClearListener()

	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()

	var received []*api.Console
	send := func(u *api.Console) error {
		received = append(received, u)
		cancel2()
		return nil
	}
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	err := c.ConsoleEvents(ctx2, send, func() error { return nil }, ticker)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, []*api.Console{{}}, received)
}

func TestConsoleEvents_Tick(t *testing.T) {
	c := newConsoleEvents("test")
	require.NoError(t, c.SetListener())
	defer c.ClearListener()

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
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	err := c.ConsoleEvents(ctx, func(*api.Console) error { return nil }, tick, ticker)
	assert.ErrorIs(t, err, context.Canceled)
	assert.GreaterOrEqual(t, tickCount, 2)
}
