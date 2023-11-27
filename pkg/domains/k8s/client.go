// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package k8s

import (
	"time"

	"github.com/go-logr/logr"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"net/http"
)

// SetLogger sets the logger for controller-runtime.
func SetLogger(l logr.Logger) {
	log.SetLogger(l)
}

// NewClient provides a general-purpose k8s client.
// It may be used by other domains that need to talk to the cluster.
// If cfg is nil, use GetConfig() to get a default config.
func NewClient(cfg *rest.Config) (c client.Client, err error) {
	if cfg == nil {
		if cfg, err  = GetConfig(); err != nil {
			return nil,err
		}
	}
	hc, err := rest.HTTPClientFor(cfg)
	if err != nil {
		return nil, err
	}
	mapper, err := apiutil.NewDiscoveryRESTMapper(cfg, hc)
	if err != nil {
		return nil, err
	}
	return client.New(cfg, client.Options{Scheme: Scheme, Mapper: mapper})
}

// NewHTTPClient returns a new client for GetConfig()
func NewHTTPClient() (*http.Client, error){
	cfg, err  := GetConfig()
	if err != nil {
		return nil,err
	}
	return rest.HTTPClientFor(cfg)
}

// GetConfig returns a rest.Config with settings for use by korrel8r.
func GetConfig() (*rest.Config, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	// TODO configurable settings for k8s client.
	// Reduce client-side throttling for rapid results.
	cfg.QPS = 100
	cfg.Burst = 1000
	cfg.Timeout = 5 * time.Second
	return cfg, nil
}
