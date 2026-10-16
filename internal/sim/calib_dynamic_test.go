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
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/scenario"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
	"github.com/georgezero/dreamteam-go/internal/version"
)

// The V0.2 dynamic staffing benchmark (docs/calibration/dynamic-staffing):
// fixed, hand-authored action plans — hire, terminate, set_roster and the
// V0.1 controls — applied through World.Apply to the sustainable starting
// practice, every seed the same actions at the same instants, from the
// takeover for 180 days. It measures what the plans cost and earn against
// the paired no-action run, day by day; it changes nothing in the model.
//
//	go test -tags calibration ./internal/sim -run 'DynamicStaffing$' -v -parallel 4 -timeout 300m
//	DT_DYN_SET=dev|eval   which seed set and plans (benchmark.json; default eval)
//	DT_DYN_PLANS=a,b      only these plans of the set
//	DT_DYN_SEEDS=901     other seeds (smoke runs; results are not the set's)
//	DT_DYN_OUT=dir        write results there instead
//	DT_DYN_CHECK=1        also check every engine invariant after each event (slow)
//
// Results: <dir>/results/<set>/<plan>.json, then report.py.

var dynDir = envOr("DT_DYN_DIR", "../../docs/calibration/dynamic-staffing")

type dynBenchmark struct {
	Scenario  string            `json:"scenario"` // a calibration manifest, relative to the benchmark directory
	Days      int               `json:"days"`
	Endpoints []int             `json:"endpoints"`
	Sets      map[string]dynSet `json:"sets"`
}

type dynSet struct {
	Seeds string   `json:"seeds"`
	Plans []string `json:"plans"`
}

type dynPlan struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Batches     []dynBatch  `json:"batches"`
	Expect      []dynExpect `json:"expect"`
}

type dynBatch struct {
	Time    string       `json:"time"` // practice-local ISO time
	Actions []api.Action `json:"actions"`
}

// dynExpect is an employee's intended lifecycle; every seed must match.
type dynExpect struct {
	Employee   string   `json:"employee"` // a scenario ID or a hire's requestId
	Start      string   `json:"start,omitempty"`
	Productive string   `json:"productive,omitempty"`
	LastDay    string   `json:"lastDay,omitempty"`
	Replaces   string   `json:"replaces,omitempty"` // whose work a hire takes over: their pay once it is productive is overlap
	Rosters    []string `json:"rosterEffective,omitempty"`
}

type dynResult struct {
	Plan                dynPlan   `json:"plan"`
	PlanSHA256          string    `json:"planSHA256"`
	Set                 string    `json:"set"`
	Engine              string    `json:"engine"`
	ScenarioManifest    string    `json:"scenarioManifest"`
	BaseFingerprint     string    `json:"baseScenarioFingerprint"`
	ScenarioFingerprint string    `json:"scenarioFingerprint"`
	OverlaySHA256       string    `json:"overlaySHA256"`
	Days                int       `json:"days"`
	Seeds               []int64   `json:"seeds"`
	PerSeed             []dynSeed `json:"perSeed"`
}

type dynSeed struct {
	Seed    int64           `json:"seed"`
	Actions []dynActionNote `json:"actions"`
	Staff   []dynEmployee   `json:"staff"`
	// Daily, d = 1..days, at the takeover's clock time d days later:
	// cumulative evaluation-window figures (baseline-adjusted accrual
	// profit; cash) and balances.
	Daily dynDaily `json:"daily"`
	// Cumulative windows from the takeover ("T30", …) and their extras.
	Windows map[string]*window   `json:"windows"`
	Extra   map[string]*dynExtra `json:"extra"`
	Spans   map[string]*dynSpan  `json:"spans"` // "D31-90", "D91-180": studies and orders of the span
	Checks  dynChecks            `json:"checks"`
	Labor   map[string]*dynLabor `json:"labor"` // by endpoint, cumulative
	ByEmp   map[string]*dynLabor `json:"laborByEmployee"`
	Notes   []string             `json:"notes,omitempty"`
}

