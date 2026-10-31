package bench

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// llm-agent 3 (agents/llm/llm_agent3.py, V0.8c) against the mock model
// server: it decides exactly as llm-agent 2, records every attempt
// (dreamteam.llm-decision-record/v3), and its replay checks every attempt.

func llmPolicy3(t *testing.T, id, model string, extra map[string]any) ExternalPolicy {
	p := llmPolicy(t, id, model, extra)
	p.Command[3], p.Version = "./agents/llm/llm_agent3.py", "3"
	return p
}

// agentAttempt is one attempt of a v3 record, in part.
type agentAttempt struct {
	ID      string            `json:"id"`
	Attempt int               `json:"attempt"`
	Turn    string            `json:"turn"`
	Input   map[string]any    `json:"input"`
	Reason  *string           `json:"reason"`
	Actions []json.RawMessage `json:"actions"`
	Refused []map[string]any  `json:"refusals"`
	Error   *string           `json:"error"`
	Used    bool              `json:"used"`
}

// agentRecord3 is one line of llm-agent 3's decisions.jsonl, in part.
type agentRecord3 struct {
	Schema       string              `json:"schema"`
	Point        int                 `json:"point"`
	Outcome      string              `json:"outcome"`
	PromptSha256 string              `json:"promptSha256"`
	MemorySha256 string              `json:"memorySha256"`
	Attempts     []agentAttempt      `json:"attempts"`
	Final        map[string]any      `json:"final"`
	Batches      [][]json.RawMessage `json:"batches"`
	Completeness map[string]any      `json:"completeness"`
}

// Versions 2 and 3 on the same mock models: the same decisions, prompts
// and memory, so the same simulated practice; version 3's record has every
// attempt, the refused ones with their own actions.
func TestLLMAgent3SameAsVersion2(t *testing.T) {
	llmSetup(t)
	modes := []string{"mixed", "fixafter", "details", "badparams", "retry"}
	var ext []ExternalPolicy
	for _, mode := range modes {
		ext = append(ext, llmPolicy(t, "v2-"+mode, mode, nil), llmPolicy3(t, "v3-"+mode, mode, nil))
	}
	c := externalConfig(t, ext...)
	c.Evaluation.T90 = false
	dir := runMatrix(t, c)
	_, _, recs, err := LoadResults(dir)
	if err != nil {
		t.Fatal(err)
	}
	for i, mode := range modes {
		r2, r3 := recs[1+2*i], recs[2+2*i]
		if r2.Status != StatusCompleted || r3.Status != StatusCompleted || r2.Final.Digests != r3.Final.Digests {
			t.Fatalf("%s: %s %s, final digests differ", mode, r2.Status, r3.Status)
		}
		d2, _ := os.ReadFile(filepath.Join(dir, "runs", r2.ID, FileDecisions))
		d3, _ := os.ReadFile(filepath.Join(dir, "runs", r3.ID, FileDecisions))
		if !bytes.Equal(d2, d3) {
			t.Errorf("%s: the submitted decisions differ", mode)
		}
		e2, e3 := filepath.Join(dir, "runs", r2.ID, FileEvidence), filepath.Join(dir, "runs", r3.ID, FileEvidence)
		m2, _ := os.ReadFile(filepath.Join(e2, "memory.jsonl"))
		m3, _ := os.ReadFile(filepath.Join(e3, "memory.jsonl"))
		if !bytes.Equal(m2, m3) {
			t.Errorf("%s: the memory differs", mode)
		}
		v2 := readJSONL[agentDecision](t, filepath.Join(e2, "decisions.jsonl"))
		v3 := readJSONL[agentRecord3](t, filepath.Join(e3, "decisions.jsonl"))
		calls2 := readJSONL[map[string]any](t, filepath.Join(e2, "model.jsonl"))
		calls3 := readJSONL[map[string]any](t, filepath.Join(e3, "model.jsonl"))
		if len(calls2) != len(calls3) {
			t.Fatalf("%s: %d and %d model calls", mode, len(calls2), len(calls3))
		}
		for k := range calls2 {
			x, _ := json.Marshal(calls2[k]["request"])
			y, _ := json.Marshal(calls3[k]["request"])
			if !bytes.Equal(x, y) {
				t.Errorf("%s: model request %d differs", mode, k)
			}
		}
		log := readJSONL[agentAttempt](t, filepath.Join(e3, "attempts.jsonl"))
		n := 0
		for k, d := range v3 {
			if d.Schema != "dreamteam.llm-decision-record/v3" || d.Outcome != v2[k].Outcome || len(d.Attempts) != v2[k].Attempts ||
				d.MemorySha256 != v2[k].MemorySha256 {
				t.Errorf("%s: point %d: %+v against %+v", mode, k, d, v2[k])
			}
			if d.Completeness["attemptsRecorded"] != true || int(d.Completeness["attempts"].(float64)) != len(d.Attempts) {
				t.Errorf("%s: point %d: completeness %v", mode, k, d.Completeness)
			}
			for j, a := range d.Attempts {
				if a.Attempt != j+1 || a.ID != log[n].ID || a.Turn != log[n].Turn {
					t.Errorf("%s: point %d: attempt %d out of order or not logged", mode, k, j+1)
				}
				n++
			}
			switch mode {
			case "fixafter", "badparams":
				// The refused proposal is kept with its own actions.
				a := d.Attempts[0]
				if len(a.Actions) != 1 || len(a.Refused) != 1 || a.Error == nil || !strings.HasPrefix(*a.Error, "refused: ") {
					t.Errorf("%s: point %d: first attempt %+v", mode, k, a)
				}
				if d.Attempts[1].Input["feedback"] != *a.Error {
					t.Errorf("%s: point %d: the retry's feedback is not the refusal", mode, k)
				}
				// fixafter's retry raises while a raise is left to make.
				want := map[string]any{"attempt": 2.0, "outcome": v2[k].Outcome}
				if mode == "badparams" {
					want = map[string]any{"attempt": nil, "outcome": "fallback"}
				}
				if d.Final["attempt"] != want["attempt"] || d.Final["outcome"] != want["outcome"] || want["outcome"] == "no_action" && k == 0 {
					t.Errorf("%s: point %d: final %v", mode, k, d.Final)
				}
			}
		}
		if n != len(log) {
			t.Errorf("%s: %d attempts logged, %d recorded", mode, len(log), n)
		}
	}
}

