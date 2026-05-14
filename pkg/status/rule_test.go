// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package status

import (
	"testing"
	"text/template"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateStatus_Apply(t *testing.T) {
	d := mock.NewDomain("test", "foo")
	c := d.Class("foo")

	for _, tc := range []struct {
		name     string
		tmpl     string
		obj      any
		expected []string
	}{
		{
			name:     "single status",
			tmpl:     `hello`,
			obj:      map[string]any{"x": 1},
			expected: []string{"hello"},
		},
		{
			name:     "multiple statuses",
			tmpl:     "status1\nstatus2\nstatus3",
			obj:      map[string]any{},
			expected: []string{"status1", "status2", "status3"},
		},
		{
			name:     "blank lines skipped",
			tmpl:     "status1\n\n  \nstatus2\n",
			obj:      map[string]any{},
			expected: []string{"status1", "status2"},
		},
		{
			name:     "all blank",
			tmpl:     "\n  \n  \n",
			obj:      map[string]any{},
			expected: nil,
		},
		{
			name:     "template with data",
			tmpl:     `{{.name}}`,
			obj:      map[string]any{"name": "my-status"},
			expected: []string{"my-status"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmpl := template.Must(template.New(tc.name).Parse(tc.tmpl))
			lb := New([]korrel8r.Class{c}, tmpl)
			assert.Equal(t, tc.name, lb.Name())
			assert.Equal(t, []korrel8r.Class{c}, lb.Start())
			statuses, err := lb.Apply(tc.obj)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, statuses)
		})
	}
}

func TestTemplateStatus_Apply_Error(t *testing.T) {
	d := mock.NewDomain("test", "foo")
	c := d.Class("foo")
	tmpl := template.Must(template.New("err").Option("missingkey=error").Parse(`{{.missing}}`))
	lb := New([]korrel8r.Class{c}, tmpl)
	_, err := lb.Apply(map[string]any{})
	assert.Error(t, err)
}
