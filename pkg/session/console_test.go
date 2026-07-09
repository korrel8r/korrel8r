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
	require.NoError(t, c.Listen())
	defer c.Close()
	assert.ErrorIs(t, c.Listen(), ErrConsoleBusy)
}

func TestSetListener_Reuse(t *testing.T) {
	c := newConsoleEvents("test")
	require.NoError(t, c.Listen())
	c.Close()
	require.NoError(t, c.Listen())
	c.Close()
}

func TestShowInConsole_NoListener(t *testing.T) {
	c := newConsoleEvents("test")
	assert.ErrorIs(t, c.ShowInConsole(console("v1")), ErrNoConsole)
}

func TestShowInConsole_Send(t *testing.T) {
	c := newConsoleEvents("test")
	require.NoError(t, c.Listen())
	defer c.Close()
	require.NoError(t, c.ShowInConsole(console("v1")))
	select {
	case got := <-c.toConsole:
		assert.Equal(t, console("v1"), got)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for update")
	}
}

func TestShowInConsole_LatestValueWins(t *testing.T) {
	c := newConsoleEvents("test")
	require.NoError(t, c.Listen())
	defer c.Close()
	require.NoError(t, c.ShowInConsole(console("v1")))
	require.NoError(t, c.ShowInConsole(console("v2")))
	select {
	case got := <-c.toConsole:
		assert.Equal(t, console("v2"), got)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for update")
	}
}

func TestClearListener_DrainsPending(t *testing.T) {
	c := newConsoleEvents("test")
	require.NoError(t, c.Listen())
	require.NoError(t, c.ShowInConsole(console("v1")))
	c.Close()
	select {
	case <-c.toConsole:
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
	require.NoError(t, c.Listen())
	c.SetConsoleState(console("state1"))
	c.Close()
	assert.Nil(t, c.ConsoleState())
}

func TestConsoleEvents_SendsUpdates(t *testing.T) {
	c := newConsoleEvents("test")
	require.NoError(t, c.Listen())
	defer c.Close()

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

	// ShowInConsole sets mcpConnected, so ConsoleEvents sends an initial empty event.
	require.NoError(t, c.ShowInConsole(console("v1")))

	go func() {
		time.Sleep(10 * time.Millisecond)
		require.NoError(t, c.ShowInConsole(console("v2")))
	}()

	err := c.ConsoleEvents(ctx, send, func() error { return nil }, time.Hour)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, []*api.Console{{}, console("v1"), console("v2")}, received)
}

func TestConsoleEvents_SendBeforeListen(t *testing.T) {
	c := newConsoleEvents("test")
	c.Send(&api.Console{})

	require.NoError(t, c.Listen())
	defer c.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var received []*api.Console
	send := func(u *api.Console) error {
		received = append(received, u)
		cancel()
		return nil
	}

	err := c.ConsoleEvents(ctx, send, func() error { return nil }, time.Hour)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, []*api.Console{{}}, received)
}

func TestConsoleEvents_ReconnectSendsEmpty(t *testing.T) {
	c := newConsoleEvents("test")
	require.NoError(t, c.Listen())

	// MCP connects via ShowInConsole, consumed by the first loop.
	require.NoError(t, c.ShowInConsole(console("v1")))
	ctx1, cancel1 := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel1()
	}()
	_ = c.ConsoleEvents(ctx1, func(*api.Console) error { return nil }, func() error { return nil }, time.Hour)
	c.Close()

	// New listener reconnects — gets initial empty because mcpConnected persists.
	require.NoError(t, c.Listen())
	defer c.Close()

	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()

	var received []*api.Console
	err := c.ConsoleEvents(ctx2, func(u *api.Console) error {
		received = append(received, u)
		cancel2()
		return nil
	}, func() error { return nil }, time.Hour)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, []*api.Console{{}}, received)
}

func TestConsoleEvents_Tick(t *testing.T) {
	c := newConsoleEvents("test")
	require.NoError(t, c.Listen())
	defer c.Close()

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

	err := c.ConsoleEvents(ctx, func(*api.Console) error { return nil }, tick, 10*time.Millisecond)
	assert.ErrorIs(t, err, context.Canceled)
	assert.GreaterOrEqual(t, tickCount, 2)
}
