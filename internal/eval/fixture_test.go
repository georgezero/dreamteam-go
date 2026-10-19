package eval

import (
	"fmt"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Synthetic records for the evaluator's tests: deterministic studies,
// orders, samples and captures built from a few rules, consistent with
// each other the way a recorded run is (samples reconcile with studies,
// remote fees with the ledger), unless a test breaks that on purpose.

const (
	t0     = int64(1_000_000_000) // the takeover (06:00, Tue 2026-10-13)
	t0Date = "2026-10-13"
	hour   = int64(3600)
)

var testTargets = map[string]int64{"urgent": 60, "soon": 240, "routine": 1440}

// synth describes one plan on one seed.
type synth struct {
	plan string
	seed int64
	// Studies a day by priority; breach permille by priority (a study
	// breaches when its index's permille slot falls under it); remote: a
	// routine study's share read remotely (permille).
	perDay     map[string]int
	breachPM   map[string]int
	neverPM    map[string]int // of the breaching, the permille never finalized
	remotePM   int
	fee        int64
	retainerPD int64 // remote retainer per day
	// Orders: one per study; declinePM of each priority's orders are lost
	// (never imaged), so their studies are not acquired.
	declinePM map[string]int
	walkIns   int // walk-ins a day, all served unless walkLeftPM
	walkLeft  int // permille that leave
	// Money: cumulative profit and cash on day d (fractional days pro rata).
	profitPD, cashPD, cash0 int64
	profitNoise             int64 // added per seed: (seed % 7 − 3) × noise a day
	upfront                 int64 // a one-off accrued cost from day 10 (profit only)
	minCash                 int64 // the posting-level minimum since the takeover
	minCashSet              bool
	lastAction, effect      int64 // seconds after t0 (0: none)
	inherited               int   // inherited unfinished routine studies
	teleOnlyOf, apptSHA     string
	observeShort            int64 // end observation this much earlier (seconds)
	horizon                 int64 // the common horizon, seconds after t0 (0: this record's own tail end)
	payables                int64 // obligations at every capture and sample (cents)
	// late: extra soon CT referrals a seed, received and wanted on the
	// given days, resolved (with out) an hour after resolve (open when out
	// is OutOpen).
	late []lateOrders
	// bursts: breach permille of one priority's studies acquired in days
	// [from, to); rare: a routine US order every rare-th day, lost (never
	// served) when rareLost.
	bursts   []burst
	rare     int
	rareLost bool
}

type lateOrders struct {
	n                      int
	recv, want, resolveDay float64
	out                    string
}

type burst struct {
	q        string
	from, to int64
	pm       int
}

// breachPMAt is the breach permille of a study of priority q acquired at
// acq.
func (s synth) breachPMAt(q string, acq int64) int {
	for _, b := range s.bursts {
		if b.q == q && acq >= t0+b.from*daySec && acq < t0+b.to*daySec {
			return b.pm
		}
	}
	return s.breachPM[q]
}

func base(plan string, seed int64) synth {
	return synth{plan: plan, seed: seed,
		perDay:    map[string]int{"urgent": 1, "soon": 2, "routine": 3},
		breachPM:  map[string]int{"urgent": 100, "soon": 200, "routine": 30},
		neverPM:   map[string]int{},
		declinePM: map[string]int{},
		profitPD:  10_000, cashPD: 5_000, cash0: 1_000_000, apptSHA: "same"}
}

// tailEnd is the record's common horizon.
func (s synth) tailEnd() int64 {
	if s.horizon > 0 {
		return t0 + s.horizon
	}
	end := t0 + 150*daySec
	if s.effect > 0 {
		end = max(end, t0+s.effect+30*daySec)
	}
	return end
}

var fixturePolicy *Policy

// observedUntil is what the frozen policy needs (every closure and the
// backlog block covering the horizon), less observeShort.
func (s synth) observedUntil() int64 {
	if fixturePolicy == nil {
		p, _, err := LoadPolicy("../../" + PolicyPath)
		if err != nil {
			panic(err)
		}
		fixturePolicy = p
	}
	r := &Record{TakeoverSec: t0, TakeoverDate: t0Date, HorizonEndSec: s.tailEnd()}
	return observationNeeded(fixturePolicy, r) - s.observeShort
}

func (s synth) money(at int64) Money {
	d := float64(at-t0) / float64(daySec)
	pd := s.profitPD + (s.seed%7-3)*s.profitNoise
	m := Money{OperatingProfitCents: int64(float64(pd) * d), NetRevenueCents: int64(float64(4*s.profitPD) * d),
		CashCents: s.cash0 + int64(float64(s.cashPD)*d), CashBridgeReconciles: true}
	if at >= t0+10*daySec {
		m.OperatingProfitCents -= s.upfront
	}
	m.OperatingExpenseCents = m.NetRevenueCents - m.OperatingProfitCents
	m.MinCashCents, m.MinCashSec = min(s.cash0, m.CashCents), t0
	if m.CashCents < s.cash0 {
		m.MinCashSec = at
	}
	if s.minCashSet {
		m.MinCashCents, m.MinCashSec = s.minCash, t0+hour
	}
	m.PayablesCents = s.payables
	return m
}

// pm reports whether item i of a stream falls in the first x permille of
// its thousand-item cycle (shifted by the seed).
func pm(i int, seed int64, x int) bool { return (i*379+int(seed)*17)%1000 < x }

func (s synth) record() Record {
	r := Record{Schema: RecordSchema, Plan: s.plan, Seed: s.seed, Engine: "test", ScenarioFingerprint: "fp",
		TATTargetMin: testTargets, TakeoverSec: t0, TakeoverDate: t0Date, TeleOnlyOf: s.teleOnlyOf, AppointmentsSHA256: s.apptSHA,
		HorizonEndSec: s.tailEnd()}
	if s.lastAction > 0 {
		r.LastActionSec = t0 + s.lastAction
	}
	if s.effect > 0 {
		r.LatestEffectSec = t0 + s.effect
	}
	r.ObservedUntilSec = s.observedUntil()
	for k := range s.inherited {
		acq := t0 - 2*hour + int64(k)*60
		r.Studies = append(r.Studies, Study{ID: "I", Priority: "routine", Modality: "XR", AcquiredSec: acq, EnteredSec: acq,
			ClaimedSec: acq + 31*hour, ReadStartSec: acq + 31*hour, FinalSec: acq + 32*hour})
		r.Orders = append(r.Orders, Order{ID: "IO", Origin: fmt.Sprintf("IO/%d", k), Priority: "routine", Modality: "XR", ReceivedSec: t0 - 3*daySec, WantedSec: t0 - 3*daySec, ResolvedSec: t0 + hour})
	}
	mods := map[string]string{"urgent": "MR", "soon": "CT", "routine": "XR"}
	idx := map[string]int{}
	days := int((r.ObservedUntilSec - t0) / daySec)
	type served struct {
		at  int64
		out string
	}
	var outs []served // per order: the time it resolves and how (0: open)
	sec := func(d float64) int64 { return t0 + int64(d*daySec) }
	for li, l := range s.late {
		for k := range l.n {
			o := Order{ID: "L", Origin: fmt.Sprintf("L/%d/%d", li, k), Priority: "soon", Modality: "CT", ReceivedSec: sec(l.recv) + int64(k)*60, WantedSec: sec(l.want) + int64(k)*60}
			r.Orders = append(r.Orders, o)
			if l.out == OutOpen {
				outs = append(outs, served{0, OutOpen})
			} else {
				outs = append(outs, served{sec(l.resolveDay) + hour, l.out})
			}
		}
	}
	for d := range days {
		for _, q := range Priorities {
			for k := range s.perDay[q] {
				i := idx[q]
				idx[q]++
				acq := t0 + int64(d)*daySec + hour + int64(k)*600
				o := Order{ID: "O", Origin: fmt.Sprintf("O/%s/%d/%d", q, d, k), Priority: q, Modality: mods[q], ReceivedSec: acq - 30*60, WantedSec: acq - 30*60, FirstApptSec: acq - 10*60}
				if acq > r.ObservedUntilSec {
					continue
				}
				r.Orders = append(r.Orders, o)
				if pm(i, s.seed+1, s.declinePM[q]) {
					outs = append(outs, served{o.ReceivedSec, OutLost})
					continue
				}
				outs = append(outs, served{acq, OutServed})
				st := Study{ID: "S", Priority: q, Modality: mods[q], AcquiredSec: acq, EnteredSec: acq}
				tg := testTargets[q] * 60
				tat := tg / 2
				if pm(i, s.seed, s.breachPMAt(q, acq)) {
					tat = tg + 30*60
					if pm(i, s.seed+3, s.neverPM[q]) {
						tat = -1
					}
				}
				if q == "routine" && pm(i, s.seed+5, s.remotePM) {
					st.Remote, st.ClaimedSec, st.ReadStartSec, st.FeeCents = true, acq+60, acq+660, s.fee
				} else {
					st.ClaimedSec, st.ReadStartSec = acq+60, acq+60
				}
				if tat >= 0 && acq+tat <= r.ObservedUntilSec {
					st.FinalSec = acq + tat
				}
				r.Studies = append(r.Studies, st)
			}
		}
		if s.rare > 0 && d%s.rare == 0 {
			at := t0 + int64(d)*daySec + 2*hour
			if at+hour <= r.ObservedUntilSec {
				r.Orders = append(r.Orders, Order{ID: "R", Origin: fmt.Sprintf("R/%d", d), Priority: "routine", Modality: "US", ReceivedSec: at, WantedSec: at})
				if s.rareLost {
					outs = append(outs, served{at, OutLost})
				} else {
					outs = append(outs, served{at + hour, OutServed})
				}
			}
		}
		for k := range s.walkIns {
			at := t0 + int64(d)*daySec + 3*hour + int64(k)*60
			if at > r.ObservedUntilSec {
				continue
			}
			r.Orders = append(r.Orders, Order{ID: "W", Origin: fmt.Sprintf("W/%d/%d", d, k), Priority: "routine", Modality: "XR", WalkIn: true, ReceivedSec: at, WantedSec: at})
			out := OutServed
			if pm(d*s.walkIns+k, s.seed, s.walkLeft) {
				out = OutLeft
			}
			outs = append(outs, served{at, out})
		}
	}
	for i := s.inherited; i < len(r.Orders); i++ {
		r.Orders[i].ResolvedSec = outs[i-s.inherited].at
	}
	for d := 1; d <= days; d++ {
		at := t0 + int64(d)*daySec
		m := s.money(at)
		r.Samples = append(r.Samples, s.sample(&r, d, at, m))
	}
	ws := windowsAt(s)
	for _, c := range ws {
		m := s.money(c.at)
		for _, x := range r.Studies {
			if x.Remote && x.FinalSec != 0 && x.FinalSec <= c.at {
				m.RemoteReadsCents += x.FeeCents
			}
		}
		if s.remotePM > 0 {
			m.RemoteRetainerCents = s.retainerPD * (c.at - t0) / daySec
		}
		var oo []string
		for i, o := range r.Orders {
			switch {
			case o.ReceivedSec > c.at:
				oo = append(oo, OutNotDisclosed)
			case i < s.inherited:
				oo = append(oo, OutServed)
			case outs[i-s.inherited].at != 0 && outs[i-s.inherited].at <= c.at:
				oo = append(oo, outs[i-s.inherited].out)
			default:
				oo = append(oo, OutOpen)
			}
		}
		r.Captures = append(r.Captures, Capture{Name: c.name, AtSec: c.at, OrderOutcomes: oo, Money: m})
	}
	return r
}

type capAt struct {
	name string
	at   int64
}

func windowsAt(s synth) []capAt {
	return []capAt{{"T30", t0 + 30*daySec}, {"T90", t0 + 90*daySec}, {"tailEnd", s.tailEnd()}, {"T180", t0 + 180*daySec}, {"end", s.observedUntil()}}
}

// sample counts the record's studies at at, as a recorder would.
func (s synth) sample(r *Record, d int, at int64, m Money) Sample {
	date, _ := simtime.ParseDate(t0Date)
	x := Sample{Day: d, AtSec: at, Date: date.AddDays(int64(d)).String(), ProfitCents: m.OperatingProfitCents, CashCents: m.CashCents,
		ObligationsCents: m.Obligations()}
	for _, st := range r.Studies {
		if st.AcquiredSec > t0 && st.AcquiredSec <= at {
			x.AcquiredCum++
		}
		if st.FinalSec != 0 && st.FinalSec <= at {
			if st.FinalSec > t0 {
				x.FinalCum++
			}
			continue
		}
		if st.AcquiredSec > at {
			continue
		}
		x.Unread++
		if at-st.AcquiredSec > testTargets[st.Priority]*60 {
			x.Overdue++
		}
	}
	return x
}

func records(n int, mk func(seed int64) synth) []Record {
	var out []Record
	for i := range n {
		out = append(out, mk(int64(901+i)).record())
	}
	return out
}

func testPolicy(t *testing.T) (*Policy, string) {
	t.Helper()
	p, d := policyNow(t)
	p.Scenario.Fingerprint = "fp" // the synthetic scenario
	return p, d
}

// policyNow is the policy in force, as committed.
func policyNow(t *testing.T) (*Policy, string) {
	t.Helper()
	p, d, err := LoadPolicy("../../" + PolicyPath)
	if err != nil {
		t.Fatal(err)
	}
	return p, d
}
