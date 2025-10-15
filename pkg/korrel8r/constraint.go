// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package korrel8r

import (
	"context"
	"encoding/json"
	"time"

	"github.com/korrel8r/korrel8r/pkg/ptr"
)

// Constraint included in a store Get operation to restrict the resulting objects.
type Constraint struct {
	Limit   *int           `json:"limit,omitempty"`   // Limit number of objects returned per query.
	Timeout *time.Duration `json:"timeout,omitempty"` // Timeout per request, h/m/s/ms/ns format
	Start   *time.Time     `json:"start,omitempty"`   // Start of time interval (RFC 3339).
	End     *time.Time     `json:"end,omitempty"`     // End of time interval (RFC 3339).
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

// Default values can be modified in init() or main(), but not after korrel8r functions are called.
var (
	// DefaultDuration is the global default duration for query constraints.
	DefaultDuration = time.Hour
	// DefaultLimit is the global default max items limit for query constraints.
	DefaultLimit = 1000
	// DefaultTimeout is default max timeout for requests and queries.
	DefaultTimeout = time.Second * 10
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
	if c.End == nil {
		c.End = ptr.To(time.Now())
	}
	if c.Start == nil {
		c.Start = ptr.To(c.End.Add(-DefaultDuration))
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

func (c *Constraint) GetTimeout() time.Duration {
	if c != nil && c.Timeout != nil {
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

type constraintKey struct{}

// WithConstraint attaches a constraint to a context.
func WithConstraint(ctx context.Context, c *Constraint) (context.Context, context.CancelFunc) {
	if c != nil {
		ctx = context.WithValue(ctx, constraintKey{}, c) // Attach the constraint
		if c.GetTimeout() > 0 {                          // Add the timeout if defined.
			return context.WithTimeout(ctx, c.GetTimeout())
		}
	}
	return ctx, func() {}
}

// ConstraintFrom returns the constraint attached to the context.
// Returns nil if there is none.
func ConstraintFrom(ctx context.Context) *Constraint {
	c, _ := ctx.Value(constraintKey{}).(*Constraint)
	return c
}
