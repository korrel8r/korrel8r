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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	hasCluster     bool
	hasClusterOnce sync.Once
)

func HasCluster() bool {
	hasClusterOnce.Do(func() {
		cfg, err := config.GetConfig()
		if err != nil {
			return
		}
		c, err := client.New(cfg, client.Options{})
		if err != nil {
			return
		}
		ns := corev1.Namespace{}
		ns.Name = "default"
		err = c.Get(context.Background(), types.NamespacedName{Name: "default"}, &ns)
		hasCluster = (err == nil)
	})
	return hasCluster
}

// SkipIfNoCluster calls t.Skip if no cluster is detected.
func SkipIfNoCluster(t *testing.T) {
	t.Helper()
	if !HasCluster() {
		skipf(t, "no cluster running")
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
		t.Fatalf("TEST_NO_SKIP: %v", fmt.Sprintf(format, args...))
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

func ExitError(err error) error {
	if ex, ok := err.(*exec.ExitError); ok {
		return fmt.Errorf("%v: %v", err, string(ex.Stderr))
	}
	return err
}
