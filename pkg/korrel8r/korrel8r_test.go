// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package korrel8r

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppenderFunc(t *testing.T) {
	var got []Object
	f := AppenderFunc(func(o ...Object) { got = append(got, o...) })
	f.Append("a", "b", "c")
	assert.Equal(t, []Object{"a", "b", "c"}, got)
}

func TestAppenderFunc_Empty(t *testing.T) {
	called := false
	f := AppenderFunc(func(o ...Object) { called = len(o) > 0 })
	f.Append()
	assert.False(t, called)
}
