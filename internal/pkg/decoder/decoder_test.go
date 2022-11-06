package decoder

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func assertReadOK(t *testing.T, want string, buf []byte, n int, err error) {
	t.Helper()
	assert.NoError(t, err, "read error")
	assert.Equal(t, len(want), n, "length mismatch")
	assert.Equal(t, want, string(buf[:len(want)]), "data mismatch")
}

func TestLineCountReader(t *testing.T) {
	s := "one\ntwo\nthree\n"
	buf := make([]byte, len(s))

	t.Run("read full line", func(t *testing.T) {
		lr := NewLineCountReader(strings.NewReader(s))
		assert.Equal(t, 0, lr.Line)
		n, err := lr.Read(buf[:4])
		assertReadOK(t, "one\n", buf, n, err)
		assert.Equal(t, 1, lr.Line)
	})

	t.Run("read part line", func(t *testing.T) {
		lr := NewLineCountReader(strings.NewReader(s))
		n, err := lr.Read(buf[:2])
		assertReadOK(t, "on", buf, n, err)
		assert.Equal(t, 0, lr.Line)

		n, err = lr.Read(buf[:4])
		assertReadOK(t, "e\n", buf, n, err)
		assert.Equal(t, 1, lr.Line)
	})

	t.Run("read multi-line", func(t *testing.T) {
		lr := NewLineCountReader(strings.NewReader(s))
		for i, s := range []string{"one\n", "two\n", "three\n"} {
			n, err := lr.Read(buf)
			assertReadOK(t, s, buf, n, err)
			assert.NoError(t, err)
			assert.Equal(t, i+1, lr.Line)
		}
	})
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
		{`
a: b
: x`, "error converting YAML to JSON: yaml: line 2: did not find expected key", 2},
		{`{"a":"b"`, "unexpected EOF", 0},
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
