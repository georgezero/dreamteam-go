package eval

import (
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// Slice 5C.4n: the certification pipeline (audit 5C.4m M1, M2, M4). The
// auditor's witnesses are reproduced here: a fabricated, complete,
// all-PASS file of lines; missing and malformed measurements; malformed
// settings and blank output.

// fabricatedRun is a complete file's worth of lines, constructed without
// any record, that every mandatory criterion PASSes (as the 5C.4m
// auditor's 3,904 rows did): every witness as specified, 225 harmed
// seeds each flagged and failing, and the near-margin replicates
// consistent with their own (fabricated) predictions.
func fabricatedRun(m *manifestDoc) []groupResult {
	ls := fakeRun(m)
	witnessesMet(ls)
	for i := range ls {
		x := &ls[i]
		s := manScenarioOf(m, x.Scenario)
		switch {
		case slices.Contains(s.Criteria, "T2"):
			for j := range min(3, len(x.Pairs)) {
				seed := int64(x.Pairs[j].Seed)
				x.Pairs[j].Harmed = true
				x.Deteriorated["urgent"] = append(x.Deteriorated["urgent"], seed)
				x.Failing = append(x.Failing, seed)
			}
		case s.Replicates > 0 && s.Candidate.Inject != nil:
			x.UrgentSeeds[0].Baseline, x.UrgentSeeds[0].Candidate, x.UrgentSeeds[0].Difference = 0, 30, 1
			x.GateB["urgent"], x.Rejected, x.Rules = Fail, true, []string{"gateB:urgent"}
		}
	}
	return ls
}

// writeLines writes lines as the execution does: one JSON object a line.
func writeLines(t *testing.T, path string, ls []groupResult) {
	t.Helper()
	var b strings.Builder
	for _, g := range ls {
		raw, err := json.Marshal(g)
		if err != nil {
			t.Fatal(err)
		}
		b.Write(append(raw, '\n'))
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

// M1: a fabricated file of 3,904 lines, every one claiming valid
// measurements and every mandatory criterion PASSing on it, is never
// certified: not by the reader, whatever its settings, and not by a
// trusted execution on any family but the reserved one. Only a trusted
// execution on the reserved family at the manifest's seeds can certify,
// and only the execution sets trust.
func TestFabricatedResultsCannotCertify(t *testing.T) {
	m := loadManifest(t)
	tol, err := manifestTolerances(m)
	if err != nil {
		t.Fatal(err)
	}
	full := func(s *manScenario) int { return s.Seeds }
	ls := fabricatedRun(m)
	if len(ls) != 3904 {
		t.Fatalf("%d lines, want the manifest's 3,904", len(ls))
	}
	r := reservedCriteria(m, ls, tol, full)
	for _, c := range mandatoryCriteria {
		if r[c].Status != Pass {
			t.Fatalf("the fabrication should pass every criterion: %s %s %v", c, r[c].Status, r[c].Detail)
		}
	}
	// The reader, on the file, under every setting it accepts.
	path := filepath.Join(t.TempDir(), "fabricated.jsonl")
	writeLines(t, path, ls)
	for _, env := range []map[string]string{{}, {"DT_MANIFEST_SEEDS": "100000"}, {"DT_MANIFEST_FAMILY": "20261201"}} {
		for _, k := range []string{"DT_MANIFEST_SEEDS", "DT_MANIFEST_SCENARIOS", "DT_MANIFEST_FAMILY"} {
			t.Setenv(k, env[k])
		}
		r, bad, err := analyzeFile(m, path)
		if err != nil || len(bad) > 0 {
			t.Fatalf("%v: %v %v", env, err, bad)
		}
		if r["invocation"].Status != "non-certifying analysis" || r["certification"].Status != "not certified" {
			t.Errorf("%v: the reader gave %s, %s", env, r["invocation"].Status, r["certification"].Status)
		}
		out := filepath.Join(t.TempDir(), "criteria.json")
		if err := writeCriteria(out, r); err != nil {
			t.Fatal(err)
		}
		raw, _ := os.ReadFile(out)
		if strings.Contains(string(raw), "certified GO") || strings.Contains(string(raw), `"confirmatory"`) {
			t.Errorf("%v: the reader's output claims certification", env)
		}
	}
	// The certification's decision table.
	for _, x := range []struct {
		ctx  certContext
		seed func(*manScenario) int
		inv  string
		cert string
	}{
		{certContext{}, full, "non-certifying analysis", "not certified"},
		{certContext{Family: reservedFamily}, full, "non-certifying analysis", "not certified"},
		{certContext{Trusted: true, Family: devManifestFamily}, full, "development", "not certified"},
		{certContext{Trusted: true, Family: reservedFamily}, func(s *manScenario) int { return min(s.Seeds, 40) }, "development", "not certified"},
		{certContext{Trusted: true, Family: reservedFamily}, full, "confirmatory", "certified GO: confirmation succeeded"},
	} {
		r := reservedCriteria(m, ls, tol, x.seed)
		certify(r, x.ctx)
		if r["invocation"].Status != x.inv || r["certification"].Status != x.cert {
			t.Errorf("%+v: %s, %s; want %s, %s", x.ctx, r["invocation"].Status, r["certification"].Status, x.inv, x.cert)
		}
	}
	// A confirmatory execution certifies NO GO on any FAIL or
	// INCONCLUSIVE (protocol §6).
	for c, want := range map[string]string{"T1": "certified NO GO: confirmation failed", "T8": "certified NO GO: confirmation not achieved"} {
		r := reservedCriteria(m, ls, tol, full)
		r[c] = criterion{map[string]string{"T1": Fail, "T8": "inconclusive"}[c], nil}
		certify(r, certContext{Trusted: true, Family: reservedFamily})
		if r["certification"].Status != want {
			t.Errorf("%s: %s", c, r["certification"].Status)
		}
	}
}

// M1: the execution's runtime inputs are verified against this build's
// digests, and the manifest's pin of the candidate against the
// candidate's.
func TestCertifiedInputsPinned(t *testing.T) {
	got, err := verifyInputs()
	if err != nil {
		t.Fatal(err)
	}
	roles := []string{"candidate", "manifest", "protocol", "policyInForce", "simProfile"}
	if len(got) != len(roles) {
		t.Fatalf("%v", got)
	}
	for _, r := range roles {
		if len(got[r]) != 64 {
			t.Errorf("%s not verified", r)
		}
	}
	// A changed input is refused, and so is a manifest pinning another
	// candidate.
	dir := t.TempDir()
	copyTo := func(src, name string, edit func(string) string) string {
		raw, err := os.ReadFile(src)
		if err != nil {
			t.Fatal(err)
		}
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(edit(string(raw))), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	same := func(s string) string { return s }
	list := slices.Clone(certifiedInputs)
	for i := range list {
		list[i].Path = copyTo(list[i].Path, list[i].Role, same)
	}
	if _, err := verifyInputList(list, list[1].Path); err != nil {
		t.Fatalf("unchanged copies: %v", err)
	}
	for i := range list {
		changed := slices.Clone(list)
		changed[i].Path = copyTo(list[i].Path, list[i].Role+".changed", func(s string) string { return s + " " })
		if _, err := verifyInputList(changed, list[1].Path); err == nil {
			t.Errorf("a changed %s accepted", list[i].Role)
		}
	}
	zero := copyTo(manifestPath, "manifest.zero", func(s string) string {
		return strings.Replace(s, certifiedInputs[0].SHA256, strings.Repeat("0", 64), 1)
	})
	unpinned := slices.Clone(list)
	unpinned[1].Path, unpinned[1].SHA256 = zero, sha256File(t, zero)
	if _, err := verifyInputList(unpinned, zero); err == nil {
		t.Error("a manifest pinning another candidate accepted")
	}
	if src, err := sourceDigest(); err != nil || len(src) != 64 {
		t.Errorf("source digest %q %v", src, err)
	}
}

// M2: every measurement a criterion reads must be present, exact, finite
// and in range; the auditor's summary-shape witnesses no longer pass.
func TestMeasurementCompleteness(t *testing.T) {
	m := loadManifest(t)
	tol, err := manifestTolerances(m)
	if err != nil {
		t.Fatal(err)
	}
	full := func(s *manScenario) int { return s.Seeds }
	judge := func(ls []groupResult) map[string]criterion { return reservedCriteria(m, ls, tol, full) }
	each := func(ls []groupResult, id string, f func(*groupResult)) []groupResult {
		for i := range ls {
			if ls[i].Scenario == id || id == "" {
				f(&ls[i])
			}
		}
		return ls
	}
	t2g := func() []string {
		var ids []string
		for _, s := range m.Scenarios {
			if slices.Contains(s.Criteria, "T2G") {
				ids = append(ids, s.ID)
			}
		}
		return ids
	}()
	// T2G: every deterioration group without its materiality inputs.
	ls := fabricatedRun(m)
	for _, id := range t2g {
		each(ls, id, func(g *groupResult) { g.Severe, g.WorstSoonPast, g.Increase = nil, nil, nil })
	}
	if r := judge(ls); r["T2G"].Status == Pass {
		t.Error("T2G passes without its materiality inputs")
	}
	// T2G on well-formed lines with no acquired soon or routine study:
	// materiality undetermined, never non-material.
	ls = fabricatedRun(m)
	for _, id := range t2g {
		each(ls, id, func(g *groupResult) {
			g.Severe = map[string][2]int64{"urgent": {0, 400}, "soon": {0, 0}, "routine": {0, 0}}
		})
	}
	if r := judge(ls); r["T2G"].Status != "inconclusive" {
		t.Errorf("T2G %s with no acquired soon or routine study", r["T2G"].Status)
	}
	// T5a: every near-margin replicate without its seed differences.
	ls = fabricatedRun(m)
	for _, id := range []string{"NM1", "NM2", "NM2.1", "NM3"} {
		each(ls, id, func(g *groupResult) { g.UrgentSeeds = nil })
	}
	if r := judge(ls); r["T5a"].Status == Pass || r["T8"].Status == Pass {
		t.Errorf("T5a %s, T8 %s without seed differences", r["T5a"].Status, r["T8"].Status)
	}
	if p := predictAcceptance(groupResult{}, &manInject{Kind: "bernoulli", Effect: 0.02}, 100); !math.IsNaN(p) {
		t.Errorf("a prediction from no seed: %v", p)
	}
	// T8: impossible matched counts.
	ls = each(fabricatedRun(m), "", func(g *groupResult) {
		for q := range g.Matched {
			g.Matched[q] = 999_999
		}
	})
	if r := judge(ls); r["T8"].Status == Pass {
		t.Error("T8 passes with 999,999 matched seeds a group")
	}
	// T8, exactly: an identity witness's group matched on more pairs
	// than its construction allows (within the pairs' count, so only the
	// equality catches it). A removed origin leaves every priority
	// unmatched on its seed; a changed urgent request, urgent alone.
	for id, q := range map[string]string{"W-ID-REMOVED": "soon", "W-ID-CHANGED": "urgent"} {
		ls = fabricatedRun(m)
		each(ls, id, func(g *groupResult) {
			if g.Group == 0 {
				g.Matched[q] = len(g.Pairs)
			}
		})
		if r := judge(ls); r["T8"].Status == Pass {
			t.Errorf("%s: T8 passes with %s matched on every pair", id, q)
		}
	}
	// And the changed request's own counts, soon and routine matched on
	// every pair as constructed, pass.
	if r := judge(fabricatedRun(m)); r["T8"].Status != Pass {
		t.Errorf("T8 %s on the constructed counts: %v", r["T8"].Status, r["T8"].Detail)
	}
	// T4: the witness's violations named on a nonexistent seed, forty
	// times.
	ls = each(fabricatedRun(m), "W-HORIZON", func(g *groupResult) {
		g.Violations = map[string][]int64{"pastDue soon": slices.Repeat([]int64{-999}, len(g.Pairs))}
	})
	if r := judge(ls); r["T4"].Status == Pass || r["T8"].Status == Pass {
		t.Errorf("T4 %s, T8 %s on a nonexistent seed", r["T4"].Status, r["T8"].Status)
	}
	// T4 alone, past the coverage check: the right count on the wrong
	// seeds is not the witness.
	g := fabricatedRun(m)[0]
	for _, id := range []string{"W-URG-LATE", "W-HORIZON", "W-PENDING"} {
		x := g
		x.Rejected = true
		x.Rules = []string{map[string]string{"W-PENDING": "pending"}[id]}
		if x.Rules[0] == "" {
			x.Rules[0] = "safety"
		}
		first := int64(x.Pairs[0].Seed)
		twice := slices.Repeat([]int64{first}, len(x.Pairs))
		x.Violations = map[string][]int64{"pastDue urgent": twice, "pastDue soon": twice}
		x.Pending = twice
		if witnessMet(id, x) {
			t.Errorf("%s met by one seed named %d times", id, len(x.Pairs))
		}
	}
	// lineProblems, field by field, on a valid line of each kind.
	ls = fabricatedRun(m)
	var whole, rep *groupResult
	for i := range ls {
		s := manScenarioOf(m, ls[i].Scenario)
		switch {
		case whole == nil && s.Replicates == 0 && ls[i].Scenario == "S-A85":
			whole = &ls[i]
		case rep == nil && s.Replicates > 0 && s.Candidate.Inject != nil:
			rep = &ls[i]
		}
	}
	for _, x := range []*groupResult{whole, rep} {
		if bad := lineProblems(manScenarioOf(m, x.Scenario), x); len(bad) > 0 {
			t.Fatalf("%s: a valid line: %v", x.Scenario, bad)
		}
	}
	clone := func(g *groupResult) groupResult {
		raw, _ := json.Marshal(g)
		var c groupResult
		json.Unmarshal(raw, &c)
		return c
	}
	for name, f := range map[string]func(*groupResult){
		"Gate B status missing":     func(g *groupResult) { delete(g.GateB, "soon") },
		"Gate B status unknown":     func(g *groupResult) { g.GateB["soon"] = "fine" },
		"increase missing":          func(g *groupResult) { delete(g.Increase, "urgent") },
		"increase not finite":       func(g *groupResult) { g.Increase["urgent"] = math.Inf(1) },
		"matched missing":           func(g *groupResult) { delete(g.Matched, "routine") },
		"matched above the pairs":   func(g *groupResult) { g.Matched["routine"] = len(g.Pairs) + 1 },
		"matched negative":          func(g *groupResult) { g.Matched["routine"] = -1 },
		"deteriorated missing":      func(g *groupResult) { delete(g.Deteriorated, "urgent") },
		"deteriorated foreign seed": func(g *groupResult) { g.Deteriorated["urgent"] = []int64{-1} },
		"deteriorated twice":        func(g *groupResult) { s := int64(g.Pairs[0].Seed); g.Deteriorated["urgent"] = []int64{s, s} },
		"provenance foreign seed":   func(g *groupResult) { g.Provenance = []int64{99_999} },
		"unmatched foreign seed":    func(g *groupResult) { g.UnmatchedUrgent = 99_999 },
		"rejected without rules":    func(g *groupResult) { g.Rejected, g.Rules = true, nil },
		"rules without rejection":   func(g *groupResult) { g.Rejected, g.Rules = false, []string{"safety"} },
		"rule unknown":              func(g *groupResult) { g.Rejected, g.Rules = true, []string{"whim"} },
		"rule repeated":             func(g *groupResult) { g.Rejected, g.Rules = true, []string{"safety", "safety"} },
		"pair counts negative":      func(g *groupResult) { g.Pairs[0].Injected = -1 },
		"served above eligible":     func(g *groupResult) { g.Pairs[0].Served, g.Pairs[0].Eligible = 5, 4 },
		"urgent seed foreign":       func(g *groupResult) { g.UrgentSeeds = append(g.UrgentSeeds, SeedDifference{Seed: -5, Orders: 1}) },
		"urgent seed without orders": func(g *groupResult) {
			g.UrgentSeeds = append(g.UrgentSeeds, SeedDifference{Seed: int64(g.Pairs[1].Seed)})
		},
		"urgent seeds above matched": func(g *groupResult) { g.Matched["urgent"] = 0 },
		"urgent difference not finite": func(g *groupResult) {
			g.UrgentSeeds = append(g.UrgentSeeds, SeedDifference{Seed: int64(g.Pairs[1].Seed), Orders: 3, Difference: math.NaN()})
		},
	} {
		for _, base := range []*groupResult{whole, rep} {
			x := clone(base)
			if name == "urgent seeds above matched" && len(x.UrgentSeeds) == 0 {
				x.UrgentSeeds = []SeedDifference{{Seed: int64(x.Pairs[0].Seed), Orders: 3}}
			}
			f(&x)
			if len(lineProblems(manScenarioOf(m, x.Scenario), &x)) == 0 {
				t.Errorf("%s (%s): accepted", name, x.Scenario)
			}
		}
	}
	for name, f := range map[string]func(*groupResult){
		"claims missing":            func(g *groupResult) { g.Claims = nil },
		"severe missing":            func(g *groupResult) { delete(g.Severe, "soon") },
		"severe above acquired":     func(g *groupResult) { g.Severe["soon"] = [2]int64{5, 4} },
		"worst delays short":        func(g *groupResult) { g.WorstSoonPast = g.WorstSoonPast[1:] },
		"worst delay negative":      func(g *groupResult) { g.WorstSoonPast[0] = -1 },
		"failing foreign seed":      func(g *groupResult) { g.Failing = []int64{-3} },
		"failing twice":             func(g *groupResult) { s := int64(g.Pairs[0].Seed); g.Failing = []int64{s, s} },
		"insufficient foreign seed": func(g *groupResult) { g.Insufficient = []int64{-3} },
		"pending foreign seed":      func(g *groupResult) { g.Pending = []int64{-3} },
		"violation foreign seed":    func(g *groupResult) { g.Violations = map[string][]int64{"pastDue soon": {-999}} },
		"gate failure foreign seed": func(g *groupResult) { g.FailingByGate = map[string][]int64{"backlog_stability": {-1}} },
		"raw insufficient foreign":  func(g *groupResult) { g.RawInsuff = map[string][]int64{GateSevereWeek + " soon": {-1}} },
		"fraction out of range":     func(g *groupResult) { g.Fractions = map[string][2]float64{"urgent": {1.5, 0}} },
		"fraction not finite":       func(g *groupResult) { g.Fractions = map[string][2]float64{"urgent": {math.NaN(), 0}} },
	} {
		x := clone(whole)
		f(&x)
		if len(lineProblems(manScenarioOf(m, x.Scenario), &x)) == 0 {
			t.Errorf("%s: accepted", name)
		}
	}
	// A seed pending in two priorities is named twice: its form.
	x := clone(whole)
	s := int64(x.Pairs[0].Seed)
	x.Pending = []int64{s, s}
	if bad := lineProblems(manScenarioOf(m, x.Scenario), &x); len(bad) > 0 {
		t.Errorf("a seed pending in two priorities: %v", bad)
	}
	// The near-margin replicate: without any seed difference, or a served
	// count, its prediction has nothing to draw.
	for name, f := range map[string]func(*groupResult){
		"no seed difference": func(g *groupResult) { g.UrgentSeeds = nil },
		"no served order":    func(g *groupResult) { g.Pairs[0].Served = 0 },
	} {
		x := clone(rep)
		f(&x)
		if len(lineProblems(manScenarioOf(m, x.Scenario), &x)) == 0 {
			t.Errorf("replicate %s: accepted", name)
		}
	}
}

// M4: malformed settings and inputs fail explicitly; the adjudication is
// never blank or unparseable, even on an empty or partial input; an
// incomplete input is reported, never a success.
func TestReaderErrors(t *testing.T) {
	m := loadManifest(t)
	dir := t.TempDir()
	for _, k := range []string{"DT_MANIFEST_SEEDS", "DT_MANIFEST_SCENARIOS", "DT_MANIFEST_FAMILY"} {
		t.Setenv(k, "")
	}
	full := filepath.Join(dir, "full.jsonl")
	writeLines(t, full, fabricatedRun(m))
	for k, vs := range map[string][]string{
		"DT_MANIFEST_SEEDS":     {"nonsense", "-1", "0", "2.5", " 40"},
		"DT_MANIFEST_SCENARIOS": {"nowhere", "S-A85,S-A85", "S-A85,"},
		"DT_MANIFEST_FAMILY":    {"x", "-3"},
	} {
		for _, v := range vs {
			t.Setenv(k, v)
			if _, _, err := analyzeFile(m, full); err == nil {
				t.Errorf("%s=%q accepted", k, v)
			}
			if _, _, err := manInvocation(m); err == nil {
				t.Errorf("%s=%q accepted by the execution", k, v)
			}
			t.Setenv(k, "")
		}
	}
	// Malformed input: never judged.
	raw, _ := os.ReadFile(full)
	cut := filepath.Join(dir, "cut.jsonl")
	os.WriteFile(cut, raw[:len(raw)/2], 0o644)
	extra := filepath.Join(dir, "extra.jsonl")
	os.WriteFile(extra, []byte(`{"scenario":"S-A85","group":0,"claimedVerdict":"GO"}`+"\n"), 0o644)
	for _, p := range []string{cut, extra, filepath.Join(dir, "absent.jsonl")} {
		if _, _, err := analyzeFile(m, p); err == nil {
			t.Errorf("%s judged", filepath.Base(p))
		}
	}
	// Empty, and a near-margin scenario missing: a parseable adjudication,
	// nothing passing that needs the missing lines, and the input
	// reported incomplete.
	empty := filepath.Join(dir, "empty.jsonl")
	os.WriteFile(empty, nil, 0o644)
	var noNM []groupResult
	for _, g := range fabricatedRun(m) {
		if g.Scenario != "NM2.1" {
			noNM = append(noNM, g)
		}
	}
	missing := filepath.Join(dir, "missing.jsonl")
	writeLines(t, missing, noNM)
	for name, x := range map[string]struct {
		path    string
		notPass []string
	}{"empty": {empty, mandatoryCriteria}, "NM2.1 missing": {missing, []string{"T5a", "T5b", "T8"}}} {
		r, bad, err := analyzeFile(m, x.path)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(bad) == 0 {
			t.Errorf("%s: reported complete", name)
		}
		for _, c := range x.notPass {
			if r[c].Status == Pass {
				t.Errorf("%s: %s PASS", name, c)
			}
		}
		out := filepath.Join(dir, name+".criteria.json")
		if err := writeCriteria(out, r); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		var back map[string]criterion
		raw, _ := os.ReadFile(out)
		if err := json.Unmarshal(raw, &back); err != nil || len(back) < len(mandatoryCriteria)+2 {
			t.Errorf("%s: the adjudication does not parse back: %v (%d bytes)", name, err, len(raw))
		}
	}
	// An unserializable adjudication is an error, and no file.
	nan := filepath.Join(dir, "nan.json")
	if err := writeCriteria(nan, map[string]criterion{"T1": {Pass, map[string]any{"x": math.NaN()}}}); err == nil {
		t.Error("NaN serialized")
	}
	if _, err := os.Stat(nan); !errors.Is(err, os.ErrNotExist) {
		t.Error("a file written for an unserializable adjudication")
	}
}

// M1, end to end: the execution on a permitted family runs, judges and
// certifies in process. Here one scenario, one group, on the debug
// family: every line's measurements complete, the invocation a
// development one and nothing certified.
func TestCertifiedExecutionSmall(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	m := loadManifest(t)
	inputs, err := verifyInputs()
	if err != nil {
		t.Fatal(err)
	}
	seeds := func(s *manScenario) int {
		if s.ID == "W-HORIZON" {
			return 40
		}
		return 0
	}
	lines, r := runCertified(t, m, 1, seeds, certContext{Trusted: true, Family: 1, Inputs: inputs})
	if len(lines) != 1 {
		t.Fatalf("%d lines", len(lines))
	}
	s := manScenarioOf(m, "W-HORIZON")
	if bad := lineProblems(s, &lines[0]); len(bad) > 0 || !witnessMet(s.ID, lines[0]) {
		t.Fatalf("%v; witness met %v", bad, witnessMet(s.ID, lines[0]))
	}
	if r["invocation"].Status != "development" || r["certification"].Status != "not certified" {
		t.Errorf("%s, %s", r["invocation"].Status, r["certification"].Status)
	}
	if r["T4"].Status == Pass {
		t.Error("T4 PASS with thirteen witness scenarios not run")
	}
	// The reserved family is refused before anything runs.
	if err := familyAllowed(reservedFamily); err == nil {
		t.Fatal("the reserved family is allowed in this build")
	}
}

func sha256File(t *testing.T, p string) string {
	t.Helper()
	d, err := fileDigest(p)
	if err != nil {
		t.Fatal(err)
	}
	return d
}
