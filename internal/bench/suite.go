package bench

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"

	"github.com/georgezero/dreamteam-go/docs/evaluation"
	"github.com/georgezero/dreamteam-go/internal/scenario"
)

// A benchmark suite (V0.6a, docs/BENCHMARK_SUITE.md) is a versioned set of
// scenario entries, each a bundled scenario with its recorded fingerprint,
// and the evaluation contract's dimensions. A suite run (V0.6b) plays a
// selection of its entries against policies and seeds through the same
// harness as any benchmark: the suite-run configuration is expanded into
// an ordinary benchmark configuration (config.json in the results), and
// every run's files are the ones `benchmark run` of that configuration
// gives. A suite run adds four files, hashed in the manifest: the suite-run
// configuration and the suite file, verbatim, and the contract report
// (contract.json, contract.txt).

// Schemas.
const (
	SuiteSchema    = "dreamteam.benchmark-suite/v1"
	SuiteRunSchema = "dreamteam.benchmark-suite-run/v1"
	ContractSchema = "dreamteam.benchmark-contract/v1"
)

// The suite run's own files in a results directory.
const (
	FileSuiteRun     = "suite-run.json" // the suite-run configuration, verbatim
	FileSuite        = "suite.json"     // the suite file, verbatim
	FileContractJSON = "contract.json"
	FileContractText = "contract.txt"
)

// Suite is a suite file.
type Suite struct {
	Schema      string       `json:"schema"`
	ID          string       `json:"id"`
	Version     int          `json:"version"`
	Split       string       `json:"split"`
	Contract    string       `json:"contract"`
	Description string       `json:"description"`
	HorizonDays int          `json:"horizonDays"`
	Cadence     Cadence      `json:"cadence"`
	Scenarios   []SuiteEntry `json:"scenarios"`
	// Dimensions are the contract's six dimensions, each a list of the
	// harness's metric names.
	Dimensions map[string][]string `json:"dimensions"`
	// Missing are the measures the contract asks for that no metric gives.
	Missing []string `json:"missing"`
}

// SuiteEntry is one scenario of a suite.
type SuiteEntry struct {
	ID          string `json:"id"`
	Version     int    `json:"version"`
	Category    string `json:"category"`
	Scenario    string `json:"scenario"` // a bundled scenario ID
	Fingerprint string `json:"fingerprint"`
	Source      string `json:"source"`
	Challenge   string `json:"challenge"`
	// T90 is the evaluation policy that applies, or "not_applicable".
	T90 string `json:"t90"`
}

// Dimensions in report order, with their names.
var Dimensions = []struct{ Key, Name string }{
	{"operational", "Operational performance"},
	{"financial", "Financial performance"},
	{"staffingCapacity", "Staffing and capacity"},
	{"serviceReliability", "Service reliability"},
	{"sustainability", "Long-term sustainability"},
	{"actions", "Actions"},
}

// SuiteRunConfig is a suite run: which entries of which suite, with which
// policies and seeds, for how long. Days and Cadence default to the
// suite's; Entries to all of them, in the suite's order.
type SuiteRunConfig struct {
	Schema      string           `json:"schema"`
	ID          string           `json:"id"`
	Description string           `json:"description"`
	Suite       string           `json:"suite"` // the suite file, relative to this configuration's directory
	Entries     []string         `json:"entries,omitempty"`
	Seeds       []int64          `json:"seeds"`
	SeedPlan    string           `json:"seedPlan"`
	Days        int              `json:"days,omitempty"`
	Cadence     *Cadence         `json:"cadence,omitempty"`
	Policies    []string         `json:"policies"`
	External    []ExternalPolicy `json:"external,omitempty"`
	Baseline    string           `json:"baseline"`
	Output      string           `json:"output"`
	Evaluation  Evaluation       `json:"evaluation"`
}

