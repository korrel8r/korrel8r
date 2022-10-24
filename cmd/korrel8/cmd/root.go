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
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/korrel8/korrel8/internal/pkg/logging"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "korrel8",
	Short:   "Command line correlation tool",
	Version: "0.1.0",
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

var (
	// Flags
	output      *string
	verbose     *int
	lokiBaseURL URLFlag
	rulePaths   *[]string
	panicOnErr  *bool
)

// defaultRulePaths looks for a default "rules" directory in a few places.
func defaultRulePaths() []string {
	for _, f := range []func() string{
		// Check for env var.
		func() string { return os.Getenv("KORREL8_RULE_DIR") },
		// Check for "rules" directory beside executable.
		func() string {
			path, _ := os.Executable()
			if path != "" {
				path = filepath.Join(filepath.Dir(path), "rules")
			}
			return path
		},
		// Check if there is a source tree for development.
		func() string {
			_, path, _, _ := runtime.Caller(1)
			if srcRoot := strings.TrimSuffix(path, "/cmd/korrel8/cmd/root.go"); srcRoot != path {
				return filepath.Join(srcRoot, "rules")
			}
			return ""
		},
	} {
		if path := f(); path != "" {
			if _, err := os.Stat(path); err == nil {
				return []string{path}
			}
		}
	}
	return nil
}

type URLFlag struct{ *url.URL }

func (u *URLFlag) String() string {
	if u.URL != nil {
		return u.URL.String()
	}
	return ""
}

func (u *URLFlag) Set(s string) error {
	var err error
	u.URL, err = url.Parse(s)
	return err
}

func (u *URLFlag) Type() string { return "URL" }

func init() {
	panicOnErr = rootCmd.PersistentFlags().Bool("panic", false, "panic on error instead of exit code 1")
	output = rootCmd.PersistentFlags().StringP("output", "o", "yaml", "Output format: json, json-pretty or yaml")
	verbose = rootCmd.PersistentFlags().IntP("verbose", "v", 0, "Verbosity for logging")

	rulePaths = rootCmd.PersistentFlags().StringArrayP("rules", "r", defaultRulePaths(), "Files or directories containing rules.")

	rootCmd.PersistentFlags().Var(&lokiBaseURL, "loki", "Loki base URL, up to .../v1")

	cobra.OnInitialize(func() { logging.Init(*verbose) })
}
