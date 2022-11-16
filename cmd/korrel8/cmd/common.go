package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/korrel8/korrel8/internal/pkg/decoder"
	"github.com/korrel8/korrel8/internal/pkg/logging"
	alert "github.com/korrel8/korrel8/pkg/amalert"
	"github.com/korrel8/korrel8/pkg/engine"
	"github.com/korrel8/korrel8/pkg/k8s"
	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/korrel8/korrel8/pkg/loki"
	"github.com/korrel8/korrel8/pkg/templaterule"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/yaml"
)

var (
	log = logging.Log
	ctx = context.Background()
)

func check(err error, format ...any) {
	if err != nil {
		if len(format) == 0 {
			panic(err)
		} else {
			panic(fmt.Errorf(format[0].(string), format[1:]...))
		}
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

func newEngine() *engine.Engine {
	cfg := restConfig()
	e := engine.New("korrel8")
	for _, x := range []struct {
		d      korrel8.Domain
		create func() (korrel8.Store, error)
	}{
		{k8s.Domain, func() (korrel8.Store, error) { return k8s.NewStore(k8sClient(cfg), cfg) }},
		{alert.Domain, func() (korrel8.Store, error) { return alert.NewOpenshiftAlertManagerStore(ctx, cfg) }},
		{loki.Domain, func() (korrel8.Store, error) { return loki.NewOpenshiftLokiStackStore(ctx, k8sClient(cfg), cfg) }},
	} {
		s, err := x.create()
		if err != nil {
			log.Error(err, "error creating store", "domain", x.d)
		}
		e.AddDomain(x.d, s)
	}
	// Load rules
	for _, path := range *rulePaths {
		check(loadRules(e, path))
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
		return printer{print: func(o korrel8.Object) { check(encoder.Encode(o)) }}

	case "yaml":
		return printer{print: func(o korrel8.Object) { fmt.Fprintf(w, "---\n%s", must(yaml.Marshal(&o))) }}

	default:
		check(fmt.Errorf("invalid output type: %v", *output))
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
	return filepath.WalkDir(root, func(path string, info fs.DirEntry, err error) (reterr error) {
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
			return fmt.Errorf("%v:%v: error loading rules: %v", path, d.Line(), err)
		}
		return nil
	})
}

// queryFromArgs treats args[0] as the initial URI, and args[1:] as NAME=VALUE strings for query parameters.
func queryFromArgs(args []string) (*korrel8.Query, error) {
	if len(args) == 0 {
		return &korrel8.Query{}, nil
	}
	u, err := url.Parse(args[0])
	if err != nil {
		return nil, err
	}
	q := u.Query()
	for _, nv := range args[1:] {
		n, v, ok := strings.Cut(nv, "=")
		if !ok {
			return nil, fmt.Errorf("not a name=value argument: %v", nv)
		}
		q.Set(n, v)
	}
	u.RawQuery = q.Encode()
	return u, err
}
