// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package impl

import (
	"fmt"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

// Domain is a minimal implementation of korrel8r.Domain.
// New domains can embed it and redefine methods as needed.
// [korrel8r.Domain.Query] and  [korrel8r.Domain.Store] are not provided.
type Domain struct {
	name     string
	summary  string
	detail   string
	classes  []korrel8r.Class
	classMap map[string]korrel8r.Class
}

// NewDomain creates a new domain.
// Domains with a static list of classes can provide it here.
func NewDomain(name, summary, detail string, classes ...korrel8r.Class) *Domain {
	d := &Domain{
		name:     name,
		summary:  summary,
		detail:   detail,
		classes:  classes,
		classMap: make(map[string]korrel8r.Class),
	}
	for _, c := range classes {
		d.classMap[c.Name()] = c
	}
	return d
}

func DomainDescription(name, summary, detail string) (string, string) {
	return summary, fmt.Sprintf("# %s\n\n%s\n\n%s", name, summary, detail)
}
func (d *Domain) Description() (string, string) {
	return DomainDescription(d.name, d.summary, d.detail)
}
func (d *Domain) Name() string                     { return d.name }
func (d *Domain) Class(name string) korrel8r.Class { return d.classMap[name] }
func (d *Domain) Classes() []korrel8r.Class        { return d.classes }

func (d *Domain) String() string { return d.name }
