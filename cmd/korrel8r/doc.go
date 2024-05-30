// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	"time"

	stdlog "log"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func init() {
	docCmd := &cobra.Command{
		Use:    "doc",
		Short:  "Generate documentation",
		Hidden: true,
	}
	rootCmd.AddCommand(docCmd)

	manCmd := &cobra.Command{
		Use:   "man DIR",
		Short: "Generate man pages in directory DIR",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			header := &doc.GenManHeader{
				Title:   cmd.Name(),
				Section: "1",
				Date:    func() *time.Time { t := time.Now(); return &t }(),
			}
			if err := doc.GenManTree(rootCmd, header, args[0]); err != nil {
				stdlog.Fatalln(err)
			}
		},
	}
	docCmd.AddCommand(manCmd)

	markdownCmd := &cobra.Command{
		Use:   "markdown DIR",
		Short: "Generate markdown documentation in directory DIR",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if err := doc.GenMarkdownTree(rootCmd, args[0]); err != nil {
				stdlog.Fatalln(err)
			}
		},
	}
	docCmd.AddCommand(markdownCmd)
}
