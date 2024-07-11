// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package korrel8r

import (
	"errors"
	"fmt"
)

type DomainNotFoundError struct{ Domain string }

func (e DomainNotFoundError) Error() string { return fmt.Sprintf("domain not found: %q", e.Domain) }

type ClassNotFoundError struct {
	Class  string
	Domain Domain
}

func (e ClassNotFoundError) Error() string {
	return fmt.Sprintf("class not found in domain %v: %q", e.Domain, e.Class)
}

type StoreNotFoundError struct {
	Domain Domain
}

func (e StoreNotFoundError) Error() string {
	return fmt.Sprintf("no stores found for domain %v", e.Domain)
}

// RuleSkipped is returned from Rule.Apply if the rule has pre-conditions that are not met by the starting object.
// This signals an "expected" skipping of the rule, rather than a template error or returning a bad query.
type RuleSkipped struct{ Rule Rule }

func (e RuleSkipped) Error() string {
	return fmt.Sprintf("rule skipped, not applicable: %v", e.Rule)
}

func IsRuleSkipped(err error) bool {
	return errors.As(err, &RuleSkipped{})
}
