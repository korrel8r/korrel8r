// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSampler struct {
	used, limit uint64
	err         error
}

func (s *fakeSampler) Sample() (uint64, uint64, error) { return s.used, s.limit, s.err }

func TestGuardPressureAndReset(t *testing.T) {
	s := &fakeSampler{used: 89, limit: 100}
	g, err := New(s, 90, 80, 0, time.Second)
	require.NoError(t, err)

	canceled := 0
	unregister := g.Register(func() { canceled++ })
	g.sample()
	assert.Equal(t, 0, canceled)

	s.used = 90
	g.sample()
	assert.Equal(t, 1, canceled)
	g.sample()
	assert.Equal(t, 1, canceled, "a pressure episode cancels each registered search once")

	registeredUnderPressure := 0
	g.Register(func() { registeredUnderPressure++ })
	assert.Equal(t, 1, registeredUnderPressure)

	s.used = 80
	g.sample()
	s.used = 95
	g.sample()
	assert.Equal(t, 1, canceled, "canceled searches are not canceled again after pressure resets")
	assert.Equal(t, 1, registeredUnderPressure)
	unregister()
}

func TestGuardCancelOnlyOnceUnderPressure(t *testing.T) {
	g, err := New(&fakeSampler{used: 90, limit: 100}, 90, 80, 0, time.Second)
	require.NoError(t, err)
	g.sample()

	canceled := 0
	unregister := g.Register(func() { canceled++ })
	defer unregister()
	// Simulate a sample that captured this search before Register canceled it.
	g.mu.Lock()
	for _, cancel := range g.active {
		cancel()
	}
	g.mu.Unlock()
	assert.Equal(t, 1, canceled)
}

func TestGuardSampleErrorPreservesState(t *testing.T) {
	s := &fakeSampler{used: 90, limit: 100}
	g, err := New(s, 90, 80, 0, time.Second)
	require.NoError(t, err)
	g.sample()

	s.err = errors.New("sample failed")
	g.sample()
	canceled := 0
	g.Register(func() { canceled++ })
	assert.Equal(t, 1, canceled)
}

func TestGuardValidation(t *testing.T) {
	_, err := New(&fakeSampler{}, 90, 90, 0, time.Second)
	assert.Error(t, err)
	_, err = New(&fakeSampler{}, 101, 80, 0, time.Second)
	assert.Error(t, err)
	_, err = New(&fakeSampler{}, 90, 80, 0, 0)
	assert.Error(t, err)
	g, err := New(nil, 0, 0, 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, g)
}

func TestGuardAbsoluteLimit(t *testing.T) {
	s := &fakeSampler{used: 499, limit: 1000}
	g, err := New(s, 80, 70, 500, time.Second)
	require.NoError(t, err)
	canceled := 0
	g.Register(func() { canceled++ })
	g.sample()
	assert.Zero(t, canceled)
	s.used = 500
	g.sample()
	assert.Equal(t, 1, canceled)
}

func TestGuardSamplesImmediately(t *testing.T) {
	g, err := New(&fakeSampler{used: 90, limit: 100}, 90, 80, 0, time.Hour)
	require.NoError(t, err)
	canceled := make(chan struct{})
	g.Register(func() { close(canceled) })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go g.Run(ctx)
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("guard did not perform its initial sample")
	}
}

func TestNewFromTuningInterval(t *testing.T) {
	g, err := NewFromTuning(nil)
	require.NoError(t, err)
	assert.Equal(t, 200*time.Millisecond, g.interval)
}

func TestNewFromTuningDisabled(t *testing.T) {
	g, err := NewFromTuning(&config.Tuning{MemoryLimit: -1})
	require.NoError(t, err)
	assert.Nil(t, g.sampler)
}
