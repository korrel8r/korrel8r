// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package mock

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQuery_String(t *testing.T) {
	q := NewQuery(Domain("foo").Class("x"), "a", "b", "c")
	s := `{"class":"x", "results": ["a","b","c"]}`
	assert.JSONEq(t, s, q.String())
	q2, err := Domain("foo").Query(s)
	assert.NoError(t, err)
	assert.Equal(t, q, q2)
}
