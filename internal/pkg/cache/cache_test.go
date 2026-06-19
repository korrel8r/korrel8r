// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTTL_GetPut(t *testing.T) {
	c := NewTTL[string, int](time.Hour)
	c.Put("a", 1)
	v, ok := c.Get("a")
	assert.True(t, ok)
	assert.Equal(t, 1, v)
}

func TestTTL_Miss(t *testing.T) {
	c := NewTTL[string, int](time.Hour)
	_, ok := c.Get("missing")
	assert.False(t, ok)
}

func TestTTL_Expiry(t *testing.T) {
	c := NewTTL[string, int](time.Millisecond)
	c.Put("a", 1)
	time.Sleep(2 * time.Millisecond)
	_, ok := c.Get("a")
	assert.False(t, ok)
}

func TestTTL_Overwrite(t *testing.T) {
	c := NewTTL[string, int](time.Hour)
	c.Put("a", 1)
	c.Put("a", 2)
	v, ok := c.Get("a")
	assert.True(t, ok)
	assert.Equal(t, 2, v)
}

func TestTTL_Clear(t *testing.T) {
	c := NewTTL[string, int](time.Hour)
	c.Put("a", 1)
	c.Put("b", 2)
	c.Clear()
	_, ok := c.Get("a")
	assert.False(t, ok)
	_, ok = c.Get("b")
	assert.False(t, ok)
}

func TestTTL_MissReturnsZero(t *testing.T) {
	c := NewTTL[string, int](time.Hour)
	v, ok := c.Get("missing")
	assert.False(t, ok)
	assert.Equal(t, 0, v)
}

func TestTTL_ExpiryEvictsOnlyExpired(t *testing.T) {
	c := NewTTL[string, int](50 * time.Millisecond)
	c.Put("a", 1)
	time.Sleep(30 * time.Millisecond)
	c.Put("b", 2)
	time.Sleep(30 * time.Millisecond)
	// "a" should be expired, "b" should still be alive
	_, ok := c.Get("a")
	assert.False(t, ok)
	v, ok := c.Get("b")
	assert.True(t, ok)
	assert.Equal(t, 2, v)
}
