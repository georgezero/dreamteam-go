package eval

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// The holdout protocol's artifact contract (docs/evaluation/README.md §9):
// the strategy set and its strategy files, the owner's unseal record, and
// the run manifest a canonical run commits. Authorize validates the whole
// contract and is the only way to obtain the holdout's seeds for a run;
// CommitRun writes a first run's outputs once and afterwards only accepts
// a byte-identical reproduction.

const (
	StrategySchema    = "dreamteam.strategy/v1"
	StrategySetSchema = "dreamteam.strategyset/v1"
	UnsealSchema      = "dreamteam.holdout-unseal/v1"
	RunSchema         = "dreamteam.holdoutrun/v1"
)

// Strategy is one frozen plan: dated batches of actions, applied through
// World.Apply on every seed at the same instants.
type Strategy struct {
	Schema      string `json:"schema"`
	Name        string `json:"name"`
	Description string `json:"description"`
	// TeleOnlyOf names the strategy this one differs from only by
	// teleradiology actions: their bookings must be identical.
	TeleOnlyOf string          `json:"teleOnlyOf,omitempty"`
	Batches    []StrategyBatch `json:"batches"`
}

// StrategyBatch is one batch, at a practice-local ISO time.
type StrategyBatch struct {
	Time    string       `json:"time"`
	Actions []api.Action `json:"actions"`
}

// StrategySet is the frozen strategy set: the scenario and every plan,
// each pinned by the SHA-256 of its file. The baseline is the no-action
// plan every candidate is judged against.
type StrategySet struct {
	Schema           string `json:"schema"`
	ID               string `json:"id"`
	Description      string `json:"description"`
	ScenarioManifest string `json:"scenarioManifest"` // a calibration manifest (scenario and overlay), repository-relative
	Baseline         string `json:"baseline"`
	Plans            []struct {
		Name   string `json:"name"`
		Path   string `json:"path"` // repository-relative
		SHA256 string `json:"sha256"`
	} `json:"plans"`
}

// Unseal is the owner's one-time record that opens the holdout for the
// canonical runner. It pins every artifact of the run by digest, the
// exact engine, and the one primary strategy the holdout confirms (the
// others are exploratory: no multiplicity procedure is frozen).
type Unseal struct {
	Schema           string `json:"schema"`
	Holdout          string `json:"holdout"`
	HoldoutSHA256    string `json:"holdoutManifestSHA256"`
	Policy           string `json:"policy"` // repository-relative
	PolicyDigest     string `json:"policyDigest"`
	StrategySet      string `json:"strategySet"` // repository-relative
	StrategiesSHA256 string `json:"strategiesSHA256"`
	Engine           string `json:"engine"`
	Primary          string `json:"primary"`
	ApprovedBy       string `json:"approvedBy"`
	ApprovedOn       string `json:"approvedOn"` // YYYY-MM-DD
	Statement        string `json:"statement"`
}

// Repo answers the protocol's questions about the repository: whether
// files are committed with no change, and who committed a file first.
type Repo interface {
	Committed(paths ...string) error
	FirstAuthor(path string) (string, error)
}

// Protocol is an authorized holdout run: every artifact validated against
// the unseal record. Its zero value is not authorized, and only Authorize
// makes one that is.
type Protocol struct {
	Root          string
	Unseal        Unseal
	UnsealSHA256  string
	Holdout       *Holdout
	Seeds         []int64
	Policy        *Policy
	PolicyDigest  string
	Set           StrategySet
	Strategies    map[string]Strategy
	StrategySHA   map[string]string // plan → its file's SHA-256
	Committer     string            // who committed the unseal record
	authorized    bool
	unsealPath    string
	holdoutPath   string
	strategyPaths []string
}

// Authorized reports whether p came from Authorize.
func (p *Protocol) Authorized() bool { return p != nil && p.authorized }

// Covers reports whether seed is one of an authorized protocol's holdout
// seeds.
func (p *Protocol) Covers(seed int64) bool { return p.Authorized() && slices.Contains(p.Seeds, seed) }

