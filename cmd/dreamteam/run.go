package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/session"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Exit codes (documented in usage).
const (
	exitOK           = 0
	exitError        = 1 // I/O, scenario errors
	exitInvalid      = 2 // invalid flags or inputs
	exitReplay       = 3 // a session did not replay
	exitMismatch     = 4 // the result differs from the golden reference
	exitIncompatible = 5 // incompatible golden or session, unsupported format
)

// output holds the result flags shared by run and replay.
type output struct {
	format, golden, writeGolden string
	migrations                  string
	overwrite, allowEngine      bool
}

func (o *output) flags(fs *flag.FlagSet) {
	fs.StringVar(&o.format, "format", "text", "text or json")
	fs.StringVar(&o.golden, "golden", "", "compare the result with this golden reference")
	fs.StringVar(&o.writeGolden, "write-golden", "", "write the result as a golden reference")
	fs.BoolVar(&o.overwrite, "overwrite", false, "let -write-golden replace an existing file")
	fs.BoolVar(&o.allowEngine, "allow-engine-mismatch", false, "accept a golden or session from another build (reported)")
	fs.StringVar(&o.migrations, "golden-migrations", "", "migration manifest: fields of the -golden reference changed by a documented schema change")
}

func (o *output) check() error {
	if o.format != "text" && o.format != "json" {
		return fmt.Errorf("unsupported -format %q (text or json)", o.format)
	}
	return nil
}

// emit writes the result (text or canonical JSON on stdout), then writes
// or compares the golden reference. Diagnostics go to stderr.
func (o *output) emit(r *session.Result, stdout, stderr io.Writer) int {
	raw, err := r.Encode()
	if err != nil {
		fmt.Fprintln(stderr, "dreamteam:", err)
		return exitError
	}
	if o.format == "json" {
		stdout.Write(raw)
	} else {
		writeSummary(stdout, r)
	}
	code := exitOK
	if r.Replay != nil && !r.Replay.Verified {
		fmt.Fprintln(stderr, "dreamteam: the run's own record did not replay")
		code = exitReplay
	}
	if o.writeGolden != "" {
		if _, err := os.Stat(o.writeGolden); err == nil && !o.overwrite {
			fmt.Fprintf(stderr, "dreamteam: %s exists; golden references are replaced only with -overwrite\n", o.writeGolden)
			return exitInvalid
		}
		if err := os.WriteFile(o.writeGolden, raw, 0o644); err != nil {
			fmt.Fprintln(stderr, "dreamteam:", err)
			return exitError
		}
		fmt.Fprintf(stderr, "wrote golden reference %s\n", o.writeGolden)
	}
	if o.golden != "" {
		want, err := os.ReadFile(o.golden)
		if err != nil {
			fmt.Fprintln(stderr, "dreamteam:", err)
			return exitError
		}
		var migrated map[string]session.FieldChange
		if o.migrations != "" {
			raw, err := os.ReadFile(o.migrations)
			if err != nil {
				fmt.Fprintln(stderr, "dreamteam:", err)
				return exitError
			}
			m, err := session.DecodeMigrations(raw)
			if err != nil {
				fmt.Fprintln(stderr, "dreamteam:", err)
				return exitIncompatible
			}
			migrated = m.References[filepath.Base(o.golden)]
		}
		c, err := session.CompareMigrated(want, r, o.allowEngine, migrated)
		if c != nil {
			fmt.Fprint(stderr, c)
		}
		switch {
		case errors.Is(err, session.ErrGoldenMismatch):
			return exitMismatch
		case err != nil:
			fmt.Fprintln(stderr, "dreamteam:", err)
			return exitIncompatible
		}
	}
	return code
}

