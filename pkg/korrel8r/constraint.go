// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package korrel8r

import (
	"time"

	"github.com/korrel8r/korrel8r/pkg/ptr"
)

// Constraint included in a reference to restrict the resulting objects.
type Constraint struct {
	Limit *uint      `json:"limit,omitempty"` // Limit number of objects returned per query.
	Start *time.Time `json:"start,omitempty"` // Start of time interval to include.
	End   *time.Time `json:"end,omitempty"`   // End of time interval to include.
}

// CompareTime returns -1 if t is before the constraint interval, +1 if it is after,
// and 0 if it is in the interval, or if there is no interval.
// Safe to call with c == nil
func (c *Constraint) CompareTime(t time.Time) int {
	switch {
	case c == nil:
		return 0
	case c.Start != nil && t.Before(*c.Start):
		return -1
	case c.End != nil && t.After(*c.End):
		return +1
	}
	return 0
}

var (
	// DefaultDuration is the global default duration for query constraints.
	DefaultDuration = time.Minute * 10
	// DefaultLimit is the global default max items limit for query constraints.
	DefaultLimit uint = 1000
)

// Default returns a default constraint if c is nil, c otherwise.
func (c *Constraint) Default() *Constraint {
	if c == nil {
		now := time.Now()
		return &Constraint{
			Start: ptr.To(now.Add(-DefaultDuration)),
			End:   ptr.To(now),
			Limit: ptr.To(DefaultLimit),
		}
	}
	return c
}
