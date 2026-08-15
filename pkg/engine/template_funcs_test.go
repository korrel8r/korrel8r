// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package engine

import (
	"bytes"
	"testing"
	"text/template"

	"github.com/korrel8r/korrel8r/pkg/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func execTemplate(t *testing.T, tmplStr string, data any) (string, error) {
	t.Helper()
	funcs := template.FuncMap{
		"requireAll": func(values ...any) string { rules.RequireAll(values...); return "" },
		"require":    func(v any) any { return rules.Require(v) },
	}
	tmpl, err := template.New("test").Funcs(funcs).Parse(tmplStr)
	require.NoError(t, err)
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	return buf.String(), err
}

func TestRequireAll(t *testing.T) {
	for _, tc := range []struct {
		name    string
		tmpl    string
		data    any
		wantErr bool
	}{
		{name: "true", tmpl: `{{ requireAll true }}`},
		{name: "false", tmpl: `{{ requireAll false }}`, wantErr: true},
		{name: "multiple_pass", tmpl: `{{ requireAll "a" 1 true }}`},
		{name: "multiple_fail", tmpl: `{{ requireAll "a" 0 true }}`, wantErr: true},
		{name: "expression", tmpl: `{{ requireAll (eq .X "yes") }}`, data: map[string]any{"X": "yes"}},
		{name: "expression_fail", tmpl: `{{ requireAll (eq .X "yes") }}`, data: map[string]any{"X": "no"}, wantErr: true},
		{name: "piped", tmpl: `{{ true | requireAll }}`},
		{name: "piped_fail", tmpl: `{{ false | requireAll }}`, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := execTemplate(t, tc.tmpl, tc.data)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestIsEmpty(t *testing.T) {
	type s struct{ X int }
	for _, tc := range []struct {
		name string
		v    any
		want bool
	}{
		{name: "nil", v: nil, want: true},
		{name: "true", v: true, want: false},
		{name: "false", v: false, want: true},
		{name: "string", v: "hello", want: false},
		{name: "empty_string", v: "", want: true},
		{name: "int_zero", v: 0, want: true},
		{name: "int_nonzero", v: 42, want: false},
		{name: "int64_zero", v: int64(0), want: true},
		{name: "int64_nonzero", v: int64(1), want: false},
		{name: "float64_zero", v: float64(0), want: true},
		{name: "float64_nonzero", v: 3.14, want: false},
		{name: "empty_slice", v: []any{}, want: true},
		{name: "nonempty_slice", v: []any{1}, want: false},
		{name: "empty_int_slice", v: []int{}, want: true},
		{name: "nonempty_int_slice", v: []int{1}, want: false},
		{name: "empty_map", v: map[string]any{}, want: true},
		{name: "nonempty_map", v: map[string]any{"a": 1}, want: false},
		{name: "empty_int_map", v: map[int]int{}, want: true},
		{name: "nonempty_int_map", v: map[int]int{1: 2}, want: false},
		{name: "zero_struct", v: s{}, want: true},
		{name: "nonzero_struct", v: s{X: 1}, want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, rules.Empty(tc.v))
		})
	}
}

func TestRequire(t *testing.T) {
	for _, tc := range []struct {
		name    string
		tmpl    string
		data    any
		want    string
		wantErr bool
	}{
		{name: "string", tmpl: `{{ require .X }}`, data: map[string]any{"X": "hello"}, want: "hello"},
		{name: "empty_string", tmpl: `{{ require .X }}`, data: map[string]any{"X": ""}, wantErr: true},
		{name: "nil", tmpl: `{{ require .X }}`, data: map[string]any{"X": nil}, wantErr: true},
		{name: "piped", tmpl: `{{ .X | require }}`, data: map[string]any{"X": "val"}, want: "val"},
		{name: "piped_fail", tmpl: `{{ .X | require }}`, data: map[string]any{"X": ""}, wantErr: true},
		{name: "zero_int", tmpl: `{{ require .X }}`, data: map[string]any{"X": 0}, wantErr: true},
		{name: "nonzero_int", tmpl: `{{ require .X }}`, data: map[string]any{"X": 42}, want: "42"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := execTemplate(t, tc.tmpl, tc.data)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.want, got)
			}
		})
	}
}
