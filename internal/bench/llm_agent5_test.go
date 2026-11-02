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

// llm-agent 5 (agents/llm/llm_agent5.py) against the engine and the mock
// model server, with each workflow (V0.11.0 --workflow):
//   - validation-first-preview (llm-agent-workflow/v1, V0.11a): every
//     proposal is validated before anything else; a refusal is answered
//     with the simulator's own results; a valid staffing proposal's preview
//     is shown when the engine's answer shows a direct effect; only
//     validated batches are submitted;
//   - validation-first (the default): the same without the preview.
// The replay checks it all without a model, with the recording's workflow.

const (
	workflowVF = "validation-first"
	workflowPV = "validation-first-preview"
)

// llmPolicy5 runs llm-agent 5 live with the named workflow ("" names none:
// the default).
func llmPolicy5(t *testing.T, id, model string, extra map[string]any, workflow ...string) ExternalPolicy {
	p := llmPolicy(t, id, model, extra)
	p.Command[3], p.Version = "./agents/llm/llm_agent5.py", "5"
	for _, w := range workflow {
		if w != "" {
			p.Command = append(p.Command, "--workflow", w)
		}
	}
	return p
}

// agentRecord5 is one line of llm-agent 5's decisions.jsonl, in part.
type agentRecord5 struct {
	Schema          string             `json:"schema"`
	Workflow        string             `json:"workflow"`
	WorkflowVersion string             `json:"workflowVersion"`
	Point           int                `json:"point"`
	Outcome         string             `json:"outcome"`
	Batches         [][]map[string]any `json:"batches"`
	Attempts        []struct {
		Turn       string         `json:"turn"`
		Input      map[string]any `json:"input"`
		Validation []struct {
			Answer struct {
				Valid bool `json:"valid"`
			} `json:"answer"`
		} `json:"validation"`
		Refusals []map[string]any `json:"refusals"`
		Preview  *struct {
			Status           string         `json:"status"`
			Answer           map[string]any `json:"answer"`
			DigestsUnchanged bool           `json:"digestsUnchanged"`
			Eligibility      struct {
				Eligible bool     `json:"eligible"`
				Reasons  []string `json:"reasons"`
			} `json:"eligibility"`
		} `json:"preview"`
		Used bool `json:"used"`
	} `json:"attempts"`
	Stages struct {
		Previewed *struct {
			Status string `json:"status"`
		} `json:"previewed"`
	} `json:"stages"`
	DigestsBeforeSubmit struct {
		Unchanged bool `json:"unchanged"`
	} `json:"digestsBeforeSubmit"`
	PreviousExecution *struct {
		Point   int `json:"point"`
		Batches []struct {
			Applied bool `json:"applied"`
		} `json:"batches"`
	} `json:"previousExecution"`
}

