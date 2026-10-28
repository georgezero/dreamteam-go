package bench

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/georgezero/dreamteam-go/internal/agent"
	"github.com/georgezero/dreamteam-go/internal/version"
)

// The files of a results directory.
const (
	FileConfig     = "config.json"   // the configuration, verbatim
	FileManifest   = "manifest.json" // every deterministic file's SHA-256
	FileReportJSON = "report.json"
	FileReportText = "report.txt"
	FileTiming     = "timing.json" // wall-clock times: not deterministic, not hashed
)

// Schemas of the manifest and the report.
const (
	ManifestSchema = "dreamteam.benchmark-results/v1"
	ReportSchema   = "dreamteam.benchmark-report/v1"
)

// Manifest is a results directory's index: what ran (the configuration,
// engine, protocol, policies and metric definitions) and the SHA-256 of
// every deterministic file. Identical configurations and policy versions
// on the same engine give byte-identical manifests.
type Manifest struct {
	Schema    string        `json:"schema"`
	Benchmark string        `json:"benchmark"`
	Engine    string        `json:"engine"`
	Protocol  string        `json:"protocol"`
	Config    FileHash      `json:"config"`
	Policies  []PolicySpec  `json:"policies"`
	PolicyOps []string      `json:"policyOps"`
	Metrics   []Metric      `json:"metrics"`
	Runs      []ManifestRun `json:"runs"`
	Reports   []FileHash    `json:"reports"`
	// Suite are a suite run's own files (V0.6b): the suite-run
	// configuration and the suite file, verbatim, and the contract report.
	// Absent for an ordinary benchmark.
	Suite            []FileHash `json:"suite,omitempty"`
	Nondeterministic []string   `json:"nondeterministic"`
}

// suiteFiles are a suite run's own files, in manifest order.
var suiteFiles = []string{FileSuiteRun, FileSuite, FileContractJSON, FileContractText}

// FileHash is a file (relative to the results directory) and its SHA-256.
type FileHash struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// ManifestRun is one run's status and files.
type ManifestRun struct {
	ID     string     `json:"id"`
	Status string     `json:"status"`
	Files  []FileHash `json:"files"`
}

// runFiles are a run's files, in manifest order (those that exist).
var runFiles = []string{FileRun, FileDecisions, FileTranscript, FileSession, FileResult, FileT90}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func hashFile(dir, rel string) (FileHash, error) {
	raw, err := os.ReadFile(filepath.Join(dir, rel))
	if err != nil {
		return FileHash{}, err
	}
	sum := sha256.Sum256(raw)
	return FileHash{filepath.ToSlash(rel), hex.EncodeToString(sum[:])}, nil
}

