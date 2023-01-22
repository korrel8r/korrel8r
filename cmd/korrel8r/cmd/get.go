/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"os"

	"github.com/korrel8r/korrel8r/internal/pkg/must"
	"github.com/spf13/cobra"
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get CLASS [QUERY]",
	Short: "Execute QUERY for CLASS and print the results",
	Long: `
`,
	Args: cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		e := newEngine()
		c := must.Must1(e.Class(args[0]))
		s := must.Must1(e.StoreErr(c.Domain().String()))
		q := c.Domain().Query(c)

		log.V(1).Info("get", "query", q, "class", c)
		result := newPrinter(os.Stdout)
		must.Must(s.Get(context.Background(), q, result))
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
}
