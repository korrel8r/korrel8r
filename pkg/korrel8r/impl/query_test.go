// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package impl

import (
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseQuery(t *testing.T) {
	d := mock.NewDomain("test", "foo", "bar")

	c, selector, err := ParseQuery(d, "test:foo:mydata")
	require.NoError(t, err)
	assert.Equal(t, "foo", c.Name())
	assert.Equal(t, "mydata", selector)

	for _, x := range []struct {
		name, query, wantErr string
	}{
		{"wrong domain", "other:foo:data", "wrong domain"},
		{"bad class", "test:nosuch:data", "class not found"},
		{"invalid format", "invalid", ""},
	} {
		t.Run(x.name, func(t *testing.T) {
			_, _, err := ParseQuery(d, x.query)
			require.Error(t, err)
			if x.wantErr != "" {
				assert.ErrorContains(t, err, x.wantErr)
			}
		})
	}
}

func TestUnmarshalQueryString(t *testing.T) {
	d := mock.NewDomain("test", "foo")
	type Data struct{ Name string }

	c, data, err := UnmarshalQueryString[Data](d, `test:foo:{"name":"hello"}`)
	require.NoError(t, err)
	assert.Equal(t, "foo", c.Name())
	assert.Equal(t, "hello", data.Name)

	_, _, err = UnmarshalQueryString[Data](d, `test:foo:not valid json or yaml`)
	assert.ErrorContains(t, err, "invalid query")
}
