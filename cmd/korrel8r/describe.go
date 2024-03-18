// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	"fmt"

	"github.com/korrel8r/korrel8r/internal/pkg/must"
	"github.com/spf13/cobra"
)

var describeCmd = &cobra.Command{
	Use:   "describe NAME",
	Short: "Describe NAME, which can be a domain or class name.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		e, _ := newEngine()
		if c, err := e.Class(args[0]); err == nil {
			fmt.Println(c.Description())
		} else if d, err := e.DomainErr(args[0]); err == nil {
			fmt.Println(d.Description())
		} else {
			must.Must(fmt.Errorf("not a domain or class name: %q", args[0]))
		}
	},
}

func init() {
	rootCmd.AddCommand(describeCmd)
}
