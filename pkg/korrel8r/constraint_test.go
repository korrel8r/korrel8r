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