type dynActionNote struct {
	Time   string `json:"time"`
	Type   string `json:"type"`
	Target string `json:"target"`
	Code   string `json:"code"`
	OK     bool   `json:"ok"`
}

type dynEmployee struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	RequestID  string `json:"requestId,omitempty"`
	Start      string `json:"start,omitempty"`
	Productive string `json:"productive,omitempty"`
	LastDay    string `json:"lastDay,omitempty"`
	Contract   int64  `json:"contractMin,omitempty"`
	// Stages: the first day (1-based from the takeover) each lifecycle
	// stage was observed at the daily sample.
	Stages map[string]int `json:"stages"`
}

type dynDaily struct {
	ProfitCents     []int64 `json:"profitCents"` // operating profit, cumulative
	NetRevenueCents []int64 `json:"netRevenueCents"`
	OpexCents       []int64 `json:"opexCents"`
	CashCents       []int64 `json:"cashCents"`
	ARCents         []int64 `json:"arCents"`
	LiabCents       []int64 `json:"liabilitiesCents"` // payables + accrued payroll + severance payable
	Exams           []int   `json:"exams"`            // cumulative
	Reports         []int   `json:"reports"`          // cumulative final reports
	Backlog         []int   `json:"backlog"`          // unread + reading at the sample
	Technologists   []int   `json:"technologists"`    // headcount (recruiting, onboarding, active, notice)
}

// dynExtra: what measure() does not keep, at a cumulative endpoint.
type dynExtra struct {
	Displacement      api.DisplacementMetrics `json:"displacement"`
	RecruitingCents   int64                   `json:"recruitingCents"`
	SeveranceCents    int64                   `json:"severanceCents"`
	SevPayableCents   int64                   `json:"severancePayableCents"`
	LiabilitiesCents  int64                   `json:"liabilitiesCents"`
	ReconciledCash    bool                    `json:"reconciled"`
	RoomBusySec       map[string]int64        `json:"roomBusySec"`
	RoomStaffedSec    map[string]int64        `json:"roomStaffedSec"`
	RoomOpenSec       map[string]int64        `json:"roomOpenSec"`
	TechBusySec       int64                   `json:"techBusySec"`
	TechDutySec       int64                   `json:"techDutySec"`
	ReadInBlockSec    int64                   `json:"readInBlockSec"`
	ReadBlockSec      int64                   `json:"readBlockSec"`
	WalkInsLeft       int                     `json:"walkInsLeft"`
	ReportsByModality map[string]int          `json:"reportsByModality"`
	Backlog           int                     `json:"backlog"`           // unread + reading
	BacklogOldestMin  int64                   `json:"backlogOldestMin"`  // oldest unfinished study
	BacklogByPriority map[string]int          `json:"backlogByPriority"` // unread + reading
	InconsistentBooks int                     `json:"inconsistentBookings"`
}

// dynSpan: orders received and studies acquired in a span of days.
type dynSpan struct {
	From, To     int                       `json:"-"`
	TAT          map[string]*tatStat       `json:"tatByPriority"`
	Orders       map[string]int            `json:"orders"`
	Lost         map[string]int            `json:"lost"`
	LossPrimary  map[string]map[string]int `json:"lossPrimary"`
	LeadP50Days  map[string]int64          `json:"leadP50Days"`
	LeadP90Days  map[string]int64          `json:"leadP90Days"`
	WalkInsLeftN int                       `json:"-"`
}

// dynLabor decomposes labor and transition expense (wages with benefits).
type dynLabor struct {
	RecruitingCents int64 `json:"recruitingCents"`
	SeveranceCents  int64 `json:"severanceCents"`
	TrainingCents   int64 `json:"trainingCents"`  // a hire's worked pay before its productive date
	OverlapCents    int64 `json:"overlapCents"`   // a replaced employee's worked pay once the replacement is productive
	GuaranteeCents  int64 `json:"guaranteeCents"` // guaranteed pay for contracted time not worked
	WorkedCents     int64 `json:"workedCents"`    // all hourly worked pay (training and overlap included)
	SalaryCents     int64 `json:"salaryCents"`
}

