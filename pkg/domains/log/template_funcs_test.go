// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package log

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogType(t *testing.T) {
	for _, x := range [][]string{
		{"default", "infrastructure"},
		{"openshift", "infrastructure"},
		{"openshift-", "infrastructure"},
		{"openshift-foo", "infrastructure"},
		{"kube", "infrastructure"}, {"kube", "infrastructure"},
		{"kube-", "infrastructure"},
		{"kube-foo", "infrastructure"},
		{"foo", "application"},
		{"foo-kube", "application"},
		{"foo-openshift", "application"},
	} {
		t.Run(x[0], func(t *testing.T) {
			assert.Equal(t, x[1], logTypeForNamespace(x[0]))
		})
	}
}
