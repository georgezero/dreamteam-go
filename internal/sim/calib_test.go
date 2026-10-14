//go:build calibration

package sim_test

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/scenario"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Calibration diagnostics (ROADMAP step 18): the established practice over
// several seeds, 30 and 90 days after the takeover. Run with
//
//	go test -tags calibration ./internal/sim -run Calibration -v -timeout 60m
//	DT_SEEDS=1-9 DT_DAYS=90 DT_CALIB_OUT=docs/calibration/baseline.json
//
// It only reads: observations, summaries, the evaluation window and the
// test exports of disclosed orders and appointments.

type seedRun struct {
	Seed     int64              `json:"seed"`
	Takeover takeoverSnap       `json:"takeover"`
	Windows  map[string]*window `json:"windows"` // "30d", "90d"
}

type takeoverSnap struct {
	Unread, Reading, OpenClaims int
	ReceivablesCents            int64
}

type window struct {
	Days int `json:"days"`
	// Demand and flow, by modality (room order).
	Mods       []string `json:"mods"`
	Orders     []int    `json:"orders"`       // referral orders received
	WalkIns    []int    `json:"walkIns"`      // walk-ins arrived
	Booked     []int    `json:"booked"`       // orders received in the window that got an appointment
	LostDecl   []int    `json:"lostDeclined"` // lost: earliest slot beyond the patient's tolerance
	LostNoSlot []int    `json:"lostNoSlot"`   // lost: no slot within the horizon
	LeadP50    []int64  `json:"leadP50Days"`  // received → appointment day, booked orders
	LeadP90    []int64  `json:"leadP90Days"`
	// The lead split (step 18.1): requested (received → earliest wanted
	// day, the referral's own timing) and capacity delay (earliest wanted
	// day → appointment day).
	ReqLeadP50 []int64   `json:"requestedLeadP50Days"`
	ReqLeadP90 []int64   `json:"requestedLeadP90Days"`
	DelayP50   []int64   `json:"capacityDelayP50Days"`
	DelayP90   []int64   `json:"capacityDelayP90Days"`
	Exams      []int     `json:"exams"`
	TueFriDay  []float64 `json:"tueFriPerDay"` // exams per Tue–Fri day
	MonDay     []float64 `json:"monPerDay"`
	SatDay     []float64 `json:"satPerDay"`
	UtilOpen   []int64   `json:"utilOpenBP"`
	UtilStaff  []int64   `json:"utilStaffedBP"`
	Coverage   []int64   `json:"coverageBP"`
	Reports    int       `json:"reports"`
	Scheduled  int       `json:"scheduled"`
	Cancelled  int       `json:"cancelled"`
	NoShows    int       `json:"noShows"`
	WalkInLeft int       `json:"walkInsLeft"`
	TechUtil   int64     `json:"techUtilBP"`
	ReadUtil   int64     `json:"readUtilBP"`
	TATP50     int64     `json:"tatP50Min"` // all priorities
	TATP90     int64     `json:"tatP90Min"`
	// By priority, and by priority and modality ("urgent/CT"), over the
	// studies acquired in the window, finished or not.
	TATByPri   map[string]*tatStat `json:"tatByPriority"`
	OverTarget int                 `json:"overTarget"`
	Acquired   int                 `json:"acquired"`
	Backlog730 []int               `json:"backlog0730TueFri"` // unread+reading at 07:30, each Tue–Fri
	UnreadEOD  []int               `json:"unreadAtClose"`
	RevPerRpt  []int64             `json:"revenuePerReportCents"` // by modality
	NetRev     int64               `json:"netRevenueCents"`
	Labor      int64               `json:"laborCents"`
	Fixed      int64               `json:"fixedCents"`
	Variable   int64               `json:"variableCents"`
	Opex       int64               `json:"opexCents"`
	Profit     int64               `json:"profitCents"`
	MarginBP   int64               `json:"marginBP"`
	Collect    int64               `json:"collectionsCents"`
	Disburse   int64               `json:"disbursementsCents"`
	ARChange   int64               `json:"arChangeCents"`
	CashEnd    int64               `json:"cashEndCents"`
	OTMin      int64               `json:"overtimeMin"`
	ApprOTMin  int64               `json:"approvedOvertimeMin"` // of OTMin, approved ahead (extend_shift)
	// Cohort: the orders received in the window by origin ("referral",
	// "walk-in") and their state at its end — each order once.
	Cohort map[string]map[string]int `json:"orderCohort"`
	Extra  map[string]int            `json:"extra"`
	// Per person: busy share of duty (technologists, desk), read share of
	// site blocks (radiologists), basis points; overtime minutes.
	StaffUtil map[string]int64 `json:"staffUtilBP"`
	StaffOT   map[string]int64 `json:"staffOvertimeMin"`
}

