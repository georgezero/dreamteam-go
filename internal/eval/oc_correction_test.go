package eval

import (
	"encoding/json"
	"math"
	"math/rand/v2"
	"os"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// The calibration correction of the combined clinical operating
// characteristics (Slice 5C.4b, docs/calibration/clinical-5c4b/
// correction-protocol.md). The exploratory run's stable practice was
// meant to read at utilization 0.85 but read at about 0.80 or less: its
// rate was set over 66.5 reading hours a week while it read 70.5, its
// cancellations were not counted, and it started empty. Here the
// utilization is a measured property of each practice, the practice runs
// in its steady state from the takeover, and it reads in the simulator's
// worklist order. The original model and its results are unchanged.

// ocUtil is a record's measured utilization: acquisitions over the
// general reading capacity offered (studies), from the takeover to the
// horizon and over its last 30 days, and imaging slots booked over slots
// offered (cancelled bookings included) to the horizon.
type ocUtil struct{ read, readLate, imaging float64 }

func (q ocPractice) utilization(r *Record) ocUtil {
	date, _ := simtime.ParseDate(r.TakeoverDate)
	weekday := func(d int) int { return (int(date.AddDays(int64(d)).Weekday()) + 6) % 7 }
	start, H := r.TakeoverSec, r.HorizonEndSec
	late := H - 30*daySec
	rate := q.readRate()
	var offered, offeredLate float64
	for h := 0; start+int64(h)*hour < H; h++ {
		day := (6 + h) / 24
		share, urgentOnly := readingHours(q.Coverage, weekday(day), (6+h)%24)
		if urgentOnly {
			continue
		}
		c := rate * share * capFactor(day, q.ReadCap, q.ReadErode)
		offered += c
		if start+int64(h)*hour >= late {
			offeredLate += c
		}
	}
	var acq, acqLate float64
	for i := range r.Studies {
		if a := r.Studies[i].AcquiredSec; a > start && a <= H {
			acq++
			if a > late {
				acqLate++
			}
		}
	}
	var slots, booked float64
	for d := 0; d < int((H-start)/daySec); d++ {
		slots += float64(q.slots(d, weekday(d)))
	}
	for i := range r.Orders {
		if a := r.Orders[i].FirstApptSec; a > start && a <= H {
			booked++
		}
	}
	return ocUtil{acq / offered, acqLate / offeredLate, booked / slots}
}

// The corrected practice: the effective reading utilization u, a warm-up
// of 181 days (the longest routine lead), the simulator's worklist.
func ocCorrected(u float64) ocPractice { return ocPractice{EffUtil: u, Warmup: 181, Worklist: "sim"} }

type ocCorrectedDef struct {
	ocScenarioDef
	regime string
}

// ocCorrectedScenarios are the correction protocol's §4 scenarios, in
// stream order.
func ocCorrectedScenarios() []ocCorrectedDef {
	var defs []ocCorrectedDef
	add := func(regime, name, desc, role string, seeds int, q ocPractice) {
		defs = append(defs, ocCorrectedDef{ocScenarioDef{name, desc, role, q, seeds}, regime})
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
		add(r.regime, "A"+s, "stationary", role, 2500, ocCorrected(r.u))
		add(r.regime, "B"+s, "seasonal", role, 2500, with(ocCorrected(r.u), func(q *ocPractice) { q.Seasonal = true }))
		add(r.regime, "C"+s, "bursty (day CV 0.15)", role, 2500, with(ocCorrected(r.u), func(q *ocPractice) { q.DayCV = 0.15 }))
		add(r.regime, "Cs"+s, "seasonal and bursty", role, 2500, with(ocCorrected(r.u), func(q *ocPractice) { q.Seasonal, q.DayCV = true, 0.15 }))
		add(r.regime, "Wg"+s, "no reading Saturday after 14:00 or Sunday, at the same utilization (the weekend's capacity on weekdays)", "structural", 1000,
			with(ocCorrected(r.u), func(q *ocPractice) { q.Coverage = "satgap" }))
		add(r.regime, "WgS"+s, "no reading Saturday after 14:00 or Sunday, at A"+s+"'s team rate (the weekend's capacity removed)", "structural", 1000,
			with(ocCorrected(r.u), func(q *ocPractice) { q.Coverage, q.RateHours = "satgap", generalHours("") }))
		if r.u < 0.9 {
			add(r.regime, "R0"+s, "the scenario's own reading rota (Mon–Fri 07:30–20:00, Sat 10:00–14:00), at the same utilization", "structural", 1000,
				with(ocCorrected(r.u), func(q *ocPractice) { q.Coverage = "rota" }))
			add(r.regime, "P"+s, "A"+s+" read urgent, then soon, then routine (the original model's worklist)", "model variant", 1000,
				with(ocCorrected(r.u), func(q *ocPractice) { q.Worklist = "" }))
		}
		add(r.regime, "L10-"+s, "reading capacity 10% lower from day 60", "deterioration", 1000, with(ocCorrected(r.u), func(q *ocPractice) { q.ReadCap = []ocCap{{From: 60, Factor: 0.9}} }))
		if r.u < 0.9 {
			add(r.regime, "L20-"+s, "reading capacity 20% lower from day 60", "deterioration", 1000, with(ocCorrected(r.u), func(q *ocPractice) { q.ReadCap = []ocCap{{From: 60, Factor: 0.8}} }))
			add(r.regime, "D-"+s, "reading capacity 25% lower from day 30", "deterioration", 1000, with(ocCorrected(r.u), func(q *ocPractice) { q.ReadCap = []ocCap{{From: 30, Factor: 0.75}} }))
		}
		add(r.regime, "K-"+s, "reading capacity eroding 0.2% a day from day 30", "deterioration", 1000, with(ocCorrected(r.u), func(q *ocPractice) { q.ReadErode = ocErode{30, 0.002} }))
		add(r.regime, "IL10-"+s, "imaging capacity 10% lower from day 60", "deterioration", 1000, with(ocCorrected(r.u), func(q *ocPractice) { q.ImgCap = []ocCap{{From: 60, Factor: 0.9}} }))
		add(r.regime, "IK-"+s, "imaging capacity eroding 0.2% a day from day 30", "deterioration", 1000, with(ocCorrected(r.u), func(q *ocPractice) { q.ImgErode = ocErode{30, 0.002} }))
		if r.u < 0.9 {
			add(r.regime, "I92-"+s, "imaging utilization 0.92", "stress", 1000, with(ocCorrected(r.u), func(q *ocPractice) { q.ImgUtil = 0.92 }))
		}
	}
	return defs
}

// ocExtra is a seed's consequence measures beyond the gates' inputs.
type ocExtra struct {
	util   ocUtil
	lost   float64 // orders received after the takeover and by the horizon, lost
	unread int64   // unread at the horizon sample
}

func ocExtraOf(q ocPractice, r *Record) ocExtra {
	x := ocExtra{util: q.utilization(r)}
	end := &r.Captures[len(r.Captures)-1]
	var n, lost float64
	for i, o := range r.Orders {
		if o.ReceivedSec > r.TakeoverSec && o.ReceivedSec <= r.HorizonEndSec {
			n++
			if end.OrderOutcomes[i] == OutLost {
				lost++
			}
		}
	}
	x.lost = lost / max(n, 1)
	for _, s := range r.Samples {
		if s.AtSec == r.HorizonEndSec {
			x.unread = int64(s.Unread)
		}
	}
	return x
}

// ocComponents are the components of a bundle a seed fails ("absolute":
// an absolute-limit violation, outside the shared allowance).
func (x ocSeed) components(p *Policy, b ocBundle) []string {
	var cs []string
	add := func(name string, bad bool) {
		if bad && !slices.Contains(cs, name) {
			cs = append(cs, name)
		}
	}
	add("invalid", !x.valid)
	add("v4Service", x.service != Pass)
	add("v4Backlog", x.backlog != "stable")
	for _, q := range Priorities {
		add("severeRate", x.severeFails(p, q, b.SevereRate[q]))
		add("severeWeek", x.week[q] > b.SeverePerWeek[q])
		add("timelyAccess", x.accessFails(p, q, b.Access[q], b.Timely))
		add("maxWait", x.maxWait[q] > b.MaxWait[q])
		add("absolute", x.pastDue[q] > b.Absolute[q])
	}
	return cs
}

type ocFloatDist struct {
	Mean float64 `json:"mean"`
	P05  float64 `json:"p05"`
	P50  float64 `json:"p50"`
	P95  float64 `json:"p95"`
	Max  float64 `json:"max"`
}

func floatDist(xs []float64) ocFloatDist {
	s := slices.Clone(xs)
	slices.Sort(s)
	at := func(p float64) float64 { return s[int(math.Ceil(p*float64(len(s))))-1] }
	sum := 0.0
	for _, x := range s {
		sum += x
	}
	return ocFloatDist{sum / float64(len(s)), at(0.05), at(0.5), at(0.95), s[len(s)-1]}
}

type ocConsequences struct {
	ReadingUtilization     ocFloatDist            `json:"effectiveReadingUtilization"`
	ReadingUtilizationLate ocFloatDist            `json:"effectiveReadingUtilizationLast30Days"`
	ImagingUtilization     ocFloatDist            `json:"imagingSlotUtilization"`
	SevereShare            map[string]ocFloatDist `json:"severeDelayShareGated"`
	WorstPastDueMin        map[string]Dist        `json:"worstPastDueMinPerSeed"`
	TimelyModerate         map[string]ocFloatDist `json:"timelyAccessShareGatedModerateLimits"`
	LostShare              ocFloatDist            `json:"lostOrderShare"`
	UnreadAtHorizon        Dist                   `json:"unreadAtHorizon"`
}

// ocBundleDetail: the bundle's failing components and its rejection's
// Monte Carlo uncertainty. The combined interval is simultaneous: the
// rejection is increasing in the failing and the violating fractions, so
// it is bounded by their Clopper–Pearson 97.5% bounds (Bonferroni).
type ocBundleDetail struct {
	Bundle         string         `json:"bundle"`
	Failing        map[string]int `json:"seedsFailingComponent"`
	Sole           map[string]int `json:"seedsFailingOnlyThisComponent"`
	Rejection40CI  [2]float64     `json:"rejection40Simultaneous95"`
	SharedOnlyCI   [2]float64     `json:"rejection40SharedOnly95"`
	AbsoluteOnlyCI [2]float64     `json:"rejection40AbsoluteOnly95"`
	Groups         Interval       `json:"groupsRejectedFraction"` // Clopper–Pearson 95%
}

type ocCorrectedOut struct {
	ocScenarioOut
	Regime       string           `json:"regime"`
	Consequences ocConsequences   `json:"consequences"`
	BundleDetail []ocBundleDetail `json:"bundleDetail"`
}

// cpBounds is the exact two-sided interval of k of n at level 1 − a.
func cpBounds(k, n int, a float64) (lo, hi float64) {
	hi = 1
	if k > 0 {
		lo = betaQuantile(float64(k), float64(n-k+1), a/2)
	}
	if k < n {
		hi = betaQuantile(float64(k+1), float64(n-k), 1-a/2)
	}
	return lo, hi
}

func ocDetail(p *Policy, def ocCorrectedDef, seeds []ocSeed, extras []ocExtra, base ocScenarioOut) ocCorrectedOut {
	o := ocCorrectedOut{ocScenarioOut: base, Regime: def.regime}
	n := len(seeds)
	var ru, rl, iu, lost []float64
	var unread []int64
	sev, tim := map[string][]float64{}, map[string][]float64{}
	worst := map[string][]int64{}
	for i, x := range seeds {
		e := extras[i]
		ru, rl, iu, lost, unread = append(ru, e.util.read), append(rl, e.util.readLate), append(iu, e.util.imaging), append(lost, e.lost), append(unread, e.unread)
		for _, q := range Priorities {
			c := x.severe[q+"/"+WinGated]
			sev[q] = append(sev[q], float64(c.num)/float64(max(c.den, 1)))
			a := x.access[q+"/"+WinGated+"/"+strconv.FormatInt(ocBundles[1].Access[q], 10)]
			tim[q] = append(tim[q], float64(a.num)/float64(max(a.den, 1)))
			worst[q] = append(worst[q], x.pastDue[q])
		}
	}
	o.Consequences = ocConsequences{ReadingUtilization: floatDist(ru), ReadingUtilizationLate: floatDist(rl), ImagingUtilization: floatDist(iu),
		SevereShare: map[string]ocFloatDist{}, WorstPastDueMin: map[string]Dist{}, TimelyModerate: map[string]ocFloatDist{},
		LostShare: floatDist(lost), UnreadAtHorizon: dist(unread)}
	for _, q := range Priorities {
		o.Consequences.SevereShare[q], o.Consequences.TimelyModerate[q], o.Consequences.WorstPastDueMin[q] = floatDist(sev[q]), floatDist(tim[q]), dist(worst[q])
	}
	for bi, b := range ocBundles {
		d := ocBundleDetail{Bundle: b.Name, Failing: map[string]int{}, Sole: map[string]int{}}
		fail, viol := 0, 0
		for _, x := range seeds {
			cs := x.components(p, b)
			for _, c := range cs {
				d.Failing[c]++
			}
			if len(cs) == 1 {
				d.Sole[cs[0]]++
			}
			switch {
			case slices.Contains(cs, "absolute"):
				viol++
			case len(cs) > 0:
				fail++
			}
		}
		br := base.Bundles[bi]
		if viol != br.AbsoluteViolating || fail != br.ClinicalFailing-countBoth(p, b, seeds) {
			panic("bundle components disagree with the bundle rule: " + def.name + " " + b.Name)
		}
		fLo, fHi := cpBounds(fail, n, 0.025)
		vLo, vHi := cpBounds(viol, n, 0.025)
		d.Rejection40CI = [2]float64{rejectionBoth(max(0, 1-fLo-vLo), fLo), rejectionBoth(max(0, 1-fHi-vHi), fHi)}
		cLo, cHi := cpBounds(br.ClinicalFailing, n, 0.05)
		d.SharedOnlyCI = [2]float64{rejection40(cLo), rejection40(cHi)}
		aLo, aHi := cpBounds(viol, n, 0.05)
		d.AbsoluteOnlyCI = [2]float64{1 - math.Pow(1-aLo, 40), 1 - math.Pow(1-aHi, 40)}
		d.Groups = clopperPearson(br.GroupsRejected, br.Groups)
		o.BundleDetail = append(o.BundleDetail, d)
	}
	return o
}

// countBoth: seeds failing the shared rule and also violating an
// absolute limit (the bundle rows count them in both).
func countBoth(p *Policy, b ocBundle, seeds []ocSeed) int {
	k := 0
	for _, x := range seeds {
		if c, a := x.bundle(p, b); c && a {
			k++
		}
	}
	return k
}

// ocCorrectedCompute runs defs (only those named in names, when given)
// on stream PCG(stream, 100000·k + i), k the scenario's index in defs.
func ocCorrectedCompute(t *testing.T, defs []ocCorrectedDef, stream uint64, names []string) []ocCorrectedOut {
	base := ocClinicalPolicy(t, ocBundles[1])
	var outs []ocCorrectedOut
	for k, def := range defs {
		if len(names) > 0 && !slices.Contains(names, def.name) {
			continue
		}
		n := def.seeds
		seeds, extras, studies := make([]ocSeed, n), make([]ocExtra, n), make([]int, n)
		var wg sync.WaitGroup
		work := make(chan int)
		for range runtime.NumCPU() {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := range work {
					rec := def.q.record(ocRNG{rand.New(rand.NewPCG(stream, uint64(100000*k+i)))}, base)
					rec.Seed = int64(i)
					seeds[i], extras[i], studies[i] = ocExtract(base, rec), ocExtraOf(def.q, rec), len(rec.Studies)
				}
			}()
		}
		for i := range n {
			work <- i
		}
		close(work)
		wg.Wait()
		outs = append(outs, ocDetail(base, def, seeds, extras, ocTabulate(base, def.ocScenarioDef, seeds, studies)))
	}
	return outs
}

