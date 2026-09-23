// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"

	"github.com/korrel8r/korrel8r/internal/pkg/build"
	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/internal/pkg/must"
	"github.com/korrel8r/korrel8r/internal/pkg/test/mock"
	"github.com/korrel8r/korrel8r/pkg/config"
	"github.com/korrel8r/korrel8r/pkg/domains"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
	"github.com/korrel8r/korrel8r/pkg/engine"
	"github.com/korrel8r/korrel8r/pkg/memory"
	"github.com/korrel8r/korrel8r/pkg/rules/quickrules"
	"github.com/spf13/cobra"
)

var (
	log = logging.Log()

	rootCmd = &cobra.Command{
		Use:     "korrel8r",
		Short:   "Correlate observability data in a cluster",
		Version: build.Version,
	}

	// Global Flags
	verboseFlag = rootCmd.PersistentFlags().IntP("verbose", "v", 0, "Verbosity for logging (0: notice/error, 1: info/warn, 2: debug, 3: per-request, 4: per-rule, 5: per-query, 9: extra detail")
	configFlag  = rootCmd.PersistentFlags().StringP("config", "c", getConfig(), "Configuration file")
	panicFlag   = rootCmd.PersistentFlags().Bool("panic", false, "Panic on error")
)

const (
	configEnv     = "KORREL8R_CONFIG"
	defaultConfig = "/etc/korrel8r/korrel8r.yaml"
)

func init() {
	rootCmd.PersistentFlags().VarP(outputFlag, "output", "o", outputFlag.DocString(""))
	_ = rootCmd.PersistentFlags().MarkHidden("panic")
	_ = rootCmd.PersistentFlags().MarkHidden("sync")
	rootCmd.CompletionOptions.HiddenDefaultCmd = true

	var profileStop func()
	cobra.OnInitialize(func() {
		logging.Init(verboseFlag)
		k8s.SetLogger(logging.Log())
		profileStop = startProfile()
		startMetrics()
	})
	cobra.OnFinalize(func() {
		if profileStop != nil {
			profileStop()
		}
		if metricsStop != nil {
			metricsStop()
		}
	})
}

// getConfig looks for the default configuration file.
func getConfig() string {
	if config := os.Getenv(configEnv); config != "" {
		return config
	}
	return defaultConfig
}

func main() {
	defer func() {
		// Code in this package will panic with an error to cause an exit.
		if r := recover(); r != nil {
			err, ok := r.(error)
			if !ok || *panicFlag {
				panic(r)
			}
			fmt.Fprintf(os.Stderr, "\n%v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}()
	must.Must(rootCmd.Execute())
}

func newEngineWithConfigsAndGuard(c config.Configs, guard engine.SearchGuard) (*engine.Engine, error) {
	b := engine.Build()
	return b.Domains(append(domains.All, mock.NewDomain("mock"))...).
		Config(c).
		SearchGuard(guard).
		Rules(quickrules.Rules(b.GetDomains())...).
		StatusRules(quickrules.StatusRules(b.GetDomains())...).
		Engine()
}

func newMemoryGuard(c config.Configs) (*memory.Guard, error) {
	var tuning *config.Tuning
	if len(c) > 0 && c[0].Tuning != nil {
		tuning = c[0].Tuning
	}
	return memory.NewFromTuning(tuning)
}

func newEngine() *engine.Engine {
	configs := must.Must1(config.Load(*configFlag))
	guard := must.Must1(newMemoryGuard(configs))
	go guard.Run(context.Background())
	return must.Must1(newEngineWithConfigsAndGuard(configs, guard))
}
