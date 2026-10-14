package session

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/scenario"
)

// Step 14 review finding 2: the run digest binds the scenario: two runs
// identical so far but with different future rules (a cost not yet
// incurred) must not share a run digest, and a record whose scenario was
// swapped for such a variant must not verify.
func TestRegress14RunDigestBindsScenario(t *testing.T) {
	a := openFile(t, "mini", 1, nil)
	b := openFile(t, "mini", 1, func(f *scenario.File) { f.Finance.PerStudyITCents++ })
	if a.Digests().Run == b.Digests().Run {
		t.Error("different scenarios, same run digest")
	}
	p := saveTemp(t, a)
	rewrite(t, p, func(r *Record) {
		f, _ := scenario.Decode(r.Scenario.Resolved)
		f.Finance.PerStudyITCents++
		r.Scenario.Resolved, _ = scenario.Canonical(f)
		r.Scenario.Fingerprint, _ = scenario.Fingerprint(f)
	})
	if _, _, err := Load(p, ReplayOptions{}); err == nil {
		t.Error("a record with a swapped scenario and its old digests verified")
	}
}

// Step 14 review finding 3: a session keeps its own copy of the scenario:
// changing the caller's file afterwards cannot break its saves.
func TestRegress14FromFileCopiesScenario(t *testing.T) {
	f, err := scenario.ResolveFile("mini")
	if err != nil {
		t.Fatal(err)
	}
	s, err := FromFile(f, 1)
	if err != nil {
		t.Fatal(err)
	}
	f.OpeningCashCents++
	f.Employees[0].Skills = append(f.Employees[0].Skills[:0], "MR")
	f.Demand.HourProfileBP["09"]++
	p := saveTemp(t, s)
	if _, _, err := Load(p, ReplayOptions{}); err != nil {
		t.Fatalf("a save made after the caller changed its scenario does not load: %v", err)
	}
}

// Step 14 review finding 5: nothing may follow the record, not even a
// stray closing delimiter.
func TestRegress14RecordRejectsTrailing(t *testing.T) {
	s := openFile(t, "mini", 2, nil)
	good := saveTemp(t, s)
	raw, _ := os.ReadFile(good)
	for _, tail := range []string{"]", "}", "x", "[]"} {
		p := copyFile(t, good)
		os.WriteFile(p, append(append([]byte{}, raw...), tail...), 0o644)
		if _, err := ReadRecord(p); !errors.Is(err, ErrCorrupt) {
			t.Errorf("trailing %q: %v", tail, err)
		}
	}
	p := copyFile(t, good)
	os.WriteFile(p, append(append([]byte{}, raw...), "\n\n"...), 0o644)
	if _, err := ReadRecord(p); err != nil {
		t.Errorf("trailing whitespace rejected: %v", err)
	}
}

// Step 14 review finding 6: every redundant field is checked against what
// it describes; edits that keep the checksum consistent are caught.
func TestRegress14MetadataChecked(t *testing.T) {
	s := openFile(t, "mini", 3, nil)
	play(t, s)
	good := saveTemp(t, s)
	edits := map[string]func(*Record){
		"batch numbers": func(r *Record) { r.Batches[0].Batch += 100; r.Batches[1].Batch += 100; r.Batches[2].Batch += 100 },
		"batch time":    func(r *Record) { r.Batches[1].Time = "2026-10-13T15:38:00" },
		"scenario id":   func(r *Record) { r.Scenario.ID = "established" },
		"scenario name": func(r *Record) { r.Scenario.Name = "Another Practice" },
		"start text":    func(r *Record) { r.Start = "2026-10-13T07:00:00" },
		"until text":    func(r *Record) { r.Until = "2026-10-20T00:00:00" },
		"epoch":         func(r *Record) { r.EpochDate = "2026-10-12" },
		"summary cash":  func(r *Record) { r.Summary.CashCents++ },
		"summary exams": func(r *Record) { r.Summary.ExamsSince++ },
		"result batch":  func(r *Record) { r.Batches[1].Results[0].Batch = 9 },
	}
	for name, edit := range edits {
		p := copyFile(t, good)
		rewrite(t, p, edit)
		if _, _, err := Load(p, ReplayOptions{}); err == nil {
			t.Errorf("%s: edited record verified", name)
		}
	}
}

// Step 14 review finding 7: if the directory cannot be synced after the
// rename, the save is not reported as durable.
func TestRegress14DirectorySyncFailure(t *testing.T) {
	s := openFile(t, "mini", 4, nil)
	at(t, s, 0, "12:00")
	p := filepath.Join(t.TempDir(), "x.json")
	failDirSync = func() error { return errors.New("I/O error") }
	err := s.Save(p)
	failDirSync = nil
	if err == nil || !strings.Contains(err.Error(), "durab") || !s.Unsaved() {
		t.Fatalf("err %v, unsaved %v", err, s.Unsaved())
	}
}

// Step 14 review finding 8: a failed replay never reports success, and a
// mismatch reports where the replay actually stood.
func TestRegress14FailureReports(t *testing.T) {
	s := openFile(t, "mini", 5, nil)
	play(t, s)
	good := saveTemp(t, s)
	p := copyFile(t, good)
	rewrite(t, p, func(r *Record) { r.Engine = "0.1.0-dev.1" })
	_, rep, err := Load(p, ReplayOptions{})
	if err == nil || rep == nil || rep.OK || strings.HasPrefix(rep.String(), "OK") {
		t.Errorf("engine refusal reported as %v / %v", err, rep)
	}
	p = copyFile(t, good)
	rewrite(t, p, func(r *Record) { r.Batches[1].Results[0].Message = "something else" })
	_, rep, err = Load(p, ReplayOptions{})
	if !errors.Is(err, ErrMismatch) || rep.RepSummary.Time != "2026-10-13T15:37:00" {
		t.Errorf("action mismatch: replayed summary %+v", rep.RepSummary)
	}
}
