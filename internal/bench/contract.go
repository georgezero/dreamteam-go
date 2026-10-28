package bench

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/georgezero/dreamteam-go/internal/version"
)

// The contract report (dreamteam.benchmark-contract/v1,
// docs/BENCHMARK_SUITE.md §3 and §5): a suite run's runs on the contract's
// six dimensions, from the run records and decisions only. Every value is
// the harness's metric as recorded; an absent value is marked absent, with
// the result's own meaning of absence, never replaced. No score, weighting
// or ranking.

// staffingTypes are the staffing actions; every other action type is an
// operations action (docs/LLM_AGENT.md §7).
var staffingTypes = []string{"hire", "terminate", "set_roster", "set_reading_allocation", "contract_teleradiology"}

// Contract is the contract report.
type Contract struct {
	Schema   string  `json:"schema"`
	Engine   string  `json:"engine"`
	Suite    SuiteID `json:"suite"`
	SuiteRun string  `json:"suiteRun"`
	// Status: "complete" when every run completed, else "partial".
	Status    string          `json:"status"`
	Expected  int             `json:"expectedRuns"`
	Completed int             `json:"completedRuns"`
	Failed    []string        `json:"failedRuns"`
	Seeds     []int64         `json:"seeds"`
	Days      int             `json:"days"`
	Cadence   Cadence         `json:"cadence"`
	Policies  []PolicyRef     `json:"policies"`
	Baseline  string          `json:"baseline"`
	Entries   []ContractEntry `json:"entries"`
	Missing   []string        `json:"missing"`
	Note      string          `json:"note"`
	Runs      []ContractRun   `json:"runs"`
	Pairs     []ContractPair  `json:"pairs"`
}

// SuiteID names a suite and its version.
type SuiteID struct {
	ID      string `json:"id"`
	Version int    `json:"version"`
	Split   string `json:"split"`
}

// ContractEntry is a suite entry as run: its identity and, per seed, the
// starting state its runs share.
type ContractEntry struct {
	ID          string          `json:"id"`
	Version     int             `json:"version"`
	Category    string          `json:"category"`
	Scenario    string          `json:"scenario"`
	Fingerprint string          `json:"fingerprint"`
	T90         string          `json:"t90"`
	Starts      []ContractStart `json:"startingStates"`
}

// ContractStart is one seed's starting state across the policies:
// identical when every run of the entry and seed began at the same time
// with the same state digest.
type ContractStart struct {
	Seed      int64  `json:"seed"`
	Runs      int    `json:"runs"`
	Identical bool   `json:"identical"`
	Time      string `json:"time"`
	State     string `json:"stateDigest"`
}

// ContractRun is one run on the six dimensions.
type ContractRun struct {
	ID          string          `json:"id"`
	Entry       string          `json:"entry"`
	Scenario    string          `json:"scenario"`
	Fingerprint string          `json:"fingerprint"`
	Seed        int64           `json:"seed"`
	Policy      PolicyRef       `json:"policy"`
	Days        int             `json:"days"`
	Start       string          `json:"startingStateDigest"`
	Status      string          `json:"status"`
	Replay      bool            `json:"replayVerified"`
	Actions     ActionCounts    `json:"actions"`
	T90         ContractT90     `json:"t90"`
	Dimensions  []DimensionVals `json:"dimensions"`
}

// ActionCounts are a run's actions: submitted in applied batches, by
// category and type, and in refused batches.
type ActionCounts struct {
	Operations int            `json:"operations"`
	Staffing   int            `json:"staffing"`
	ByType     map[string]int `json:"byType"`
	Refused    int            `json:"refused"`
}

// ContractT90 is T90's applicability (the suite entry's) and the run's
// outcome.
type ContractT90 struct {
	Applicability  string `json:"applicability"`
	Status         string `json:"status"`
	Verdict        string `json:"verdict,omitempty"`
	Classification string `json:"classification,omitempty"`
}

// DimensionVals are one dimension's metrics in a run.
type DimensionVals struct {
	Dimension string        `json:"dimension"`
	Metrics   []ContractVal `json:"metrics"`
}

// ContractVal is one metric: its value as recorded, or absent with the
// result's meaning of absence.
type ContractVal struct {
	Name   string          `json:"name"`
	Value  json.RawMessage `json:"value,omitempty"`
	Absent string          `json:"absent,omitempty"`
}

