package alternatives

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/agent"
	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/session"
)

// The V0.13c branch-point inventory (docs/benchmark/v13c-dev/points): the
// V0.12b pool of 20 recorded staffing proposals with committed contexts,
// each opened on the real engine at its recorded state (the recording's
// replay transcript fed through the agent protocol up to the recorded
// digests, as the V0.12b driver does), and rule v2 and both arms' turns
// built for the recorded proposal. It reads no model response other than
// the recorded proposal itself. The pool, the categories and the selection
// rule dreamteam-v13c-select/1 are fixed in BRANCH_POINTS.md before any
// inference.
//
// It needs the V0.8c and V0.9c replays (as CI builds them):
// DREAMTEAM_V13C_REPLAYS=<dir> runs it; DREAMTEAM_V13C_WRITE=1 writes the
// files instead of comparing them.

const (
	pointsRule     = "dreamteam-v13c-select/1"
	poolManifest   = "../../docs/benchmark/v12b-dev/points/manifest.json"
	maxPerRecordng = 2
)

type poolEntry struct {
	ID               string   `json:"id"`
	Model            string   `json:"model"`
	Point            int      `json:"point"`
	Recording        string   `json:"recording"`
	Source           string   `json:"source"`
	SourceLineSha256 string   `json:"sourceLineSha256"`
	PriorExposure    []string `json:"priorExposure"`
	StakeCents       int64    `json:"stakeCents"`
	Tags             []string `json:"tags"`
	Validation       string   `json:"validation"`
}

type contextLine struct {
	ID       string          `json:"id"`
	Digests  session.Digests `json:"digests"`
	Proposal struct {
		Actions []api.Action `json:"actions"`
	} `json:"proposal"`
	Validation string `json:"validation"`
}

// InventoryRow is one pooled point on the engine.
type InventoryRow struct {
	ID            string       `json:"id"`
	Model         string       `json:"model"`
	Recording     string       `json:"recording"`
	Point         int          `json:"point"`
	PriorExposure []string     `json:"priorExposure"`
	At            string       `json:"at"`
	Proposal      []api.Action `json:"proposal"`   // the reply's staffing batch
	Operations    []api.Action `json:"operations"` // the reply's operations batch
	Valid         bool         `json:"valid"`
	Codes         []string     `json:"codes"`
	Applies       bool         `json:"ruleApplies"`
	Why           string       `json:"ruleWhy"`
	Alternatives  []string     `json:"alternatives"`
	Removed       []string     `json:"removed"`
	Omitted       []string     `json:"omitted"`
	Displaced     int          `json:"displaced"`
	SlotsClosed   *int         `json:"slotsClosed"`
	StakeCents    int64        `json:"stakeCents"`
	Tags          []string     `json:"tags"`
	StepDue       bool         `json:"stepDue"`
	StepWhy       string       `json:"stepWhy"`
	Category      string       `json:"category"`
	Treatment     string       `json:"treatmentMessageSha256"`
	Control       string       `json:"controlMessageSha256"`
}

// The categories of dreamteam-v13c-select/1, in order: a point takes the
// first that fits. Every category is decided by the engine at the recorded
// state, or by the V0.10c inventory's tags carried in the pool.
var pointCategories = []struct {
	Name string
	Fits func(r InventoryRow) bool
}{
	{"termination_keep", func(r InventoryRow) bool { return r.StepDue && hasType(r, api.ActTerminate) && r.Displaced == 0 }},
	{"termination_displacement", func(r InventoryRow) bool { return r.StepDue && hasType(r, api.ActTerminate) && r.Displaced > 0 }},
	{"staffing_adjustment_displacement", func(r InventoryRow) bool {
		return r.StepDue && r.Displaced > 0 && (hasType(r, api.ActSetRoster) || hasType(r, api.ActSetReadingAllocation))
	}},
	{"retained_obligation", func(r InventoryRow) bool {
		return r.StepDue && (hasType(r, api.ActContractTeleradiology) || hasType(r, api.ActHire))
	}},
	{"credit_supported_not_due", func(r InventoryRow) bool { return r.Valid && !r.StepDue && slices.Contains(r.Tags, "credit_supported") }},
	{"valid_expensive_not_due", func(r InventoryRow) bool { return r.Valid && !r.StepDue && r.StakeCents >= 1000000 }},
	{"nonqualifying_control", func(r InventoryRow) bool { return r.Valid && !r.StepDue }},
	// Refused, the rule applying to the proposal as made: a correction that
	// keeps its irreversible action would make the step due.
	{"refused_qualifying", func(r InventoryRow) bool { return !r.Valid && r.Applies }},
	{"refused_other", func(r InventoryRow) bool { return !r.Valid }},
}

