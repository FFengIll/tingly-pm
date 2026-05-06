package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/FFengIll/tingly-pm/board"
)

func runInit(args []string) {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	common := addCommonFlags(fs)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: pm init [--dir DIR]")
		fs.PrintDefaults()
	}
	parseInterspersed(fs, args)

	pm := common.pmDir()
	if err := board.EnsureInit(pm); err != nil {
		fail(err)
	}
	printOK(fmt.Sprintf("Initialized %s", pm), common.json, map[string]any{"path": pm})
}
