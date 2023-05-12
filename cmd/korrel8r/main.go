// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

/*
Copyright © 2022 Alan Conway

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/korrel8r/korrel8r/internal/pkg/logging"
	"github.com/korrel8r/korrel8r/internal/pkg/must"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:     "korrel8r",
		Short:   "Command line correlation tool",
		Version: "0.1.1",
	}
	log = logging.Log()

	// Global Flags
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

	cobra.OnInitialize(func() { logging.Init(*verbose) }) // After flags are parsed
}

// defaultRulePaths looks for a default "rules" directory in a few places.
func defaultRulePaths() []string {
	_, srcPath, _, _ := runtime.Caller(1)
	for _, path := range []string{
		os.Getenv("KORREL8R_RULE_DIR"),                                               // Try env var first.
		filepath.Join(filepath.Dir(must.Must1(os.Executable())), "rules"),            // Beside executable.
		filepath.Join(strings.TrimSuffix(srcPath, "/cmd/korrel8r/main.go"), "rules"), // In source tree
	} {
		if _, err := os.Stat(path); err == nil {
			return []string{path}
		}
	}
	return nil
}

func main() {
	// Code in this package panics with an error to exit.
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintln(os.Stderr, r)
			if *panicOnErr {
				panic(r)
			}
			os.Exit(1)
		}
		os.Exit(0)
	}()
	must.Must(rootCmd.Execute())
}
