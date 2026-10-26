// Command dreamteam runs the DreamTeam outpatient radiology practice
// simulator.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/georgezero/dreamteam-go/internal/scenario"
	"github.com/georgezero/dreamteam-go/internal/session"
	"github.com/georgezero/dreamteam-go/internal/tui"
	"github.com/georgezero/dreamteam-go/internal/version"
)

func main() {
	args := os.Args[1:]
	cmd := "play"
	if len(args) > 0 && len(args[0]) > 0 && args[0][0] != '-' {
		cmd, args = args[0], args[1:]
	}
	switch cmd {
	case "play":
		os.Exit(play(args))
	case "scenario":
		os.Exit(scenarioCmd(args))
	case "run":
		os.Exit(runCmd(args, os.Stdout, os.Stderr))
	case "replay":
		os.Exit(replayCmd(args, os.Stdout, os.Stderr, false))
	case "verify":
		os.Exit(replayCmd(args, os.Stdout, os.Stderr, true))
	case "t90":
		os.Exit(t90Cmd(args, os.Stdout, os.Stderr))
	case "agent":
		os.Exit(agentCmd(args, os.Stdin, os.Stdout, os.Stderr))
	case "benchmark":
		os.Exit(benchmarkCmd(args, os.Stdout, os.Stderr))
	case "version":
		fmt.Println("dreamteam", version.String())
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "dreamteam: unknown command %q\n\n", cmd)
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `Usage:
  dreamteam [play] [-scenario ID|FILE] [-seed N] [-load FILE] [-save FILE]
                                                   play in the terminal (default); -load resumes
                                                   a saved session; ctrl+s saves (to -save, the
                                                   loaded file, or SCENARIO-seedN.session.json)
  dreamteam run [-scenario ID|FILE] [-seed N] -days D [-actions FILE] [-save FILE] [-log FILE] [OUTPUT]
                                                   run headless from the takeover for D days,
                                                   applying timestamped action batches; -log
                                                   writes the observable event log from the
                                                   takeover as JSON lines (seq, time, severity,
                                                   category, text)
  dreamteam replay FILE [-allow-engine-mismatch] [OUTPUT]
                                                   rebuild a saved session from its scenario,
                                                   seed and actions; report every mismatch
  dreamteam verify FILE [-allow-engine-mismatch]   the same, quietly: "ok <run digest>"
  dreamteam t90 FILE [-format text|json]         the T90 evaluation of a saved session's plan
                                                   (the sustainable scenario): the metric vector,
                                                   service gates and verdict of the evaluation
                                                   policy in force, against no action on the same
                                                   seed; days not yet played are projected
  dreamteam agent --stdio [-scenario ID|FILE] [-seed N] [-load FILE]
                                                   the agent protocol (docs/AGENT_PROTOCOL.md): one
                                                   JSON request per line on stdin, one JSON response
                                                   per line on stdout; diagnostics on stderr
  dreamteam agent -policy noop|greedy [-days N] [-transcript FILE] [-save FILE] [...]
                                                   a scripted policy through the same protocol (an
                                                   interface check, not a strategy)
  dreamteam benchmark run CONFIG [-out DIR]        run a benchmark (docs/BENCHMARK.md): every policy
                                                   on every scenario and seed through the agent
                                                   protocol, each session saved, replayed and measured
  dreamteam benchmark report RESULTS [-format text|json]
                                                   compare a results directory's runs
  dreamteam benchmark verify RESULTS [-t90]        check a results directory's hashes, replay its
                                                   sessions and rebuild its results and report
  OUTPUT (run, replay):
    -format text|json       text summary (default) or the canonical result JSON on stdout
                            (-summary is an alias)
    -verify-replay          (run) also replay the run's own record and include the outcome
    -golden FILE            compare the result with a golden reference
    -write-golden FILE      write the result as a golden reference (refuses to replace one
                            unless -overwrite)
  Exit codes: 0 ok · 1 error (I/O, scenario) · 2 invalid input · 3 replay failure
              · 4 golden mismatch · 5 incompatible or unsupported format
  dreamteam scenario list | validate ID|FILE | coverage ID|FILE
                                                   bundled scenarios; check one; show its
                                                   weekly staffing coverage
  dreamteam version                                print the version

In the terminal UI press ? for the keys. Colors follow the terminal; NO_COLOR=1 turns them off.
`)
}

func play(args []string) int {
	fs := flag.NewFlagSet("play", flag.ContinueOnError)
	seed := fs.Int64("seed", 1, "simulation seed")
	scen := fs.String("scenario", scenario.Default, "bundled scenario ID or .json file")
	load := fs.String("load", "", "resume a saved session file")
	save := fs.String("save", "", "file ctrl+s saves to")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	var s *session.Session
	if *load != "" {
		fmt.Fprintf(os.Stderr, "Replaying %s…\n", *load)
		var rep *session.Report
		var err error
		s, rep, err = session.Load(*load, session.ReplayOptions{})
		if err != nil {
			fmt.Fprintln(os.Stderr, "dreamteam:", err)
			if rep != nil {
				fmt.Fprint(os.Stderr, rep)
			}
			return 1
		}
	} else {
		f, err := scenario.ResolveFile(*scen)
		if err != nil {
			fmt.Fprintln(os.Stderr, "dreamteam:", err)
			return 1
		}
		if f.WarmupDays > 0 {
			fmt.Fprintf(os.Stderr, "Building %d days of practice history…\n", f.WarmupDays)
		}
		if s, err = session.FromFile(f, *seed); err != nil {
			fmt.Fprintln(os.Stderr, "dreamteam:", err)
			return 1
		}
	}
	m := tui.New(s)
	m.SetSavePath(*save)
	// The model changes at most 10 times a second (its frame clock) plus on
	// keys; a 20 fps renderer shows every change within 50 ms without
	// polling at the default 60.
	if _, err := tea.NewProgram(m, tea.WithFPS(20)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "dreamteam:", err)
		return 1
	}
	return 0
}

func scenarioCmd(args []string) int {
	if len(args) == 0 {
		usage()
		return 2
	}
	switch args[0] {
	case "list":
		for _, id := range scenario.List() {
			fmt.Println(id)
		}
		return 0
	case "validate", "coverage":
		if len(args) != 2 {
			usage()
			return 2
		}
		cfg, err := scenario.Resolve(args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if args[0] == "validate" {
			fmt.Printf("%s: ok (%d rooms, %d employees)\n", cfg.ID, len(cfg.Rooms), len(cfg.Employees))
		} else {
			fmt.Print(scenario.Report(cfg))
		}
		return 0
	}
	usage()
	return 2
}
