package scenario

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"

	"github.com/georgezero/dreamteam-go/docs/evaluation"
)

// The bundled sustainable scenario is exactly the practice the evaluation
// policy in force is defined for: the Step 0C shared-staffing control at
// ×1.30 demand (V0.2_DESIGN §5.6, W130-SHR-MRft), the established practice
// with that experiment's overlay, byte for byte. Its internal id stays
// "established": the policy pins those exact bytes (its fingerprint).
func TestSustainableIsThePolicyScenario(t *testing.T) {
	f, err := ResolveFile("sustainable")
	if err != nil {
		t.Fatal(err)
	}
	fp, err := Fingerprint(f)
	if err != nil {
		t.Fatal(err)
	}
	var pol struct {
		Scenario struct {
			Fingerprint string `json:"fingerprint"`
		} `json:"scenario"`
	}
	if err := json.Unmarshal(evaluation.PolicyV4, &pol); err != nil {
		t.Fatal(err)
	}
	if fp != pol.Scenario.Fingerprint || fp != "e8718dbd1031feb326b4934d291f81eb14fd867d4a8fa6acc1673a75b55b76f8" {
		t.Fatalf("sustainable %s, the policy's %s", fp, pol.Scenario.Fingerprint)
	}
	// The Step 0C experiment's overlay on the established practice.
	raw, err := os.ReadFile("../../docs/calibration/feasibility/step0c/experiments/W130-SHR-MRft-s20.json")
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		Scenario string          `json:"scenario"`
		Overlay  json.RawMessage `json:"overlay"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	base, err := ResolveFile(m.Scenario)
	if err != nil {
		t.Fatal(err)
	}
	canon, _ := Canonical(base)
	var b, o any
	json.Unmarshal(canon, &b)
	json.Unmarshal(m.Overlay, &o)
	merged, _ := json.Marshal(Merge(b, o))
	g, err := Decode(merged)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := Canonical(g)
	got, _ := Canonical(f)
	if !bytes.Equal(got, want) {
		t.Fatal("the bundled scenario is not the Step 0C control")
	}
	if len(f.Employees) != len(base.Employees)+1 {
		t.Fatalf("%d employees, the established practice's %d plus Alex", len(f.Employees), len(base.Employees))
	}
	cfg, err := Compile(f)
	if err != nil || cfg == nil {
		t.Fatal(err)
	}
}

// The game embeds the evaluation policy in force and the holdout manifest
// byte for byte as committed.
func TestEmbeddedEvaluationFiles(t *testing.T) {
	for name, got := range map[string][]byte{"policy-v0.2-eval-4.json": evaluation.PolicyV4, "holdout-v0.2.json": evaluation.Holdout} {
		want, err := os.ReadFile("../../docs/evaluation/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s: the embedded copy differs", name)
		}
	}
	sum := sha256.Sum256(evaluation.PolicyV4)
	if d := hex.EncodeToString(sum[:]); d != "4e277c7263dcff7ddabe63741beb44abfc9a46d3bc23f6b6418715b58ffb6fb3" {
		t.Fatalf("policy v0.2-eval-4 digest %s", d)
	}
}
