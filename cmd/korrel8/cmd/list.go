package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var (
	domainsCmd = &cobra.Command{
		Use:   "domains",
		Short: "List known domains",
		Run: func(cmd *cobra.Command, args []string) {
			e := newEngine()
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			for _, d := range e.Domains() {
				fmt.Fprintln(w, d.String())
			}
			w.Flush()
		},
	}

	classesCmd = &cobra.Command{
		Use:   "classes DOMAIN",
		Short: "List classes in DOMAIN",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			e := newEngine()
			d := e.Domain(args[0])
			if d == nil {
				check(fmt.Errorf("unknown domain: %v", d))
			}
			for _, c := range d.Classes() {
				fmt.Println(c.String())
			}
		},
	}

	rulesCmd = &cobra.Command{
		Use:   "rules DOMAIN",
		Short: "List classes in DOMAIN",
		Run: func(cmd *cobra.Command, args []string) {
			e := newEngine()
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			for _, r := range e.Rules() {
				fmt.Fprintf(w, "%v\t%v\t%v\n", r, r.Start(), r.Goal())
			}
			w.Flush()
		},
	}
)

func init() {
	rootCmd.AddCommand(domainsCmd, classesCmd, rulesCmd)
}
