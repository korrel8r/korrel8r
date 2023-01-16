package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/korrel8/korrel8/internal/pkg/decoder"
	"github.com/korrel8/korrel8/internal/pkg/logging"
	"github.com/korrel8/korrel8/internal/pkg/must"
	"github.com/korrel8/korrel8/pkg/alert"
	"github.com/korrel8/korrel8/pkg/engine"
	"github.com/korrel8/korrel8/pkg/k8s"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/korrel8/korrel8/pkg/loki"
	"github.com/korrel8/korrel8/pkg/templaterule"
	"github.com/korrel8/korrel8/pkg/uri"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/yaml"
)

var (
	log = logging.Log()
	ctx = context.Background()
)

func restConfig() *rest.Config {
	cfg, err := config.GetConfig()
	if err == nil {
		cfg.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(100, 1000)
	}
	return must.Must1(cfg, err)
}

func k8sClient(cfg *rest.Config) client.Client {
	log.V(2).Info("create k8s client")
	return must.Must1(client.New(cfg, client.Options{}))
}

func newEngine() *engine.Engine {
	log.V(2).Info("create engine")
	cfg := restConfig()
	e := engine.New()
	for _, x := range []struct {
		d      korrel8.Domain
		create func() (korrel8.Store, error)
	}{
		{k8s.Domain, func() (korrel8.Store, error) { return k8s.NewStore(k8sClient(cfg), cfg) }},
		{alert.Domain, func() (korrel8.Store, error) { return alert.NewOpenshiftAlertManagerStore(ctx, cfg) }},
		{loki.Domain, func() (korrel8.Store, error) { return loki.NewOpenshiftLokiStackStore(ctx, k8sClient(cfg), cfg) }},
	} {
		log.V(2).Info("add domain", "domain", x.d)
		s, err := x.create()
		if err != nil {
			log.Error(err, "error creating store", "domain", x.d)
		}
		e.AddDomain(x.d, s)
	}
	// Load rules
	for _, path := range *rulePaths {
		must.Must(loadRules(e, path))
	}
	return e
}

func jsonString(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return err.Error()
	}
	return string(b)
}

type printer struct{ print func(o korrel8.Object) }

func newPrinter(w io.Writer) printer {
	switch *output {

	case "json":
		return printer{print: func(o korrel8.Object) { fmt.Fprintln(w, jsonString(o)) }}

	case "json-pretty":
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return printer{print: func(o korrel8.Object) { must.Must(encoder.Encode(o)) }}

	case "yaml":
		return printer{print: func(o korrel8.Object) { fmt.Fprintf(w, "---\n%s", must.Must1(yaml.Marshal(&o))) }}

	default:
		must.Must(fmt.Errorf("invalid output type: %v", *output))
		return printer{}
	}
}

func (p printer) Append(objects ...korrel8.Object) {
	for _, o := range objects {
		p.print(o)
	}
}

// loadRules from a file or walk a directory to find files.
func loadRules(e *engine.Engine, root string) error {
	log.V(3).Info("loading rules from", "root", root)
	return filepath.WalkDir(root, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		ext := filepath.Ext(path)
		if !info.Type().IsRegular() || (ext != ".yaml" && ext != ".yml" && ext != ".json") {
			return nil // Skip file
		}
		log.V(3).Info("loading rules", "path", path)
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		d := decoder.New(f)
		if err := templaterule.AddRules(d, e); err != nil {
			return fmt.Errorf("%v: error loading rules: %v", path, err)
		}
		return nil
	})
}

// referenceArgs treats args[0] as the initial URI, and args[1:] as NAME=VALUE strings for query parameters.
func referenceArgs(args []string) (uri.Reference, error) {
	if len(args) == 0 {
		return uri.Reference{}, nil
	}
	ref := uri.Make(args[0], args[1:]...)
	return uri.Parse(ref.String()) // Make sure the path is valid for a URI reference
}
