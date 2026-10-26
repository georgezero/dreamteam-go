// Package bench is the agent evaluation harness (V0.4a): it runs agent
// policies on identical scenarios and seeds through the public agent
// protocol (dreamteam.agent/v1), records their decisions and outcomes,
// verifies each saved session by replay, and compares the policies.
//
// The harness is a client of the protocol, as an external agent is. A
// policy sees the session only through a gate that passes the protocol's
// read and check operations (hello, observe, sections, actions, validate,
// preview, digests); the harness alone applies the policy's batches,
// advances time, saves and asks for the T90 evaluation. Outcomes are the
// simulator's own measurements: the canonical result of the replayed
// session (dreamteam.result/v1) and, where the evaluation policy in force
// applies, its T90 report. The harness adds no scoring rule.
// docs/BENCHMARK.md is the specification.
package bench

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"

	"github.com/georgezero/dreamteam-go/docs/evaluation"
	"github.com/georgezero/dreamteam-go/internal/eval"
	"github.com/georgezero/dreamteam-go/internal/scenario"
)

// ConfigSchema identifies the configuration format.
const ConfigSchema = "dreamteam.benchmark/v1"

// Config is a benchmark: the scenarios, seeds, horizon, policies and
// decision cadence of a matrix of runs, and where its results go.
type Config struct {
	Schema      string `json:"schema"`
	ID          string `json:"id"`
	Description string `json:"description"`
	// Scenarios are bundled scenario IDs (dreamteam scenario list).
	Scenarios []string `json:"scenarios"`
	// Seeds are explicit; SeedPlan says how they were chosen, before any
	// outcome was seen.
	Seeds    []int64 `json:"seeds"`
	SeedPlan string  `json:"seedPlan"`
	// Days is the horizon: decision points run from the first cadence
	// time after the takeover for Days days.
	Days    int     `json:"days"`
	Cadence Cadence `json:"cadence"`
	// Policies are built-in policies' IDs and External policies' IDs.
	Policies []string `json:"policies"`
	// External are external policy processes (V0.4b), named in Policies
	// by their IDs.
	External []ExternalPolicy `json:"external,omitempty"`
	// Baseline is the policy the others are paired with in the report.
	Baseline   string     `json:"baseline"`
	Output     string     `json:"output"` // relative to the configuration file's directory
	Evaluation Evaluation `json:"evaluation"`
}

// Cadence is when a policy decides: every EveryDays days at At
// (practice-local "HH:MM").
type Cadence struct {
	EveryDays int    `json:"everyDays"`
	At        string `json:"at"`
}

// ExternalPolicy is a policy run as a separate process (V0.4b), speaking
// the policy protocol (dreamteam.benchmark-policy/v1) on its standard
// input and output. Command is its program and arguments: an element that
// begins with "./" or "../" is a file relative to the directory the
// benchmark is run from (its SHA-256 goes in the manifest); every other
// element is passed as it is, and a program without a path is looked up
// in PATH. TimeoutSeconds bounds each decision point and the exit after
// the last one.
type ExternalPolicy struct {
	ID             string   `json:"id"`
	Version        string   `json:"version"`
	Description    string   `json:"description"`
	Command        []string `json:"command"`
	TimeoutSeconds int      `json:"timeoutSeconds"`
}

// Evaluation options. T90 asks for the T90 evaluation of each run where
// the evaluation policy in force applies (otherwise it is reported as not
// applicable).
type Evaluation struct {
	T90 bool `json:"t90"`
}

// ErrConfig marks an invalid configuration.
var ErrConfig = errors.New("invalid benchmark configuration")

// referenceSeeds are the V0.2 evaluation's reference seeds (1001–1020):
// never a benchmark's development seeds.
func referenceSeed(s int64) bool { return s >= 1001 && s <= 1020 }

