// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package metric

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQuery_Selectors(t *testing.T) {
	for _, x := range []struct {
		query Query
		want  []string
	}{
		{`{namespace="foo"}`, []string{`{namespace="foo"}`}},
		{`blah{namespace="foo"}`, []string{`blah{namespace="foo"}`}},
		{`{__name__="fred",namespace="foo"}`, []string{`{__name__="fred",namespace="foo"}`}},
		{`count(fred{namespace="foo"}) + count(barney{name="bar"})`, []string{`fred{namespace="foo"}`, `barney{name="bar"}`}},
		{`sum by(namespace, app) (rate(received_bytes_total{component_kind="source", component_type!="internal_metrics"}[5m]))`,
			[]string{"received_bytes_total{component_kind=\"source\",component_type!=\"internal_metrics\"}"}},
	} {
		t.Run(x.query.String(), func(t *testing.T) {
			got, err := x.query.Selectors()
			if assert.NoError(t, err) {
				assert.Equal(t, x.want, got)
			}
		})
	}
}
