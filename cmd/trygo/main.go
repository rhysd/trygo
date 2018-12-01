package main

import (
	"flag"
	"fmt"
	"github.com/rhysd/trygo"
	"os"
)

const usageHeader = `Usage: trygo [flags] {dirs...}

  trygo is a translator from TryGo sources into Go sources. Directory

Flags:`

var (
	outDir = flag.String("o", "", "Output directory path")
	debug  = flag.Bool("debug", false, "Output debug log")
)

func exit(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "trygo: error: %v\n", err)
		os.Exit(111)
	}
	os.Exit(0)
}

func usage() {
	fmt.Fprintln(os.Stderr, usageHeader)
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()

	trygo.InitLog(*debug)

	gen, err := trygo.NewGen(*outDir)
	if err != nil {
		exit(err)
	}

	if err := gen.Generate(flag.Args(), *debug); err != nil {
		exit(err)
	}
}
