//go:build calibration

package sim_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/scenario"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/version"
)

// Reproducible calibration experiments (step 18.1). Each manifest in
// docs/calibration/experiments names a scenario, an optional overlay
// merged onto it (the scenario system's own merge), an optional action
// strategy applied at the takeover, the seeds and the horizon. The runner
// writes docs/calibration/results/<name>.json — provenance (engine,
// scenario fingerprints, overlay hash), per-seed windows and aggregates —
// and regenerates docs/calibration/results/summary.md from every result.
//
//	go test -tags calibration ./internal/sim -run CalibrationExperiments -v -timeout 120m
//	DT_EXPERIMENTS=three-techs,strategy-b   (default: all manifests)
//	DT_EXP_DIR, DT_RES_DIR   another manifest set and its results, relative
//	                         to this package (the V0.2 feasibility sweep:
//	                         ../../docs/calibration/feasibility/{experiments,results})
//
// Manifests run in parallel (go test -parallel), seeds within one serially;
// every result depends only on its manifest.

var expDir, resDir = envOr("DT_EXP_DIR", "../../docs/calibration/experiments"), envOr("DT_RES_DIR", "../../docs/calibration/results")

func envOr(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}

type manifest struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Scenario    string          `json:"scenario"`
	Overlay     json.RawMessage `json:"overlay,omitempty"`
	Strategy    string          `json:"strategy,omitempty"` // a letter of calib_actions_test.go's strategies
	Seeds       string          `json:"seeds"`              // "1-9"
	Days        int             `json:"days"`               // 30 and this horizon are measured
}

type agg struct {
	Mean float64 `json:"mean"`
	Min  float64 `json:"min"`
	Max  float64 `json:"max"`
}

type expResult struct {
	Experiment          manifest                  `json:"experiment"`
	Engine              string                    `json:"engine"`
	BaseFingerprint     string                    `json:"baseScenarioFingerprint"`
	ScenarioFingerprint string                    `json:"scenarioFingerprint"` // after the overlay
	OverlaySHA256       string                    `json:"overlaySHA256,omitempty"`
	Seeds               []int64                   `json:"seeds"`
	Aggregate           map[string]map[string]agg `json:"aggregate"` // window → metric
	PerSeed             []seedRun                 `json:"perSeed"`
}

func TestCalibrationExperiments(t *testing.T) {
	paths, _ := filepath.Glob(filepath.Join(expDir, "*.json"))
	want := map[string]bool{}
	for _, n := range strings.Split(os.Getenv("DT_EXPERIMENTS"), ",") {
		if n != "" {
			want[n] = true
		}
	}
	t.Run("run", func(t *testing.T) {
		for _, p := range paths {
			raw, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			var m manifest
			dec := json.NewDecoder(strings.NewReader(string(raw)))
			dec.DisallowUnknownFields()
			if err := dec.Decode(&m); err != nil {
				t.Fatalf("%s: %v", p, err)
			}
			if len(want) > 0 && !want[m.Name] {
				continue
			}
			t.Run(m.Name, func(t *testing.T) {
				t.Parallel()
				res := runExperiment(t, m)
				out, _ := json.MarshalIndent(res, "", " ")
				if err := os.WriteFile(filepath.Join(resDir, m.Name+".json"), append(out, '\n'), 0o644); err != nil {
					t.Fatal(err)
				}
				t.Logf("%s: %s", m.Name, oneLine(res))
			})
		}
	})
	writeSummary(t)
}

