package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/FFengIll/tingly-pm/board"
)

const memberUsage = `Usage: pm member <subcommand> [flags]

Subcommands:
  add <name>      Create or update a member (upsert)
  list            List members (filterable by --type)
  search          Search members by labels (--labels) or name (--query)
  remove <name>   Remove a member
`

func runMember(args []string) {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, memberUsage)
		os.Exit(2)
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "-h", "--help", "help":
		fmt.Print(memberUsage)
	case "add":
		memberAdd(rest)
	case "list":
		memberList(rest)
	case "search":
		memberSearch(rest)
	case "remove":
		memberRemove(rest)
	default:
		fmt.Fprintf(os.Stderr, "unknown member subcommand: %s\n\n%s", sub, memberUsage)
		os.Exit(2)
	}
}

func memberAdd(args []string) {
	fs := flag.NewFlagSet("member add", flag.ExitOnError)
	common := addCommonFlags(fs)
	mtype := fs.String("type", "", "human|agent (required when creating new)")
	labels := fs.String("labels", "", "comma-separated capability labels")
	by := fs.String("by", "agent", "actor recorded in the timeline event")
	pos := parseInterspersed(fs, args)
	if len(pos) < 1 {
		failf("member name is required: pm member add <name>")
	}
	name := pos[0]
	pm := common.requireInit()

	// Detect existing for accurate timeline event + message
	existing, _ := board.QueryMembers(pm, "", "")
	exists := false
	for _, m := range existing {
		if m.Name == name {
			exists = true
			break
		}
	}

	if err := board.UpsertMember(pm, name, *mtype, splitLabels(*labels)); err != nil {
		fail(err)
	}

	if exists {
		board.AppendEvent(pm, &board.TimelineEvent{Event: "member_updated", Name: name, By: *by})
		printOK(fmt.Sprintf("Updated %s", name), common.json,
			map[string]any{"name": name, "created": false})
	} else {
		board.AppendEvent(pm, &board.TimelineEvent{Event: "member_registered", Name: name, Type: *mtype, By: *by})
		printOK(fmt.Sprintf("Registered %s (%s)", name, *mtype), common.json,
			map[string]any{"name": name, "type": *mtype, "created": true})
	}
}

func memberList(args []string) {
	fs := flag.NewFlagSet("member list", flag.ExitOnError)
	common := addCommonFlags(fs)
	mtype := fs.String("type", "", "filter: human|agent")
	parseInterspersed(fs, args)
	pm := common.requireInit()
	members, err := board.QueryMembers(pm, "", *mtype)
	if err != nil {
		fail(err)
	}
	printMembers(members, common.json)
}

func memberSearch(args []string) {
	fs := flag.NewFlagSet("member search", flag.ExitOnError)
	common := addCommonFlags(fs)
	labels := fs.String("labels", "", "comma-separated labels (any-match, fuzzy)")
	query := fs.String("query", "", "case-insensitive substring match on name or labels")
	mtype := fs.String("type", "", "limit search to type: human|agent")
	parseInterspersed(fs, args)
	pm := common.requireInit()

	// --query uses the unified QueryMembers; --labels keeps the legacy
	// any-of-labels semantics from board.SearchMembers.
	var (
		members []board.Member
		err     error
	)
	if *labels != "" {
		members, err = board.SearchMembers(pm, *labels)
	} else {
		members, err = board.QueryMembers(pm, *query, *mtype)
	}
	if err != nil {
		fail(err)
	}
	printMembers(members, common.json)
}

func memberRemove(args []string) {
	fs := flag.NewFlagSet("member remove", flag.ExitOnError)
	common := addCommonFlags(fs)
	by := fs.String("by", "agent", "actor recorded in the timeline event")
	pos := parseInterspersed(fs, args)
	if len(pos) < 1 {
		failf("member name is required: pm member remove <name>")
	}
	name := pos[0]
	pm := common.requireInit()

	if err := board.RemoveMember(pm, name); err != nil {
		fail(err)
	}
	board.AppendEvent(pm, &board.TimelineEvent{Event: "member_removed", Name: name, By: *by})
	printOK(fmt.Sprintf("Removed member %s", name), common.json,
		map[string]any{"name": name})
}
