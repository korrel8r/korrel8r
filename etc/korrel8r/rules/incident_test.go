// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rules_test

import (
	"testing"

	"github.com/korrel8r/korrel8r/pkg/domains/alert"
	"github.com/korrel8r/korrel8r/pkg/domains/incident"
	"github.com/stretchr/testify/assert"
)

func TestAlertToIncident(t *testing.T) {
	e := setup()
	t.Run("AlertToIncident", func(t *testing.T) {
		tested("AlertToIncident")
		got, err := e.Rule("AlertToIncident").Apply(
			&alert.Object{Labels: map[string]string{"namespace": "foo", "deployment": "bar"}})
		assert.NoError(t, err)
		assert.Equal(t, `incident:incident:{"alertLabels":{"deployment":"bar","namespace":"foo"}}`, got.String())
	})
}

func TestIncidentToAlert(t *testing.T) {
	e := setup()
	t.Run("IncidentToAlert", func(t *testing.T) {
		tested("IncidentToAlert")
		got, err := e.Rule("IncidentToAlert").Apply(
			&incident.Object{Id: "incident-id", AlertsLabels: []map[string]string{
				{"namespace": "foo", "deployment": "bar"},
				{"namespace": "foobaz", "deployment": "barbaz"},
			}})
		assert.NoError(t, err)
		assert.Equal(t, `alert:alert:[{"deployment":"bar","namespace":"foo"},{"deployment":"barbaz","namespace":"foobaz"}]}`, got.String())
	})
}
