// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package adoc

import (
	"go/doc"
	"go/doc/comment"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrinter(t *testing.T) {
	// Only interested in package docs.
	p := doc.Package{
		Doc: `
package dummy is a fake package to test comment parsing.
This line continues the paragraph.

This is a new paragraph and a list:
 * foo
 * bar

# Heading 1

  code sample
  more code

  yet more

Heading 2

Numbered list
 1. foo
 2. bar
`,
	}
	want := `package dummy is a fake package to test comment parsing. This line continues the paragraph.

This is a new paragraph and a list:

* foo
* bar

=== Heading 1

----
code sample
more code

yet more
----

=== Heading 2

Numbered list

1. foo
2. bar
`
	printer := Printer{Printer: p.Printer()}
	printer.HeadingLevel = 2
	var parser comment.Parser
	doc := parser.Parse(p.Doc)
	adoc := string(printer.Asciidoc(doc))
	assert.Equal(t, want, adoc, "%v", adoc)
}
