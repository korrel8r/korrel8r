// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package korrel8r

import (
	"errors"
	"fmt"
)

type DomainNotFoundError string

func (err DomainNotFoundError) Error() string {
	return fmt.Sprintf("domain not found: %v", string(err))
}

func IsDomainNotFoundError(err error) bool { return IsErrorType[DomainNotFoundError](err) }

type ClassNotFoundError string

func (err ClassNotFoundError) Error() string {
	return fmt.Sprintf("class not found: %v", string(err))
}

func IsClassNotFoundError(err error) bool { return IsErrorType[ClassNotFoundError](err) }

func IsErrorType[T error](err error) bool {
	var target T
	return errors.As(err, &target)
}
