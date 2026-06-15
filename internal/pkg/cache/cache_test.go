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
