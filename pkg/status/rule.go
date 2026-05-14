// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package status adds statuses to graph nodes.
package status

import (
	"bytes"
	"strings"
	"text/template"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

// Rule generates labels from an object.
type Rule interface {
	// Apply generates labels from a start object.
	// Returns nil if the template produces only blank output.
	Apply(start korrel8r.Object) ([]string, error)
	// Start returns the classes this labeler applies to.
	Start() []korrel8r.Class
	// Name returns the labeler's name.
	Name() string
}

type templateStatus struct {
	tmpl  *template.Template
	start []korrel8r.Class
}

// New returns a status Rule that uses a Go template to generate labels.
func New(start []korrel8r.Class, tmpl *template.Template) Rule {
	return &templateStatus{start: start, tmpl: tmpl}
}

func (l *templateStatus) Name() string            { return l.tmpl.Name() }
func (l *templateStatus) Start() []korrel8r.Class { return l.start }

func (l *templateStatus) Apply(start korrel8r.Object) ([]string, error) {
	b := &bytes.Buffer{}
	if err := l.tmpl.Execute(b, start); err != nil {
		return nil, err
	}
	var statuses []string
	for status := range strings.SplitSeq(b.String(), "\n") {
		status = strings.TrimSpace(status)
		if status == "" {
			continue
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}
