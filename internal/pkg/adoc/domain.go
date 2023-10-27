// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package adoc renders go/doc comemnts as asciidoc
//
// Provides objects specifically for documentation korrel8r domain packages.
package adoc

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/ast"
	"go/doc"
	"go/doc/comment"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"regexp"

	"golang.org/x/exp/slices"
)

type Domain struct {
	*doc.Package
	fset *token.FileSet
}

func NewDomain(dir string) (*Domain, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	var files []*ast.File
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if filepath.Ext(path) == ".go" {
			if f, err := parser.ParseFile(fset, path, nil, parser.ParseComments); err != nil {
				return nil, err
			} else {
				files = append(files, f)
			}
		}
	}
	p, err := doc.NewFromFiles(fset, files, "github.com/korrel8r/korrel8r")
	return &Domain{Package: p, fset: fset}, err
}

func (p *Domain) Printer() *Printer {
	return NewPrinter(p.Package.Printer())
}

// Asciidoc formats the domain documentation in Asciidoc.
// Heading level 0 is a section heading "=".
func (p *Domain) Asciidoc(headingLevel int) string {
	pr := p.Printer()
	pr.HeadingLevel = headingLevel
	// Expand doc links to local types into source code.
	pr.DocLinkURL = func(link *comment.DocLink) string {
		if len(link.Text) == 1 {
			if name, ok := link.Text[0].(comment.Plain); ok {
				if t := p.findType(string(name)); t != nil {
					out := &bytes.Buffer{}
					out.WriteString("\n\n----\n")
					printer.Fprint(out, p.fset, t.Decl)
					out.WriteString("\n----\n\n")
					return out.String()
				} else {
					return fmt.Sprintf("`%v`", name)
				}
			}
		}
		return ""
	}
	adoc := strings.TrimSpace(pr.Asciidoc(p.Parser().Parse(p.Doc)))
	// Clean up stray spaces caused by expanding DocURLs as code blocks:
	adoc = regexp.MustCompile(`(?m)----\n\n `).ReplaceAllString(adoc, "----\n\n")
	adoc = regexp.MustCompile(`(?m) \n\n----$`).ReplaceAllString(adoc, "\n\n----")
	// Replace the leading "package name" if present with "Domain name"
	adoc = strings.Replace(adoc, "package "+p.Name, "Domain "+p.Name, 1)
	return adoc
}

func (p *Domain) findType(name string) *doc.Type {
	if i := slices.IndexFunc(p.Types, func(d *doc.Type) bool { return d.Name == name }); i >= 0 {
		return p.Types[i]
	}
	return nil
}
