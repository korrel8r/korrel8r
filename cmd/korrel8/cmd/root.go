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

	"github.com/alanconway/korrel8/internal/pkg/logging"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "korrel8",
	Short:   "Command line correlation tool",
	Version: "0.1.0",
}

func Execute() (exitCode int) {
	defer func() {
		if err, _ := recover().(error); err != nil {
			fmt.Fprintln(os.Stderr, err)
			exitCode = 1
		}
	}()
	check(rootCmd.Execute())
	return 0
}

var (
	// Flags
	pretty  *bool
	output  *string
	verbose *int
)

func init() {
	pretty = rootCmd.PersistentFlags().BoolP("pretty", "p", true, "Pretty-print output with indentation")
	output = rootCmd.PersistentFlags().StringP("output", "o", "yaml", "Output format, json or yaml")
	verbose = rootCmd.PersistentFlags().IntP("verbose", "v", 0, "Verbosity for logging")

	cobra.OnInitialize(func() { logging.Init(*verbose) })
}
