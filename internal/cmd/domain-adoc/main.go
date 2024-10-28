// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package main

import (
	_ "embed"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/korrel8r/korrel8r/internal/pkg/asciidoc"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] PKGSPEC...:\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}
	flag.Parse()

	if len(flag.Args()) < 1 {
		flag.Usage()
	}
	log.Print(flag.Args())
	for i := 0; i < len(flag.Args()); i++ {
		log.Print(flag.Args()[i])
		domains, err := asciidoc.Load(flag.Args()[i])
		check(err)
		for _, d := range domains {
			check(d.Write(os.Stdout))
		}
	}
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

}
