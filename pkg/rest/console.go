// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rest

import (
	"encoding/json"
	"fmt"

	"github.com/korrel8r/korrel8r/pkg/engine"
)

// DeepCopy does a deep copy via JSON marshal.
func DeepCopy(dst, src any) error {
	bytes, err := json.Marshal(src)
	if err != nil {
		return err
	}
	err = json.Unmarshal(bytes, dst)
	return err
}

// ConsoleOK validates and normalizes received console state.
func ConsoleOK(e *engine.Engine, c *Console) error {
	if c.View != "" {
		q, err := e.Query(c.View)
		if err != nil {
			return err
		}
		// Normalize query strings.
		c.View = q.String()
	}
	if c.Search != nil {
		var start *Start
		switch {
		case c.Search.Goals != nil && c.Search.Neighbors == nil:
			start = &c.Search.Goals.Start
			if _, err := e.Classes(c.Search.Goals.Goals); err != nil {
				return err
			}
		case c.Search.Neighbors != nil && c.Search.Goals == nil:
			start = &c.Search.Neighbors.Start
		default:
			return fmt.Errorf("search must have exactly one of .goals or .neighbors")
		}
		if start != nil {
			if _, err := TraverseStart(e, *start); err != nil {
				return err
			}
		}
	}
	return nil
}
