package eval

import (
	"math"
	"math/rand/v2"
	"slices"

	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Slice 5C.2's operating characteristics (oc-v0.2-eval-3.json): the frozen
// per-seed service rule under the run's day and week structure, and the
// backlog rule with and without workload adjustment on a synthetic queue.
// Synthetic only: the dependence range is bounded by the development
// diagnostics (docs/calibration/policy-5c2), and no threshold, window or
// allowance was chosen from them. Regenerated with DT_OC=1.

// ocWeekday is the scenario's weekday demand profile (Mon 1.1, Tue–Fri 1,
// Sat 0.25, Sun 0), Monday first, scaled to a mean of 1 a day.
var ocWeekday = func() [7]float64 {
	w := [7]float64{1.1, 1, 1, 1, 1, 0.25, 0}
	var s float64
	for _, x := range w {
		s += x
	}
	for i := range w {
		w[i] *= 7 / s
	}
	return w
}()

// ocModel is a dependence model of the service OC.
type ocModel struct {
	Name     string             `json:"name"`
	Weekday  bool               `json:"weekdayProfile"`    // volumes follow the weekday profile
	VolumeCV float64            `json:"dailyVolumeCV"`     // gamma-distributed day factor on the volume
	DayICC   float64            `json:"dayICC"`            // random day effect on the breach probability
	WeekICC  map[string]float64 `json:"weekICCByPriority"` // random week effect, by priority
}

func ocModels() []ocModel {
	w := func(u, s, r float64) map[string]float64 {
		return map[string]float64{"urgent": u, "soon": s, "routine": r}
	}
	return []ocModel{
		{Name: "independent", WeekICC: w(0, 0, 0)},
		{Name: "variable volume (weekday profile, day CV 0.25)", Weekday: true, VolumeCV: 0.25, WeekICC: w(0, 0, 0)},
		{Name: "day-clustered, ICC 0.05", Weekday: true, VolumeCV: 0.25, DayICC: 0.05, WeekICC: w(0, 0, 0)},
		{Name: "day-clustered, ICC 0.15", Weekday: true, VolumeCV: 0.25, DayICC: 0.15, WeekICC: w(0, 0, 0)},
		{Name: "week-clustered, ICC 0.02", Weekday: true, VolumeCV: 0.25, WeekICC: w(0.02, 0.02, 0.02)},
		{Name: "week-clustered, ICC 0.05", Weekday: true, VolumeCV: 0.25, WeekICC: w(0.05, 0.05, 0.05)},
		{Name: "week-clustered, ICC 0.10", Weekday: true, VolumeCV: 0.25, WeekICC: w(0.10, 0.10, 0.10)},
		// The upper end of the development diagnostics for plans near the
		// gates (weekday-adjusted day ICC up to 0.15; week ICC urgent 0.03,
		// soon 0.11, routine 0.012).
		{Name: "development range (day 0.15; week urgent 0.03, soon 0.11, routine 0.012)", Weekday: true, VolumeCV: 0.25, DayICC: 0.15,
			WeekICC: w(0.03, 0.11, 0.012)},
	}
}

// ocRates is a setting of the underlying breach rates.
type ocRates struct {
	Label string             `json:"label"`
	Rates map[string]float64 `json:"rates"`
}

func ocRateSettings(p *Policy) []ocRates {
	plan := func(q string, x float64, label string) ocRates {
		r := map[string]float64{}
		for k, v := range p.Service.PlanningRate {
			r[k] = v
		}
		if q != "" {
			r[q] = x
		}
		return ocRates{label, r}
	}
	out := []ocRates{plan("", 0, "planning rates (false rejection)")}
	for _, x := range []float64{0.05, 0.06, 0.07, 0.09, 0.10, 0.12, 0.14} {
		out = append(out, plan("urgent", x, "urgent"))
	}
	for _, x := range []float64{0.15, 0.18, 0.24, 0.27, 0.32, 0.36} {
		out = append(out, plan("soon", x, "soon"))
	}
	for _, x := range []float64{0.02, 0.05, 0.06, 0.07} {
		out = append(out, plan("routine", x, "routine"))
	}
	return out
}

type ocClusterRow struct {
	Model   string             `json:"model"`
	Setting string             `json:"setting"`
	Rates   map[string]float64 `json:"rates"`
	Seeds   int                `json:"simulatedSeeds"`
	PerSeed float64            `json:"perSeedFailure"`
	SE      float64            `json:"perSeedFailureSE"`
	Reject  map[string]float64 `json:"rejectionBySeeds"` // "10", "40": P(more than floor(10%) of seeds fail)
}

// binomInv draws Binomial(n, p) by inversion (O(np)).
func (r ocRNG) binomInv(n int, p float64) int {
	if n <= 0 || p <= 0 {
		return 0
	}
	if p >= 1 {
		return n
	}
	if n > 400 {
		return r.binom(n, p)
	}
	u := r.Float64()
	pk := math.Pow(1-p, float64(n))
	cdf, k := pk, 0
	for u > cdf && k < n {
		pk *= float64(n-k) / float64(k+1) * p / (1 - p)
		k++
		cdf += pk
	}
	return k
}

// ocSeedFails simulates one seed of a 150-day gated period (T90 = days
// 1–90, tail = 91–150) and applies the per-seed gates exactly: urgent over
// the gated period, soon and routine in each window, with the evidence
// minimums.
func ocSeedFails(p *Policy, m ocModel, rates map[string]float64, r ocRNG) bool {
	type cnt struct{ n, b int64 }
	for _, q := range Priorities {
		v := ocVolume[q]
		lam := v[0] + (v[1]-v[0])*r.Float64()
		rate := rates[q]
		var t90, tail cnt
		var pw float64
		for d := range 150 {
			if d%7 == 0 {
				pw = rate
				if icc := m.WeekICC[q]; icc > 0 {
					pw = r.beta(rate, math.Sqrt(icc*rate*(1-rate)))
				}
			}
			mu := lam
			if m.Weekday {
				mu *= ocWeekday[d%7]
			}
			if m.VolumeCV > 0 && mu > 0 {
				k := 1 / (m.VolumeCV * m.VolumeCV)
				mu *= r.gamma(k) / k
			}
			n := r.poisson(mu)
			pd := pw
			if m.DayICC > 0 && n > 0 {
				pd = r.beta(pw, math.Sqrt(m.DayICC*pw*(1-pw)))
			}
			b := r.binomInv(n, pd)
			if d < 90 {
				t90.n, t90.b = t90.n+int64(n), t90.b+int64(b)
			} else {
				tail.n, tail.b = tail.n+int64(n), tail.b+int64(b)
			}
		}
		fails := func(c cnt) bool {
			return c.n < p.Service.MinCohort[q] || rateAbove(c.b, c.n, bp(p.Service.BreachGate[q]))
		}
		for _, w := range p.Service.PerSeedWindows[q] {
			switch w {
			case WinGated:
				if fails(cnt{t90.n + tail.n, t90.b + tail.b}) {
					return true
				}
			case WinT90:
				if fails(t90) {
					return true
				}
			case WinTail:
				if fails(tail) {
					return true
				}
			}
		}
	}
	return false
}

// binomAbove is P(Binomial(n, q) > lim).
func binomAbove(n int, q float64, lim int) float64 {
	qs := make([]float64, n)
	for i := range qs {
		qs[i] = q
	}
	return poissonBinomialAbove(qs, lim)
}

func ocClusterCompute(p *Policy, m ocModel, s ocRates, seeds int, stream uint64) ocClusterRow {
	r := ocRNG{rand.New(rand.NewPCG(20261018, stream))}
	f := 0
	for range seeds {
		if ocSeedFails(p, m, s.Rates, r) {
			f++
		}
	}
	q := float64(f) / float64(seeds)
	row := ocClusterRow{Model: m.Name, Setting: s.Label, Rates: s.Rates, Seeds: seeds, PerSeed: round6(q),
		SE: round6(math.Sqrt(q * (1 - q) / float64(seeds))), Reject: map[string]float64{}}
	for _, S := range []int{10, 40} {
		row.Reject[map[int]string{10: "10", 40: "40"}[S]] = round6(binomAbove(S, q, failLimit(bp(p.Service.MaxFailingSeedFraction), S)))
	}
	return row
}

// ---- backlog ----

type ocBacklogRow struct {
	Scenario  string             `json:"scenario"`
	Seeds     int                `json:"simulatedSeeds"`
	Unstable  map[string]float64 `json:"unstableSeedFraction"`        // by rule variant (ocBacklogVariants)
	Reject    map[string]float64 `json:"rejectionAt40Seeds"`          // P(more than 4 of 40 unstable)
	Reject10  map[string]float64 `json:"rejectionAt10Seeds"`          // P(more than 1 of 10 unstable)
	MaxOnly   map[string]float64 `json:"unstableOnPeakAloneFraction"` // seeds failing the peak check and nothing else
	MeanRatio float64            `json:"meanLaterBlockWorkloadRatio"` // later blocks' acquisitions over the reference's
}

// ocBacklogVariants: the frozen rules and, labelled not adopted, the
// alternatives the owner's open decision on the backlog numbers can be
// judged against.
var ocBacklogVariants = []string{
	"v0.2-eval-2 (unadjusted)",
	"v0.2-eval-3 (workload-adjusted, frozen)",
	"not adopted: v0.2-eval-3 without the peak check",
	"not adopted: v0.2-eval-3 with 20% growth tolerance",
	"not adopted: v0.2-eval-3 with 20% growth tolerance, without the peak check",
}

// ocSeason is the scenario's monthly season factor (scenario demand
// seasonFactorBP, January first).
var ocSeason = [12]float64{1.05, 1.00, 1.00, 1.00, 0.98, 0.95, 0.90, 0.92, 1.00, 1.02, 1.00, 0.93}

// ocQueueRecord simulates a reading queue hour by hour from the takeover
// (Tue 2026-10-13 06:00) for 181 days: arrivals at 52 a day (Poisson, the
// weekday profile, the hourly profile 08:00–20:00, the season factor when
// seasonal), reading capacity Mon–Fri 07:30–20:00 and Sat 10:00–14:00 at a
// utilization of 0.85 of the season-1 demand, reduced by loss from day
// lossFrom; urgent first, then soon, then routine, first in first out. It
// returns a record with the studies and the daily 06:00 samples.
func ocQueueRecord(r ocRNG, seasonal bool, lossFrom int, loss float64) *Record {
	const perDay = 52.0
	start := t0 // 06:00 on the takeover date
	date, _ := simtime.ParseDate(t0Date)
	rec := &Record{TakeoverSec: start, TakeoverDate: t0Date, HorizonEndSec: start + 150*daySec, TATTargetMin: testTargets}
	// Weekly reading hours: 5 × 12.5 + 4 = 66.5; capacity per reading hour.
	capPerHour := perDay * 7 / 66.5 / 0.85
	var queue [3][]int // indices into rec.Studies, by priority (0 urgent)
	carry := 0.0
	sample := func(d int, at int64) {
		s := Sample{Day: d, AtSec: at, Date: date.AddDays(int64(d)).String()}
		for k := range queue {
			s.Unread += len(queue[k])
		}
		for i := range rec.Studies {
			if rec.Studies[i].AcquiredSec <= at && rec.Studies[i].AcquiredSec > start {
				s.AcquiredCum++
			}
		}
		rec.Samples = append(rec.Samples, s)
	}
	for h := 0; h < 181*24; h++ {
		at := start + int64(h)*hour
		clock := (6 + h) % 24
		day := (6 + h) / 24 // days after the takeover date's midnight
		dt := date.AddDays(int64(day))
		wd := (int(dt.Weekday()) + 6) % 7 // Monday 0
		if h > 0 && clock == 6 {
			sample(day, at)
		}
		// Arrivals this hour.
		if clock >= 8 && clock < 20 {
			lam := perDay * ocWeekday[wd] / 12
			if seasonal {
				lam *= ocSeason[dt.Month-1]
			}
			for range r.poisson(lam) {
				u := r.Float64()
				q := 2
				switch {
				case u < 0.02:
					q = 0
				case u < 0.14:
					q = 1
				}
				acq := at + int64(r.Float64()*3600)
				rec.Studies = append(rec.Studies, Study{ID: "s", Priority: Priorities[q], AcquiredSec: acq, EnteredSec: acq})
				queue[q] = append(queue[q], len(rec.Studies)-1)
			}
		}
		// Reading this hour.
		open := (wd < 5 && clock >= 8 && clock < 20) || (wd < 5 && clock == 7) || (wd == 5 && clock >= 10 && clock < 14)
		if open {
			c := capPerHour
			if wd < 5 && clock == 7 {
				c /= 2 // from 07:30
			}
			if lossFrom > 0 && day >= lossFrom {
				c *= 1 - loss
			}
			carry += c
			for carry >= 1 {
				fin := at + hour - int64(carry/c*3600)
				k := 0
				for k < 3 && (len(queue[k]) == 0 || rec.Studies[queue[k][0]].AcquiredSec >= fin) {
					k++
				}
				if k == 3 {
					carry = min(carry, 1) // nothing ready: idle, no capacity banked
					break
				}
				s := &rec.Studies[queue[k][0]]
				queue[k] = queue[k][1:]
				s.ClaimedSec, s.ReadStartSec, s.FinalSec = max(fin-60, s.EnteredSec), max(fin-60, s.EnteredSec), fin
				carry--
			}
		}
	}
	rec.ObservedUntilSec = start + 181*daySec
	return rec
}

func ocBacklogRows(p *Policy, seeds int) []ocBacklogRow {
	v2 := *p
	v2.Backlog.WorkloadAdjustedChecks, v2.Backlog.AbsoluteChecks = nil, slices.Clone(backlogChecks)
	g20 := *p
	g20.Backlog.Growth = 0.20
	type sc struct {
		name     string
		seasonal bool
		from     int
		loss     float64
	}
	// unstable reports a variant's outcome; skipPeak ignores the peak check.
	unstable := func(bm BacklogMeasures, skipPeak bool) (bad, peakOnly bool) {
		failed := map[string]bool{}
		for _, c := range bm.Checks {
			if !c.Pass {
				failed[c.Check] = true
			}
		}
		peakOnly = failed["max"] && len(failed) == 1 && bm.Persistent == 0
		if skipPeak {
			delete(failed, "max")
		}
		return len(failed) > 0 || bm.Persistent > 0, peakOnly
	}
	var out []ocBacklogRow
	for k, s := range []sc{
		{"stationary demand, stable capacity (utilization 0.85)", false, 0, 0},
		{"scenario seasonality, stable capacity", true, 0, 0},
		{"stationary demand, capacity 5% lower from day 100", false, 100, 0.05},
		{"scenario seasonality, capacity 5% lower from day 100", true, 100, 0.05},
		{"scenario seasonality, capacity 10% lower from day 100", true, 100, 0.10},
	} {
		r := ocRNG{rand.New(rand.NewPCG(20261018, uint64(1000+k)))}
		row := ocBacklogRow{Scenario: s.name, Seeds: seeds, Unstable: map[string]float64{}, Reject: map[string]float64{}, Reject10: map[string]float64{},
			MaxOnly: map[string]float64{}}
		un, peak := map[string]int{}, map[string]int{}
		var ratio float64
		for range seeds {
			rec := ocQueueRecord(r, s.seasonal, s.from, s.loss)
			t90 := t0 + 90*daySec
			b2, b3, b20 := measureBacklog(&v2, rec, t90), measureBacklog(p, rec, t90), measureBacklog(&g20, rec, t90)
			for i, x := range []struct {
				bm   BacklogMeasures
				skip bool
			}{{b2, false}, {b3, false}, {b3, true}, {b20, false}, {b20, true}} {
				bad, po := unstable(x.bm, x.skip)
				if bad {
					un[ocBacklogVariants[i]]++
				}
				if po && !x.skip {
					peak[ocBacklogVariants[i]]++
				}
			}
			var a float64
			for _, b := range b3.Later {
				a += float64(b.Acquired) / float64(b3.Reference.Acquired)
			}
			ratio += a / float64(len(b3.Later))
		}
		row.MeanRatio = round6(ratio / float64(seeds))
		lim40, lim10 := failLimit(bp(p.Backlog.MaxUnstableSeedFrac), 40), failLimit(bp(p.Backlog.MaxUnstableSeedFrac), 10)
		for _, name := range ocBacklogVariants {
			q := float64(un[name]) / float64(seeds)
			row.Unstable[name] = round6(q)
			row.Reject[name] = round6(binomAbove(40, q, lim40))
			row.Reject10[name] = round6(binomAbove(10, q, lim10))
			if _, ok := peak[name]; ok {
				row.MaxOnly[name] = round6(float64(peak[name]) / float64(seeds))
			}
		}
		out = append(out, row)
	}
	return out
}
