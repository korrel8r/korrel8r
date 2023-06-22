// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package impl

import (
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
)

var domain = mock.Domain("x")

func TestConvert(t *testing.T) {
	d, err := TypeAssert[korrel8r.Domain](domain)
	assert.NoError(t, err)
	assert.Equal(t, domain, d)

	_, err = TypeAssert[korrel8r.Query](nil)
	assert.EqualError(t, err, "wrong type: want korrel8r.Query, got (<nil>)(<nil>)")

	_, err = TypeAssert[korrel8r.Query](d)
	assert.EqualError(t, err, "wrong type: want korrel8r.Query, got (mock.Domain)(\"x\")")
}

type dummyQuery struct {
	Foo string
	Bar int
}

func (*dummyQuery) Class() korrel8r.Class { return domain.Class("x") }
func (d *dummyQuery) String() string      { return korrel8r.JSONString(d) }

func TestQuery(t *testing.T) {
	var q dummyQuery
	// JSON style
	got, err := Query(`{"Foo":"hello","Bar":3}`, &q)
	assert.NoError(t, err)
	assert.Equal(t, &dummyQuery{Foo: "hello", Bar: 3}, got)

	// YAML style
	got, err = Query("{Foo:hello,Bar:3}", &q)
	assert.NoError(t, err)
	assert.Equal(t, &dummyQuery{Foo: "hello", Bar: 3}, got)
}
