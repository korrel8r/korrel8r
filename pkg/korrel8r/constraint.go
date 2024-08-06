// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package korrel8r

import (
	"time"

	"github.com/korrel8r/korrel8r/pkg/ptr"
)

// Constraint included in a reference to restrict the resulting objects.
type Constraint struct {
	Limit   *int           `json:"limit,omitempty"`   // Limit number of objects returned per query, <=0 means no limit.
	Timeout *time.Duration `json:"timeout,omitempty"` // Timeout per request.
	Start   *time.Time     `json:"start,omitempty"`   // Start of time interval to include.
	End     *time.Time     `json:"end,omitempty"`     // End of time interval to include.
}

// CompareTime returns -1 if t is before the constraint interval, +1 if it is after,
// and 0 if it is in the interval, or if there is no interval.
// Safe to call with c == nil
func (c *Constraint) CompareTime(when time.Time) int {
	switch {
	case c == nil:
		return 0
	case c.Start != nil && when.Before(*c.Start):
		return -1
	case c.End != nil && when.After(*c.End):
		return +1
	}
	return 0
}

// Default values can be modified in init() or main(), but not after korrel8r functions are called.
var (
	// DefaultDuration is the global default duration for query constraints.
	DefaultDuration = time.Minute * 10
	// DefaultLimit is the global default max items limit for query constraints.
	DefaultLimit = 1000
	// DefaultTimeout is default max timeout for queries.
	DefaultTimeout = time.Second * 5
)

// Default fills in default values. Safe to call with c == nil.
func (c *Constraint) Default() *Constraint {
	if c == nil {
		return (&Constraint{}).Default()
	}
	if c.Limit == nil {
		c.Limit = ptr.To(DefaultLimit)
	}
	if c.Timeout == nil {
		c.Timeout = ptr.To(DefaultTimeout)
	}
	if c.Start == nil || c.End == nil {
		now := time.Now() // Avoid excessive calls to time.Now() on critical path.
		c.Start = ptr.To(now.Add(-DefaultDuration))
	}
	if c.End == nil {
		c.End = ptr.To(time.Now())
	}
	return c
}

// GetLimit returns limit or 0, safe to call with c == nil
func (c *Constraint) GetLimit() int {
	if c != nil && c.Limit != nil && *c.Limit > 0 {
		return *c.Limit
	}
	return 0
}

func (c *Constraint) GetTimeout() time.Duration {
	if c != nil && c.Timeout != nil && *c.Timeout > 0 {
		return *c.Timeout
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
