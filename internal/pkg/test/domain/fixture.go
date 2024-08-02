// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// Package domain provides unit tests and benchmarks to be run against all domains.
// The domain  must have a mock store in "testdata/domain_test.yaml" containing
// and provide a query that returns BatchLen objects.
// It should have test functions TestDomain and BenchmarkDomain that call [domain.Test] and [domain.Benchmark].
package domain

import (
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/domains/alert"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/domains/log"
	"github.com/korrel8r/korrel8r/pkg/domains/metric"
	"github.com/korrel8r/korrel8r/pkg/domains/netflow"
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
	Query         korrel8r.Query
	SkipOpenshift bool
	MockEngine    *engine.Engine
}

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

func (f *Fixture) OpenshiftEngine(t testing.TB) *engine.Engine {
	if f.SkipOpenshift {
		t.Skipf("Skip: domain %v skipping openshift tests", f.Query.Class().Domain())
	}
	out, err := exec.Command("git", "root").Output()
	require.NoError(t, err)
	config := filepath.Join(strings.TrimSpace(string(out)), "etc", "korrel8r", "openshift-route.yaml")
	e, err := engine.Build().
		Domains(k8s.Domain, log.Domain, netflow.Domain, alert.Domain, metric.Domain).
		ConfigFile(config).
		Engine()
	require.NoError(t, err)
	return e
}

func funcName(f any) string {
	strs := strings.Split((runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()), ".")
	return strs[len(strs)-1]
}
