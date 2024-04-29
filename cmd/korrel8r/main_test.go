// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/korrel8r/korrel8r/internal/pkg/test"
)

// TestMain builds the korrel8r executable for functional testing,
// with support for code coverage measurements.
func TestMain(m *testing.M) {
	// Build korrel8r once to run in tests, much faster than using 'go run' for each test.
	tmpDir = test.Must(os.MkdirTemp("", "korrel8r_test"))
	defer func() { _ = os.RemoveAll(tmpDir) }()
	test.PanicErr(os.MkdirAll(coverDir, 0777))
	cmd := exec.Command("go", "build", "-cover", "-o", tmpDir)
	cmd.Stderr = os.Stderr
	test.PanicErr(cmd.Run())
	os.Exit(m.Run())
}

var tmpDir string

const coverDir = "_covdata" // Not in tmpDir, accumulate results over multiple runs.

type testWriter struct{ t *testing.T }

func (w *testWriter) Write(data []byte) (int, error) { w.t.Log(string(data)); return len(data), nil }

// command returns an exec.Cmd to run the korrel8r executable in the context of a testing.T test.
func command(t *testing.T, args ...string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(filepath.Join(tmpDir, "korrel8r"), args...)
	// Redirect stderr to test output.
	cmd.Stderr = &testWriter{t: t}
	cmd.Env = append(cmd.Env, "GOCOVERDIR="+coverDir)
	return cmd
}
