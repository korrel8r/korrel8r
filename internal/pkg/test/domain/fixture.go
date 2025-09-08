// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package domain

import (
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/domains"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/stretchr/testify/require"
)

const (
	BatchLen     = 10
	MockDataFile = "testdata/domain_test.yaml"
)

// TODO document use of fixture.
type Fixture struct {
	Query        korrel8r.Query        // Query returns BatchLen objects.
	BatchLen     int                   // BatchLen is the length of Query result.
	ClusterSetup func(testing.TB) bool // Set up for cluster part of test. Return false to run only mock tests.
	MockEngine   *engine.Engine        // MockEngine is initialized by [Fixuture.Init()]
}

// Init initializes [Fixture.MockEngine] with a mock store for f.Query().Class().Domain
func (f *Fixture) Init(t testing.TB) {
	if f.MockEngine == nil {
		d := f.Query.Class().Domain()
		var err error
		f.MockEngine, err = engine.Build().
			Domains(d).
			StoreConfigs(config.Store{config.StoreKeyDomain: d.Name(), config.StoreKeyMock: MockDataFile}).
			Engine()
		require.NoError(t, err)
	}
}

// ClusterEngine returns an engine with all known domains backed by an Openshift cluster.
// Used for cluster testing with multiple domains.
func (f *Fixture) ClusterEngine(t testing.TB) *engine.Engine {
	// TODO review test logic for cluster vs. no-cluster tests.
	t.Helper()
	if f.ClusterSetup != nil && !f.ClusterSetup(t) {
		t.Skipf("Skip: domain %v skipping cluster tests", f.Query.Class().Domain())
	}
	out, err := exec.Command("git", "root").Output()
	require.NoError(t, err)
	config := filepath.Join(strings.TrimSpace(string(out)), "etc", "korrel8r", "openshift-route.yaml")
	e, err := engine.Build().
		Domains(domains.All...).
		ConfigFile(config).
		Engine()
	require.NoError(t, err)
	return e
}

func funcName(f any) string {
	strs := strings.Split((runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()), ".")
	return strs[len(strs)-1]
}
