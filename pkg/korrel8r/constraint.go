// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package korrel8r

import (
	"encoding/json"
	"time"

	"github.com/korrel8r/korrel8r/pkg/ptr"
)

// Constraint included in a store Get operation to restrict the resulting objects.
type Constraint struct {
	// Limit number of objects returned per query
	Limit *int `json:"limit,omitempty"`
	// Start ignore data before this time (RFC 3339)
	Start *time.Time `json:"start,omitempty"`
	// End ignore data after this time (RFC 3339)
	End *time.Time `json:"end,omitempty"`
}

func (c *Constraint) String() string {
	s, err := json.Marshal(c)
	if err != nil {
		return err.Error()
	}
	return string(s)
}

// CompareTime returns -1 if t is before the constraint interval, +1 if it is after,
// and 0 if: in the interval, there is no interval or when.IsZero().
// Safe to call with c == nil
func (c *Constraint) CompareTime(when time.Time) int {
	switch {
	case c == nil:
		return 0
	case c.Start != nil && !when.IsZero() && when.Before(*c.Start):
		return -1
	case c.End != nil && !when.IsZero() && when.After(*c.End):
		return +1
	}
	return 0
}

// Default fills in default values. Safe to call with c == nil.
func (c *Constraint) Default() *Constraint {
	// Hard-wired fallback defaults
	const defaultDuration = time.Hour
	const defaultLimit = 1000

	if c == nil {
		return (&Constraint{}).Default()
	}
	if c.Limit == nil {
		c.Limit = ptr.To(defaultLimit)
	}
	if c.End == nil {
		c.End = ptr.To(time.Now())
	}
	if c.Start == nil {
		c.Start = ptr.To(c.End.Add(-defaultDuration))
	}
	return c
}

// GetLimit returns limit or 0, safe to call with c == nil
func (c *Constraint) GetLimit() int {
	if c != nil && c.Limit != nil {
		return *c.Limit
	}
	return 0
}

func (c *Constraint) GetStart() time.Time {
	if c != nil && c.Start != nil {
		return *c.Start
	}
	return time.Time{}
}

func (c *Constraint) GetEnd() time.Time {
	if c != nil && c.End != nil {
		return *c.End
	}
	return time.Time{}
}
