// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package unique

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrors(t *testing.T) {
	var errs Errors
	assert.Nil(t, errs.Err())
	errs.Add(errors.New("one"))
	assert.EqualError(t, errs.Err(), "one")
	errs.Add(errors.New("two"))
	assert.EqualError(t, errs.Err(), "one\ntwo")

}
