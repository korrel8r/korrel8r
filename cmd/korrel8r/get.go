// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/must"
	"github.com/korrel8r/korrel8r/pkg/engine/traverse"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/ptr"
	"github.com/korrel8r/korrel8r/pkg/rest"
	"github.com/spf13/cobra"
)

var (
	getCmd = &cobra.Command{
		Use:   "get DOMAIN:CLASS:QUERY",
		Short: "Execute QUERY and print the results",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			e, _ := newEngine()
			q := must.Must1(e.Query(args[0]))
			result := newPrinter(os.Stdout)
			c := getFlags.Constraint()
			must.Must(e.Get(context.Background(), q, c, result))
		},
	}
	getFlags *constraintFlags
)

func init() {
	getFlags = newConstraintFlags(getCmd)
	rootCmd.AddCommand(getCmd)
}

var (
	neighboursCmd = &cobra.Command{
		Use:   "neighbours DOMAIN:CLASS:QUERY [DEPTH]",
		Short: "Execute QUERY, compute a neighbourhood of the results up to DEPTH.",
		Args:  cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			e, _ := newEngine()
			q := must.Must1(e.Query(args[0]))
			d := 3
			if len(args) == 2 {
				d = must.Must1(strconv.Atoi(args[1]))
			}
			ctx := korrel8r.WithConstraint(context.Background(), neighboursFlags.Constraint())
			g := must.Must1(traverse.New(e, e.Graph()).Neighbours(ctx, traverse.Start{Class: q.Class(), Queries: []korrel8r.Query{q}}, d))
			newPrinter(os.Stdout).Print(rest.NewGraph(g))
		},
	}
	neighboursFlags *constraintFlags
)

func init() {
	neighboursFlags = newConstraintFlags(neighboursCmd)
	rootCmd.AddCommand(neighboursCmd)
}

var (
	goalsCmd = &cobra.Command{
		Use:   "goals DOMAIN:CLASS:QUERY [GOAL...]",
		Short: "Execute QUERY, find all paths to GOAL classes.",
		Args:  cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			e, _ := newEngine()
			q := must.Must1(e.Query(args[0]))
			var goals []korrel8r.Class
			for _, g := range args[1:] {
				goals = append(goals, must.Must1(e.Class(g)))
			}
			ctx := korrel8r.WithConstraint(context.Background(), goalsFlags.Constraint())
			g := must.Must1(traverse.New(e, e.Graph()).Goals(ctx, traverse.Start{Class: q.Class(), Queries: []korrel8r.Query{q}}, goals))
			newPrinter(os.Stdout).Print(rest.NewGraph(g))
		},
	}
	goalsFlags *constraintFlags
)

func init() {
	goalsFlags = newConstraintFlags(goalsCmd)
	rootCmd.AddCommand(goalsCmd)
}

type constraintFlags struct {
	sinceFlag, untilFlag, timeoutFlag *time.Duration
	limitFlag                         *int
}

func newConstraintFlags(cmd *cobra.Command) *constraintFlags {
	cf := &constraintFlags{}
	cf.limitFlag = cmd.Flags().Int("limit", 0, "Limit total number of results.")
	cf.sinceFlag = cmd.Flags().Duration("since", 0, "Only get results since this long ago.")
	cf.untilFlag = cmd.Flags().Duration("until", 0, "Only get results until this long ago.")
	cf.timeoutFlag = cmd.Flags().Duration("timeout", 0, "Timeout for store requests.")
	return cf
}

func (cf *constraintFlags) Constraint() *korrel8r.Constraint {
	c := &korrel8r.Constraint{}
	if *cf.limitFlag > 0 {
		c.Limit = cf.limitFlag
	}
	if *cf.timeoutFlag > 0 {
		c.Timeout = cf.timeoutFlag
	}
	now := time.Now()
	if *cf.sinceFlag > 0 {
		c.Start = ptr.To(now.Add(-*cf.sinceFlag))
	}
	if *cf.untilFlag > 0 {
		c.End = ptr.To(now.Add(-*cf.untilFlag))
	}
	return c
}
