// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package impl

import (
	"context"
	"errors"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

var _ korrel8r.Store = TryStores{}

// TryStores Get tries each store in turn. Uses the first store to satisfy other Store methods.
type TryStores []korrel8r.Store

func (ps TryStores) Domain() korrel8r.Domain                 { return ps[0].Domain() }
func (ps TryStores) StoreClasses() ([]korrel8r.Class, error) { return ps[0].StoreClasses() }
func (ps TryStores) Get(ctx context.Context, q korrel8r.Query, c *korrel8r.Constraint, a korrel8r.Appender) error {
	var errs error
	for _, s := range ps {
		if err := s.Get(ctx, q, c, a); err == nil {
			return nil
		} else {
			errs = errors.Join(errs, err)
		}
	}
	return errs
}
