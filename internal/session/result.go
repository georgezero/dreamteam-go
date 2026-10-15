package session

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/version"
)

// Run results (ROADMAP step 15): a run's canonical outcome, for golden
// references, regression checks and, later, benchmark infrastructure.
//
// A result is built from what the practice can observe — daily summaries,
// the evaluation window, balances, obligations, alerts, the action record —
// plus the verification digests. No hidden patient, payer or future-event
// data: the private state appears only as its digest. Every number is an
// integer (cents, seconds, counts; ratios as numerator and denominator);
// every array has a defined order; nothing depends on the wall clock, the
// machine or the locale. Encoding is json.MarshalIndent of these structs
// (fixed field order, one-space indent, trailing newline): repeated runs
// are byte-identical.

// ResultSchema identifies the result format.
const ResultSchema = "dreamteam.result/v1"

// Result is one run's canonical outcome.
type Result struct {
	Schema   string         `json:"schema"`
	Engine   string         `json:"engine"`
	Scenario ResultScenario `json:"scenario"`
	Seed     int64          `json:"seed"`
	// Times: the scenario epoch (day 0), the takeover, and the end of the
	// run, as simulation seconds with the practice-local ISO time.
	EpochDate string        `json:"epochDate"`
	Takeover  TimePoint     `json:"takeover"`
	End       TimePoint     `json:"end"`
	PlayDays  int64         `json:"playDays"` // whole days from takeover to end
	Actions   ResultActions `json:"actions"`
	Digests   Digests       `json:"digests"`
	// Calendar reporting: completed calendar days from the takeover day
	// (which includes its hours before the takeover), midnight to midnight.
	Calendar []ResultDay `json:"calendar"`
	// The evaluation window: from the exact takeover to the end of the run
	// (or the configured endpoint, if earlier).
	Window ResultWindow `json:"window"`
	// Point-in-time balances and obligations at the end of the run.
	Balances    ResultBalances    `json:"balances"`
	Obligations ResultObligations `json:"obligations"`
	Alerts      ResultAlerts      `json:"alerts"`
	// Replay is present when the run was verified by replaying its record.
	Replay *ResultReplay `json:"replay,omitempty"`
}

// ResultScenario identifies the scenario.
type ResultScenario struct {
	ID          string `json:"id"`
	Fingerprint string `json:"fingerprint"`
}

// TimePoint is a simulation time.
type TimePoint struct {
	Sec  int64  `json:"sec"`
	Time string `json:"time"`
}

// ResultActions is the action history: counts, its digest, and each
// action's result in order (rejections and no-ops included).
type ResultActions struct {
	Attempted int            `json:"attempted"`
	Accepted  int            `json:"accepted"`
	Rejected  int            `json:"rejected"`
	Digest    string         `json:"digest"`
	Results   []ResultAction `json:"results"`
}

// ResultAction is one action's outcome.
type ResultAction struct {
	ID    string     `json:"id"`
	Batch int64      `json:"batch"`
	Time  string     `json:"time"`
	Act   api.Action `json:"action"`
	OK    bool       `json:"ok"`
	Code  string     `json:"code"`
}

// ResultDay is one completed calendar day's headline figures (flows).
type ResultDay struct {
	Date               string `json:"date"`
	Weekday            string `json:"weekday"`
	Takeover           bool   `json:"takeover,omitempty"` // includes hours before the takeover
	Orders             int    `json:"orders"`
	Lost               int    `json:"lost"`
	WalkIns            int    `json:"walkIns"`
	NoShows            int    `json:"noShows"`
	Exams              int    `json:"exams"`
	ReportsFinal       int    `json:"reportsFinal"`
	UnreadAtClose      int    `json:"unreadAtClose"`
	WaitP50Sec         int64  `json:"waitP50Sec"`
	OvertimeSec        int64  `json:"overtimeSec"`
	NetRevenueCents    int64  `json:"netRevenueCents"`
	ExpenseCents       int64  `json:"expenseCents"`
	ProfitCents        int64  `json:"operatingProfitCents"`
	CollectionsCents   int64  `json:"collectionsCents"`
	DisbursementsCents int64  `json:"disbursementsCents"`
	CashEndCents       int64  `json:"cashEndCents"` // a balance: end of day
	Alerts             int    `json:"alertsRaised"`
}

