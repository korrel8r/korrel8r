package templaterule_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/alanconway/korrel8/pkg/korrel8"
	"github.com/alanconway/korrel8/pkg/templaterule"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

type mockClass string

func (c mockClass) Domain() korrel8.Domain { return nil }
func (c mockClass) New() korrel8.Object    { return nil }
func (c mockClass) String() string         { return "" }

type mockObject string

func (o mockObject) Native() any                    { return o }
func (o mockObject) Identifier() korrel8.Identifier { return o }

func TestRule_Apply(t *testing.T) {
	tr, err := templaterule.New("myrule", mockClass(""), mockClass(""), "object: {{.}}, constraint: {{ constraint }}")
	require.NoError(t, err)
	now := time.Now()
	constraint := korrel8.Constraint{Start: &now, End: &now}
	q, err := tr.Apply(mockObject("thing"), &constraint)
	assert.NoError(t, err)
	assert.Equal(t, []string{fmt.Sprintf("object: thing, constraint: %v", constraint)}, q)
}

func TestRule_DoesNotApply(t *testing.T) {
	tr, err := templaterule.New("myrule", mockClass(""), mockClass(""), `{{doesnotapply}}`)
	require.NoError(t, err)
	_, err = tr.Apply(mockObject("thing"), nil)
	assert.Contains(t, err.Error(), "does not apply")
}

func TestRule_MissingKey(t *testing.T) {
	tr, err := templaterule.New(t.Name(), mockClass(""), mockClass(""), `{{.nosuchkey}}`)
	require.NoError(t, err)
	_, err = tr.Apply(mockObject("thing"), nil)
	assert.Contains(t, err.Error(), "can't evaluate field nosuchkey")
}