func runExperiment(t *testing.T, m manifest) expResult {
	f, err := scenario.ResolveFile(m.Scenario)
	if err != nil {
		t.Fatal(err)
	}
	res := expResult{Experiment: m, Engine: version.Number, Seeds: parseSeeds(m.Seeds)}
	if res.BaseFingerprint, err = scenario.Fingerprint(f); err != nil {
		t.Fatal(err)
	}
	if len(m.Overlay) > 0 {
		sum := sha256.Sum256(m.Overlay)
		res.OverlaySHA256 = hex.EncodeToString(sum[:])
		raw, err := scenario.Canonical(f)
		if err != nil {
			t.Fatal(err)
		}
		var base, over any
		json.Unmarshal(raw, &base)
		if err := json.Unmarshal(m.Overlay, &over); err != nil {
			t.Fatal(err)
		}
		merged, _ := json.Marshal(scenario.Merge(base, over))
		if f, err = scenario.Decode(merged); err != nil {
			t.Fatalf("%s: overlay: %v", m.Name, err)
		}
	}
	if res.ScenarioFingerprint, err = scenario.Fingerprint(f); err != nil {
		t.Fatal(err)
	}
	cfg, err := scenario.Compile(f)
	if err != nil {
		t.Fatalf("%s: %v", m.Name, err)
	}
	var plan func(*sim.World, int) []api.Action
	if m.Strategy != "" {
		s, ok := strategyByKey(m.Strategy)
		if !ok {
			t.Fatalf("%s: unknown strategy %q", m.Name, m.Strategy)
		}
		plan = s.plan
	}
	for _, seed := range res.Seeds {
		res.PerSeed = append(res.PerSeed, calibrate(t, cfg, seed, m.Days, plan))
	}
	res.Aggregate = map[string]map[string]agg{}
	for key := range res.PerSeed[0].Windows {
		per := map[string][]float64{}
		for _, r := range res.PerSeed {
			for k, v := range keyMetrics(r.Windows[key]) {
				per[k] = append(per[k], v)
			}
		}
		a := map[string]agg{}
		for k, xs := range per {
			s := stats(xs)
			a[k] = agg{Mean: round1(s.mean), Min: round1(s.min), Max: round1(s.max)}
		}
		res.Aggregate[key] = a
	}
	return res
}

func round1(x float64) float64 {
	if x < 0 {
		return -round1(-x)
	}
	return float64(int64(x*10+0.5)) / 10
}

