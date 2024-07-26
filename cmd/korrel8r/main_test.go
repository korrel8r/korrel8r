// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

var tmpDir, korrel8rExe string

// TestMain creates a temp dir and a korrel8r executable for testing.
func TestMain(m *testing.M) {
	var err error
	if tmpDir, err = os.MkdirTemp("", "korrel8r_test"); err != nil {
		panic(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()
	korrel8rExe = filepath.Join(tmpDir, "korrel8r")
	// Build a temporary korrel8r exe, faster than using 'go run' for each test.
	cmd := exec.Command("go", "build", "-cover", "-o", korrel8rExe, ".")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

type testWriter struct{ t *testing.T }

func (w *testWriter) Write(data []byte) (int, error) { w.t.Log(string(data)); return len(data), nil }

// command returns an exec.Cmd to run the korrel8r.test executable in the context of a testing.T test.
func command(t *testing.T, args ...string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(korrel8rExe, append([]string{"--panic"}, args...)...)
	// Redirect stderr to test output.
	cmd.Stderr = &testWriter{t: t}
	return cmd
}
