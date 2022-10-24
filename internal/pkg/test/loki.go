package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"strconv"
	"sync"
	"testing"
	"time"

	"net"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

// LokiServer is a single-process Loki instance running in a container.
type LokiServer struct {
	Port int
	Cmd  *exec.Cmd
}

const (
	lokiImage = "grafana/loki:2.5.0"
)

var (
	pullLokiOnce sync.Once
)

func NewLokiServer() (server *LokiServer, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("error starting loki server: %w", err)
		}
	}()
	pullLokiOnce.Do(func() {
		if _, err = exec.LookPath("podman"); err != nil {
			return
		}
		if err = exec.Command("podman", "pull", lokiImage).Run(); err != nil {
			return
		}
	})
	if err != nil {
		return nil, err
	}

	server = &LokiServer{}
	server.Port, err = ListenPort()
	if err != nil {
		return nil, err
	}
	server.Cmd = exec.Command("podman", "run", "-p", fmt.Sprintf("%v:3100", server.Port), lokiImage)
	if err := server.Cmd.Start(); err != nil {
		return nil, err
	}
	return server, nil
}

func (s *LokiServer) URL() *url.URL {
	return &url.URL{Scheme: "http", Host: net.JoinHostPort("localhost", strconv.Itoa(s.Port)), Path: "/loki/api/v1"}
}

// Check if server is ready.
func (s *LokiServer) Check(p func(format string, args ...any)) error {
	u := s.URL().String() + "/query_range?direction=FORWARD&limit=30&query=%7Bx%3D%22y%22%7D"
	resp, err := httpError(http.Get(u))
	if err != nil {
		return err
	}
	resp.Body.Close()
	return s.Push(map[string]string{"x": "y"}, "z")
}

func RequireLokiServer(t *testing.T) *LokiServer {
	t.Helper()
	s, err := NewLokiServer()
	require.NoError(t, err)
	defer func() { _ = s.Cmd.Process.Kill() }()
	require.Eventually(t, func() bool {
		if err := s.Check(t.Logf); err != nil {
			t.Logf("waiting for Loki on port %v (%v)", s.Port, err)
			return false
		}
		return true
	}, time.Minute, time.Second/2)
	return s
}

func makeValues(lines []string) (values [][]string) {
	t := time.Now()
	for _, line := range lines {
		values = append(values, []string{fmt.Sprintf("%v", t.UnixNano()), line})
	}
	return values
}

func (s *LokiServer) Push(labels map[string]string, lines ...string) error {
	u := fmt.Sprintf("http://localhost:%v/loki/api/v1/push", s.Port)
	b, err := json.Marshal(map[string][]streamValues{"streams": {{Stream: labels, Values: makeValues(lines)}}})
	if err != nil {
		return fmt.Errorf("post %q: %w", u, err)
	}
	resp, err := httpError(http.Post(u, "application/json", bytes.NewReader(b)))
	if err != nil {
		return fmt.Errorf("post %q: %w", u, err)
	}
	resp.Body.Close()
	return nil
}

// streamValues is a set of log values ["time", "line"] for a log stream.
type streamValues struct {
	Stream map[string]string `json:"stream"` // Labels for the stream
	Values [][]string        `json:"values"`
}

// httpError returns err if not nil, an error constructed from resp.Body if resp.Status is not 2xx, nil otherwise.
func httpError(resp *http.Response, err error) (*http.Response, error) {
	if err != nil {
		return resp, err
	}
	if resp.Status[0] == '2' {
		return resp, nil
	}
	node, _ := html.Parse(resp.Body)
	defer resp.Body.Close()
	return resp, fmt.Errorf("%v: %v", resp.Status, node.Data)
}
