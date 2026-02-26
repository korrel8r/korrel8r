// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/korrel8r/korrel8r/pkg/engine"
)

// ConsoleState shared by REST and MCP operations.
// Holds console state reported via REST, makes available to agents via MCP.
// Receives console updates from MCP and makes the available to REST.
type ConsoleState struct {
	Console // Console is actual state reported by console
	Updates chan *Console
	m       sync.Mutex
}

func NewConsoleState() *ConsoleState {
	return &ConsoleState{Updates: make(chan *Console)}
}

// Get a deep copy of the console display state.
func (c *ConsoleState) Get() *Console {
	c.m.Lock()
	defer c.m.Unlock()
	r := Console{}
	_ = DeepCopy(&r, c.Console)
	return &r
}

// Set the console state to a deep copy of the given state.
func (c *ConsoleState) Set(state *Console) {
	c.m.Lock()
	defer c.m.Unlock()
	c.Console = Console{} // Reset existing state
	_ = DeepCopy(&c.Console, state)
}

// Send an update to the console.
// Updates are dropped if they are not handled promptly.
func (c *ConsoleState) Send(update *Console) error {
	select {
	case c.Updates <- update:
		return nil
	case <-time.After(time.Second):
		return errors.New("no console connected, update not sent")
	}
}

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
