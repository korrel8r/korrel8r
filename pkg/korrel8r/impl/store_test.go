// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package impl

import (
	"context"
	"testing"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
)

type testStore struct{ *Store }

func (testStore) Get(context.Context, korrel8r.Query, *korrel8r.Constraint, korrel8r.Appender) error {
	return nil
}

// verify that example implements korrel8r.Domain
var _ korrel8r.Store = testStore{}

func TestStore(t *testing.T) {
	d := testDomain{NewDomain("foo", "mystery domain", testClass("a"), testClass("b"))}
	testDomainSingleton = d
	s := testStore{NewStore(d)}
	assert.Equal(t, d, s.Domain())
	cs, err := s.StoreClasses()
	assert.NoError(t, err)
	assert.Equal(t, []korrel8r.Class{testClass("a"), testClass("b")}, cs)
}
