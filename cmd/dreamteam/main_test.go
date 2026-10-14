package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// run → save → replay/verify, and a damaged file is refused.
func TestRunSaveReplayVerify(t *testing.T) {
	dir := t.TempDir()
	acts := filepath.Join(dir, "actions.json")
	os.WriteFile(acts, []byte(`[
 {"time": "2026-10-13T07:00:00", "actions": [{"type": "extend_shift", "employee": "E06", "minutes": 60, "date": "2026-10-15"},
                                             {"type": "extend_shift", "employee": "E99", "minutes": 5}]},
 {"time": "2026-10-14T12:30:00", "actions": [{"type": "set_worklist_policy", "policy": "fifo"}]}
]`), 0o644)
	save := filepath.Join(dir, "run.session.json")
	var out, errs bytes.Buffer
	if code := runCmd([]string{"-scenario", "mini", "-seed", "3", "-days", "3", "-actions", acts, "-save", save}, &out, &errs); code != 0 {
		t.Fatalf("run exit %d:\n%s%s", code, out.String(), errs.String())
	}
	o := out.String()
	for _, want := range []string{"X00001 2026-10-13T07:00:00 ok  extend_shift (accepted)", "X00002 2026-10-13T07:00:00 REJ extend_shift (unknown_employee)",
		"X00003 2026-10-14T12:30:00 ok  set_worklist_policy", "digest: run "} {
		if !strings.Contains(o, want) {
			t.Errorf("run output lacks %q:\n%s", want, o)
		}
	}
	if !strings.Contains(errs.String(), "saved "+save) {
		t.Errorf("stderr %q", errs.String())
	}
	run := o[strings.Index(o, "digest: run ")+12:]
	run = run[:64]
	out.Reset()
	errs.Reset()
	if code := replayCmd([]string{save}, &out, &errs, false); code != 0 || !strings.Contains(errs.String(), "OK: 2 batches, 3 actions replayed") ||
		!strings.Contains(out.String(), "replay: verified true (2 batches)") {
		t.Fatalf("replay exit %d:\n%s%s", code, out.String(), errs.String())
	}
	out.Reset()
	if code := replayCmd([]string{save}, &out, &errs, true); code != 0 || out.String() != "ok "+run+"\n" {
		t.Fatalf("verify exit %d: %q (run %s)", code, out.String(), run)
	}
	raw, _ := os.ReadFile(save)
	os.WriteFile(save, bytes.Replace(raw, []byte(`"policy": "fifo"`), []byte(`"policy": "oldest_first"`), 1), 0o644)
	out.Reset()
	errs.Reset()
	if code := replayCmd([]string{save}, &out, &errs, true); code != exitReplay || !strings.Contains(errs.String(), "checksum") {
		t.Fatalf("damaged file: exit %d %q", code, errs.String())
	}
	// Bad batch times are refused before anything runs.
	os.WriteFile(acts, []byte(`[{"time": "2026-10-20T07:00:00", "actions": [{"type": "set_worklist_policy", "policy": "fifo"}]}]`), 0o644)
	if code := runCmd([]string{"-scenario", "mini", "-days", "1", "-actions", acts}, &out, &errs); code != exitInvalid {
		t.Fatalf("out-of-range batch: exit %d", code)
	}
}

// Step 14 review finding 5: an actions file with anything after the batch
// list is refused, not partly applied.
func TestRegress14ActionsFileTrailing(t *testing.T) {
	p := filepath.Join(t.TempDir(), "a.json")
	os.WriteFile(p, []byte(`[] [{"time": "2026-10-13T07:00:00", "actions": [{"type": "set_worklist_policy", "policy": "fifo"}]}]`), 0o644)
	var out, errs bytes.Buffer
	if code := runCmd([]string{"-scenario", "mini", "-days", "1", "-actions", p}, &out, &errs); code != exitInvalid {
		t.Fatalf("exit %d:\n%s", code, errs.String())
	}
}
