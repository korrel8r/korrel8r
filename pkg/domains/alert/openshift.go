package alert

import (
	"context"
	"net/url"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/openshift"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	MonitoringNS     = "openshift-monitoring"
	AlertmanagerMain = "alertmanager-main"
)

// OpenshiftManagerStore creates a store client for the in-cluster OpenShift Alertmanager's route.
func NewOpenshiftAlertManagerStore(ctx context.Context, cfg *rest.Config) (korrel8r.Store, error) {
	c, err := client.New(cfg, client.Options{})
	if err != nil {
		return nil, err
	}

	host, err := openshift.RouteHost(ctx, c, openshift.AlertmanagerMainNSName)
	if err != nil {
		return nil, err
	}

	hc, err := rest.HTTPClientFor(cfg)
	if err != nil {
		return nil, err
	}

	u := &url.URL{
		Scheme: "https",
		Host:   host,
	}

	return NewStore(u, hc)
}
