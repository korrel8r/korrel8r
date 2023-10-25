// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/korrel8r/korrel8r/internal/pkg/must"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list [DOMAIN]",
	Short: "List domains or classes in DOMAIN.",
	Args:  cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		e := newEngine()
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
		defer w.Flush()
		switch len(args) {
		case 0:
			for _, d := range e.Domains() {
				fmt.Fprintln(w, d.Name(), "\t", d.Description())
			}
		case 1:
			d := must.Must1(e.DomainErr(args[0]))
			for _, c := range d.Classes() {
				fmt.Fprintln(w, c.Name(), "\t", c.Description())
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
