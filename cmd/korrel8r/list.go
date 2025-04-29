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
		e, _ := newEngine()
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		defer func() { _ = w.Flush() }()
		switch len(args) {
		case 0:
			for _, d := range e.Domains() {
				fmt.Fprintf(w, "%v\t%v\n", d.Name(), d.Description())
			}
		case 1:
			d := must.Must1(e.Domain(args[0]))
			classes := must.Must1(e.ClassesFor(d))
			for _, c := range classes {
				fmt.Fprintln(w, c.Name())
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
