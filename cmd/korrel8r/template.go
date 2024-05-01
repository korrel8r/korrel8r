// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	"io"
	"os"

	"text/template"

	"github.com/korrel8r/korrel8r/internal/pkg/must"
	"github.com/spf13/cobra"
)

var templateCmd = &cobra.Command{
	Use:   "template [--file FILE|--template STRING]",
	Short: `Apply a Go template to the korrel8r engine.`,
	Long: `Apply a Go template to the korrel8r engine.
Reads stdin by default if neither --file nor --template is provided.
Useful for testing rule and store templates.`,
	Run: func(cmd *cobra.Command, args []string) {
		if *templateString == "" { // Read from file
			switch *templateFile {
			case "", "-":
				*templateString = string(must.Must1(io.ReadAll(os.Stdin)))
			default:
				*templateString = string(must.Must1(os.ReadFile(*templateFile)))
			}
		}
		e, _ := newEngine()
		t := template.Must(template.New(*templateString).Funcs(e.TemplateFuncs()).Parse(*templateString))
		must.Must(t.Execute(os.Stdout, e))
	},
}

var templateFile, templateString *string

func init() {
	templateFile = templateCmd.Flags().StringP("file", "f", "", "read template from file")
	templateString = templateCmd.Flags().StringP("template", "t", "", "use template string")
	rootCmd.AddCommand(templateCmd)
}
