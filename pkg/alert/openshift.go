package alert

import (
	"context"

	"github.com/alanconway/korrel8/pkg/korrel8"
	routev1 "github.com/openshift/api/route/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	MonitoringNS     = "openshift-monitoring"
	AlertmanagerMain = "alertmanager-main"
)

func init() {
	runtime.Must(routev1.AddToScheme(scheme.Scheme))
}

// OpenshiftManagerHost finds the main alert-manager host in an openshift cluster.
func OpenshiftManagerHost(c client.Client) (string, error) {
	r := routev1.Route{}
	nsName := client.ObjectKey{Name: AlertmanagerMain, Namespace: MonitoringNS}
	if err := c.Get(context.Background(), nsName, &r); err != nil {
		return "", err
	}
	return r.Spec.Host, nil
}

// OpenshiftManagerStore creates a store client for the openshift alert manager.
func OpenshiftManagerStore(cfg *rest.Config) (korrel8.Store, error) {
	c, err := client.New(cfg, client.Options{})
	if err != nil {
		return nil, err
	}
	host, err := OpenshiftManagerHost(c)
	if err != nil {
		return nil, err
	}
	hc, err := rest.HTTPClientFor(cfg)
	if err != nil {
		return nil, err
	}
	return NewStore(host, hc), nil
}
