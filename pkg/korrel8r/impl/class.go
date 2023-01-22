package impl

import (
	"fmt"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
)

// GetClass calls domain.Class(name) and returns a "not found" error if the value is nil.
func GetClass(domain korrel8r.Domain, name string) (korrel8r.Class, error) {
	if v := domain.Class(name); v != nil {
		return v, nil
	}
	return nil, fmt.Errorf("class not found: %v/%v", domain, name)
}
