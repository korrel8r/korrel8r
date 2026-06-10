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
}

func TestParseQuery_WrongDomain(t *testing.T) {
	d := mock.NewDomain("test", "foo")
	_, _, err := ParseQuery(d, "other:foo:data")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "wrong domain")
}

func TestParseQuery_BadClass(t *testing.T) {
	d := mock.NewDomain("test", "foo")
	_, _, err := ParseQuery(d, "test:nosuch:data")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "class not found")
}

func TestParseQuery_Invalid(t *testing.T) {
	d := mock.NewDomain("test", "foo")
	_, _, err := ParseQuery(d, "invalid")
	assert.Error(t, err)
}

func TestUnmarshalQueryString(t *testing.T) {
	d := mock.NewDomain("test", "foo")

	type Data struct{ Name string }
	c, data, err := UnmarshalQueryString[Data](d, `test:foo:{"name":"hello"}`)
	require.NoError(t, err)
	assert.Equal(t, "foo", c.Name())
	assert.Equal(t, "hello", data.Name)
}

func TestUnmarshalQueryString_BadData(t *testing.T) {
	d := mock.NewDomain("test", "foo")

	type Data struct{ Name string }
	_, _, err := UnmarshalQueryString[Data](d, `test:foo:not valid json or yaml`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid query")
}
