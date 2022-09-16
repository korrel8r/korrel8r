// package test contains helpers for writing tests
package test

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	hasCluster     error
	hasClusterOnce sync.Once
)

func HasCluster() error {
	// Contact the cluster once per test run, after that assume nothing changes.
	hasClusterOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		var err error
		defer func() {
			hasCluster = err
			cancel()
		}()
		cfg, err := config.GetConfig()
		if err != nil {
			return
		}
		cfg.Timeout = time.Second
		c, err := client.New(cfg, client.Options{})
		if err != nil {
			return
		}
		ns := corev1.Namespace{}
		ns.Name = "default"
		err = c.Get(ctx, types.NamespacedName{Name: "default"}, &ns)
	})
	return hasCluster
}

// SkipIfNoCluster calls t.Skip if no cluster is detected.
func SkipIfNoCluster(t *testing.T) {
	t.Helper()
	if err := HasCluster(); err != nil {
		skipf(t, "no cluster running: %v", err)
	}
}

func SkipIfNoCommand(t *testing.T, cmd string) {
	t.Helper()
	if _, err := exec.LookPath(cmd); err != nil {
		skipf(t, "command %q not available", cmd)
	}
}

func skipf(t *testing.T, format string, args ...interface{}) {
	t.Helper()
	if os.Getenv("TEST_NO_SKIP") != "" {
		t.Fatalf(format, args...)
	} else {
		t.Skipf(format, args...)
	}
}

// ListenPort returns a free ephemeral port for listening.
func ListenPort() (int, error) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func ExecError(err error) error {
	if ex, ok := err.(*exec.ExitError); ok {
		return fmt.Errorf("%v: %v", err, string(ex.Stderr))
	}
	return err
}
