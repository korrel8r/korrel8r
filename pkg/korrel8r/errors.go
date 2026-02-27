// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package korrel8r

import "fmt"

type ClassNotFoundError struct {
	Domain, Name string
}

func (e *ClassNotFoundError) Error() string {
	return fmt.Sprintf("class not found: %v: %v", e.Domain, e.Name)
}

func NewClassNotFoundError(domain, name string) error {
	return &ClassNotFoundError{Domain: domain, Name: name}
}

type DomainNotFoundError struct {
	Domain string
}

func (e *DomainNotFoundError) Error() string {
	return fmt.Sprintf("domain not found: %v", e.Domain)
}

func NewDomainNotFoundError(domain string) error {
	return &DomainNotFoundError{Domain: domain}
}
