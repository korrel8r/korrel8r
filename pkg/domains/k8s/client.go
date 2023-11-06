// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package k8s

import (
	"time"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// NewClient provides a general-purpose k8s client.
// It may be used by other domains that need to talk to the cluster.
func NewClient() (client.Client, *rest.Config, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, nil, err
	}
	// TODO configurable settings for k8s client.
	// Reduce client-side throttling for rapid results.
	cfg.QPS = 100
	cfg.Burst = 1000
	cfg.Timeout = 5 * time.Second
	httpClient, err := rest.HTTPClientFor(cfg)
	if err != nil {
		return nil, nil, err
	}
	mapper, err := apiutil.NewDiscoveryRESTMapper(cfg, httpClient)
	if err != nil {
		return nil, nil, err
	}
	c, err := client.New(cfg, client.Options{Scheme: Scheme, Mapper: mapper})
	return c, cfg, err
}