type dynChecks struct {
	AllAccepted      bool `json:"allAccepted"`
	LifecycleMatches bool `json:"lifecycleMatches"`
	MaxTechnologists int  `json:"maxTechnologists"`
	Stranded         int  `json:"stranded"` // displaced events not resolved (rebooked or lost)
	Inconsistent     int  `json:"inconsistentBookings"`
	Reconciled       bool `json:"reconciled"`
	LaborReconciles  bool `json:"laborReconciles"` // postings sum to the ledger's labor, recruiting and severance
}

func TestDynamicStaffing(t *testing.T) {
	var bm dynBenchmark
	readJSON(t, filepath.Join(dynDir, "benchmark.json"), &bm)
	set := envOr("DT_DYN_SET", "eval")
	s, ok := bm.Sets[set]
	if !ok {
		t.Fatalf("no set %q", set)
	}
	only := map[string]bool{}
	for _, n := range strings.Split(os.Getenv("DT_DYN_PLANS"), ",") {
		if n != "" {
			only[n] = true
		}
	}
	var m manifest
	readJSON(t, filepath.Join(dynDir, bm.Scenario), &m)
	cfg, base, fp, ov := manifestConfig(t, m)
	out := envOr("DT_DYN_OUT", filepath.Join(dynDir, "results", set))
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Run("run", func(t *testing.T) {
		for _, name := range s.Plans {
			if len(only) > 0 && !only[name] {
				continue
			}
			raw, err := os.ReadFile(filepath.Join(dynDir, "plans", name+".json"))
			if err != nil {
				t.Fatal(err)
			}
			var p dynPlan
			dec := json.NewDecoder(strings.NewReader(string(raw)))
			dec.DisallowUnknownFields()
			if err := dec.Decode(&p); err != nil || p.Name != name {
				t.Fatalf("%s: %v (name %q)", name, err, p.Name)
			}
			digest := sha256.Sum256(raw)
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				res := dynResult{Plan: p, PlanSHA256: hex.EncodeToString(digest[:]), Set: set, Engine: version.Number,
					ScenarioManifest: bm.Scenario, BaseFingerprint: base, ScenarioFingerprint: fp, OverlaySHA256: ov,
					Days: bm.Days, Seeds: parseSeeds(envOr("DT_DYN_SEEDS", s.Seeds))}
				for _, seed := range res.Seeds {
					r := runDynamic(t, cfg, seed, bm.Days, bm.Endpoints, p)
					res.PerSeed = append(res.PerSeed, r)
					w := r.Windows[fmt.Sprintf("T%d", bm.Days)]
					t.Logf("%s seed %d: T%d profit $%.1fk, exams %d, accepted %v, lifecycle %v",
						name, seed, bm.Days, float64(w.Profit)/1e5, sum(w.Exams), r.Checks.AllAccepted, r.Checks.LifecycleMatches)
				}
				raw, _ := json.MarshalIndent(res, "", " ")
				if err := os.WriteFile(filepath.Join(out, name+".json"), append(raw, '\n'), 0o644); err != nil {
					t.Fatal(err)
				}
			})
		}
	})
}

func readJSON(t *testing.T, path string, v any) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
}

// manifestConfig compiles a calibration manifest's scenario with its
// overlay: the config, base and merged fingerprints and the overlay hash.
func manifestConfig(t *testing.T, m manifest) (*sim.Config, string, string, string) {
	t.Helper()
	f, err := scenario.ResolveFile(m.Scenario)
	if err != nil {
		t.Fatal(err)
	}
	base, err := scenario.Fingerprint(f)
	if err != nil {
		t.Fatal(err)
	}
	var ov string
	if len(m.Overlay) > 0 {
		sum := sha256.Sum256(m.Overlay)
		ov = hex.EncodeToString(sum[:])
		raw, err := scenario.Canonical(f)
		if err != nil {
			t.Fatal(err)
		}
		var b, o any
		json.Unmarshal(raw, &b)
		if err := json.Unmarshal(m.Overlay, &o); err != nil {
			t.Fatal(err)
		}
		merged, _ := json.Marshal(scenario.Merge(b, o))
		if f, err = scenario.Decode(merged); err != nil {
			t.Fatalf("%s: overlay: %v", m.Name, err)
		}
	}
	fp, err := scenario.Fingerprint(f)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := scenario.Compile(f)
	if err != nil {
		t.Fatal(err)
	}
	return cfg, base, fp, ov
}

