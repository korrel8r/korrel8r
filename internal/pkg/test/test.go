// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package test contains helpers for writing tests
package test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	hasClusterOnce sync.Once
	clusterErr     error

	// These variables are initialized if HasCluster succeeds.
	RESTConfig *rest.Config
	K8sClient  client.WithWatch
	HTTPClient *http.Client
)

func HasCluster() error {
	// Contact the cluster once per test run, after that assume nothing changes.
	hasClusterOnce.Do(func() {
		log.SetLogger(logging.Log())
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		RESTConfig, clusterErr = config.GetConfig()
		if clusterErr != nil {
			return
		}
		RESTConfig.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(100, 1000)
		K8sClient, clusterErr = client.NewWithWatch(RESTConfig, client.Options{})
		if clusterErr != nil {
			return
		}
		HTTPClient, clusterErr = rest.HTTPClientFor(RESTConfig)
		ns := &corev1.Namespace{}
		clusterErr = K8sClient.Get(ctx, types.NamespacedName{Name: "default"}, ns)
	})
	return clusterErr
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

// ExecError extracts stderr if err is an exec.ExitError
func ExecError(err error) error {
	if ex, ok := err.(*exec.ExitError); ok {
		return fmt.Errorf("%v: %v", err, string(ex.Stderr))
	}
	return err
}

// TempNamespace creates a unique namespace.
func TempNamespace(t *testing.T, c client.Client) string {
	t.Helper()
	// Server-generated unique name.
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
			Labels:       map[string]string{"test": t.Name()},
		},
	}
	require.NoError(t, c.Create(context.Background(), &ns))
	require.NotEmpty(t, ns.Name)
	t.Cleanup(func() {
		t.Helper()
		const env = "KORREL8R_TEST_KEEP_NS"
		if t.Failed() && os.Getenv(env) != "" {
			t.Logf("keeping test namespace: %v", ns.Name)
		} else {
			t.Logf("deleting test namespace: %v - To keep test namespace set environment %v=1", ns.Name, env)
			_ = c.Delete(context.Background(), &ns)
		}
	})
	return ns.Name
}

// Watch in a loop, call f for each event, return when f returns true.
// Fatal if watch closes or times out
func Watch(t *testing.T, w watch.Interface, timeout time.Duration, f func(e watch.Event) (finished bool)) {
	t.Helper()
	defer w.Stop()
	for {
		select {
		case e, ok := <-w.ResultChan():
			if !ok {
				t.Fatal("watch closed")
			}
			if f(e) {
				return
			}
		case <-time.After(timeout):
			t.Fatal("timeout in watch")
		}
	}
}
