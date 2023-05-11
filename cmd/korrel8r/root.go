package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:     "korrel8r",
		Short:   "Command line correlation tool",
		Version: "0.1.1",
	}
	// Flags
	output          *string
	verbose         *int
	rulePaths       *[]string
	metricsAPI      *string
	alertmanagerAPI *string
	logsAPI         *string
	panicOnErr      *bool
)

func init() {
	panicOnErr = rootCmd.PersistentFlags().Bool("panic", false, "panic on error instead of exit code 1")
	output = rootCmd.PersistentFlags().StringP("output", "o", "yaml", "Output format: json, json-pretty or yaml")
	verbose = rootCmd.PersistentFlags().IntP("verbose", "v", 0, "Verbosity for logging")
	rulePaths = rootCmd.PersistentFlags().StringArray("rules", defaultRulePaths(), "Files or directories containing rules.")
	metricsAPI = rootCmd.PersistentFlags().StringP("metrics-url", "", "", "URL to the metrics API")
	alertmanagerAPI = rootCmd.PersistentFlags().StringP("alertmanager-url", "", "", "URL to the Alertmanager API")
	logsAPI = rootCmd.PersistentFlags().StringP("logs-url", "", "", "URL to the logs API")
	cobra.OnInitialize(func() { logging.Init(*verbose) })
}

// defaultRulePaths looks for a default "rules" directory in a few places.
func defaultRulePaths() []string {
	for _, f := range []func() string{
		func() string { return os.Getenv("KORREL8R_RULE_DIR") },                                       // Environment directory
		func() string { exe, _ := os.Executable(); return filepath.Join(filepath.Dir(exe), "rules") }, // Beside executable
		func() string { // Source tree
			_, path, _, _ := runtime.Caller(1)
			if _, err := os.Stat(path); err == nil {
				return filepath.Join(strings.TrimSuffix(path, "/cmd/korrel8r/root.go"), "rules")
			}
			return ""
		},
	} {
		path := f()
		if _, err := os.Stat(path); err == nil {
			return []string{path}
		}
	}
	return nil
}
