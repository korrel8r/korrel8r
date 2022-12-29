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
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/korrel8/korrel8/internal/pkg/logging"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:     "korrel8",
		Short:   "Command line correlation tool",
		Version: "0.1.1",
	}
	// Flags
	output     *string
	verbose    *int
	rulePaths  *[]string
	panicOnErr *bool
)

func init() {
	panicOnErr = rootCmd.PersistentFlags().Bool("panic", false, "panic on error instead of exit code 1")
	output = rootCmd.PersistentFlags().StringP("output", "o", "yaml", "Output format: json, json-pretty or yaml")
	verbose = rootCmd.PersistentFlags().IntP("verbose", "v", 0, "Verbosity for logging")
	rulePaths = rootCmd.PersistentFlags().StringArray("rules", defaultRulePaths(), "Files or directories containing rules.")
	cobra.OnInitialize(func() { logging.Init(*verbose) })
}

func Execute() (exitCode int) {
	defer func() {
		if !*panicOnErr { // Suppress panic
			if r := recover(); r != nil {
				fmt.Fprintln(os.Stderr, r)
				exitCode = 1
			}
		}
	}()
	check(rootCmd.Execute())
	return 0
}

// defaultRulePaths looks for a default "rules" directory in a few places.
func defaultRulePaths() []string {
	for _, f := range []func() string{
		func() string { return os.Getenv("KORREL8_RULE_DIR") },                                        // Environment directory
		func() string { exe, _ := os.Executable(); return filepath.Join(filepath.Dir(exe), "rules") }, // Beside executable
		func() string { // Check for source tree
			_, path, _, _ := runtime.Caller(1)
			if _, err := os.Stat(path); err == nil {
				return filepath.Join(strings.TrimSuffix(path, "/cmd/korrel8/cmd/root.go"), "rules")
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
