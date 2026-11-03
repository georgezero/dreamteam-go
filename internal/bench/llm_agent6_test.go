package bench

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// llm-agent 6 (agents/llm/llm_agent6.py, V0.12a) against the engine and the
// mock model server: a termination and a new retained obligation (a
// teleradiology contract) are submitted only after the model acknowledges
// the consequences the engine reports for exactly that proposal; a generic
// acknowledgment withholds the termination. The replay re-runs the
// workflow with the recorded replies, without a model, and refuses a
// tampered recording.

func llmPolicy6(t *testing.T, id, model string) ExternalPolicy {
	p := llmPolicy(t, id, model, nil)
	p.Command[3], p.Version = "./agents/llm/llm_agent6.py", "6"
	return p
}

type ackItem struct {
	ID           string                    `json:"id"`
	Action       *int                      `json:"action"`
	Type         string                    `json:"type"`
	Target       any                       `json:"target"`
	Consequences map[string]map[string]any `json:"consequences"`
}

// agentRecord6 is one line of llm-agent 6's decisions.jsonl, in part.
type agentRecord6 struct {
	Schema   string             `json:"schema"`
	Workflow string             `json:"workflow"`
	Contract string             `json:"contract"`
	Point    int                `json:"point"`
	Outcome  string             `json:"outcome"`
	Batches  [][]map[string]any `json:"batches"`
	Withheld *struct {
		Reason string `json:"reason"`
	} `json:"withheld"`
	Attempts []struct {
		Input      map[string]any `json:"input"`
		Assessment *struct {
			Status           string         `json:"status"`
			Answer           map[string]any `json:"answer"`
			Items            []ackItem      `json:"items"`
			DigestsUnchanged bool           `json:"digestsUnchanged"`
		} `json:"assessment"`
		Acknowledgment *struct {
			Match struct {
				Accepted bool             `json:"accepted"`
				Failures []map[string]any `json:"failures"`
			} `json:"match"`
		} `json:"acknowledgment"`
		FinalCheck *struct {
			Agrees bool `json:"agrees"`
		} `json:"finalCheck"`
		Used bool `json:"used"`
	} `json:"attempts"`
	DigestsBeforeSubmit struct {
		Unchanged bool `json:"unchanged"`
	} `json:"digestsBeforeSubmit"`
}

