package bench

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// llm-agent 4 (agents/llm/llm_agent4.py, V0.9c) against the engine and the
// mock model server: llm-agent 3 with the preview step (llm-agent-preview/v1).
// The preview is the engine's own, it changes nothing, and the replay
// checks it.

func llmPolicy4(t *testing.T, id, model string, extra map[string]any) ExternalPolicy {
	p := llmPolicy(t, id, model, extra)
	p.Command[3], p.Version = "./agents/llm/llm_agent4.py", "4"
	return p
}

// agentRecord4 is one line of llm-agent 4's decisions.jsonl, in part.
type agentRecord4 struct {
	Schema   string `json:"schema"`
	Point    int    `json:"point"`
	Outcome  string `json:"outcome"`
	Attempts []struct {
		Turn    string         `json:"turn"`
		Input   map[string]any `json:"input"`
		Preview *struct {
			Status           string         `json:"status"`
			Answer           map[string]any `json:"answer"`
			DigestsUnchanged bool           `json:"digestsUnchanged"`
		} `json:"preview"`
		Used bool `json:"used"`
	} `json:"attempts"`
	Preview struct {
		Status   string `json:"status"`
		Response string `json:"response"`
		Final    string `json:"final"`
	} `json:"preview"`
	Completeness map[string]any `json:"completeness"`
}

// A kept proposal is version 3's decision: the same submitted decisions and
// so the same session, byte for byte (the preview alone changes nothing),
// and the same first message at every point. A revised one is the reply
// after the preview; an abandoned one submits nothing.
func TestLLMAgent4Preview(t *testing.T) {
	m := llmSetup(t)
	c := externalConfig(t, llmPolicy3(t, "v3", "raise", nil), llmPolicy4(t, "keep", "pvkeep", nil),
		llmPolicy4(t, "revise", "pvrevise", nil), llmPolicy4(t, "abandon", "pvabandon", nil))
	c.Evaluation.T90 = false
	dir := runMatrix(t, c)
	_, _, recs, err := LoadResults(dir)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]*RunRecord{}
	for _, r := range recs {
		if r.Status != StatusCompleted {
			t.Fatalf("%s: %s %+v", r.ID, r.Status, r.Failure)
		}
		byID[r.Policy.ID] = r
	}
	file := func(id, name string) []byte {
		b, _ := os.ReadFile(filepath.Join(dir, "runs", byID[id].ID, name))
		return b
	}
	for _, f := range []string{FileDecisions, FileSession} {
		if !bytes.Equal(file("v3", f), file("keep", f)) {
			t.Errorf("kept proposals: %s differs from version 3's", f)
		}
	}
	if byID["v3"].Final.Digests != byID["keep"].Final.Digests {
		t.Error("kept proposals: the final digests differ from version 3's")
	}
	ev := func(id, name string) string { return filepath.Join(dir, "runs", byID[id].ID, FileEvidence, name) }
	calls3 := readJSONL[map[string]any](t, ev("v3", "model.jsonl"))
	calls4 := readJSONL[map[string]any](t, ev("keep", "model.jsonl"))
	var first3, first4 [][]byte
	reviews := 0
	// The messages: the requests differ in the mock model's name only.
	messages := func(c map[string]any) any { return c["request"].(map[string]any)["messages"] }
	for _, c := range calls3 {
		b, _ := json.Marshal(messages(c))
		first3 = append(first3, b)
	}
	for _, c := range calls4 {
		if c["turn"] == "review" {
			reviews++
			continue
		}
		b, _ := json.Marshal(messages(c))
		first4 = append(first4, b)
	}
	if len(first3) != len(first4) || reviews == 0 {
		t.Fatalf("%d and %d first requests, %d reviews", len(first3), len(first4), reviews)
	}
	for k := range first3 {
		if !bytes.Equal(first3[k], first4[k]) {
			t.Errorf("model request %d's messages differ from version 3's", k)
		}
	}
	shown := 0
	for _, id := range []string{"keep", "revise", "abandon"} {
		for _, d := range readJSONL[agentRecord4](t, ev(id, "decisions.jsonl")) {
			if d.Schema != "dreamteam.llm-decision-record/v4" {
				t.Fatalf("%s: point %d: schema %s", id, d.Point, d.Schema)
			}
			if d.Preview.Status == "not_applicable" {
				continue
			}
			shown++
			a := d.Attempts[0]
			if d.Preview.Status != "ok" || a.Preview == nil || a.Preview.Status != "ok" || !a.Preview.DigestsUnchanged ||
				a.Used || d.Attempts[1].Turn != "review" || d.Attempts[1].Input["kind"] != "preview" {
				t.Errorf("%s: point %d: %+v", id, d.Point, d)
				continue
			}
			// The engine's preview: the allocation change and its guaranteed cost
			// (the mini scenario has no funding rule, so no funding test).
			pv, _ := a.Preview.Answer["preview"].(map[string]any)
			if pv == nil || pv["allocations"] == nil || pv["guaranteed"] == nil {
				t.Errorf("%s: point %d: not the engine's preview: %v", id, d.Point, a.Preview.Answer)
			}
			want := map[string][2]string{"keep": {"kept", "kept"}, "revise": {"revised", "revised"},
				"abandon": {"abandoned", "abandoned"}}[id]
			if d.Preview.Response != want[0] || d.Preview.Final != want[1] {
				t.Errorf("%s: point %d: response %s, final %s", id, d.Point, d.Preview.Response, d.Preview.Final)
			}
			if id == "abandon" && d.Outcome != "no_action" {
				t.Errorf("abandon: point %d: outcome %s", d.Point, d.Outcome)
			}
		}
	}
	if shown == 0 {
		t.Fatal("no preview was shown")
	}
	if strings.Contains(string(file("abandon", FileDecisions)), "set_reading_allocation") {
		t.Error("abandon: an allocation change was submitted")
	}
	if m.count("pvkeep") != len(calls4) {
		t.Errorf("pvkeep: %d calls served, %d recorded", m.count("pvkeep"), len(calls4))
	}
}

