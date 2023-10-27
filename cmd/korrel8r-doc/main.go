// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/korrel8r/korrel8r/internal/pkg/adoc"
)

func main() {
	dirs := os.Args[1:]
	if len(dirs) == 0 {
		fmt.Fprintf(os.Stderr, `Generate asciidoc with a section for for each package directory.
usage: %v PKGDIRS...`, filepath.Base(os.Args[0]))
		os.Exit(1)
	}
	exit := 0
	for _, dir := range dirs {
		if err := genDoc(dir); err != nil {
			fmt.Fprintf(os.Stderr, "%v: %v", dir, err)
			exit = 1
		}
	}
	os.Exit(exit)
}

func genDoc(dir string) error {
	d, err := adoc.NewDomain(dir)
	if err != nil {
		return err
	}
	fmt.Printf("\n== %v\n\n", d.Name)
	fmt.Println(d.Asciidoc(2))
	return nil
}
