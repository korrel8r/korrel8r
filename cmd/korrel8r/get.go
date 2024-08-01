// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"context"
	"os"

	"time"

	"github.com/korrel8r/korrel8r/internal/pkg/must"
	"github.com/korrel8r/korrel8r/pkg/korrel8r"
	"github.com/korrel8r/korrel8r/pkg/ptr"
	"github.com/spf13/cobra"
)

// getCmd represents the get command
var (
	sinceFlag, untilFlag *time.Duration
	limitFlag            *uint

	getCmd = &cobra.Command{
		Use:   "get DOMAIN:CLASS:QUERY",
		Short: "Execute QUERY and print the results",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			e, _ := newEngine()
			q := must.Must1(e.Query(args[0]))
			result := newPrinter(os.Stdout)
			c := &korrel8r.Constraint{}
			if *limitFlag > 0 {
				c.Limit = limitFlag
			}
			now := time.Now()
			if *sinceFlag > 0 {
				c.Start = ptr.To(now.Add(-*sinceFlag))
			}
			if *untilFlag > 0 {
				c.End = ptr.To(now.Add(-*untilFlag))
			}
			must.Must(e.Get(context.Background(), q, c, result))
		},
	}
)

func init() {
	sinceFlag = getCmd.Flags().Duration("since", 0, "Only get results since this long ago.")
	untilFlag = getCmd.Flags().Duration("until", 0, "Only get results until this long ago.")
	limitFlag = getCmd.Flags().Uint("limit", 0, "Limit total number of results.")
	rootCmd.AddCommand(getCmd)
}
