// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package k8s

import (
	"net/http"

	"github.com/go-logr/logr"
	kconfig "github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/rest/auth"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
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
		if cfg, err = GetConfig(); err != nil {
			return nil, err
		}
	}
	return client.New(cfg, client.Options{})
}

// NewHTTPClient returns a new client with TLS settings from Store config.
func NewHTTPClient(s kconfig.Store) (*http.Client, error) {
	cfg, err := GetConfig()
	if err != nil {
		return nil, err
	}
	ca := s[kconfig.StoreKeyCA]
	if ca != "" {
		cfg.TLSClientConfig.CAFile = ca
	}
	return rest.HTTPClientFor(cfg)
}

// GetConfig returns a rest.Config with settings for use by korrel8r.
func GetConfig() (*rest.Config, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	// TODO make these configurable.
	cfg.QPS = float32(korrel8r.DefaultLimit)
	cfg.Burst = korrel8r.DefaultLimit
	cfg.Wrap(auth.Wrap)
	return cfg, nil
}
