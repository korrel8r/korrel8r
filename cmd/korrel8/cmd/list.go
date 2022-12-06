package cmd

import (
	"fmt"
	"os"
	"regexp"
	"text/tabwriter"

	"github.com/korrel8/korrel8/pkg/korrel8"
	"github.com/spf13/cobra"
)

var domainsCmd = &cobra.Command{
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

var classesCmd = &cobra.Command{
	Use:   "classes DOMAIN [REGEXP]",
	Short: "List classes in DOMAIN with names matching REGEXP",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		e := newEngine()
		d, err := e.Domain(args[0])
		check(err)
		var match = regexp.MustCompile("")
		if len(args) > 1 {
			match = must(regexp.Compile(args[1]))
		}
		for _, c := range d.Classes() {
			if match.MatchString(c.String()) {
				fmt.Println(c.String())
			}
		}
	},
}

var classCmd = &cobra.Command{
	Use:   "class NAME [NAME...]",
	Short: "Verify and expand class names",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		e := newEngine()
		for _, name := range args {
			c, err := e.ParseClass(name)
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Println(c.String())
			}
		}
	},
}

var rulesCmd = &cobra.Command{
	Use:   "rules",
	Short: "List all rules",
	Run: func(cmd *cobra.Command, args []string) {
		e := newEngine()
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
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

	rootCmd.AddCommand(domainsCmd, classesCmd, classCmd, rulesCmd)
}
