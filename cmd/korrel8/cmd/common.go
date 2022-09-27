package cmd

import (
	"os"

	"github.com/alanconway/korrel8/pkg/alert"
	"github.com/alanconway/korrel8/pkg/k8s"
	"github.com/alanconway/korrel8/pkg/korrel8"
	"github.com/alanconway/korrel8/pkg/loki"
	"github.com/alanconway/korrel8/pkg/openshift"
	"github.com/alanconway/korrel8/pkg/rules"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func must[T any](v T, err error) T { check(err); return v }

func open(name string) (f *os.File) {
	if name == "-" {
		return os.Stdin
	} else {
		return must(os.Open(name))
	}
}

func restConfig() *rest.Config {
	cfg, err := config.GetConfig()
	if err == nil {
		cfg.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(100, 1000)
	}
	return must(cfg, err)
}

func k8sClient(cfg *rest.Config) client.Client {
	return must(client.New(cfg, client.Options{}))
}

func engine() *korrel8.Engine {
	cfg := restConfig()
	e := korrel8.NewEngine()
	rules.AddTo(e.Rules)
	e.Add(k8s.Domain, must(k8s.NewStore(k8sClient(cfg))))
	e.Add(alert.Domain, must(openshift.AlertManagerStore(cfg)))
	e.Add(loki.Domain, must(openshift.AlertManagerStore(cfg)))
	return e
}
