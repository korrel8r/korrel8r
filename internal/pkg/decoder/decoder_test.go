package decoder

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLineCountReader(t *testing.T) {
	r := strings.NewReader(`
one
two
three
`)
	lr := NewLineCountReader(r)
	assert.Equal(t, 1, lr.Line())
	data := make([]byte, 3)
	n, err := lr.Read(data)
	assert.NoError(t, err)
	assert.Equal(t, 3, n)
	n, _ = lr.Read(data[:1])
	assert.Equal(t, 2, lr.Line())
	data = make([]byte, 100)
	n, _ = lr.Read(data)
	assert.Equal(t, 11, n)
	assert.Equal(t, 5, lr.Line())
}

func TestDecoder_DecodeGood(t *testing.T) {
	for _, x := range []struct {
		data string
		want []map[string]any
	}{
		{`{"a":"b"}{"c":"d"}`, []map[string]any{{"a": "b"}, {"c": "d"}}},
		{`
a: b
---
c: d
`, []map[string]any{{"a": "b"}, {"c": "d"}}},
	} {
		t.Run(x.data, func(t *testing.T) {
			d := New(strings.NewReader(x.data))
			for _, w := range x.want {
				var got map[string]any
				assert.NoError(t, d.Decode(&got))
				assert.Equal(t, w, got)
			}
			var got map[string]any
			assert.Equal(t, io.EOF, d.Decode(&got))
		})
	}
}

func TestDecoder_DecodeBad(t *testing.T) {
	for _, x := range []struct {
		data string
		err  string
		line int
	}{
		{`{"a":"b"`, "unexpected EOF", 1},
		{`
a: b
: x`, "error converting YAML to JSON: yaml: line 2: did not find expected key", 3},
	} {
		t.Run(x.data, func(t *testing.T) {
			d := New(strings.NewReader(x.data))
			var got map[string]any
			assert.EqualError(t, d.Decode(&got), x.err)
			assert.Equal(t, x.line, d.Line())
			assert.Nil(t, got)

		})
	}
}