// tatStat: studies acquired in a window, as of its end. Percentiles and
// max are over the completed ones; Breaches counts completed over target
// plus unfinished already past it, Overnight the completed breaches first
// claimed on a later day than acquired (they waited for the next reading
// block: a coverage gap, not worklist order); OldestMin is the age of the
// oldest unfinished study.
type tatStat struct {
	Acquired   int   `json:"acquired"`
	Completed  int   `json:"completed"`
	Unfinished int   `json:"unfinished"`
	P50Min     int64 `json:"p50Min"`
	P90Min     int64 `json:"p90Min"`
	MaxMin     int64 `json:"maxMin"`
	Breaches   int   `json:"breaches"`
	Overnight  int   `json:"overnightBreaches"`
	OldestMin  int64 `json:"oldestUnfinishedMin"`
	tats       []int64
}

func seedsFromEnv() []int64 { return parseSeeds(os.Getenv("DT_SEEDS")) }

func parseSeeds(spec string) []int64 {
	if spec == "" {
		spec = "1-9"
	}
	var out []int64
	for _, part := range strings.Split(spec, ",") {
		a, b, ok := strings.Cut(part, "-")
		lo, _ := strconv.ParseInt(a, 10, 64)
		hi := lo
		if ok {
			hi, _ = strconv.ParseInt(b, 10, 64)
		}
		for s := lo; s <= hi; s++ {
			out = append(out, s)
		}
	}
	return out
}

