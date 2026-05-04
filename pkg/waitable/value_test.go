// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package waitable

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetSet(t *testing.T) {
	c := NewValue(0)
	v, _ := c.GetChan()
	assert.Equal(t, 0, v)
	c.Set(42)
	v, _ = c.GetChan()
	assert.Equal(t, 42, v)
}

func TestUpdateChannelClosesOnSet(t *testing.T) {
	c := NewValue(0)
	_, update := c.GetChan()
	c.Set(1)
	select {
	case <-update:
	case <-time.After(time.Second):
		t.Fatal("update channel not closed after Set")
	}
}

func TestUpdateChannelBlocksBeforeSet(t *testing.T) {
	c := NewValue(0)
	_, update := c.GetChan()
	select {
	case <-update:
		t.Fatal("update channel closed before Set")
	default:
	}
}

func TestNewUpdateChannelAfterSet(t *testing.T) {
	c := NewValue(0)
	_, first := c.GetChan()
	c.Set(1)
	_, second := c.GetChan()
	// First channel should be closed, second should be open.
	select {
	case <-first:
	default:
		t.Fatal("first update channel should be closed")
	}
	select {
	case <-second:
		t.Fatal("second update channel should not be closed")
	default:
	}
}

func TestMultipleWaiters(t *testing.T) {
	c := NewValue("a")
	var wg sync.WaitGroup
	results := make([]string, 3)
	for i := range results {
		_, update := c.GetChan()
		wg.Add(1)
		go func(i int, update <-chan struct{}) {
			defer wg.Done()
			<-update
			results[i], _ = c.GetChan()
		}(i, update)
	}
	time.Sleep(10 * time.Millisecond)
	c.Set("b")
	wg.Wait()
	for _, r := range results {
		assert.Equal(t, "b", r)
	}
}

func TestWaitWithContext(t *testing.T) {
	c := NewValue(0)
	_, update := c.GetChan()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		select {
		case <-update:
			done <- nil
		case <-ctx.Done():
			done <- ctx.Err()
		}
	}()
	time.Sleep(10 * time.Millisecond)
	cancel()
	assert.ErrorIs(t, <-done, context.Canceled)
}

func TestWaitWithTimeout(t *testing.T) {
	c := NewValue(0)
	_, update := c.GetChan()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	select {
	case <-update:
		t.Fatal("should not have received update")
	case <-ctx.Done():
		assert.ErrorIs(t, ctx.Err(), context.DeadlineExceeded)
	}
}
