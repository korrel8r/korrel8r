package alert

import (
	"context"

	"github.com/korrel8/korrel8/internal/pkg/openshift"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	MonitoringNS     = "openshift-monitoring"
	AlertmanagerMain = "alertmanager-main"
)

// OpenshiftManagerStore creates a store client for the openshift alert manager.
func NewOpenshiftAlertManagerStore(ctx context.Context, cfg *rest.Config) (korrel8.Store, error) {
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
	return NewStore(host, hc), nil
}
