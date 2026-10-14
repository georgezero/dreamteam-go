// Command dreamteam runs the DreamTeam outpatient radiology practice
// simulator.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/scenario"
	"github.com/georgezero/dreamteam-go/internal/session"
	"github.com/georgezero/dreamteam-go/internal/simtime"
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
		os.Exit(runCmd(args, os.Stdout))
	case "replay":
		os.Exit(replayCmd(args, os.Stdout, false))
	case "verify":
		os.Exit(replayCmd(args, os.Stdout, true))
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
  dreamteam run [-scenario ID|FILE] [-seed N] -days D [-actions FILE] [-save FILE]
                                                   run headless from the takeover for D days,
                                                   applying timestamped action batches; print
                                                   a summary and the digests
  dreamteam replay FILE [-allow-engine-mismatch]   rebuild a saved session from its scenario,
                                                   seed and actions; report every mismatch
  dreamteam verify FILE [-allow-engine-mismatch]   the same, quietly: exit 0 if reproduced
  dreamteam scenario list                          list bundled scenarios
  dreamteam scenario validate ID|FILE              check a scenario
  dreamteam scenario coverage ID|FILE              staffed vs open hours, fortnight rota
  dreamteam version                                print the version
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
	if _, err := tea.NewProgram(m).Run(); err != nil {
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

// batchFile is one timestamped decision in a run's actions file.
type batchFile struct {
	Time    string       `json:"time"` // practice-local ISO time
	Actions []api.Action `json:"actions"`
}

// runCmd runs a scenario headless from its takeover.
func runCmd(args []string, out io.Writer) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	seed := fs.Int64("seed", 1, "simulation seed")
	scen := fs.String("scenario", scenario.Default, "bundled scenario ID or .json file")
	days := fs.Int64("days", 1, "days to run from the takeover")
	actions := fs.String("actions", "", "JSON file of action batches: [{\"time\": \"2026-10-13T07:00:00\", \"actions\": [...]}]")
	save := fs.String("save", "", "save the session to this file")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	var batches []batchFile
	if *actions != "" {
		raw, err := os.ReadFile(*actions)
		if err == nil {
			dec := json.NewDecoder(bytes.NewReader(raw))
			dec.DisallowUnknownFields()
			err = dec.Decode(&batches)
			if err == nil && dec.Decode(&struct{}{}) != io.EOF {
				err = fmt.Errorf("data after the list of batches")
			}
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "dreamteam: actions:", err)
			return 1
		}
	}
	s, err := session.Open(*scen, *seed)
	if err != nil {
		fmt.Fprintln(os.Stderr, "dreamteam:", err)
		return 1
	}
	cal := s.Calendar()
	end := s.Now().Add(simtime.Duration(*days) * simtime.Day)
	for _, b := range batches {
		at, err := cal.ParseISO(b.Time)
		if err != nil || at < s.Now() || at > end {
			fmt.Fprintf(os.Stderr, "dreamteam: batch time %q must be a time from %s to %s, in order\n", b.Time, cal.ISO(s.Now()), cal.ISO(end))
			return 1
		}
		s.AdvanceTo(at)
		for _, r := range s.Apply(b.Actions...) {
			mark := "ok "
			if !r.OK {
				mark = "REJ"
			}
			fmt.Fprintf(out, "%s %s %s %s: %s\n", r.ID, r.Time, mark, r.Action.Type, r.Message)
		}
	}
	s.AdvanceTo(end)
	ev := s.Evaluation()
	o := s.Observe(0)
	fmt.Fprintf(out, "scenario %s · seed %d · takeover %s · now %s\n", s.ScenarioID(), s.Seed(), ev.Baseline.Time, o.Time)
	fmt.Fprintf(out, "since takeover: %d exams · %d orders, %d lost · operating profit %s (baseline-adjusted) · cash %s\n",
		ev.Ops.Exams, ev.Ops.Demand.Received, ev.Ops.Demand.Lost, cents(ev.Money.OperatingProfitCents), cents(ev.Money.CashEndCents))
	d := s.Digests()
	fmt.Fprintf(out, "digests: run %s\n         log %s\n         ledger %s\n         actions %s\n         state %s\n",
		d.Run, d.Log, d.Ledger, d.Actions, d.State)
	if *save != "" {
		if err := s.Save(*save); err != nil {
			fmt.Fprintln(os.Stderr, "dreamteam:", err)
			return 1
		}
		fmt.Fprintf(out, "saved %s\n", *save)
	}
	return 0
}

func cents(c int64) string {
	sign := ""
	if c < 0 {
		sign, c = "-", -c
	}
	return fmt.Sprintf("%s$%d.%02d", sign, c/100, c%100)
}

// replayCmd rebuilds a saved session and checks it.
func replayCmd(args []string, out io.Writer, quiet bool) int {
	name := "replay"
	if quiet {
		name = "verify"
	}
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	allow := fs.Bool("allow-engine-mismatch", false, "replay a session recorded by another build")
	// Flags may follow the file name.
	var file string
	for len(args) > 0 {
		if err := fs.Parse(args); err != nil {
			return 2
		}
		if fs.NArg() == 0 {
			break
		}
		if file != "" {
			usage()
			return 2
		}
		file, args = fs.Arg(0), fs.Args()[1:]
	}
	if file == "" {
		usage()
		return 2
	}
	r, err := session.ReadRecord(file)
	if err != nil {
		fmt.Fprintln(out, "dreamteam:", err)
		return 1
	}
	if !quiet {
		fmt.Fprintf(out, "%s · scenario %s (%.12s…) · seed %d · recorded by %s\n", file, r.Scenario.ID, r.Scenario.Fingerprint, r.Seed, r.Engine)
		fmt.Fprintf(out, "takeover %s → saved %s · %d batches\n", r.Start, r.Until, len(r.Batches))
	}
	_, rep, err := session.Replay(r, session.ReplayOptions{AllowEngineMismatch: *allow})
	if err != nil {
		fmt.Fprintln(out, "dreamteam:", err)
		if rep != nil && len(rep.Mismatches) > 0 {
			fmt.Fprint(out, rep)
		}
		return 1
	}
	if quiet {
		fmt.Fprintf(out, "ok %s\n", rep.Replayed.Run)
	} else {
		fmt.Fprint(out, rep)
	}
	return 0
}