// ContractPair is a run paired with the baseline's on the same entry and
// seed: policy − baseline for each integer metric, by dimension; null
// when either is absent.
type ContractPair struct {
	Entry      string       `json:"entry"`
	Seed       int64        `json:"seed"`
	Policy     string       `json:"policy"`
	Against    string       `json:"against"`
	Dimensions []PairDeltas `json:"dimensions"`
	Actions    [2]int       `json:"actionsOperationsStaffing"` // the policy's, less the baseline's
}

// PairDeltas are one dimension's differences.
type PairDeltas struct {
	Dimension string  `json:"dimension"`
	Deltas    []Delta `json:"deltas"`
}

const contractNote = "The evaluation contract dreamteam.benchmark-contract/v1 (docs/BENCHMARK_SUITE.md): the harness's own metrics on six dimensions, each reported on its own; no composite score, weighting, leaderboard or ranking. Absent values are marked, not estimated. T90 is reported only where the evaluation policy in force applies. Paired differences are policy minus baseline on the same entry and seed."

// BuildContract builds the contract report of a suite run's results in
// dir from its run records and their decisions.
func BuildContract(sr *SuiteRun, dir string, recs []*RunRecord) (*Contract, error) {
	c := sr.Config
	k := &Contract{Schema: ContractSchema, Engine: version.Number, Suite: SuiteID{sr.Suite.ID, sr.Suite.Version, sr.Suite.Split},
		SuiteRun: sr.Run.ID, Expected: len(c.Scenarios) * len(c.Seeds) * len(c.Policies), Failed: []string{},
		Seeds: c.Seeds, Days: c.Days, Cadence: c.Cadence, Baseline: c.Baseline, Missing: sr.Suite.Missing, Note: contractNote,
		Runs: []ContractRun{}, Pairs: []ContractPair{}}
	for _, p := range c.Policies {
		k.Policies = append(k.Policies, policyRef(c, p))
	}
	entryOf := map[string]SuiteEntry{}
	for _, e := range sr.Entries {
		entryOf[e.Scenario] = e
	}
	absent := map[string]string{}
	for _, m := range Metrics {
		absent[m.Name] = m.Absent
	}
	byKey := map[string]*ContractRun{}
	for _, rec := range recs {
		e := entryOf[rec.Scenario.Ref]
		r := ContractRun{ID: rec.ID, Entry: e.ID, Scenario: rec.Scenario.Ref, Fingerprint: rec.Scenario.Fingerprint, Seed: rec.Seed,
			Policy: rec.Policy, Days: rec.Days, Start: rec.Initial.Digests.State, Status: rec.Status,
			Replay: rec.Replay != nil && rec.Replay.Verified,
			T90:    ContractT90{Applicability: e.T90, Status: rec.T90.Status, Verdict: rec.T90.Verdict, Classification: rec.T90.Classification}}
		if rec.Status == StatusCompleted {
			k.Completed++
		} else {
			k.Failed = append(k.Failed, rec.ID)
		}
		acts, err := countActions(filepath.Join(dir, "runs", rec.ID, FileDecisions))
		if err != nil {
			return nil, err
		}
		r.Actions = acts
		vals := map[string]json.RawMessage{}
		for _, v := range rec.Metrics {
			vals[v.Name] = v.Value
		}
		for _, d := range Dimensions {
			dv := DimensionVals{Dimension: d.Key}
			for _, name := range sr.Suite.Dimensions[d.Key] {
				v, ok := vals[name]
				if !ok || string(v) == "null" {
					a := absent[name]
					if a == "" {
						a = "not recorded"
						if rec.Status != StatusCompleted {
							a = "run did not complete"
						}
					}
					dv.Metrics = append(dv.Metrics, ContractVal{Name: name, Absent: a})
					continue
				}
				dv.Metrics = append(dv.Metrics, ContractVal{Name: name, Value: v})
			}
			r.Dimensions = append(r.Dimensions, dv)
		}
		k.Runs = append(k.Runs, r)
		byKey[fmt.Sprintf("%s|%d|%s", r.Entry, r.Seed, r.Policy.ID)] = &k.Runs[len(k.Runs)-1]
	}
	k.Status = "complete"
	if k.Completed != k.Expected || len(recs) != k.Expected {
		k.Status = "partial"
	}

	for _, e := range sr.Entries {
		ce := ContractEntry{ID: e.ID, Version: e.Version, Category: e.Category, Scenario: e.Scenario, Fingerprint: e.Fingerprint, T90: e.T90}
		for _, seed := range c.Seeds {
			st := ContractStart{Seed: seed, Identical: true}
			var first *RunRecord
			for _, rec := range recs {
				if rec.Scenario.Ref != e.Scenario || rec.Seed != seed {
					continue
				}
				if first == nil {
					first, st.Time, st.State = rec, rec.Initial.Time, rec.Initial.Digests.State
				} else if rec.Initial.Time != first.Initial.Time || rec.Initial.Digests != first.Initial.Digests {
					st.Identical = false
				}
				st.Runs++
			}
			if first == nil {
				st.Identical = false
			}
			ce.Starts = append(ce.Starts, st)
		}
		k.Entries = append(k.Entries, ce)
		for _, seed := range c.Seeds {
			base := byKey[fmt.Sprintf("%s|%d|%s", e.ID, seed, c.Baseline)]
			for _, p := range c.Policies {
				r := byKey[fmt.Sprintf("%s|%d|%s", e.ID, seed, p)]
				if p == c.Baseline || r == nil || base == nil || r.Status != StatusCompleted || base.Status != StatusCompleted {
					continue
				}
				pr := ContractPair{Entry: e.ID, Seed: seed, Policy: p, Against: c.Baseline,
					Actions: [2]int{r.Actions.Operations - base.Actions.Operations, r.Actions.Staffing - base.Actions.Staffing}}
				for i, d := range r.Dimensions {
					pd := PairDeltas{Dimension: d.Dimension}
					for j, v := range d.Metrics {
						pd.Deltas = append(pd.Deltas, Delta{Name: v.Name, Value: intDelta(v, base.Dimensions[i].Metrics[j])})
					}
					pr.Dimensions = append(pr.Dimensions, pd)
				}
				k.Pairs = append(k.Pairs, pr)
			}
		}
	}
	return k, nil
}

