// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package must

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorIf_NilErr(t *testing.T) {
	assert.NoError(t, ErrorIf(nil, "should not appear"))
}

func TestErrorIf_NonNilErr(t *testing.T) {
	err := ErrorIf(errors.New("x"), "wrapped: %v", "detail")
	assert.EqualError(t, err, "wrapped: detail")
}

func TestMust_NilErr(t *testing.T) {
	assert.NotPanics(t, func() { Must(nil) })
}

func TestMust_PanicsWithErr(t *testing.T) {
	assert.PanicsWithError(t, "boom", func() { Must(errors.New("boom")) })
}

func TestMust_PanicsWithFormat(t *testing.T) {
	assert.PanicsWithError(t, "bad: thing", func() { Must(errors.New("x"), "bad: %v", "thing") })
}

func TestMust1(t *testing.T) {
	assert.Equal(t, 42, Must1(42, nil))
	assert.Panics(t, func() { Must1(0, errors.New("err")) })
}

func TestMust2(t *testing.T) {
	a, b := Must2(1, "two", nil)
	assert.Equal(t, 1, a)
	assert.Equal(t, "two", b)
	assert.Panics(t, func() { Must2(0, "", errors.New("err")) })
}
