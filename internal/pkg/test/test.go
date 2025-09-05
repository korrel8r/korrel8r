// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package test contains helpers for writing tests
package test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	klog "sigs.k8s.io/controller-runtime/pkg/log"
)

func init() {
	klog.SetLogger(logging.Log()) // Send controller-runtime logs to our logger.
}

// RequireCluster returns a client or fails the test if there is no connected cluster.
func RequireCluster(t testing.TB) client.Client {
	cfg, err := config.GetConfig()
	require.NoError(t, err)
	c, err := client.New(cfg, client.Options{})
	require.NoError(t, err)
	ns := &corev1.Namespace{}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err = c.Get(ctx, types.NamespacedName{Name: "default"}, ns)
	require.NoError(t, err)
	return c
}

// ListenPort returns a free ephemeral port for listening.
func ListenPort() (int, error) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer func() { _ = l.Close() }()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// ExecError extracts stderr if err is an exec.ExitError
func ExecError(err error) error {
	if ex, ok := err.(*exec.ExitError); ok {
		return fmt.Errorf("%v: %v", err, string(ex.Stderr))
	}
	return err
}

// JSONString returns the simple JSON string for v, or an error message string if marshal fails.
func JSONString(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return err.Error()
	}
	return string(b)
}

// JSONPretty returns the intended JSON string for v, or an error message string if marshal fails.
func JSONPretty(v any) string {
	w := &strings.Builder{}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(v); err != nil {
		return err.Error()
	}
	return w.String()
}

func RandomName(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
