// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package ptr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTo(t *testing.T) {
	p := To(42)
	assert.Equal(t, 42, *p)

	s := To("hello")
	assert.Equal(t, "hello", *s)
}

func TestToSlice(t *testing.T) {
	assert.Nil(t, ToSlice([]int(nil)))
	assert.Nil(t, ToSlice([]int{}))

	p := ToSlice([]int{1, 2, 3})
	assert.Equal(t, []int{1, 2, 3}, *p)
}

func TestToBool(t *testing.T) {
	assert.Nil(t, ToBool(false))

	p := ToBool(true)
	assert.NotNil(t, p)
	assert.True(t, *p)
}

func TestDeref(t *testing.T) {
	assert.Equal(t, 0, Deref[int](nil))
	assert.Equal(t, "", Deref[string](nil))

	v := 42
	assert.Equal(t, 42, Deref(&v))
}
