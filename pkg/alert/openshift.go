package alert

import (
	"context"
	"fmt"

	"github.com/korrel8/korrel8/pkg/korrel8"
	routev1 "github.com/openshift/api/route/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
	v1 "k8s.io/api/core/v1"
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

// openshiftHostFromRoute returns the OpenShift alerts endpoint accessible from the outside of the cluster.
func openshiftHostFromRoute(c client.Client) (string, error) {
	r := routev1.Route{}
	nsName := client.ObjectKey{Name: thanosService, Namespace: monitoringNS}
	if err := c.Get(context.Background(), nsName, &r); err != nil {
		return "", fmt.Errorf("failed to get route %s: %w", nsName.String(), err)
	}

	return r.Spec.Host, nil
}

// openshiftHostFromInClusterService returns the OpenShift alerts endpoint accessible from within the cluster.
func openshiftHostFromInClusterService(c client.Client) (string, error) {
	svc := v1.Service{}
	nsName := client.ObjectKey{Name: thanosService, Namespace: monitoringNS}
	if err := c.Get(context.Background(), nsName, &svc); err != nil {
		return "", fmt.Errorf("failed to get service %s: %w", nsName.String(), err)
	}

	for _, port := range svc.Spec.Ports {
		if port.Name == alertsPort {
			return fmt.Sprintf("%s.%s:%d", svc.Name, svc.Namespace, port.Port), nil
		}
	}

	return "", fmt.Errorf("failed to find port %q in service %s", alertsPort, nsName.String())
}

// NewStore creates a store client for the openshift alerts endpoint.
func NewStore(cfg *rest.Config) (korrel8.Store, error) {
	c, err := client.New(cfg, client.Options{})
	if err != nil {
		return nil, err
	}

	host, err := openshiftHostFromRoute(c)
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

	return newAlertStore(host, rt)
}
