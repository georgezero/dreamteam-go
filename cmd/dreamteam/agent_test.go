package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// dreamteam agent --stdio writes only protocol responses to stdout, one
// per non-blank input line, in order, whatever the input; diagnostics go
// to stderr; a session it saved reopens with -load and verifies.
func TestAgentStdio(t *testing.T) {
	dir := t.TempDir()
	save := filepath.Join(dir, "a.session.json")
	in := strings.Join([]string{
		`{"id":1,"op":"hello"}`,
		``,
		`garbage {`,
		`{"id":2,"op":"apply","args":{"actions":[{"type":"set_worklist_policy","policy":"fifo"}]}}`,
		`   `,
		`{"id":3,"op":"advance","args":{"minutes":600}}`,
		`{"id":4,"op":"nope"}`,
		`{"id":5,"op":"save","args":{"path":` + mustQuote(save) + `}}`,
		`{"id":6,"op":"digests"}`,
	}, "\n") // no final newline: EOF ends the last request
	var out, errs bytes.Buffer
	if code := agentCmd([]string{"--stdio", "-scenario", "mini", "-seed", "905"}, strings.NewReader(in), &out, &errs); code != exitOK {
		t.Fatalf("exit %d: %s", code, errs.String())
	}
	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	wantIDs := []string{"1", "null", "2", "3", "4", "5", "6"}
	if len(lines) != len(wantIDs) {
		t.Fatalf("%d responses for %d requests:\n%s", len(lines), len(wantIDs), out.String())
	}
	var last struct {
		Result struct {
			Digests map[string]string `json:"digests"`
		} `json:"result"`
	}
	for i, l := range lines {
		var r struct {
			Protocol string          `json:"protocol"`
			ID       json.RawMessage `json:"id"`
			OK       bool            `json:"ok"`
		}
		if err := json.Unmarshal([]byte(l), &r); err != nil || r.Protocol != "dreamteam.agent/v1" || string(r.ID) != wantIDs[i] {
			t.Fatalf("line %d: %q", i, l)
		}
		if ok := wantIDs[i] != "null" && wantIDs[i] != "4"; r.OK != ok {
			t.Fatalf("line %d ok %v: %s", i, r.OK, l)
		}
		if i == len(lines)-1 {
			json.Unmarshal([]byte(l), &last)
		}
	}
	if !strings.Contains(errs.String(), "ready") {
		t.Errorf("no diagnostic on stderr: %q", errs.String())
	}
	// Reopen the saved session: it replays, and its digests are the agent's.
	out.Reset()
	if code := agentCmd([]string{"--stdio", "-load", save}, strings.NewReader(`{"id":1,"op":"digests"}`), &out, &errs); code != exitOK {
		t.Fatalf("load: exit %d: %s", code, errs.String())
	}
	var again struct {
		Result struct {
			Digests map[string]string `json:"digests"`
		} `json:"result"`
	}
	if json.Unmarshal(out.Bytes(), &again) != nil || again.Result.Digests["run"] != last.Result.Digests["run"] || again.Result.Digests["run"] == "" {
		t.Fatalf("the reopened session's digests differ: %s", out.String())
	}
	for _, bad := range [][]string{{}, {"--stdio", "-policy", "noop"}, {"-policy", "best"}, {"--stdio", "extra"}} {
		out.Reset()
		if code := agentCmd(bad, strings.NewReader(""), &out, &errs); code != exitInvalid || out.Len() > 0 {
			t.Errorf("%v: exit %d, stdout %q", bad, code, out.String())
		}
	}
}

func mustQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
