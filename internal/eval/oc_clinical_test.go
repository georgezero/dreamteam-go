package eval

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"runtime"
	"slices"
	"sync"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// The combined clinical operating characteristics (Slice 5C.4b,
// docs/calibration/clinical-5c4b/protocol.md): a synthetic practice with
// an imaging stage and the 5C.3 reading queue produces complete records,
// judged by the production evaluator; each gate's failing seeds are
// tabulated over the protocol's value grids, and the protocol's bundles
// give the shared-allowance 40-seed rejection. Exploratory: nothing is
// selected.

// ocPractice is the synthetic practice of the protocol §2. The zero
// value of every departure is the stable stationary practice.
type ocPractice struct {
	Seasonal  bool    `json:"seasonal"`
	DayCV     float64 `json:"dailyArrivalCV"`
	ReadUtil  float64 `json:"readingUtilization"` // 0.85 when 0
	ImgUtil   float64 `json:"imagingUtilization"` // 0.80 when 0
	ReadCap   []ocCap `json:"readingCapacity,omitempty"`
	ImgCap    []ocCap `json:"imagingCapacity,omitempty"`
	ReadErode ocErode `json:"readingErosion,omitzero"`
	ImgErode  ocErode `json:"imagingErosion,omitzero"`
	// Coverage: the reading hours (readingHours). "" is the stable
	// practice's, which covers its imaging.
	Coverage string `json:"readingCoverage,omitempty"`
	// The calibration correction (docs/calibration/clinical-5c4b/
	// correction-protocol.md); each zero value is the original model.
	//   - EffUtil: the reading rate is set so that the expected acquisition
	//     workload (ocWorkload) fills this share of the general reading
	//     hours (generalHours), rather than ReadUtil of 66.5 hours.
	//   - RateHours: the general hours a week that rate is set over (0: the
	//     coverage's own), for a practice keeping another's team rate.
	//   - Warmup: days the practice runs before the takeover, so that it
	//     is in its steady state then; the record inherits the orders and
	//     studies still open at the takeover.
	//   - Worklist "sim": the simulator's urgent_then_fifo (simPick).
	EffUtil   float64 `json:"effectiveReadingUtilization,omitempty"`
	RateHours float64 `json:"rateHours,omitempty"`
	Warmup    int     `json:"warmupDays,omitempty"`
	Worklist  string  `json:"worklist,omitempty"`
}

// ocWorkload is the expected weekly acquisition workload at season-1
// demand: 52 orders a day less the routine cancellations (86% × 5%).
// Orders lost at imaging utilization 0.80 are negligible (measured:
// utilization).
const ocWorkload = 52.0 * 7 * (1 - 0.86*0.05)

// generalHours are the reading hours a week open to every priority, at
// the team's rate (readingHours; urgent-only hours excluded).
func generalHours(coverage string) float64 {
	h := 0.0
	for wd := range 7 {
		for clock := range 24 {
			if s, u := readingHours(coverage, wd, clock); !u {
				h += s
			}
		}
	}
	return h
}

// readRate is the team's reading rate, studies an hour.
func (q ocPractice) readRate() float64 {
	if q.EffUtil == 0 {
		return 52.0 * 7 / 66.5 / cmpOr(q.ReadUtil, 0.85)
	}
	return ocWorkload / cmpOr(q.RateHours, generalHours(q.Coverage)) / q.EffUtil
}

// slots is day d's imaging slot capacity (wd its weekday).
func (q ocPractice) slots(d, wd int) int {
	o, c := imagingHours(wd)
	return int(math.Floor(float64(c-o)*(52.0*7/68/cmpOr(q.ImgUtil, 0.80))*capFactor(d, q.ImgCap, q.ImgErode) + 0.5))
}

// simPick is the simulator's default worklist, urgent_then_fifo
// (internal/sim/reading.go): urgent studies and studies past their
// target first, then the rest, each in entry order. An urgent-only
// reader takes studies of urgent priority only. open is in entry order;
// it returns the index of the study to read at fin, or -1.
func simPick(rec *Record, open []int, fin int64, urgentOnly bool) int {
	first := -1
	for j, i := range open {
		s := &rec.Studies[i]
		if s.EnteredSec >= fin {
			break
		}
		urgent := s.Priority == "urgent"
		if urgentOnly && !urgent {
			continue
		}
		if urgent || fin-s.EnteredSec > target(rec, s.Priority) {
			return j
		}
		if first < 0 {
			first = j
		}
	}
	return first
}

