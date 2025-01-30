// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package asciidoc

import (
	_ "embed"
	"go/ast"
	"go/doc"
	"go/doc/comment"
	"go/parser"
	"go/token"
	"io"
	"slices"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/davecgh/go-spew/spew"
	"golang.org/x/tools/go/packages"
)

// Load domains from a package spec.
func Load(pkgSpec string) ([]*Domain, error) {
	conf := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedSyntax | packages.NeedName,
		ParseFile: func(fset *token.FileSet, filename string, src []byte) (*ast.File, error) {
			return parser.ParseFile(fset, filename, src, parser.ParseComments)
		},
	}
	pkgs, err := packages.Load(conf, pkgSpec)
	if err != nil {
		return nil, err
	}
	domains := make([]*Domain, len(pkgs))
	for i := range pkgs {
		docPkg, err := doc.NewFromFiles(pkgs[i].Fset, pkgs[i].Syntax, pkgs[i].PkgPath)
		if err != nil {
			return nil, err
		}
		domains[i] = &Domain{pkg: pkgs[i], Package: docPkg, DocLinkBaseURL: "https://pkg.go.dev"}
	}
	return domains, nil
}

// Domain provides go/doc objects for generating documentation.
type Domain struct {
	Package        *doc.Package // Package is the doc.Package object.
	DocLinkBaseURL string
	pkg            *packages.Package // Package for internal type manipulation.
}

//go:embed templates/domains.tmpl.adoc
var domainTmpl string

// Write domain documentation to a writer.
func (d *Domain) Write(out io.Writer) error {
	funcs := sprig.TxtFuncMap()
	funcs["dump"] = spew.Sdump
	tmpl, err := template.New("domains.tmpl.adoc").Funcs(funcs).Parse(domainTmpl)
	if err != nil {
		return err
	}
	err = tmpl.Execute(out, d)
	return err
}

// Asciidoc parses a godoc string and reformats as asciidoc
func (d *Domain) Asciidoc(godoc string) string {
	doc := d.Package.Parser().Parse(godoc)
	p := NewPrinter(d.Package.Printer())
	// DocURL returns a URL to a normal godoc site.
	p.DocLinkBaseURL = d.DocLinkBaseURL
	p.DocLinkURL = d.docLinkURL
	p.HeadingLevel = 1
	return p.Asciidoc(doc)
}

// Type returns the doc/type definition for name.
func (d *Domain) Type(name string) *doc.Type {
	if i := slices.IndexFunc(d.Package.Types, func(d *doc.Type) bool { return d.Name == name }); i >= 0 {
		return d.Package.Types[i]
	}
	return &doc.Type{}
}

func (d *Domain) docLinkURL(link *comment.DocLink) string {
	// Convert local refs into package-qualified refs for lookup in real Go doc.
	if link.ImportPath == "" {
		link.ImportPath = d.Package.ImportPath
	}
	return link.DefaultURL(d.DocLinkBaseURL)
}

func (d *Domain) DocLinkURL(name string, pathRecv ...string) string {
	link := &comment.DocLink{Name: name}
	if len(pathRecv) > 0 {
		link.ImportPath = pathRecv[0]
	}
	if len(pathRecv) > 1 {
		link.Recv = pathRecv[1]
	}
	return d.docLinkURL(link)
}
