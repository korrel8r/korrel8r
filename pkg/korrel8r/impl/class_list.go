// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package impl

import (
	"sync"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/unique"
)

// ClassList is a goroutine-safe set of classes.
// Used to accumulate classes in domains with a dynamic class set.
type ClassList struct {
	m sync.Mutex
	// New classes are added to the mutable list.
	// The immutable list is returned to callers.
	mutable, immutable *unique.List[korrel8r.Class]
}

func NewClassList(c ...korrel8r.Class) *ClassList {
	return &ClassList{mutable: unique.NewList(c...)}
}

func (cs *ClassList) Append(c ...korrel8r.Class) {
	cs.m.Lock()
	defer cs.m.Unlock()
	cs.mutable.Append(c...)
	// Mutable has changed, immutable is outdated. Clear it.
	cs.immutable = nil
}

func (cs *ClassList) List() []korrel8r.Class {
	cs.m.Lock()
	defer cs.m.Unlock()
	if cs.immutable == nil {
		// mutable has changed, update immutable.
		cs.immutable = unique.NewList(cs.mutable.List...) //  Update
	}
	return cs.immutable.List // No changes, return the same list
}