// runDynamic runs one seed of a plan: each batch at its instant, every
// action required to be accepted; a sample every day at the takeover's
// clock time; full measurements at the endpoints.
func runDynamic(t *testing.T, cfg *sim.Config, seed int64, days int, endpoints []int, p dynPlan) dynSeed {
	w := sim.New(cfg, rng.SeedFromInt(seed))
	if os.Getenv("DT_DYN_CHECK") != "" {
		w.Engine().CheckInvariants = true
	}
	var losses []sim.LossDiagnosis
	w.SetLossProbe(func(d sim.LossDiagnosis) { losses = append(losses, d) })
	start := w.StartTime()
	cal := w.Calendar()
	w.Engine().RunUntil(start, nil)
	r := dynSeed{Seed: seed, Windows: map[string]*window{}, Extra: map[string]*dynExtra{}, Spans: map[string]*dynSpan{},
		Labor: map[string]*dynLabor{}, Checks: dynChecks{AllAccepted: true, LifecycleMatches: true, Reconciled: true, LaborReconciles: true}}
	type timed struct {
		at simtime.Time
		b  dynBatch
	}
	var batches []timed
	for _, b := range p.Batches {
		at, err := cal.ParseISO(b.Time)
		if err != nil || at < start || at > start.Add(simtime.Duration(days)*simtime.Day) {
			t.Fatalf("%s: batch time %q: %v", p.Name, b.Time, err)
		}
		batches = append(batches, timed{at, b})
	}
	slices.SortStableFunc(batches, func(a, b timed) int { return int(a.at - b.at) })
	staff := map[string]*dynEmployee{}
	var order []string
	sample := func(d int) {
		o := w.Observe(0)
		techs := 0
		for _, e := range o.Staff {
			x := staff[e.ID]
			if x == nil {
				x = &dynEmployee{ID: e.ID, Name: e.Name, Stages: map[string]int{}}
				staff[e.ID] = x
				order = append(order, e.ID)
			}
			if _, seen := x.Stages[e.Employment]; !seen && e.Employment != "" {
				x.Stages[e.Employment] = d
			}
			x.Contract = e.ContractMin
			if h := e.Hired; h != nil {
				x.RequestID, x.Start, x.Productive = h.RequestID, h.StartDate, h.ProductiveDate
			}
			if l := e.Leaving; l != nil {
				x.LastDay = l.LastDay
			}
			if e.Role == "technologist" && e.Employment != api.EmploymentTerminated {
				techs++
			}
		}
		r.Checks.MaxTechnologists = max(r.Checks.MaxTechnologists, techs)
		if d == 0 {
			return
		}
		ev := w.Evaluation()
		m := ev.Money
		bs := o.Finance.BalanceSheet
		dl := &r.Daily
		dl.ProfitCents = append(dl.ProfitCents, m.OperatingProfitCents)
		dl.NetRevenueCents = append(dl.NetRevenueCents, m.NetRevenueCents)
		dl.OpexCents = append(dl.OpexCents, m.OperatingExpenseCents)
		dl.CashCents = append(dl.CashCents, o.Finance.CashCents)
		dl.ARCents = append(dl.ARCents, o.Finance.ARCents)
		dl.LiabCents = append(dl.LiabCents, bs.LiabilitiesCents)
		exams := 0
		for _, rm := range ev.Ops.Rooms {
			exams += rm.Exams
		}
		dl.Exams = append(dl.Exams, exams)
		dl.Reports = append(dl.Reports, ev.Ops.Reporting.Final)
		dl.Backlog = append(dl.Backlog, o.Worklist.Unread+o.Worklist.Reading)
		dl.Technologists = append(dl.Technologists, techs)
	}
	sample(0)
	next := 0
	minCash, minDay := int64(1<<62), 0
	for d := 1; d <= days; d++ {
		end := start.Add(simtime.Duration(d) * simtime.Day)
		for next < len(batches) && batches[next].at <= end {
			b := batches[next]
			w.Engine().RunUntil(b.at, nil)
			for i, res := range w.Apply(b.b.Actions) {
				a := b.b.Actions[i]
				target := a.Employee
				if a.Type == api.ActHire {
					target = a.RequestID
				}
				r.Actions = append(r.Actions, dynActionNote{Time: b.b.Time, Type: a.Type, Target: target, Code: res.Code, OK: res.OK})
				if !res.OK || res.Code != api.ActionAccepted {
					r.Checks.AllAccepted = false
					t.Errorf("%s seed %d: %s %s at %s: %s %s", p.Name, seed, a.Type, target, b.b.Time, res.Code, res.Message)
				}
			}
			next++
		}
		w.Engine().RunUntil(end, nil)
		sample(d)
		if c := r.Daily.CashCents[d-1]; c < minCash {
			minCash, minDay = c, d
		}
		if !slices.Contains(endpoints, d) {
			continue
		}
		key := fmt.Sprintf("T%d", d)
		win := measure(w, cfg, start, d)
		win.MinCash, win.MinCashDay = minCash, minDay
		lossCauses(win, losses, cal, start, end)
		r.Windows[key] = win
		r.Extra[key] = dynMeasureExtra(t, w)
		if i := slices.Index(endpoints, d); i > 0 {
			from := endpoints[i-1]
			r.Spans[fmt.Sprintf("D%d-%d", from+1, d)] = dynMeasureSpan(w, cfg, start, from, d, losses)
		}
	}
	for _, id := range order {
		r.Staff = append(r.Staff, *staff[id])
	}
	ex := r.Extra[fmt.Sprintf("T%d", days)]
	r.Checks.Stranded = ex.Displacement.Events - ex.Displacement.Rebooked - ex.Displacement.NoFeasibleSlot - ex.Displacement.Declined
	for _, k := range endpoints {
		e := r.Extra[fmt.Sprintf("T%d", k)]
		r.Checks.Inconsistent += e.InconsistentBooks
		r.Checks.Reconciled = r.Checks.Reconciled && e.ReconciledCash
	}
	if r.Checks.Stranded != 0 || r.Checks.Inconsistent != 0 || !r.Checks.Reconciled {
		t.Errorf("%s seed %d: stranded %d, inconsistent bookings %d, reconciled %v", p.Name, seed, r.Checks.Stranded, r.Checks.Inconsistent, r.Checks.Reconciled)
	}
	if r.Checks.MaxTechnologists > sim.MaxTechnologists {
		t.Errorf("%s seed %d: %d technologists", p.Name, seed, r.Checks.MaxTechnologists)
	}
	dynCheckLifecycle(t, w, p, &r)
	dynLaborCosts(t, w, cfg, start, endpoints, p, &r)
	return r
}

