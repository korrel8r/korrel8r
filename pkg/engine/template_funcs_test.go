// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package engine

import (
	"bytes"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func execTemplate(t *testing.T, tmplStr string, data any) (string, error) {
	t.Helper()
	funcs := template.FuncMap{
		"assert":   templateAssert,
		"required": templateRequired,
	}
	tmpl, err := template.New("test").Funcs(funcs).Parse(tmplStr)
	require.NoError(t, err)
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	return buf.String(), err
}

func TestAssert(t *testing.T) {
	for _, tc := range []struct {
		name    string
		tmpl    string
		data    any
		wantErr string
	}{
		{name: "true", tmpl: `{{ assert true }}`},
		{name: "false", tmpl: `{{ assert false }}`, wantErr: "assertion failed"},
		{name: "message", tmpl: `{{ assert "bad value" false }}`, wantErr: "assertion failed: bad value"},
		{name: "message_pass", tmpl: `{{ assert "bad value" true }}`},
		{name: "expression", tmpl: `{{ assert (eq .X "yes") }}`, data: map[string]any{"X": "yes"}},
		{name: "expression_fail", tmpl: `{{ assert (eq .X "yes") }}`, data: map[string]any{"X": "no"}, wantErr: "assertion failed"},
		{name: "piped", tmpl: `{{ true | assert }}`},
		{name: "piped_fail", tmpl: `{{ false | assert }}`, wantErr: "assertion failed"},
		{name: "piped_message", tmpl: `{{ false | assert "oops" }}`, wantErr: "assertion failed: oops"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := execTemplate(t, tc.tmpl, tc.data)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
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
			assert.Equal(t, tc.want, isEmpty(tc.v))
		})
	}
}

func TestRequired(t *testing.T) {
	for _, tc := range []struct {
		name    string
		tmpl    string
		data    any
		want    string
		wantErr string
	}{
		{name: "string", tmpl: `{{ required .X }}`, data: map[string]any{"X": "hello"}, want: "hello"},
		{name: "empty_string", tmpl: `{{ required .X }}`, data: map[string]any{"X": ""}, wantErr: "required value was not set"},
		{name: "nil", tmpl: `{{ required .X }}`, data: map[string]any{"X": nil}, wantErr: "required value was not set"},
		{name: "message", tmpl: `{{ required "need X" .X }}`, data: map[string]any{"X": ""}, wantErr: "need X"},
		{name: "message_pass", tmpl: `{{ required "need X" .X }}`, data: map[string]any{"X": "val"}, want: "val"},
		{name: "piped", tmpl: `{{ .X | required }}`, data: map[string]any{"X": "val"}, want: "val"},
		{name: "piped_fail", tmpl: `{{ .X | required }}`, data: map[string]any{"X": ""}, wantErr: "required value was not set"},
		{name: "piped_message", tmpl: `{{ .X | required "missing" }}`, data: map[string]any{"X": ""}, wantErr: "missing"},
		{name: "zero_int", tmpl: `{{ required .X }}`, data: map[string]any{"X": 0}, wantErr: "required value was not set"},
		{name: "nonzero_int", tmpl: `{{ required .X }}`, data: map[string]any{"X": 42}, want: "42"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := execTemplate(t, tc.tmpl, tc.data)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.want, got)
			}
		})
	}
}