// A version 3 recording replays without the model, with every attempt's
// input and validate answers checked; the replay is the live run's; and
// tampered or version 2 recordings are refused.
func TestLLMAgent3Replay(t *testing.T) {
	m := llmSetup(t)
	c := externalConfig(t, llmPolicy3(t, "llm", "fixafter", nil))
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
	calls := m.count("fixafter")
	replay := func(args ...string) ExternalPolicy {
		return ExternalPolicy{ID: "llm", Version: "3", Description: "llm-agent 3, replayed",
			Command: append([]string{"python3", "-I", "-B", "./agents/llm/llm_agent3.py"}, args...), TimeoutSeconds: 60}
	}
	c2 := externalConfig(t, replay(recordings...))
	c2.Seeds, c2.Evaluation.T90 = c.Seeds, false
	a, b := runMatrix(t, c2), runMatrix(t, c2)
	if m.count("fixafter") != calls {
		t.Fatal("the replay called the model")
	}
	x, _ := os.ReadFile(filepath.Join(a, FileManifest))
	y, _ := os.ReadFile(filepath.Join(b, FileManifest))
	if !bytes.Equal(x, y) {
		t.Fatal("two replays differ")
	}
	_, _, recs, _ := LoadResults(a)
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
		// The replayed records are the recorded ones, checked.
		rd := readJSONL[map[string]any](t, filepath.Join(a, "runs", r.ID, FileEvidence, "decisions.jsonl"))
		ld := readJSONL[map[string]any](t, filepath.Join(live, "runs", r.ID, FileEvidence, "decisions.jsonl"))
		for k := range ld {
			chk, _ := rd[k]["replayChecked"].(map[string]any)
			if chk == nil || chk["inputs"].(float64) != 2 || chk["validations"].(float64) < 1 {
				t.Errorf("%s: point %d: checked %v", r.ID, k, rd[k]["replayChecked"])
			}
			for _, f := range []string{"attempts", "final", "batches", "completeness", "promptSha256"} {
				p, _ := json.Marshal(ld[k][f])
				q, _ := json.Marshal(rd[k][f])
				if !bytes.Equal(p, q) {
					t.Errorf("%s: point %d: %s differs", r.ID, k, f)
				}
			}
		}
	}

	one := recordings[1]
	tamper := func(f func(d map[string]any)) string {
		p := filepath.Join(t.TempDir(), "decisions.jsonl")
		raw, _ := os.ReadFile(one)
		lines := bytes.Split(bytes.TrimSpace(raw), []byte("\n"))
		var d map[string]any
		json.Unmarshal(lines[1], &d)
		f(d)
		lines[1], _ = json.Marshal(d)
		os.WriteFile(p, append(bytes.Join(lines, []byte("\n")), '\n'), 0o644)
		return p
	}
	first := func(d map[string]any) map[string]any { return d["attempts"].([]any)[0].(map[string]any) }
	answer := tamper(func(d map[string]any) {
		v := first(d)["validation"].([]any)[0].(map[string]any)["answer"].(map[string]any)
		v["results"].([]any)[0].(map[string]any)["code"] = "not_employed"
	})
	dropped := tamper(func(d map[string]any) { d["attempts"] = d["attempts"].([]any)[1:] })
	feedback := tamper(func(d map[string]any) {
		d["attempts"].([]any)[1].(map[string]any)["input"].(map[string]any)["feedback"] = "refused: nothing"
	})
	v2 := filepath.Join(t.TempDir(), "decisions.jsonl")
	{
		c := externalConfig(t, llmPolicy(t, "llm", "fixafter", nil))
		dir := runMatrix(t, c)
		_, _, rs, _ := LoadResults(dir)
		raw, _ := os.ReadFile(filepath.Join(dir, "runs", rs[1].ID, FileEvidence, "decisions.jsonl"))
		os.WriteFile(v2, raw, 0o644)
	}
	for _, tc := range []struct{ file, want string }{
		{answer, "decision point 1, attempt 1: validate answers batch 0 differently from the recording"},
		{dropped, "decision point 1, attempt 1: missing or out of order"},
		{feedback, "decision point 1, attempt 2: the feedback is not the previous attempt's error"},
		{v2, "llm-agent 1 and 2 recordings replay with agents/llm/llm_agent.py"},
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