// A version 4 recording replays without the model, with every preview
// asked again and its review message rebuilt; the replay is the live run's;
// tampered previews and version 3 recordings are refused.
func TestLLMAgent4Replay(t *testing.T) {
	m := llmSetup(t)
	c := externalConfig(t, llmPolicy4(t, "llm", "pvrevise", nil))
	c.Seeds, c.Evaluation.T90 = []int64{privateSeed, 5}, false
	live := runMatrix(t, c)
	_, _, liveRecs, err := LoadResults(live)
	if err != nil {
		t.Fatal(err)
	}
	var recordings []string
	for _, r := range liveRecs {
		if r.Policy.ID == "llm" {
			recordings = append(recordings, "--replay", filepath.Join(live, "runs", r.ID, FileEvidence, "decisions.jsonl"))
		}
	}
	calls := m.count("pvrevise")
	replay := func(args ...string) ExternalPolicy {
		return ExternalPolicy{ID: "llm", Version: "4", Description: "llm-agent 4, replayed",
			Command: append([]string{"python3", "-I", "-B", "./agents/llm/llm_agent4.py"}, args...), TimeoutSeconds: 60}
	}
	c2 := externalConfig(t, replay(recordings...))
	c2.Seeds, c2.Evaluation.T90 = c.Seeds, false
	a, b := runMatrix(t, c2), runMatrix(t, c2)
	if m.count("pvrevise") != calls {
		t.Fatal("the replay called the model")
	}
	x, _ := os.ReadFile(filepath.Join(a, FileManifest))
	y, _ := os.ReadFile(filepath.Join(b, FileManifest))
	if !bytes.Equal(x, y) {
		t.Fatal("two replays differ")
	}
	_, _, recs, _ := LoadResults(a)
	previews := 0
	for i, r := range recs {
		l := liveRecs[i]
		if r.Status != StatusCompleted || r.Final.Digests != l.Final.Digests {
			t.Fatalf("%s: %s, final digests differ from the live run's", r.ID, r.Status)
		}
		for _, f := range []string{FileResult, FileSession, FileDecisions} {
			p, _ := os.ReadFile(filepath.Join(live, "runs", r.ID, f))
			q, _ := os.ReadFile(filepath.Join(a, "runs", r.ID, f))
			if !bytes.Equal(p, q) {
				t.Errorf("%s: %s differs from the live run's", r.ID, f)
			}
		}
		if r.Policy.ID != "llm" {
			continue
		}
		rd := readJSONL[map[string]any](t, filepath.Join(a, "runs", r.ID, FileEvidence, "decisions.jsonl"))
		ld := readJSONL[map[string]any](t, filepath.Join(live, "runs", r.ID, FileEvidence, "decisions.jsonl"))
		for k := range ld {
			chk, _ := rd[k]["replayChecked"].(map[string]any)
			if chk == nil {
				t.Fatalf("%s: point %d: not checked", r.ID, k)
			}
			previews += int(chk["previews"].(float64))
			for _, f := range []string{"attempts", "final", "batches", "preview", "completeness", "promptSha256"} {
				p, _ := json.Marshal(ld[k][f])
				q, _ := json.Marshal(rd[k][f])
				if !bytes.Equal(p, q) {
					t.Errorf("%s: point %d: %s differs", r.ID, k, f)
				}
			}
		}
	}
	if previews == 0 {
		t.Fatal("no preview was checked")
	}

	// The first decision point whose first attempt was previewed.
	one := recordings[1]
	raw, _ := os.ReadFile(one)
	at := -1
	for k, l := range bytes.Split(bytes.TrimSpace(raw), []byte("\n")) {
		var d agentRecord4
		json.Unmarshal(l, &d)
		if at < 0 && len(d.Attempts) > 0 && d.Attempts[0].Preview != nil {
			at = k
		}
	}
	if at < 0 {
		t.Fatal("no previewed decision point to tamper with")
	}
	tamper := func(f func(d map[string]any)) string {
		p := filepath.Join(t.TempDir(), "decisions.jsonl")
		lines := bytes.Split(bytes.TrimSpace(raw), []byte("\n"))
		var d map[string]any
		json.Unmarshal(lines[at], &d)
		f(d)
		lines[at], _ = json.Marshal(d)
		os.WriteFile(p, append(bytes.Join(lines, []byte("\n")), '\n'), 0o644)
		return p
	}
	first := func(d map[string]any) map[string]any { return d["attempts"].([]any)[0].(map[string]any) }
	answer := tamper(func(d map[string]any) {
		pv := first(d)["preview"].(map[string]any)["answer"].(map[string]any)["preview"].(map[string]any)
		pv["asOf"] = "2026-01-01T07:00:00"
	})
	dropped := tamper(func(d map[string]any) { first(d)["preview"] = nil })
	v3 := filepath.Join(t.TempDir(), "decisions.jsonl")
	{
		c := externalConfig(t, llmPolicy3(t, "llm", "raise", nil))
		dir := runMatrix(t, c)
		_, _, rs, _ := LoadResults(dir)
		raw, _ := os.ReadFile(filepath.Join(dir, "runs", rs[1].ID, FileEvidence, "decisions.jsonl"))
		os.WriteFile(v3, raw, 0o644)
	}
	for _, tc := range []struct{ file, want string }{
		{answer, fmt.Sprintf("decision point %d, attempt 1: the preview's answer differs from the recording's", at)},
		{dropped, fmt.Sprintf("decision point %d, attempt 1: a preview missing at this attempt", at)},
		{v3, "version 3 recordings replay with agents/llm/llm_agent3.py"},
	} {
		c3 := externalConfig(t, replay("--replay", tc.file))
		c3.Evaluation.T90 = false
		dir := runMatrix(t, c3)
		_, _, rs, _ := LoadResults(dir)
		stderr, _ := os.ReadFile(filepath.Join(dir, "runs", rs[1].ID, FileStderr))
		if rs[1].Status != StatusCrash || !strings.Contains(string(stderr), tc.want) {
			t.Errorf("%s: %s, stderr %q", tc.want, rs[1].Status, stderr)
		}
	}
}