func hasType(r InventoryRow, t string) bool {
	return slices.ContainsFunc(r.Proposal, func(a api.Action) bool { return a.Type == t })
}

func TestInventoryV13c(t *testing.T) {
	replays := os.Getenv("DREAMTEAM_V13C_REPLAYS")
	if replays == "" {
		t.Skip("DREAMTEAM_V13C_REPLAYS is not set (the V0.8c and V0.9c replays)")
	}
	var man struct {
		Pool []poolEntry `json:"pool"`
	}
	readJSON(t, poolManifest, &man)
	if len(man.Pool) != 20 {
		t.Fatalf("%d pooled points", len(man.Pool))
	}
	runs := replayRuns(t, replays)
	var rows []InventoryRow
	for _, p := range man.Pool {
		cl := contextOf(t, p)
		dir, ok := runs[filepath.Base(p.Recording)]
		if !ok {
			t.Fatalf("%s: no replay of %s", p.ID, p.Recording)
		}
		s := atPoint(t, dir, cl.Digests)
		rows = append(rows, inventoryRow(t, s, p, cl))
	}
	for k := range rows {
		for _, c := range pointCategories {
			if c.Fits(rows[k]) {
				rows[k].Category = c.Name
				break
			}
		}
	}
	sel, unavailable := selectPoints(rows)
	evidenceV13c(t, "points/inventory.json", encode(t, struct {
		Schema string         `json:"schema"`
		Pool   string         `json:"pool"`
		Rule   string         `json:"rule"`
		Points []InventoryRow `json:"points"`
	}{"dreamteam.v13c-points-inventory/v1", "docs/benchmark/v12b-dev/points/manifest.json (pool)", SelectionRuleV2, rows}))
	cats := []string{}
	for _, c := range pointCategories {
		cats = append(cats, c.Name)
	}
	evidenceV13c(t, "points/manifest.json", encode(t, struct {
		Schema       string              `json:"schema"`
		Rule         string              `json:"rule"`
		Categories   []string            `json:"categories"`
		MaxPerRecord int                 `json:"maxPerRecording"`
		Selected     []string            `json:"selected"`
		ByCategory   map[string][]string `json:"byCategory"`
		Unavailable  []string            `json:"unavailable"`
	}{"dreamteam.v13c-points/v1", pointsRule, cats, maxPerRecordng, sel.ids, sel.byCat, unavailable}))
}

type selection struct {
	ids   []string
	byCat map[string][]string
}

// selectPoints is dreamteam-v13c-select/1: every point whose step is due
// (the only points where the arms differ); one point of each valid not-due
// category, as controls; two refused points the rule applies to, one per
// model; none of the other refused points. Each point once, at most two
// per recording, in each category in the order of SHA-256(rule|id). It
// reads no model response.
func selectPoints(rows []InventoryRow) (selection, []string) {
	out := selection{ids: []string{}, byCat: map[string][]string{}}
	perRec := map[string]int{}
	var unavailable []string
	for _, c := range pointCategories {
		var cands []InventoryRow
		for _, r := range rows {
			if r.Category == c.Name {
				cands = append(cands, r)
			}
		}
		sort.Slice(cands, func(i, j int) bool { return orderKey(cands[i].ID) < orderKey(cands[j].ID) })
		limit, perModel := len(cands), len(cands) // due categories: every point
		switch c.Name {
		case "credit_supported_not_due", "valid_expensive_not_due", "nonqualifying_control":
			limit = 1
		case "refused_qualifying":
			limit, perModel = 2, 1
		case "refused_other":
			limit = 0
		}
		got, models := []string{}, map[string]int{}
		for _, r := range cands {
			if len(got) >= limit || perRec[r.Recording] >= maxPerRecordng || models[r.Model] >= perModel {
				continue
			}
			perRec[r.Recording]++
			models[r.Model]++
			got = append(got, r.ID)
		}
		if len(got) == 0 && c.Name != "refused_other" {
			unavailable = append(unavailable, c.Name)
		}
		out.byCat[c.Name] = got
		out.ids = append(out.ids, got...)
	}
	return out, append([]string{}, unavailable...)
}

