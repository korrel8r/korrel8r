package templaterule_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/korrel8/korrel8/internal/pkg/test/mock"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/korrel8/korrel8/pkg/templaterule"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestRule_Apply(t *testing.T) {
	tr, err := templaterule.New("myrule", mock.Class(""), mock.Class(""), `"object: {{.Name}}, constraint: {{ constraint }}"`)
	require.NoError(t, err)
	now := time.Now()
	constraint := korrel8.Constraint{Start: &now, End: &now}
	q, err := tr.Apply(mock.NewObject("thing", ""), &constraint)
	assert.NoError(t, err)
	assert.Equal(t, mock.NewQuery(fmt.Sprintf("object: thing, constraint: %v", constraint)), q)
}

func TestRule_Error(t *testing.T) {
	tr, err := templaterule.New("myrule", mock.Class(""), mock.Class(""), `{{fail "foobar"}}`)
	require.NoError(t, err)
	_, err = tr.Apply(mock.NewObject("thing", ""), nil)
	assert.Equal(t, "error applying myrule to mock: template: myrule:1:2: executing \"myrule\" at <fail \"foobar\">: error calling fail: foobar", err.Error())
}

func TestRule_MissingKey(t *testing.T) {
	tr, err := templaterule.New(t.Name(), mock.Class(""), mock.Class(""), `{{.nosuchkey}}`)
	require.NoError(t, err)
	_, err = tr.Apply(mock.NewObject("thing", ""), nil)
	assert.Contains(t, err.Error(), "can't evaluate field nosuchkey")
}