var (
	idPattern    = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{0,63}$`)
	clockPattern = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)
)

// LoadConfig reads and validates a configuration. It returns the file's
// bytes too: the results keep them verbatim.
func LoadConfig(path string) (*Config, []byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	c, err := DecodeConfig(raw)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", path, err)
	}
	return c, raw, nil
}

// DecodeConfig decodes a configuration strictly and validates it.
func DecodeConfig(raw []byte) (*Config, error) {
	var c Config
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConfig, err)
	}
	if dec.More() {
		return nil, fmt.Errorf("%w: data after the configuration object", ErrConfig)
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

// Validate checks every field before anything runs.
func (c *Config) Validate() error {
	bad := func(f string, a ...any) error { return fmt.Errorf("%w: "+f, append([]any{ErrConfig}, a...)...) }
	switch {
	case c.Schema != ConfigSchema:
		return bad("schema must be %q, not %q", ConfigSchema, c.Schema)
	case !idPattern.MatchString(c.ID):
		return bad("id %q must be 1–64 lowercase letters, digits, dots or hyphens", c.ID)
	case len(c.Scenarios) == 0:
		return bad("scenarios: at least one bundled scenario ID")
	case len(c.Seeds) == 0:
		return bad("seeds: an explicit list of development seeds (none are generated)")
	case c.SeedPlan == "":
		return bad("seedPlan: say how the seeds were chosen")
	case c.Days < 1 || c.Days > 366:
		return bad("days must be 1–366, not %d", c.Days)
	case c.Cadence.EveryDays < 1 || c.Cadence.EveryDays > c.Days:
		return bad("cadence.everyDays must be 1–days (%d), not %d", c.Days, c.Cadence.EveryDays)
	case !clockPattern.MatchString(c.Cadence.At):
		return bad("cadence.at must be a practice-local time HH:MM, not %q", c.Cadence.At)
	case len(c.Policies) == 0:
		return bad("policies: at least one policy (%v, or an external policy)", PolicyIDs())
	case !slices.Contains(c.Policies, c.Baseline):
		return bad("baseline %q must be one of the policies %v", c.Baseline, c.Policies)
	case c.Output == "" || filepath.IsAbs(c.Output):
		return bad("output must be a directory relative to the configuration file, not %q", c.Output)
	}
	bundled := scenario.List()
	for i, s := range c.Scenarios {
		if !slices.Contains(bundled, s) {
			return bad("scenarios[%d] %q is not a bundled scenario %v", i, s, bundled)
		}
		if slices.Contains(c.Scenarios[:i], s) {
			return bad("scenarios[%d] %q is listed twice", i, s)
		}
	}
	for i, s := range c.Seeds {
		switch {
		case s <= 0:
			return bad("seeds[%d] must be positive, not %d", i, s)
		case slices.Contains(c.Seeds[:i], s):
			return bad("seeds[%d] %d is listed twice", i, s)
		case referenceSeed(s):
			return bad("seeds[%d] %d is a reference evaluation seed (1001–1020), not a development seed", i, s)
		}
	}
	// The sealed holdout: refused without naming which seed it is.
	if err := eval.GuardManifest(evaluation.Holdout, c.Seeds); err != nil {
		return bad("seeds: %v", err)
	}
	for i, x := range c.External {
		switch {
		case !idPattern.MatchString(x.ID):
			return bad("external[%d].id %q must be 1–64 lowercase letters, digits, dots or hyphens", i, x.ID)
		case slices.Contains(PolicyIDs(), x.ID):
			return bad("external[%d].id %q is a built-in policy's", i, x.ID)
		case slices.ContainsFunc(c.External[:i], func(y ExternalPolicy) bool { return y.ID == x.ID }):
			return bad("external[%d].id %q is listed twice", i, x.ID)
		case x.Version == "" || x.Description == "":
			return bad("external[%d] (%s): version and description are required", i, x.ID)
		case len(x.Command) == 0 || x.Command[0] == "":
			return bad("external[%d] (%s): command names the program and its arguments", i, x.ID)
		case x.TimeoutSeconds < 1 || x.TimeoutSeconds > 3600:
			return bad("external[%d] (%s): timeoutSeconds must be 1–3600, not %d", i, x.ID, x.TimeoutSeconds)
		case !slices.Contains(c.Policies, x.ID):
			return bad("external[%d] (%s) is not one of the policies %v", i, x.ID, c.Policies)
		}
	}
	for i, p := range c.Policies {
		if _, ok := c.policy(p); !ok {
			return bad("policies[%d] %q is not a policy %v or an external one", i, p, PolicyIDs())
		}
		if slices.Contains(c.Policies[:i], p) {
			return bad("policies[%d] %q is listed twice", i, p)
		}
	}
	return nil
}

// policy finds a policy by ID: a built-in one, or one of External (with
// no constructor: an external policy is a process, started per run).
func (c *Config) policy(id string) (PolicySpec, bool) {
	if p, ok := lookupPolicy(id); ok {
		return p, true
	}
	i := slices.IndexFunc(c.External, func(x ExternalPolicy) bool { return x.ID == id })
	if i < 0 {
		return PolicySpec{}, false
	}
	x := c.External[i]
	return PolicySpec{ID: x.ID, Version: x.Version, Description: x.Description,
		External: &ExternalSpec{Command: x.Command, TimeoutSeconds: x.TimeoutSeconds}}, true
}

// ref is a policy's ID and version.
func (c *Config) ref(id string) PolicyRef {
	spec, _ := c.policy(id)
	return PolicyRef{spec.ID, spec.Version}
}
