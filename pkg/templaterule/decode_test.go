package templaterule

import (
	"strings"
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/decoder"
	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddRules(t *testing.T) {
	e := engine.New()
	foo := mock.Domain("foo a z")
	e.AddDomain(foo, nil)
	a, z := foo.Class("a"), foo.Class("z")

	d := decoder.New(strings.NewReader(`
# Comment creates empty doc
---
---
name:   "one"
start:  {domain: "foo", classes: [a]}
goal:   {domain: "foo", classes: [z]}
result: {query: dummy, class: dummy}
---
# Comment creates empty doc
---
---
name:   "two"
start:  {domain: "foo", classes: [a]}
goal:   {domain: "foo", classes: [z]}
result: {query: dummy, class: dummy}
`))

	require.NoError(t, AddRules(d, e))
	want := []mock.Rule{mockRule("one", a, z), mockRule("two", a, z)}
	assert.Equal(t, want, mockRules(e.Rules()...))
}
