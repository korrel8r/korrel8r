package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// LokiServer is a single-process Loki instance running in a container.
type LokiServer struct {
	Port        int
	ContainerID string
}

const (
	lokiImage = "docker.io/grafana/loki:2.5.0"
)

var (
	pullLokiOnce sync.Once
)

func NewLokiServer() (server *LokiServer, err error) {
	defer func() {
		if exitErr, ok := err.(*exec.ExitError); ok {
			err = fmt.Errorf("loki server exit: %w\n%v", exitErr, string(exitErr.Stderr))
		}
	}()
	pullLokiOnce.Do(func() { err = exec.Command("podman", "pull", lokiImage).Run() })
	if err != nil {
		return nil, fmt.Errorf("pull %v failed: %w", lokiImage, err)
	}

	server = &LokiServer{}
	server.Port, err = ListenPort()
	if err != nil {
		return nil, err
	}

	cmd := exec.Command("podman", "run", "-d", "-p", fmt.Sprintf("%v:3100", server.Port), lokiImage)
	cmd.Stderr = os.Stderr
	id, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("run %v failed: %w", cmd, err)
	}
	server.ContainerID = strings.TrimSpace(string(id))
	return server, nil
}

func (s *LokiServer) URL() *url.URL {
	return &url.URL{Scheme: "http", Host: fmt.Sprintf("127.0.0.1:%v", s.Port)}
}

func (s *LokiServer) Close() error {
	cmd := exec.Command("podman", "kill", s.ContainerID)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (s *LokiServer) Push(labels map[string]string, lines ...string) error {
	u := s.URL()
	u.Path = "/loki/api/v1/push"
	b, err := json.Marshal(map[string][]streamValues{"streams": {{Stream: labels, Values: makeValues(lines)}}})
	if err != nil {
		return err
	}
	resp, err := http.Post(u.String(), "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func RequireLokiServer(t *testing.T) *LokiServer {
	t.Helper()
	s, err := NewLokiServer()
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })
	require.Eventually(t, func() bool {
		u := s.URL()
		u.Path = "/ready"
		resp, err := http.Get(u.String())
		t.Logf("waiting for Loki on port %v, retry: %v: %v", s.Port, resp.Status, err)
		if err == nil {
			resp.Body.Close()
		}
		return err == nil && resp.StatusCode/100 == 2
	}, time.Minute, time.Second, "timed out waiting for Loki: %v", s.URL())
	return s
}

func makeValues(lines []string) (values [][]string) {
	t := time.Now()
	for _, line := range lines {
		values = append(values, []string{fmt.Sprintf("%v", t.UnixNano()), line})
	}
	return values
}

// streamValues is a set of log values ["time", "line"] for a log stream.
type streamValues struct {
	Stream map[string]string `json:"stream"` // Labels for the stream
	Values [][]string        `json:"values"`
}
