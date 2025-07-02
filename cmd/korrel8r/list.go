// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	"os"

	"github.com/korrel8r/korrel8r/internal/pkg/must"
	"github.com/korrel8r/korrel8r/internal/pkg/text"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list [DOMAIN]",
	Short: "List domains or classes in DOMAIN.",
	Args:  cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		e, _ := newEngine()
		w := text.NewPrinter(e)
		switch len(args) {
		case 0:
			w.ListDomains(os.Stdout)
		case 1:
			w.ListClasses(os.Stdout, must.Must1(e.Domain(args[0])))
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