func policyRef(c *Config, id string) PolicyRef {
	if spec, ok := lookupPolicy(id); ok {
		return PolicyRef{spec.ID, spec.Version}
	}
	for _, x := range c.External {
		if x.ID == id {
			return PolicyRef{x.ID, x.Version}
		}
	}
	return PolicyRef{ID: id}
}

// intDelta is a − b for two integer values, or nil (a ratio, or absent).
func intDelta(a, b ContractVal) *int64 {
	var x, y int64
	if a.Value == nil || b.Value == nil || json.Unmarshal(a.Value, &x) != nil || json.Unmarshal(b.Value, &y) != nil {
		return nil
	}
	d := x - y
	return &d
}

// countActions counts a run's actions from its decisions.
func countActions(path string) (ActionCounts, error) {
	a := ActionCounts{ByType: map[string]int{}}
	var ds []Decision
	if err := readJSON(path, &ds); err != nil {
		if os.IsNotExist(err) {
			return a, nil
		}
		return a, err
	}
	for _, d := range ds {
		for _, b := range d.Batches {
			for _, raw := range b.Actions {
				var x struct{ Type string }
				json.Unmarshal(raw, &x)
				if !b.Applied {
					a.Refused++
					continue
				}
				a.ByType[x.Type]++
				if slices.Contains(staffingTypes, x.Type) {
					a.Staffing++
				} else {
					a.Operations++
				}
			}
		}
	}
	return a, nil
}