// The original exploratory practices' measured utilization, on the first
// 200 seeds of each one's own stream (PCG(20261101, 100000·k + i)).
type ocOriginalUtil struct {
	Name     string      `json:"name"`
	Nominal  float64     `json:"nominalReadingUtilization"`
	Measured ocFloatDist `json:"effectiveReadingUtilization"`
	Late     ocFloatDist `json:"effectiveReadingUtilizationLast30Days"`
	Imaging  ocFloatDist `json:"imagingSlotUtilization"`
}

func ocOriginalUtilization(t *testing.T) []ocOriginalUtil {
	base := ocClinicalPolicy(t, ocBundles[1])
	var outs []ocOriginalUtil
	for k, def := range ocClinicalScenarios() {
		const n = 200
		us := make([]ocUtil, n)
		var wg sync.WaitGroup
		work := make(chan int)
		for range runtime.NumCPU() {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := range work {
					us[i] = def.q.utilization(def.q.record(ocRNG{rand.New(rand.NewPCG(20261101, uint64(100000*k+i)))}, base))
				}
			}()
		}
		for i := range n {
			work <- i
		}
		close(work)
		wg.Wait()
		var r, l, im []float64
		for _, u := range us {
			r, l, im = append(r, u.read), append(l, u.readLate), append(im, u.imaging)
		}
		outs = append(outs, ocOriginalUtil{def.name, cmpOr(def.q.ReadUtil, 0.85), floatDist(r), floatDist(l), floatDist(im)})
	}
	return outs
}

