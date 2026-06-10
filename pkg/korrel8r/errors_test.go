// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package korrel8r

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassNotFoundError(t *testing.T) {
	err := NewClassNotFoundError("k8s", "Pod")
	assert.EqualError(t, err, "class not found: k8s: Pod")

	var cnf *ClassNotFoundError
	assert.ErrorAs(t, err, &cnf)
	assert.Equal(t, "k8s", cnf.Domain)
	assert.Equal(t, "Pod", cnf.Name)
}

func TestDomainNotFoundError(t *testing.T) {
	err := NewDomainNotFoundError("nosuch")
	assert.EqualError(t, err, "domain not found: nosuch")

	var dnf *DomainNotFoundError
	assert.ErrorAs(t, err, &dnf)
	assert.Equal(t, "nosuch", dnf.Domain)
}
