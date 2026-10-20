package sim_test

import (
	"testing"

	"github.com/georgezero/dreamteam-go/internal/eval"
)

// Audit 5C.4m M6: the simulator validation helper recorded its records
// with an empty scenario fingerprint, which the candidate's integrity
// check refuses, so "every seed is valid" was not established by them.
// The helper now records the evaluation scenario's own fingerprint and
// refuses any record that fails integrity; here a record of the helper,
// under the candidate and under v0.2-eval-4, carries the policy's
// fingerprint and passes every integrity check.
func TestValidationHelperIntegrity(t *testing.T) {
	cand, _, err := eval.LoadPolicy("../../" + candidatePolicy)
	if err != nil {
		t.Fatal(err)
	}
	pol4, _, err := eval.LoadPolicy("../../" + eval.PolicyPath)
	if err != nil {
		t.Fatal(err)
	}
	cfg, fp := profileConfig(t)
	if fp != cand.Scenario.Fingerprint || fp != pol4.Scenario.Fingerprint {
		t.Fatalf("the evaluation scenario's fingerprint %s; the candidate's %s, v4's %s", fp, cand.Scenario.Fingerprint, pol4.Scenario.Fingerprint)
	}
	r := recordBaseline(cand, cfg, fp, 901)
	if r.ScenarioFingerprint != fp {
		t.Fatalf("recorded fingerprint %q", r.ScenarioFingerprint)
	}
	if bad := eval.Measure(cand, &r).Integrity; len(bad) > 0 {
		t.Fatalf("the helper's record fails integrity: %v", bad)
	}
	// The record the helper made before (no fingerprint) fails, as audit
	// 5C.4m reproduced, and the helper refuses to return it.
	r.ScenarioFingerprint = ""
	if bad := eval.Measure(cand, &r).Integrity; len(bad) == 0 {
		t.Fatal("a record without the scenario's fingerprint passes integrity")
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Error("the helper returned a record that fails integrity")
			}
		}()
		recordBaseline(cand, cfg, "", 902)
	}()
}
