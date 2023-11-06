// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package adoc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDomainDoc(t *testing.T) {
	p, err := NewDomain("testdata")
	require.NoError(t, err)
	want := "Domain `fake` represents fake resoures in a fake world." + `

=== Class

High class fakery.

=== Query

Queries are JSON-serialized instances of type

----
type Query struct {
	Foo Foo	// fooable
}
----

and blah foobar`
	got := p.Asciidoc(2)
	assert.Equal(t, want, got, got)
}
