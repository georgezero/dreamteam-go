package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/session"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
	"github.com/georgezero/dreamteam-go/internal/version"
)

// Step 15: golden references (testdata/golden, see its README) and the
// result format's guarantees.

const goldenDir = "../../testdata/golden"

var references = []struct {
	file string
	args []string
	long bool
}{
	{"established-seed1-7d.json", []string{"-scenario", "established", "-seed", "1", "-days", "7"}, false},
	{"established-seed1-90d.json", []string{"-scenario", "established", "-seed", "1", "-days", "90"}, true},
	{"mini-seed1-7d.json", []string{"-scenario", "mini", "-seed", "1", "-days", "7"}, false},
	{"established-seed1-7d-actions.json", []string{"-scenario", "established", "-seed", "1", "-days", "7",
		"-actions", goldenDir + "/established-7d.actions.json"}, false},
	// V0.2 Slice 2: hire and terminate (written by 0.2.0-dev.2; its mixed
	// batch's message changed in Slice 3: MIGRATIONS.json).
	{"established-seed1-60d-staffing.json", []string{"-scenario", "established", "-seed", "1", "-days", "60",
		"-actions", goldenDir + "/established-staffing.actions.json"}, true},
	// V0.2 Slice 3: set_roster (written by 0.2.0-dev.3).
	{"established-seed1-60d-roster.json", []string{"-scenario", "established", "-seed", "1", "-days", "60",
		"-actions", goldenDir + "/established-roster.actions.json"}, true},
	// V0.2 Slice 4: set_reading_allocation (written by 0.2.0-dev.5).
	{"established-seed1-60d-allocation.json", []string{"-scenario", "established", "-seed", "1", "-days", "60",
		"-actions", goldenDir + "/established-allocation.actions.json"}, true},
}

func runJSON(t *testing.T, args ...string) ([]byte, string, int) {
	t.Helper()
	var out, errs bytes.Buffer
	code := runCmd(append(args, "-format", "json"), &out, &errs)
	return out.Bytes(), errs.String(), code
}

// Every pinned reference reproduces: the CLI compares its run with the
// file and reports no difference. When the engine version is the one that
// wrote the file, the output is byte-identical to it.
func TestGoldenReferences(t *testing.T) {
	for _, ref := range references {
		if ref.long && testing.Short() {
			continue
		}
		path := filepath.Join(goldenDir, ref.file)
		golden, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		out, errs, code := runJSON(t, append(ref.args, "-verify-replay", "-golden", path, "-allow-engine-mismatch",
			"-golden-migrations", filepath.Join(goldenDir, "MIGRATIONS.json"))...)
		if code != exitOK {
			t.Errorf("%s: exit %d\n%s", ref.file, code, errs)
			continue
		}
		var g struct{ Engine string }
		json.Unmarshal(golden, &g)
		if g.Engine == version.Number && !bytes.Equal(out, golden) {
			t.Errorf("%s: output not byte-identical to the reference", ref.file)
		}
	}
}

// Repeated runs give byte-identical JSON; a different seed or scenario
// gives a different result.
func TestResultsRepeatableAndDistinct(t *testing.T) {
	a, _, _ := runJSON(t, "-scenario", "mini", "-seed", "1", "-days", "3")
	b, _, _ := runJSON(t, "-scenario", "mini", "-seed", "1", "-days", "3")
	c, _, _ := runJSON(t, "-scenario", "mini", "-seed", "2", "-days", "3")
	if !bytes.Equal(a, b) || len(a) == 0 {
		t.Fatal("repeated runs differ")
	}
	var ra, rc session.Result
	json.Unmarshal(a, &ra)
	json.Unmarshal(c, &rc)
	if ra.Digests.Run == rc.Digests.Run || ra.Digests.State == rc.Digests.State {
		t.Fatal("different seeds, same result")
	}
	golden, _ := os.ReadFile(filepath.Join(goldenDir, "established-seed1-7d.json"))
	if _, err := session.Compare(golden, &ra, true); !errors.Is(err, session.ErrGoldenIncompatible) {
		t.Fatalf("a mini run compared against an established reference: %v", err)
	}
}