// readingHours is the reading capacity in an hour (wd Monday 0, clock the
// hour) as a share of the team's hourly rate, and whether only urgent
// studies are read:
//
//   - "" (stable): Mon–Fri 07:30–20:00, and one reader for urgent studies
//     20:00–22:00; Sat 10:00–14:00, then one reader to 18:00; Sun one
//     reader 08:00–12:00. Every study acquired is read within its
//     priority's target or soon after.
//   - "satgap": the stable hours without Saturday after 14:00 or Sunday
//     (an urgent weeknight reader only, as the E-family strategies).
//   - "rota": the scenario's own reading rota (the 5C.3 queue's): Mon–Fri
//     07:30–20:00, Sat 10:00–14:00.
func readingHours(coverage string, wd, clock int) (float64, bool) {
	switch {
	case wd < 5 && clock == 7:
		return 0.5, false // from 07:30
	case wd < 5 && clock >= 8 && clock < 20, wd == 5 && clock >= 10 && clock < 14:
		return 1, false
	case coverage == "rota":
		return 0, false
	case wd < 5 && clock >= 20 && clock < 22:
		return 0.5, true
	case coverage == "satgap":
		return 0, false
	case wd == 5 && clock >= 14 && clock < 18, wd == 6 && clock >= 8 && clock < 12:
		return 0.5, false
	}
	return 0, false
}

type ocErode struct {
	FromDay int     `json:"fromDay"`
	PerDay  float64 `json:"perDay"`
}

func capFactor(day int, cs []ocCap, e ocErode) float64 {
	f := 1.0
	if e.PerDay > 0 && day >= e.FromDay {
		f = max(0, 1-e.PerDay*float64(day-e.FromDay))
	}
	for _, c := range cs {
		if day >= c.From && (c.To == 0 || day < c.To) {
			f *= c.Factor
		}
	}
	return f
}

// The scenario's routine booking lead (leadTimeDays) as quantile points.
var ocLead = [][2]float64{{0, 0}, {0.08, 1}, {0.30, 4}, {0.70, 15}, {0.92, 61}, {1, 181}}

func leadDays(u float64) float64 {
	for i := 1; i < len(ocLead); i++ {
		if u <= ocLead[i][0] {
			a, b := ocLead[i-1], ocLead[i]
			return a[1] + (b[1]-a[1])*(u-a[0])/(b[0]-a[0])
		}
	}
	return ocLead[len(ocLead)-1][1]
}

// ocTolerance: the longest wait (days past the wanted day) any patient of
// each priority accepts (waitToleranceDays p100); beyond it the order is
// lost.
var ocTolerance = map[string]int{"urgent": 2, "soon": 10, "routine": 60}

// imaging hours by weekday (Monday 0): opening and closing hour.
func imagingHours(wd int) (int, int) {
	switch {
	case wd < 5:
		return 8, 20
	case wd == 5:
		return 10, 18
	}
	return 0, 0
}

