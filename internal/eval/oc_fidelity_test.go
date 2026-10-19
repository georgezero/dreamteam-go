package eval

import (
	"cmp"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// The final calibration correction of Slice 5C.4b (docs/calibration/
// clinical-5c4b/fidelity-protocol.md): the synthetic practice made
// operationally like the simulator where the clinical delay tails depend
// on it. Its demand by weekday and priority, acquisition hours,
// modalities, documentation and read times come from the simulator's
// measured profile (testdata/sim-profile-5c4b.json, written by
// internal/sim TestSimulatorClinicalProfile); it reads with individual
// readers in the simulator's reading blocks, with stochastic read times,
// the shift-end rule, overrun debt and the urgent_then_fifo worklist.
// The earlier models (ocPractice) and their results are unchanged.

const simProfilePath = "testdata/sim-profile-5c4b.json"

// simProfileData is the part of the simulator profile the model uses.
type simProfileData struct {
	Days     [7]int                    `json:"daysByWeekday"`
	Orders   map[string][7]int         `json:"ordersByWeekday"`
	Acquired map[string][7]int         `json:"acquiredByWeekday"`
	Hours    map[string][24]int        `json:"acquisitionsByHour"`
	Modality map[string]map[string]int `json:"acquiredByPriorityModality"`
	ReadSec  map[string][]int64        `json:"readSecPercentiles"`
	EntrySec map[string][]int64        `json:"entrySecPercentiles"`
	Readers  []struct {
		Name         string         `json:"name"`
		AllocationBP int64          `json:"siteAllocationBP"`
		Shifts       [2][7][2]int64 `json:"shiftsMin"`
	} `json:"readers"`
	PerSeed []struct {
		Acquired    int     `json:"acquiredInWindow"`
		Utilization float64 `json:"readingUtilization"`
	} `json:"perSeed"`
}

var simProfile = sync.OnceValues(func() (*simProfileData, error) {
	raw, err := os.ReadFile(simProfilePath)
	if err != nil {
		return nil, err
	}
	var p simProfileData
	return &p, json.Unmarshal(raw, &p)
})

func mustProfile(t testing.TB) *simProfileData {
	t.Helper()
	p, err := simProfile()
	if err != nil {
		t.Fatal(err)
	}
	return p
}

var simModalities = []string{"XR", "US", "CT", "MR"}

// Derived demand. windowAcquired: the simulator's acquisitions in the
// gated window (takeover to horizon, 150 days on the same calendar as
// every synthetic record), a seed's mean; weeklyAcquired, a week's.
func (p *simProfileData) windowAcquired() float64 {
	s := 0.0
	for _, x := range p.PerSeed {
		s += float64(x.Acquired)
	}
	return s / float64(len(p.PerSeed))
}

func (p *simProfileData) weeklyAcquired() float64 { return p.windowAcquired() / 150 * 7 }

// orderRate: the simulator's orders received a day on each weekday
// (Monday 0), all priorities.
func (p *simProfileData) orderRate() [7]float64 {
	var r [7]float64
	for d := range 7 {
		for _, q := range Priorities {
			r[d] += float64(p.Orders[q][d])
		}
		if p.Days[d] > 0 {
			r[d] /= float64(p.Days[d])
		}
	}
	return r
}

// rawShares: each weekday's share of the window's acquisitions (the
// same calendar window as every synthetic record's).
func (p *simProfileData) rawShares() [7]float64 {
	var w [7]float64
	s := 0.0
	for d := range 7 {
		for _, q := range Priorities {
			w[d] += float64(p.Acquired[q][d])
		}
		s += w[d]
	}
	for d := range w {
		w[d] /= s
	}
	return w
}

func (p *simProfileData) mix() [3]float64 {
	var c [3]float64
	s := 0.0
	for k, q := range Priorities {
		for _, n := range p.Acquired[q] {
			c[k] += float64(n)
		}
		s += c[k]
	}
	for k := range c {
		c[k] /= s
	}
	return c
}

// shares: per-day volume of each weekday as a share of the week (counts
// over the number of such days), from by.
func (p *simProfileData) shares(by map[string][7]int) [7]float64 {
	var w [7]float64
	s := 0.0
	for d := range 7 {
		for _, q := range Priorities {
			w[d] += float64(by[q][d])
		}
		if p.Days[d] > 0 {
			w[d] /= float64(p.Days[d])
		}
		s += w[d]
	}
	for d := range w {
		w[d] /= s
	}
	return w
}

// meanReadSec is the mean read time of an acquisition (mix and modality
// mixes weighted; percentile tables as uniform quantiles).
func (p *simProfileData) meanReadSec() float64 {
	mix := p.mix()
	m := 0.0
	for k, q := range Priorities {
		tot := 0
		for _, n := range p.Modality[q] {
			tot += n
		}
		for mod, n := range p.Modality[q] {
			m += mix[k] * float64(n) / float64(tot) * tableMean(p.ReadSec[mod])
		}
	}
	return m
}

func tableMean(xs []int64) float64 {
	s := 0.0
	for i := 1; i < len(xs); i++ {
		s += float64(xs[i]+xs[i-1]) / 2
	}
	return s / float64(len(xs)-1)
}

// tableDraw is a draw from a percentile table (linear between points).
func tableDraw(xs []int64, u float64) float64 {
	x := u * float64(len(xs)-1)
	i := min(int(x), len(xs)-2)
	f := x - float64(i)
	return float64(xs[i]) + f*float64(xs[i+1]-xs[i])
}

// ocBlock is a span a reader may claim in: [from, to), its shift ending
// at shiftEnd (a claim's longest read must end by it).
type ocBlock struct{ from, to, shiftEnd int64 }

type ocReader struct {
	blocks     []ocBlock
	urgentOnly bool
	handoff    int64 // a remote lane's handoff before reading
	general    bool  // its block time counts as general reading capacity
	core       bool  // one of the simulator's rota radiologists
}

// ocFidelity is the corrected synthetic practice. The zero value of every
// departure is the stable stationary practice.
type ocFidelity struct {
	Seasonal bool    `json:"seasonal"`
	DayCV    float64 `json:"dailyArrivalCV"`
	// Util: the target utilization of the core reading capacity (the
	// simulator's rota radiologists' blocks), which sets the read-time
	// scale; 0 reads at the simulator's own read times. Weekend readers
	// and lanes add coverage on top of it.
	Util float64 `json:"coreReadingUtilization"`
	// Coverage (ocReaders): "" stable, "satgap", "rota".
	Coverage string `json:"readingCoverage,omitempty"`
	// EqualOverall: the read-time scale is set instead so that the
	// practice's general utilization (all its general readers) is the
	// stable practice's at Util (equal realized utilization).
	EqualOverall bool    `json:"equalRealizedUtilization,omitempty"`
	ImgUtil      float64 `json:"imagingUtilization,omitempty"` // 0.80 when 0
	ReadCap      []ocCap `json:"readingCapacity,omitempty"`
	ReadErode    ocErode `json:"readingErosion,omitzero"`
	ImgCap       []ocCap `json:"imagingCapacity,omitempty"`
	ImgErode     ocErode `json:"imagingErosion,omitzero"`
	Worklist     string  `json:"worklist,omitempty"` // "": urgent_then_fifo; "strict": urgent, soon, routine
	// Takeover: the takeover instant (0: t0); fidAlignedTakeover aligns the
	// evaluator's weeks with the synthetic calendar's.
	Takeover int64 `json:"takeoverSec,omitempty"`
}

func (q ocFidelity) origin() int64 { return cmp.Or(q.Takeover, t0) }

const fidWarmup = 181

// fidImgAdjust: the imaging slot capacity of each weekday (Monday 0) is
// its share of the simulator's acquisitions times this factor (the
// factors weighted by the shares sum to 1, so the week's slots keep the
// imaging utilization). Fitted on the debug stream PCG(1, ·) so that the
// practice's acquisitions by weekday reproduce the simulator's (booking
// does not depend on reading, so one fit serves every coverage).
var fidImgAdjust = [7]float64{0.8624, 0.8570, 0.8665, 0.8982, 1.5282, 0.8434, 0.9527}

// slots is day d's imaging slot capacity.
func (q ocFidelity) slots(weekly float64, acqShare [7]float64, d int) int {
	wd := fidWeekday(d)
	return int(math.Floor(weekly*acqShare[wd]*fidImgAdjust[wd]/cmpOr(q.ImgUtil, 0.80)*capFactor(d, q.ImgCap, q.ImgErode) + 0.5))
}

// fidelityDate: the synthetic calendar (the takeover Tue 2026-10-13
// 06:00, as every synthetic record).
func fidWeekday(d int) int {
	date, _ := simtime.ParseDate(t0Date)
	return (int(date.AddDays(int64(d)).Weekday()) + 6) % 7
}

func fidMidnight(d int) int64 { return fidMidnightAt(t0, d) }

// fidMidnightAt: day d's midnight for a takeover (06:00 on day 0) at o.
func fidMidnightAt(o int64, d int) int64 { return o - 6*hour + int64(d)*daySec }

// fidAlignedTakeover is a takeover instant at 06:00 on a day the
// evaluator's Monday-based weeks (floorDiv(sec/day + 3, 7)) count as a
// Tuesday, as the synthetic calendar's takeover date is: with it the
// evaluator's weeks are the synthetic calendar's weeks. t0 (1e9 s) falls
// in an evaluator week that starts on the synthetic Wednesday at about
// 04:13.
const fidAlignedTakeover = int64(11576)*86400 + 6*3600

// rotaWeekOf: 0 in rota week A (the scenario's rotaWeekA, Mon
// 2026-10-12, and every second week), 1 in week B.
func rotaWeekOf(d int) int {
	// Day 0 is Tue 2026-10-13, in week A (its Monday is day −1).
	return int(floorDiv(int64(d+1), 7)) & 1
}

// ocReaders are the coverage's readers, with their blocks from day from
// to day to (exclusive), in employee order (remote lanes last):
//
//   - "rota": the simulator's radiologists (the profile's shifts by rota
//     week, reading blocks of their site allocation: 100-minute cycles,
//     the allocation's share reading).
//   - "satgap": the rota and an urgent-only weeknight lane pair, Mon–Fri
//     18:00 to 08:00 (the simulator's teleradiology weeknights window,
//     two lanes, a 10-minute handoff): the E-family practice.
//   - "" (stable): the satgap practice and weekend reading, a reader
//     Saturday 14:00–18:00 and Sunday 08:00–12:00.
func ocReaders(p *simProfileData, coverage string, from, to int) []ocReader {
	return ocReadersAt(p, coverage, from, to, t0)
}

// ocReadersAt is ocReaders for a takeover at o.
func ocReadersAt(p *simProfileData, coverage string, from, to int, o int64) []ocReader {
	fidMidnight := func(d int) int64 { return fidMidnightAt(o, d) }
	var rs []ocReader
	for _, r := range p.Readers {
		x := ocReader{general: true, core: true}
		for d := from; d < to; d++ {
			s := r.Shifts[rotaWeekOf(d)][fidWeekday(d)]
			if s[1] == 0 {
				continue
			}
			m := fidMidnight(d)
			end := m + s[1]*60
			site := 100 * r.AllocationBP / 10000
			for at := s[0]; at < s[1]; at += 100 {
				x.blocks = append(x.blocks, ocBlock{m + at*60, min(m+(at+site)*60, end), end})
			}
		}
		rs = append(rs, x)
	}
	if coverage == "" {
		x := ocReader{general: true}
		for d := from; d < to; d++ {
			m := fidMidnight(d)
			switch fidWeekday(d) {
			case 5:
				x.blocks = append(x.blocks, ocBlock{m + 14*hour, m + 18*hour, m + 18*hour})
			case 6:
				x.blocks = append(x.blocks, ocBlock{m + 8*hour, m + 12*hour, m + 12*hour})
			}
		}
		rs = append(rs, x)
	}
	if coverage != "rota" {
		for range 2 {
			x := ocReader{urgentOnly: true, handoff: 600}
			for d := from; d < to; d++ {
				if fidWeekday(d) < 5 {
					m := fidMidnight(d)
					x.blocks = append(x.blocks, ocBlock{m + 18*hour, m + 32*hour, math.MaxInt64})
				}
			}
			rs = append(rs, x)
		}
	}
	return rs
}

// generalBlockSec: the general readers' block time in a fortnight (both
// rota weeks), per week.
func generalBlockSec(p *simProfileData, coverage string) float64 {
	return blockSecIn(p, coverage, -1, 13) / 2
}

// blockSecIn: the general readers' block time in days [from, to) after the
// takeover date, within [takeover, horizon).
func blockSecIn(p *simProfileData, coverage string, from, to int) float64 {
	s := 0.0
	lo, hi := int64(math.MinInt64), int64(math.MaxInt64)
	if from >= 0 {
		lo, hi = t0, t0+150*daySec
	}
	for _, r := range ocReaders(p, coverage, from, to) {
		if r.general {
			for _, b := range r.blocks {
				s += float64(max(0, min(b.to, hi)-max(b.from, lo)))
			}
		}
	}
	return s
}

// scale is the read-time multiplier: the simulator's read times, scaled
// so that the expected workload of the gated window (the simulator's
// acquisitions there at their mean read time) fills Util of the general
// block time offered in it.
func (q ocFidelity) scale(p *simProfileData) float64 {
	if q.Util == 0 {
		return 1
	}
	u := q.Util
	if q.EqualOverall {
		u *= blockSecIn(p, "rota", 0, 151) / blockSecIn(p, "", 0, 151) * blockSecIn(p, q.Coverage, 0, 151) / blockSecIn(p, "rota", 0, 151)
	}
	return u * blockSecIn(p, "rota", 0, 151) / (p.windowAcquired() * p.meanReadSec())
}

// fidStudy: a study's modality and documentation time beside its record.
type fidStudy struct {
	modality string
	dur      float64 // read time drawn (seconds, unscaled)
}

// record is recordPaired with one stream for demand and everything else.
func (q ocFidelity) record(r ocRNG, pol *Policy) (*Record, fidBusy) { return q.recordPaired(r, r, pol) }

// recordPaired builds the practice's record with the demand (order
// arrivals, priorities, receipt times, wanted times and day factors)
// drawn from rd and everything else (booking choices, cancellations,
// modalities, documentation and read times) from rb. With rd and rb the
// same stream it is the committed model's record; with a separate demand
// stream, practices that differ in capacity receive the same eligible
// demand, as the simulator's do (TestDemandExogenousAcrossPlans), so the
// lost-demand gate pairs them.
func (q ocFidelity) recordPaired(rd, rb ocRNG, pol *Policy) (*Record, fidBusy) {
	p, err := simProfile()
	if err != nil {
		panic(err)
	}
	start := q.origin()
	fidMidnight := func(d int) int64 { return fidMidnightAt(start, d) }
	date, _ := simtime.ParseDate(t0Date)
	rec := &Record{Schema: RecordSchema, Plan: "P", Engine: "oc", ScenarioFingerprint: "fp", TATTargetMin: testTargets,
		TakeoverSec: start, TakeoverDate: t0Date, HorizonEndSec: start + 150*daySec, AppointmentsSHA256: "oc"}
	rec.ObservedUntilSec = observationNeeded(pol, rec)
	E := rec.ObservedUntilSec
	days := int((E - start) / daySec)
	w := fidWarmup
	mix := p.mix()
	weekly := p.weeklyAcquired()
	acqShare := p.shares(p.Acquired)
	// Orders a day by weekday: the simulator's order rates, scaled so that
	// the window's acquisitions (after the routine cancellations) are the
	// simulator's: the same calendar, so the same weekday composition.
	rate := p.orderRate()
	winOrders := 0.0
	for d := range 150 {
		winOrders += rate[fidWeekday(d)]
	}
	kappa := p.windowAcquired() / (winOrders * (1 - mix[2]*0.05))
	// Imaging slots by weekday: the acquisitions' weekday shares at
	// imaging utilization ImgUtil.
	capOf := make([]int, w+days+300)
	for i := range capOf {
		capOf[i] = q.slots(weekly, acqShare, i-w)
	}
	booked := make([]int, len(capOf))
	routineCap := func(d int) int { return int(math.Floor(0.85 * float64(capOf[d+w]))) }
	// Order receipt hours: the scenario's hourProfileBP, 07:00–19:00.
	recvHours := []float64{400, 900, 1100, 1100, 1000, 700, 900, 1000, 1000, 900, 600, 400}
	hs := 0.0
	for _, x := range recvHours {
		hs += x
	}
	// Appointment slots: each day's slots at the quantiles of the
	// simulator's acquisition hours (weekday or Saturday), within the
	// imaging hours; a booking takes a free slot at random among those at
	// or after its earliest time.
	slotTimes := make([][]int64, len(capOf))
	free := make([][]bool, len(capOf))
	slotsOf := func(d int) []int64 {
		i := d + w
		if slotTimes[i] == nil {
			n := capOf[i]
			hp := p.Hours["weekday"]
			if fidWeekday(d) == 5 {
				hp = p.Hours["saturday"]
			}
			op, cl := imagingHours(fidWeekday(d))
			tot := 0.0
			for h := op; h < cl; h++ {
				tot += float64(hp[h])
			}
			ts := make([]int64, n)
			for j := range n {
				u := (float64(j) + 0.5) / float64(n) * tot
				h := op
				for ; h < cl-1 && u >= float64(hp[h]); h++ {
					u -= float64(hp[h])
				}
				ts[j] = fidMidnight(d) + int64(h)*hour + int64(min(u/float64(max(hp[h], 1)), 1)*3600)
			}
			slotTimes[i], free[i] = ts, make([]bool, n)
			for j := range free[i] {
				free[i][j] = true
			}
		}
		return slotTimes[i]
	}
	// freeAfter: day d's free slots at or after lo.
	freeAfter := func(d int, lo int64) []int {
		var js []int
		for j, at := range slotsOf(d) {
			if free[d+w][j] && at >= lo {
				js = append(js, j)
			}
		}
		return js
	}
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
		wd := fidWeekday(day)
		if clock == 0 && q.DayCV > 0 {
			k := 1 / (q.DayCV * q.DayCV)
			dayFactor = rd.gamma(k) / k
		}
		if clock < 7 || clock >= 19 {
			continue
		}
		lam := rate[wd] * kappa * recvHours[clock-7] / hs * dayFactor
		if q.Seasonal {
			lam *= ocSeason[dt.Month-1]
		}
		for range rd.poisson(lam) {
			u := rd.Float64()
			k := 2
			switch {
			case u < mix[0]:
				k = 0
			case u < mix[0]+mix[1]:
				k = 1
			}
			pri := Priorities[k]
			recv := at + int64(rd.Float64()*3600)
			var lead float64
			switch k {
			case 1:
				lead = 2 * rd.Float64()
			case 2:
				lead = leadDays(rd.Float64())
			}
			wantDay := day + int(lead)
			wanted := max(recv, fidMidnight(wantDay))
			o := Order{ID: "o", Priority: pri, Modality: "XR", ReceivedSec: recv, WantedSec: wanted}
			bookDay, found := 0, false
			var cands []int
			for d := wantDay; d <= wantDay+ocTolerance[pri] && d+w < len(capOf); d++ {
				_, cl := imagingHours(fidWeekday(d))
				if cl == 0 || fidMidnight(d)+int64(cl)*hour <= wanted {
					continue
				}
				limit := capOf[d+w]
				if k == 2 {
					limit = routineCap(d)
				}
				if booked[d+w] < limit {
					if cands = freeAfter(d, wanted); len(cands) > 0 {
						bookDay, found = d, true
						break
					}
				}
			}
			out, res := OutOpen, int64(0)
			if !found {
				out, res = OutLost, recv
			} else {
				booked[bookDay+w]++
				j := cands[rb.IntN(len(cands))]
				free[bookDay+w][j] = false
				appt := slotTimes[bookDay+w][j]
				o.FirstApptSec = appt
				switch {
				case k == 2 && rb.Float64() < 0.05:
					c := recv + int64(rb.Float64()*float64(appt-recv))
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
	{
		var os []Order
		var oo []string
		for i, o := range rec.Orders {
			if o.ReceivedSec > start || o.ResolvedSec == 0 || o.ResolvedSec > start {
				os, oo = append(os, o), append(oo, outs[i])
			}
		}
		rec.Orders, outs = os, oo
	}
	// Studies: modality by priority, documentation time by modality, read
	// time drawn at acquisition (the simulator's: one draw per case).
	slices.SortStableFunc(acqs, func(a, b acq) int { return int(a.at - b.at) })
	var info []fidStudy
	for _, a := range acqs {
		q := Priorities[a.k]
		tot := 0
		for _, m := range simModalities {
			tot += p.Modality[q][m]
		}
		u := rb.Float64() * float64(tot)
		mod := simModalities[len(simModalities)-1]
		for _, m := range simModalities {
			if u -= float64(p.Modality[q][m]); u < 0 {
				mod = m
				break
			}
		}
		entered := a.at + int64(tableDraw(p.EntrySec[mod], rb.Float64()))
		rec.Studies = append(rec.Studies, Study{ID: "s", Priority: q, Modality: mod, AcquiredSec: a.at, EnteredSec: entered})
		info = append(info, fidStudy{mod, tableDraw(p.ReadSec[mod], rb.Float64())})
	}
	// Studies enter the worklist in entry order.
	order := make([]int, len(rec.Studies))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int { return int(rec.Studies[a].EnteredSec - rec.Studies[b].EnteredSec) })
	busy := q.read(p, rec, info, order, -w, days+1)
	for i := range rec.Studies {
		if s := &rec.Studies[i]; s.FinalSec > E {
			s.ClaimedSec, s.ReadStartSec, s.FinalSec = 0, 0, 0
			busy.general[i], busy.core[i] = 0, 0
		}
	}
	keep := make([]bool, len(rec.Studies))
	for i, s := range rec.Studies {
		keep[i] = !(s.AcquiredSec <= start && s.FinalSec != 0 && s.FinalSec <= start)
	}
	var ss []Study
	var bs fidBusy
	for i, s := range rec.Studies {
		if keep[i] {
			ss = append(ss, s)
			bs.general, bs.core = append(bs.general, busy.general[i]), append(bs.core, busy.core[i])
		}
	}
	rec.Studies = ss
	fidSamples(rec, outs, start, days)
	return rec, bs
}

// fidBusy: each study's read time by a general reader, and by a core
// one (0 when read by another or unread).
type fidBusy struct{ general, core []float64 }

// read runs the readers from day from to day to over the studies (order:
// their indices in entry order).
func (q ocFidelity) read(p *simProfileData, rec *Record, info []fidStudy, order []int, from, to int) fidBusy {
	scale := q.scale(p)
	busy := fidBusy{make([]float64, len(rec.Studies)), make([]float64, len(rec.Studies))}
	maxRead := map[string]float64{}
	for _, m := range simModalities {
		xs := p.ReadSec[m]
		maxRead[m] = float64(xs[len(xs)-1])
	}
	rs := ocReadersAt(p, q.Coverage, from, to, q.origin())
	type state struct {
		bi       int   // current block
		readFrom int64 // when the reader may next claim
	}
	sts := make([]state, len(rs))
	for i := range rs {
		if len(rs[i].blocks) > 0 {
			sts[i].readFrom = rs[i].blocks[0].from
		}
	}
	// avail moves reader i to its first block time at or after f with
	// debt owed (repaid out of block time).
	avail := func(i int, f, debt int64) {
		bl := rs[i].blocks
		j := sts[i].bi
		for j < len(bl) && bl[j].to <= f {
			j++
		}
		if j == len(bl) {
			sts[i] = state{j, math.MaxInt64}
			return
		}
		at := max(f, bl[j].from) + debt
		for at >= bl[j].to {
			debt = at - bl[j].to
			j++
			if j == len(bl) {
				sts[i] = state{j, math.MaxInt64}
				return
			}
			at = bl[j].from + debt
		}
		sts[i] = state{j, at}
	}
	// outside is the time in [a, b) outside reader i's blocks, from block j.
	outside := func(i, j int, a, b int64) int64 {
		in := int64(0)
		for bl := rs[i].blocks; j < len(bl) && bl[j].from < b; j++ {
			in += max(0, min(b, bl[j].to)-max(a, bl[j].from))
		}
		return (b - a) - in
	}
	var open []int // in entry order
	next := 0
	for {
		// The reader with the earliest claim time (employee order on ties).
		i := -1
		for k := range rs {
			if sts[k].readFrom != math.MaxInt64 && (i < 0 || sts[k].readFrom < sts[i].readFrom) {
				i = k
			}
		}
		if i < 0 {
			break
		}
		t := sts[i].readFrom
		for next < len(order) && rec.Studies[order[next]].EnteredSec <= t {
			open = append(open, order[next])
			next++
		}
		b := rs[i].blocks[sts[i].bi]
		day := int(floorDiv(t-(q.origin()-6*hour), daySec))
		f := 1 / max(0.05, capFactor(day, q.ReadCap, q.ReadErode))
		pick := -1
		fits := func(s int) bool {
			return rs[i].urgentOnly || float64(t)+maxRead[info[s].modality]*scale*f <= float64(b.shiftEnd)
		}
		if q.Worklist == "strict" {
			for k := 0; k < 3 && pick < 0; k++ {
				for j, s := range open {
					if rec.Studies[s].Priority == Priorities[k] && (!rs[i].urgentOnly || k == 0) && fits(s) {
						pick = j
						break
					}
				}
			}
		} else {
			first := -1
			for j, s := range open {
				x := &rec.Studies[s]
				urgent := x.Priority == "urgent"
				if (rs[i].urgentOnly && !urgent) || !fits(s) {
					continue
				}
				if urgent || t-x.AcquiredSec >= target(rec, x.Priority) {
					pick = j
					break
				}
				if first < 0 {
					first = j
				}
			}
			if pick < 0 {
				pick = first
			}
		}
		if pick < 0 {
			// Idle until the next entry within this block, or the next block.
			if next < len(order) && rec.Studies[order[next]].EnteredSec < b.to {
				sts[i].readFrom = max(t+1, rec.Studies[order[next]].EnteredSec)
			} else {
				sts[i].bi++
				if sts[i].bi == len(rs[i].blocks) {
					sts[i].readFrom = math.MaxInt64
				} else {
					sts[i].readFrom = rs[i].blocks[sts[i].bi].from
				}
			}
			continue
		}
		s := open[pick]
		open = slices.Delete(open, pick, pick+1)
		dur := int64(max(1, math.Round(info[s].dur*scale*f)))
		x := &rec.Studies[s]
		x.ClaimedSec, x.ReadStartSec = t, t+rs[i].handoff
		x.FinalSec = x.ReadStartSec + dur
		if rs[i].general {
			busy.general[s] = float64(dur)
		}
		if rs[i].core {
			busy.core[s] = float64(dur)
		}
		fin := x.FinalSec
		if rs[i].urgentOnly {
			sts[i].readFrom = fin
			if fin >= b.to {
				avail(i, fin, 0)
			}
			continue
		}
		avail(i, fin, outside(i, sts[i].bi, t, fin))
	}
	return busy
}

// fidSamples are the daily samples and the captures (the studies are in
// acquisition order; outs are the orders' outcomes).
func fidSamples(rec *Record, outs []string, start int64, days int) {
	date, _ := simtime.ParseDate(t0Date)
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
	for _, c := range []capAt{{"T30", start + 30*daySec}, {"T90", start + 90*daySec}, {"tailEnd", rec.HorizonEndSec}, {"T180", start + 180*daySec}, {"end", rec.ObservedUntilSec}} {
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
}

// ocFidExtra: a seed's measures beyond the gates' inputs.
type ocFidExtra struct {
	ocExtra
	satShare        float64    // acquisitions on Saturday
	weekdayShare    [7]float64 // acquisitions by weekday
	mix             [3]float64 // acquisition priority mix
	tkUnread        int64      // unread at the takeover
	tkOverdue       int64      // of them, past their target
	tkOrders        int64      // orders open at the takeover
	unreadEarly     float64    // mean unread at the daily samples, days 1–30
	unreadLate      float64    // days 121–150
	ageMaxP90       int64      // the daily samples' oldest unread age, p90 (minutes)
	satPMUrgent     int64      // worst past due of urgent studies acquired Saturday after 14:00 (minutes)
	scale           float64
	generalBlockSec float64
	core, coreLate  float64 // core utilization: the rota radiologists' read time over their block time
	load            float64 // the window's acquisitions' read time (any reader) over the core block time: the calibration target
	// breach: the window's acquisitions final (or observed) past their
	// target, by priority (the simulator profile's breachShare).
	breach [3]float64
}

func ocFidExtraOf(q ocFidelity, r *Record, busy fidBusy) ocFidExtra {
	p, _ := simProfile()
	x := ocFidExtra{scale: q.scale(p)}
	start, H := r.TakeoverSec, r.HorizonEndSec
	late := H - 30*daySec
	rs := ocReadersAt(p, q.Coverage, 0, 151, r.TakeoverSec)
	var blk, blkLate, cblk, cblkLate float64
	for _, rd := range rs {
		if !rd.general {
			continue
		}
		for _, b := range rd.blocks {
			a, e := max(b.from, start), min(b.to, H)
			if e > a {
				blk += float64(e - a)
				la := max(a, late)
				if e > la {
					blkLate += float64(e - la)
				}
				if rd.core {
					cblk += float64(e - a)
					if e > la {
						cblkLate += float64(e - la)
					}
				}
			}
		}
	}
	x.generalBlockSec = blk
	var rd, rdLate, cr, crLate, all, n float64
	var cnt [3]float64
	for i, s := range r.Studies {
		if s.AcquiredSec <= start {
			x.tkUnread++
			if start-s.AcquiredSec > target(r, s.Priority) {
				x.tkOverdue++
			}
			continue
		}
		if s.AcquiredSec > H {
			continue
		}
		n++
		if s.FinalSec != 0 {
			all += float64(s.FinalSec - s.ReadStartSec)
		}
		rd += busy.general[i]
		cr += busy.core[i]
		if s.AcquiredSec > late {
			rdLate += busy.general[i]
			crLate += busy.core[i]
		}
		d := int(floorDiv(s.AcquiredSec-(r.TakeoverSec-6*hour), daySec))
		wd := fidWeekday(d)
		x.weekdayShare[wd]++
		k := slices.Index(Priorities, s.Priority)
		cnt[k]++
		if fin := cmp.Or(s.FinalSec, r.ObservedUntilSec); fin-s.AcquiredSec > target(r, s.Priority) {
			x.breach[k]++
		}
		if s.Priority == "urgent" && wd == 5 && (s.AcquiredSec-fidMidnightAt(r.TakeoverSec, d)) >= 14*hour {
			fin := s.FinalSec
			if fin == 0 {
				fin = r.ObservedUntilSec
			}
			x.satPMUrgent = max(x.satPMUrgent, (fin-s.AcquiredSec-target(r, s.Priority))/60)
		}
	}
	x.util = ocUtil{read: rd / blk, readLate: rdLate / blkLate}
	x.core, x.coreLate, x.load = cr/cblk, crLate/cblkLate, all/cblk
	for k := range x.weekdayShare {
		x.weekdayShare[k] /= n
	}
	x.satShare = x.weekdayShare[5]
	for k := range cnt {
		x.mix[k] = cnt[k] / n
		x.breach[k] /= max(cnt[k], 1)
	}
	// Imaging slot utilization, lost orders, the unread trend.
	var slots, booked float64
	for i := range r.Orders {
		if r.Orders[i].ReceivedSec <= start {
			x.tkOrders++
		}
		if a := r.Orders[i].FirstApptSec; a > start && a <= H {
			booked++
		}
	}
	weekly, acqShare := p.weeklyAcquired(), p.shares(p.Acquired)
	for d := 0; d < 150; d++ {
		slots += float64(q.slots(weekly, acqShare, d))
	}
	x.util.imaging = booked / slots
	end := &r.Captures[len(r.Captures)-1]
	var no, lost float64
	for i, o := range r.Orders {
		if o.ReceivedSec > start && o.ReceivedSec <= H {
			no++
			if end.OrderOutcomes[i] == OutLost {
				lost++
			}
		}
	}
	x.lost = lost / max(no, 1)
	var ages []int64
	var e, l float64
	for _, s := range r.Samples {
		switch {
		case s.Day <= 30:
			e += float64(s.Unread) / 30
		case s.Day > 120 && s.Day <= 150:
			l += float64(s.Unread) / 30
		}
		if s.AtSec == H {
			x.unread = int64(s.Unread)
		}
	}
	x.unreadEarly, x.unreadLate = e, l
	// Oldest unread age at each daily sample of the gated period.
	for _, s := range r.Samples {
		if s.AtSec > H {
			continue
		}
		oldest := int64(0)
		for _, st := range r.Studies {
			if st.AcquiredSec > s.AtSec {
				break
			}
			if st.FinalSec == 0 || st.FinalSec > s.AtSec {
				oldest = max(oldest, (s.AtSec-st.AcquiredSec)/60)
			}
		}
		ages = append(ages, oldest)
	}
	slices.Sort(ages)
	x.ageMaxP90 = ages[int(math.Ceil(0.9*float64(len(ages))))-1]
	return x
}

type ocFidelityDef struct {
	name, desc, role, regime string
	q                        ocFidelity
	seeds                    int
}

// ocFidelityScenarios are the fidelity protocol's §5 scenarios, in stream
// order.
func ocFidelityScenarios() []ocFidelityDef {
	var defs []ocFidelityDef
	add := func(regime, name, desc, role string, seeds int, q ocFidelity) {
		defs = append(defs, ocFidelityDef{name, desc, role, regime, q, seeds})
	}
	for _, r := range []struct {
		regime string
		u      float64
	}{{"0.85", 0.85}, {"0.92", 0.92}} {
		s := strings.TrimPrefix(r.regime, "0.")
		role := "stable"
		if r.u > 0.9 {
			role = "stress"
		}
		base := ocFidelity{Util: r.u}
		with := func(f func(*ocFidelity)) ocFidelity { q := base; f(&q); return q }
		add(r.regime, "A"+s, "stationary", role, 2500, base)
		add(r.regime, "B"+s, "seasonal", role, 2500, with(func(q *ocFidelity) { q.Seasonal = true }))
		add(r.regime, "C"+s, "bursty (day CV 0.15)", role, 2500, with(func(q *ocFidelity) { q.DayCV = 0.15 }))
		add(r.regime, "Cs"+s, "seasonal and bursty", role, 2500, with(func(q *ocFidelity) { q.Seasonal, q.DayCV = true, 0.15 }))
		add(r.regime, "Wg"+s, "E3-like: no reading Saturday after 14:00 or Sunday, at A"+s+"'s staffing and read times (the weekend reader removed)", "structural", 1000,
			with(func(q *ocFidelity) { q.Coverage = "satgap" }))
		add(r.regime, "WgU"+s, "E3-like at A"+s+"'s realized general utilization (faster reading: the weekend's capacity moved to weekdays)", "structural", 1000,
			with(func(q *ocFidelity) { q.Coverage, q.EqualOverall = "satgap", true }))
		if r.u < 0.9 {
			add(r.regime, "R0"+s, "the simulator's rota alone (no lanes, no weekend reader), at the same core load", "structural", 1000,
				with(func(q *ocFidelity) { q.Coverage = "rota" }))
		}
		add(r.regime, "L10-"+s, "reading capacity 10% lower from day 60", "deterioration", 1000, with(func(q *ocFidelity) { q.ReadCap = []ocCap{{From: 60, Factor: 0.9}} }))
		if r.u < 0.9 {
			add(r.regime, "L20-"+s, "reading capacity 20% lower from day 60", "deterioration", 1000, with(func(q *ocFidelity) { q.ReadCap = []ocCap{{From: 60, Factor: 0.8}} }))
			add(r.regime, "D-"+s, "reading capacity 25% lower from day 30", "deterioration", 1000, with(func(q *ocFidelity) { q.ReadCap = []ocCap{{From: 30, Factor: 0.75}} }))
		}
		add(r.regime, "K-"+s, "reading capacity eroding 0.2% a day from day 30", "deterioration", 1000, with(func(q *ocFidelity) { q.ReadErode = ocErode{30, 0.002} }))
		add(r.regime, "IL10-"+s, "imaging capacity 10% lower from day 60", "deterioration", 1000, with(func(q *ocFidelity) { q.ImgCap = []ocCap{{From: 60, Factor: 0.9}} }))
		add(r.regime, "IK-"+s, "imaging capacity eroding 0.2% a day from day 30", "deterioration", 1000, with(func(q *ocFidelity) { q.ImgErode = ocErode{30, 0.002} }))
	}
	add("simulator", "M", "the simulator mirror: its rota alone at its own read times (validation against the simulator profile)", "validation", 1000,
		ocFidelity{Coverage: "rota"})
	return defs
}

type ocFidSummary struct {
	ScaleOfReadTimes  float64                `json:"readTimeScale"`
	SaturdayShare     ocFloatDist            `json:"saturdayAcquisitionShare"`
	WeekdayShare      [7]float64             `json:"acquisitionWeekdayShareMean"`
	Mix               [3]float64             `json:"acquisitionMixMean"` // urgent, soon, routine
	TakeoverUnread    Dist                   `json:"takeoverUnread"`
	TakeoverOverdue   Dist                   `json:"takeoverOverdue"`
	TakeoverOrders    Dist                   `json:"takeoverOpenOrders"`
	UnreadEarly       ocFloatDist            `json:"meanUnreadDays1To30"`
	UnreadLate        ocFloatDist            `json:"meanUnreadDays121To150"`
	UnreadDrift       ocFloatDist            `json:"unreadDriftLateMinusEarly"`
	AgeMaxP90         Dist                   `json:"sampleOldestUnreadAgeP90Min"`
	SatPMUrgent       Dist                   `json:"urgentAcquiredSaturdayAfter14WorstPastDueMin"`
	SevereShareDetail map[string]Dist        `json:"severeDelaysPerSeed"`
	BreachShare       map[string]ocFloatDist `json:"breachShare"`
	CoreLoad          ocFloatDist            `json:"coreLoad"`
	CoreUtilization   ocFloatDist            `json:"coreReadersUtilization"`
}

type ocFidOut struct {
	ocCorrectedOut
	Fidelity ocFidelity   `json:"fidelityPractice"`
	Summary  ocFidSummary `json:"fidelity"`
}

func ocFidelityCompute(t *testing.T, defs []ocFidelityDef, stream uint64, names []string) []ocFidOut {
	base := ocClinicalPolicy(t, ocBundles[1])
	p := mustProfile(t)
	var outs []ocFidOut
	for k, def := range defs {
		if len(names) > 0 && !slices.Contains(names, def.name) {
			continue
		}
		n := def.seeds
		seeds, extras, fx, studies := make([]ocSeed, n), make([]ocExtra, n), make([]ocFidExtra, n), make([]int, n)
		severe := make([]map[string]int64, n)
		var wg sync.WaitGroup
		work := make(chan int)
		for range runtime.NumCPU() {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := range work {
					rec, busy := def.q.record(ocRNG{rand.New(rand.NewPCG(stream, uint64(100000*k+i)))}, base)
					rec.Seed = int64(i)
					seeds[i], studies[i] = ocExtract(base, rec), len(rec.Studies)
					fx[i] = ocFidExtraOf(def.q, rec, busy)
					extras[i] = fx[i].ocExtra
					severe[i] = map[string]int64{}
					for _, q := range Priorities {
						severe[i][q] = seeds[i].severe[q+"/"+WinGated].num
					}
				}
			}()
		}
		for i := range n {
			work <- i
		}
		close(work)
		wg.Wait()
		sd := ocScenarioDef{def.name, def.desc, def.role, ocPractice{}, def.seeds}
		cd := ocCorrectedDef{sd, def.regime}
		o := ocFidOut{ocCorrectedOut: ocDetail(base, cd, seeds, extras, ocTabulate(base, sd, seeds, studies)), Fidelity: def.q}
		o.Practice = ocPractice{}
		var sat, ue, ul, drift []float64
		var tu, to, tord, age, satpm []int64
		sum := ocFidSummary{ScaleOfReadTimes: def.q.scale(p), SevereShareDetail: map[string]Dist{}, BreachShare: map[string]ocFloatDist{}}
		brs := map[string][]float64{}
		var loads, cores []float64
		sev := map[string][]int64{}
		for i, x := range fx {
			sat, ue, ul, drift = append(sat, x.satShare), append(ue, x.unreadEarly), append(ul, x.unreadLate), append(drift, x.unreadLate-x.unreadEarly)
			tu, to, tord, age, satpm = append(tu, x.tkUnread), append(to, x.tkOverdue), append(tord, x.tkOrders), append(age, x.ageMaxP90), append(satpm, x.satPMUrgent)
			for w := range 7 {
				sum.WeekdayShare[w] += x.weekdayShare[w] / float64(n)
			}
			for k := range 3 {
				sum.Mix[k] += x.mix[k] / float64(n)
			}
			for k, q := range Priorities {
				sev[q] = append(sev[q], severe[i][q])
				brs[q] = append(brs[q], x.breach[k])
			}
			loads, cores = append(loads, x.load), append(cores, x.core)
		}
		sum.CoreLoad, sum.CoreUtilization = floatDist(loads), floatDist(cores)
		for _, q := range Priorities {
			sum.BreachShare[q] = floatDist(brs[q])
		}
		sum.SaturdayShare, sum.UnreadEarly, sum.UnreadLate, sum.UnreadDrift = floatDist(sat), floatDist(ue), floatDist(ul), floatDist(drift)
		sum.TakeoverUnread, sum.TakeoverOverdue, sum.TakeoverOrders, sum.AgeMaxP90, sum.SatPMUrgent = dist(tu), dist(to), dist(tord), dist(age), dist(satpm)
		for _, q := range Priorities {
			sum.SevereShareDetail[q] = dist(sev[q])
		}
		o.Summary = sum
		outs = append(outs, o)
	}
	return outs
}

// The simulator profile behind the model is the committed one, and the
// derived demand is what the protocol states.
func TestSimProfileDerived(t *testing.T) {
	p := mustProfile(t)
	if len(p.Readers) != 2 || len(p.PerSeed) != 10 {
		t.Fatalf("profile: %d readers, %d seeds", len(p.Readers), len(p.PerSeed))
	}
	acq := p.shares(p.Acquired)
	if acq[5] < 0.04 || acq[5] > 0.08 || acq[6] != 0 {
		t.Errorf("simulator Saturday acquisition share %.4f, Sunday %.4f", acq[5], acq[6])
	}
	if m := p.mix(); math.Abs(m[0]+m[1]+m[2]-1) > 1e-12 {
		t.Errorf("mix %v", m)
	}
}

// The fidelity practices meet their core load and reproduce the
// simulator's weekday and priority distribution, on the debug stream
// PCG(1, ·), and their records are valid with inherited work.
func TestOCFidelityCalibration(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	base := ocClinicalPolicy(t, ocBundles[1])
	p := mustProfile(t)
	acq, mix := p.rawShares(), p.mix()
	for _, c := range []struct {
		q    ocFidelity
		want float64
	}{{ocFidelity{Util: 0.85}, 0.85}, {ocFidelity{Util: 0.92}, 0.92}, {ocFidelity{Util: 0.85, Coverage: "satgap"}, 0.85}} {
		const n = 3
		var u, sat float64
		var m [3]float64
		booked := 0
		for i := range n {
			rec, busy := c.q.record(ocRNG{rand.New(rand.NewPCG(1, uint64(i)))}, base)
			if ms := Measure(base, rec); len(ms.Integrity) > 0 {
				t.Fatalf("%+v: %v", c.q, ms.Integrity)
			}
			for _, o := range rec.Orders {
				if o.ReceivedSec <= rec.TakeoverSec {
					if o.ResolvedSec != 0 && o.ResolvedSec <= rec.TakeoverSec {
						t.Fatalf("%+v: an inherited order resolved before the takeover", c.q)
					}
					booked++
				}
			}
			x := ocFidExtraOf(c.q, rec, busy)
			u, sat = u+x.load/n, sat+x.satShare/n
			for k := range m {
				m[k] += x.mix[k] / n
			}
		}
		if math.Abs(u-c.want) > 0.02 {
			t.Errorf("%+v: measured core load %.4f, want %.2f ± 0.02", c.q, u, c.want)
		}
		if math.Abs(sat-acq[5]) > 0.01 {
			t.Errorf("%+v: Saturday share %.4f, the simulator's %.4f", c.q, sat, acq[5])
		}
		for k := range m {
			if math.Abs(m[k]-mix[k]) > 0.01 {
				t.Errorf("%+v: %s share %.4f, the simulator's %.4f", c.q, Priorities[k], m[k], mix[k])
			}
		}
		if booked == 0 {
			t.Errorf("%+v: no inherited order", c.q)
		}
	}
}

// The individual-reader rules: blocks of the allocation, the shift-end
// rule, overrun debt repaid out of later block time, urgent-only lanes,
// the worklist.
func TestOCReaders(t *testing.T) {
	p := mustProfile(t)
	rota := ocReaders(p, "rota", 0, 1) // day 0: Tuesday, week A
	if len(rota) != 2 {
		t.Fatalf("%d readers", len(rota))
	}
	// Avery 07:30–16:00 at 60%: blocks of 60 minutes every 100, the last
	// cut at the shift end.
	av := rota[0].blocks
	m := fidMidnight(0)
	if len(av) != 6 || av[0] != (ocBlock{m + 450*60, m + 510*60, m + 960*60}) || av[5] != (ocBlock{m + 950*60, m + 960*60, m + 960*60}) {
		t.Fatalf("Avery's blocks %+v", av)
	}
	if g := generalBlockSec(p, "rota") / 60; g < 2900 || g > 3100 {
		t.Errorf("rota block minutes a week %v", g)
	}
	if generalBlockSec(p, "")-generalBlockSec(p, "satgap") != 8*3600 {
		t.Error("the stable practice's weekend reading is not 8 hours")
	}
	// One routine study at 09:00 on day 0 read by Avery at the simulator's
	// time: claimed at 09:00 inside the block 09:10? Avery's blocks are
	// 07:30–08:30, 09:10–10:10.
	rec := &Record{TATTargetMin: testTargets}
	mk := func(q, mod string, at int64) Study {
		return Study{Priority: q, Modality: mod, AcquiredSec: at, EnteredSec: at}
	}
	rec.Studies = []Study{mk("routine", "XR", m+9*hour), mk("routine", "MR", m+8*hour+25*60), mk("urgent", "XR", m+21*hour)}
	info := []fidStudy{{"XR", 100}, {"MR", 1400}, {"XR", 200}}
	q := ocFidelity{Coverage: "satgap"}
	busy := q.read(p, rec, info, []int{1, 0, 2}, 0, 1)
	s := rec.Studies
	// The MR (longest read 1,499 s) at 08:25 fits before Avery's shift end
	// but she reads it from 08:25 to 08:48:20, 18:20 past her block: the
	// debt delays her next block (09:10) to 09:28:20.
	if s[1].ClaimedSec != m+8*hour+25*60 || s[1].FinalSec != m+8*hour+25*60+1400 {
		t.Errorf("MR read %d–%d", s[1].ClaimedSec-m, s[1].FinalSec-m)
	}
	if want := m + 9*hour + 10*60 + (1400 - 5*60); s[0].ClaimedSec != want {
		t.Errorf("XR claimed at %d, want %d (debt repaid)", s[0].ClaimedSec-m, want-m)
	}
	// The urgent study at 21:00 goes to a weeknight lane after a 10-minute
	// handoff; its read time is not general.
	if s[2].ClaimedSec != m+21*hour || s[2].ReadStartSec != m+21*hour+600 || busy.general[2] != 0 || busy.general[0] != 100 || busy.core[0] != 100 {
		t.Errorf("lane read %+v busy %v", s[2], busy)
	}
}

// The worklist and claim rules: the shift-end rule (no read whose
// longest time would end after the shift), the past-target promotion
// (a past-target study ahead of a later urgent one, in entry order) and
// the lanes' urgent-only scope.
func TestOCReaderRules(t *testing.T) {
	p := mustProfile(t)
	q := ocFidelity{Coverage: "satgap"}
	mk := func(pri, mod string, at int64) Study {
		return Study{Priority: pri, Modality: mod, AcquiredSec: at, EnteredSec: at}
	}
	m0, m1 := fidMidnight(0), fidMidnight(1)
	// Shift end: an MR entered at 15:52 on Tuesday, in Avery's last block
	// (15:50–16:00, her shift ending at 16:00), waits for Blake's 16:30
	// block.
	rec := &Record{TATTargetMin: testTargets}
	rec.Studies = []Study{mk("routine", "MR", m0+15*hour+52*60)}
	q.read(p, rec, []fidStudy{{"MR", 900}}, []int{0}, 0, 1)
	if got := rec.Studies[0].ClaimedSec; got != m0+16*hour+30*60 {
		t.Errorf("shift-end rule: the MR claimed at %d min, want 16:30", (got-m0)/60)
	}
	// Promotion: at Avery's first block on Wednesday (07:30) the routine
	// study acquired Tuesday 07:00 (past its 24 hours) and the soon one
	// acquired Tuesday 09:00 (past 4 hours) come before the urgent one
	// entered at 07:20, in entry order; the routine one acquired Tuesday
	// 12:00 (within target) after them.
	rec = &Record{TATTargetMin: testTargets}
	rec.Studies = []Study{mk("routine", "XR", m0+7*hour), mk("soon", "XR", m0+9*hour), mk("routine", "XR", m0+12*hour), mk("urgent", "XR", m1+7*hour+20*60)}
	info := []fidStudy{{"XR", 100}, {"XR", 100}, {"XR", 100}, {"XR", 100}}
	q.read(p, rec, info, []int{0, 1, 2, 3}, 1, 2)
	order := []int{0, 1, 2, 3}
	slices.SortFunc(order, func(a, b int) int { return int(rec.Studies[a].ClaimedSec - rec.Studies[b].ClaimedSec) })
	if !slices.Equal(order, []int{0, 1, 3, 2}) || rec.Studies[0].ClaimedSec != m1+7*hour+30*60 {
		t.Errorf("promotion: claim order %v, first at %d min", order, (rec.Studies[0].ClaimedSec-m1)/60)
	}
	// Lanes read urgent studies only: a routine study acquired Tuesday
	// 21:30 waits for Wednesday's readers (none in this run).
	rec = &Record{TATTargetMin: testTargets}
	rec.Studies = []Study{mk("routine", "XR", m0+21*hour+30*60)}
	q.read(p, rec, []fidStudy{{"XR", 100}}, []int{0}, 0, 1)
	if got := rec.Studies[0].ClaimedSec; got != 0 {
		t.Errorf("a lane read a routine study at %d min", (got-m0)/60)
	}
}

// The tabulation of the fidelity practices agrees exactly with the
// production evaluator, on sampled seeds of a stable, a structural and a
// deterioration scenario of the fidelity stream.
func TestClinicalOCFidelityAgreesWithEvaluate(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	seen := map[bool]bool{}
	base := ocClinicalPolicy(t, ocBundles[1])
	for k, def := range ocFidelityScenarios() {
		if !slices.Contains([]string{"A85", "Wg85", "D-85"}, def.name) {
			continue
		}
		var recs []Record
		var xs []ocSeed
		for i := range 2 {
			rec, _ := def.q.record(ocRNG{rand.New(rand.NewPCG(20261103, uint64(100000*k+i)))}, base)
			rec.Seed = int64(i)
			recs = append(recs, *rec)
			xs = append(xs, ocExtract(base, rec))
		}
		for _, b := range ocBundles {
			p := ocClinicalPolicy(t, b)
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
	if !seen[true] || !seen[false] {
		t.Errorf("the sampled decisions do not include both a passing and a failing seed: %v", seen)
	}
}

// The run (DT_OC_FIDELITY=1; fidelity protocol §8). DT_OC_SCENARIOS and
// DT_OC_OUT reproduce a subset into a scratch file.
func TestClinicalOCFidelity(t *testing.T) {
	if os.Getenv("DT_OC_FIDELITY") == "" {
		t.Skip("set DT_OC_FIDELITY=1 for the fidelity clinical operating characteristics (docs/calibration/clinical-5c4b/fidelity-protocol.md)")
	}
	var names []string
	out := "../../docs/evaluation/oc-v0.2-eval-5-exploratory-fidelity.json"
	if s := os.Getenv("DT_OC_SCENARIOS"); s != "" {
		names, out = strings.Split(s, ","), os.Getenv("DT_OC_OUT")
		if out == "" {
			t.Fatal("DT_OC_SCENARIOS needs DT_OC_OUT")
		}
	}
	doc := map[string]any{
		"schema":    "dreamteam.oc-clinical/v3",
		"stage":     "exploratory, simulator-fidelity calibration: nothing selected; the owner chooses the v0.2-eval-5 values",
		"protocol":  "docs/calibration/clinical-5c4b/fidelity-protocol.md",
		"profile":   "internal/eval/" + simProfilePath,
		"streams":   "seed i of scenario k: PCG(20261103, 100000·k + i)",
		"policy":    "v0.2-eval-4 with the candidate clinical rules (as oc-v0.2-eval-5-exploratory.json)",
		"bundles":   ocBundles,
		"scenarios": ocFidelityCompute(t, ocFidelityScenarios(), 20261103, names),
	}
	raw, err := json.MarshalIndent(doc, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The fidelity model's default records are those of the committed run
// (oc-v0.2-eval-5-exploratory-fidelity.json, 18b21d6): digests of sampled
// records on its stream, taken before the takeover instant became a
// field.
func TestOCFidelityModelUnchanged(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	want := map[string][2]string{
		"A85":  {"6b3024e4bd64c9d983ff815ed7a7997c889c377ad4b42e05b74c643d33c2425d", "2c497a4e121e0d1b30f3ac7500e6379d083bf185745dbfdf7f70e0f1fc14c25e"},
		"D-85": {"2adb4b21a3dd0bbf5f2e536105e516376ae46e60c90fc964071460deb2a1f649", "3aa3ecbb62b78aac3b354553fc2501f46f7869e36e469d44638bddc93a83c00b"},
		"Wg92": {"e504f5fd00325d0454a5c906e45f56016f82c73a006c0790da111513841f212e", "23cee2e858b06eedb6ce5133e590bcd7537c0fd5b85b1bb2c73981f0da684699"},
	}
	base := ocClinicalPolicy(t, ocBundles[1])
	for k, def := range ocFidelityScenarios() {
		w, ok := want[def.name]
		if !ok {
			continue
		}
		for i := range 2 {
			rec, _ := def.q.record(ocRNG{rand.New(rand.NewPCG(20261103, uint64(100000*k+i)))}, base)
			raw, _ := json.Marshal(struct {
				R *Record
				B fidBusy
			}{rec, fidBusy{}})
			if got := fmt.Sprintf("%x", sha256.Sum256(raw)); got != w[i] {
				t.Errorf("%s seed %d: record digest %s, the committed run's %s", def.name, i, got, w[i])
			}
		}
	}
}

// The aligned takeover is 06:00 on an evaluator Tuesday, so that the
// evaluator's Monday weeks start at the synthetic calendar's Monday
// midnight; t0's start on the synthetic Wednesday. A practice at the
// aligned takeover is the same practice shifted in time.
func TestOCFidelityAlignedWeeks(t *testing.T) {
	week := func(sec int64) int64 { return floorDiv(sec/daySec+3, 7) }
	o := fidAlignedTakeover
	if o%daySec != 6*hour || floorDiv(o/daySec+3, 7)*7-3 != o/daySec-1 {
		t.Fatalf("aligned takeover %d is not an evaluator Tuesday 06:00", o)
	}
	mon := fidMidnightAt(o, 6) // day 0 Tuesday, day 6 Monday
	if week(mon) == week(mon-1) || week(mon) != week(mon+7*daySec-1) {
		t.Error("the aligned evaluator week does not run Monday to Sunday")
	}
	mon0 := fidMidnight(6)
	if week(mon0) != week(mon0-1) {
		t.Error("t0's evaluator weeks unexpectedly start on Monday")
	}
	base := ocClinicalPolicy(t, ocBundles[1])
	q := ocFidelity{Util: 0.85}
	a, _ := q.record(ocRNG{rand.New(rand.NewPCG(1, 0))}, base)
	q.Takeover = o
	b, _ := q.record(ocRNG{rand.New(rand.NewPCG(1, 0))}, base)
	shift := o - t0
	if len(a.Studies) != len(b.Studies) || len(a.Orders) != len(b.Orders) {
		t.Fatalf("studies %d / %d, orders %d / %d", len(a.Studies), len(b.Studies), len(a.Orders), len(b.Orders))
	}
	for i := range a.Studies {
		if b.Studies[i].AcquiredSec-a.Studies[i].AcquiredSec != shift || (a.Studies[i].FinalSec != 0 && b.Studies[i].FinalSec-a.Studies[i].FinalSec != shift) {
			t.Fatalf("study %d not shifted by the takeover", i)
		}
	}
}

// With a demand stream of their own, a practice and the same practice
// with 60% less imaging capacity from day 30 (each with the same booking
// stream, as two plans on one seed) receive the same eligible demand: the
// lost-demand gate pairs every seed, and sees the cut's lost orders.
func TestOCFidelityPairedDemand(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	pol := ocClinicalPolicy(t, ocBundles[1])
	cand, _ := candidatePolicy(t)
	a, b := ocFidelity{Util: 0.85}, ocFidelity{Util: 0.85, ImgCap: []ocCap{{From: 30, Factor: 0.4}}}
	var ra, rb []Record
	for i := range 3 {
		demand := func() ocRNG { return ocRNG{rand.New(rand.NewPCG(1, uint64(2*i)))} }
		x, _ := a.recordPaired(demand(), ocRNG{rand.New(rand.NewPCG(1, uint64(2*i+1)))}, pol)
		y, _ := b.recordPaired(demand(), ocRNG{rand.New(rand.NewPCG(1, uint64(2*i+1)))}, pol)
		x.Seed, y.Seed = int64(i), int64(i)
		ra, rb = append(ra, *x), append(rb, *y)
	}
	for _, g := range lostDemandGates(cand, rb, ra) {
		if g.MatchedSeeds != 3 || len(g.UnmatchedSeeds) != 0 {
			t.Fatalf("%s: not paired: %+v", g.Priority, g)
		}
		if g.Priority == "routine" && (g.Candidate.Lost <= g.Baseline.Lost || g.Status != Fail) {
			t.Errorf("routine: the imaging cut's losses not seen: %+v", g)
		}
	}
	// One stream for both (the committed model's): the cut changes the
	// booking draws and with them the demand.
	x, _ := a.record(ocRNG{rand.New(rand.NewPCG(1, 0))}, pol)
	y, _ := b.record(ocRNG{rand.New(rand.NewPCG(1, 0))}, pol)
	x.Seed, y.Seed = 0, 0
	if gs := lostDemandGates(cand, []Record{*y}, []Record{*x}); gs[2].MatchedSeeds != 0 {
		t.Errorf("one stream: demand unexpectedly matched")
	}
}
