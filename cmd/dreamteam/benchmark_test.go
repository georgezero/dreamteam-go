package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const benchConfig = `{
 "schema": "dreamteam.benchmark/v1",
 "id": "cmd-test",
 "description": "test",
 "scenarios": ["sustainable"],
 "seeds": [77],
 "seedPlan": "a test seed",
 "days": 2,
 "cadence": {"everyDays": 1, "at": "07:00"},
 "policies": ["greedy"],
 "baseline": "greedy",
 "output": "results",
 "evaluation": {"t90": true}
}`

// dreamteam benchmark run, report and verify end to end; T90 is the
// evaluation dreamteam t90 gives for the saved session (the v4 policy,
// unchanged), and result.json is dreamteam replay's canonical result.
func TestBenchmarkCommand(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.json")
	os.WriteFile(cfg, []byte(benchConfig), 0o644)
	var out, errb bytes.Buffer
	if code := benchmarkCmd([]string{"run", cfg}, &out, &errb); code != exitOK {
		t.Fatalf("run: exit %d\n%s", code, errb.String())
	}
	if !strings.Contains(out.String(), "1 of 1 saved sessions replayed") {
		t.Fatalf("report:\n%s", out.String())
	}
	res := filepath.Join(dir, "results")
	rd := filepath.Join(res, "runs", "sustainable-s77-greedy")

	var t90out bytes.Buffer
	if code := t90Cmd([]string{filepath.Join(rd, "session.json"), "-format", "json"}, &t90out, &errb); code != exitOK {
		t.Fatalf("t90: exit %d %s", code, errb.String())
	}
	var stored struct {
		Evaluation json.RawMessage `json:"evaluation"`
	}
	raw, _ := os.ReadFile(filepath.Join(rd, "t90.json"))
	json.Unmarshal(raw, &stored)
	var a, b bytes.Buffer
	json.Compact(&a, stored.Evaluation)
	json.Compact(&b, t90out.Bytes())
	if a.String() != b.String() || !strings.Contains(a.String(), `"policyId":"v0.2-eval-4"`) || !strings.Contains(a.String(), `"projected":true`) {
		t.Fatalf("t90.json differs from dreamteam t90:\n%s\n%s", a.String()[:200], b.String()[:200])
	}

	var replayOut bytes.Buffer
	if code := replayCmd([]string{filepath.Join(rd, "session.json"), "-format", "json"}, &replayOut, &errb, false); code != exitOK {
		t.Fatalf("replay: exit %d", code)
	}
	result, _ := os.ReadFile(filepath.Join(rd, "result.json"))
	if !bytes.Equal(result, replayOut.Bytes()) {
		t.Fatal("result.json differs from dreamteam replay -format json")
	}

	out.Reset()
	if code := benchmarkCmd([]string{"verify", res, "-t90"}, &out, &errb); code != exitOK || !strings.Contains(out.String(), "t90.json recomputed") {
		t.Fatalf("verify: exit %d\n%s", code, out.String())
	}
	out.Reset()
	if code := benchmarkCmd([]string{"report", res, "-format", "json"}, &out, &errb); code != exitOK {
		t.Fatalf("report: exit %d", code)
	}
	stored2, _ := os.ReadFile(filepath.Join(res, "report.json"))
	if !bytes.Equal(out.Bytes(), stored2) {
		t.Fatal("report -format json differs from report.json")
	}

	// A rerun of the recorded configuration reproduces every
	// deterministic file; a manifest that records another hash fails.
	out.Reset()
	again := filepath.Join(dir, "again")
	if code := benchmarkCmd([]string{"reproduce", res, "-out", again}, &out, &errb); code != exitOK || !strings.Contains(out.String(), "ok    manifest.json byte-identical") {
		t.Fatalf("reproduce: exit %d\n%s", code, out.String())
	}
	m, _ := os.ReadFile(filepath.Join(again, "manifest.json"))
	var man struct {
		Runs []struct {
			Files []struct{ Path, SHA256 string }
		}
	}
	json.Unmarshal(m, &man)
	sum := man.Runs[0].Files[4].SHA256 // result.json
	os.WriteFile(filepath.Join(again, "manifest.json"), bytes.Replace(m, []byte(sum), bytes.Repeat([]byte("0"), 64), 1), 0o644)
	out.Reset()
	if code := benchmarkCmd([]string{"reproduce", again, "-out", filepath.Join(dir, "third")}, &out, &errb); code != exitReplay ||
		!strings.Contains(out.String(), "FAIL  file runs/sustainable-s77-greedy/result.json") || !strings.Contains(out.String(), "FAIL  manifest.json byte-identical") {
		t.Fatalf("reproduce against a changed manifest: exit %d\n%s", code, out.String())
	}

	// Results are written once.
	errb.Reset()
	if code := benchmarkCmd([]string{"run", cfg}, &out, &errb); code != exitError || !strings.Contains(errb.String(), "not empty") {
		t.Fatalf("rerun into the same output: exit %d %s", code, errb.String())
	}
	// A tampered result fails verification.
	os.WriteFile(filepath.Join(rd, "result.json"), append(result, ' '), 0o644)
	out.Reset()
	if code := benchmarkCmd([]string{"verify", res}, &out, &errb); code != exitReplay || !strings.Contains(out.String(), "FAIL  hash runs/sustainable-s77-greedy/result.json") {
		t.Fatalf("tampered verify: exit %d\n%s", code, out.String())
	}
}

// An invalid configuration is refused before anything runs, with the
// reason; usage errors exit 2.
func TestBenchmarkCommandErrors(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.json")
	os.WriteFile(cfg, []byte(strings.Replace(benchConfig, `[77]`, `[1003]`, 1)), 0o644)
	var out, errb bytes.Buffer
	if code := benchmarkCmd([]string{"run", cfg}, &out, &errb); code != exitInvalid || !strings.Contains(errb.String(), "reference evaluation seed") {
		t.Fatalf("exit %d %s", code, errb.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "results")); !os.IsNotExist(err) {
		t.Fatal("an invalid configuration wrote results")
	}
	for _, args := range [][]string{nil, {"run"}, {"bogus"}, {"report"}, {"report", dir, "-format", "xml"}, {"verify"}, {"reproduce"}} {
		if code := benchmarkCmd(args, &out, &errb); code != exitInvalid {
			t.Errorf("%v: exit %d", args, code)
		}
	}
	if code := benchmarkCmd([]string{"run", filepath.Join(dir, "missing.json")}, &out, &errb); code != exitError {
		t.Errorf("missing config: exit %d", code)
	}
	if code := benchmarkCmd([]string{"verify", dir}, &out, &errb); code != exitError {
		t.Errorf("verify without a manifest: exit %d", code)
	}
}
