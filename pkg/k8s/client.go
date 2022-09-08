package k8s

import (
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

// GetConfig gets the default REST configuration from the environment.
func GetConfig() (*rest.Config, error) {
	return config.GetConfig()
}
