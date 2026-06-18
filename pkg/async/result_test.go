// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package async

import (
	"sync"
	"testing"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
)

type noIDClass struct{}

func (noIDClass) Domain() korrel8r.Domain       { return nil }
func (noIDClass) Name() string                  { return "noid" }
func (noIDClass) String() string                { return "test:noid" }
func (noIDClass) Unmarshal([]byte) (any, error) { return nil, nil }

type idClass struct{ noIDClass }

func (idClass) ID(o korrel8r.Object) any { return o }
func (idClass) Name() string             { return "withid" }

func TestResult_Add(t *testing.T) {
	r := New(noIDClass{})
	assert.True(t, r.Add("a"))
	assert.True(t, r.Add("b"))
	assert.Equal(t, []korrel8r.Object{"a", "b"}, r.List())
}

func TestResult_Append(t *testing.T) {
	r := New(noIDClass{})
	r.Append("x", "y", "z")
	assert.Equal(t, []korrel8r.Object{"x", "y", "z"}, r.List())
}

func TestResult_Dedup(t *testing.T) {
	r := New(idClass{})
	assert.True(t, r.Add("a"))
	assert.True(t, r.Add("b"))
	assert.False(t, r.Add("a"))
	assert.Equal(t, []korrel8r.Object{"a", "b"}, r.List())
}

func TestResult_Wait(t *testing.T) {
	r := New(noIDClass{})
	var got []korrel8r.Object
	var wg sync.WaitGroup
	wg.Go(func() {
		got = r.Wait(0)
	})
	r.Add("a")
	r.Add("b")
	wg.Wait()
	assert.NotEmpty(t, got)
	assert.Contains(t, got, "a")
}

func TestResult_Wait_Incremental(t *testing.T) {
	r := New(noIDClass{})
	r.Add("a")
	r.Add("b")

	got := r.Wait(0)
	assert.Equal(t, []korrel8r.Object{"a", "b"}, got)

	var wg sync.WaitGroup
	var got2 []korrel8r.Object
	wg.Go(func() {
		got2 = r.Wait(2)
	})
	r.Add("c")
	wg.Wait()
	assert.Equal(t, []korrel8r.Object{"c"}, got2)
}

func TestResult_Wait_Close(t *testing.T) {
	r := New(noIDClass{})
	var got []korrel8r.Object
	var wg sync.WaitGroup
	wg.Go(func() {
		got = r.Wait(0)
	})
	r.Close()
	wg.Wait()
	assert.Nil(t, got)
}

func TestResult_ConcurrentAdd(t *testing.T) {
	r := New(noIDClass{})
	var wg sync.WaitGroup
	for i := range 100 {
		wg.Go(func() {
			r.Add(i)
		})
	}
	wg.Wait()
	assert.Len(t, r.List(), 100)
}
