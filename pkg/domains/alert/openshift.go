// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package alert

import (
	"context"
	"net/url"

	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/openshift"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// OpenshifStore creates a store client for the in-cluster OpenShift monitoring stack.
func NewOpenshiftStore(ctx context.Context, cfg *rest.Config) (korrel8r.Store, error) {
	c, err := client.New(cfg, client.Options{})
	if err != nil {
		return nil, err
	}

	alertmanagerHost, err := openshift.RouteHost(ctx, c, openshift.AlertmanagerMainNSName)
	if err != nil {
		return nil, err
	}

	prometheusHost, err := openshift.RouteHost(ctx, c, openshift.ThanosQuerierNSName)
	if err != nil {
		return nil, err
	}

	hc, err := rest.HTTPClientFor(cfg)
	if err != nil {
		return nil, err
	}

	return NewStore(
		&url.URL{Scheme: "https", Host: alertmanagerHost},
		&url.URL{Scheme: "https", Host: prometheusHost},
		hc,
	)
}
