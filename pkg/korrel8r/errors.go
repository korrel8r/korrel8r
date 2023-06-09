package korrel8r

import "fmt"

type DomainNotFoundErr struct{ Domain string }

func (e DomainNotFoundErr) Error() string { return fmt.Sprintf("domain not found: %v", e.Domain) }

type ClassNotFoundErr struct {
	Class  string
	Domain Domain
}

func (e ClassNotFoundErr) Error() string {
	return fmt.Sprintf("class not found in domain %v: %v", e.Domain, e.Class)
}

type StoreNotFoundErr struct {
	Domain Domain
}

func (e StoreNotFoundErr) Error() string {
	return fmt.Sprintf("no stores found for domain %v", e.Domain)
}