// ResultWindow is the evaluation window's outcome.
type ResultWindow struct {
	From     TimePoint         `json:"from"`
	To       TimePoint         `json:"to"`
	Complete bool              `json:"complete"` // the configured endpoint was reached
	Baseline string            `json:"baselineDigest"`
	Ops      api.PeriodMetrics `json:"ops"`
	Money    api.WindowMoney   `json:"money"`
}

// ResultBalances are the balances at the end (point in time).
type ResultBalances struct {
	CashCents           int64 `json:"cashCents"`
	ReceivablesCents    int64 `json:"receivablesCents"`
	PayablesCents       int64 `json:"payablesCents"`
	AccruedPayrollCents int64 `json:"accruedPayrollCents"`
	EquityCents         int64 `json:"equityCents"`
	MinCashCents        int64 `json:"minCashCents"` // lowest since day 1
	DaysBelowZero       int64 `json:"daysBelowZero"`
}

// ResultObligations is what is outstanding at the end.
type ResultObligations struct {
	NextPayroll        api.Obligation    `json:"nextPayroll"`
	NextPayables       api.Obligation    `json:"nextPayables"`
	OpenClaims         int               `json:"openClaims"`
	ClaimsAging        []api.AgingBucket `json:"claimsAging"`
	Unread             int               `json:"unread"`
	Reading            int               `json:"reading"`
	RecallList         int               `json:"recallList"`
	BookedAhead        int               `json:"bookedAhead"` // appointments from tomorrow within the horizon
	Policies           api.Policies      `json:"policies"`
	ApprovedExtensions []api.Extension   `json:"approvedExtensions"`
}

// ResultAlerts summarizes alerts.
type ResultAlerts struct {
	RaisedWarn  int `json:"raisedWarn"`  // in the window
	RaisedAlert int `json:"raisedAlert"` // in the window
	ActiveAtEnd int `json:"activeAtEnd"`
}

// ResultReplay is the replay verification of the run's own record.
type ResultReplay struct {
	Verified bool   `json:"verified"`
	Batches  int    `json:"batches"`
	Run      string `json:"runDigest"`
}