// The general reading hours and the original model's arithmetic: its
// rate was set over 66.5 hours a week, but its stable practice reads
// 70.5 (the weekend hours), so its nominal 0.85 is 0.80 of its capacity
// before cancellations and the empty start.
func TestOCGeneralHours(t *testing.T) {
	for cov, want := range map[string]float64{"": 70.5, "satgap": 66.5, "rota": 66.5} {
		if got := generalHours(cov); got != want {
			t.Errorf("general reading hours of %q: %v, want %v", cov, got, want)
		}
	}
	if got := 0.85 * 66.5 / generalHours(""); math.Abs(got-0.8018) > 1e-3 {
		t.Errorf("the original stable practice's capacity utilization %v", got)
	}
}

// The simulator's worklist (internal/sim/reading.go urgent_then_fifo):
// urgent and past-target studies first, in entry order; soon is not ahead
// of routine before its target; an urgent-only reader reads urgent
// studies only, never a promoted one.
func TestSimPick(t *testing.T) {
	rec := &Record{TATTargetMin: testTargets}
	at := int64(1_000_000)
	st := func(q string, ago int64) Study {
		return Study{Priority: q, AcquiredSec: at - ago*60, EnteredSec: at - ago*60}
	}
	rec.Studies = []Study{st("routine", 2000), st("routine", 300), st("soon", 250), st("soon", 200), st("urgent", 10)}
	open := []int{0, 1, 2, 3, 4}
	cases := []struct {
		name       string
		open       []int
		urgentOnly bool
		want       int
	}{
		{"past-due routine ahead of urgent (earlier entry)", open, false, 0},
		{"past-due soon ahead of urgent", []int{1, 2, 3, 4}, false, 1},
		{"urgent ahead of soon within target", []int{1, 3, 4}, false, 2},
		{"soon not ahead of routine within target", []int{1, 3}, false, 0},
		{"urgent-only reader skips promoted studies", open, true, 4},
		{"urgent-only reader with no urgent study", []int{0, 1, 2}, true, -1},
		{"nothing entered yet", []int{4}, false, 0},
	}
	for _, c := range cases {
		if got := simPick(rec, c.open, at, c.urgentOnly); got != c.want {
			t.Errorf("%s: picked %d, want %d", c.name, got, c.want)
		}
	}
	if got := simPick(rec, []int{4}, at-20*60, false); got != -1 {
		t.Errorf("a study entered after fin was picked: %d", got)
	}
}

