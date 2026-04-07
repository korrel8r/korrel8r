// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package session

import (
	"encoding/json"
	"errors"
	"sync"
)

// Console connection between REST and MCP operations.
//
// Holds console state put via REST, makes it available to get_console MCP tool.
// Receives console updates from show_in_console MCP tool, and makes available as SSE events in REST.
//
// State values are JSON-encoded any, so the types can change without updating this code.
type Console struct {
	data    json.RawMessage
	Updates chan json.RawMessage

	closeOnce sync.Once
	m         sync.Mutex
}

func NewConsoleState() *Console {
	return &Console{Updates: make(chan json.RawMessage, 1)}
}

func (c *Console) Close() {
	c.closeOnce.Do(func() {
		close(c.Updates)
	})
}

// Get unmarshals the current console state into dst, which should be a pointer.
// If no state has been set, dst is not modified and nil is returned.
func (c *Console) Get(dst any) error {
	c.m.Lock()
	defer c.m.Unlock()
	if c.data == nil {
		return nil
	}
	return json.Unmarshal(c.data, dst)
}

// Set the console state by marshaling src to JSON.
func (c *Console) Set(state any) error {
	b, err := json.Marshal(state)
	if err != nil {
		return err
	}
	c.m.Lock()
	defer c.m.Unlock()
	c.data = b
	return nil
}

// Send an update to the console.
// Returns error if there is already an update being sent.
func (c *Console) Send(update any) error {
	b, err := json.Marshal(update)
	if err != nil {
		return err
	}
	select {
	case c.Updates <- b:
		return nil
	default:
		return errors.New("console connection is busy, try again")
	}
}