// Result builds the run's canonical outcome now. With verify, the run's
// own record is replayed from scratch and the outcome included.
func (s *Session) Result(verify bool) (*Result, error) {
	if s.src == nil {
		return nil, ErrUnsavable
	}
	cal := s.Calendar()
	o := s.Observe(0)
	ev := s.Evaluation()
	start, now := s.w.StartTime(), s.Now()
	r := &Result{Schema: ResultSchema, Engine: version.Number, Seed: s.seed,
		Scenario:  ResultScenario{ID: s.src.ID, Fingerprint: s.fp},
		EpochDate: cal.Date(0).String(),
		Takeover:  TimePoint{int64(start), cal.ISO(start)}, End: TimePoint{int64(now), cal.ISO(now)},
		PlayDays: int64(now.Sub(start) / 86400), Digests: s.Digests()}

	log := s.w.ActionLog()
	r.Actions = ResultActions{Attempted: len(log), Digest: r.Digests.Actions, Results: []ResultAction{}}
	for _, a := range log {
		if a.OK {
			r.Actions.Accepted++
		} else {
			r.Actions.Rejected++
		}
		r.Actions.Results = append(r.Actions.Results, ResultAction{ID: a.ID, Batch: a.Batch, Time: a.Time, Act: a.Action, OK: a.OK, Code: a.Code})
	}

	startDay := cal.DayIndex(start)
	r.Calendar = []ResultDay{}
	for d := startDay; d < cal.DayIndex(now); d++ {
		x, ok := s.w.DaySummary(d)
		if !ok || !x.Complete {
			continue
		}
		r.Calendar = append(r.Calendar, ResultDay{Date: x.Date, Weekday: x.Weekday, Takeover: x.Takeover != "",
			Orders: x.Orders, Lost: x.Lost, WalkIns: x.WalkIns, NoShows: x.NoShows, Exams: x.Exams,
			ReportsFinal: x.ReportsFinal, UnreadAtClose: x.UnreadAtClose, WaitP50Sec: x.Ops.Waits.ToExam.P50Sec,
			OvertimeSec: x.Ops.Workforce.OvertimeSec, NetRevenueCents: x.NetRevenueCents, ExpenseCents: x.ExpenseCents,
			ProfitCents: x.OperatingProfitCents, CollectionsCents: x.CollectionsCents, DisbursementsCents: x.DisbursementsCents,
			CashEndCents: x.CashEndCents, Alerts: x.Ops.Alerts.Warn + x.Ops.Alerts.Alert})
	}

	ops := ev.Ops
	ops.Alerts.Top = nil // texts are in the log; counts are the outcome
	r.Window = ResultWindow{From: TimePoint{int64(start), ev.Baseline.Time}, To: TimePoint{}, Complete: ev.Ended,
		Baseline: ev.Baseline.Digest, Ops: ops, Money: ev.Money}
	to, err := cal.ParseISO(ev.AsOf)
	if err != nil {
		return nil, err
	}
	r.Window.To = TimePoint{int64(to), ev.AsOf}

	f := o.Finance
	r.Balances = ResultBalances{CashCents: f.CashCents, ReceivablesCents: f.ARCents, PayablesCents: f.APCents,
		AccruedPayrollCents: f.AccruedPayrollCents, EquityCents: f.BalanceSheet.EquityCents,
		MinCashCents: f.Financing.MinCashCents, DaysBelowZero: f.Financing.DaysBelowZero}
	ob := ResultObligations{NextPayroll: f.NextPayroll, NextPayables: f.NextPayables, OpenClaims: f.Claims.Open,
		ClaimsAging: f.Claims.Aging, Unread: o.Worklist.Unread, Reading: o.Worklist.Reading,
		RecallList: o.Demand.RecallList, Policies: o.Policies, ApprovedExtensions: o.Extensions}
	if ob.ClaimsAging == nil {
		ob.ClaimsAging = []api.AgingBucket{}
	}
	if ob.ApprovedExtensions == nil {
		ob.ApprovedExtensions = []api.Extension{}
	}
	for off := int64(1); off <= o.Practice.BookingHorizonDays; off++ {
		for _, rs := range s.Schedule(off).Rooms {
			for _, sl := range rs.Slots {
				if sl.Status == api.SlotBooked {
					ob.BookedAhead++
				}
			}
		}
	}
	r.Obligations = ob
	r.Alerts = ResultAlerts{RaisedWarn: ev.Ops.Alerts.Warn, RaisedAlert: ev.Ops.Alerts.Alert, ActiveAtEnd: len(o.Alerts.Active)}

	if verify {
		rep, err := s.Verify()
		r.Replay = &ResultReplay{Verified: err == nil}
		if rep != nil {
			r.Replay.Batches, r.Replay.Run = rep.Batches, rep.Replayed.Run
		}
	}
	return r, nil
}

