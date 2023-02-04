package impl

import (
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
)

func TestConvert(t *testing.T) {
	d, err := TypeAssert[korrel8r.Domain](mock.Domain("x"))
	assert.NoError(t, err)
	assert.Equal(t, mock.Domain("x"), d)

	_, err = TypeAssert[korrel8r.Query](nil)
	assert.EqualError(t, err, "wrong type: want korrel8r.Query, got (<nil>)(<nil>)")

	_, err = TypeAssert[korrel8r.Query](d)
	assert.EqualError(t, err, "wrong type: want korrel8r.Query, got (mock.Domain)(\"x\")")
}

type dummyQuery struct {
	Foo string
	Bar int
}

func (*dummyQuery) Class() korrel8r.Class { return mock.Class("x") }

func TestUnmarshalQuery(t *testing.T) {
	var q dummyQuery
	// JSON style
	got, err := UnmarshalQuery([]byte(`{"Foo":"hello","Bar":3}`), &q)
	assert.NoError(t, err)
	assert.Equal(t, &dummyQuery{Foo: "hello", Bar: 3}, got)

	// YAML style
	got, err = UnmarshalQuery([]byte("{Foo:hello,Bar:3}"), &q)
	assert.NoError(t, err)
	assert.Equal(t, &dummyQuery{Foo: "hello", Bar: 3}, got)
}
