package session

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/georgezero/dreamteam-go/internal/api"
)

// Session records, schema v1 (V0.1 to V0.2 Slice 1), read-only since
// V0.2 Slice 2 (V0.2_DESIGN §12). The v1 DTOs below are the v1 format
// exactly — the V0.1 action fields, nothing else — and are decoded
// strictly: a v1 file with a V0.2 field is not a v1 file. A v1 record is
// replayed through the legacy path (its batches hold only V0.1 actions,
// applied one by one as they always were) and, when continued, embedded
// verbatim as the prefix of a new v2 record; it is never rewritten.

// RecordSchemaV1 identifies the v1 session format.
const RecordSchemaV1 = "dreamteam.session/v1"

type recordV1 struct {
	Schema    string         `json:"schema"`
	Engine    string         `json:"engine"`
	Scenario  RecordScenario `json:"scenario"`
	Seed      int64          `json:"seed"`
	EpochDate string         `json:"epochDate"`
	StartSec  int64          `json:"startSec"`
	Start     string         `json:"start"`
	UntilSec  int64          `json:"untilSec"`
	Until     string         `json:"until"`
	Batches   []batchV1      `json:"batches"`
	Digests   Digests        `json:"digests"`
	Summary   Summary        `json:"summary"`
	Checksum  string         `json:"checksum"`
}

type batchV1 struct {
	Batch   int64      `json:"batch"`
	AtSec   int64      `json:"atSec"`
	Time    string     `json:"time"`
	Actions []actionV1 `json:"actions"`
	Results []resultV1 `json:"results"`
}

type actionV1 struct {
	Type     string `json:"type"`
	Employee string `json:"employee,omitempty"`
	Minutes  int64  `json:"minutes,omitempty"`
	Date     string `json:"date,omitempty"`
	Policy   string `json:"policy,omitempty"`
	Cutoff   string `json:"cutoff,omitempty"`
}

type resultV1 struct {
	ID      string   `json:"id"`
	Batch   int64    `json:"batch"`
	Time    string   `json:"time"`
	Action  actionV1 `json:"action"`
	OK      bool     `json:"ok"`
	Code    string   `json:"code"`
	Message string   `json:"message"`
}

