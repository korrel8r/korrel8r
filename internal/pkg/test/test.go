// package test contains helpers for writing tests
package test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	hasClusterOnce sync.Once
	clusterErr     error

	// These variables are initialized if HasCluster succeeds.
	// Safe for use in tests after calling SkipIfNoCluster.
	RESTConfig *rest.Config
	K8sClient  client.WithWatch
	HTTPClient *http.Client
)

func HasCluster() error {
	// Contact the cluster once per test run, after that assume nothing changes.
	hasClusterOnce.Do(func() {
		if os.Getenv("TEST_NO_CLUSTER") != "" {
			clusterErr = errors.New("TEST_NO_CLUSTER is set")
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		RESTConfig, clusterErr = config.GetConfig()
		if clusterErr != nil {
			return
		}
		RESTConfig.Timeout = time.Second
		RESTConfig.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(100, 1000)
		K8sClient, clusterErr = client.NewWithWatch(RESTConfig, client.Options{})
		if clusterErr != nil {
			return
		}
		HTTPClient, clusterErr = rest.HTTPClientFor(RESTConfig)
		ns := &corev1.Namespace{}
		ns.Name = "default"
		clusterErr = K8sClient.Get(ctx, client.ObjectKeyFromObject(ns), ns)
	})
	return clusterErr
}

// SkipIfNoCluster calls t.Skip if no cluster is detected.
func SkipIfNoCluster(t *testing.T) {
	t.Helper()
	if err := HasCluster(); err != nil {
		skipf(t, "no cluster available: %v", err)
	}
}

// SkipIfNoCommand skips a test if the cmd is not found in PATH
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

// ExecError extracts stderr if err is an exec.ExitError
func ExecError(err error) error {
	if ex, ok := err.(*exec.ExitError); ok {
		return fmt.Errorf("%v: %v", err, string(ex.Stderr))
	}
	return err
}

// CreateUniqueNamespace creates a unique namespace.
func CreateUniqueNamespace(t *testing.T, c client.Client) string {
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
	t.Logf("test namespace: %v", ns.Name)
	t.Cleanup(func() { _ = c.Delete(context.Background(), &ns) })
	return ns.Name
}

// FakeMain temporarily sets os.Args and captures stdout and stderr
// while calling f()
func FakeMain(args []string, f func()) (stdout, stderr string) {
	saveArgs := os.Args
	saveOut, saveErr := os.Stdout, os.Stderr
	defer func() {
		os.Args = saveArgs
		os.Stdout, os.Stderr = saveOut, saveErr
	}()
	os.Args = args
	outBuf, errBuf := &bytes.Buffer{}, &bytes.Buffer{}
	g := sync.WaitGroup{}
	os.Stdout = pump(outBuf, &g)
	os.Stderr = pump(errBuf, &g)
	defer func() {
		_ = os.Stdout.Close()
		_ = os.Stderr.Close()
		g.Wait()
		stdout, stderr = outBuf.String(), errBuf.String()
	}()
	f()
	return "", "" // Return value set in defer
}

func pump(buf *bytes.Buffer, g *sync.WaitGroup) *os.File {
	r, w, err := os.Pipe()
	PanicErr(err)
	g.Add(1)
	go func() { defer g.Done(); _, err = io.Copy(buf, r); PanicErr(err) }()
	return w
}

// PanicErr panics if err is not nil
func PanicErr(err error) {
	if err != nil {
		panic(err)
	}
}

// Must panics if err is not nil, else returns v.
func Must[T any](v T, err error) T { PanicErr(err); return v }

// JSONString returns the JSON marshaled string from v, or the error message if marshal fails
func JSONString(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return err.Error()
	}
	return string(b)
}