// writeSummary is the concise human-readable result.
func writeSummary(w io.Writer, r *session.Result) {
	fmt.Fprintf(w, "scenario %s (%.12s…) · seed %d · engine %s\n", r.Scenario.ID, r.Scenario.Fingerprint, r.Seed, r.Engine)
	fmt.Fprintf(w, "takeover %s → end %s (%d days)\n", r.Takeover.Time, r.End.Time, r.PlayDays)
	a := r.Actions
	fmt.Fprintf(w, "actions: %d attempted, %d accepted, %d rejected\n", a.Attempted, a.Accepted, a.Rejected)
	for _, x := range a.Results {
		mark := "ok "
		if !x.OK {
			mark = "REJ"
		}
		fmt.Fprintf(w, "  %s %s %s %s (%s)\n", x.ID, x.Time, mark, x.Act.Type, x.Code)
	}
	ops, m := r.Window.Ops, r.Window.Money
	fmt.Fprintf(w, "since takeover: %d exams · %d orders, %d lost · walk-ins %d (%d left) · wait p90 %dm · overtime %dm\n",
		ops.Exams, ops.Demand.Received, ops.Demand.Lost, ops.WalkIns.Arrived, ops.WalkIns.Left,
		ops.Waits.ToExam.P90Sec/60, ops.Workforce.OvertimeSec/60)
	fmt.Fprintf(w, "money: net revenue %s · expenses %s · operating profit %s (baseline-adjusted; as posted %s)\n",
		cents(m.NetRevenueCents), cents(m.OperatingExpenseCents), cents(m.OperatingProfitCents), cents(m.Ledger.OperatingProfitCents))
	fmt.Fprintf(w, "cash: collected %s · paid %s · end %s (low %s) · bridge reconciles: %v\n",
		cents(m.CollectionsCents), cents(m.DisbursementsCents), cents(m.CashEndCents), cents(m.MinCashCents), m.Reconciled)
	b := r.Balances
	fmt.Fprintf(w, "balances: A/R %s · A/P %s · payroll owed %s · open claims %d · unread %d · booked ahead %d\n",
		cents(b.ReceivablesCents), cents(b.PayablesCents), cents(b.AccruedPayrollCents), r.Obligations.OpenClaims,
		r.Obligations.Unread, r.Obligations.BookedAhead)
	fmt.Fprintf(w, "alerts: %d warnings, %d alerts raised; %d active at the end\n", r.Alerts.RaisedWarn, r.Alerts.RaisedAlert, r.Alerts.ActiveAtEnd)
	if r.Replay != nil {
		fmt.Fprintf(w, "replay: verified %v (%d batches)\n", r.Replay.Verified, r.Replay.Batches)
	}
	fmt.Fprintf(w, "digest: run %s\n", r.Digests.Run)
}

func cents(c int64) string {
	sign := ""
	if c < 0 {
		sign, c = "-", -c
	}
	s := fmt.Sprint(c / 100)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return fmt.Sprintf("%s$%s.%02d", sign, s, c%100)
}

// batchFile is one timestamped decision in a run's actions file.
type batchFile struct {
	Time    string       `json:"time"` // practice-local ISO time
	Actions []api.Action `json:"actions"`
}

// parseFlags parses flags that may come before or after one positional
// argument (the file, for replay and verify).
func parseFlags(fs *flag.FlagSet, args []string) (pos []string, err error) {
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		if fs.NArg() == 0 {
			return pos, nil
		}
		pos = append(pos, fs.Arg(0))
		args = fs.Args()[1:]
	}
}

// runCmd runs a scenario headless from its takeover.
func runCmd(args []string, stdout, stderr io.Writer) (code int) {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(stderr)
	seed := fs.Int64("seed", 1, "simulation seed")
	scen := fs.String("scenario", "established", "bundled scenario ID or .json file")
	days := fs.Int64("days", 1, "days to run from the takeover")
	actions := fs.String("actions", "", "JSON file of action batches: [{\"time\": \"2026-10-13T07:00:00\", \"actions\": [...]}]")
	save := fs.String("save", "", "save the session to this file")
	verify := fs.Bool("verify-replay", false, "replay the run's own record and include the outcome")
	logFile := fs.String("log", "", "write the observable event log from the takeover to this file, one JSON object per line")
	var o output
	o.flags(fs)
	fs.StringVar(&o.format, "summary", "text", "alias of -format: text or json")
	if pos, err := parseFlags(fs, args); err != nil || len(pos) > 0 {
		return exitInvalid
	}
	if err := o.check(); err != nil {
		fmt.Fprintln(stderr, "dreamteam:", err)
		return exitIncompatible
	}
	if *days < 0 || *days > 3650 {
		fmt.Fprintln(stderr, "dreamteam: -days must be 0–3650")
		return exitInvalid
	}
	var batches []batchFile
	if *actions != "" {
		raw, err := os.ReadFile(*actions)
		if err != nil {
			fmt.Fprintln(stderr, "dreamteam:", err)
			return exitError
		}
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.DisallowUnknownFields()
		err = dec.Decode(&batches)
		if err == nil && dec.Decode(&struct{}{}) != io.EOF {
			err = fmt.Errorf("data after the list of batches")
		}
		if err != nil {
			fmt.Fprintln(stderr, "dreamteam: actions:", err)
			return exitInvalid
		}
	}
	s, err := session.Open(*scen, *seed)
	if err != nil {
		fmt.Fprintln(stderr, "dreamteam:", err)
		return exitError
	}
	if *logFile != "" {
		f, err := os.Create(*logFile)
		if err != nil {
			fmt.Fprintln(stderr, "dreamteam:", err)
			return exitError
		}
		bw := bufio.NewWriter(f)
		enc := json.NewEncoder(bw)
		var werr error
		s.SetLogSink(func(e api.LogEntry) {
			if werr == nil {
				werr = enc.Encode(e)
			}
		})
		defer func() {
			s.SetLogSink(nil)
			if err := bw.Flush(); werr == nil {
				werr = err
			}
			if err := f.Close(); werr == nil {
				werr = err
			}
			if werr != nil {
				fmt.Fprintln(stderr, "dreamteam: -log:", werr)
				code = exitError
			}
		}()
	}
	cal := s.Calendar()
	end := s.Now().Add(simtime.Duration(*days) * simtime.Day)
	for _, b := range batches {
		at, err := cal.ParseISO(b.Time)
		if err != nil || at < s.Now() || at > end || len(b.Actions) == 0 {
			fmt.Fprintf(stderr, "dreamteam: batch %q: needs a time from %s to %s, in order, and at least one action\n",
				b.Time, cal.ISO(s.Now()), cal.ISO(end))
			return exitInvalid
		}
		s.AdvanceTo(at)
		s.Apply(b.Actions...)
	}
	s.AdvanceTo(end)
	if *save != "" {
		if err := s.Save(*save); err != nil {
			fmt.Fprintln(stderr, "dreamteam:", err)
			return exitError
		}
		fmt.Fprintf(stderr, "saved %s\n", *save)
	}
	r, err := s.Result(*verify)
	if err != nil {
		fmt.Fprintln(stderr, "dreamteam:", err)
		return exitError
	}
	return o.emit(r, stdout, stderr)
}

