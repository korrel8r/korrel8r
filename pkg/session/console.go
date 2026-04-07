// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package session

import (
	"encoding/json"
	"errors"
	"sync"
	"time"
)

// ConsoleState shared by REST and MCP operations.
// Holds console state reported via REST, makes available to agents via MCP.
// Receives console updates from MCP and makes the available to REST.
//
// State is stored as JSON internally, so Get/Set/Send perform
// automatic deep copies via JSON marshal/unmarshal.
type ConsoleState struct {
	data    json.RawMessage
	Updates chan json.RawMessage
	m       sync.Mutex
}

func NewConsoleState() *ConsoleState {
	return &ConsoleState{Updates: make(chan json.RawMessage)}
}

// Get unmarshals the current console state into dst, which should be a pointer.
// If no state has been set, dst is not modified and nil is returned.
func (c *ConsoleState) Get(dst any) error {
	c.m.Lock()
	defer c.m.Unlock()
	if c.data == nil {
		return nil
	}
	return json.Unmarshal(c.data, dst)
}

// Set the console state by marshaling src to JSON.
func (c *ConsoleState) Set(state any) error {
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
// Updates are dropped if they are not handled promptly.
func (c *ConsoleState) Send(update any) error {
	b, err := json.Marshal(update)
	if err != nil {
		return err
	}
	select {
	case c.Updates <- b:
		return nil
	case <-time.After(time.Second):
		return errors.New("no console connected, update not sent")
	}
}
