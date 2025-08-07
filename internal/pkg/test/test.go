// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package test contains helpers for writing tests
package test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	clusterOnce sync.Once
	errCluster  = errors.New("cluster status unknown")
)

// SkipIfNoCluster call t.Skip() if not logged in to a cluster.
// Returns a client.Client if the cluster is available.
func SkipIfNoCluster(t testing.TB) client.Client {
	var c client.Client
	clusterOnce.Do(func() {
		log.SetLogger(logging.Log())
		var cfg *rest.Config
		cfg, errCluster = config.GetConfig()
		if errCluster != nil {
			return
		}
		c, errCluster = client.New(cfg, client.Options{})
		if errCluster != nil {
			return
		}
		ns := &corev1.Namespace{}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		errCluster = c.Get(ctx, types.NamespacedName{Name: "default"}, ns)
	})
	if errCluster != nil {
		t.Skipf("Skipping: no cluster: %v", errCluster)
	}
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