// finish writes the report and the manifest.
func finish(c *Config, dir string, recs []*RunRecord, specs []PolicySpec, sr *SuiteRun) (*Manifest, error) {
	r := BuildReport(c, recs)
	if err := writeJSON(filepath.Join(dir, FileReportJSON), r); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(dir, FileReportText), []byte(r.Text()), 0o644); err != nil {
		return nil, err
	}
	m := &Manifest{Schema: ManifestSchema, Benchmark: c.ID, Engine: version.Number, Protocol: agent.Protocol,
		PolicyOps: PolicyOps, Metrics: Metrics, Nondeterministic: []string{FileTiming}}
	var err error
	if m.Config, err = hashFile(dir, FileConfig); err != nil {
		return nil, err
	}
	m.Policies = specs
	for _, rec := range recs {
		mr := ManifestRun{ID: rec.ID, Status: rec.Status}
		// An external policy's standard error: kept, not hashed (it is
		// the process's own output, outside the protocol).
		if rel := filepath.ToSlash(filepath.Join("runs", rec.ID, FileStderr)); fileExists(filepath.Join(dir, rel)) {
			m.Nondeterministic = append(m.Nondeterministic, rel)
		}
		// Its evidence files (V0.5a), likewise.
		filepath.WalkDir(filepath.Join(dir, "runs", rec.ID, FileEvidence), func(p string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() {
				rel, _ := filepath.Rel(dir, p)
				m.Nondeterministic = append(m.Nondeterministic, filepath.ToSlash(rel))
			}
			return nil
		})
		for _, f := range runFiles {
			rel := filepath.Join("runs", rec.ID, f)
			if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
				continue
			}
			h, err := hashFile(dir, rel)
			if err != nil {
				return nil, err
			}
			mr.Files = append(mr.Files, h)
		}
		m.Runs = append(m.Runs, mr)
	}
	for _, f := range []string{FileReportJSON, FileReportText} {
		h, err := hashFile(dir, f)
		if err != nil {
			return nil, err
		}
		m.Reports = append(m.Reports, h)
	}
	if sr != nil {
		k, err := BuildContract(sr, dir, recs)
		if err != nil {
			return nil, err
		}
		if err := writeJSON(filepath.Join(dir, FileContractJSON), k); err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(dir, FileContractText), []byte(k.Text()), 0o644); err != nil {
			return nil, err
		}
		for _, f := range suiteFiles {
			h, err := hashFile(dir, f)
			if err != nil {
				return nil, err
			}
			m.Suite = append(m.Suite, h)
		}
	}
	return m, writeJSON(filepath.Join(dir, FileManifest), m)
}

// LoadResults reads a results directory: its manifest, configuration and
// run records, in the manifest's order.
func LoadResults(dir string) (*Manifest, *Config, []*RunRecord, error) {
	var m Manifest
	if err := readJSON(filepath.Join(dir, FileManifest), &m); err != nil {
		return nil, nil, nil, err
	}
	if m.Schema != ManifestSchema {
		return nil, nil, nil, fmt.Errorf("%s: schema %q, want %q", FileManifest, m.Schema, ManifestSchema)
	}
	c, _, err := LoadConfig(filepath.Join(dir, FileConfig))
	if err != nil {
		return nil, nil, nil, err
	}
	var recs []*RunRecord
	for _, mr := range m.Runs {
		var rec RunRecord
		if err := readJSON(filepath.Join(dir, "runs", mr.ID, FileRun), &rec); err != nil {
			return nil, nil, nil, err
		}
		recs = append(recs, &rec)
	}
	return &m, c, recs, nil
}

func readJSON(path string, v any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, v); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

// Report is the comparison of a results directory's runs. It reports the
// simulator's metrics as recorded: no composite score, ranking or test.
type Report struct {
	Schema    string      `json:"schema"`
	Benchmark string      `json:"benchmark"`
	Engine    string      `json:"engine"`
	Protocol  string      `json:"protocol"`
	Scenarios []string    `json:"scenarios"`
	Seeds     []int64     `json:"seeds"`
	Days      int         `json:"days"`
	Cadence   Cadence     `json:"cadence"`
	Policies  []PolicyRef `json:"policies"`
	Baseline  string      `json:"baseline"`
	External  []string    `json:"external,omitempty"` // the policies that are external processes
	Note      string      `json:"note"`

	Runs     []RunRow      `json:"runs"`
	Initial  []Initial     `json:"initialConditions"`
	Summary  []PolicyStats `json:"policySummary"`
	Pairs    []Pair        `json:"pairs"`
	Counts   []PairCount   `json:"pairCounts"`
	Failures []Failed      `json:"failures"`
	Status   []StatusCount `json:"statusCounts"`
	Replay   ReplaySummary `json:"replay"`
}

const reportNote = "Development seeds: a harness check, not a performance claim. Metrics are the simulator's own (the replayed session's dreamteam.result/v1) and, where it applies, the T90 evaluation of the policy in force. No composite score, ranking or significance test. Spreads are the minimum, median (the lower middle value) and maximum over completed runs; paired differences are policy minus baseline on the same scenario and seed."

