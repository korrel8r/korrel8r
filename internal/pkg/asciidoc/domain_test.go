// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package asciidoc

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDomain(t *testing.T) {
	d, err := Load("github.com/korrel8r/korrel8r/internal/pkg/asciidoc")
	require.NoError(t, err)
	got := d[0].Asciidoc(d[0].Package.Doc)
	want := `
package asciidoc renders go/doc comments as asciidoc

NOTE: the rest of this package comment is link:http://fakery.example[fake] comment to _self-test_ the doc generator.

This documentation is a fake domain, used to self-test the domain doc generator.

== Class

High class fakery.

== Subclass

This ^ is an implicit heading.

== Query

Queries are JSON-serialized instances of type link:https://pkg.go.dev/github.com/korrel8r/korrel8r/internal/pkg/asciidoc#Query[Query] and blah foobar

== Object

link:https://pkg.go.dev/github.com/korrel8r/korrel8r/internal/pkg/asciidoc#Object[Object] is a link:https://pkg.go.dev/regexp#Regexp[regexp.Regexp]

Here is

----
some
pre-formatted
code
----
`
	require.Equal(t, want, got, got)
}