// Text renders the contract report.
func (k *Contract) Text() string {
	var b strings.Builder
	fmt.Fprintf(&b, "DreamTeam benchmark contract report · %s · suite %s version %d (%s) · suite run %s · engine %s\n",
		k.Schema, k.Suite.ID, k.Suite.Version, k.Suite.Split, k.SuiteRun, k.Engine)
	if k.Status != "complete" {
		fmt.Fprintf(&b, "\n*** PARTIAL SUITE: %d of %d runs completed. Not a complete suite result. ***\nRuns that did not complete: %s\n",
			k.Completed, k.Expected, strings.Join(k.Failed, ", "))
	} else {
		fmt.Fprintf(&b, "complete: %d of %d runs\n", k.Completed, k.Expected)
	}
	var pols []string
	for _, p := range k.Policies {
		pols = append(pols, p.ID+"@"+p.Version)
	}
	fmt.Fprintf(&b, "seeds %s · %d days · decisions every %d day(s) at %s · policies %s · paired against %s\n\n",
		joinInts(k.Seeds), k.Days, k.Cadence.EveryDays, k.Cadence.At, strings.Join(pols, ", "), k.Baseline)
	b.WriteString(wrap(k.Note, 76) + "\n")

	b.WriteString("\n1. Scenarios and starting states\n\n")
	rows := [][]string{{"entry", "version", "scenario", "fingerprint", "T90", "seed", "runs", "start", "identical"}}
	for _, e := range k.Entries {
		for _, s := range e.Starts {
			rows = append(rows, []string{e.ID, fmt.Sprint(e.Version), e.Scenario, e.Fingerprint[:12] + "…", e.T90,
				fmt.Sprint(s.Seed), fmt.Sprint(s.Runs), abbrev(s.State), yesNo(s.Identical)})
		}
	}
	table(&b, rows)

	b.WriteString("\n2. Runs: status, replay, actions (operations/staffing/refused) and T90\n\n")
	rows = [][]string{{"entry", "seed", "policy", "status", "replay", "actions o/s/r", "T90"}}
	for _, r := range k.Runs {
		t := r.T90.Status
		if r.T90.Verdict != "" {
			t = r.T90.Verdict
		}
		rows = append(rows, []string{r.Entry, fmt.Sprint(r.Seed), r.Policy.ID + "@" + r.Policy.Version, r.Status, okText(r.Replay),
			fmt.Sprintf("%d/%d/%d", r.Actions.Operations, r.Actions.Staffing, r.Actions.Refused), t})
	}
	table(&b, rows)

	for di, d := range Dimensions {
		fmt.Fprintf(&b, "\n%d. %s: minimum / median / maximum over completed runs\n\n", 3+di, d.Name)
		names := k.dimNames(d.Key)
		head := []string{"entry", "policy"}
		for _, n := range names {
			head = append(head, label(n))
		}
		if d.Key == "actions" {
			head = append(head, "operations", "staffing")
		}
		rows = [][]string{head}
		for _, e := range k.Entries {
			for _, p := range k.Policies {
				row := []string{e.ID, p.ID}
				var runs []ContractRun
				for _, r := range k.Runs {
					if r.Entry == e.ID && r.Policy.ID == p.ID && r.Status == StatusCompleted {
						runs = append(runs, r)
					}
				}
				for _, n := range names {
					row = append(row, spreadText(n, runs, d.Key))
				}
				if d.Key == "actions" {
					var o, s []int64
					for _, r := range runs {
						o, s = append(o, int64(r.Actions.Operations)), append(s, int64(r.Actions.Staffing))
					}
					row = append(row, spreadInts("", o), spreadInts("", s))
				}
				rows = append(rows, row)
			}
		}
		table(&b, rows)
	}

	fmt.Fprintf(&b, "\n9. Paired with %s (policy − %s, same entry and seed): median [minimum, maximum] over seeds\n", k.Baseline, k.Baseline)
	for _, d := range Dimensions {
		names := k.dimNames(d.Key)
		fmt.Fprintf(&b, "\n%s\n\n", d.Name)
		head := []string{"entry", "policy", "seeds"}
		for _, n := range names {
			head = append(head, label(n))
		}
		rows = [][]string{head}
		for _, e := range k.Entries {
			for _, p := range k.Policies {
				if p.ID == k.Baseline {
					continue
				}
				var ps []ContractPair
				for _, x := range k.Pairs {
					if x.Entry == e.ID && x.Policy == p.ID {
						ps = append(ps, x)
					}
				}
				if len(ps) == 0 {
					continue
				}
				row := []string{e.ID, p.ID, fmt.Sprint(len(ps))}
				for _, n := range names {
					var vs []int64
					missing := 0
					for _, x := range ps {
						for _, pd := range x.Dimensions {
							for _, dl := range pd.Deltas {
								if dl.Name == n {
									if dl.Value == nil {
										missing++
									} else {
										vs = append(vs, *dl.Value)
									}
								}
							}
						}
					}
					row = append(row, deltaText(n, vs, missing))
				}
				rows = append(rows, row)
			}
		}
		table(&b, rows)
	}

	b.WriteString("\n10. Missing\n\n")
	for _, m := range k.Missing {
		b.WriteString(wrap("- "+m, 76) + "\n")
	}
	var marks []string
	for _, r := range k.Runs {
		for _, d := range r.Dimensions {
			for _, v := range d.Metrics {
				if v.Value == nil {
					marks = append(marks, fmt.Sprintf("%s %s: %s", r.ID, v.Name, v.Absent))
				}
			}
		}
	}
	if len(marks) == 0 {
		b.WriteString("- No metric is absent in any run.\n")
	} else {
		counts := map[string]int{}
		for _, m := range marks {
			counts[m[strings.Index(m, " ")+1:]]++
		}
		keys := make([]string, 0, len(counts))
		for x := range counts {
			keys = append(keys, x)
		}
		sort.Strings(keys)
		for _, x := range keys {
			fmt.Fprintf(&b, "- absent in %d run(s): %s\n", counts[x], x)
		}
	}
	return b.String()
}