// editGolden rewrites one field of a golden reference.
func editGolden(t *testing.T, raw []byte, edit func(m map[string]any)) []byte {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		t.Fatal(err)
	}
	edit(m)
	out, _ := json.MarshalIndent(m, "", " ")
	return out
}

func get(m map[string]any, path ...string) map[string]any {
	for _, p := range path {
		m = m[p].(map[string]any)
	}
	return m
}

// The comparison names what changed: the first differing path with both
// values, missing and unexpected fields, the digests that differ; and it
// refuses incompatible or malformed references.
func TestGoldenComparisonReports(t *testing.T) {
	path := filepath.Join(goldenDir, "mini-seed1-7d.json")
	golden, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	out, _, code := runJSON(t, "-scenario", "mini", "-seed", "1", "-days", "7", "-verify-replay")
	if code != 0 {
		t.Fatal(code)
	}
	var r session.Result
	if err := json.Unmarshal(out, &r); err != nil {
		t.Fatal(err)
	}
	// The reference may name an older engine and state encoding (outcomes
	// unchanged, MIGRATIONS.json): stamp this build's, so each case below
	// changes exactly one thing.
	golden = editGolden(t, golden, func(m map[string]any) {
		m["engine"] = version.Number
		get(m, "digests")["state"], get(m, "digests")["run"], get(m, "replay")["runDigest"] = r.Digests.State, r.Digests.Run, r.Digests.Run
	})
	cases := []struct {
		name string
		edit func(m map[string]any)
		want error
		text string
	}{
		{"exams", func(m map[string]any) { get(m, "window", "ops")["exams"] = json.Number("1") }, session.ErrGoldenMismatch, ".window.ops.exams: golden 1, run "},
		{"cash", func(m map[string]any) { get(m, "balances")["cashCents"] = json.Number("1") }, session.ErrGoldenMismatch, ".balances.cashCents"},
		{"digest", func(m map[string]any) { get(m, "digests")["state"] = "00" }, session.ErrGoldenMismatch, "digests differ: state"},
		{"missing", func(m map[string]any) { delete(get(m, "obligations"), "recallList") }, session.ErrGoldenMismatch, ".obligations.recallList: unexpected field"},
		{"unexpected", func(m map[string]any) { get(m, "alerts")["extra"] = json.Number("1") }, session.ErrGoldenMismatch, ".alerts.extra: missing from the run"},
		{"day", func(m map[string]any) { m["calendar"] = m["calendar"].([]any)[:6] }, session.ErrGoldenMismatch, ".calendar: golden has 6 elements, run 7"},
		{"engine", func(m map[string]any) { m["engine"] = "0.1.0-dev.1" }, session.ErrGoldenIncompatible, "engine: golden 0.1.0-dev.1"},
		{"fingerprint", func(m map[string]any) { get(m, "scenario")["fingerprint"] = "ff" }, session.ErrGoldenIncompatible, "scenario.fingerprint"},
		{"seed", func(m map[string]any) { m["seed"] = json.Number("2") }, session.ErrGoldenIncompatible, "seed: golden 2"},
		{"length", func(m map[string]any) { get(m, "end")["sec"] = json.Number("1") }, session.ErrGoldenIncompatible, "end.sec"},
		{"schema", func(m map[string]any) { m["schema"] = "dreamteam.result/v0" }, session.ErrGoldenIncompatible, "schema"},
	}
	for _, c := range cases {
		cmp, err := session.Compare(editGolden(t, golden, c.edit), &r, false)
		text := ""
		if cmp != nil {
			text = cmp.String()
		}
		if !errors.Is(err, c.want) || !strings.Contains(text+errString(err), c.text) {
			t.Errorf("%s: %v\n%s", c.name, err, text)
		}
	}
	// Another engine with otherwise identical outcomes: compared when
	// allowed, and the incompatibility is still reported.
	cmp, err := session.Compare(editGolden(t, golden, func(m map[string]any) { m["engine"] = "0.1.0-dev.1" }), &r, true)
	if err != nil || !cmp.Match || cmp.Compatible || !strings.Contains(cmp.String(), "compatible: NO") {
		t.Errorf("engine allowed: %v %s", err, cmp)
	}
	for name, bad := range map[string][]byte{"truncated": golden[:len(golden)/2], "empty": nil, "array": []byte("[]"),
		"trailing": append(append([]byte{}, golden...), '}')} {
		if _, err := session.Compare(bad, &r, false); !errors.Is(err, session.ErrGoldenFormat) {
			t.Errorf("%s golden: %v", name, err)
		}
	}
	// The CLI's exit codes: mismatch 4, incompatible or malformed 5, and a
	// golden file is never replaced without -overwrite.
	dir := t.TempDir()
	changed := filepath.Join(dir, "changed.json")
	os.WriteFile(changed, editGolden(t, golden, func(m map[string]any) { get(m, "window", "ops")["exams"] = json.Number("1") }), 0o644)
	if _, errs, code := runJSON(t, "-scenario", "mini", "-days", "7", "-verify-replay", "-golden", changed); code != exitMismatch || !strings.Contains(errs, "first: .window.ops.exams") {
		t.Errorf("mismatch exit %d\n%s", code, errs)
	}
	os.WriteFile(changed, golden[:100], 0o644)
	if _, _, code := runJSON(t, "-scenario", "mini", "-days", "7", "-golden", changed); code != exitIncompatible {
		t.Errorf("malformed golden exit %d", code)
	}
	if _, _, code := runJSON(t, "-scenario", "mini", "-days", "7", "-write-golden", changed); code != exitInvalid {
		t.Errorf("overwrote a golden file without -overwrite: exit %d", code)
	}
	var o, e bytes.Buffer
	if code := runCmd([]string{"-scenario", "mini", "-days", "1", "-format", "xml"}, &o, &e); code != exitIncompatible {
		t.Errorf("unsupported format exit %d", code)
	}
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// A changed action history is detected, and rejected actions stay in the
// result.
func TestActionHistoryInResult(t *testing.T) {
	path := filepath.Join(goldenDir, "established-seed1-7d-actions.json")
	golden, _ := os.ReadFile(path)
	var g session.Result
	json.Unmarshal(golden, &g)
	var rejected []string
	for _, a := range g.Actions.Results {
		if !a.OK {
			rejected = append(rejected, a.Code)
		}
	}
	if g.Actions.Rejected != 3 || strings.Join(rejected, ",") != "unknown_employee,shift_ended,invalid_minutes" {
		t.Fatalf("rejected actions in the reference: %v", rejected)
	}
	acts, _ := os.ReadFile(filepath.Join(goldenDir, "established-7d.actions.json"))
	alt := filepath.Join(t.TempDir(), "alt.json")
	os.WriteFile(alt, bytes.Replace(acts, []byte(`"cutoff": "19:00"`), []byte(`"cutoff": "18:30"`), 1), 0o644)
	_, errs, code := runJSON(t, "-scenario", "established", "-seed", "1", "-days", "7", "-actions", alt, "-golden", path, "-allow-engine-mismatch")
	if code != exitMismatch || !strings.Contains(errs, "digests differ: log, ledger, actions, state, run") ||
		!strings.Contains(errs, ".actions.results[1].action.cutoff") {
		t.Fatalf("changed action: exit %d\n%s", code, errs)
	}
}

// The result does not depend on how the clock was advanced: one jump,
// small chunks, or the TUI's event-budget (Max speed) steps.
func TestResultIndependentOfChunkingAndSpeed(t *testing.T) {
	results := map[string][]byte{}
	for name, advance := range map[string]func(s *session.Session, to simtime.Time){
		"jump": func(s *session.Session, to simtime.Time) { s.AdvanceTo(to) },
		"chunks": func(s *session.Session, to simtime.Time) {
			for s.Now() < to {
				s.AdvanceTo(min(to, s.Now().Add(37*simtime.Minute)))
			}
		},
		"max speed": func(s *session.Session, to simtime.Time) {
			for s.Now() < to {
				s.Advance(min(simtime.Day, to.Sub(s.Now())), session.Options{EventBudget: 500})
			}
		},
		"alert stops": func(s *session.Session, to simtime.Time) {
			for s.Now() < to {
				s.Advance(to.Sub(s.Now()), session.Options{StopOnAlert: true})
			}
		},
	} {
		s, err := session.Open("mini", 4)
		if err != nil {
			t.Fatal(err)
		}
		start := s.Now()
		advance(s, start.Add(26*simtime.Hour))
		s.Apply(api.Action{Type: api.ActSetWorklistPolicy, Policy: api.WorklistFIFO})
		advance(s, start.Add(3*simtime.Day))
		r, err := s.Result(false)
		if err != nil {
			t.Fatal(err)
		}
		results[name], _ = r.Encode()
	}
	for name, raw := range results {
		if !bytes.Equal(raw, results["jump"]) {
			t.Errorf("%s: result differs from a single jump", name)
		}
	}
}

// Results hold aggregates and digests only: no patient, study, claim or
// order identifiers, no latent or future fields.
func TestResultHasNoHiddenData(t *testing.T) {
	for _, ref := range references {
		raw, _ := os.ReadFile(filepath.Join(goldenDir, ref.file))
		if m := regexp.MustCompile(`"[PSCOVA]\d{5}"`).Find(raw); m != nil {
			t.Errorf("%s: an individual record identifier %s", ref.file, m)
		}
		low := bytes.ToLower(raw)
		for _, hidden := range []string{`"draw`, `"outcome`, `"latent`, `"patience`, `"tolerance`, `"pending`, `"queue`, `"seq"`} {
			if bytes.Contains(low, []byte(hidden)) {
				t.Errorf("%s: hidden field %s", ref.file, hidden)
			}
		}
	}
}

// The reference results reconcile: the balance sheet balances, cash rolls
// forward day by day (with the takeover capital on day 1), the window's
// cash bridge holds, and the window runs from the exact takeover to the
// end.
func TestResultReconciles(t *testing.T) {
	for _, ref := range references {
		raw, _ := os.ReadFile(filepath.Join(goldenDir, ref.file))
		var r session.Result
		if err := json.Unmarshal(raw, &r); err != nil {
			t.Fatal(err)
		}
		b, m := r.Balances, r.Window.Money
		if b.CashCents+b.ReceivablesCents != b.PayablesCents+b.AccruedPayrollCents+b.SeverancePayableCents+b.EquityCents {
			t.Errorf("%s: assets ≠ liabilities + equity", ref.file)
		}
		if !m.Reconciled || m.CashEndCents != b.CashCents {
			t.Errorf("%s: window money %+v", ref.file, m)
		}
		if r.Window.From != r.Takeover || r.Window.To != r.End || r.PlayDays != (r.End.Sec-r.Takeover.Sec)/86400 {
			t.Errorf("%s: window %+v, takeover %+v, end %+v", ref.file, r.Window.From, r.Takeover, r.End)
		}
		if len(r.Calendar) != int(r.PlayDays) || !r.Calendar[0].Takeover || r.Replay == nil || !r.Replay.Verified ||
			r.Replay.Run != r.Digests.Run {
			t.Errorf("%s: %d calendar days for %d play days; replay %+v", ref.file, len(r.Calendar), r.PlayDays, r.Replay)
		}
		var exams int
		for i, d := range r.Calendar {
			exams += d.Exams
			if i > 0 {
				prev := r.Calendar[i-1].CashEndCents
				if d.CashEndCents != prev+d.CollectionsCents-d.DisbursementsCents {
					t.Errorf("%s %s: cash does not roll forward", ref.file, d.Date)
				}
			}
		}
		// The window starts at 06:00 on day 1, the calendar at midnight:
		// nothing is imaged before 06:00, so exams agree.
		if exams != r.Window.Ops.Exams {
			t.Errorf("%s: calendar exams %d, window %d", ref.file, exams, r.Window.Ops.Exams)
		}
	}
}

// V0.2 Slice 1: the historical references are kept byte-for-byte; the
// state-schema change is documented field by field in MIGRATIONS.json.
// Without the manifest exactly those fields differ; under it each must
// hold its recorded old value in the reference and new value in the run.
func TestGoldenMigrations(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(goldenDir, "MIGRATIONS.json"))
	if err != nil {
		t.Fatal(err)
	}
	m, err := session.DecodeMigrations(raw)
	if err != nil {
		t.Fatal(err)
	}
	// Every reference written by another engine is in the manifest (the
	// Slice 3 reference with no changed field); one written by this build
	// would need no migration.
	historical := 0
	for _, ref := range references {
		raw, _ := os.ReadFile(filepath.Join(goldenDir, ref.file))
		var g struct{ Engine string }
		json.Unmarshal(raw, &g)
		_, listed := m.References[ref.file]
		if g.Engine != version.Number {
			historical++
		}
		if listed != (g.Engine != version.Number) {
			t.Errorf("%s (engine %s): in the manifest %v", ref.file, g.Engine, listed)
		}
	}
	if m.StateSchema != sim.StateSchema || len(m.References) != historical || historical != 6 {
		t.Fatalf("manifest: state schema %q (build %q), %d references, %d historical", m.StateSchema, sim.StateSchema, len(m.References), historical)
	}
	golden, _ := os.ReadFile(filepath.Join(goldenDir, "mini-seed1-7d.json"))
	out, _, _ := runJSON(t, "-scenario", "mini", "-seed", "1", "-days", "7", "-verify-replay")
	var r session.Result
	json.Unmarshal(out, &r)
	mig := m.References["mini-seed1-7d.json"]

	// Without the manifest: the migrated fields, and nothing else, differ.
	c, err := session.Compare(golden, &r, true)
	if !errors.Is(err, session.ErrGoldenMismatch) || c.Total != len(mig) {
		t.Fatalf("unmigrated comparison: %v, %d differences, want %d\n%s", err, c.Total, len(mig), c)
	}
	for _, d := range c.Differences {
		if _, ok := mig[strings.SplitN(strings.TrimPrefix(d, "."), ":", 2)[0]]; !ok {
			t.Errorf("difference %s is not a migrated field", d)
		}
	}
	// Under it: identical.
	if c, err := session.CompareMigrated(golden, &r, true, mig); err != nil || !c.Match || len(c.Migrated) != len(mig) {
		t.Fatalf("migrated comparison: %v\n%s", err, c)
	}
	// A run that does not reproduce the recorded new state fails, and so
	// does a reference that is not the recorded old one.
	wrongNew := map[string]session.FieldChange{}
	for k, v := range mig {
		wrongNew[k] = v
	}
	wrongNew["digests.state"] = session.FieldChange{Old: mig["digests.state"].Old, New: "00"}
	if _, err := session.CompareMigrated(golden, &r, true, wrongNew); !errors.Is(err, session.ErrGoldenMismatch) {
		t.Errorf("a different new state digest passed: %v", err)
	}
	tampered := editGolden(t, golden, func(m map[string]any) { get(m, "digests")["state"] = "00" })
	if _, err := session.CompareMigrated(tampered, &r, true, mig); !errors.Is(err, session.ErrGoldenMismatch) {
		t.Errorf("a reference with another old state digest passed: %v", err)
	}
	// Outcomes are still compared under a migration.
	changed := editGolden(t, golden, func(m map[string]any) { get(m, "balances")["cashCents"] = json.Number("1") })
	if _, err := session.CompareMigrated(changed, &r, true, mig); !errors.Is(err, session.ErrGoldenMismatch) {
		t.Errorf("an outcome difference passed under the migration: %v", err)
	}
	if _, err := session.DecodeMigrations([]byte(`{"schema": "dreamteam.golden-migration/v1", "extra": 1}`)); !errors.Is(err, session.ErrGoldenFormat) {
		t.Errorf("unknown manifest field: %v", err)
	}
}