func (r *recordV1) checksum() string {
	c := *r
	c.Checksum = ""
	raw, err := json.Marshal(&c)
	if err != nil {
		panic(err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func (a actionV1) action() api.Action {
	return api.Action{Type: a.Type, Employee: a.Employee, Minutes: a.Minutes, Date: a.Date, Policy: a.Policy, Cutoff: a.Cutoff}
}

// decodeV1 parses and checks a v1 record strictly and returns it as a
// Record (schema v1) holding its canonical v1 encoding.
func decodeV1(raw []byte) (*Record, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var v recordV1
	if err := dec.Decode(&v); err != nil {
		return nil, fmt.Errorf("%w: cannot parse as %s (%v; truncated?)", ErrCorrupt, RecordSchemaV1, err)
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return nil, fmt.Errorf("%w: data after the record", ErrCorrupt)
	}
	if v.Schema != RecordSchemaV1 {
		return nil, fmt.Errorf("%w: schema %q is not %q", ErrIncompatible, v.Schema, RecordSchemaV1)
	}
	if want := v.checksum(); v.Checksum != want {
		return nil, fmt.Errorf("%w: checksum %.12s… does not match the content (%.12s…)", ErrCorrupt, v.Checksum, want)
	}
	r := &Record{Schema: v.Schema, Engine: v.Engine, Scenario: v.Scenario, Seed: v.Seed, EpochDate: v.EpochDate,
		StartSec: v.StartSec, Start: v.Start, UntilSec: v.UntilSec, Until: v.Until, Digests: v.Digests,
		Summary: v.Summary, Checksum: v.Checksum}
	for _, b := range v.Batches {
		nb := Batch{Batch: b.Batch, AtSec: b.AtSec, Time: b.Time}
		for _, a := range b.Actions {
			nb.Actions = append(nb.Actions, a.action())
		}
		for _, x := range b.Results {
			nb.Results = append(nb.Results, api.ActionResult{ID: x.ID, Batch: x.Batch, Time: x.Time, Action: x.Action.action(),
				OK: x.OK, Code: x.Code, Message: x.Message})
		}
		r.Batches = append(r.Batches, nb)
	}
	canon, err := json.Marshal(&v)
	if err != nil {
		return nil, err
	}
	r.v1 = canon
	return r, nil
}

// encodeV1 writes a record holding only V0.1 actions in the v1 format
// (the inverse of decodeV1; for tests and fixtures). It refuses a record
// with a V0.2 action or result field.
func encodeV1(r *Record) ([]byte, error) {
	v := recordV1{Schema: RecordSchemaV1, Engine: r.Engine, Scenario: r.Scenario, Seed: r.Seed, EpochDate: r.EpochDate,
		StartSec: r.StartSec, Start: r.Start, UntilSec: r.UntilSec, Until: r.Until, Digests: r.Digests, Summary: r.Summary}
	if r.Legacy != nil {
		return nil, fmt.Errorf("a continued record has a v1 prefix already")
	}
	conv := func(a api.Action) (actionV1, error) {
		x := actionV1{Type: a.Type, Employee: a.Employee, Minutes: a.Minutes, Date: a.Date, Policy: a.Policy, Cutoff: a.Cutoff}
		if x.action().Type != a.Type || !legacyOnly(a) {
			return x, fmt.Errorf("action %q has V0.2 fields", a.Type)
		}
		return x, nil
	}
	for _, b := range r.Batches {
		nb := batchV1{Batch: b.Batch, AtSec: b.AtSec, Time: b.Time}
		for _, a := range b.Actions {
			x, err := conv(a)
			if err != nil {
				return nil, err
			}
			nb.Actions = append(nb.Actions, x)
		}
		for _, res := range b.Results {
			x, err := conv(res.Action)
			if err != nil || res.EmployeeID != "" || res.EffectiveAt != "" {
				return nil, fmt.Errorf("result %s has V0.2 fields", res.ID)
			}
			nb.Results = append(nb.Results, resultV1{ID: res.ID, Batch: res.Batch, Time: res.Time, Action: x, OK: res.OK,
				Code: res.Code, Message: res.Message})
		}
		v.Batches = append(v.Batches, nb)
	}
	v.Checksum = v.checksum()
	raw, err := json.MarshalIndent(&v, "", " ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

// legacyOnly reports whether a carries no V0.2 field. (Its type may be
// any string: a V0.1 build recorded "hire" as an unknown action.)
func legacyOnly(a api.Action) bool {
	return a.RequestID == "" && a.Role == "" && a.Skills == nil && a.RateCents == 0 && a.ContractedWeeklyMinutes == 0 &&
		a.Roster == nil && a.StartDate == "" && a.LastDay == "" && a.Op == "" && a.Effective == "" && a.SiteAllocationBP == 0 &&
		a.Plan == "" && a.Windows == nil && a.Scope == "" && a.Contract == "" && a.EndDate == ""
}

// Engine compatibility registry (V0.2_DESIGN §12): the builds whose
// records this build replays with their outcomes verified but whose state
// encoding differs, so their state and run digests cannot be compared.
// Only these named builds qualify; any other engine needs every digest to
// match (and the engine mismatch allowed).
var legacyEngines = []struct {
	prefix, exact string // a version prefix, or one exact version
	state         string // its state digest encoding
}{
	{prefix: "0.1.", state: "dreamteam.state/v1"},         // V0.1
	{prefix: "0.2.0-dev.0.", state: "dreamteam.state/v1"}, // V0.2 Step 0 builds
	{exact: "0.2.0-dev.1", state: "dreamteam.state/v2"},   // V0.2 Slice 1
}

// Earlier V0.2 builds that wrote v2 records, with the V0.2 action set they
// knew (V0.2_DESIGN §12). Their state encoding is this build's, so their
// records replay with every digest verified; their batches apply with
// their own action set (a later action type was an unknown V0.1-path
// action then), and a session continuing one embeds it as its prefix.
var earlierV2Engines = map[string]string{
	"0.2.0-dev.2":  batchModeSlice2, // V0.2 Slice 2: hire, terminate
	"0.2.0-dev.3":  batchModeSlice3, // V0.2 Slice 3: hire, terminate, set_roster
	"0.2.0-dev.4":  batchModeSlice3, // the dynamic staffing benchmark: Slice 3's action set
	"0.2.0-dev.5":  batchModeSlice4, // V0.2 Slice 4: this action set, dev.5's reading and allocation rules
	"0.2.0-dev.6":  batchModeDev6,   // pre-Slice 5 remediation: Slice 4's action set, this build's rules
	"0.2.0-dev.7":  batchModeDev7,   // V0.2 Slice 5A (remote reading engine): Slice 4's action set, this build's rules
	"0.2.0-dev.8":  batchModeDev8,   // V0.2 Slice 5B (contract_teleradiology): this build's action set and rules
	"0.2.0-dev.9":  batchModeDev9,   // V0.2 Slice 5C.0 (frozen evaluation): this build's action set and rules
	"0.2.0-dev.10": batchModeDev10,  // V0.2 Slice 5C.0a (evaluation policy v0.2-eval-2): this build's action set and rules
	"0.2.0-dev.11": batchModeDev11,  // V0.2 Slice 5C.0b (canonical holdout runner): this build's action set and rules
}

// legacyState returns the state encoding of a registered legacy engine.
func legacyState(engine string) (string, bool) {
	for _, e := range legacyEngines {
		if (e.exact != "" && engine == e.exact) || (e.prefix != "" && strings.HasPrefix(engine, e.prefix)) {
			return e.state, true
		}
	}
	return "", false
}