// dynMeasureExtra records an endpoint's displacement, staffing-change
// expense, balances, raw utilization counters and worklist, and checks
// every booked appointment against the staffing (no change): none may
// have lost its coverage without being displaced.
func dynMeasureExtra(t *testing.T, w *sim.World) *dynExtra {
	ev := w.Evaluation()
	ops, m := ev.Ops, ev.Money
	o := w.Observe(0)
	x := &dynExtra{RecruitingCents: m.Ledger.RecruitingCents, SeveranceCents: m.Ledger.SeveranceCents,
		SevPayableCents: o.Finance.BalanceSheet.SeverancePayableCents, LiabilitiesCents: o.Finance.BalanceSheet.LiabilitiesCents,
		ReconciledCash: m.Reconciled, WalkInsLeft: ops.WalkIns.Left,
		RoomBusySec: map[string]int64{}, RoomStaffedSec: map[string]int64{}, RoomOpenSec: map[string]int64{},
		ReportsByModality: map[string]int{}, BacklogByPriority: map[string]int{},
		Backlog: o.Worklist.Unread + o.Worklist.Reading, BacklogOldestMin: o.Worklist.OldestMin}
	if ops.Displacement != nil {
		x.Displacement = *ops.Displacement
	}
	for _, rm := range ops.Rooms {
		x.RoomBusySec[rm.Modality] += rm.BusySec
		x.RoomStaffedSec[rm.Modality] += rm.StaffedSec
		x.RoomOpenSec[rm.Modality] += rm.OpenSec
	}
	for _, s := range ops.Staff {
		switch s.Role {
		case "technologist":
			x.TechBusySec += s.BusySec
			x.TechDutySec += s.OnDutySec
		case "radiologist":
			x.ReadInBlockSec += s.ReadInBlockSec
			x.ReadBlockSec += s.BlockSec
		}
	}
	for _, mm := range m.Ledger.ByModality {
		x.ReportsByModality[mm.Modality] += mm.ReportsFinal
	}
	for _, b := range o.Worklist.ByPriority {
		x.BacklogByPriority[b.Key] = b.Unread + b.Reading
	}
	conflicts, err := w.CheckStaffing(sim.StaffingChange{From: w.DayIndex() + 1})
	if err != nil {
		t.Fatal(err)
	}
	x.InconsistentBooks = len(conflicts)
	return x
}

