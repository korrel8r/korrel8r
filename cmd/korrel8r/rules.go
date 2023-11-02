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
		for _, r := range e.Rules() {
			if (start == nil || r.Start() == start) &&
				(goal == nil || r.Goal() == goal) &&
				name.MatchString(r.Name()) {
				fmt.Fprintln(w, korrel8r.RuleName(r))
			}
		}
		w.Flush()
	},
}

var (
	ruleStart, ruleGoal, ruleName *string
)

func init() {
	ruleStart = rulesCmd.Flags().StringP("start", "s", "", "show rules with this start class")
	ruleGoal = rulesCmd.Flags().StringP("goal", "g", "", "show rules with this goal class")
	ruleName = rulesCmd.Flags().StringP("name", "n", "", "show rules with name matching this regexp")
	rootCmd.AddCommand(rulesCmd)
}