// RunRow is one run's line.
type RunRow struct {
	ID           string    `json:"id"`
	Scenario     string    `json:"scenario"`
	Seed         int64     `json:"seed"`
	Policy       PolicyRef `json:"policy"`
	Status       string    `json:"status"`
	Counts       Counts    `json:"counts"`
	Replay       bool      `json:"replayVerified"`
	ActionsMatch bool      `json:"actionsMatchSession"`
	FinalRun     string    `json:"finalRunDigest,omitempty"`
	T90          T90Info   `json:"t90"`
	Metrics      []Value   `json:"metrics,omitempty"`
}

// Initial is one scenario and seed's starting point across the policies:
// identical when every run began at the same time with the same digests.
type Initial struct {
	Scenario  string `json:"scenario"`
	Seed      int64  `json:"seed"`
	Runs      int    `json:"runs"`
	Identical bool   `json:"identical"`
	Time      string `json:"time"`
	State     string `json:"stateDigest"`
}

// PolicyStats is one policy's spread of each integer metric.
type PolicyStats struct {
	Policy    PolicyRef `json:"policy"`
	Runs      int       `json:"runs"`
	Completed int       `json:"completed"`
	Metrics   []Spread  `json:"metrics"`
}

// Spread is a metric's minimum, median (lower middle value) and maximum
// over the runs where it is present (N).
type Spread struct {
	Name   string `json:"name"`
	N      int    `json:"n"`
	Min    int64  `json:"min"`
	Median int64  `json:"median"`
	Max    int64  `json:"max"`
}

// Pair compares a policy's run with the baseline's on one scenario and
// seed. Decisions: "same" when both applied the same actions (equal
// action digests), else "different". Outcome: "identical" when the log,
// ledger and state digests are equal; "different" otherwise. Same
// decisions with a different outcome would break determinism: Flag is
// then set.
type Pair struct {
	Scenario  string  `json:"scenario"`
	Seed      int64   `json:"seed"`
	Policy    string  `json:"policy"`
	Against   string  `json:"against"`
	Decisions string  `json:"decisions"`
	Outcome   string  `json:"outcome"`
	Flag      string  `json:"flag,omitempty"`
	Deltas    []Delta `json:"deltas"`
}

// Delta is policy − baseline for one integer metric (absent when either
// run lacks it).
type Delta struct {
	Name  string `json:"name"`
	Value *int64 `json:"value"`
}

// PairCount counts, for a policy against the baseline and one metric, the
// seeds where the policy's value is higher, lower or equal.
type PairCount struct {
	Policy  string `json:"policy"`
	Against string `json:"against"`
	Name    string `json:"name"`
	Higher  int    `json:"higher"`
	Lower   int    `json:"lower"`
	Equal   int    `json:"equal"`
}

// Failed is a run that did not complete.
type Failed struct {
	ID      string  `json:"id"`
	Failure Failure `json:"failure"`
}

// StatusCount counts runs by status.
type StatusCount struct {
	Status string `json:"status"`
	Runs   int    `json:"runs"`
}

// ReplaySummary counts the runs whose saved session replayed and whose
// session batches match the decisions.
type ReplaySummary struct {
	Runs         int `json:"runs"`
	Verified     int `json:"verified"`
	ActionsMatch int `json:"actionsMatch"`
}