// dynMeasureSpan measures the referral orders received and the studies
// acquired in days (from, to] after the takeover, as of now (day to).
func dynMeasureSpan(w *sim.World, cfg *sim.Config, start simtime.Time, from, to int, losses []sim.LossDiagnosis) *dynSpan {
	a, b := start.Add(simtime.Duration(from)*simtime.Day), start.Add(simtime.Duration(to)*simtime.Day)
	s := &dynSpan{TAT: tatByPriority(w, cfg, a, b), Orders: map[string]int{}, Lost: map[string]int{},
		LossPrimary: map[string]map[string]int{}, LeadP50Days: map[string]int64{}, LeadP90Days: map[string]int64{}}
	mod := map[string]string{}
	for _, x := range cfg.Catalog {
		mod[x.Code] = x.Modality
	}
	appt := map[string]int64{}
	for _, ap := range w.Appointments() {
		if _, ok := appt[ap.Order]; !ok {
			appt[ap.Order] = ap.Day
		}
	}
	cal := w.Calendar()
	leads := map[string][]int64{}
	for _, o := range w.Orders() {
		if o.WalkIn || o.Received <= a || o.Received > b {
			continue
		}
		md := mod[o.Exam]
		s.Orders[md]++
		if o.State == "lost" {
			s.Lost[md]++
		}
		if d, ok := appt[o.ID]; ok {
			leads[md] = append(leads[md], d-cal.DayIndex(o.Received))
		}
	}
	for md, xs := range leads {
		s.LeadP50Days[md], s.LeadP90Days[md] = pctl(xs, 50), pctl(xs, 90)
	}
	for _, d := range losses {
		if d.WalkIn || d.Received <= a || d.Received > b {
			continue
		}
		if s.LossPrimary[d.Modality] == nil {
			s.LossPrimary[d.Modality] = map[string]int{}
		}
		s.LossPrimary[d.Modality][d.Primary]++
	}
	return s
}

// dynCheckLifecycle compares each employee's observed lifecycle with the
// plan's expectations.
func dynCheckLifecycle(t *testing.T, w *sim.World, p dynPlan, r *dynSeed) {
	byKey := map[string]*dynEmployee{}
	for i := range r.Staff {
		e := &r.Staff[i]
		byKey[e.ID] = e
		if e.RequestID != "" {
			byKey[e.RequestID] = e
		}
	}
	bad := func(format string, args ...any) {
		r.Checks.LifecycleMatches = false
		t.Errorf("%s seed %d: "+format, append([]any{p.Name, r.Seed}, args...)...)
	}
	for _, x := range p.Expect {
		e := byKey[x.Employee]
		if e == nil {
			bad("no employee %s", x.Employee)
			continue
		}
		if x.Start != "" && e.Start != x.Start {
			bad("%s starts %s, want %s", x.Employee, e.Start, x.Start)
		}
		if x.Productive != "" && e.Productive != x.Productive {
			bad("%s productive %s, want %s", x.Employee, e.Productive, x.Productive)
		}
		if x.LastDay != "" && e.LastDay != x.LastDay {
			bad("%s last day %s, want %s", x.Employee, e.LastDay, x.LastDay)
		}
		if len(x.Rosters) > 0 {
			plan, ok := w.RosterPlan(e.ID)
			var got []string
			for _, v := range plan.Versions {
				if v.Effective != "" {
					got = append(got, v.Effective)
				}
			}
			if !ok || !slices.Equal(got, x.Rosters) {
				bad("%s roster versions %v, want %v", x.Employee, got, x.Rosters)
			}
		}
	}
}

