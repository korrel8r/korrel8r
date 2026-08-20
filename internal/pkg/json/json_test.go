// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package json_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalUnmarshal(t *testing.T) {
	type T struct {
		Name  string         `json:"name"`
		Count int            `json:"count"`
		Tags  map[string]int `json:"tags"`
	}
	want := T{Name: "test", Count: 42, Tags: map[string]int{"a": 1}}
	b, err := json.Marshal(want)
	require.NoError(t, err)

	var got T
	require.NoError(t, json.Unmarshal(b, &got))
	assert.Equal(t, want, got)
}

func TestMarshalIndent(t *testing.T) {
	v := map[string]int{"a": 1}
	b, err := json.MarshalIndent(v, "", "  ")
	require.NoError(t, err)
	assert.Contains(t, string(b), "\n")
	assert.Contains(t, string(b), `"a"`)
}

func TestValid(t *testing.T) {
	assert.True(t, json.Valid([]byte(`{"a":1}`)))
	assert.False(t, json.Valid([]byte(`{invalid`)))
}

func TestEncoderDecoder(t *testing.T) {
	type T struct{ X int }
	want := T{X: 99}

	var buf bytes.Buffer
	require.NoError(t, json.NewEncoder(&buf).Encode(want))

	var got T
	require.NoError(t, json.NewDecoder(strings.NewReader(buf.String())).Decode(&got))
	assert.Equal(t, want, got)
}

func TestRawMessage(t *testing.T) {
	raw := json.RawMessage(`{"nested":true}`)
	b, err := json.Marshal(raw)
	require.NoError(t, err)

	var got json.RawMessage
	require.NoError(t, json.Unmarshal(b, &got))
	assert.JSONEq(t, `{"nested":true}`, string(got))
}
