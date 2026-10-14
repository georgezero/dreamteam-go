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
	var out bytes.Buffer
	if code := runCmd([]string{"-scenario", "mini", "-seed", "3", "-days", "3", "-actions", acts, "-save", save}, &out); code != 0 {
		t.Fatalf("run exit %d:\n%s", code, out.String())
	}
	o := out.String()
	for _, want := range []string{"X00001 2026-10-13T07:00:00 ok  extend_shift", "X00002 2026-10-13T07:00:00 REJ extend_shift",
		"X00003 2026-10-14T12:30:00 ok  set_worklist_policy", "digests: run ", "saved " + save} {
		if !strings.Contains(o, want) {
			t.Errorf("run output lacks %q:\n%s", want, o)
		}
	}
	run := o[strings.Index(o, "digests: run ")+13:]
	run = run[:64]
	out.Reset()
	if code := replayCmd([]string{save}, &out, false); code != 0 || !strings.Contains(out.String(), "OK: 2 batches, 3 actions replayed") {
		t.Fatalf("replay exit %d:\n%s", code, out.String())
	}
	out.Reset()
	if code := replayCmd([]string{save}, &out, true); code != 0 || out.String() != "ok "+run+"\n" {
		t.Fatalf("verify exit %d: %q (run %s)", code, out.String(), run)
	}
	raw, _ := os.ReadFile(save)
	os.WriteFile(save, bytes.Replace(raw, []byte(`"policy": "fifo"`), []byte(`"policy": "oldest_first"`), 1), 0o644)
	out.Reset()
	if code := replayCmd([]string{save}, &out, true); code != 1 || !strings.Contains(out.String(), "checksum") {
		t.Fatalf("damaged file: exit %d %q", code, out.String())
	}
	// Bad batch times are refused before anything runs.
	os.WriteFile(acts, []byte(`[{"time": "2026-10-20T07:00:00", "actions": [{"type": "set_worklist_policy", "policy": "fifo"}]}]`), 0o644)
	out.Reset()
	if code := runCmd([]string{"-scenario", "mini", "-days", "1", "-actions", acts}, &out); code != 1 {
		t.Fatalf("out-of-range batch: exit %d", code)
	}
}