func TestLLMAgent5Workflow(t *testing.T) {
	m := llmSetup(t)
	c := externalConfig(t, llmPolicy5(t, "keep", "pvkeep", nil, workflowPV), llmPolicy5(t, "revise", "pvrevise", nil, workflowPV),
		llmPolicy5(t, "abandon", "pvabandon", nil, workflowPV), llmPolicy5(t, "fix", "fixafter", nil, workflowPV),
		llmPolicy5(t, "mixed", "mixed", nil, workflowPV))
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
	ev := func(id, name string) string { return filepath.Join(dir, "runs", byID[id].ID, FileEvidence, name) }
	shown, fixed, executed := 0, 0, 0
	for _, id := range []string{"keep", "revise", "abandon", "fix", "mixed"} {
		ds := readJSONL[agentRecord5](t, ev(id, "decisions.jsonl"))
		var harness []Decision
		if err := readJSON(filepath.Join(dir, "runs", byID[id].ID, FileDecisions), &harness); err != nil || len(harness) != len(ds) {
			t.Fatalf("%s: decisions.json: %v (%d points, %d records)", id, err, len(harness), len(ds))
		}
		for k, d := range ds {
			if d.Schema != "dreamteam.llm-decision-record/v5" || !d.DigestsBeforeSubmit.Unchanged {
				t.Fatalf("%s: point %d: schema %s, state unchanged %v", id, d.Point, d.Schema, d.DigestsBeforeSubmit.Unchanged)
			}
			if d.Workflow != workflowPV || d.WorkflowVersion != "llm-agent-workflow/v1" {
				t.Fatalf("%s: point %d: workflow %q (%s)", id, d.Point, d.Workflow, d.WorkflowVersion)
			}
			// Only a validated proposal is submitted, and the harness applies it.
			var used *int
			for i, a := range d.Attempts {
				if a.Used {
					i := i
					used = &i
				}
			}
			if len(d.Batches) > 0 {
				if used == nil || len(d.Attempts[*used].Refusals) > 0 || len(d.Attempts[*used].Validation) != len(d.Batches) {
					t.Errorf("%s: point %d: submitted batches without a fully validated attempt", id, d.Point)
				}
				for _, v := range d.Attempts[*used].Validation {
					if !v.Answer.Valid {
						t.Errorf("%s: point %d: a refused batch was submitted", id, d.Point)
					}
				}
				for _, b := range harness[k].Batches {
					if !b.Applied {
						t.Errorf("%s: point %d: the harness refused a validated batch", id, d.Point)
					}
				}
			}
			if d.PreviousExecution != nil && len(d.PreviousExecution.Batches) > 0 {
				for _, b := range d.PreviousExecution.Batches {
					if !b.Applied {
						t.Errorf("%s: point %d: the previous point's batch is recorded as not applied", id, d.Point)
					}
				}
				executed++
			}
			if len(d.Attempts) == 0 {
				continue
			}
			a := d.Attempts[0]
			if len(a.Refusals) > 0 {
				// fixafter: refused first, then the simulator's results are shown.
				if id != "fix" || len(d.Attempts) < 2 || d.Attempts[1].Input["kind"] != "validation" {
					t.Errorf("%s: point %d: a refusal not answered with validation feedback", id, d.Point)
				}
				fixed++
				continue
			}
			if a.Preview == nil {
				continue
			}
			if !a.Preview.DigestsUnchanged || len(a.Validation) == 0 {
				t.Errorf("%s: point %d: a preview before validation, or one that changed the state", id, d.Point)
			}
			if a.Preview.Status != "shown" {
				continue
			}
			// The mini scenario's allocation change: a new salary obligation.
			pv, _ := a.Preview.Answer["preview"].(map[string]any)
			if pv == nil || pv["allocations"] == nil || !a.Preview.Eligibility.Eligible ||
				!strings.Contains(strings.Join(a.Preview.Eligibility.Reasons, ","), "obligation") {
				t.Errorf("%s: point %d: not the engine's preview or eligibility: %+v", id, d.Point, a.Preview)
			}
			if a.Used || len(d.Attempts) < 2 || d.Attempts[1].Input["kind"] != "preview" || d.Attempts[1].Turn != "review" {
				t.Errorf("%s: point %d: the preview was not answered by the model", id, d.Point)
			}
			shown++
			if id == "abandon" && d.Outcome != "no_action" {
				t.Errorf("abandon: point %d: outcome %s", d.Point, d.Outcome)
			}
		}
		calls := readJSONL[map[string]any](t, ev(id, "model.jsonl"))
		if m.count(map[string]string{"keep": "pvkeep", "revise": "pvrevise", "abandon": "pvabandon", "fix": "fixafter",
			"mixed": "mixed"}[id]) != len(calls) {
			t.Errorf("%s: model calls served and recorded differ", id)
		}
	}
	if shown == 0 || fixed == 0 || executed == 0 {
		t.Fatalf("shown %d, fixed %d, executed %d", shown, fixed, executed)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "runs", byID["abandon"].ID, FileDecisions))
	if strings.Contains(string(b), "set_reading_allocation") {
		t.Error("abandon: an allocation change was submitted")
	}
}