// record builds the practice's evaluation record for policy p (its
// horizon day 150 and observation end).
func (q ocPractice) record(r ocRNG, p *Policy) *Record {
	const perDay = 52.0
	start := t0
	date, _ := simtime.ParseDate(t0Date)
	rec := &Record{Schema: RecordSchema, Plan: "P", Engine: "oc", ScenarioFingerprint: "fp", TATTargetMin: testTargets,
		TakeoverSec: start, TakeoverDate: t0Date, HorizonEndSec: start + 150*daySec, AppointmentsSHA256: "oc"}
	rec.ObservedUntilSec = observationNeeded(p, rec)
	E := rec.ObservedUntilSec
	days := int((E - start) / daySec)
	w := q.Warmup // days before the takeover; indices into capOf and booked are d + w
	midnight := func(d int) int64 { return start - 6*hour + int64(d)*daySec }
	weekday := func(d int) int { return (int(date.AddDays(int64(d)).Weekday()) + 6) % 7 }
	// Imaging capacity by day.
	capOf := make([]int, w+days+300)
	for i := range capOf {
		capOf[i] = q.slots(i-w, weekday(i-w))
	}
	booked := make([]int, len(capOf))
	routineCap := func(d int) int { return int(math.Floor(0.85 * float64(capOf[d+w]))) }
	type acq struct {
		at int64
		k  int
	}
	var acqs []acq
	var resolved []int64
	var outs []string
	dayFactor := 1.0
	for h := -w * 24; h < days*24; h++ {
		at := start + int64(h)*hour
		day := int(floorDiv(int64(6+h), 24))
		clock := 6 + h - 24*day
		dt := date.AddDays(int64(day))
		wd := weekday(day)
		if clock == 0 && q.DayCV > 0 {
			k := 1 / (q.DayCV * q.DayCV)
			dayFactor = r.gamma(k) / k
		}
		if clock < 8 || clock >= 20 {
			continue
		}
		lam := perDay * ocWeekday[wd] / 12 * dayFactor
		if q.Seasonal {
			lam *= ocSeason[dt.Month-1]
		}
		for range r.poisson(lam) {
			u := r.Float64()
			k := 2
			switch {
			case u < 0.02:
				k = 0
			case u < 0.14:
				k = 1
			}
			pri := Priorities[k]
			recv := at + int64(r.Float64()*3600)
			var lead float64
			switch k {
			case 1:
				lead = 2 * r.Float64()
			case 2:
				lead = leadDays(r.Float64())
			}
			wantDay := day + int(lead)
			wanted := max(recv, midnight(wantDay))
			o := Order{ID: "o", Priority: pri, Modality: "XR", ReceivedSec: recv, WantedSec: wanted}
			// Book the first day at or after the wanted day with a free slot
			// and time left after the wanted time.
			bookDay, found := 0, false
			for d := wantDay; d <= wantDay+ocTolerance[pri] && d+w < len(capOf); d++ {
				_, cl := imagingHours(weekday(d))
				if cl == 0 || midnight(d)+int64(cl)*hour <= wanted {
					continue
				}
				limit := capOf[d+w]
				if k == 2 {
					limit = routineCap(d)
				}
				if booked[d+w] < limit {
					bookDay, found = d, true
					break
				}
			}
			out, res := OutOpen, int64(0)
			if !found {
				out, res = OutLost, recv
			} else {
				booked[bookDay+w]++
				op, cl := imagingHours(weekday(bookDay))
				lo := max(wanted, midnight(bookDay)+int64(op)*hour)
				hi := midnight(bookDay) + int64(cl)*hour
				appt := lo + int64(r.Float64()*float64(hi-lo))
				o.FirstApptSec = appt
				switch {
				case k == 2 && r.Float64() < 0.05:
					c := recv + int64(r.Float64()*float64(appt-recv))
					if c <= E {
						out, res = OutPatient, c
					}
				case appt <= E:
					out, res = OutServed, appt
					acqs = append(acqs, acq{appt, k})
				}
			}
			rec.Orders = append(rec.Orders, o)
			resolved = append(resolved, res)
			outs = append(outs, out)
		}
	}
	for i := range rec.Orders {
		rec.Orders[i].ResolvedSec = resolved[i]
	}
	if w > 0 {
		// Inherited: only the orders still open at the takeover.
		var os []Order
		var oo []string
		for i, o := range rec.Orders {
			if o.ReceivedSec > start || o.ResolvedSec == 0 || o.ResolvedSec > start {
				os, oo = append(os, o), append(oo, outs[i])
			}
		}
		rec.Orders, outs = os, oo
	}
	// Reading (the 5C.3 model; arrivals in acquisition order).
	slices.SortStableFunc(acqs, func(a, b acq) int { return int(a.at - b.at) })
	for _, a := range acqs {
		rec.Studies = append(rec.Studies, Study{ID: "s", Priority: Priorities[a.k], Modality: "XR", AcquiredSec: a.at, EnteredSec: a.at})
	}
	readPerHour := q.readRate()
	var queue [3][]int
	var open []int // the "sim" worklist, in entry order
	next := 0
	carry := 0.0
	for h := -w * 24; h < days*24; h++ {
		at := start + int64(h)*hour
		day := int(floorDiv(int64(6+h), 24))
		clock := 6 + h - 24*day
		wd := weekday(day)
		for next < len(rec.Studies) && rec.Studies[next].AcquiredSec < at+hour {
			if q.Worklist == "sim" {
				open = append(open, next)
			} else {
				k := slices.Index(Priorities, rec.Studies[next].Priority)
				queue[k] = append(queue[k], next)
			}
			next++
		}
		share, urgentOnly := readingHours(q.Coverage, wd, clock)
		if share == 0 {
			continue
		}
		c := readPerHour * share * capFactor(day, q.ReadCap, q.ReadErode)
		carry += c
		for carry >= 1 {
			fin := at + hour - int64(carry/c*3600)
			var s *Study
			if q.Worklist == "sim" {
				j := simPick(rec, open, fin, urgentOnly)
				if j < 0 {
					carry = min(carry, 1)
					break
				}
				s = &rec.Studies[open[j]]
				open = slices.Delete(open, j, j+1)
			} else {
				k := 0
				for k < 3 && (len(queue[k]) == 0 || rec.Studies[queue[k][0]].AcquiredSec >= fin || (urgentOnly && k > 0)) {
					k++
				}
				if k == 3 {
					carry = min(carry, 1)
					break
				}
				s = &rec.Studies[queue[k][0]]
				queue[k] = queue[k][1:]
			}
			st := max(fin-60, s.EnteredSec)
			s.ClaimedSec, s.ReadStartSec, s.FinalSec = st, st, max(fin, st)
			carry--
		}
	}
	for i := range rec.Studies {
		if s := &rec.Studies[i]; s.FinalSec > E {
			s.ClaimedSec, s.ReadStartSec, s.FinalSec = 0, 0, 0
		}
	}
	if w > 0 {
		// Inherited: only the studies unread at the takeover.
		rec.Studies = slices.DeleteFunc(rec.Studies, func(s Study) bool {
			return s.AcquiredSec <= start && s.FinalSec != 0 && s.FinalSec <= start
		})
	}
	// Samples and captures.
	m := Money{CashCents: 1_000_000, MinCashCents: 1_000_000, MinCashSec: start, CashBridgeReconciles: true}
	for d := 1; d <= days; d++ {
		at := start + int64(d)*daySec
		s := Sample{Day: d, AtSec: at, Date: date.AddDays(int64(d)).String(), CashCents: m.CashCents}
		for i := range rec.Studies {
			x := &rec.Studies[i]
			if x.AcquiredSec > at {
				break
			}
			if x.AcquiredSec > start {
				s.AcquiredCum++
			}
			if x.FinalSec != 0 && x.FinalSec <= at {
				if x.FinalSec > start {
					s.FinalCum++
				}
				continue
			}
			s.Unread++
			if at-x.AcquiredSec > target(rec, x.Priority) {
				s.Overdue++
			}
		}
		rec.Samples = append(rec.Samples, s)
	}
	for _, c := range []capAt{{"T30", start + 30*daySec}, {"T90", start + 90*daySec}, {"tailEnd", rec.HorizonEndSec}, {"T180", start + 180*daySec}, {"end", E}} {
		oo := make([]string, len(rec.Orders))
		for i, o := range rec.Orders {
			switch {
			case o.ReceivedSec > c.at:
				oo[i] = OutNotDisclosed
			case o.ResolvedSec == 0 || o.ResolvedSec > c.at:
				oo[i] = OutOpen
			default:
				oo[i] = outs[i]
			}
		}
		rec.Captures = append(rec.Captures, Capture{Name: c.name, AtSec: c.at, OrderOutcomes: oo, Money: m})
	}
	return rec
}

