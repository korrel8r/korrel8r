// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package asciidoc

import (
	"bytes"
	"fmt"
	"go/doc/comment"
	"regexp"
	"strings"
)

// Printer prints godoc comments in asciidoc format.
//
// Based on [https://cs.opensource.google/go/go/+/refs/tags/go1.23.2:src/go/doc/comment/markdown.go]
type Printer struct {
	*comment.Printer
	raw bytes.Buffer // Temporary buffer
}

func NewPrinter(p *comment.Printer) *Printer {
	return &Printer{Printer: p}
}

// Asciidoc formats a comment.Doc as asciidoc.
// The returned string contains leading and trailing empty lines for safety.
// See the [comment.Printer] documentation for ways to customize the output.
func (p *Printer) Asciidoc(d *comment.Doc) string {
	out := &bytes.Buffer{}
	for _, x := range d.Content {
		ensureNewline(out)
		out.WriteByte('\n') // Separate blocks with blank lines.
		p.block(out, x)
	}
	return "\n" + strings.TrimSpace(out.String()) + "\n"
}

// block prints the block x to out.
func (p *Printer) block(out *bytes.Buffer, x comment.Block) {
	switch x := x.(type) {
	default:
		fmt.Fprintf(out, "// ERROR - skipping unknown type %T\n", x)

	case *comment.Paragraph:
		p.Text(out, x.Text)
		ensureNewline(out)

	case *comment.Heading:
		out.WriteString(strings.Repeat("=", p.HeadingLevel+1)) // Heading prefix
		out.WriteString(" ")
		p.Text(out, x.Text)
		ensureNewline(out)

	case *comment.Code:
		out.WriteString("----\n")
		out.WriteString(x.Text)
		ensureNewline(out)
		out.WriteString("----\n")

	case *comment.List:
		loose := x.BlankBetween()
		for i, item := range x.Items {
			ensureNewline(out)
			if i > 0 && loose {
				out.WriteString("\n")
			}
			if n := item.Number; n != "" {
				out.WriteString(n)
				out.WriteString(". ")
			} else {
				out.WriteString("* ")
			}
			for i, blk := range item.Content {
				if i > 0 {
					out.WriteString("\n    ")
				}
				p.Text(out, blk.(*comment.Paragraph).Text)
				ensureNewline(out)
			}
		}
	}
}

// needEscape matches beginning-of-line sequences that Asciidoc would treat as special.
var (
	needEscape    = regexp.MustCompile(`^([+*#-]|([0-9]*)\\.)(.*)$`)
	replaceEscape = []byte(`\\$1$3`)
)

// Text prints the Text sequence x to out, escaping sequences that Asciidoc would treat as special.
func (p *Printer) Text(out *bytes.Buffer, x []comment.Text) {
	p.raw.Reset()
	p.rawText(&p.raw, x)
	line := bytes.TrimSpace(p.raw.Bytes())
	if len(line) == 0 {
		return
	}
	// Escape what Asciidoc would treat as the start of an unordered list or heading.
	line = needEscape.ReplaceAll(line, replaceEscape)
	out.Write(line)
}

// rawText prints the text sequence x to out,
// without worrying about escaping characters
// that have special meaning at the start of an Asciidoc line.
func (p *Printer) rawText(out *bytes.Buffer, x []comment.Text) {
	for _, t := range x {
		switch t := t.(type) {
		case comment.Plain:
			p.escape(out, string(t))
		case comment.Italic:
			out.WriteString("_")
			p.escape(out, string(t))
			out.WriteString("_")
		case *comment.Link:
			p.link(out, t.URL, t.Text)
		case *comment.DocLink:
			p.link(out, p.docLinkURL(t), t.Text)
		}
	}
}

func (p *Printer) link(out *bytes.Buffer, url string, text []comment.Text) {
	out.WriteString("link:")
	out.WriteString(url)
	out.WriteString("[")
	p.rawText(out, text)
	out.WriteString("]")
}

// escape prints s to out as plain text,
// escaping special characters to avoid being misinterpreted
// as Asciidoc markup sequences.
func (p *Printer) escape(out *bytes.Buffer, s string) {
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\n':
			// Turn all \n into spaces, for a few reasons:
			//   - Avoid introducing paragraph breaks accidentally.
			//   - Avoid the need to reindent after the newline.
			//   - Avoid problems with Asciidoc renderers treating
			//     every mid-paragraph newline as a <br>.
			out.WriteString(s[start:i])
			out.WriteByte(' ')
			start = i + 1
			continue
		case '[', '<', '\\':
			// Not all of these need to be escaped all the time,
			// but is valid and easy to do so.
			// We assume the Asciidoc is being passed to a
			// Asciidoc renderer, not edited by a person,
			// so it's fine to have escapes that are not strictly
			// necessary in some cases.
			out.WriteString(s[start:i])
			out.WriteByte('\\')
			out.WriteByte(s[i])
			start = i + 1
		}
	}
	out.WriteString(s[start:])
}

func (p *Printer) docLinkURL(link *comment.DocLink) string {
	if p.DocLinkURL != nil {
		return p.DocLinkURL(link)
	}
	return link.DefaultURL(p.DocLinkBaseURL)
}

func ensureNewline(out *bytes.Buffer) {
	b := out.Bytes()
	if len(b) > 0 && b[len(b)-1] != '\n' {
		out.WriteByte('\n')
	}
}
