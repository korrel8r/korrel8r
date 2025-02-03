// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rules_test

import (
	"testing"

	"github.com/korrel8r/korrel8r/pkg/domains/alert"
	"github.com/korrel8r/korrel8r/pkg/domains/incident"
)

func TestIncident(t *testing.T) {
	for _, x := range []ruleTest{
		{
			rule:  "AlertToIncident",
			start: &alert.Object{Labels: map[string]string{"namespace": "foo", "deployment": "bar"}},
			query: `incident:incident:{"alertLabels":{"deployment":"bar","namespace":"foo"}}`,
		},
		{
			rule: "IncidentToAlert",
			start: &incident.Object{
				Id: "incident-id",
				AlertsLabels: []map[string]string{
					{"namespace": "foo", "deployment": "bar"},
					{"namespace": "foobaz", "deployment": "barbaz"},
				}},
			query: `alert:alert:[{"deployment":"bar","namespace":"foo"},{"deployment":"barbaz","namespace":"foobaz"}]`,
		},
	} {
		x.Run(t)
	}
}
