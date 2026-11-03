package bench

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/sim"
)

// llm-agent 6 against the engine with a hire, a new retained obligation
// (V0.12b, audit finding F4). The mock model (ackhire*,
// internal/bench/llm_agent_test.go) hires an XR technologist at the first
// decision point of the mini scenario. The test checks, against the
// engine's own answer and offer table:
//
//   - the one consequence item is the hire, identified by its requestId,
//     with the engine's exact recruiting fee (the offer table's, and the
//     preview's "Recruiting fee" line) and earliest-exit cost (the
//     preview's "Earliest lawful exit" line);
//   - an acknowledgment of nothing, or of a fee $1 higher, submits nothing,
//     and the session ends exactly where the no-op policy's does;
//   - an accurate acknowledgment is submitted after a final check that
//     re-validates and re-previews the same answer, and the engine applies
//     the hire;
//   - the replay, without a model, reproduces every run byte for byte.
func TestLLMAgent6Hire(t *testing.T) {
	m := llmSetup(t)
	modes := []string{"ackhire", "ackhirenone", "ackhireoff"}
	var pols []ExternalPolicy
	for _, mode := range modes {
		p := llmPolicy(t, mode, mode, map[string]any{"maxRetries": 2})
		p.Command[3], p.Version = "./agents/llm/llm_agent6.py", "6"
		pols = append(pols, p)
	}
	c := externalConfig(t, pols...)
	c.Seeds, c.Days, c.Evaluation.T90 = []int64{privateSeed, 5}, 2, false
	live := runMatrix(t, c)
	_, _, liveRecs, err := LoadResults(live)
	if err != nil {
		t.Fatal(err)
	}
	fee := int64(-1)
	for _, o := range sim.Offers {
		if o.Role == sim.Technologist && slices.Equal(o.Skills, []string{"XR"}) {
			fee = o.FeeCents
		}
	}
	noop := map[int64]*RunRecord{}
	for _, r := range liveRecs {
		if r.Policy.ID == "noop" {
			noop[r.Seed] = r
		}
	}
	recordings := map[string][]string{}
	checked := map[string]int{}
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
		d := readJSONL[agentRecord6](t, path)[0]
		a := d.Attempts[0].Assessment
		if d.Point != 0 || a == nil || a.Status != "qualifying" || !a.DigestsUnchanged {
			t.Fatalf("%s: point 0: assessment %+v", r.ID, a)
		}
		pv := a.Answer["preview"].(map[string]any)
		hires := pv["hires"].([]any)
		guar := pv["guaranteed"].([]any)
		line := func(k int) map[string]any { return guar[k].(map[string]any) }
		offer := hires[0].(map[string]any)["offer"]
		if len(hires) != 1 || hires[0].(map[string]any)["requestId"] != "hire-xr-1" ||
			line(0)["label"] != "Recruiting fee · "+offer.(string) || line(2)["label"] != "Earliest lawful exit · "+offer.(string) {
			t.Fatalf("%s: the engine's hire answer: %v", r.ID, pv)
		}
		if len(a.Items) != 1 {
			t.Fatalf("%s: items %+v", r.ID, a.Items)
		}
		it := a.Items[0]
		ro := it.Consequences["retained_obligation"]
		if it.Type != "hire" || it.Action == nil || *it.Action != 0 || it.Target != "hire-xr-1" || len(ro) != 2 ||
			ro["recruitingFeeCents"] != line(0)["cents"] || ro["earliestExitCents"] != line(2)["cents"] ||
			int64(ro["recruitingFeeCents"].(float64)) != fee || ro["earliestExitCents"].(float64) <= 0 {
			t.Errorf("%s: the hire item is not the engine's (fee %d, lines %v, %v): %+v", r.ID, fee, line(0), line(2), it)
		}
		var used *int
		for i, at := range d.Attempts {
			if at.Used {
				used = &i
			}
		}
		switch id {
		case "ackhire":
			if used == nil || len(d.Batches) != 1 || len(harness[0].Batches) != 1 || !harness[0].Batches[0].Applied {
				t.Fatalf("%s: the acknowledged hire was not submitted and applied: %+v", r.ID, d)
			}
			u := d.Attempts[*used]
			fa := d.Attempts[u.Acknowledgment.ForAttempt-1].Assessment // forAttempt counts from 1
			if !u.Acknowledgment.Match.Accepted || u.FinalCheck == nil || !u.FinalCheck.Agrees ||
				u.FinalCheck.AssessmentSha256 != fa.AnswerSha256 || u.FinalCheck.PayloadSha256 != fa.Payload.Sha256 {
				t.Fatalf("%s: submitted without an accepted acknowledgment and an agreeing final check", r.ID)
			}
			res := harness[0].Batches[0].Results
			if len(res) != 1 || !res[0].OK || res[0].EmployeeID == "" {
				t.Errorf("%s: the hire's result %+v", r.ID, res)
			}
			t.Logf("%s: hire-xr-1 (%v) acknowledged with fee %v and earliest exit %v; applied as %s", r.ID, offer,
				ro["recruitingFeeCents"], ro["earliestExitCents"], res[0].EmployeeID)
		case "ackhirenone", "ackhireoff":
			code := map[string]string{"ackhirenone": "missing_item", "ackhireoff": "wrong_value"}[id]
			rejected := 0
			for _, at := range d.Attempts {
				if at.Acknowledgment != nil {
					if at.Acknowledgment.Match.Accepted || !slices.Contains(failureCodes(at.Acknowledgment.Match.Failures), code) {
						t.Fatalf("%s: an acknowledgment %+v", r.ID, at.Acknowledgment.Match)
					}
					rejected++
				}
			}
			if used != nil || len(d.Batches) != 0 || len(harness[0].Batches) != 0 || rejected != 3 ||
				d.Withheld == nil || d.Withheld.Reason != "acknowledgment_not_accepted" ||
				r.Final.Digests != noop[r.Seed].Final.Digests {
				t.Fatalf("%s: submitted, not withheld, or the session differs from the no-op run's: %+v", r.ID, d)
			}
			t.Logf("%s: 3 acknowledgments rejected (%s); withheld; final digests equal the no-op run's", r.ID, code)
		}
		checked[id]++
	}
	for _, id := range modes {
		if checked[id] != 2 {
			t.Fatalf("checked %v", checked)
		}
	}

	// The replay, without a model, reproduces every run.
	calls := 0
	for _, id := range modes {
		calls += m.count(id)
	}
	var reps []ExternalPolicy
	for _, id := range modes {
		reps = append(reps, ExternalPolicy{ID: id, Version: "6", Description: "llm-agent 6, replayed",
			Command: append([]string{"python3", "-I", "-B", "./agents/llm/llm_agent6.py"}, recordings[id]...), TimeoutSeconds: 60})
	}
	c2 := externalConfig(t, reps...)
	c2.Seeds, c2.Days, c2.Evaluation.T90 = c.Seeds, c.Days, false
	a := runMatrix(t, c2)
	after := 0
	for _, id := range modes {
		after += m.count(id)
	}
	if after != calls {
		t.Fatal("the replay called the model")
	}
	_, _, recs, _ := LoadResults(a)
	for i, r := range recs {
		if r.Status != StatusCompleted || r.Final.Digests != liveRecs[i].Final.Digests {
			t.Fatalf("%s: %s, final digests differ from the live run's", r.ID, r.Status)
		}
		for _, f := range []string{FileResult, FileSession, FileDecisions} {
			p, _ := os.ReadFile(filepath.Join(live, "runs", r.ID, f))
			q, _ := os.ReadFile(filepath.Join(a, "runs", r.ID, f))
			if !bytes.Equal(p, q) {
				t.Errorf("%s: %s differs from the live run's", r.ID, f)
			}
		}
	}
}
