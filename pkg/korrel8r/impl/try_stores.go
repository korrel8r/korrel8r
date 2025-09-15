// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package impl

import (
	"context"
	"errors"

	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

var _ korrel8r.Store = TryStores{}

// TryStores Get tries each store in turn. Uses the first store to satisfy other Store methods.
type TryStores []korrel8r.Store

func (ts TryStores) Domain() korrel8r.Domain { return ts[0].Domain() }

func (ts TryStores) Get(ctx context.Context, q korrel8r.Query, c *korrel8r.Constraint, a korrel8r.Appender) error {
	var errs error
	for i, s := range ts {
		if err := s.Get(ctx, q, c, a); err == nil {
			return nil
		} else {
			log.V(5).Info("try-stores store error", "store", i, "remaining", len(ts)-i, "error", err)
			errs = errors.Join(errs, err)
		}
	}
	return errs
}

var log = logging.Log()