// keyMetrics are the per-seed figures aggregated across seeds. Counts are
// totals over the window; exams also per 30 days.
func keyMetrics(w *window) map[string]float64 {
	m := map[string]float64{}
	exams := float64(sum(w.Exams))
	m["exams"] = exams
	m["examsPer30d"] = exams * 30 / float64(w.Days)
	m["referralOrders"] = float64(sum(w.Orders))
	m["walkIns"] = float64(sum(w.WalkIns))
	m["referralLost"] = float64(sum(w.LostDecl) + sum(w.LostNoSlot))
	m["walkInsLeft"] = float64(w.WalkInLeft)
	m["marginPct"] = float64(w.MarginBP) / 100
	m["profitK"] = float64(w.Profit) / 100_000
	m["netRevenueK"] = float64(w.NetRev) / 100_000
	m["opexK"] = float64(w.Opex) / 100_000
	m["overtimeH"] = float64(w.OTMin) / 60
	m["approvedOvertimeH"] = float64(w.ApprOTMin) / 60
	m["walkInsDeclined"] = float64(w.Extra["walkInsDeclined"])
	m["reports"] = float64(w.Reports)
	m["reportsPer30d"] = float64(w.Reports) * 30 / float64(w.Days)
	m["laborK"] = float64(w.Labor) / 100_000
	m["fixedK"] = float64(w.Fixed) / 100_000
	m["variableK"] = float64(w.Variable) / 100_000
	m["cashEndK"] = float64(w.CashEnd) / 100_000
	m["arChangeK"] = float64(w.ARChange) / 100_000
	m["noShows"] = float64(w.NoShows)
	m["cancelled"] = float64(w.Cancelled)
	// The order cohort: arrived = completed + ended without an exam +
	// outstanding, by origin.
	class := map[string]string{"imaged": "Completed", "reported": "Completed", "lost": "Ended", "cancelled": "Ended",
		"no-show": "Ended", "left": "Ended", "booked": "Outstanding", "recall": "Outstanding", "visit": "Outstanding"}
	for origin, states := range w.Cohort {
		o := "cohortReferral"
		if origin == "walk-in" {
			o = "cohortWalkIn"
		}
		// Every key in every seed, so means are over all seeds.
		for _, c := range []string{"Arrived", "Completed", "Ended", "Outstanding"} {
			m[o+c] += 0
		}
		for st := range class {
			m[o+"_"+st] += 0
		}
		for st, n := range states {
			m[o+"Arrived"] += float64(n)
			m[o+class[st]] += float64(n)
			m[o+"_"+st] += float64(n)
		}
	}
	m["techBusyPct"] = float64(w.TechUtil) / 100
	m["readUtilPct"] = float64(w.ReadUtil) / 100
	m["tatP50Min"] = float64(w.TATP50)
	m["tatP90Min"] = float64(w.TATP90)
	m["overTargetPct"] = pct(w.OverTarget, w.Acquired)
	for k, md := range w.Mods {
		m["tueFri"+md] = w.TueFriDay[k]
		m["delayP50Days"+md] = float64(w.DelayP50[k])
		m["delayP90Days"+md] = float64(w.DelayP90[k])
		m["leadP50Days"+md] = float64(w.LeadP50[k])
		m["leadP90Days"+md] = float64(w.LeadP90[k])
		m["exams"+md] = float64(w.Exams[k])
		m["orders"+md] = float64(w.Orders[k])
		m["lost"+md] = float64(w.LostDecl[k] + w.LostNoSlot[k])
		m["utilStaffedPct"+md] = float64(w.UtilStaff[k]) / 100
		m["utilOpenPct"+md] = float64(w.UtilOpen[k]) / 100
		m["reqLeadP50Days"+md] = float64(w.ReqLeadP50[k])
	}
	for key, ts := range w.TATByPri {
		p := strings.ReplaceAll(key, "/", "")
		m[p+"Acquired"] = float64(ts.Acquired)
		m[p+"Completed"] = float64(ts.Completed)
		m[p+"Unfinished"] = float64(ts.Unfinished)
		m[p+"P50Min"] = float64(ts.P50Min)
		m[p+"P90Min"] = float64(ts.P90Min)
		m[p+"MaxMin"] = float64(ts.MaxMin)
		m[p+"Breaches"] = float64(ts.Breaches)
		m[p+"OvernightBreaches"] = float64(ts.Overnight)
		m[p+"OldestUnfinishedMin"] = float64(ts.OldestMin)
	}
	return m
}

// windowKeys are r's windows, shortest first ("30d" before "90d").
func windowKeys(r expResult) []string {
	var keys []string
	for k := range r.Aggregate {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, func(a, b string) int {
		if len(a) != len(b) {
			return len(a) - len(b)
		}
		return strings.Compare(a, b)
	})
	return keys
}

// TestCalibrationSummary regenerates summary.md from the saved results
// without running anything.
func TestCalibrationSummary(t *testing.T) { writeSummary(t) }

func oneLine(r expResult) string {
	a := r.Aggregate["30d"]
	return fmt.Sprintf("30d exams %.0f, margin %.1f%%, urgent p90 %.0f min", a["exams"].Mean, a["marginPct"].Mean, a["urgentP90Min"].Mean)
}

