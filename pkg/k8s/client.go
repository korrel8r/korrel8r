package k8s

import (
	"net/http"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// NewClient creates a k8s client using the default kube config from the environment.
//
// This is just a convenience. For more control of client configuration,
// use the "sigs.k8s.io/controller-runtime/pkg/client" package directly.
func NewClient() (client.Client, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	return client.New(cfg, client.Options{})
}

// NewDefaultHTTPClient returns a HTTP client using the default kube config from the environment.
//
// This is just a convenience. For more control the client configuration,
// use the "k8s.io/client-go/rest" package directly.
func NewHTTPClient() (*http.Client, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	return rest.HTTPClientFor(cfg)
}

// GetConfig gets the default REST configuration from the environment.
func GetConfig() (*rest.Config, error) {
	return config.GetConfig()
}