// SuiteRun is a loaded suite run: both files verbatim, decoded, the
// selected entries and the expanded benchmark configuration.
type SuiteRun struct {
	RunRaw, SuiteRaw, ConfigRaw []byte
	Run                         *SuiteRunConfig
	Suite                       *Suite
	Entries                     []SuiteEntry
	Config                      *Config
}

// LoadSuiteRun reads a suite-run configuration and the suite file it
// names, and checks and expands them.
func LoadSuiteRun(path string) (*SuiteRun, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var head struct{ Suite string }
	json.Unmarshal(raw, &head)
	if head.Suite == "" {
		return nil, fmt.Errorf("%s: %w: suite: the suite file", path, ErrConfig)
	}
	suite, err := os.ReadFile(filepath.Join(filepath.Dir(path), filepath.FromSlash(head.Suite)))
	if err != nil {
		return nil, fmt.Errorf("%s: suite: %w", path, err)
	}
	sr, err := DecodeSuiteRun(raw, suite)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return sr, nil
}

func strict(raw []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	if dec.More() {
		return fmt.Errorf("data after the object")
	}
	return nil
}

// DecodeSuiteRun decodes a suite-run configuration and its suite file
// strictly, checks every selected entry's bundled scenario against its
// recorded fingerprint and T90 applicability, and expands the run into a
// benchmark configuration, validated like any other.
func DecodeSuiteRun(runRaw, suiteRaw []byte) (*SuiteRun, error) {
	bad := func(f string, a ...any) error { return fmt.Errorf("%w: "+f, append([]any{ErrConfig}, a...)...) }
	var r SuiteRunConfig
	if err := strict(runRaw, &r); err != nil {
		return nil, bad("%v", err)
	}
	if r.Schema != SuiteRunSchema {
		return nil, bad("schema must be %q, not %q", SuiteRunSchema, r.Schema)
	}
	var s Suite
	if err := strict(suiteRaw, &s); err != nil {
		return nil, bad("suite: %v", err)
	}
	if err := s.Check(); err != nil {
		return nil, bad("suite %s: %v", s.ID, err)
	}
	ids := make([]string, len(s.Scenarios))
	for i, e := range s.Scenarios {
		ids[i] = e.ID
	}
	sel := r.Entries
	if len(sel) == 0 {
		sel = ids
	}
	sr := &SuiteRun{RunRaw: runRaw, SuiteRaw: suiteRaw, Run: &r, Suite: &s}
	var scens []string
	for i, id := range sel {
		j := slices.Index(ids, id)
		if j < 0 {
			return nil, bad("entries[%d] %q is not an entry of suite %s %v", i, id, s.ID, ids)
		}
		if slices.Contains(sel[:i], id) {
			return nil, bad("entries[%d] %q is listed twice", i, id)
		}
		sr.Entries = append(sr.Entries, s.Scenarios[j])
		scens = append(scens, s.Scenarios[j].Scenario)
	}
	days, cad := r.Days, s.Cadence
	if days == 0 {
		days = s.HorizonDays
	}
	if r.Cadence != nil {
		cad = *r.Cadence
	}
	c := &Config{Schema: ConfigSchema, ID: r.ID,
		Description: fmt.Sprintf("%s [suite %s version %d, entries %v; expanded from %s]", r.Description, s.ID, s.Version, sel, FileSuiteRun),
		Scenarios:   scens, Seeds: r.Seeds, SeedPlan: r.SeedPlan, Days: days, Cadence: cad, Policies: r.Policies,
		External: r.External, Baseline: r.Baseline, Output: r.Output, Evaluation: r.Evaluation}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	raw, err := json.MarshalIndent(c, "", " ")
	if err != nil {
		return nil, err
	}
	sr.Config, sr.ConfigRaw = c, append(raw, '\n')
	return sr, nil
}