func cmpOr(x, def float64) float64 {
	if x == 0 {
		return def
	}
	return x
}

// The protocol's grids (§5) and bundles.
var (
	ocAbsGrid = map[string][]int64{"urgent": {12 * 60, 24 * 60, 36 * 60, 48 * 60, 72 * 60, 120 * 60}, "soon": {24 * 60, 48 * 60, 72 * 60, 120 * 60, 168 * 60},
		"routine": {7 * 1440, 10 * 1440, 14 * 1440}}
	ocSevereRateGrid = map[string][]float64{"urgent": {0.05, 0.10, 0.15, 0.20}, "soon": {0.05, 0.10, 0.15, 0.20}, "routine": {0.01, 0.02, 0.05, 0.10}}
	ocWeekGrid       = map[string][]int64{"urgent": {1, 2, 3, 5}, "soon": {3, 5, 8, 12}, "routine": {5, 10, 20, 40}}
	ocAccessGrid     = map[string][]int64{"urgent": {1440, 2 * 1440, 3 * 1440}, "soon": {3 * 1440, 7 * 1440, 10 * 1440}, "routine": {14 * 1440, 21 * 1440, 30 * 1440}}
	ocTimelyGrid     = []float64{0.80, 0.85, 0.90, 0.95}
	ocMaxWaitGrid    = map[string][]int64{"urgent": {3 * 1440, 7 * 1440, 14 * 1440}, "soon": {14 * 1440, 21 * 1440, 31 * 1440}, "routine": {31 * 1440}}
)

type ocBundle struct {
	Name          string             `json:"name"`
	Absolute      map[string]int64   `json:"absoluteMaxPastDueMin"`
	SevereRate    map[string]float64 `json:"maxSevereRate"`
	SeverePerWeek map[string]int64   `json:"maxSeverePerWeek"`
	Access        map[string]int64   `json:"accessLimitMin"`
	Timely        float64            `json:"minTimelyFraction"`
	MaxWait       map[string]int64   `json:"maxWaitMin"`
}

var ocBundles = []ocBundle{
	{"lenient", map[string]int64{"urgent": 120 * 60, "soon": 168 * 60, "routine": 14 * 1440}, map[string]float64{"urgent": 0.20, "soon": 0.20, "routine": 0.10},
		map[string]int64{"urgent": 5, "soon": 12, "routine": 40}, map[string]int64{"urgent": 3 * 1440, "soon": 10 * 1440, "routine": 30 * 1440}, 0.80,
		map[string]int64{"urgent": 14 * 1440, "soon": 31 * 1440, "routine": 31 * 1440}},
	{"moderate", map[string]int64{"urgent": 72 * 60, "soon": 120 * 60, "routine": 14 * 1440}, map[string]float64{"urgent": 0.10, "soon": 0.10, "routine": 0.05},
		map[string]int64{"urgent": 3, "soon": 8, "routine": 20}, map[string]int64{"urgent": 2 * 1440, "soon": 7 * 1440, "routine": 21 * 1440}, 0.90,
		map[string]int64{"urgent": 7 * 1440, "soon": 21 * 1440, "routine": 31 * 1440}},
	{"strict", map[string]int64{"urgent": 24 * 60, "soon": 48 * 60, "routine": 14 * 1440}, map[string]float64{"urgent": 0.05, "soon": 0.05, "routine": 0.02},
		map[string]int64{"urgent": 1, "soon": 5, "routine": 10}, map[string]int64{"urgent": 1440, "soon": 3 * 1440, "routine": 14 * 1440}, 0.95,
		map[string]int64{"urgent": 3 * 1440, "soon": 14 * 1440, "routine": 31 * 1440}},
}

