// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	"fmt"
	"os"
	"regexp"
	"text/tabwriter"

	"github.com/korrel8r/korrel8r/internal/pkg/must"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list [DOMAIN]",
	Short: "List domains or classes in DOMAIN.",
	Args:  cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		e := newEngine()
		switch len(args) {
		case 0:
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			defer w.Flush()
			for _, d := range e.Domains() {
				fmt.Fprintln(w, d.Name())
			}
		case 1:
			d := must.Must1(e.DomainErr(args[0]))
			for _, c := range d.Classes() {
				fmt.Println(c.Name())
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
		var start, goal korrel8r.Class
		if *ruleStart != "" {
			start = must.Must1(e.Class(*ruleStart))
		}
		if *ruleGoal != "" {
			goal = must.Must1(e.Class(*ruleGoal))
		}
		name := must.Must1(regexp.Compile(*ruleName))
		fmt.Fprintln(w, "RULE\tSTART\tGOAL")
		for _, r := range e.Rules() {
			if (start == nil || r.Start() == start) &&
				(goal == nil || r.Goal() == goal) &&
				name.MatchString(r.Name()) {
				fmt.Fprintf(w, "%v\t%v/%v\t%v/%v\n", r, r.Start().Domain(), r.Start(), r.Goal().Domain(), r.Goal())
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