// dynLaborCosts decomposes the labor postings by endpoint and by employee
// and checks that they sum to the ledger's figures.
func dynLaborCosts(t *testing.T, w *sim.World, cfg *sim.Config, start simtime.Time, endpoints []int, p dynPlan, r *dynSeed) {
	cal := w.Calendar()
	productive, replacedBy := map[string]int64{}, map[string]string{}
	for _, e := range r.Staff {
		if e.Productive != "" {
			d, _ := simtime.ParseDate(e.Productive)
			productive[e.ID] = cal.DayIndex(cal.StartOfDate(d))
		}
	}
	ids := map[string]string{}
	for _, e := range r.Staff {
		ids[e.ID] = e.ID
		if e.RequestID != "" {
			ids[e.RequestID] = e.ID
		}
	}
	for _, x := range p.Expect {
		if x.Replaces != "" {
			replacedBy[ids[x.Replaces]] = ids[x.Employee]
		}
	}
	posts := w.LaborPostings(start)
	r.ByEmp = map[string]*dynLabor{}
	for _, k := range endpoints {
		end := start.Add(simtime.Duration(k) * simtime.Day)
		l := &dynLabor{}
		for _, ps := range posts {
			if ps.At > end {
				continue
			}
			add := func(l *dynLabor) {
				switch ps.Kind {
				case "recruiting":
					l.RecruitingCents += ps.Cents
				case "severance":
					l.SeveranceCents += ps.Cents
				case "guarantee":
					l.GuaranteeCents += ps.Cents
				case "salary":
					l.SalaryCents += ps.Cents
				case "worked":
					l.WorkedCents += ps.Cents
					if pd, ok := productive[ps.Employee]; ok && ps.Day < pd {
						l.TrainingCents += ps.Cents
					}
					if rep, ok := replacedBy[ps.Employee]; ok {
						if pd, ok := productive[rep]; ok && ps.Day >= pd {
							l.OverlapCents += ps.Cents
						}
					}
				}
			}
			add(l)
			if k == endpoints[len(endpoints)-1] {
				e := r.ByEmp[ps.Employee]
				if e == nil {
					e = &dynLabor{}
					r.ByEmp[ps.Employee] = e
				}
				add(e)
			}
		}
		r.Labor[fmt.Sprintf("T%d", k)] = l
		win := r.Windows[fmt.Sprintf("T%d", k)]
		ex := r.Extra[fmt.Sprintf("T%d", k)]
		if l.RecruitingCents != ex.RecruitingCents || l.SeveranceCents != ex.SeveranceCents {
			r.Checks.LaborReconciles = false
			t.Errorf("%s seed %d T%d: recruiting %d/%d, severance %d/%d", p.Name, r.Seed, k, l.RecruitingCents, ex.RecruitingCents,
				l.SeveranceCents, ex.SeveranceCents)
		}
		// Labor is wages and benefits (worked, salaried and guaranteed)
		// plus severance; recruiting is outside it.
		if got := l.WorkedCents + l.GuaranteeCents + l.SalaryCents + l.SeveranceCents; got != win.Labor {
			r.Checks.LaborReconciles = false
			t.Errorf("%s seed %d T%d: labor postings %d, window labor %d", p.Name, r.Seed, k, got, win.Labor)
		}
	}
}
