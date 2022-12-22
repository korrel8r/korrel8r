package templaterule

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUrlencode(t *testing.T) {
	assert.Equal(t, "a=1&b=2", urlencode(map[string]int{"a": 1, "b": 2}))
	assert.Equal(t, "", urlencode(nil))
	assert.Equal(t, "", urlencode(map[int]int{}))
}

func TestSelector(t *testing.T) {
	assert.Equal(t, "a=1,b=2", selector(map[string]int{"a": 1, "b": 2}))
	assert.Equal(t, "", selector(nil))
	assert.Equal(t, "", selector(map[int]int{}))
}