// Check checks a suite file: its header, its entries (unique, each a
// bundled scenario with the recorded fingerprint, T90 claimed exactly
// where the evaluation policy in force applies) and its dimensions (the
// six, each metric the harness's and listed once).
func (s *Suite) Check() error {
	if s.Schema != SuiteSchema || !idPattern.MatchString(s.ID) || s.Version < 1 || s.Contract != ContractSchema ||
		s.HorizonDays < 1 || s.Cadence.EveryDays < 1 || !clockPattern.MatchString(s.Cadence.At) || len(s.Scenarios) == 0 {
		return fmt.Errorf("header: schema %q, id %q, version %d, contract %q, horizon %d, cadence %+v, %d scenarios",
			s.Schema, s.ID, s.Version, s.Contract, s.HorizonDays, s.Cadence, len(s.Scenarios))
	}
	var pol struct {
		ID       string
		Scenario struct{ Fingerprint string }
	}
	if err := json.Unmarshal(evaluation.PolicyV4, &pol); err != nil {
		return err
	}
	bundled := scenario.List()
	var ids, scens []string
	for _, e := range s.Scenarios {
		if !idPattern.MatchString(e.ID) || e.Version < 1 || slices.Contains(ids, e.ID) || slices.Contains(scens, e.Scenario) {
			return fmt.Errorf("entry %q: id, version or scenario invalid or repeated", e.ID)
		}
		ids, scens = append(ids, e.ID), append(scens, e.Scenario)
		if !slices.Contains(bundled, e.Scenario) {
			return fmt.Errorf("entry %s: %q is not a bundled scenario", e.ID, e.Scenario)
		}
		f, err := scenario.ResolveFile(e.Scenario)
		if err != nil {
			return fmt.Errorf("entry %s: %w", e.ID, err)
		}
		fp, err := scenario.Fingerprint(f)
		if err != nil {
			return err
		}
		if fp != e.Fingerprint {
			return fmt.Errorf("entry %s: scenario %s has fingerprint %s, the suite records %s", e.ID, e.Scenario, fp, e.Fingerprint)
		}
		want := T90NotApplicable
		if fp == pol.Scenario.Fingerprint {
			want = pol.ID
		}
		if e.T90 != want {
			return fmt.Errorf("entry %s: t90 %q, the evaluation policy in force says %q", e.ID, e.T90, want)
		}
	}
	known := map[string]bool{}
	for _, m := range Metrics {
		known[m.Name] = true
	}
	var seen []string
	for _, d := range Dimensions {
		if len(s.Dimensions[d.Key]) == 0 {
			return fmt.Errorf("dimension %s is empty", d.Key)
		}
		for _, m := range s.Dimensions[d.Key] {
			if !known[m] || slices.Contains(seen, m) {
				return fmt.Errorf("dimension %s: metric %q unknown or listed twice", d.Key, m)
			}
			seen = append(seen, m)
		}
	}
	if len(s.Dimensions) != len(Dimensions) {
		return fmt.Errorf("%d dimensions, want %d", len(s.Dimensions), len(Dimensions))
	}
	return nil
}

// RunSuite plays a suite run into dir (which must not exist or be empty):
// the expanded configuration's matrix, then the contract report, with
// the suite's files in the manifest.
func RunSuite(sr *SuiteRun, dir string, progress io.Writer) (*Manifest, error) {
	return matrix(sr.Config, sr.ConfigRaw, dir, progress, sr)
}

// LoadSuiteResults reads a results directory's suite run, or nil when the
// results are not a suite run's.
func LoadSuiteResults(dir string, m *Manifest) (*SuiteRun, error) {
	if len(m.Suite) == 0 {
		return nil, nil
	}
	runRaw, err := os.ReadFile(filepath.Join(dir, FileSuiteRun))
	if err != nil {
		return nil, err
	}
	suiteRaw, err := os.ReadFile(filepath.Join(dir, FileSuite))
	if err != nil {
		return nil, err
	}
	return DecodeSuiteRun(runRaw, suiteRaw)
}