// ocClinicalPolicy is the policy in force made a v5 candidate with b's
// values (the severe threshold 1,440 minutes, inherited work from the
// takeover, the 10% shared allowance, the access cohorts the breach
// gates' minimums, the maximum wait seed-level).
func ocClinicalPolicy(t *testing.T, b ocBundle) *Policy {
	p, _ := testPolicy(t)
	p.Schema = PolicyV5Schema
	p.Backlog.InformationalChecks = append(slices.Clone(p.Backlog.InformationalChecks), "persistent")
	timely := map[string]float64{}
	for _, q := range Priorities {
		timely[q] = b.Timely
	}
	p.Clinical = &Clinical{AbsoluteMaxPastDueMin: b.Absolute, SevereAfterDueMin: map[string]int64{"urgent": 1440, "soon": 1440, "routine": 1440},
		MaxSevereRate: b.SevereRate, MaxSeverePerWeek: b.SeverePerWeek, AccessLimitMin: b.Access, MinTimelyFraction: timely,
		MinAccessCohort: p.Service.MinCohort, MaxWaitMin: b.MaxWait, InheritedFrom: "takeover", MaxFailingSeedFraction: 0.1}
	if err := p.validate(); err != nil {
		t.Fatal(err)
	}
	return p
}

// ocCount is one cohort's counts.
type ocCount struct{ num, den, undecided int64 }

// ocSeed is one seed's tabulation input: everything the gates of the
// grids need, from the production measures.
type ocSeed struct {
	valid   bool
	service string // v4 per-seed service status
	backlog string // v4 backlog (the sampled age limit reported only)
	pastDue map[string]int64
	severe  map[string]ocCount // priority/window
	week    map[string]int64
	access  map[string]ocCount // priority/window/limit
	maxWait map[string]int64
	breach  map[string]ocCount // priority/window, gated windows (pooled)
}

// perSeedWindows of each priority (the policy's).
func ocWindows(p *Policy, q string) []string { return p.Service.PerSeedWindows[q] }

// ocExtract measures a record under the base candidate and each access
// limit of the grid.
func ocExtract(p *Policy, r *Record) ocSeed {
	m := Measure(p, r)
	x := ocSeed{valid: len(m.Integrity) == 0, pastDue: map[string]int64{}, severe: map[string]ocCount{}, week: map[string]int64{},
		access: map[string]ocCount{}, maxWait: map[string]int64{}, breach: map[string]ocCount{}}
	gs := serviceGates(p, m)
	x.service, x.backlog = seedStatus(p, gs, "gated"), m.Backlog.Status
	for _, g := range gs {
		if g.Gate == "breach_rate" && slices.Contains(p.Service.Gated, g.Window) {
			u := int64(0)
			if g.Undecided {
				u = 1
			}
			x.breach[g.Stratum+"/"+g.Window] = ocCount{g.Num, g.Den, u}
		}
	}
	cm := m.Clinical
	for _, a := range cm.Absolute {
		if a.Limit == "pastDue" {
			x.pastDue[a.Priority] = a.WorstMin
		}
	}
	for _, q := range Priorities {
		x.week[q] = cm.WorstWeekSevere[q]
		x.maxWait[q] = cm.MaxWaitGatedMin[q]
		for _, w := range []string{WinT90, WinTail, WinGated} {
			d := cm.Delay[w][q]
			x.severe[q+"/"+w] = ocCount{d.Severe, d.Acquired, d.Undecided}
		}
	}
	c := *p.Clinical
	ws := windows(p, r)
	end := &r.Captures[len(r.Captures)-1]
	for i := range ocAccessGrid["urgent"] {
		c.AccessLimitMin = map[string]int64{}
		for _, q := range Priorities {
			c.AccessLimitMin[q] = ocAccessGrid[q][i]
		}
		pc := *p
		pc.Clinical = &c
		cm := measureClinical(&pc, r, ws, end)
		for _, q := range Priorities {
			for _, w := range []string{WinT90, WinTail, WinGated} {
				a := cm.Access[w][q]
				x.access[fmt.Sprintf("%s/%s/%d", q, w, c.AccessLimitMin[q])] = ocCount{a.Timely, a.Timely + a.Untimely, a.Undecided}
			}
		}
	}
	return x
}

// Gate decisions from the tabulation (the production rules: insufficient
// is failing).
func (x ocSeed) severeFails(p *Policy, q string, r float64) bool {
	for _, w := range ocWindows(p, q) {
		c := x.severe[q+"/"+w]
		if c.undecided > 0 || c.den < p.Service.MinCohort[q] || rateAbove(c.num, c.den, bp(r)) {
			return true
		}
	}
	return false
}

func (x ocSeed) accessFails(p *Policy, q string, limit int64, g float64) bool {
	for _, w := range ocWindows(p, q) {
		c := x.access[fmt.Sprintf("%s/%s/%d", q, w, limit)]
		if c.undecided > 0 || c.den < p.Service.MinCohort[q] || rateBelow(c.num, c.den, bp(g)) {
			return true
		}
	}
	return false
}

// bundle reports whether the seed fails the bundle's shared clinical rule
// and whether it violates an absolute limit.
func (x ocSeed) bundle(p *Policy, b ocBundle) (clinical, absolute bool) {
	clinical = !x.valid || x.service != Pass || x.backlog != "stable"
	for _, q := range Priorities {
		clinical = clinical || x.severeFails(p, q, b.SevereRate[q]) || x.week[q] > b.SeverePerWeek[q] ||
			x.accessFails(p, q, b.Access[q], b.Timely) || x.maxWait[q] > b.MaxWait[q]
		absolute = absolute || x.pastDue[q] > b.Absolute[q]
	}
	return clinical, absolute
}

