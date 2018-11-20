package main

import (
	"flag"
	"fmt"
	"github.com/rhysd/trygo"
	"os"
)

const usageHeader = `Usage: trygo [flags] {paths...}

  trygo is a translator from TryGo sources into Go sources. Directory

Flags:`

var (
	outDir = flag.String("o", "", "Output directory path")
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
	if args := flag.Args(); len(args) == 0 {
		trygo.NewGenGoGenerate(*outDir)
	}

	gen, err := trygo.NewGen(flag.Args(), *outDir)
	if err != nil {
		exit(err)
	}

	// TODO
	fmt.Println(gen)
}
