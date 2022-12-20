package templaterule

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestURLQueryMap(t *testing.T) {
	assert.Equal(t, "a=1&b=2", urlQueryMap(map[string]int{"a": 1, "b": 2}))
	assert.Equal(t, "", urlQueryMap(nil))
	assert.Equal(t, "", urlQueryMap(map[int]int{}))
}

func TestSelector(t *testing.T) {
	assert.Equal(t, "a=1,b=2", selector(map[string]int{"a": 1, "b": 2}))
	assert.Equal(t, "", selector(nil))
	assert.Equal(t, "", selector(map[int]int{}))
}
