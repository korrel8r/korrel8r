/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/korrel8/korrel8/internal/pkg/decoder"
	"github.com/spf13/cobra"
)

// correlateCmd represents the correlate command
var correlateCmd = &cobra.Command{
	Use:   "correlate START_CLASS GOAL_CLASS [START_FILE]",
	Short: "Correlate from START_CLASS to GOAL_CLASS starting from in START_FILE. '-' means use stdin",
	Args:  cobra.RangeArgs(2, 3),
	Run: func(cmd *cobra.Command, args []string) {
		e := newEngine()
		startClass, goalClass := must(e.ParseClass(args[0])), must(e.ParseClass(args[1]))
		startReader := os.Stdin
		if len(args) > 2 && args[2] != "-" {
			startReader = open(args[2])
			defer startReader.Close()
		}
		start := startClass.New()
		check(decoder.New(startReader).Decode(&start))
		path := e.Graph.ShortestPath(startClass, goalClass)
		// FIXME allow constraint
		queries, err := e.Follow(context.Background(), start, nil, path)
		check(err)
		fmt.Printf("\nresulting queries: %v\n\n", queries)
	},
}

var (
	focusTime *time.Time
	interval  *time.Duration
)

type TimeValue struct {
	Time *time.Time
}

func (v TimeValue) String() string {
	if v.Time != nil {
		return v.Time.String()
	}
	return ""
}

func (v TimeValue) Set(s string) error {
	if t, err := time.ParseInLocation(time.RFC3339, s, time.Local); err != nil {
		return err
	} else {
		*v.Time = t
	}
	return nil
}

func init() {
	rootCmd.AddCommand(correlateCmd)
	correlateCmd.Flags().Duration("interval", time.Minute*10, "limit results to this interval around --time")
	correlateCmd.Flags()
}
