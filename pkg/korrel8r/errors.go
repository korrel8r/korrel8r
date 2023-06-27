// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package korrel8r

import "fmt"

type DomainNotFoundErr struct{ Domain string }

func (e DomainNotFoundErr) Error() string { return fmt.Sprintf("domain not found: %q", e.Domain) }

type ClassNotFoundErr struct {
	Class  string
	Domain Domain
}

func (e ClassNotFoundErr) Error() string {
	return fmt.Sprintf("class not found in domain %v: %q", e.Domain, e.Class)
}

type StoreNotFoundErr struct {
	Domain Domain
}

func (e StoreNotFoundErr) Error() string {
	return fmt.Sprintf("no stores found for domain %v", e.Domain)
}