// replayCmd rebuilds a saved session and checks it; replay also emits the
// replayed run's result.
func replayCmd(args []string, stdout, stderr io.Writer, quiet bool) int {
	name := "replay"
	if quiet {
		name = "verify"
	}
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	var o output
	o.flags(fs)
	pos, err := parseFlags(fs, args)
	if err != nil || len(pos) != 1 {
		fmt.Fprintf(stderr, "usage: dreamteam %s FILE [flags]\n", name)
		return exitInvalid
	}
	if err := o.check(); err != nil {
		fmt.Fprintln(stderr, "dreamteam:", err)
		return exitIncompatible
	}
	file := pos[0]
	r, err := session.ReadRecord(file)
	switch {
	case errors.Is(err, session.ErrIncompatible):
		fmt.Fprintln(stderr, "dreamteam:", err)
		return exitIncompatible
	case err != nil:
		fmt.Fprintln(stderr, "dreamteam:", err)
		return exitReplay
	}
	if !quiet {
		fmt.Fprintf(stderr, "%s · scenario %s (%.12s…) · seed %d · recorded by %s · %d batches\n",
			file, r.Scenario.ID, r.Scenario.Fingerprint, r.Seed, r.Engine, len(r.Batches))
	}
	s, rep, err := session.Replay(r, session.ReplayOptions{AllowEngineMismatch: o.allowEngine})
	if err != nil {
		fmt.Fprintln(stderr, "dreamteam:", err)
		if rep != nil && len(rep.Mismatches) > 0 {
			fmt.Fprint(stderr, rep)
		}
		if errors.Is(err, session.ErrIncompatible) {
			return exitIncompatible
		}
		return exitReplay
	}
	switch c := rep.Compatibility(); {
	case r.Engine == s.Engine():
	case rep.Earlier || rep.StateChanged || rep.Prefix != nil && c != "native_exact":
		// A registered earlier build: replayed in its own batch mode.
		fmt.Fprintf(stderr, "compatibility: %s — recorded by %s, replayed by %s in its own batch mode\n", c, r.Engine, s.Engine())
	default:
		fmt.Fprintf(stderr, "compatible: NO — recorded by %s, replayed by %s (allowed); verification: digests match\n", r.Engine, s.Engine())
	}
	if quiet {
		fmt.Fprintf(stdout, "ok %s\n", rep.Replayed.Run)
		return exitOK
	}
	fmt.Fprint(stderr, strings.TrimRight(rep.String(), "\n")+"\n")
	res, err := s.Result(false)
	if err != nil {
		fmt.Fprintln(stderr, "dreamteam:", err)
		return exitError
	}
	res.Replay = &session.ResultReplay{Verified: true, Batches: rep.Batches, Run: rep.Replayed.Run}
	return o.emit(res, stdout, stderr)
}
