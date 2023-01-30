package impl

import (
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/assert"
)

func TestConvert(t *testing.T) {
	d, err := TypeAssert[korrel8r.Domain](mock.Domain("x"))
	assert.NoError(t, err)
	assert.Equal(t, mock.Domain("x"), d)

	_, err = TypeAssert[korrel8r.Query](nil)
	assert.EqualError(t, err, "wrong type: want korrel8r.Query, got (<nil>)(<nil>)")

	_, err = TypeAssert[korrel8r.Query](d)
	assert.EqualError(t, err, "wrong type: want korrel8r.Query, got (mock.Domain)(\"x\")")
}