// Encode is the result's canonical file content.
func (r *Result) Encode() ([]byte, error) {
	raw, err := json.MarshalIndent(r, "", " ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

// Result comparison errors.
var (
	ErrGoldenFormat       = errors.New("golden file is malformed")
	ErrGoldenIncompatible = errors.New("golden reference is not comparable")
	ErrGoldenMismatch     = errors.New("run differs from the golden reference")
)

// Comparison is the outcome of comparing a run with a golden reference.
// Compatibility (schema, engine, scenario, seed) is reported apart from
// verification (the outcomes).
type Comparison struct {
	Compatible   bool
	Incompatible []string // why not (schema, engine, scenario, seed, times)
	Match        bool
	Digests      []string // digests that differ
	Differences  []string // field paths with expected and actual, document order
	Total        int      // differing leaves (Differences is capped)
	// Migrated lists the fields a documented migration accounted for:
	// each held exactly its recorded old value in the reference and its
	// recorded new value in the run.
	Migrated []string
}

// MigrationSchema identifies a golden migration manifest.
const MigrationSchema = "dreamteam.golden-migration/v1"

// Migrations is a golden migration manifest (V0.2_DESIGN §11): historical
// golden references stay byte-for-byte as their engine wrote them, and
// the fields a documented state-schema change altered are listed per
// reference with their old and new values. Every other field still has
// to match exactly; a listed field must hold exactly its old value in the
// reference and its new value in the run, so the new state stays pinned.
type Migrations struct {
	Schema      string   `json:"schema"`
	Engine      string   `json:"engine"`      // the first build writing the new values
	StateSchema string   `json:"stateSchema"` // the state digest's encoding since
	Explanation []string `json:"explanation"`
	// References: golden file name → field path ("digests.state") → change.
	References map[string]map[string]FieldChange `json:"references"`
}

// FieldChange is one migrated field's value in the reference and now.
type FieldChange struct {
	Old string `json:"old"`
	New string `json:"new"`
}

// DecodeMigrations reads a migration manifest strictly.
func DecodeMigrations(raw []byte) (*Migrations, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var m Migrations
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("%w: migration manifest: %v", ErrGoldenFormat, err)
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return nil, fmt.Errorf("%w: data after the migration manifest", ErrGoldenFormat)
	}
	if m.Schema != MigrationSchema {
		return nil, fmt.Errorf("%w: migration schema %q, this build reads %q", ErrGoldenIncompatible, m.Schema, MigrationSchema)
	}
	return &m, nil
}

// String is a readable report.
func (c *Comparison) String() string {
	var b strings.Builder
	if c.Compatible {
		b.WriteString("compatible: yes\n")
	} else {
		fmt.Fprintf(&b, "compatible: NO — %s\n", strings.Join(c.Incompatible, "; "))
	}
	if len(c.Migrated) > 0 {
		fmt.Fprintf(&b, "migrated (documented schema change, old and new values as recorded): %s\n", strings.Join(c.Migrated, ", "))
	}
	if c.Match {
		b.WriteString("outcomes: identical to the golden reference\n")
		return b.String()
	}
	fmt.Fprintf(&b, "outcomes: %d difference(s)", c.Total)
	if len(c.Digests) > 0 {
		fmt.Fprintf(&b, "; digests differ: %s", strings.Join(c.Digests, ", "))
	}
	b.WriteString("\n")
	for i, d := range c.Differences {
		if i == 0 {
			fmt.Fprintf(&b, "  first: %s\n", d)
			continue
		}
		fmt.Fprintf(&b, "         %s\n", d)
	}
	return b.String()
}

const maxDifferences = 20

// DecodeGolden reads a golden result strictly: well-formed JSON, nothing
// after it, the known schema.
func DecodeGolden(raw []byte) (map[string]any, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, fmt.Errorf("%w: %v (truncated?)", ErrGoldenFormat, err)
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return nil, fmt.Errorf("%w: data after the result", ErrGoldenFormat)
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%w: not a JSON object", ErrGoldenFormat)
	}
	if m["schema"] != ResultSchema {
		return nil, fmt.Errorf("%w: schema %v, this build writes %q", ErrGoldenIncompatible, m["schema"], ResultSchema)
	}
	return m, nil
}

// Compare compares a run's result with a golden reference. Unknown or
// missing fields are differences; nothing is ignored. Incompatibility
// (another engine, scenario, seed or run length) is reported first and is
// an error unless allowEngine is set and only the engine differs — then the
// outcomes are still compared and the engine difference is reported.
func Compare(golden []byte, r *Result, allowEngine bool) (*Comparison, error) {
	return CompareMigrated(golden, r, allowEngine, nil)
}

