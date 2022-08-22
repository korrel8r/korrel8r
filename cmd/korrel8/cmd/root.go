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
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "korrel8",
	Short:   "Command line correlation tool - work in progress",
	Long:    `Command line correlation tool - work in progress`,
	Version: "0.0.0",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

var (
	pretty *bool
	output *string
)

func init() {
	pretty = rootCmd.PersistentFlags().BoolP("pretty", "p", true, "Pretty-print output with indentation")
	output = rootCmd.PersistentFlags().StringP("output", "o", "yaml", "Output format, json or yaml")
}