func orderKey(id string) string {
	h := sha256.Sum256([]byte(pointsRule + "|" + id))
	return hex.EncodeToString(h[:])
}

func inventoryRow(t *testing.T, s *session.Session, p poolEntry, cl contextLine) InventoryRow {
	t.Helper()
	// The reply's batches as the agents make them (llm_agent3.split):
	// its operations actions, then its staffing actions. The comparison is
	// of the staffing batch; the operations batch is validated and carried
	// as the workflow carries it, the same in both arms.
	var ops, staff []api.Action
	for _, a := range cl.Proposal.Actions {
		if isStaffing(a.Type) {
			staff = append(staff, a)
		} else {
			ops = append(ops, a)
		}
	}
	prop := canon(staff)
	r := InventoryRow{ID: p.ID, Model: p.Model, Recording: p.Recording, Point: p.Point, PriorExposure: p.PriorExposure, At: s.Observe(0).Time,
		Proposal: prop, Operations: canon(ops), Codes: []string{}, Removed: []string{}, Omitted: []string{}, Alternatives: []string{},
		StakeCents: p.StakeCents, Tags: p.Tags}
	r.Valid = true
	for _, b := range [][]api.Action{canon(ops), prop} {
		if len(b) == 0 {
			continue
		}
		for _, v := range s.ValidateBatch(b...) {
			r.Valid = r.Valid && v.OK
			if !v.OK {
				r.Codes = append(r.Codes, v.Action.Type+":"+v.Code)
			}
		}
	}
	// The recorded validation: the context line's, or (V0.10b's lines have
	// none) the pool's, from the V0.10c inventory.
	rec := cl.Validation
	if rec == "" {
		rec = p.Validation
	}
	if want := rec == "accepted"; want != r.Valid || p.Validation != rec {
		t.Fatalf("%s: validation %v %v, recorded %s", p.ID, r.Valid, r.Codes, rec)
	}
	c, tr, sel, err := Steps(s, prop)
	if err != nil {
		t.Fatalf("%s: %v", p.ID, err)
	}
	r.Applies, r.Why = sel.Applies, sel.Why
	r.Alternatives = append(r.Alternatives, ids(sel)...)
	r.Removed = append(r.Removed, removedIDs(sel)...)
	for _, o := range sel.Omitted {
		r.Omitted = append(r.Omitted, o.ID)
	}
	if r.Valid && slices.ContainsFunc(prop, func(a api.Action) bool { return isStaffing(a.Type) }) {
		r.Displaced = len(s.PreviewStaffing(prop...).Displacements)
	}
	if r.Valid && !slices.ContainsFunc(prop, func(a api.Action) bool { return a.Type != api.ActTerminate }) {
		n := s.PreviewSlotsClosed(prop...).Total
		r.SlotsClosed = &n
	}
	r.StepDue, r.StepWhy = c.Due, c.Why
	if c.Due != tr.Due || c.Why != tr.Why {
		t.Fatalf("%s: the arms disagree on the step", p.ID)
	}
	r.Treatment, r.Control = tr.MessageSha256, c.MessageSha256
	return r
}

func readJSON(t *testing.T, path string, v any) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, v); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
}