// The corrected practices meet their utilization, on the protocol's debug
// stream PCG(1, ·) (never used for results), and their records are valid
// with inherited work at the takeover: only orders still open then (some
// booked), only studies unread then (some, after a shortfall).
func TestOCCorrectedCalibration(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	base := ocClinicalPolicy(t, ocBundles[1])
	unread := 0
	for _, c := range []struct {
		q    ocPractice
		want float64
	}{{ocCorrected(0.85), 0.85}, {ocCorrected(0.92), 0.92}, {with(ocCorrected(0.85), func(q *ocPractice) { q.Coverage = "satgap" }), 0.85},
		// Reading 30% short for the 30 days before the takeover: a backlog
		// inherited (its utilization not checked).
		{with(ocCorrected(0.85), func(q *ocPractice) { q.ReadCap = []ocCap{{From: -30, To: 0, Factor: 0.7}} }), 0}} {
		var sum float64
		const n = 3
		booked := 0
		for i := range n {
			rec := c.q.record(ocRNG{rand.New(rand.NewPCG(1, uint64(i)))}, base)
			m := Measure(base, rec)
			if len(m.Integrity) > 0 {
				t.Fatalf("%+v: %v", c.q, m.Integrity)
			}
			for _, o := range rec.Orders {
				if o.ReceivedSec <= rec.TakeoverSec {
					if o.ResolvedSec != 0 && o.ResolvedSec <= rec.TakeoverSec {
						t.Fatalf("%+v: an inherited order resolved before the takeover", c.q)
					}
					booked++
				}
			}
			for _, s := range rec.Studies {
				if s.AcquiredSec <= rec.TakeoverSec {
					if s.FinalSec != 0 && s.FinalSec <= rec.TakeoverSec {
						t.Fatalf("%+v: an inherited study read before the takeover", c.q)
					}
					unread++
				}
			}
			sum += c.q.utilization(rec).read
		}
		if got := sum / n; c.want > 0 && math.Abs(got-c.want) > 0.015 {
			t.Errorf("%+v: measured reading utilization %.4f, want %.2f ± 0.015", c.q, got, c.want)
		}
		if booked == 0 {
			t.Errorf("%+v: no inherited order at the takeover", c.q)
		}
	}
	if unread == 0 {
		t.Error("no practice inherited an unread study")
	}
}

