package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list [DOMAIN|rules]",
	Short: "List known domains or classes",
	Long: `
list         # list all known domains
list DOMAIN  # list all known classes in DOMAIN
list rules   # list all known rules
`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		e := newEngine()
		fmt.Println()
		switch {
		case len(args) == 0:
			for d := range e.Domains {
				fmt.Println(d)
			}
		case args[0] == "rules":
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			for _, r := range e.Rules {
				fmt.Fprintf(w, "%v\t%v\t%v\n", r, r.Start(), r.Goal())
			}
			w.Flush()
		default:
			if d := e.Domains[args[0]]; d == nil {
				check(fmt.Errorf("unknown domain name: %q", args[0]))
			} else {
				for _, c := range d.KnownClasses() {
					fmt.Println(c.String())
				}
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
