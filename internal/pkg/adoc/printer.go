// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package adoc

import (
	"bytes"
	"fmt"
	"go/doc/comment"
	"strings"
)

// Printer prints go/doc comments in asciidoc format.
//
// Based on [comment.MarkdownPrinter]
type Printer struct {
	*comment.Printer
	raw bytes.Buffer
}

func NewPrinter(p *comment.Printer) *Printer { return &Printer{Printer: p} }

// Asciidoc returns a Asciidoc formatting of a comment.Doc object.
// See the [comment.Printer] documentation for ways to customize the Asciidoc output.
func (p *Printer) Asciidoc(d *comment.Doc) string {
	var out bytes.Buffer
	for i, x := range d.Content {
		if i > 0 {
			out.WriteByte('\n')
		}
		p.block(&out, x)
	}
	return out.String()
}

// block prints the block x to out.
func (p *Printer) block(out *bytes.Buffer, x comment.Block) {
	switch x := x.(type) {
	default:
		fmt.Fprintf(out, "?%T", x)

	case *comment.Paragraph:
		p.text(out, x.Text)
		out.WriteString("\n")

	case *comment.Heading:
		out.WriteString(strings.Repeat("=", p.HeadingLevel+1))
		out.WriteString(" ")
		p.text(out, x.Text)
		out.WriteString("\n")

	case *comment.Code:
		out.WriteString("----\n")
		out.WriteString(x.Text)
		if !strings.HasSuffix(x.Text, "\n") {
			out.WriteString("\n")
		}
		out.WriteString("----\n")

	case *comment.List:
		loose := x.BlankBetween()
		for i, item := range x.Items {
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
				const fourSpace = "    "
				if i > 0 {
					out.WriteString("\n" + fourSpace)
				}
				p.text(out, blk.(*comment.Paragraph).Text)
				out.WriteString("\n")
			}
		}
	}
}

// text prints the text sequence x to out.
func (p *Printer) text(out *bytes.Buffer, x []comment.Text) {
	p.raw.Reset()
	p.rawText(&p.raw, x)
	line := bytes.TrimSpace(p.raw.Bytes())
	if len(line) == 0 {
		return
	}
	switch line[0] {
	case '+', '-', '*', '#':
		// Escape what would be the start of an unordered list or heading.
		out.WriteByte('\\')
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		i := 1
		for i < len(line) && '0' <= line[i] && line[i] <= '9' {
			i++
		}
		if i < len(line) && (line[i] == '.' || line[i] == ')') {
			// Escape what would be the start of an ordered list.
			out.Write(line[:i])
			out.WriteByte('\\')
			line = line[i:]
		}
	}
	out.Write(line)
}

// rawText prints the text sequence x to out,
// without worrying about escaping characters
// that have special meaning at the start of a Asciidoc line.
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
			out.WriteString(t.URL)
			out.WriteString("[")
			p.rawText(out, t.Text)
			out.WriteString("]")
		case *comment.DocLink:
			if f := p.DocLinkURL; f != nil {
				out.WriteString(f(t))
			} else {
				out.WriteString("`")
				p.rawText(out, t.Text)
				out.WriteString("`")
			}
		}
	}
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