func TestLLMAgent6Acknowledgment(t *testing.T) {
	m := llmSetup(t)
	c := externalConfig(t, llmPolicy6(t, "fire", "ackfire"), llmPolicy6(t, "tele", "acktele"),
		llmPolicy6(t, "generic", "ackgeneric"), llmPolicy6(t, "all", "ackfireall"))
	c.Seeds, c.Evaluation.T90 = []int64{privateSeed, 5}, false
	live := runMatrix(t, c)
	_, _, liveRecs, err := LoadResults(live)
	if err != nil {
		t.Fatal(err)
	}
	recordings := map[string][]string{}
	seen := map[string]int{}
	for _, r := range liveRecs {
		if r.Status != StatusCompleted {
			stderr, _ := os.ReadFile(filepath.Join(live, "runs", r.ID, FileStderr))
			t.Fatalf("%s: %s %+v\n%s", r.ID, r.Status, r.Failure, stderr)
		}
		id := r.Policy.ID
		if id == "noop" {
			continue
		}
		path := filepath.Join(live, "runs", r.ID, FileEvidence, "decisions.jsonl")
		recordings[id] = append(recordings[id], "--replay", path)
		var harness []Decision
		if err := readJSON(filepath.Join(live, "runs", r.ID, FileDecisions), &harness); err != nil {
			t.Fatal(err)
		}
		for k, d := range readJSONL[agentRecord6](t, path) {
			if d.Schema != "dreamteam.llm-decision-record/v6" || d.Workflow != "acknowledgment" ||
				d.Contract != "dreamteam.action-acknowledgment/v1" || !d.DigestsBeforeSubmit.Unchanged {
				t.Fatalf("%s: point %d: %s %s %s", r.ID, d.Point, d.Schema, d.Workflow, d.Contract)
			}
			// Nothing that needed acknowledgment is submitted without an
			// accepted one and a final check that agrees.
			qualifying, acknowledged := false, false
			for _, a := range d.Attempts {
				if a.Assessment != nil && a.Assessment.Status == "qualifying" {
					qualifying = true
				}
				if a.Used && a.Acknowledgment != nil && a.Acknowledgment.Match.Accepted && a.FinalCheck != nil &&
					a.FinalCheck.Agrees {
					acknowledged = true
				}
			}
			if len(d.Batches) > 0 && qualifying && !acknowledged {
				t.Errorf("%s: point %d: a qualifying proposal submitted without acknowledgment", r.ID, d.Point)
			}
			for _, b := range harness[k].Batches {
				if !b.Applied {
					t.Errorf("%s: point %d: the harness refused a batch", r.ID, d.Point)
				}
			}
			if d.Point != 0 || len(d.Attempts) == 0 || d.Attempts[0].Assessment == nil {
				continue
			}
			a := d.Attempts[0].Assessment
			pv, _ := a.Answer["preview"].(map[string]any)
			if a.Status != "qualifying" || pv == nil || !a.DigestsUnchanged {
				t.Fatalf("%s: point 0: assessment %+v", r.ID, a)
			}
			// Every item's values are the engine's own.
			for _, it := range a.Items {
				switch it.Type {
				case "terminate":
					var tp map[string]any
					for _, x := range pv["terminations"].([]any) {
						if x.(map[string]any)["employee"] == it.Target {
							tp = x.(map[string]any)
						}
					}
					if tp == nil || it.Consequences["severance"]["severanceCents"] != tp["severanceCents"] ||
						it.Consequences["termination"]["lastDay"] != tp["lastDay"] {
						t.Errorf("%s: the termination item is not the engine's: %+v / %v", r.ID, it, tp)
					}
					seen["termination"]++
				case "contract_teleradiology":
					tp := pv["teleradiology"].([]any)[0].(map[string]any)
					ro := it.Consequences["retained_obligation"]
					if ro["guaranteedCents"] != tp["guaranteedCents"] || ro["guaranteedThrough"] != tp["guaranteedThrough"] {
						t.Errorf("%s: the obligation item is not the engine's: %+v / %v", r.ID, it, tp)
					}
					seen["obligation"]++
				case "staffing_batch":
					if int(it.Consequences["displaced_bookings"]["count"].(float64)) != len(pv["displacements"].([]any)) {
						t.Errorf("%s: the displacement count is not the engine's", r.ID)
					}
					seen["displacement"]++
				}
			}
			if len(d.Attempts) < 2 || d.Attempts[1].Input["kind"] != "acknowledgment" || d.Attempts[1].Acknowledgment == nil {
				t.Fatalf("%s: point 0: no acknowledgment requested", r.ID)
			}
			ack := d.Attempts[1].Acknowledgment.Match
			switch id {
			case "fire", "tele", "all":
				if !ack.Accepted || len(d.Batches) != 1 || len(harness[k].Batches) != 1 || !harness[k].Batches[0].Applied {
					t.Errorf("%s: point 0: acknowledged %v, batches %d", r.ID, ack.Accepted, len(d.Batches))
				}
				seen["submitted"]++
			case "generic":
				if ack.Accepted || ack.Failures[0]["code"] != "malformed" || len(d.Batches) != 0 || d.Withheld == nil ||
					d.Withheld.Reason != "acknowledgment_not_accepted" || len(harness[k].Batches) != 0 {
					t.Errorf("%s: point 0: a generic acknowledgment: %+v, withheld %+v", r.ID, ack, d.Withheld)
				}
				seen["withheld"]++
			}
		}
	}
	// "all": several terminations in one batch, each acknowledged. (The
	// mini scenario's terminations displace no booking: displacement items
	// from the real engine's answers are tested on the V0.11b recordings,
	// agents/llm/test_llm_agent6.py.)
	if seen["termination"] <= 6 || seen["obligation"] != 2 || seen["submitted"] != 6 || seen["withheld"] != 2 {
		t.Fatalf("seen %v", seen)
	}
	for _, r := range liveRecs {
		b, _ := os.ReadFile(filepath.Join(live, "runs", r.ID, FileDecisions))
		if r.Policy.ID == "generic" && strings.Contains(string(b), `"terminate"`) {
			t.Errorf("%s: a termination was submitted", r.ID)
		}
	}

	// The replay: the workflow re-run with the recorded replies, no model.
	calls := m.count("ackfire") + m.count("acktele") + m.count("ackgeneric") + m.count("ackfireall")
	replay := func(id string, args ...string) ExternalPolicy {
		return ExternalPolicy{ID: id, Version: "6", Description: "llm-agent 6, replayed",
			Command: append([]string{"python3", "-I", "-B", "./agents/llm/llm_agent6.py"}, args...), TimeoutSeconds: 60}
	}
	c2 := externalConfig(t, replay("fire", recordings["fire"]...), replay("tele", recordings["tele"]...),
		replay("generic", recordings["generic"]...), replay("all", recordings["all"]...))
	c2.Seeds, c2.Evaluation.T90 = c.Seeds, false
	a := runMatrix(t, c2)
	if m.count("ackfire")+m.count("acktele")+m.count("ackgeneric")+m.count("ackfireall") != calls {
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
			if d["replayChecked"] == nil || d["mode"] != "replay" {
				t.Fatalf("%s: a record not replayed", r.ID)
			}
		}
	}

	// A tampered consequence value, acknowledgment or action, and a
	// version 5 recording, are refused.
	raw, _ := os.ReadFile(recordings["fire"][1])
	tamper := func(f func(d map[string]any)) string {
		p := filepath.Join(t.TempDir(), "decisions.jsonl")
		lines := bytes.Split(bytes.TrimSpace(raw), []byte("\n"))
		var d map[string]any
		json.Unmarshal(lines[0], &d)
		f(d)
		lines[0], _ = json.Marshal(d)
		os.WriteFile(p, append(bytes.Join(lines, []byte("\n")), '\n'), 0o644)
		return p
	}
	att := func(d map[string]any, k int) map[string]any { return d["attempts"].([]any)[k].(map[string]any) }
	value := tamper(func(d map[string]any) {
		it := att(d, 0)["assessment"].(map[string]any)["items"].([]any)[0].(map[string]any)
		it["consequences"].(map[string]any)["severance"].(map[string]any)["severanceCents"] = 1
	})
	given := tamper(func(d map[string]any) {
		it := att(d, 1)["acknowledgment"].(map[string]any)["given"].(map[string]any)["items"].([]any)[0].(map[string]any)
		it["target"] = "E99"
	})
	action := tamper(func(d map[string]any) {
		d["batches"].([]any)[0].([]any)[0].(map[string]any)["employee"] = "E99"
	})
	v5 := tamper(func(d map[string]any) { d["schema"], d["agentVersion"] = "dreamteam.llm-decision-record/v5", "5" })
	for _, tc := range []struct{ file, want string }{
		{value, "differs from the recording at record.attempts[0].assessment.items[0].consequences.severance.severanceCents"},
		{given, "differs from the recording at record.attempts[1].acknowledgment.given.items[0].target"},
		{action, "differs from the recording at record.batches[0][0].employee"},
		{v5, "version 5 recordings replay with agents/llm/llm_agent5.py"},
	} {
		c3 := externalConfig(t, replay("fire", "--replay", tc.file))
		c3.Evaluation.T90 = false
		dir := runMatrix(t, c3)
		_, _, rs, _ := LoadResults(dir)
		stderr, _ := os.ReadFile(filepath.Join(dir, "runs", rs[1].ID, FileStderr))
		if rs[1].Status != StatusCrash || !strings.Contains(string(stderr), tc.want) {
			t.Errorf("%s: %s, stderr %q", tc.want, rs[1].Status, stderr)
		}
	}
}
