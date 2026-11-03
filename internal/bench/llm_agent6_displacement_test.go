package bench

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// llm-agent 6 against the engine where a termination displaces booked
// appointments (V0.12b Stage A). In the mini scenario, terminations at
// the third decision point (last day 2026-10-29) take bookings away; the
// mock model (ackdisp*, internal/bench/llm_agent_test.go) proposes one or
// two there. The test checks, against the engine's own answers:
//
//   - the consequence items carry the engine's exact displaced-booking
//     count as one batch-level item over every action of the batch, and
//     each termination as its own item;
//   - an acknowledgment that leaves out the displaced bookings, or states
//     one fewer, submits nothing, and the session ends exactly where the
//     no-op policy's does (validation and preview changed nothing);
//   - a proposal changed after an acknowledgment is not submitted on that
//     acknowledgment; the changed proposal is submitted only once its own
//     consequences are acknowledged;
//   - the final re-preview agrees with what was acknowledged;
//   - the applied termination displaces exactly the acknowledged number
//     of bookings (the run's displacement events);
//   - the replay, without a model, reproduces every run byte for byte.
func TestLLMAgent6Displacement(t *testing.T) {
	m := llmSetup(t)
	modes := []string{"ackdisp", "ackdisp2", "ackdispnone", "ackdispoff", "ackdispswap"}
	var pols []ExternalPolicy
	for _, mode := range modes {
		p := llmPolicy(t, mode, mode, map[string]any{"maxRetries": 2})
		p.Command[3], p.Version = "./agents/llm/llm_agent6.py", "6"
		pols = append(pols, p)
	}
	c := externalConfig(t, pols...)
	c.Seeds, c.Days, c.Evaluation.T90 = []int64{privateSeed, 5}, 4, false
	live := runMatrix(t, c)
	_, _, liveRecs, err := LoadResults(live)
	if err != nil {
		t.Fatal(err)
	}
	metric := func(r *RunRecord, name string) int {
		for _, v := range r.Metrics {
			if v.Name == name {
				var n int
				json.Unmarshal(v.Value, &n)
				return n
			}
		}
		return 0
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
			if n := metric(r, "displacedAppointments"); n != 0 {
				t.Fatalf("%s: the no-op run displaced %d", r.ID, n)
			}
			continue
		}
		path := filepath.Join(live, "runs", r.ID, FileEvidence, "decisions.jsonl")
		recordings[id] = append(recordings[id], "--replay", path)
		var harness []Decision
		if err := readJSON(filepath.Join(live, "runs", r.ID, FileDecisions), &harness); err != nil {
			t.Fatal(err)
		}
		recs := readJSONL[agentRecord6](t, path)
		if len(recs) != 4 {
			t.Fatalf("%s: %d decision records", r.ID, len(recs))
		}
		for k, d := range recs {
			if d.Point != 2 {
				if len(d.Batches) != 0 || len(harness[k].Batches) != 0 {
					t.Fatalf("%s: point %d: something was submitted", r.ID, d.Point)
				}
				continue
			}
			a := d.Attempts[0].Assessment
			if a == nil || a.Status != "qualifying" || !a.DigestsUnchanged {
				t.Fatalf("%s: point 2: assessment %+v", r.ID, a)
			}
			pv := a.Answer["preview"].(map[string]any)
			disp, _ := pv["displacements"].([]any)
			terms, _ := pv["terminations"].([]any)
			want := 1
			if id == "ackdisp2" {
				want = 2
			}
			if len(disp) == 0 || len(terms) != want {
				t.Fatalf("%s: point 2: %d displacements, %d terminations", r.ID, len(disp), len(terms))
			}
			// One item per termination, then one batch item over every
			// action carrying the whole batch's count, and nothing else.
			if len(a.Items) != want+1 {
				t.Fatalf("%s: point 2: items %+v", r.ID, a.Items)
			}
			for i, it := range a.Items[:want] {
				tp := terms[i].(map[string]any)
				if it.Type != "terminate" || it.Action == nil || *it.Action != i || it.Target != tp["employee"] ||
					it.Consequences["severance"]["severanceCents"] != tp["severanceCents"] ||
					it.Consequences["termination"]["lastDay"] != tp["lastDay"] || it.Consequences["displaced_bookings"] != nil {
					t.Errorf("%s: item %d is not the engine's termination %v: %+v", r.ID, i, tp, it)
				}
				if int(tp["displaced"].(float64)) != len(disp) {
					t.Errorf("%s: the termination's batch count %v is not the batch's %d", r.ID, tp["displaced"], len(disp))
				}
			}
			bi := a.Items[want]
			wantActs := []int{0}
			if want == 2 {
				wantActs = []int{0, 1}
			}
			if bi.Type != "staffing_batch" || bi.Action != nil || !slices.Equal(bi.Actions, wantActs) ||
				int(bi.Consequences["displaced_bookings"]["count"].(float64)) != len(disp) {
				t.Errorf("%s: the batch item is not the engine's %d displacements over %v: %+v", r.ID, len(disp), wantActs, bi)
			}
			// What was submitted, and on what acknowledgment.
			applied := 0
			for _, b := range harness[k].Batches {
				if b.Applied {
					applied++
				}
			}
			var used *int
			for i, at := range d.Attempts {
				if at.Used {
					used = &i
				}
			}
			switch id {
			case "ackdisp", "ackdisp2", "ackdispswap":
				if used == nil || len(d.Batches) != 1 || applied != 1 || len(harness[k].Batches) != 1 {
					t.Fatalf("%s: point 2: not submitted: %+v", r.ID, d)
				}
				u := d.Attempts[*used]
				if u.Acknowledgment == nil || !u.Acknowledgment.Match.Accepted || u.FinalCheck == nil || !u.FinalCheck.Agrees {
					t.Fatalf("%s: point 2: submitted without an accepted, agreeing acknowledgment", r.ID)
				}
				// The final re-preview is the acknowledged one: the same
				// engine answer and consequence payload.
				fa := d.Attempts[u.Acknowledgment.ForAttempt-1].Assessment // forAttempt counts from 1
				if fa == nil || fa.Payload == nil || u.FinalCheck.AssessmentSha256 != fa.AnswerSha256 ||
					u.FinalCheck.PayloadSha256 != fa.Payload.Sha256 {
					t.Fatalf("%s: point 2: the final check is not of the acknowledged assessment", r.ID)
				}
				fdisp, _ := fa.Answer["preview"].(map[string]any)["displacements"].([]any)
				got := metric(r, "displacedAppointments")
				if got != len(fdisp) || got == 0 {
					t.Errorf("%s: the termination displaced %d, the acknowledged preview %d", r.ID, got, len(fdisp))
				}
				if id == "ackdispswap" {
					// The first acknowledgment was for the first proposal:
					// it is not accepted for the second, which is submitted
					// on its own acknowledgment, terminating someone else.
					first := d.Attempts[1]
					if first.Acknowledgment == nil || first.Acknowledgment.Match.Accepted {
						t.Fatalf("%s: the first proposal's acknowledgment was accepted for the second", r.ID)
					}
					sent := d.Batches[0][0]["employee"]
					if sent == terms[0].(map[string]any)["employee"] || *used < 2 {
						t.Errorf("%s: submitted %v at attempt %d", r.ID, sent, *used)
					}
					codes := failureCodes(first.Acknowledgment.Match.Failures)
					if !slices.Contains(codes, "proposal_changed") && !slices.Contains(codes, "wrong_proposal") {
						t.Errorf("%s: the changed proposal's failures %v", r.ID, codes)
					}
				}
				t.Logf("%s: proposed %d terminations displacing %d; submitted %v at attempt %d on an accepted acknowledgment; %d displaced when applied", r.ID, len(terms), len(disp), d.Batches[0], *used+1, got)
				checked[id]++
			case "ackdispnone", "ackdispoff":
				code := map[string]string{"ackdispnone": "missing_item", "ackdispoff": "wrong_value"}[id]
				rejected := 0
				for _, at := range d.Attempts {
					if at.Acknowledgment != nil {
						if at.Acknowledgment.Match.Accepted ||
							!slices.Contains(failureCodes(at.Acknowledgment.Match.Failures), code) {
							t.Fatalf("%s: an acknowledgment %+v", r.ID, at.Acknowledgment.Match)
						}
						rejected++
					}
				}
				if used != nil || len(d.Batches) != 0 || len(harness[k].Batches) != 0 || rejected != 3 ||
					d.Withheld == nil || d.Withheld.Reason != "acknowledgment_not_accepted" {
					t.Fatalf("%s: point 2: submitted or not withheld: %+v", r.ID, d)
				}
				// Nothing submitted, nothing displaced: the session is
				// exactly where the no-op policy left it.
				if metric(r, "displacedAppointments") != 0 || r.Final.Digests != noop[r.Seed].Final.Digests {
					t.Errorf("%s: the session differs from the no-op run's", r.ID)
				}
				t.Logf("%s: %d terminations displacing %d; %d acknowledgments rejected (%s); nothing submitted; final digests equal the no-op run's", r.ID, len(terms), len(disp), rejected, code)
				checked[id]++
			}
		}
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

func failureCodes(fs []map[string]any) []string {
	var out []string
	for _, f := range fs {
		if c, ok := f["code"].(string); ok {
			out = append(out, c)
		}
	}
	return out
}