func TestCalibrationReport(t *testing.T) {
	ref := os.Getenv("DT_SCENARIO")
	if ref == "" {
		ref = "established"
	}
	f, err := scenario.ResolveFile(ref)
	if err != nil {
		t.Fatal(err)
	}
	// DT_OVERLAY: a JSON overlay merged onto the resolved scenario (the
	// scenario system's own merge), for trying candidate parameters.
	if ov := os.Getenv("DT_OVERLAY"); ov != "" {
		raw, err := scenario.Canonical(f)
		if err != nil {
			t.Fatal(err)
		}
		var base, over any
		json.Unmarshal(raw, &base)
		if err := json.Unmarshal([]byte(ov), &over); err != nil {
			t.Fatal(err)
		}
		merged, _ := json.Marshal(scenario.Merge(base, over))
		if f, err = scenario.Decode(merged); err != nil {
			t.Fatal(err)
		}
	}
	cfg, err := scenario.Compile(f)
	if err != nil {
		t.Fatal(err)
	}
	days := 90
	if v := os.Getenv("DT_DAYS"); v != "" {
		days, _ = strconv.Atoi(v)
	}
	var runs []seedRun
	for _, seed := range seedsFromEnv() {
		runs = append(runs, calibrate(t, cfg, seed, days, nil))
	}
	if out := os.Getenv("DT_CALIB_OUT"); out != "" {
		raw, _ := json.MarshalIndent(runs, "", " ")
		if err := os.WriteFile(out, append(raw, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, w := range []string{"30d", fmt.Sprintf("%dd", days)} {
		t.Log("\n" + summarize(runs, w))
	}
}

// calibrate runs one seed for days from the takeover, applying plan's
// actions (if any) at the takeover; every one must be accepted.
func calibrate(t *testing.T, cfg *sim.Config, seed int64, days int, plan func(*sim.World, int) []api.Action) seedRun {
	w := sim.New(cfg, rng.SeedFromInt(seed))
	start := w.StartTime()
	w.Engine().RunUntil(start, nil)
	if plan != nil {
		if as := plan(w, days); len(as) > 0 {
			for i, r := range w.Apply(as) {
				if !r.OK {
					t.Fatalf("seed %d: %+v rejected: %s %s", seed, as[i], r.Code, r.Message)
				}
			}
		}
	}
	o := w.Observe(0)
	r := seedRun{Seed: seed, Windows: map[string]*window{},
		Takeover: takeoverSnap{Unread: o.Worklist.Unread, Reading: o.Worklist.Reading, OpenClaims: o.Finance.Claims.Open,
			ReceivablesCents: o.Finance.ARCents}}
	cal := w.Calendar()
	d0 := cal.DayIndex(start)
	var backlog []int
	for d := 1; d <= days; d++ {
		if d == 30 || d == days {
			w.Engine().RunUntil(start.Add(simtime.Duration(d)*simtime.Day), nil)
			win := measure(w, cfg, start, d)
			win.Backlog730 = slices.Clone(backlog)
			r.Windows[fmt.Sprintf("%dd", d)] = win
		}
		at := cal.StartOfDay(d0 + int64(d)).Add(7*simtime.Hour + 30*simtime.Minute)
		if wd := cal.Weekday(at); d < days && wd >= simtime.Tuesday && wd <= simtime.Friday {
			w.Engine().RunUntil(at, nil)
			wl := w.Observe(0).Worklist
			backlog = append(backlog, wl.Unread+wl.Reading)
		}
	}
	return r
}

func measure(w *sim.World, cfg *sim.Config, start simtime.Time, days int) *window {
	ev := w.Evaluation()
	ops, m := ev.Ops, ev.Money
	end := start.Add(simtime.Duration(days) * simtime.Day)
	cal := w.Calendar()
	win := &window{Days: days, Extra: map[string]int{}, Cohort: map[string]map[string]int{}}
	modIdx := map[string]int{}
	for _, r := range cfg.Rooms {
		if _, ok := modIdx[r.Modality]; !ok {
			modIdx[r.Modality] = len(win.Mods)
			win.Mods = append(win.Mods, r.Modality)
		}
	}
	n := len(win.Mods)
	win.Orders, win.WalkIns, win.Booked, win.LostDecl, win.LostNoSlot, win.Exams =
		make([]int, n), make([]int, n), make([]int, n), make([]int, n), make([]int, n), make([]int, n)
	win.LeadP50, win.LeadP90, win.UtilOpen, win.UtilStaff, win.Coverage, win.RevPerRpt =
		make([]int64, n), make([]int64, n), make([]int64, n), make([]int64, n), make([]int64, n), make([]int64, n)
	win.TueFriDay, win.MonDay, win.SatDay = make([]float64, n), make([]float64, n), make([]float64, n)
	win.ReqLeadP50, win.ReqLeadP90, win.DelayP50, win.DelayP90 = make([]int64, n), make([]int64, n), make([]int64, n), make([]int64, n)
	exam := map[string]string{}
	for _, x := range cfg.Catalog {
		exam[x.Code] = x.Modality
	}
	apptDay := map[string]int64{}
	for _, a := range w.Appointments() {
		if _, ok := apptDay[a.Order]; !ok {
			apptDay[a.Order] = a.Day // first appointment booked for the order
		}
	}
	leads, reqs, delays := make([][]int64, n), make([][]int64, n), make([][]int64, n)
	for _, o := range w.Orders() {
		if o.Received <= start || o.Received > end {
			continue
		}
		k := modIdx[exam[o.Exam]]
		origin := "referral"
		if o.WalkIn {
			origin = "walk-in"
		}
		if win.Cohort[origin] == nil {
			win.Cohort[origin] = map[string]int{}
		}
		win.Cohort[origin][o.State]++
		if o.WalkIn {
			win.WalkIns[k]++
			continue
		}
		win.Orders[k]++
		switch {
		case o.State == "lost" && strings.HasPrefix(o.LostWhy, "declined"):
			win.LostDecl[k]++
		case o.State == "lost":
			win.LostNoSlot[k]++
		}
		if d, ok := apptDay[o.ID]; ok {
			win.Booked[k]++
			leads[k] = append(leads[k], d-cal.DayIndex(o.Received))
			reqs[k] = append(reqs[k], o.Earliest-cal.DayIndex(o.Received))
			delays[k] = append(delays[k], d-o.Earliest)
		}
	}
	for k := range leads {
		win.LeadP50[k], win.LeadP90[k] = pctl(leads[k], 50), pctl(leads[k], 90)
		win.ReqLeadP50[k], win.ReqLeadP90[k] = pctl(reqs[k], 50), pctl(reqs[k], 90)
		win.DelayP50[k], win.DelayP90[k] = pctl(delays[k], 50), pctl(delays[k], 90)
	}
	win.TATByPri = tatByPriority(w, cfg, start, end)
	for _, r := range ops.Rooms {
		k := modIdx[r.Modality]
		win.Exams[k] += r.Exams
		win.UtilOpen[k], win.UtilStaff[k], win.Coverage[k] = r.UtilOpen.BP(), r.UtilStaffed.BP(), r.Coverage.BP()
	}
	// Per weekday type, from the calendar days fully inside the window.
	var tf, mon, sat int
	for d := cal.DayIndex(start) + 1; d < cal.DayIndex(end); d++ {
		s, _ := w.DaySummary(d)
		var add []float64
		switch s.Weekday {
		case "Tue", "Wed", "Thu", "Fri":
			add, tf = win.TueFriDay, tf+1
		case "Mon":
			add, mon = win.MonDay, mon+1
		case "Sat":
			add, sat = win.SatDay, sat+1
		default:
			continue
		}
		for _, md := range s.ByModality {
			add[modIdx[md.Modality]] += float64(md.Exams)
		}
		win.UnreadEOD = append(win.UnreadEOD, s.UnreadAtClose)
	}
	for k := range win.Mods {
		if tf > 0 {
			win.TueFriDay[k] /= float64(tf)
		}
		if mon > 0 {
			win.MonDay[k] /= float64(mon)
		}
		if sat > 0 {
			win.SatDay[k] /= float64(sat)
		}
	}
	rp := ops.Reporting
	win.Reports, win.Acquired, win.OverTarget = rp.Final, rp.Acquired, rp.OverTarget
	win.TATP50, win.TATP90 = rp.TAT.P50Sec/60, rp.TAT.P90Sec/60
	ap := ops.Appointments
	win.Scheduled, win.Cancelled, win.NoShows = ap.Scheduled, ap.Cancelled, ap.NoShows
	win.WalkInLeft = ops.WalkIns.Left
	var tb, td, rb, rd int64
	for _, s := range ops.Staff {
		switch s.Role {
		case "technologist":
			tb, td = tb+s.BusySec, td+s.OnDutySec
		case "radiologist":
			rb, rd = rb+s.ReadInBlockSec, rd+s.BlockSec
		}
	}
	win.TechUtil, win.ReadUtil = api.Ratio{Num: tb, Den: td}.BP(), api.Ratio{Num: rb, Den: rd}.BP()
	win.StaffUtil, win.StaffOT = map[string]int64{}, map[string]int64{}
	for _, s := range ops.Staff {
		u := s.Util.BP()
		if s.Role == "radiologist" {
			u = s.ReadUtil.BP()
		}
		win.StaffUtil[s.Name], win.StaffOT[s.Name] = u, s.OvertimeSec/60
	}
	for _, mm := range m.Ledger.ByModality {
		if mm.ReportsFinal > 0 {
			win.RevPerRpt[modIdx[mm.Modality]] = mm.NetRevenueCents / int64(mm.ReportsFinal)
		}
	}
	win.NetRev, win.Opex, win.Profit = m.NetRevenueCents, m.OperatingExpenseCents, m.OperatingProfitCents
	win.Labor, win.Fixed, win.Variable = m.Ledger.LaborCents, m.Ledger.FixedCents, m.Ledger.VariableCents
	win.MarginBP = m.Margin.BP()
	win.Collect, win.Disburse, win.ARChange, win.CashEnd = m.CollectionsCents, m.DisbursementsCents, m.ARChangeCents, m.CashEndCents
	win.OTMin, win.ApprOTMin = ops.Workforce.OvertimeSec/60, ops.Workforce.ApprovedOvertimeSec/60
	win.Extra["walkInsDeclined"] = ops.WalkIns.Declined
	win.Extra["walkInsImaged"] = ops.WalkIns.SameVisitExams
	win.Extra["heldOver"] = ops.Workforce.HeldOver
	win.Extra["noCoverage"] = ops.Workforce.NoCoverage
	return win
}

// pctl is the nearest-rank p-th percentile of xs (0 when empty); it sorts xs.
func pctl(xs []int64, p int) int64 {
	if len(xs) == 0 {
		return 0
	}
	slices.Sort(xs)
	return xs[(len(xs)*p+99)/100-1]
}

// tatByPriority measures report turnaround of the studies acquired in
// (start, end], by priority and by priority and modality, as of end.
func tatByPriority(w *sim.World, cfg *sim.Config, start, end simtime.Time) map[string]*tatStat {
	mod := map[string]string{}
	for _, x := range cfg.Catalog {
		mod[x.Code] = x.Modality
	}
	target := map[string]simtime.Duration{}
	for p := sim.Routine; p <= sim.Urgent; p++ {
		target[p.String()] = cfg.Policies.TATTarget[p]
	}
	cal := w.Calendar()
	out := map[string]*tatStat{}
	for _, s := range w.Studies() {
		if s.Acquired <= start || s.Acquired > end {
			continue
		}
		for _, key := range []string{s.Priority, s.Priority + "/" + mod[s.Exam]} {
			ts := out[key]
			if ts == nil {
				ts = &tatStat{}
				out[key] = ts
			}
			ts.Acquired++
			if s.State == "final" && s.Final <= end {
				tat := int64(s.Final.Sub(s.Acquired))
				ts.Completed++
				ts.tats = append(ts.tats, tat)
				if tat > int64(target[s.Priority]) {
					ts.Breaches++
					if cal.DayIndex(s.Claimed) != cal.DayIndex(s.Acquired) {
						ts.Overnight++
					}
				}
				continue
			}
			ts.Unfinished++
			age := end.Sub(s.Acquired)
			if age > target[s.Priority] {
				ts.Breaches++
			}
			ts.OldestMin = max(ts.OldestMin, int64(age/simtime.Minute))
		}
	}
	for _, ts := range out {
		ts.P50Min, ts.P90Min = pctl(ts.tats, 50)/60, pctl(ts.tats, 90)/60
		if len(ts.tats) > 0 {
			ts.MaxMin = ts.tats[len(ts.tats)-1] / 60
		}
	}
	return out
}

// --- aggregation -----------------------------------------------------------

type stat struct{ mean, median, min, max float64 }

func stats(xs []float64) stat {
	if len(xs) == 0 {
		return stat{}
	}
	s := slices.Clone(xs)
	sort.Float64s(s)
	var sum float64
	for _, x := range s {
		sum += x
	}
	med := s[len(s)/2]
	if len(s)%2 == 0 {
		med = (s[len(s)/2-1] + s[len(s)/2]) / 2
	}
	return stat{sum / float64(len(s)), med, s[0], s[len(s)-1]}
}

func summarize(runs []seedRun, key string) string {
	var b strings.Builder
	get := func(f func(*window) float64) stat {
		var xs []float64
		for _, r := range runs {
			if w := r.Windows[key]; w != nil {
				xs = append(xs, f(w))
			}
		}
		return stats(xs)
	}
	w0 := runs[0].Windows[key]
	if w0 == nil {
		return "no window " + key
	}
	per30 := func(w *window, x float64) float64 { return x * 30.4 / float64(w.Days) }
	row := func(name string, f func(*window) float64) {
		s := get(f)
		fmt.Fprintf(&b, "| %-40s | %10.1f | %10.1f | %10.1f | %10.1f |\n", name, s.mean, s.median, s.min, s.max)
	}
	fmt.Fprintf(&b, "Window %s, %d seeds (mean | median | min | max)\n", key, len(runs))
	fmt.Fprintf(&b, "| %-40s | %10s | %10s | %10s | %10s |\n|---|---|---|---|---|\n", "metric", "mean", "median", "min", "max")
	row("takeover unread+reading", func(*window) float64 { return 0 })
	b.Reset()
	fmt.Fprintf(&b, "Window %s, %d seeds (mean | median | min | max)\n", key, len(runs))
	fmt.Fprintf(&b, "| %-40s | %10s | %10s | %10s | %10s |\n|---|---|---|---|---|\n", "metric", "mean", "median", "min", "max")
	var tk []float64
	for _, r := range runs {
		tk = append(tk, float64(r.Takeover.Unread+r.Takeover.Reading))
	}
	s := stats(tk)
	fmt.Fprintf(&b, "| %-40s | %10.1f | %10.1f | %10.1f | %10.1f |\n", "takeover worklist (unread+reading)", s.mean, s.median, s.min, s.max)
	row("exams / month", func(w *window) float64 { return per30(w, float64(sum(w.Exams))) })
	row("reports final / month", func(w *window) float64 { return per30(w, float64(w.Reports)) })
	row("referral orders / month", func(w *window) float64 { return per30(w, float64(sum(w.Orders))) })
	row("walk-ins / month", func(w *window) float64 { return per30(w, float64(sum(w.WalkIns))) })
	for k, m := range w0.Mods {
		k := k
		row(m+" orders / month", func(w *window) float64 { return per30(w, float64(w.Orders[k])) })
		row(m+" walk-ins / month", func(w *window) float64 { return per30(w, float64(w.WalkIns[k])) })
		row(m+" lost declined / month", func(w *window) float64 { return per30(w, float64(w.LostDecl[k])) })
		row(m+" lost no slot / month", func(w *window) float64 { return per30(w, float64(w.LostNoSlot[k])) })
		row(m+" lead p50 days (incl. requested)", func(w *window) float64 { return float64(w.LeadP50[k]) })
		row(m+" requested lead p50 days", func(w *window) float64 { return float64(w.ReqLeadP50[k]) })
		row(m+" capacity delay p50 days", func(w *window) float64 { return float64(w.DelayP50[k]) })
		row(m+" lead p90 days", func(w *window) float64 { return float64(w.LeadP90[k]) })
		row(m+" exams / Tue–Fri day", func(w *window) float64 { return w.TueFriDay[k] })
		row(m+" exams / Monday", func(w *window) float64 { return w.MonDay[k] })
		row(m+" exams / Saturday", func(w *window) float64 { return w.SatDay[k] })
		row(m+" busy/open %", func(w *window) float64 { return float64(w.UtilOpen[k]) / 100 })
		row(m+" busy/staffed %", func(w *window) float64 { return float64(w.UtilStaff[k]) / 100 })
		row(m+" staffed/open %", func(w *window) float64 { return float64(w.Coverage[k]) / 100 })
		row(m+" net revenue / report $", func(w *window) float64 { return float64(w.RevPerRpt[k]) / 100 })
	}
	for _, name := range []string{"Jordan", "Taylor", "Riley", "Sam", "Casey", "Morgan", "Dr. Avery", "Dr. Blake"} {
		name := name
		row(name+" busy (read) % of duty (blocks)", func(w *window) float64 { return float64(w.StaffUtil[name]) / 100 })
		row(name+" overtime h / month", func(w *window) float64 { return per30(w, float64(w.StaffOT[name])) / 60 })
	}
	row("Tue–Fri exams / day (all)", func(w *window) float64 { return fsum(w.TueFriDay) })
	row("cancel rate %", func(w *window) float64 { return pct(w.Cancelled, w.Scheduled) })
	row("no-show rate % (of not cancelled)", func(w *window) float64 { return pct(w.NoShows, w.Scheduled-w.Cancelled) })
	row("walk-ins left / month", func(w *window) float64 { return per30(w, float64(w.WalkInLeft)) })
	row("technologist busy/duty %", func(w *window) float64 { return float64(w.TechUtil) / 100 })
	row("radiologist read/block %", func(w *window) float64 { return float64(w.ReadUtil) / 100 })
	row("backlog 07:30 Tue–Fri, mean", func(w *window) float64 { return mean(w.Backlog730) })
	row("backlog 07:30 Tue–Fri, max", func(w *window) float64 { return float64(slices.Max(append(w.Backlog730, 0))) })
	row("share of Tue–Fri 07:30 in 5–20 %", func(w *window) float64 { return share(w.Backlog730, 5, 20) })
	row("unread at close, mean", func(w *window) float64 { return mean(w.UnreadEOD) })
	row("TAT p50 min, all priorities (censored)", func(w *window) float64 { return float64(w.TATP50) })
	row("TAT p90 min, all priorities", func(w *window) float64 { return float64(w.TATP90) })
	row("TAT over target %, all priorities", func(w *window) float64 { return pct(w.OverTarget, w.Acquired) })
	row("net revenue $k / month", func(w *window) float64 { return per30(w, float64(w.NetRev)) / 100_000 })
	row("labor $k / month", func(w *window) float64 { return per30(w, float64(w.Labor)) / 100_000 })
	row("fixed $k / month", func(w *window) float64 { return per30(w, float64(w.Fixed)) / 100_000 })
	row("variable $k / month", func(w *window) float64 { return per30(w, float64(w.Variable)) / 100_000 })
	row("opex $k / month (adjusted)", func(w *window) float64 { return per30(w, float64(w.Opex)) / 100_000 })
	row("operating profit $k / month", func(w *window) float64 { return per30(w, float64(w.Profit)) / 100_000 })
	row("operating margin %", func(w *window) float64 { return float64(w.MarginBP) / 100 })
	row("collections $k / month", func(w *window) float64 { return per30(w, float64(w.Collect)) / 100_000 })
	row("disbursements $k / month", func(w *window) float64 { return per30(w, float64(w.Disburse)) / 100_000 })
	row("A/R change $k", func(w *window) float64 { return float64(w.ARChange) / 100_000 })
	row("cash at end $k", func(w *window) float64 { return float64(w.CashEnd) / 100_000 })
	row("overtime h / month", func(w *window) float64 { return per30(w, float64(w.OTMin)) / 60 })
	return b.String()
}

func sum(xs []int) int {
	t := 0
	for _, x := range xs {
		t += x
	}
	return t
}

func fsum(xs []float64) float64 {
	t := 0.0
	for _, x := range xs {
		t += x
	}
	return t
}

func mean(xs []int) float64 {
	if len(xs) == 0 {
		return 0
	}
	return float64(sum(xs)) / float64(len(xs))
}

func share(xs []int, lo, hi int) float64 {
	if len(xs) == 0 {
		return 0
	}
	n := 0
	for _, x := range xs {
		if x >= lo && x <= hi {
			n++
		}
	}
	return 100 * float64(n) / float64(len(xs))
}

func pct(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return 100 * float64(a) / float64(b)
}