// writeSummary tabulates every result file: one row per experiment and
// window, mean (min–max) across seeds.
func writeSummary(t *testing.T) {
	paths, _ := filepath.Glob(filepath.Join(resDir, "*.json"))
	var rs []expResult
	for _, p := range paths {
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		var r expResult
		if err := json.Unmarshal(raw, &r); err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		rs = append(rs, r)
	}
	sort.Slice(rs, func(i, j int) bool { return rs[i].Experiment.Name < rs[j].Experiment.Name })
	var b strings.Builder
	b.WriteString("# Calibration experiment results\n\nGenerated by `TestCalibrationExperiments` (build tag `calibration`) from the manifests in\n`docs/calibration/experiments/`; do not edit by hand. Each cell is the mean across seeds,\nwith the range in parentheses. TAT is report turnaround (acquisition → final) of the\nstudies acquired in the window; urgent breaches count completed over target plus\nunfinished already past it.\n\n")
	b.WriteString("| experiment | window | engine | seeds | scenario | exams / 30 d | margin % | profit $k | overtime h | referral lost | walk-ins left | TAT p50 / p90 min (all) | urgent p90 / max min | urgent breaches | routine p50 / p90 min |\n|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|\n")
	cell := func(a agg) string { return fmt.Sprintf("%.1f (%.1f–%.1f)", a.Mean, a.Min, a.Max) }
	cell0 := func(a agg) string { return fmt.Sprintf("%.0f (%.0f–%.0f)", a.Mean, a.Min, a.Max) }
	for _, r := range rs {
		for _, k := range windowKeys(r) {
			a := r.Aggregate[k]
			fmt.Fprintf(&b, "| %s | %s | %s | %s | `%s` | %s | %s | %s | %s | %s | %s | %.0f / %.0f | %.0f / %.0f | %s | %.0f / %.0f |\n",
				r.Experiment.Name, k, r.Engine, r.Experiment.Seeds, r.ScenarioFingerprint[:12], cell0(a["examsPer30d"]), cell(a["marginPct"]),
				cell(a["profitK"]), cell0(a["overtimeH"]), cell0(a["referralLost"]), cell0(a["walkInsLeft"]),
				a["tatP50Min"].Mean, a["tatP90Min"].Mean, a["urgentP90Min"].Mean, a["urgentMaxMin"].Mean, cell0(a["urgentBreaches"]),
				a["routineP50Min"].Mean, a["routineP90Min"].Mean)
		}
	}
	// Turnaround by priority, then urgent by modality: completed and
	// unfinished studies of the window's acquisitions, as of its end.
	b.WriteString("\n## Report turnaround by priority\n\nStudies acquired in the window, as of its end; minutes. p50, p90 and max are over the\ncompleted studies; breaches are completed over target plus unfinished past it, and overnight\nthose of them first claimed on a later day than acquired (they waited for the next reading\nblock: coverage, not worklist order); oldest is the age of the oldest unfinished study. Means\nacross seeds; breaches and max with the range.\n\n")
	tatTable := func(prefixes []string) {
		b.WriteString("| experiment | window | group | acquired | completed | unfinished | p50 | p90 | max | breaches | of them overnight | oldest unfinished |\n|---|---|---|---|---|---|---|---|---|---|---|---|\n")
		for _, r := range rs {
			for _, k := range windowKeys(r) {
				a := r.Aggregate[k]
				for _, p := range prefixes {
					if _, ok := a[p+"Acquired"]; !ok {
						continue
					}
					fmt.Fprintf(&b, "| %s | %s | %s | %.1f | %.1f | %.1f | %.0f | %.0f | %s | %s | %.1f | %.0f |\n", r.Experiment.Name, k, p,
						a[p+"Acquired"].Mean, a[p+"Completed"].Mean, a[p+"Unfinished"].Mean, a[p+"P50Min"].Mean, a[p+"P90Min"].Mean,
						cell0(a[p+"MaxMin"]), cell0(a[p+"Breaches"]), a[p+"OvernightBreaches"].Mean, a[p+"OldestUnfinishedMin"].Mean)
				}
			}
		}
	}
	tatTable([]string{"urgent", "soon", "routine"})
	b.WriteString("\n## Urgent turnaround by modality\n\nSame columns, urgent studies only. Means are over the seeds with at least one urgent study of\nthat modality in the window (MR rarely has one).\n\n")
	tatTable([]string{"urgentXR", "urgentUS", "urgentCT", "urgentMR"})
	if err := os.WriteFile(filepath.Join(resDir, "summary.md"), []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}
