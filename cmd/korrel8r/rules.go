// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"text/tabwriter"

	"github.com/korrel8r/korrel8r/internal/pkg/must"
	"github.com/korrel8r/korrel8r/pkg/graph"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/spf13/cobra"
	"gonum.org/v1/gonum/graph/encoding/dot"
)

var rulesCmd = &cobra.Command{
	Use:   "rules",
	Short: "List rules by start, goal or name",
	Run: func(cmd *cobra.Command, args []string) {
		e, _ := newEngine()
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
		test := func(r korrel8r.Rule) bool {
			return (start == nil || slices.Contains(r.Start(), start)) &&
				(goal == nil || slices.Contains(r.Goal(), goal)) &&
				name.MatchString(r.Name())
		}
		if *ruleGraph {
			g := e.Graph().Select(func(l *graph.Line) bool { return test(l.Rule) })
			b := must.Must1(dot.MarshalMulti(g, "", "", "  "))
			os.Stdout.Write(b)
		} else { // Print rules as text
			for _, r := range e.Rules() {
				if test(r) {
					fmt.Fprintln(w, r)
					if *ruleLong {
						fmt.Fprintln(w, "  start: ", r.Start())
						fmt.Fprintln(w, "  goal: ", r.Goal())
					}
				}
			}
		}
		w.Flush()
	},
}

var (
	ruleStart, ruleGoal, ruleName *string
	ruleLong, ruleGraph           *bool
)

func init() {
	ruleStart = rulesCmd.Flags().StringP("start", "s", "", "show rules with this start class")
	ruleGoal = rulesCmd.Flags().StringP("goal", "g", "", "show rules with this goal class")
	ruleName = rulesCmd.Flags().StringP("name", "n", "", "show rules with name matching this regexp")
	ruleGraph = rulesCmd.Flags().Bool("graph", false, "write rule graph in graphviz format")
	ruleLong = rulesCmd.Flags().Bool("long", false, "show rule start and goal classes")
	rootCmd.AddCommand(rulesCmd)
}