// BuildReport compares the runs. It reads only the run records.
func BuildReport(c *Config, recs []*RunRecord) *Report {
	r := &Report{Schema: ReportSchema, Benchmark: c.ID, Engine: version.Number, Protocol: agent.Protocol,
		Scenarios: c.Scenarios, Seeds: c.Seeds, Days: c.Days, Cadence: c.Cadence, Baseline: c.Baseline, Note: reportNote,
		Initial: []Initial{}, Summary: []PolicyStats{}, Pairs: []Pair{}, Counts: []PairCount{}, Failures: []Failed{}, Status: []StatusCount{}}
	for _, p := range c.Policies {
		r.Policies = append(r.Policies, c.ref(p))
	}
	for _, x := range c.External {
		r.External = append(r.External, x.ID)
	}
	find := func(scen string, seed int64, policy string) *RunRecord {
		for _, rec := range recs {
			if rec.Scenario.Ref == scen && rec.Seed == seed && rec.Policy.ID == policy {
				return rec
			}
		}
		return nil
	}
	for _, rec := range recs {
		row := RunRow{ID: rec.ID, Scenario: rec.Scenario.Ref, Seed: rec.Seed, Policy: rec.Policy, Status: rec.Status,
			Counts: rec.Counts, ActionsMatch: rec.ActionsMatch, T90: rec.T90, Metrics: rec.Metrics}
		row.Replay = rec.Replay != nil && rec.Replay.Verified
		if rec.Final != nil {
			row.FinalRun = rec.Final.Digests.Run
		}
		r.Runs = append(r.Runs, row)
		r.Replay.Runs++
		if row.Replay {
			r.Replay.Verified++
		}
		if rec.ActionsMatch {
			r.Replay.ActionsMatch++
		}
		if rec.Failure != nil {
			r.Failures = append(r.Failures, Failed{rec.ID, *rec.Failure})
		}
		i := slices.IndexFunc(r.Status, func(s StatusCount) bool { return s.Status == rec.Status })
		if i < 0 {
			r.Status = append(r.Status, StatusCount{rec.Status, 0})
			i = len(r.Status) - 1
		}
		r.Status[i].Runs++
	}
	for _, scen := range c.Scenarios {
		for _, seed := range c.Seeds {
			in := Initial{Scenario: scen, Seed: seed, Identical: true}
			for _, p := range c.Policies {
				rec := find(scen, seed, p)
				if rec == nil || rec.Initial.Time == "" {
					in.Identical = false
					continue
				}
				if in.Runs == 0 {
					in.Time, in.State = rec.Initial.Time, rec.Initial.Digests.State
				}
				in.Runs++
				first := find(scen, seed, c.Policies[0])
				if first == nil || rec.Initial != first.Initial {
					in.Identical = false
				}
			}
			r.Initial = append(r.Initial, in)
		}
	}
	for _, p := range c.Policies {
		st := PolicyStats{Policy: c.ref(p), Metrics: []Spread{}}
		vals := map[string][]int64{}
		for _, rec := range recs {
			if rec.Policy.ID != p {
				continue
			}
			st.Runs++
			if rec.Status != StatusCompleted {
				continue
			}
			st.Completed++
			for _, v := range rec.Metrics {
				if n, ok := intValue(v); ok {
					vals[v.Name] = append(vals[v.Name], n)
				}
			}
		}
		for _, m := range Metrics {
			if m.Kind != "int" || len(vals[m.Name]) == 0 {
				continue
			}
			v := slices.Clone(vals[m.Name])
			slices.Sort(v)
			st.Metrics = append(st.Metrics, Spread{m.Name, len(v), v[0], v[(len(v)-1)/2], v[len(v)-1]})
		}
		r.Summary = append(r.Summary, st)
	}
	counts := map[[2]string]*PairCount{}
	for _, p := range c.Policies {
		if p == c.Baseline {
			continue
		}
		for _, m := range Metrics {
			if m.Kind == "int" {
				pc := &PairCount{Policy: p, Against: c.Baseline, Name: m.Name}
				counts[[2]string{p, m.Name}] = pc
			}
		}
		for _, scen := range c.Scenarios {
			for _, seed := range c.Seeds {
				a, b := find(scen, seed, p), find(scen, seed, c.Baseline)
				if a == nil || b == nil || a.Status != StatusCompleted || b.Status != StatusCompleted || a.Final == nil || b.Final == nil ||
					len(a.Metrics) != len(Metrics) || len(b.Metrics) != len(Metrics) {
					continue
				}
				pr := Pair{Scenario: scen, Seed: seed, Policy: p, Against: c.Baseline, Decisions: "different", Outcome: "different", Deltas: []Delta{}}
				fa, fb := a.Final.Digests, b.Final.Digests
				if fa.Actions == fb.Actions {
					pr.Decisions = "same"
				}
				if fa.Log == fb.Log && fa.Ledger == fb.Ledger && fa.State == fb.State {
					pr.Outcome = "identical"
				}
				if pr.Decisions == "same" && pr.Outcome != "identical" {
					pr.Flag = "same decisions, different outcome: determinism broken"
				}
				for i, m := range Metrics {
					if m.Kind != "int" {
						continue
					}
					d := Delta{Name: m.Name}
					x, okx := intValue(a.Metrics[i])
					y, oky := intValue(b.Metrics[i])
					if okx && oky {
						v := x - y
						d.Value = &v
						pc := counts[[2]string{p, m.Name}]
						switch {
						case v > 0:
							pc.Higher++
						case v < 0:
							pc.Lower++
						default:
							pc.Equal++
						}
					}
					pr.Deltas = append(pr.Deltas, d)
				}
				r.Pairs = append(r.Pairs, pr)
			}
		}
		for _, m := range Metrics {
			if m.Kind == "int" {
				r.Counts = append(r.Counts, *counts[[2]string{p, m.Name}])
			}
		}
	}
	return r
}

