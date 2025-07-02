// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package text is used to print results as text for command line and MCP.
package text

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

type Printer struct{ *engine.Engine }

func NewPrinter(e *engine.Engine) *Printer { return &Printer{Engine: e} }

func WriteString(print func(io.Writer)) string {
	w := &strings.Builder{}
	print(w)
	return w.String()
}

func (e *Printer) ListDomains(w io.Writer) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	defer func() { _ = tw.Flush() }()
	for _, d := range e.Domains() {
		fmt.Fprintf(tw, "%v\t%v", d.Name(), d.Description())
		fmt.Fprintln(tw)
	}
}

func (p *Printer) ListClasses(w io.Writer, d korrel8r.Domain) {
	classes, err := p.ClassesFor(d)
	if err != nil {
		p.Error(w, err)
		return
	}
	for _, c := range classes {
		fmt.Fprintln(w, c.Name())
	}
}

func (p *Printer) Error(w io.Writer, err error) {
	fmt.Fprintln(w, "Error: ", err)
}
