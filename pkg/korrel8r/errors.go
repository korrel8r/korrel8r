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

func IsClassNotFoundError(err error) bool { return IsErrorType[ClassNotFoundError](err) }

type StoreNotFoundError struct {
	Domain Domain
}

func (e StoreNotFoundError) Error() string {
	return fmt.Sprintf("no stores found for domain %v", e.Domain)
}

func IsStoreNotFoundError(err error) bool { return IsErrorType[StoreNotFoundError](err) }

func IsErrorType[T error](err error) bool {
	var target T
	return errors.As(err, &target)
}
