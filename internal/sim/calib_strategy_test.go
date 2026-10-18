//go:build calibration

package sim_test

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/eval"
	"github.com/georgezero/dreamteam-go/internal/version"
)

// Slice 5C.1 strategy development (docs/calibration/strategy-development):
// a strategy set on the development seeds 901–910, under the frozen
// policy, run exactly as the canonical holdout runner runs a frozen set
// (holdout_runner_test.go: the same scenario, batches, marks, common
// horizon and evaluation) but recorded by the development recorder. It
// refuses any seed that is not a development seed. Its outputs are written
// once; a rerun must reproduce them byte for byte.
//
//	DT_STRAT_SET=p1-historical go test -tags calibration ./internal/sim -run 'StrategyDevelopment$' -v -timeout 600m
//
//	DT_STRAT_SET=id        the set (sets/<id>.json); results go to results/<id>/
//	DT_STRAT_PAIRS=a:b,..  informational candidate-to-candidate comparisons
//	                       (every teleOnlyOf pair is always compared)
//	DT_STRAT_RECORDS=dir   also write every record (gzip) there, for diagnostics
//	DT_STRAT_PARALLEL=n    runs at once (default 3)

const stratDir = "docs/calibration/strategy-development"

func TestStrategyDevelopment(t *testing.T) {
	root := "../.."
	id := os.Getenv("DT_STRAT_SET")
	if id == "" {
		t.Skip("set DT_STRAT_SET to a development set id (docs/calibration/strategy-development/sets)")
	}
	pol, digest, err := eval.LoadPolicy(filepath.Join(root, eval.PolicyPath))
	if err != nil {
		t.Fatal(err)
	}
	seeds, err := eval.DevelopmentSeeds(pol)
	if err != nil {
		t.Fatal(err)
	}
	setPath := stratDir + "/sets/" + id + ".json"
	p, setSHA, err := eval.Development(root, setPath, pol, digest, seeds)
	if err != nil {
		t.Fatal(err)
	}
	parallel := 3
	fmt.Sscan(envOr("DT_STRAT_PARALLEL", "3"), &parallel)
	recDir := os.Getenv("DT_STRAT_RECORDS")
	run := devRun{Schema: "dreamteam.devrun/v1", Set: setPath, SetSHA256: setSHA, PolicyDigest: digest, Engine: version.Number, Seeds: seeds}
	var traj bytes.Buffer
	traj.WriteString("plan,seed,day,profitCents,cashCents,obligationsCents,backlog,overdue,acquiredCum,finalCum\n")
	records, horizon := devMatrix(t, p, parallel, func(r eval.Record) {
		raw, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		run.Records = append(run.Records, eval.RunRecord{Plan: r.Plan, Seed: r.Seed, SHA256: sha256Hex(raw)})
		for _, s := range r.Samples {
			fmt.Fprintf(&traj, "%s,%d,%d,%d,%d,%d,%d,%d,%d,%d\n", r.Plan, r.Seed, s.Day, s.ProfitCents, s.CashCents, s.ObligationsCents, s.Backlog(), s.Overdue, s.AcquiredCum, s.FinalCum)
		}
		if recDir != "" {
			var gz bytes.Buffer
			zw := gzip.NewWriter(&gz)
			zw.Write(raw)
			zw.Close()
			os.MkdirAll(filepath.Join(recDir, id), 0o755)
			if err := os.WriteFile(filepath.Join(recDir, id, fmt.Sprintf("%s-%d.json.gz", r.Plan, r.Seed)), gz.Bytes(), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	})
	run.HorizonEndSec = horizon
	var pairs [][2]string
	for _, name := range p.Plans() {
		if of := p.Strategies[name].TeleOnlyOf; of != "" {
			pairs = append(pairs, [2]string{name, of})
		}
	}
	for _, x := range strings.Split(os.Getenv("DT_STRAT_PAIRS"), ",") {
		if a, b, ok := strings.Cut(x, ":"); ok {
			pairs = append(pairs, [2]string{a, b})
		}
	}
	rep := eval.Evaluate(p.Policy, digest, eval.Input{Baseline: p.Set.Baseline, Plans: records, Pairs: pairs, Seeds: seeds, PlanSHA: p.StrategySHA})
	report, err := json.MarshalIndent(rep, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	report = append(report, '\n')
	run.ReportSHA256 = sha256Hex(report)
	manifest, _ := json.MarshalIndent(run, "", " ")
	manifest = append(manifest, '\n')
	out := filepath.Join(root, stratDir, "results", id)
	mode, err := writeOnce(out, map[string][]byte{"run.json": manifest, "report.json": report, "trajectory.csv": traj.Bytes()})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range rep.Comparisons {
		t.Logf("%s against %s: %s, %s %v", c.Candidate, c.Against, c.Verdict, c.Classification, c.Reasons)
	}
	t.Logf("development run %s %s in %s (horizon %d)", id, mode, out, horizon)
}
