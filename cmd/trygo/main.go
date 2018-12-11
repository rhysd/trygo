package main

import (
	"flag"
	"fmt"
	"github.com/fatih/color"
	"github.com/mattn/go-colorable"
	"github.com/rhysd/trygo"
	"os"
)

const usageHeader = `Usage: trygo [flags] {dirs...}

  trygo is a translator from TryGo sources into Go sources. Directory

Flags:`

var (
	outDir = flag.String("o", "", "Output directory path")
	check  = flag.Bool("c", false, "Check only")
	debug  = flag.Bool("debug", false, "Output debug log")
	check  = flag.Bool("check", false, "Check only")
)

func exit(err error) {
	if err != nil {
		fmt.Fprintln(colorable.NewColorableStderr(), color.RedString("trygo: error:"), err)
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

	if *check {
		// Do not use trygo.NewGen() since output directory check is not necessary
		gen := &trygo.Gen{Writer: os.Stdout}
		exit(gen.Check(flag.Args()))
	}

	gen, err := trygo.NewGen(*outDir)
	if err != nil {
		exit(err)
	}

	if err := gen.Generate(flag.Args(), *debug); err != nil {
		exit(err)
	}
}