// The run's output.
type ocGateRow struct {
	Rule              string   `json:"rule"`
	Priority          string   `json:"priority"`
	Value             float64  `json:"value"`
	Limit             int64    `json:"accessLimitMin,omitempty"`
	FailingSeeds      int      `json:"failingSeeds"`
	FailingFraction   Interval `json:"failingFraction"` // Clopper–Pearson 95%
	UpperBound95      float64  `json:"oneSidedUpper95"`
	Rejection40       float64  `json:"rejection40SharedAllowance,omitempty"`
	AbsoluteRejection float64  `json:"rejection40ZeroTolerance,omitempty"`
}

type ocBundleRow struct {
	Bundle            string  `json:"bundle"`
	ClinicalFailing   int     `json:"clinicalFailingSeeds"`
	AbsoluteViolating int     `json:"absoluteViolatingSeeds"`
	Rejection40       float64 `json:"rejection40"`                // shared allowance and zero tolerance together
	Rejection40Joint  float64 `json:"rejection40SharedOnly"`      // the shared allowance alone
	Rejection40Abs    float64 `json:"rejection40AbsoluteOnly"`    // the absolute limits alone
	Groups            int     `json:"disjointGroupsOf40"`         // the seeds split into groups of 40, judged whole
	GroupsRejected    int     `json:"groupsRejected"`             // by the seed rules or a pooled rate
	PooledRejected    int     `json:"groupsRejectedByPooledRate"` // of them, by a pooled breach or timely rate
}

type ocScenarioOut struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Role        string         `json:"role"`
	Practice    ocPractice     `json:"practice"`
	Seeds       int            `json:"seeds"`
	Invalid     int            `json:"invalidRecords"`
	Studies     float64        `json:"meanStudies"`
	UrgentWorst Dist           `json:"urgentWorstPastDueMinPerSeed"`
	SoonWorst   Dist           `json:"soonWorstPastDueMinPerSeed"`
	Components  map[string]int `json:"seedsFailingComponent"` // v4 service, v4 backlog
	Gates       []ocGateRow    `json:"gates"`
	Bundles     []ocBundleRow  `json:"bundles"`
}

type ocScenarioDef struct {
	name, desc, role string
	q                ocPractice
	seeds            int
}

func ocClinicalScenarios() []ocScenarioDef {
	return []ocScenarioDef{
		{"A", "stable, stationary", "stable", ocPractice{}, 2500},
		{"B", "stable, seasonal", "stable", ocPractice{Seasonal: true}, 2500},
		{"C", "stable, bursty (day CV 0.15)", "stable", ocPractice{DayCV: 0.15}, 2500},
		{"Cs", "stable, seasonal and bursty", "stable", ocPractice{Seasonal: true, DayCV: 0.15}, 2500},
		{"Wg", "structural: no reading Saturday after 14:00 or Sunday (an urgent weeknight reader only)", "structural", ocPractice{Coverage: "satgap"}, 1000},
		{"R0", "structural: the scenario's own reading rota (Mon–Fri 07:30–20:00, Sat 10:00–14:00)", "structural", ocPractice{Coverage: "rota"}, 1000},
		{"B92", "stress: reading utilization 0.92, seasonal", "stress", ocPractice{Seasonal: true, ReadUtil: 0.92}, 1000},
		{"I92", "stress: imaging utilization 0.92, seasonal", "stress", ocPractice{Seasonal: true, ImgUtil: 0.92}, 1000},
		{"L10", "reading capacity 10% lower from day 60", "harmful", ocPractice{ReadCap: []ocCap{{From: 60, Factor: 0.9}}}, 1000},
		{"K", "reading capacity eroding 0.2% a day from day 30", "harmful", ocPractice{ReadErode: ocErode{30, 0.002}}, 1000},
		{"IL10", "imaging capacity 10% lower from day 60", "harmful", ocPractice{ImgCap: []ocCap{{From: 60, Factor: 0.9}}}, 1000},
		{"IK", "imaging capacity eroding 0.2% a day from day 30", "harmful", ocPractice{ImgErode: ocErode{30, 0.002}}, 1000},
		{"D", "reading capacity 25% lower from day 30", "harmful", ocPractice{ReadCap: []ocCap{{From: 30, Factor: 0.75}}}, 1000},
	}
}

// rejection40 is P(Binomial(40, q) > 4).
func rejection40(q float64) float64 {
	acc := 0.0
	for k := 0; k <= 4; k++ {
		acc += binomPMF(40, k, q)
	}
	return 1 - acc
}

func binomPMF(n, k int, p float64) float64 {
	lc, _ := math.Lgamma(float64(n + 1))
	a, _ := math.Lgamma(float64(k + 1))
	b, _ := math.Lgamma(float64(n - k + 1))
	if p == 0 {
		if k == 0 {
			return 1
		}
		return 0
	}
	if p == 1 {
		if k == n {
			return 1
		}
		return 0
	}
	return math.Exp(lc - a - b + float64(k)*math.Log(p) + float64(n-k)*math.Log(1-p))
}

