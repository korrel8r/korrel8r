// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package enumflag

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	v := New("b", []string{"c", "a", "b"})
	assert.Equal(t, "b", v.Value)
	assert.Equal(t, []string{"a", "b", "c"}, v.Allowed) // sorted
}

func TestSet_Valid(t *testing.T) {
	v := New("a", []string{"a", "b", "c"})
	require.NoError(t, v.Set("b"))
	assert.Equal(t, "b", v.Value)
}

func TestSet_Invalid(t *testing.T) {
	v := New("a", []string{"a", "b", "c"})
	assert.Error(t, v.Set("x"))
	assert.Equal(t, "a", v.Value) // unchanged
}

func TestString(t *testing.T) {
	v := New("hello", []string{"hello", "world"})
	assert.Equal(t, "hello", v.String())
}

func TestType(t *testing.T) {
	v := New("a", []string{"a"})
	assert.Equal(t, "string", v.Type())
}

func TestDocString(t *testing.T) {
	v := New("a", []string{"a", "b"})
	assert.Equal(t, "One of [a b]", v.DocString(""))
	assert.Equal(t, "Pick one: One of [a b]", v.DocString("Pick one"))
}
