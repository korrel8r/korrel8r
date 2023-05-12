// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package cmd

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// korrel8r as an exec.Cmd
func korrel8r(args ...string) *exec.Cmd {
	return exec.Command("go", append([]string{"run", "./korrel8r"}, args...)...)
}

func Test_List(t *testing.T) {
	// Test list commands that don't need a cluster.
	for _, x := range []struct {
		args, want []string
	}{
		{args: []string{"list"}, want: []string{"k8s", "alert", "logs", "metric"}},
		{args: []string{"list", "logs"}, want: []string{"application", "infrastructure", "audit"}},
	} {
		t.Run(fmt.Sprint(x.args), func(t *testing.T) {
			out, err := korrel8r(x.args...).Output()
			if assert.NoError(t, err) {
				assert.ElementsMatch(t, x.want, strings.Split(strings.TrimSpace(string(out)), "\n"))
			}
		})
	}
}

func Test_Web(t *testing.T) {
	// Just verify we get a web page, doesn't do any detailed testing.
	port, err := test.ListenPort()
	require.NoError(t, err)
	cmd := korrel8r("web", "--http", net.JoinHostPort("", strconv.Itoa(port)))
	require.NoError(t, cmd.Start())
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()

	u := fmt.Sprintf("http://localhost:%v/", port)
	var resp *http.Response
	require.Eventually(t, func() bool {
		resp, err = http.Get(u)
		return err == nil
	}, 10*time.Second, time.Second, "Time out: %v: %v", cmd, err)
	b, err := io.ReadAll(resp.Body)
	assert.Contains(t, string(b), "Korrel8r")
}
