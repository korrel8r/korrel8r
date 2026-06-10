// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package korrel8r

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConstraint(t *testing.T) {
	start := time.Now()
	end := start.Add(time.Minute)
	early, ontime, late := start.Add(-1), start.Add(1), end.Add(1)
	c := Constraint{Start: &start, End: &end}
	assert.Zero(t, c.CompareTime(start))
	assert.Zero(t, c.CompareTime(end))
	assert.Zero(t, c.CompareTime(ontime))
	assert.Less(t, c.CompareTime(early), 0)
	assert.Greater(t, c.CompareTime(late), 0)
}

func TestConstraint_CompareTime_Nil(t *testing.T) {
	assert.Zero(t, (*Constraint)(nil).CompareTime(time.Now()))
}

func TestConstraint_CompareTime_ZeroTime(t *testing.T) {
	start := time.Now()
	c := Constraint{Start: &start}
	assert.Zero(t, c.CompareTime(time.Time{}))
}

func TestConstraint_String(t *testing.T) {
	limit := 10
	c := Constraint{Limit: &limit}
	assert.Contains(t, c.String(), `"limit":10`)
}

func TestConstraint_Default(t *testing.T) {
	c := (*Constraint)(nil).Default()
	assert.NotNil(t, c.Limit)
	assert.Equal(t, 1000, *c.Limit)
	assert.NotNil(t, c.Start)
	assert.NotNil(t, c.End)
	assert.True(t, c.Start.Before(*c.End))
}

func TestConstraint_Default_PreservesExisting(t *testing.T) {
	limit := 5
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	c := (&Constraint{Limit: &limit, Start: &start}).Default()
	assert.Equal(t, 5, *c.Limit)
	assert.Equal(t, start, *c.Start)
	assert.NotNil(t, c.End) // filled in
}

func TestConstraint_GetLimit(t *testing.T) {
	assert.Equal(t, 0, (*Constraint)(nil).GetLimit())
	assert.Equal(t, 0, (&Constraint{}).GetLimit())
	limit := 42
	assert.Equal(t, 42, (&Constraint{Limit: &limit}).GetLimit())
}

func TestConstraint_GetStart(t *testing.T) {
	assert.True(t, (*Constraint)(nil).GetStart().IsZero())
	assert.True(t, (&Constraint{}).GetStart().IsZero())
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, start, (&Constraint{Start: &start}).GetStart())
}

func TestConstraint_GetEnd(t *testing.T) {
	assert.True(t, (*Constraint)(nil).GetEnd().IsZero())
	assert.True(t, (&Constraint{}).GetEnd().IsZero())
	end := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, end, (&Constraint{End: &end}).GetEnd())
}
