// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package text

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testPrinter(t *testing.T, domains ...*mock.Domain) *Printer {
	t.Helper()
	b := engine.Build()
	for _, d := range domains {
		b.Domains(d)
	}
	e, err := b.Engine()
	require.NoError(t, err)
	return NewPrinter(e)
}

func TestSummary(t *testing.T) {
	assert.Equal(t, "first line", Summary("first line\nsecond line\nthird"))
	assert.Equal(t, "only line", Summary("only line"))
	assert.Equal(t, "", Summary(""))
}

func TestWriteString(t *testing.T) {
	s := WriteString(func(w io.Writer) {
		_, _ = io.WriteString(w, "hello world")
	})
	assert.Equal(t, "hello world", s)
}

func TestPrinter_ListDomains(t *testing.T) {
	d := mock.NewDomain("alpha", "x", "y")
	p := testPrinter(t, d)
	var buf bytes.Buffer
	p.ListDomains(&buf)
	out := buf.String()
	assert.Contains(t, out, "alpha")
	assert.Contains(t, out, "Mock domain.")
}

func TestPrinter_ListClasses(t *testing.T) {
	d := mock.NewDomain("testdom", "classA", "classB")
	p := testPrinter(t, d)
	var buf bytes.Buffer
	p.ListClasses(&buf, d)
	out := buf.String()
	assert.Contains(t, out, "classA")
	assert.Contains(t, out, "classB")
}

func TestPrinter_DescribeDomains(t *testing.T) {
	d1 := mock.NewDomain("dom1", "a")
	d2 := mock.NewDomain("dom2", "b")
	p := testPrinter(t, d1, d2)
	var buf bytes.Buffer
	p.DescribeDomains(&buf)
	out := buf.String()
	assert.Contains(t, out, "## dom1")
	assert.Contains(t, out, "## dom2")
}

func TestPrinter_DescribeDomain(t *testing.T) {
	d := mock.NewDomain("mydom")
	p := testPrinter(t, d)
	var buf bytes.Buffer
	p.DescribeDomain(&buf, d)
	out := buf.String()
	assert.Contains(t, out, "## mydom")
	assert.Contains(t, out, "Mock domain.")
}

func TestPrinter_Error(t *testing.T) {
	d := mock.NewDomain("d")
	p := testPrinter(t, d)
	var buf bytes.Buffer
	p.Error(&buf, fmt.Errorf("something went wrong"))
	assert.Contains(t, buf.String(), "something went wrong")
}
