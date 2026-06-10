// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package result

import (
	"testing"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
)

type noIDClass struct{}

func (noIDClass) Domain() korrel8r.Domain          { return nil }
func (noIDClass) Name() string                     { return "noid" }
func (noIDClass) String() string                   { return "test:noid" }
func (noIDClass) Unmarshal([]byte) (any, error)    { return nil, nil }

type idClass struct{ noIDClass }

func (idClass) ID(o korrel8r.Object) any { return o }
func (idClass) Name() string             { return "withid" }

func TestNew_WithoutIDer(t *testing.T) {
	r := New(noIDClass{})
	assert.IsType(t, &List{}, r)
}

func TestNew_WithIDer(t *testing.T) {
	r := New(idClass{})
	assert.IsType(t, &Set{}, r)
}

func TestList(t *testing.T) {
	r := NewList()
	assert.Empty(t, r.List())

	r.Append("a", "b")
	r.Append("c")
	assert.Equal(t, []korrel8r.Object{"a", "b", "c"}, r.List())

	// List does not deduplicate
	r.Append("a")
	assert.Len(t, r.List(), 4)
}

func TestList_Add(t *testing.T) {
	r := NewList()
	assert.True(t, r.Add("x"))
	assert.Equal(t, []korrel8r.Object{"x"}, r.List())
}

func TestSet_Dedup(t *testing.T) {
	r := NewSet(idClass{})
	assert.True(t, r.Add("a"))
	assert.True(t, r.Add("b"))
	assert.False(t, r.Add("a")) // duplicate
	assert.Equal(t, []korrel8r.Object{"a", "b"}, r.List())
}

func TestSet_Append(t *testing.T) {
	r := NewSet(idClass{})
	r.Append("x", "y", "x")
	assert.Equal(t, []korrel8r.Object{"x", "y"}, r.List())
}