// rejectionBoth: a seed is good (g), clinically failing without an
// absolute violation (f), or violating (v); a 40-seed evaluation accepts
// when no seed violates and at most 4 fail.
func rejectionBoth(g, f float64) float64 {
	acc := 0.0
	for k := 0; k <= 4; k++ {
		lc, _ := math.Lgamma(41)
		a, _ := math.Lgamma(float64(k + 1))
		b, _ := math.Lgamma(float64(40 - k + 1))
		term := math.Exp(lc - a - b)
		term *= math.Pow(f, float64(k)) * math.Pow(g, float64(40-k))
		acc += term
	}
	return 1 - acc
}

// clopperPearson is the exact two-sided 95% interval of k of n.
func clopperPearson(k, n int) Interval {
	iv := Interval{N: n, Mean: float64(k) / float64(n), Defined: n > 0}
	if k > 0 {
		iv.Lo = betaQuantile(float64(k), float64(n-k+1), 0.025)
	}
	iv.Hi = 1
	if k < n {
		iv.Hi = betaQuantile(float64(k+1), float64(n-k), 0.975)
	}
	return iv
}

func betaQuantile(a, b, p float64) float64 {
	lo, hi := 0.0, 1.0
	for range 100 {
		m := (lo + hi) / 2
		if betaInc(a, b, m) < p {
			lo = m
		} else {
			hi = m
		}
	}
	return (lo + hi) / 2
}

func ocClinicalCompute(t *testing.T, defs []ocScenarioDef, stream uint64, seedsOverride int) []ocScenarioOut {
	base := ocClinicalPolicy(t, ocBundles[1])
	var outs []ocScenarioOut
	for k, def := range defs {
		n := def.seeds
		if seedsOverride > 0 {
			n = seedsOverride
		}
		seeds := make([]ocSeed, n)
		studies := make([]int, n)
		var wg sync.WaitGroup
		work := make(chan int)
		for range runtime.NumCPU() {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := range work {
					rec := def.q.record(ocRNG{rand.New(rand.NewPCG(stream, uint64(100000*k+i)))}, base)
					rec.Seed = int64(i)
					seeds[i], studies[i] = ocExtract(base, rec), len(rec.Studies)
				}
			}()
		}
		for i := range n {
			work <- i
		}
		close(work)
		wg.Wait()
		outs = append(outs, ocTabulate(base, def, seeds, studies))
	}
	return outs
}

func ocTabulate(p *Policy, def ocScenarioDef, seeds []ocSeed, studies []int) ocScenarioOut {
	n := len(seeds)
	o := ocScenarioOut{Name: def.name, Description: def.desc, Role: def.role, Practice: def.q, Seeds: n, Components: map[string]int{}}
	var uw, sw []int64
	tot := 0
	for i, x := range seeds {
		tot += studies[i]
		if !x.valid {
			o.Invalid++
		}
		if x.service != Pass {
			o.Components["v4ServicePerSeed"]++
		}
		if x.backlog != "stable" {
			o.Components["v4BacklogWithoutAgeLimit"]++
		}
		uw, sw = append(uw, x.pastDue["urgent"]), append(sw, x.pastDue["soon"])
	}
	o.Studies = float64(tot) / float64(n)
	o.UrgentWorst, o.SoonWorst = dist(uw), dist(sw)
	row := func(rule, q string, v float64, limit int64, fails func(ocSeed) bool, absolute bool) {
		k := 0
		for _, x := range seeds {
			if fails(x) {
				k++
			}
		}
		r := ocGateRow{Rule: rule, Priority: q, Value: v, Limit: limit, FailingSeeds: k, FailingFraction: clopperPearson(k, n), UpperBound95: binomUpper(k, n, 0.95)}
		f := float64(k) / float64(n)
		if absolute {
			r.AbsoluteRejection = 1 - math.Pow(1-f, 40)
		} else {
			r.Rejection40 = rejection40(f)
		}
		o.Gates = append(o.Gates, r)
	}
	for _, q := range Priorities {
		for _, l := range ocAbsGrid[q] {
			row("absoluteMaxPastDueMin", q, float64(l), 0, func(x ocSeed) bool { return x.pastDue[q] > l }, true)
		}
		for _, r := range ocSevereRateGrid[q] {
			row("maxSevereRate", q, r, 0, func(x ocSeed) bool { return x.severeFails(p, q, r) }, false)
		}
		for _, w := range ocWeekGrid[q] {
			row("maxSeverePerWeek", q, float64(w), 0, func(x ocSeed) bool { return x.week[q] > w }, false)
		}
		for _, l := range ocAccessGrid[q] {
			for _, g := range ocTimelyGrid {
				row("minTimelyFraction", q, g, l, func(x ocSeed) bool { return x.accessFails(p, q, l, g) }, false)
			}
		}
		for _, w := range ocMaxWaitGrid[q] {
			row("maxWaitMin", q, float64(w), 0, func(x ocSeed) bool { return x.maxWait[q] > w }, false)
		}
	}
	for _, b := range ocBundles {
		br := ocBundleRow{Bundle: b.Name}
		var good, fail int
		for _, x := range seeds {
			c, a := x.bundle(p, b)
			switch {
			case a:
				br.AbsoluteViolating++
			case c:
				fail++
			default:
				good++
			}
			if c {
				br.ClinicalFailing++
			}
		}
		fn := float64(n)
		br.Rejection40 = rejectionBoth(float64(good)/fn, float64(fail)/fn)
		br.Rejection40Joint = rejection40(float64(br.ClinicalFailing) / fn)
		br.Rejection40Abs = 1 - math.Pow(1-float64(br.AbsoluteViolating)/fn, 40)
		// Disjoint groups of 40, judged whole: the seed rules and every
		// gated pooled rate (breach, timely access).
		for g := 0; g+40 <= n; g += 40 {
			br.Groups++
			bad, abs := 0, false
			type pkey struct {
				access bool
				q, w   string
			}
			pooled := map[pkey]ocCount{}
			add := func(k pkey, v ocCount) {
				s := pooled[k]
				pooled[k] = ocCount{s.num + v.num, s.den + v.den, s.undecided + v.undecided}
			}
			for _, x := range seeds[g : g+40] {
				c, a := x.bundle(p, b)
				if c {
					bad++
				}
				abs = abs || a
				for _, q := range Priorities {
					for _, w := range p.Service.Gated {
						add(pkey{false, q, w}, x.breach[q+"/"+w])
						add(pkey{true, q, w}, x.access[fmt.Sprintf("%s/%s/%d", q, w, b.Access[q])])
					}
				}
			}
			pool := false
			for k, s := range pooled {
				switch {
				case s.undecided > 0 || s.den == 0:
					pool = true
				case !k.access && rateAbove(s.num, s.den, bp(p.Service.BreachGate[k.q])):
					pool = true
				case k.access && rateBelow(s.num, s.den, bp(b.Timely)):
					pool = true
				}
			}
			if abs || bad > 4 || pool {
				br.GroupsRejected++
			}
			if pool {
				br.PooledRejected++
			}
		}
		o.Bundles = append(o.Bundles, br)
	}
	return o
}

