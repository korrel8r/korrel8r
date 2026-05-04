// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package config

import (
	"encoding/json"
	"errors"
	"time"
)

// Duration is a time.Duration with JSON marshal/unmarshal using [time.ParseDuration] format.
type Duration time.Duration

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch v := v.(type) {
	case float64:
		*d = Duration(v * float64(time.Second))
		return nil
	case string:
		var err error
		td, err := time.ParseDuration(v)
		if err != nil {
			return err
		}
		*d = Duration(td)
		return nil
	default:
		return errors.New("invalid duration")
	}
}
