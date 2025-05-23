// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	"fmt"

	"github.com/korrel8r/korrel8r/internal/pkg/build"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of this command.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(build.Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