// The tabulation agrees exactly with the production evaluator: on
// sampled seeds of a stable, a structural and a harmful scenario, every
// bundle's seed decision is planReport's, and the samples include both.
func TestClinicalOCAgreesWithEvaluate(t *testing.T) {
	defs := ocClinicalScenarios()
	seen := map[bool]bool{}
	defer func() {
		if !seen[true] || !seen[false] {
			t.Errorf("the sampled decisions do not include both a passing and a failing seed: %v", seen)
		}
	}()
	for k, def := range defs {
		if !slices.Contains([]string{"A", "R0", "D"}, def.name) {
			continue
		}
		for _, b := range ocBundles {
			p := ocClinicalPolicy(t, b)
			base := ocClinicalPolicy(t, ocBundles[1])
			var recs []Record
			var xs []ocSeed
			for i := range 3 {
				rec := def.q.record(ocRNG{rand.New(rand.NewPCG(20261101, uint64(100000*k+i)))}, base)
				rec.Seed = int64(i)
				recs = append(recs, *rec)
				xs = append(xs, ocExtract(base, rec))
			}
			pr := planReport(p, "P", recs)
			if !pr.Valid {
				t.Fatalf("%s: %v", def.name, pr.Integrity)
			}
			for i, s := range pr.PerSeed {
				c, a := xs[i].bundle(p, b)
				seen[c || a] = true
				viol := slices.Contains(pr.ClinicalSafety.Seeds, s.Seed)
				if c != (s.Clinical != Pass) || a != viol {
					t.Errorf("%s %s seed %d: tabulated clinical %v absolute %v; Evaluate %s, safety %v", def.name, b.Name, s.Seed, c, a, s.Clinical, viol)
				}
			}
		}
	}
}

// The run (DT_OC_CLINICAL=1; protocol §7).
func TestClinicalOperatingCharacteristics(t *testing.T) {
	if os.Getenv("DT_OC_CLINICAL") == "" {
		t.Skip("set DT_OC_CLINICAL=1 for the combined clinical operating characteristics (docs/calibration/clinical-5c4b/protocol.md)")
	}
	outs := ocClinicalCompute(t, ocClinicalScenarios(), 20261101, 0)
	doc := map[string]any{
		"schema":    "dreamteam.oc-clinical/v1",
		"stage":     "exploratory: nothing selected; the owner chooses the v0.2-eval-5 values (protocol §1)",
		"protocol":  "docs/calibration/clinical-5c4b/protocol.md",
		"streams":   "seed i of scenario k: PCG(20261101, 100000·k + i)",
		"policy":    "v0.2-eval-4 with the candidate clinical rules (severe after 1,440 min past due, inherited work from the takeover, the shared 10% allowance, access cohorts at the breach gates' minimums, the maximum wait seed-level, the sampled age limit reported only)",
		"bundles":   ocBundles,
		"scenarios": outs,
	}
	raw, err := json.MarshalIndent(doc, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("../../docs/evaluation/oc-v0.2-eval-5-exploratory.json", append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}
