package cmd

import (
	"fmt"
	"os"
	"regexp"
	"text/tabwriter"

	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list [DOMAIN]",
	Short: "List domains, classes or rules.",
	Args:  cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		e := newEngine()
		switch len(args) {
		case 0:
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			defer w.Flush()
			for _, d := range e.Domains() {
				fmt.Fprintln(w, d.String())
			}
		case 1:
			d, err := e.Domain(args[0])
			check(err)
			for _, c := range d.Classes() {
				fmt.Println(c.String())
			}
		}
	},
}

var rulesCmd = &cobra.Command{
	Use:   "rules",
	Short: "List rules by start, goal or name",
	Run: func(cmd *cobra.Command, args []string) {
		e := newEngine()
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		defer w.Flush()
		var start, goal korrel8.Class
		if *ruleStart != "" {
			start = must(e.ParseClass(*ruleStart))
		}
		if *ruleGoal != "" {
			goal = must(e.ParseClass(*ruleGoal))
		}
		name := must(regexp.Compile(*ruleName))
		for _, r := range e.Rules() {
			if (start == nil || r.Start() == start) &&
				(goal == nil || r.Goal() == goal) &&
				name.MatchString(r.String()) {
				fmt.Fprintf(w, "%v\t%v\t%v\n", r, r.Start(), r.Goal())
			}
		}
		w.Flush()
	},
}

var ruleStart, ruleGoal, ruleName *string

func init() {
	ruleStart = rulesCmd.Flags().String("start", "", "show rules with this start class")
	ruleGoal = rulesCmd.Flags().String("goal", "", "show rules with this goal class")
	ruleName = rulesCmd.Flags().String("name", "", "show rules with name matching this regexp")
	rootCmd.AddCommand(listCmd)
	listCmd.AddCommand(rulesCmd)
}
