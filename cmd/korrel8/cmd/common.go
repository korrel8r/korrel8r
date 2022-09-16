package cmd

import (
	"fmt"
	"os"

	"github.com/alanconway/korrel8/pkg/alert"
	"github.com/alanconway/korrel8/pkg/k8s"
	"github.com/alanconway/korrel8/pkg/korrel8"
	"github.com/alanconway/korrel8/pkg/openshift"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func exitErr(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func must[T any](v T, err error) T { exitErr(err); return v }

func open(name string) (f *os.File) {
	if name == "-" {
		return os.Stdin
	} else {
		return must(os.Open(name))
	}
}

func restConfig() (*rest.Config, error) {
	cfg, err := config.GetConfig()
	if err == nil {
		cfg.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(100, 1000)
	}
	return cfg, err
}

func newStore(d korrel8.Domain) (korrel8.Store, error) {
	cfg, err := restConfig()
	if err != nil {
		return nil, err
	}
	switch d {
	case k8s.Domain:
		c, err := client.New(cfg, client.Options{})
		if err != nil {
			return nil, err
		}
		return k8s.NewStore(c)
	case alert.Domain:
		return openshift.AlertManagerStore(cfg)
	default:
		return nil, fmt.Errorf("creating store for unknown domain %v", d)
	}
}
