// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rules

import (
	"testing"
	"text/template"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateRule_Apply(t *testing.T) {
	d := mock.NewDomain("test", "a", "b")
	a, b := d.Class("a"), d.Class("b")
	tmpl := template.Must(template.New("test-rule").Parse(`test:b:got-{{.}}`))
	rule := NewTemplateRule([]korrel8r.Class{a}, []korrel8r.Class{b}, tmpl)

	queries, err := rule.Apply("hello")
	require.NoError(t, err)
	require.Len(t, queries, 1)
	assert.Equal(t, "test:b:got-hello", queries[0].String())
}

func TestTemplateRule_Apply_MultiLine(t *testing.T) {
	d := mock.NewDomain("test", "a", "b")
	a, b := d.Class("a"), d.Class("b")
	tmpl := template.Must(template.New("multi").Parse("test:b:first\ntest:b:second"))
	rule := NewTemplateRule([]korrel8r.Class{a}, []korrel8r.Class{b}, tmpl)

	queries, err := rule.Apply("x")
	require.NoError(t, err)
	require.Len(t, queries, 2)
	assert.Equal(t, "test:b:first", queries[0].String())
	assert.Equal(t, "test:b:second", queries[1].String())
}

func TestTemplateRule_Apply_Blank(t *testing.T) {
	d := mock.NewDomain("test", "a", "b")
	a, b := d.Class("a"), d.Class("b")
	tmpl := template.Must(template.New("blank").Parse("  \n  \n  "))
	rule := NewTemplateRule([]korrel8r.Class{a}, []korrel8r.Class{b}, tmpl)

	queries, err := rule.Apply("x")
	require.NoError(t, err)
	assert.Empty(t, queries)
}

func TestTemplateRule_Accessors(t *testing.T) {
	d := mock.NewDomain("test", "a", "b")
	a, b := d.Class("a"), d.Class("b")
	tmpl := template.Must(template.New("myrule").Parse("test:b:x"))
	rule := NewTemplateRule([]korrel8r.Class{a}, []korrel8r.Class{b}, tmpl)

	assert.Equal(t, "myrule", rule.Name())
	assert.Equal(t, []korrel8r.Class{a}, rule.Start())
	assert.Equal(t, []korrel8r.Class{b}, rule.Goal())
}

func TestTemplateRule_Apply_TemplateError(t *testing.T) {
	d := mock.NewDomain("test", "a", "b")
	a, b := d.Class("a"), d.Class("b")
	tmpl := template.Must(template.New("bad").Option("missingkey=error").Parse(`test:b:{{.NoSuchField}}`))
	rule := NewTemplateRule([]korrel8r.Class{a}, []korrel8r.Class{b}, tmpl)

	_, err := rule.Apply("a string")
	assert.Error(t, err)
}