// Text is the report as plain text.
func (r *Report) Text() string {
	var b strings.Builder
	p := func(f string, a ...any) { fmt.Fprintf(&b, f, a...) }
	pols := make([]string, len(r.Policies))
	for i, x := range r.Policies {
		pols[i] = x.ID + "@" + x.Version
		if slices.Contains(r.External, x.ID) {
			pols[i] += " (external)"
		}
	}
	p("DreamTeam benchmark %s · engine %s · %s\n", r.Benchmark, r.Engine, r.Protocol)
	p("scenarios %s · seeds %s · %d days · decisions every %d day(s) at %s\n", strings.Join(r.Scenarios, ", "), joinInts(r.Seeds), r.Days, r.Cadence.EveryDays, r.Cadence.At)
	p("policies %s · paired against %s\n\n", strings.Join(pols, ", "), r.Baseline)
	p("%s\n", wrap(r.Note, 76))

	p("\n1. Runs\n\n")
	key := []string{"operatingProfitCents", "netRevenueCents", "ordersLost", "examsCompleted", "tatP90Sec", "tatBreaches", "overtimeSec", "alertsRaisedAlert"}
	head := []string{"run", "status", "appl/ref/err", "replay"}
	for _, k := range key {
		head = append(head, short(k))
	}
	head = append(head, "T90")
	rows := [][]string{head}
	for _, x := range r.Runs {
		replay := "no"
		if x.Replay && x.ActionsMatch {
			replay = "ok"
		} else if x.Replay {
			replay = "replay only"
		}
		row := []string{x.ID, x.Status, fmt.Sprintf("%d/%d/%d", x.Counts.Applied, x.Counts.Refused, x.Counts.ProtocolErrors), replay}
		for _, k := range key {
			row = append(row, metricText(k, x.Metrics))
		}
		rows = append(rows, append(row, t90Text(x.T90)))
	}
	table(&b, rows)

	p("\n2. Initial conditions (same scenario and seed)\n\n")
	rows = [][]string{{"scenario", "seed", "runs", "start", "state digest", "identical"}}
	for _, x := range r.Initial {
		rows = append(rows, []string{x.Scenario, fmt.Sprint(x.Seed), fmt.Sprint(x.Runs), x.Time, abbrev(x.State), yesNo(x.Identical)})
	}
	table(&b, rows)

	p("\n3. Policy summary: min / median / max over completed runs\n\n")
	head = []string{"metric"}
	for _, s := range r.Summary {
		head = append(head, fmt.Sprintf("%s@%s (%d/%d)", s.Policy.ID, s.Policy.Version, s.Completed, s.Runs))
	}
	rows = [][]string{head}
	for _, m := range Metrics {
		if m.Kind != "int" {
			continue
		}
		row := []string{m.Name}
		for _, s := range r.Summary {
			cell := "–"
			if i := slices.IndexFunc(s.Metrics, func(x Spread) bool { return x.Name == m.Name }); i >= 0 {
				x := s.Metrics[i]
				cell = fmt.Sprintf("%s / %s / %s", unitText(m.Name, x.Min), unitText(m.Name, x.Median), unitText(m.Name, x.Max))
			}
			row = append(row, cell)
		}
		rows = append(rows, row)
	}
	table(&b, rows)

	p("\n4. Paired comparisons: policy − %s, same scenario and seed\n\n", r.Baseline)
	if len(r.Pairs) == 0 {
		p("(none)\n")
	} else {
		head = []string{"scenario", "seed", "policy", "decisions", "outcome"}
		for _, k := range key {
			head = append(head, "Δ"+short(k))
		}
		rows = [][]string{head}
		for _, x := range r.Pairs {
			row := []string{x.Scenario, fmt.Sprint(x.Seed), x.Policy, x.Decisions, x.Outcome}
			for _, k := range key {
				cell := "–"
				for _, d := range x.Deltas {
					if d.Name == k && d.Value != nil {
						cell = signed(k, *d.Value)
					}
				}
				row = append(row, cell)
			}
			rows = append(rows, row)
		}
		table(&b, rows)
		for _, x := range r.Pairs {
			if x.Flag != "" {
				p("FLAG %s seed %d %s: %s\n", x.Scenario, x.Seed, x.Policy, x.Flag)
			}
		}
		p("\nSeeds where the policy's value is higher / lower / equal than %s's:\n\n", r.Baseline)
		head = []string{"metric"}
		var ps []string
		for _, pc := range r.Counts {
			if !slices.Contains(ps, pc.Policy) {
				ps = append(ps, pc.Policy)
				head = append(head, pc.Policy)
			}
		}
		rows = [][]string{head}
		for _, m := range Metrics {
			if m.Kind != "int" {
				continue
			}
			row := []string{m.Name}
			for _, pol := range ps {
				for _, pc := range r.Counts {
					if pc.Policy == pol && pc.Name == m.Name {
						row = append(row, fmt.Sprintf("%d / %d / %d", pc.Higher, pc.Lower, pc.Equal))
					}
				}
			}
			rows = append(rows, row)
		}
		table(&b, rows)
	}

	p("\n5. Failures, refusals and protocol errors\n\n")
	rows = [][]string{{"status", "runs"}}
	for _, s := range r.Status {
		rows = append(rows, []string{s.Status, fmt.Sprint(s.Runs)})
	}
	table(&b, rows)
	p("\n")
	rows = [][]string{{"policy", "decision points", "batches", "applied", "refused", "protocol errors", "denied requests", "request errors"}}
	for _, pol := range r.Policies {
		var n Counts
		for _, x := range r.Runs {
			if x.Policy.ID == pol.ID {
				n.DecisionPoints += x.Counts.DecisionPoints
				n.Batches += x.Counts.Batches
				n.Applied += x.Counts.Applied
				n.Refused += x.Counts.Refused
				n.ProtocolErrors += x.Counts.ProtocolErrors
				n.Denied += x.Counts.Denied
				n.RequestErrors += x.Counts.RequestErrors
			}
		}
		rows = append(rows, []string{pol.ID, fmt.Sprint(n.DecisionPoints), fmt.Sprint(n.Batches), fmt.Sprint(n.Applied),
			fmt.Sprint(n.Refused), fmt.Sprint(n.ProtocolErrors), fmt.Sprint(n.Denied), fmt.Sprint(n.RequestErrors)})
	}
	table(&b, rows)
	for _, f := range r.Failures {
		p("%s: %s at decision point %d (%s): %s\n", f.ID, f.Failure.Class, f.Failure.Point, f.Failure.Time, f.Failure.Message)
	}

	p("\n6. Replay verification\n\n")
	p("%d of %d saved sessions replayed with every digest verified; %d of %d match the applied decisions batch for batch.\n",
		r.Replay.Verified, r.Replay.Runs, r.Replay.ActionsMatch, r.Replay.Runs)
	p("This is an integrity check of these files, not an independent certification.\n")
	return b.String()
}