func with(q ocPractice, f func(*ocPractice)) ocPractice {
	f(&q)
	return q
}

// The tabulation of the corrected practices agrees exactly with the
// production evaluator, on sampled seeds of a stable, a structural and a
// deterioration scenario of the correction stream.
func TestClinicalOCCorrectedAgreesWithEvaluate(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	seen := map[bool]bool{}
	defs := ocCorrectedScenarios()
	base := ocClinicalPolicy(t, ocBundles[1])
	for k, def := range defs {
		if !slices.Contains([]string{"A85", "WgS92", "D-85"}, def.name) {
			continue
		}
		var recs []Record
		var xs []ocSeed
		for i := range 2 {
			rec := def.q.record(ocRNG{rand.New(rand.NewPCG(20261102, uint64(100000*k+i)))}, base)
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

// The run (DT_OC_CORRECTED=1; correction protocol §7). DT_OC_SCENARIOS,
// a comma-separated list of names, reproduces a subset into the scratch
// file named by DT_OC_OUT instead.
func TestClinicalOCCorrected(t *testing.T) {
	if os.Getenv("DT_OC_CORRECTED") == "" {
		t.Skip("set DT_OC_CORRECTED=1 for the corrected clinical operating characteristics (docs/calibration/clinical-5c4b/correction-protocol.md)")
	}
	var names []string
	out := "../../docs/evaluation/oc-v0.2-eval-5-exploratory-corrected.json"
	if s := os.Getenv("DT_OC_SCENARIOS"); s != "" {
		names, out = strings.Split(s, ","), os.Getenv("DT_OC_OUT")
		if out == "" {
			t.Fatal("DT_OC_SCENARIOS needs DT_OC_OUT")
		}
	}
	doc := map[string]any{
		"schema":    "dreamteam.oc-clinical/v2",
		"stage":     "exploratory, corrected calibration: nothing selected; the owner chooses the v0.2-eval-5 values",
		"protocol":  "docs/calibration/clinical-5c4b/correction-protocol.md",
		"streams":   "seed i of scenario k: PCG(20261102, 100000·k + i)",
		"policy":    "v0.2-eval-4 with the candidate clinical rules (as oc-v0.2-eval-5-exploratory.json)",
		"bundles":   ocBundles,
		"scenarios": ocCorrectedCompute(t, ocCorrectedScenarios(), 20261102, names),
	}
	if len(names) == 0 {
		doc["originalPracticesUtilization"] = ocOriginalUtilization(t)
	}
	raw, err := json.MarshalIndent(doc, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}
