// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package quickrules_test

import (
	"fmt"
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCurrentRule(t *testing.T) {
	d := mock.NewDomain("d", "start", "goal")
	domains := korrel8r.NewDomains()
	domains.Add(d)
	e, err := engine.Build().Domains(d).Engine()
	require.NoError(t, err)
	r, err := rules.NewTemplateRule(
		[]korrel8r.Class{d.Class("start")}, []korrel8r.Class{d.Class("goal")},
		e.NewTemplate("test"),
		`d:goal:{{- currentRule.Name}}({{currentRule.Start | first}})->{{currentRule.Goal | first -}}`,
		domains,
	)
	require.NoError(t, err)
	qs, err := r.Apply(`nothing`)
	require.NoError(t, err)
	assert.Equal(t, "[d:goal:test(d:start)->d:goal]", fmt.Sprintf("%v", qs))
}