// table writes rows as left-aligned columns.
func table(b *strings.Builder, rows [][]string) {
	w := map[int]int{}
	for _, r := range rows {
		for i, c := range r {
			w[i] = max(w[i], len([]rune(c)))
		}
	}
	for _, r := range rows {
		var line strings.Builder
		for i, c := range r {
			if i > 0 {
				line.WriteString("  ")
			}
			line.WriteString(c)
			if i < len(r)-1 {
				line.WriteString(strings.Repeat(" ", w[i]-len([]rune(c))))
			}
		}
		b.WriteString(strings.TrimRight(line.String(), " ") + "\n")
	}
}

var shortNames = map[string]string{"operatingProfitCents": "profit", "netRevenueCents": "net revenue", "ordersLost": "lost",
	"examsCompleted": "exams", "tatP90Sec": "TAT p90", "tatBreaches": "breaches", "overtimeSec": "overtime", "alertsRaisedAlert": "alerts"}

func short(k string) string { return shortNames[k] }

func metricText(name string, vs []Value) string {
	for _, v := range vs {
		if v.Name == name {
			if n, ok := intValue(v); ok {
				return unitText(name, n)
			}
		}
	}
	return "–"
}

// unitText shows cents as whole dollars and seconds as whole minutes
// (rounded half away from zero); counts as they are.
func unitText(name string, v int64) string {
	switch {
	case strings.HasSuffix(name, "Cents"):
		s := group(roundDiv(v, 100))
		if v < 0 && s[0] != '-' {
			return "-$0"
		}
		if strings.HasPrefix(s, "-") {
			return "-$" + s[1:]
		}
		return "$" + s
	case strings.HasSuffix(name, "Sec"):
		return group(roundDiv(v, 60)) + " min"
	}
	return group(v)
}

