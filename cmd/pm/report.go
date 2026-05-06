package main

import (
	"flag"

	"github.com/FFengIll/tingly-pm/board"
)

func runReport(args []string) {
	fs := flag.NewFlagSet("report", flag.ExitOnError)
	common := addCommonFlags(fs)
	pos := parseInterspersed(fs, args)
	reportType := "daily"
	if len(pos) > 0 {
		reportType = pos[0]
	}
	pm := common.requireInit()
	out, err := board.GenerateReport(pm, reportType)
	if err != nil {
		fail(err)
	}
	printText(out, common.json)
}

func runSummary(args []string) {
	fs := flag.NewFlagSet("summary", flag.ExitOnError)
	common := addCommonFlags(fs)
	parseInterspersed(fs, args)
	pm := common.requireInit()
	out, err := board.GenerateSummary(pm)
	if err != nil {
		fail(err)
	}
	printText(out, common.json)
}

func runTimeline(args []string) {
	fs := flag.NewFlagSet("timeline", flag.ExitOnError)
	common := addCommonFlags(fs)
	limit := fs.Int("limit", 20, "max number of events (newest first)")
	parseInterspersed(fs, args)
	pm := common.requireInit()
	events, err := board.ListEvents(pm, *limit)
	if err != nil {
		// timeline.jsonl may not exist yet — render as empty
		printEvents(nil, common.json)
		return
	}
	printEvents(events, common.json)
}
