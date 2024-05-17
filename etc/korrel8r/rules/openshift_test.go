// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rules_test

import (
	"testing"

	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	consolev1 "github.com/openshift/api/console/v1"
	"github.com/stretchr/testify/assert"
)

func TestConsolePluginToService(t *testing.T) {
	e := setup()
	cp := k8s.New[consolev1.ConsolePlugin]("", "plugin")
	cp.Spec.Backend.Service = &consolev1.ConsolePluginService{
		Name:      "backendName",
		Namespace: "backendNamespace",
	}
	got, err := apply(e, "ConsolePluginToService", cp)
	assert.NoError(t, err)
	assert.Equal(t, "k8s:Service.v1.:{\"namespace\":\"backendNamespace\",\"name\":\"backendName\"}", got.String())
}
