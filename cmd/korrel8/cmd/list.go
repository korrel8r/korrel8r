package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list [DOMAIN]",
	Short: "List known domains or classes",
	Long: `
list         # list all known domains
list DOMAIN  # list all known classes in DOMAIN
`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		e := engine()
		fmt.Println()
		switch len(args) {
		case 0:
			for d := range e.Domains {
				fmt.Println(d)
			}
		case 1:
			d := e.Domains[args[0]]
			if d == nil {
				check(fmt.Errorf("unknown domain name: %q", args[0]))
			}
			for _, c := range d.KnownClasses() {
				fmt.Println(c.String())
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
