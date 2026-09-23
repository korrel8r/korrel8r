// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package config

import (
	"encoding/json"
	"fmt"
	"strconv"

	internaljson "github.com/korrel8r/korrel8r/internal/pkg/json"
	"k8s.io/apimachinery/pkg/api/resource"
)

// Bytes is a byte count encoded as a Kubernetes quantity such as "512Mi" or "2Gi".
type Bytes int64

func (b Bytes) MarshalJSON() ([]byte, error) {
	return internaljson.Marshal(resource.NewQuantity(int64(b), resource.BinarySI).String())
}

func (b *Bytes) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] == '"' {
		var value string
		if err := json.Unmarshal(data, &value); err != nil {
			return err
		}
		quantity, err := resource.ParseQuantity(value)
		if err != nil {
			return fmt.Errorf("invalid byte quantity: %w", err)
		}
		n, ok := quantity.AsInt64()
		if !ok {
			return fmt.Errorf("invalid byte quantity: %v", value)
		}
		*b = Bytes(n)
		return nil
	}
	var value json.Number
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("invalid byte quantity: %w", err)
	}
	n, err := strconv.ParseInt(string(value), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid byte quantity: %w", err)
	}
	*b = Bytes(n)
	return nil
}
