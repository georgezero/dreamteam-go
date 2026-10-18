package eval

import (
	"math"
	"math/rand/v2"
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
