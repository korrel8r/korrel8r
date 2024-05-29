// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package test

import (
	"os"
	"os/exec"
	"sync"
	"testing"

	"github.com/korrel8r/korrel8r/client/pkg/swagger/client/operations"
	"github.com/korrel8r/korrel8r/internal/pkg/test"
	"github.com/stretchr/testify/require"
	yaml "sigs.k8s.io/yaml/goyaml.v2"
)

var (
	korrel8rcli = getenv("KORREL8RCLI", "korrel8rcli")
	namespace   = getenv("NAMESPACE", "korrel8r")
	name        = "korrel8r"
)

func TestKorrel8rcli_domains(t *testing.T) {
	b, err := exec.Command(korrel8rcli, "domains", routeURL(t)).CombinedOutput()
	require.NoError(t, err, string(b))
	var ok operations.GetDomainsOK
	require.NoError(t, yaml.Unmarshal(b, &ok.Payload))
	var names []string
	for _, d := range ok.Payload {
		names = append(names, d.Name)
	}
	require.ElementsMatch(t, []string{"k8s", "alert", "log", "metric", "netflow", "mock"}, names)
}

var (
	routeURLValue string
	deployOnce    sync.Once
)

func routeURL(t *testing.T) string {
	t.Helper()
	test.SkipIfNoCluster(t)
	deployOnce.Do(func() {
		b, err := exec.Command("make", "-C..", "deploy").CombinedOutput()
		require.NoError(t, err, string(b))
		b, err = exec.Command("oc", "get", "-n", namespace, "route/"+name, "-otemplate=https://{{.spec.host}}").CombinedOutput()
		require.NoError(t, err, string(b))
		routeURLValue = string(b)
	})
	return routeURLValue
}

func getenv(name, value string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return value
}