// CompareMigrated is Compare under a reference's migrated fields (nil:
// none): see Migrations.
func CompareMigrated(golden []byte, r *Result, allowEngine bool, migrated map[string]FieldChange) (*Comparison, error) {
	want, err := DecodeGolden(golden)
	if err != nil {
		return nil, err
	}
	raw, err := r.Encode()
	if err != nil {
		return nil, err
	}
	got, _ := DecodeGolden(raw)
	c := &Comparison{Compatible: true}
	str := func(m map[string]any, path ...string) string {
		var v any = m
		for _, p := range path {
			mm, ok := v.(map[string]any)
			if !ok {
				return ""
			}
			v = mm[p]
		}
		return fmt.Sprint(v)
	}
	engineOnly := true
	for _, k := range [][]string{{"engine"}, {"scenario", "id"}, {"scenario", "fingerprint"}, {"seed"}, {"takeover", "sec"}, {"end", "sec"}} {
		if a, b := str(want, k...), str(got, k...); a != b {
			c.Compatible = false
			c.Incompatible = append(c.Incompatible, fmt.Sprintf("%s: golden %s, run %s", strings.Join(k, "."), a, b))
			if k[0] != "engine" {
				engineOnly = false
			}
		}
	}
	if !c.Compatible && !(allowEngine && engineOnly) {
		return c, ErrGoldenIncompatible
	}
	paths := make([]string, 0, len(migrated))
	for p := range migrated {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		ch := migrated[p]
		keys := strings.Split(p, ".")
		if a := str(want, keys...); a != ch.Old {
			diff(c, "."+p, ch.Old, a) // the reference is not the one migrated
		}
		if b := str(got, keys...); b != ch.New {
			diff(c, "."+p, ch.New, b) // the run does not reproduce the migrated value
		}
		for _, m := range []map[string]any{want, got} {
			deletePath(m, keys)
		}
		c.Migrated = append(c.Migrated, p)
	}
	for _, k := range []string{"log", "ledger", "actions", "state", "run"} {
		if str(want, "digests", k) != str(got, "digests", k) {
			c.Digests = append(c.Digests, k)
		}
	}
	diff(c, "", want, got)
	c.Match = c.Total == 0
	if !c.Match {
		return c, ErrGoldenMismatch
	}
	return c, nil
}

// deletePath removes a field from a decoded JSON object.
func deletePath(m map[string]any, keys []string) {
	for _, k := range keys[:len(keys)-1] {
		next, ok := m[k].(map[string]any)
		if !ok {
			return
		}
		m = next
	}
	delete(m, keys[len(keys)-1])
}

// diff walks expected and actual JSON in document order (object keys
// sorted), recording differing leaves; the engine is compatibility, not
// outcome.
func diff(c *Comparison, path string, want, got any) {
	if path == ".engine" {
		return
	}
	add := func(s string) {
		c.Total++
		if len(c.Differences) < maxDifferences {
			c.Differences = append(c.Differences, s)
		}
	}
	switch w := want.(type) {
	case map[string]any:
		g, ok := got.(map[string]any)
		if !ok {
			add(fmt.Sprintf("%s: expected an object, got %s", path, short(got)))
			return
		}
		keys := map[string]bool{}
		for k := range w {
			keys[k] = true
		}
		for k := range g {
			keys[k] = true
		}
		var ks []string
		for k := range keys {
			ks = append(ks, k)
		}
		sort.Strings(ks)
		for _, k := range ks {
			wv, inW := w[k]
			gv, inG := g[k]
			p := path + "." + k
			switch {
			case !inG:
				add(fmt.Sprintf("%s: missing from the run (golden %s)", p, short(wv)))
			case !inW:
				add(fmt.Sprintf("%s: unexpected field (run %s)", p, short(gv)))
			default:
				diff(c, p, wv, gv)
			}
		}
	case []any:
		g, ok := got.([]any)
		if !ok {
			add(fmt.Sprintf("%s: expected an array, got %s", path, short(got)))
			return
		}
		if len(w) != len(g) {
			add(fmt.Sprintf("%s: golden has %d elements, run %d", path, len(w), len(g)))
		}
		for i := 0; i < min(len(w), len(g)); i++ {
			diff(c, fmt.Sprintf("%s[%d]", path, i), w[i], g[i])
		}
	default:
		if fmt.Sprint(want) != fmt.Sprint(got) || fmt.Sprintf("%T", want) != fmt.Sprintf("%T", got) {
			add(fmt.Sprintf("%s: golden %s, run %s", path, short(want), short(got)))
		}
	}
}

func short(v any) string {
	raw, _ := json.Marshal(v)
	s := string(raw)
	if len(s) > 80 {
		s = s[:77] + "…"
	}
	return s
}
