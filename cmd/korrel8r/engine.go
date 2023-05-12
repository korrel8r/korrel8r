package main

import (
	"context"
	"fmt"
	"net/url"

	"github.com/korrel8r/korrel8r/internal/pkg/must"
	"github.com/korrel8r/korrel8r/pkg/domains/alert"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/domains/logs"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"

	"github.com/korrel8r/korrel8r/pkg/domains/metric"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	ctx = context.Background()
)

func restConfig() *rest.Config {
	cfg := must.Must1(config.GetConfig())
	cfg.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(100, 1000)
	return cfg
}

func k8sClient(cfg *rest.Config) client.Client {
	log.V(2).Info("create k8s client")
	return must.Must1(client.New(cfg, client.Options{}))
}

func parseURL(s string) (*url.URL, error) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("%q: unsupported scheme", s)
	}
	return u, nil
}

func newEngine() *engine.Engine {
	log.V(2).Info("create engine")
	cfg := restConfig()
	e := engine.New()
	for _, x := range []struct {
		d      korrel8r.Domain
		create func() (korrel8r.Store, error)
	}{
		{k8s.Domain, func() (korrel8r.Store, error) { return k8s.NewStore(k8sClient(cfg), cfg) }},
		{alert.Domain, func() (korrel8r.Store, error) {
			if *alertmanagerAPI == "" && *metricsAPI == "" {
				return alert.NewOpenshiftStore(ctx, cfg)
			}

			log.V(1).Info("using user-specified Alertmanager API", "url", *alertmanagerAPI)
			alertmanagerURL, err := parseURL(*alertmanagerAPI)
			if err != nil {
				return nil, err
			}

			log.V(1).Info("using user-specified metrics API", "url", *metricsAPI)
			prometheusURL, err := parseURL(*metricsAPI)
			if err != nil {
				return nil, err
			}

			return alert.NewStore(alertmanagerURL, prometheusURL, nil)
		}},
		{logs.Domain, func() (korrel8r.Store, error) {
			if *logsAPI == "" {
				return logs.NewOpenshiftLokiStackStore(ctx, k8sClient(cfg), cfg)
			}

			log.V(1).Info("using user-specified logs API", "url", *logsAPI)
			u, err := parseURL(*logsAPI)
			if err != nil {
				return nil, err
			}

			return logs.NewLokiStackStore(u, nil)
		}},
		{metric.Domain, func() (korrel8r.Store, error) {
			if *metricsAPI != "" {
				log.V(1).Info("using user-specified metrics API", "url", *metricsAPI)
				u, err := parseURL(*metricsAPI)
				if err != nil {
					return nil, err
				}

				return metric.NewStore(u, nil)
			}

			return metric.NewOpenshiftStore(ctx, k8sClient(cfg), cfg)
		}},
	} {
		log.V(3).Info("add domain", "domain", x.d)
		s, err := x.create()
		if err != nil {
			log.Error(err, "error creating store", "domain", x.d)
		}
		e.AddDomain(x.d, s)
	}

	// Load rules
	for _, path := range *rulePaths {
		must.Must(e.LoadRules(path))
	}
	return e
}
