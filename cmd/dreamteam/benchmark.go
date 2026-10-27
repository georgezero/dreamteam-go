package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"

	"github.com/georgezero/dreamteam-go/internal/bench"
)

const benchmarkUsage = `usage: dreamteam benchmark run CONFIG [-out DIR]
       dreamteam benchmark report RESULTS [-format text|json]
       dreamteam benchmark verify RESULTS [-t90]
       dreamteam benchmark reproduce RESULTS [-out DIR]`

// benchmarkCmd is the agent evaluation harness (docs/BENCHMARK.md): run a
// configuration's matrix of policies, scenarios and seeds through the
// agent protocol; report a results directory; verify its integrity.
func benchmarkCmd(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, benchmarkUsage)
		return exitInvalid
	}
	switch args[0] {
	case "run":
		return benchmarkRun(args[1:], stdout, stderr)
	case "report":
		return benchmarkReport(args[1:], stdout, stderr)
	case "verify":
		return benchmarkVerify(args[1:], stdout, stderr)
	case "reproduce":
		return benchmarkReproduce(args[1:], stdout, stderr)
	}
	fmt.Fprintf(stderr, "dreamteam benchmark: unknown subcommand %q\n%s\n", args[0], benchmarkUsage)
	return exitInvalid
}

// benchmarkRun validates the configuration, runs every run, and prints
// the report. Exit 0: every run completed; 1: a run failed (it is
// classified in the results) or the results could not be written; 2: the
// configuration is invalid (nothing ran).
func benchmarkRun(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("benchmark run", flag.ContinueOnError)
	fs.SetOutput(stderr)
	out := fs.String("out", "", "results directory (default: the configuration's output, relative to its file)")
	pos, err := parseFlags(fs, args)
	if err != nil || len(pos) != 1 {
		fmt.Fprintln(stderr, benchmarkUsage)
		return exitInvalid
	}
	c, raw, err := bench.LoadConfig(pos[0])
	if err != nil {
		fmt.Fprintln(stderr, "dreamteam:", err)
		if errors.Is(err, bench.ErrConfig) {
			return exitInvalid
		}
		return exitError
	}
	dir := *out
	if dir == "" {
		dir = filepath.Join(filepath.Dir(pos[0]), c.Output)
	}
	fmt.Fprintf(stderr, "benchmark %s: %d runs into %s\n", c.ID, len(c.Scenarios)*len(c.Seeds)*len(c.Policies), dir)
	m, err := bench.Matrix(c, raw, dir, stderr)
	if err != nil {
		fmt.Fprintln(stderr, "dreamteam:", err)
		return exitError
	}
	_, _, recs, err := bench.LoadResults(dir)
	if err != nil {
		fmt.Fprintln(stderr, "dreamteam:", err)
		return exitError
	}
	fmt.Fprint(stdout, bench.BuildReport(c, recs).Text())
	for _, r := range m.Runs {
		if r.Status != bench.StatusCompleted {
			return exitError
		}
	}
	return exitOK
}

func benchmarkReport(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("benchmark report", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "text or json")
	pos, err := parseFlags(fs, args)
	if err != nil || len(pos) != 1 || (*format != "text" && *format != "json") {
		fmt.Fprintln(stderr, benchmarkUsage)
		return exitInvalid
	}
	_, c, recs, err := bench.LoadResults(pos[0])
	if err != nil {
		fmt.Fprintln(stderr, "dreamteam:", err)
		return exitError
	}
	r := bench.BuildReport(c, recs)
	if *format == "json" {
		raw, err := json.MarshalIndent(r, "", " ")
		if err != nil {
			fmt.Fprintln(stderr, "dreamteam:", err)
			return exitError
		}
		fmt.Fprintf(stdout, "%s\n", raw)
		return exitOK
	}
	fmt.Fprint(stdout, r.Text())
	return exitOK
}

// benchmarkReproduce reruns a results directory's configuration and
// compares the deterministic files with its manifest, one line per check.
// Exit 0: identical; 3: something differs; 1: the rerun could not be made.
func benchmarkReproduce(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("benchmark reproduce", flag.ContinueOnError)
	fs.SetOutput(stderr)
	out := fs.String("out", "", "where the rerun's results go (default: RESULTS.reproduce)")
	pos, err := parseFlags(fs, args)
	if err != nil || len(pos) != 1 {
		fmt.Fprintln(stderr, benchmarkUsage)
		return exitInvalid
	}
	dir := *out
	if dir == "" {
		dir = filepath.Clean(pos[0]) + ".reproduce"
	}
	fmt.Fprintf(stderr, "rerunning %s into %s\n", pos[0], dir)
	checks, err := bench.Reproduce(pos[0], dir, stderr)
	if err != nil {
		fmt.Fprintln(stderr, "dreamteam:", err)
		return exitError
	}
	return printChecks(stdout, checks, "(a rerun compared with the recorded manifest; timing, standard error and policy evidence are not compared)")
}

// printChecks prints one line per check and a summary. Exit 0: every
// check passed; 3: one failed.
func printChecks(stdout io.Writer, checks []bench.Check, note string) int {
	failed := 0
	for _, c := range checks {
		if c.OK {
			fmt.Fprintf(stdout, "ok    %s\n", c.What)
		} else {
			failed++
			fmt.Fprintf(stdout, "FAIL  %s: %s\n", c.What, c.Note)
		}
	}
	fmt.Fprintf(stdout, "%d checks, %d failed %s\n", len(checks), failed, note)
	if failed > 0 {
		return exitReplay
	}
	return exitOK
}

// benchmarkVerify prints one line per check. Exit 0: every check passed;
// 3: one failed.
func benchmarkVerify(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("benchmark verify", flag.ContinueOnError)
	fs.SetOutput(stderr)
	t90 := fs.Bool("t90", false, "also recompute each evaluated run's T90 (two simulations per run)")
	pos, err := parseFlags(fs, args)
	if err != nil || len(pos) != 1 {
		fmt.Fprintln(stderr, benchmarkUsage)
		return exitInvalid
	}
	checks, err := bench.Verify(pos[0], bench.VerifyOptions{T90: *t90})
	if err != nil {
		fmt.Fprintln(stderr, "dreamteam:", err)
		return exitError
	}
	return printChecks(stdout, checks, "(an integrity check of these files, not an independent certification)")
}