// A version 5 recording replays without the model, byte for byte; a
// tampered validation answer, a dropped preview and a version 4 recording
// are refused.
func TestLLMAgent5Replay(t *testing.T) {
	m := llmSetup(t)
	c := externalConfig(t, llmPolicy5(t, "llm", "pvrevise", nil, workflowPV), llmPolicy5(t, "fix", "fixafter", nil, workflowPV))
	c.Seeds, c.Evaluation.T90 = []int64{privateSeed, 5}, false
	live := runMatrix(t, c)
	_, _, liveRecs, err := LoadResults(live)
	if err != nil {
		t.Fatal(err)
	}
	recordings := map[string][]string{}
	for _, r := range liveRecs {
		if r.Policy.ID != "noop" {
			recordings[r.Policy.ID] = append(recordings[r.Policy.ID], "--replay",
				filepath.Join(live, "runs", r.ID, FileEvidence, "decisions.jsonl"))
		}
	}
	calls := m.count("pvrevise") + m.count("fixafter")
	replay := func(id string, args ...string) ExternalPolicy {
		return ExternalPolicy{ID: id, Version: "5", Description: "llm-agent 5, replayed",
			Command: append([]string{"python3", "-I", "-B", "./agents/llm/llm_agent5.py"}, args...), TimeoutSeconds: 60}
	}
	c2 := externalConfig(t, replay("llm", recordings["llm"]...), replay("fix", recordings["fix"]...))
	c2.Seeds, c2.Evaluation.T90 = c.Seeds, false
	a, b := runMatrix(t, c2), runMatrix(t, c2)
	if m.count("pvrevise")+m.count("fixafter") != calls {
		t.Fatal("the replay called the model")
	}
	x, _ := os.ReadFile(filepath.Join(a, FileManifest))
	y, _ := os.ReadFile(filepath.Join(b, FileManifest))
	if !bytes.Equal(x, y) {
		t.Fatal("two replays differ")
	}
	_, _, recs, _ := LoadResults(a)
	checked := map[string]int{}
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
		if r.Policy.ID == "noop" {
			continue
		}
		rd := readJSONL[map[string]any](t, filepath.Join(a, "runs", r.ID, FileEvidence, "decisions.jsonl"))
		ld := readJSONL[map[string]any](t, filepath.Join(live, "runs", r.ID, FileEvidence, "decisions.jsonl"))
		for k := range ld {
			chk, _ := rd[k]["replayChecked"].(map[string]any)
			if chk == nil {
				t.Fatalf("%s: point %d: not checked", r.ID, k)
			}
			for _, n := range []string{"inputs", "validations", "previews"} {
				checked[n] += int(chk[n].(float64))
			}
			for _, f := range []string{"attempts", "final", "batches", "stages", "previousExecution",
				"digestsBeforeSubmit", "completeness", "promptSha256"} {
				p, _ := json.Marshal(ld[k][f])
				q, _ := json.Marshal(rd[k][f])
				if !bytes.Equal(p, q) {
					t.Errorf("%s: point %d: %s differs", r.ID, k, f)
				}
			}
		}
	}
	if checked["validations"] == 0 || checked["previews"] == 0 || checked["inputs"] == 0 {
		t.Fatalf("checked %v", checked)
	}

	// The first decision point with a validated, previewed first attempt.
	one := recordings["llm"][1]
	raw, _ := os.ReadFile(one)
	at := -1
	for k, l := range bytes.Split(bytes.TrimSpace(raw), []byte("\n")) {
		var d agentRecord5
		json.Unmarshal(l, &d)
		if at < 0 && len(d.Attempts) > 0 && d.Attempts[0].Preview != nil && len(d.Attempts[0].Validation) > 0 {
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
	validation := tamper(func(d map[string]any) {
		v := first(d)["validation"].([]any)[0].(map[string]any)["answer"].(map[string]any)
		v["results"].([]any)[0].(map[string]any)["message"] = "edited"
	})
	dropped := tamper(func(d map[string]any) { first(d)["preview"] = nil })
	v4 := filepath.Join(t.TempDir(), "decisions.jsonl")
	{
		c := externalConfig(t, llmPolicy4(t, "llm", "pvkeep", nil))
		c.Evaluation.T90 = false
		dir := runMatrix(t, c)
		_, _, rs, _ := LoadResults(dir)
		raw, _ := os.ReadFile(filepath.Join(dir, "runs", rs[1].ID, FileEvidence, "decisions.jsonl"))
		os.WriteFile(v4, raw, 0o644)
	}
	for _, tc := range []struct{ file, want string }{
		{validation, fmt.Sprintf("decision point %d, attempt 1: validate answers batch 0 differently from the recording", at)},
		{dropped, fmt.Sprintf("decision point %d, attempt 1: a preview missing at this attempt", at)},
		{v4, "version 4 recordings replay with agents/llm/llm_agent4.py"},
	} {
		c3 := externalConfig(t, replay("llm", "--replay", tc.file))
		c3.Evaluation.T90 = false
		dir := runMatrix(t, c3)
		_, _, rs, _ := LoadResults(dir)
		stderr, _ := os.ReadFile(filepath.Join(dir, "runs", rs[1].ID, FileStderr))
		if rs[1].Status != StatusCrash || !strings.Contains(string(stderr), tc.want) {
			t.Errorf("%s: %s, stderr %q", tc.want, rs[1].Status, stderr)
		}
	}
}

// The default workflow (no --workflow) and --workflow validation-first:
// validation first and the simulator's results after a refusal, as with
// the preview, but no preview is ever asked for; a validated proposal is
// submitted. The recording names its workflow and replays with it, without
// a model; a --workflow that contradicts the recording is refused.
func TestLLMAgent5ValidationFirst(t *testing.T) {
	m := llmSetup(t)
	c := externalConfig(t, llmPolicy5(t, "default", "pvkeep", nil), llmPolicy5(t, "keep", "pvkeep", nil, workflowVF),
		llmPolicy5(t, "fix", "fixafter", nil, workflowVF), llmPolicy5(t, "mixed", "mixed", nil, workflowVF))
	c.Seeds, c.Evaluation.T90 = []int64{privateSeed, 5}, false
	live := runMatrix(t, c)
	_, _, liveRecs, err := LoadResults(live)
	if err != nil {
		t.Fatal(err)
	}
	recordings := map[string][]string{}
	fixed, submitted := 0, 0
	for _, r := range liveRecs {
		if r.Status != StatusCompleted {
			t.Fatalf("%s: %s %+v", r.ID, r.Status, r.Failure)
		}
		id := r.Policy.ID
		if id == "noop" {
			continue
		}
		path := filepath.Join(live, "runs", r.ID, FileEvidence, "decisions.jsonl")
		recordings[id] = append(recordings[id], "--replay", path)
		var info map[string]any
		if err := readJSON(filepath.Join(live, "runs", r.ID, FileEvidence, "agent.json"), &info); err != nil ||
			info["workflow"] != workflowVF || info["preview"] != false {
			t.Fatalf("%s: agent.json: %v %v", r.ID, err, info)
		}
		var harness []Decision
		if err := readJSON(filepath.Join(live, "runs", r.ID, FileDecisions), &harness); err != nil {
			t.Fatal(err)
		}
		for k, d := range readJSONL[agentRecord5](t, path) {
			if d.Workflow != workflowVF || d.WorkflowVersion != "llm-agent-validation-first-workflow/v1" ||
				!d.DigestsBeforeSubmit.Unchanged {
				t.Fatalf("%s: point %d: workflow %q (%s)", r.ID, d.Point, d.Workflow, d.WorkflowVersion)
			}
			if d.Stages.Previewed != nil {
				t.Errorf("%s: point %d: a preview stage", r.ID, d.Point)
			}
			for i, a := range d.Attempts {
				if a.Preview != nil || a.Input["kind"] == "preview" || a.Turn == "review" {
					t.Errorf("%s: point %d, attempt %d: a preview with validation-first", r.ID, d.Point, i+1)
				}
				if len(a.Refusals) > 0 {
					if i+1 >= len(d.Attempts) || d.Attempts[i+1].Input["kind"] != "validation" {
						if i+1 < len(d.Attempts) {
							t.Errorf("%s: point %d: a refusal not answered with validation feedback", r.ID, d.Point)
						}
					} else {
						fixed++
					}
				}
				// A valid proposal is used at once: nothing after it.
				if len(a.Validation) > 0 && len(a.Refusals) == 0 && (!a.Used || i != len(d.Attempts)-1) {
					t.Errorf("%s: point %d: a validated proposal was not submitted", r.ID, d.Point)
				}
			}
			if len(d.Batches) > 0 {
				submitted++
				for _, b := range harness[k].Batches {
					if !b.Applied {
						t.Errorf("%s: point %d: the harness refused a validated batch", r.ID, d.Point)
					}
				}
			}
		}
		b, _ := os.ReadFile(path)
		if strings.Contains(string(b), "PREVIEW.") {
			t.Errorf("%s: a preview message in the recording", r.ID)
		}
	}
	if fixed == 0 || submitted == 0 {
		t.Fatalf("fixed %d, submitted %d", fixed, submitted)
	}
	// No engine preview request: the mock served exactly the calls recorded,
	// one per point for pvkeep (its preview answer is never asked for).
	calls := m.count("pvkeep") + m.count("fixafter") + m.count("mixed")

	replay := func(id string, args ...string) ExternalPolicy {
		return ExternalPolicy{ID: id, Version: "5", Description: "llm-agent 5, replayed",
			Command: append([]string{"python3", "-I", "-B", "./agents/llm/llm_agent5.py"}, args...), TimeoutSeconds: 60}
	}
	c2 := externalConfig(t, replay("default", recordings["default"]...), replay("keep", recordings["keep"]...),
		replay("fix", recordings["fix"]...), replay("mixed", recordings["mixed"]...))
	c2.Seeds, c2.Evaluation.T90 = c.Seeds, false
	a := runMatrix(t, c2)
	if m.count("pvkeep")+m.count("fixafter")+m.count("mixed") != calls {
		t.Fatal("the replay called the model")
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
		if r.Policy.ID == "noop" {
			continue
		}
		for _, d := range readJSONL[map[string]any](t, filepath.Join(a, "runs", r.ID, FileEvidence, "decisions.jsonl")) {
			if d["workflow"] != workflowVF || d["replayChecked"] == nil {
				t.Fatalf("%s: a replayed record: workflow %v, checked %v", r.ID, d["workflow"], d["replayChecked"])
			}
		}
	}

	// The command line never overrides the recording's workflow.
	c3 := externalConfig(t, replay("keep", append(recordings["keep"][:2:2], "--workflow", workflowPV)...))
	c3.Evaluation.T90 = false
	dir := runMatrix(t, c3)
	_, _, rs, _ := LoadResults(dir)
	stderr, _ := os.ReadFile(filepath.Join(dir, "runs", rs[1].ID, FileStderr))
	if rs[1].Status != StatusCrash || !strings.Contains(string(stderr), "the recordings were made with validation-first") {
		t.Errorf("a contradicting --workflow: %s, stderr %q", rs[1].Status, stderr)
	}
}