// contextOf is the pooled point's committed context line, found by its
// SHA-256 (the line without its newline).
func contextOf(t *testing.T, p poolEntry) contextLine {
	t.Helper()
	f, err := os.Open(filepath.Join("../..", p.Source))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<26)
	for sc.Scan() {
		line := sc.Bytes()
		h := sha256.Sum256(line)
		if hex.EncodeToString(h[:]) != p.SourceLineSha256 {
			continue
		}
		var cl contextLine
		if err := json.Unmarshal(line, &cl); err != nil {
			t.Fatal(err)
		}
		if cl.ID != p.ID {
			t.Fatalf("%s: the context line is %s", p.ID, cl.ID)
		}
		return cl
	}
	t.Fatalf("%s: no context line %s in %s", p.ID, p.SourceLineSha256, p.Source)
	return contextLine{}
}

// replayRuns maps a recording's base name to its replay run directory,
// from each run's own record of what it replayed (as
// decision_experiment.replay_runs).
func replayRuns(t *testing.T, root string) map[string]string {
	t.Helper()
	evs, err := filepath.Glob(filepath.Join(root, "*", "runs", "*", "policy-evidence", "decisions.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(evs)
	out := map[string]string{}
	for _, ev := range evs {
		raw, err := os.ReadFile(ev)
		if err != nil {
			t.Fatal(err)
		}
		first, _, _ := bytes.Cut(raw, []byte("\n"))
		var d struct {
			Replayed string `json:"replayed"`
		}
		if len(bytes.TrimSpace(first)) > 0 && json.Unmarshal(first, &d) == nil && d.Replayed != "" {
			out[d.Replayed] = filepath.Dir(filepath.Dir(ev))
		}
	}
	return out
}

// atPoint opens a session on the run's scenario and seed and feeds the
// recorded requests through the agent protocol, every answer required to
// be the recorded one (hello: its clock and scenario fingerprint), until
// a digests answer equals the point's digests.
func atPoint(t *testing.T, dir string, want session.Digests) *session.Session {
	t.Helper()
	var run struct {
		Scenario struct {
			Ref string `json:"ref"`
		} `json:"scenario"`
		Seed int64 `json:"seed"`
	}
	readJSON(t, filepath.Join(dir, "run.json"), &run)
	ref := run.Scenario.Ref
	if _, err := os.Stat(filepath.Join("../..", ref)); err == nil {
		ref = filepath.Join("../..", ref)
	}
	s, err := session.Open(ref, run.Seed)
	if err != nil {
		t.Fatal(err)
	}
	h := agent.New(s)
	raw, err := os.ReadFile(filepath.Join(dir, "transcript.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	var req []byte
	n := 0
	for _, line := range bytes.Split(raw, []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var probe map[string]json.RawMessage
		if err := json.Unmarshal(line, &probe); err != nil {
			t.Fatal(err)
		}
		if _, isReq := probe["op"]; isReq {
			req = slices.Clone(line)
			continue
		}
		if req == nil {
			continue
		}
		n++
		got := h.Handle(req)
		var g, w map[string]any
		_ = json.Unmarshal(got, &g)
		_ = json.Unmarshal(line, &w)
		var op struct {
			Op string `json:"op"`
		}
		_ = json.Unmarshal(req, &op)
		if op.Op == "hello" {
			gr, _ := g["result"].(map[string]any)
			wr, _ := w["result"].(map[string]any)
			gs, _ := gr["scenario"].(map[string]any)
			ws, _ := wr["scenario"].(map[string]any)
			if !reflect.DeepEqual(gr["clock"], wr["clock"]) || !reflect.DeepEqual(gs["fingerprint"], ws["fingerprint"]) {
				t.Fatalf("%s: request %d (hello) answered differently", dir, n)
			}
		} else if !reflect.DeepEqual(g, w) {
			t.Fatalf("%s: request %d (%s) answered differently", dir, n, op.Op)
		}
		req = nil
		if op.Op == "digests" && s.Digests() == want {
			return s
		}
	}
	t.Fatalf("%s: the point %s was not reached", dir, fmt.Sprint(want.State))
	return nil
}