func signed(name string, v int64) string {
	s := unitText(name, v)
	if v > 0 {
		return "+" + s
	}
	return s
}

func roundDiv(v, d int64) int64 {
	if v < 0 {
		return -((-v + d/2) / d)
	}
	return (v + d/2) / d
}

func group(v int64) string {
	s := fmt.Sprint(v)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	if neg {
		return "-" + s
	}
	return s
}

func t90Text(t T90Info) string {
	switch t.Status {
	case T90Evaluated:
		s := t.Verdict
		if t.Projected {
			s += " (projected from day " + string(t.AsOfDay) + ")"
		}
		return s + " · " + t.PolicyID
	case T90NotApplicable:
		return "not applicable"
	case T90NotRequested:
		return "not requested"
	}
	return t.Status
}

func joinInts(v []int64) string {
	s := make([]string, len(v))
	for i, x := range v {
		s[i] = fmt.Sprint(x)
	}
	return strings.Join(s, ", ")
}

func abbrev(d string) string {
	if len(d) > 12 {
		return d[:12] + "…"
	}
	return d
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "NO"
}

func wrap(s string, width int) string {
	var b strings.Builder
	n := 0
	for i, w := range strings.Fields(s) {
		if i > 0 && n+1+len(w) > width {
			b.WriteByte('\n')
			n = 0
		} else if i > 0 {
			b.WriteByte(' ')
			n++
		}
		b.WriteString(w)
		n += len(w)
	}
	return b.String()
}
