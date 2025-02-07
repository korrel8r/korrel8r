// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package impl

import (
	"testing"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
)

type testDomain struct{ *Domain }

// verify that example implements korrel8r.Domain
var _ korrel8r.Domain = testDomain{}

func (testDomain) Query(string) (korrel8r.Query, error) { panic("Not Implemented") }
func (testDomain) Store(any) (korrel8r.Store, error)    { panic("Not Implemented") }

func TestDomain(t *testing.T) {
	d := NewDomain("x", "mystery domain", testClass("a"), testClass("b"))
	assert.Equal(t, "x", d.Name())
	assert.Equal(t, "mystery domain", d.Description())
	assert.Equal(t, []korrel8r.Class{testClass("a"), testClass("b")}, d.Classes())
	assert.Equal(t, testClass("a"), d.Class("a"))
	assert.Nil(t, d.Class("x"))
}

// dummy testClass for test
type testClass string

var _ korrel8r.Class = testClass("")

var testDomainSingleton korrel8r.Domain

func (c testClass) Domain() korrel8r.Domain                     { return testDomainSingleton }
func (c testClass) String() string                              { return ClassString(c) }
func (c testClass) Name() string                                { return string(c) }
func (c testClass) ID(o korrel8r.Object) any                    { return o }
func (c testClass) Unmarshal(b []byte) (korrel8r.Object, error) { return nil, nil }
