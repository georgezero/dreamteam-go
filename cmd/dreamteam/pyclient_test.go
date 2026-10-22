package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// The reference Python client (clients/python) runs a complete session
// against the real binary as a subprocess, checks the save with
// dreamteam verify and dreamteam t90, and its transcript is the same
// twice. Skipped without python3.
func TestPythonClient(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the binary and runs two T90 evaluations twice")
	}
	py, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not found")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "dreamteam")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	script, _ := filepath.Abs("../../clients/python/example_session.py")
	var transcripts [][]byte
	for _, run := range []string{"a", "b"} {
		wd := filepath.Join(dir, run)
		if err := os.Mkdir(wd, 0o755); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(py, "-I", script, "--binary", bin, "--days", "5", "--t90", "--transcript", "transcript.jsonl")
		cmd.Dir = wd
		if out, err := cmd.CombinedOutput(); err != nil || !bytes.Contains(out, []byte(`"checks": "passed"`)) {
			t.Fatalf("run %s: %v\n%s", run, err, out)
		}
		raw, err := os.ReadFile(filepath.Join(wd, "transcript.jsonl"))
		if err != nil {
			t.Fatal(err)
		}
		transcripts = append(transcripts, raw)
	}
	if !bytes.Equal(transcripts[0], transcripts[1]) {
		t.Fatal("two identical client sessions gave different transcripts")
	}
}