// labels are the contract report's column names.
var labels = map[string]string{
	"examsCompleted": "exams", "reportsFinal": "reports", "tatP50Sec": "TAT p50", "tatP90Sec": "TAT p90",
	"backlogEnd": "backlog end", "waitToExamP50Sec": "wait p50", "waitToExamP90Sec": "wait p90",
	"netRevenueCents": "net revenue", "operatingExpenseCents": "expenses", "operatingProfitCents": "profit",
	"margin": "margin", "operatingCashFlowCents": "op. cash flow", "ordersReceived": "orders", "ordersLost": "lost",
	"lostRate": "lost rate", "overtimeSec": "overtime", "heldOver": "held over", "skippedBreaks": "skipped breaks",
	"noCoverage": "no coverage", "tatOverTarget": "over target", "tatBreaches": "breaches", "waitsOver30": "waits >30",
	"displacedAppointments": "displaced", "alertsRaisedAlert": "alerts", "alertsActiveAtEnd": "alerts at end",
	"minCashCents": "min cash", "shortfallCents": "shortfall", "daysBelowZero": "days < 0",
	"actionsAttempted": "attempted", "actionsAccepted": "accepted", "actionsRejected": "rejected",
}

func label(n string) string {
	if l, ok := labels[n]; ok {
		return l
	}
	return n
}

func (k *Contract) dimNames(key string) []string {
	for _, r := range k.Runs {
		for _, d := range r.Dimensions {
			if d.Dimension == key {
				names := make([]string, len(d.Metrics))
				for i, v := range d.Metrics {
					names[i] = v.Name
				}
				return names
			}
		}
	}
	return nil
}

func okText(b bool) string {
	if b {
		return "ok"
	}
	return "NOT VERIFIED"
}

// spreadText is a metric's minimum / median / maximum over runs: integers
// in the metric's unit, ratios as a percentage range; "absent" when no
// run has it.
func spreadText(name string, runs []ContractRun, dim string) string {
	var ints []int64
	var pct []float64
	n := 0
	for _, r := range runs {
		for _, d := range r.Dimensions {
			if d.Dimension != dim {
				continue
			}
			for _, v := range d.Metrics {
				if v.Name != name || v.Value == nil {
					continue
				}
				n++
				var x int64
				var q struct{ Num, Den int64 }
				if json.Unmarshal(v.Value, &x) == nil {
					ints = append(ints, x)
				} else if json.Unmarshal(v.Value, &q) == nil && q.Den != 0 {
					pct = append(pct, 100*float64(q.Num)/float64(q.Den))
				}
			}
		}
	}
	switch {
	case n == 0:
		return "absent"
	case len(pct) > 0:
		slices.Sort(pct)
		lo, mid, hi := fmt.Sprintf("%.1f%%", pct[0]), fmt.Sprintf("%.1f%%", pct[(len(pct)-1)/2]), fmt.Sprintf("%.1f%%", pct[len(pct)-1])
		if lo == hi {
			return lo
		}
		return lo + " / " + mid + " / " + hi
	}
	s := spreadInts(name, ints)
	if n < len(runs) {
		s += fmt.Sprintf(" (absent in %d)", len(runs)-n)
	}
	return s
}

func spreadInts(name string, vs []int64) string {
	if len(vs) == 0 {
		return "–"
	}
	slices.Sort(vs)
	u := func(v int64) string {
		if name == "" {
			return group(v)
		}
		return unitText(name, v)
	}
	if vs[0] == vs[len(vs)-1] {
		return u(vs[0])
	}
	return u(vs[0]) + " / " + u(vs[(len(vs)-1)/2]) + " / " + u(vs[len(vs)-1])
}

func deltaText(name string, vs []int64, missing int) string {
	if len(vs) == 0 {
		if missing > 0 {
			return "n/a"
		}
		return "–"
	}
	slices.Sort(vs)
	s := signed(name, vs[(len(vs)-1)/2])
	if vs[0] != vs[len(vs)-1] {
		s += " [" + signed(name, vs[0]) + ", " + signed(name, vs[len(vs)-1]) + "]"
	}
	if missing > 0 {
		s += fmt.Sprintf(" (n/a in %d)", missing)
	}
	return s
}