// Plans are the strategy set's plan names in set order.
func (p *Protocol) Plans() []string {
	var out []string
	for _, x := range p.Set.Plans {
		out = append(out, x.Name)
	}
	return out
}

// Paths are where Authorize finds the manifest and the unseal record,
// repository-relative.
type Paths struct{ Holdout, Unseal string }

// RepoPaths are the repository's own.
var RepoPaths = Paths{Holdout: HoldoutPath, Unseal: "docs/evaluation/holdout-unseal.json"}

func sha(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func decodeStrict(raw []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

// Authorize validates the holdout protocol at root for a run by engine:
// the manifest regenerates to its digest; the unseal record exists, is
// complete and names this holdout, the manifest's bytes, the policy
// (loading, at its digest), the strategy set (at its digest, every
// strategy file at its own), this exact engine, a primary strategy that is
// not the baseline and an approval date; every one of these files is
// committed with no change; and the unseal record was committed by its
// approver. A missing unseal record is ErrSealed.
func Authorize(root string, paths Paths, repo Repo, engine string) (*Protocol, error) {
	p := &Protocol{Root: root, Strategies: map[string]Strategy{}, StrategySHA: map[string]string{}, unsealPath: paths.Unseal, holdoutPath: paths.Holdout}
	at := func(rel string) string { return filepath.Join(root, filepath.FromSlash(rel)) }
	hraw, err := os.ReadFile(at(paths.Holdout))
	if err != nil {
		return nil, err
	}
	if p.Holdout, p.Seeds, err = LoadHoldout(at(paths.Holdout)); err != nil {
		return nil, err
	}
	uraw, err := os.ReadFile(at(paths.Unseal))
	switch {
	case errors.Is(err, os.ErrNotExist):
		return nil, fmt.Errorf("%w: no unseal record %s", ErrSealed, paths.Unseal)
	case err != nil:
		return nil, err
	}
	u := &p.Unseal
	if err := decodeStrict(uraw, u); err != nil {
		return nil, fmt.Errorf("%s: %w", paths.Unseal, err)
	}
	p.UnsealSHA256 = sha(uraw)
	bad := func(format string, args ...any) error {
		return fmt.Errorf("%w: %s: %s", ErrSealed, paths.Unseal, fmt.Sprintf(format, args...))
	}
	if _, derr := simtime.ParseDate(u.ApprovedOn); derr != nil {
		return nil, bad("approvedOn %q is not a date", u.ApprovedOn)
	}
	switch {
	case u.Schema != UnsealSchema:
		return nil, bad("schema %q", u.Schema)
	case u.Holdout != p.Holdout.ID || u.HoldoutSHA256 != sha(hraw):
		return nil, bad("does not name holdout %s at its manifest's digest", p.Holdout.ID)
	case u.Engine != engine:
		return nil, bad("names engine %q, this build is %q", u.Engine, engine)
	case strings.TrimSpace(u.ApprovedBy) == "" || strings.TrimSpace(u.Statement) == "":
		return nil, bad("no approver or statement")
	}
	if p.Policy, p.PolicyDigest, err = LoadPolicy(at(u.Policy)); err != nil {
		return nil, bad("policy: %v", err)
	}
	if p.PolicyDigest != u.PolicyDigest {
		return nil, bad("policy %s is at %s, not the approved %s", u.Policy, p.PolicyDigest, u.PolicyDigest)
	}
	sraw, err := os.ReadFile(at(u.StrategySet))
	if err != nil {
		return nil, bad("strategy set: %v", err)
	}
	if sha(sraw) != u.StrategiesSHA256 {
		return nil, bad("strategy set %s is not at the approved digest", u.StrategySet)
	}
	set := &p.Set
	if err := decodeStrict(sraw, set); err != nil || set.Schema != StrategySetSchema {
		return nil, bad("strategy set %s: %v (schema %q)", u.StrategySet, err, set.Schema)
	}
	files := []string{paths.Holdout, paths.Unseal, u.Policy, u.StrategySet, set.ScenarioManifest}
	for _, x := range set.Plans {
		if _, dup := p.Strategies[x.Name]; dup || x.Name == "" {
			return nil, bad("strategy %q twice or unnamed", x.Name)
		}
		raw, err := os.ReadFile(at(x.Path))
		if err != nil {
			return nil, bad("strategy %s: %v", x.Name, err)
		}
		if sha(raw) != x.SHA256 {
			return nil, bad("strategy %s (%s) is not at its pinned digest", x.Name, x.Path)
		}
		var st Strategy
		if err := decodeStrict(raw, &st); err != nil || st.Schema != StrategySchema || st.Name != x.Name {
			return nil, bad("strategy %s: %v (schema %q, name %q)", x.Path, err, st.Schema, st.Name)
		}
		p.Strategies[x.Name], p.StrategySHA[x.Name] = st, x.SHA256
		files = append(files, x.Path)
		p.strategyPaths = append(p.strategyPaths, x.Path)
	}
	switch {
	case len(set.Plans) < 2:
		return nil, bad("the strategy set needs the baseline and at least one candidate")
	case p.Strategies[set.Baseline].Name == "":
		return nil, bad("baseline %q is not in the strategy set", set.Baseline)
	case len(p.Strategies[set.Baseline].Batches) != 0:
		return nil, bad("baseline %q takes actions", set.Baseline)
	case u.Primary == set.Baseline || p.Strategies[u.Primary].Name == "":
		return nil, bad("primary %q is not a candidate of the strategy set", u.Primary)
	}
	for _, st := range p.Strategies {
		if st.TeleOnlyOf != "" && (st.TeleOnlyOf == st.Name || p.Strategies[st.TeleOnlyOf].Name == "") {
			return nil, bad("strategy %s: teleOnlyOf %q is not another strategy of the set", st.Name, st.TeleOnlyOf)
		}
	}
	if err := repo.Committed(files...); err != nil {
		return nil, bad("not committed: %v", err)
	}
	if p.Committer, err = repo.FirstAuthor(paths.Unseal); err != nil {
		return nil, bad("provenance: %v", err)
	}
	if p.Committer != u.ApprovedBy {
		return nil, bad("committed by %q, approved by %q", p.Committer, u.ApprovedBy)
	}
	p.authorized = true
	return p, nil
}

// Run is a canonical run's manifest: what was run, under which approved
// artifacts, and the digest of every record and of the report.
type Run struct {
	Schema           string      `json:"schema"`
	Holdout          string      `json:"holdout"`
	UnsealSHA256     string      `json:"unsealSHA256"`
	PolicyDigest     string      `json:"policyDigest"`
	StrategiesSHA256 string      `json:"strategiesSHA256"`
	Engine           string      `json:"engine"`
	Primary          string      `json:"primary"`
	ApprovedBy       string      `json:"approvedBy"`
	HorizonEndSec    int64       `json:"horizonEndSec"`
	Records          []RunRecord `json:"records"` // every plan × every holdout seed
	ReportSHA256     string      `json:"reportSHA256"`
}

// RunRecord is one record's identity and digest (its canonical JSON).
type RunRecord struct {
	Plan   string `json:"plan"`
	Seed   int64  `json:"seed"`
	SHA256 string `json:"sha256"`
}

// Run outcomes.
const (
	RunWritten    = "written"    // the one-time run: outputs committed to the run directory
	RunReproduced = "reproduced" // a rerun gave the same bytes
)

// ErrNotReproduced is a rerun whose outputs differ from the first run's:
// the holdout is then exposed (README §9), and nothing is written.
var ErrNotReproduced = errors.New("the rerun does not reproduce the holdout run")

// NewRun builds the run manifest of an authorized protocol's records and
// report, checking the matrix is complete: every plan of the set on every
// holdout seed, once.
func (p *Protocol) NewRun(records map[string][]Record, horizon int64, report []byte) (*Run, error) {
	if !p.Authorized() {
		return nil, ErrSealed
	}
	r := &Run{Schema: RunSchema, Holdout: p.Holdout.ID, UnsealSHA256: p.UnsealSHA256, PolicyDigest: p.PolicyDigest,
		StrategiesSHA256: p.Unseal.StrategiesSHA256, Engine: p.Unseal.Engine, Primary: p.Unseal.Primary, ApprovedBy: p.Unseal.ApprovedBy,
		HorizonEndSec: horizon, Records: []RunRecord{}, ReportSHA256: sha(report)}
	if len(records) != len(p.Set.Plans) {
		return nil, fmt.Errorf("records of %d plans, the strategy set has %d", len(records), len(p.Set.Plans))
	}
	for _, name := range p.Plans() {
		recs := records[name]
		if len(recs) != len(p.Seeds) {
			return nil, fmt.Errorf("plan %s: %d records for %d holdout seeds", name, len(recs), len(p.Seeds))
		}
		for i, seed := range p.Seeds {
			x := &recs[i]
			if x.Seed != seed || x.Plan != name || x.PlanSHA256 != p.StrategySHA[name] {
				return nil, fmt.Errorf("plan %s record %d: seed, plan or strategy digest is not the protocol's", name, i)
			}
			raw, err := json.Marshal(x)
			if err != nil {
				return nil, err
			}
			r.Records = append(r.Records, RunRecord{Plan: name, Seed: seed, SHA256: sha(raw)})
		}
	}
	return r, nil
}

// CommitRun writes a first run's manifest and report to dir (run.json,
// report.json), which must not exist yet. If they exist, the run must
// reproduce both byte for byte: the outcome is RunReproduced, or
// ErrNotReproduced with nothing written. A first run is never
// overwritten.
func CommitRun(dir string, run *Run, report []byte) (string, error) {
	raw, err := json.MarshalIndent(run, "", " ")
	if err != nil {
		return "", err
	}
	raw = append(raw, '\n')
	runPath, repPath := filepath.Join(dir, "run.json"), filepath.Join(dir, "report.json")
	old, err := os.ReadFile(runPath)
	switch {
	case err == nil:
		oldRep, rerr := os.ReadFile(repPath)
		if rerr != nil || !bytes.Equal(old, raw) || !bytes.Equal(oldRep, report) {
			return "", fmt.Errorf("%w: %s", ErrNotReproduced, dir)
		}
		return RunReproduced, nil
	case !errors.Is(err, os.ErrNotExist):
		return "", err
	}
	if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
		return "", fmt.Errorf("%s holds files but no run manifest: refusing to write over them", dir)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(repPath, report, 0o444); err != nil {
		return "", err
	}
	if err := os.WriteFile(runPath, raw, 0o444); err != nil {
		return "", err
	}
	return RunWritten, nil
}

// GitRepo is the repository at Root, through git.
type GitRepo struct{ Root string }

func (g GitRepo) git(args ...string) (string, error) {
	out, err := exec.Command("git", append([]string{"-C", g.Root}, args...)...).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// Committed requires every path tracked, with no staged or unstaged
// change against HEAD.
func (g GitRepo) Committed(paths ...string) error {
	for _, p := range paths {
		if _, err := g.git("ls-files", "--error-unmatch", "--", p); err != nil {
			return fmt.Errorf("%s is not tracked", p)
		}
	}
	if out, err := g.git(append([]string{"status", "--porcelain", "--"}, paths...)...); err != nil || out != "" {
		return fmt.Errorf("changed since the last commit: %s %v", out, err)
	}
	return nil
}

// FirstAuthor is the author of the commit that added path.
func (g GitRepo) FirstAuthor(path string) (string, error) {
	out, err := g.git("log", "--diff-filter=A", "--format=%an", "--", path)
	if err != nil || out == "" {
		return "", fmt.Errorf("no commit adds %s: %v", path, err)
	}
	lines := strings.Split(out, "\n")
	return lines[len(lines)-1], nil
}
