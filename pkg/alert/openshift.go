package alert

import (
	"context"
	"fmt"

	"github.com/korrel8/korrel8/internal/pkg/openshift"
	"github.com/korrel8/korrel8/pkg/korrel8"
	routev1 "github.com/openshift/api/route/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/transport"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	monitoringNS  = "openshift-monitoring"
	thanosService = "thanos-querier"
	thanosRoute   = "thanos-querier"
	alertsPort    = "web"
	prometheusSA  = "prometheus-k8s"
)

func init() {
	runtime.Must(routev1.AddToScheme(scheme.Scheme))
}

// NewOpenshiftStore creates a store client for the openshift alerts endpoint.
func NewOpenshiftStore(cfg *rest.Config) (korrel8.Store, error) {
	c, err := client.New(cfg, client.Options{})
	if err != nil {
		return nil, err
	}
	nsName := client.ObjectKey{Name: thanosService, Namespace: monitoringNS}
	host, err := openshift.RouteHost(context.Background(), c, nsName)
	if err != nil {
		return nil, fmt.Errorf("failed to locate the alerts endpoint: %w", err)
	}

	// We need to provide a RoundTripper that authenticates with a bearer token
	// to talk to the alerts endpoint.
	trConfig, err := cfg.TransportConfig()
	if err != nil {
		return nil, err
	}

	if trConfig.BearerToken == "" && trConfig.BearerTokenFile == "" {
		kclient, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
		}

		tr, err := kclient.CoreV1().ServiceAccounts(monitoringNS).CreateToken(context.Background(), prometheusSA, &authenticationv1.TokenRequest{}, metav1.CreateOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to obtain token: %w", err)
		}

		trConfig.BearerToken = tr.Status.Token
	}

	rt, err := transport.New(trConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create transport: %w", err)
	}

	return NewStore(host, rt)
}
