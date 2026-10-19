package sim_test

import (
	"encoding/json"
	"maps"
	"os"
	"slices"
	"sync"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/eval"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// The no-action access validation of Slice 5C.4b's acceptance-design
// closure (docs/calibration/clinical-5c4b/access-denominators.md): the
// evaluation scenario's practice with no action on the development seeds
// (as clinical_profile_test.go), measured by the v5 evaluator under the
// candidate policy and its access variants, with the simulator's own
// diagnosis of every lost order. It reconciles orders with acquisitions,
// classifies every order of the access denominators, compares the
// calendar and service-opportunity clocks, and checks the records'
// lifecycle data. No strategy runs: the world takes no action.

const candidatePolicy = "docs/calibration/clinical-5c4b/candidate-policy-v5.json"

// accessCell: one priority's access cohort under one limit.
type accessCell struct {
	LimitMin  int64   `json:"limitMin"`
	Wanted    int64   `json:"wanted"`
	Timely    int64   `json:"timely"`
	Untimely  int64   `json:"untimely"`
	Withdrawn int64   `json:"withdrawn"`
	Undecided int64   `json:"undecided"`
	Lost      int64   `json:"lost,omitempty"` // lost and left, apart (policies gating lost demand)
	Fraction  float64 `json:"timelyFraction"`
	// The untimely orders by what happened to them.
	UntimelyBy map[string]int64 `json:"untimelyByOutcome"`
}

type accessSeed struct {
	Seed int64 `json:"seed"`
	// Orders received after the takeover and by the horizon, by priority
	// and end outcome; walk-ins among them.
	Received map[string]map[string]int64 `json:"receivedByOutcome"`
	WalkIns  map[string]int64            `json:"walkIns"`
	// The simulator's diagnosis of those lost: primary cause by priority.
	LossCause map[string]map[string]int64 `json:"lossPrimaryCause"`
	// Lost at the first offer (received and lost within the minute) and
	// lost later (after a cancellation, no-show, reschedule or recall).
	LostFirstOffer map[string]int64 `json:"lostAtFirstOffer"`
	LostLater      map[string]int64 `json:"lostLater"`
	// Served orders imaged after their first appointment's start
	// (rebooked or rescheduled), and orders displaced at least once.
	Rescheduled map[string]int64 `json:"servedAfterFirstAppointment"`
	Displaced   map[string]int64 `json:"displaced"`
	// Acquisitions in the window by priority (the studies), and the
	// orders' served count: their difference is inherited and late work.
	Acquired map[string]int64 `json:"studiesAcquired"`
	// The gated window's access cohorts (wanted after the takeover and by
	// the horizon), per limit; and the inherited cohort under the
	// candidate's limits.
	Access    map[string][]accessCell `json:"access"`
	Inherited map[string]accessCell   `json:"inheritedAccess"`
	// Urgent served orders: the wait from the wanted time in calendar
	// hours and in imaging-open hours (percentiles 0, 50, 90, 99, 100),
	// and the share within 24 / 48 calendar hours, 12 / 24 open hours.
	UrgentCalendarH []float64          `json:"urgentServedCalendarWaitH"`
	UrgentServiceH  []float64          `json:"urgentServedServiceWaitH"`
	UrgentWithin    map[string]float64 `json:"urgentServedWithin"`
	// Urgent orders wanted on a Saturday from 12:00: their outcomes.
	UrgentSaturdayPM map[string]int64 `json:"urgentWantedSaturdayFrom12"`
	// Boundary: orders wanted in the horizon's last 7 days, by outcome at
	// the end of observation; orders received after the horizon.
	NearHorizon  map[string]int64 `json:"wantedLast7DaysByOutcome"`
	AfterHorizon int64            `json:"receivedAfterHorizon"`
	// Lifecycle data checks: served before wanted; served with no
	// appointment and not a walk-in; patient outcome without a time;
	// resolved but open; unknown priority.
	Anomalies map[string]int64 `json:"lifecycleAnomalies"`
	// The gated window's timely fraction under the candidate's limits.
	CandidateTimely map[string]float64 `json:"candidateTimelyFraction"`
	// The candidate's reporting-delay measures: absolute violations and
	// the worst past due (minutes), the severe-delay rate by per-seed
	// window, and the worst week's severe count.
	AbsoluteViolations map[string]int64 `json:"absoluteViolations"`
	// The volume-normalized weekly severity (when the policy has it).
	WeeklySevere    map[string]*eval.WeeklySevere `json:"weeklySevere,omitempty"`
	WorstPastDueMin map[string]int64              `json:"worstPastDueMin"`
	SevereRate      map[string]map[string]float64 `json:"severeRateByWindow"`
	WorstWeekSevere map[string]int64              `json:"worstWeekSevere"`
}

// openHours: the scenario's imaging hours, minutes after midnight, by
// weekday (Monday 0).
var openHours = [7][2]int64{{480, 1200}, {480, 1200}, {480, 1200}, {480, 1200}, {480, 1200}, {600, 1080}, {0, 0}}

// serviceSec is the imaging-open time in [a, b).
func serviceSec(date simtime.Date, t0 int64, a, b int64) int64 {
	mid0 := t0 - t0%86400
	s := int64(0)
	for d := floorDivDays(a-mid0, 86400); mid0+d*86400 < b; d++ {
		wd := (int(date.AddDays(d).Weekday()) + 6) % 7
		o, c := mid0+d*86400+openHours[wd][0]*60, mid0+d*86400+openHours[wd][1]*60
		s += max(0, min(b, c)-max(a, o))
	}
	return s
}

func pctF(xs []float64, ps ...int) []float64 {
	if len(xs) == 0 {
		return nil
	}
	s := slices.Clone(xs)
	slices.Sort(s)
	var out []float64
	for _, p := range ps {
		out = append(out, s[min(len(s)-1, p*(len(s)-1)/100)])
	}
	return out
}

func TestSimulatorAccessValidation(t *testing.T) {
	out := os.Getenv("DT_SIM_ACCESS")
	if out == "" {
		t.Skip("set DT_SIM_ACCESS=<file> for the no-action access validation (docs/calibration/clinical-5c4b/access-denominators.md)")
	}
	pol4, _, err := eval.LoadPolicy("../../" + eval.PolicyPath)
	if err != nil {
		t.Fatal(err)
	}
	cand, _, err := eval.LoadPolicy("../../" + candidatePolicy)
	if err != nil {
		t.Fatal(err)
	}
	seeds, err := eval.DevelopmentSeeds(pol4)
	if err != nil {
		t.Fatal(err)
	}
	cfg := profileConfig(t)
	grid := map[string][]int64{"urgent": {1440, 2880, 4320}, "soon": {3 * 1440, 7 * 1440, 10 * 1440}, "routine": {14 * 1440, 21 * 1440, 30 * 1440}}
	res := make([]accessSeed, len(seeds))
	var wg sync.WaitGroup
	for i, seed := range seeds {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var losses []sim.LossDiagnosis
			rec := recordBaselineProbe(pol4, cfg, seed, func(d sim.LossDiagnosis) { losses = append(losses, d) })
			res[i] = accessOf(t, cand, grid, rec, losses)
		}()
	}
	wg.Wait()
	// The probe only observes: the first seed's record is the same without
	// it.
	var probed []sim.LossDiagnosis
	a, _ := json.Marshal(recordBaseline(pol4, cfg, seeds[0]))
	b, _ := json.Marshal(recordBaselineProbe(pol4, cfg, seeds[0], func(d sim.LossDiagnosis) { probed = append(probed, d) }))
	if string(a) != string(b) || len(probed) == 0 {
		t.Fatalf("the loss probe changed the record (%d diagnoses)", len(probed))
	}
	doc := map[string]any{
		"schema":   "dreamteam.simaccess/v1",
		"source":   "the evaluation scenario's practice with no action, development seeds, recorded by the evaluation recorder; the simulator's loss diagnosis",
		"manifest": profileManifest,
		"policy":   candidatePolicy,
		"grid":     grid,
		"seeds":    res,
	}
	raw, err := json.MarshalIndent(doc, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

// recordBaselineProbe is recordBaseline with the loss probe attached from
// the takeover (the probe only observes).
func recordBaselineProbe(p *eval.Policy, cfg *sim.Config, seed int64, probe func(sim.LossDiagnosis)) eval.Record {
	return recordBaselineWith(p, cfg, seed, func(w *sim.World) { w.SetLossProbe(probe) })
}

func accessOf(t *testing.T, cand *eval.Policy, grid map[string][]int64, rec eval.Record, losses []sim.LossDiagnosis) accessSeed {
	t0, H, E := rec.TakeoverSec, rec.HorizonEndSec, rec.ObservedUntilSec
	date, _ := simtime.ParseDate(rec.TakeoverDate)
	end := rec.Captures[len(rec.Captures)-1].OrderOutcomes
	x := accessSeed{Seed: rec.Seed, Received: map[string]map[string]int64{}, WalkIns: map[string]int64{}, LossCause: map[string]map[string]int64{},
		LostFirstOffer: map[string]int64{}, LostLater: map[string]int64{}, Rescheduled: map[string]int64{}, Displaced: map[string]int64{},
		Acquired: map[string]int64{}, Access: map[string][]accessCell{}, Inherited: map[string]accessCell{}, UrgentWithin: map[string]float64{},
		UrgentSaturdayPM: map[string]int64{}, NearHorizon: map[string]int64{}, Anomalies: map[string]int64{},
		CandidateTimely: map[string]float64{}, AbsoluteViolations: map[string]int64{}, WorstPastDueMin: map[string]int64{},
		SevereRate: map[string]map[string]float64{}, WorstWeekSevere: map[string]int64{}}
	cause := map[string]string{}
	for _, d := range losses {
		cause[d.Order] = d.Primary
	}
	var cal, svc []float64
	var n24, n48, s12, s24, nServed float64
	for i, o := range rec.Orders {
		out := end[i]
		if !slices.Contains(eval.Priorities, o.Priority) {
			x.Anomalies["unknownPriority"]++
		}
		switch {
		case out == eval.OutServed && o.ResolvedSec < o.WantedSec:
			x.Anomalies["servedBeforeWanted"]++
		case out == eval.OutServed && o.FirstApptSec == 0 && !o.WalkIn:
			x.Anomalies["servedWithoutAppointment"]++
		case out == eval.OutPatient && o.ResolvedSec == 0:
			x.Anomalies["patientWithoutTime"]++
		case out == eval.OutOpen && o.ResolvedSec != 0:
			x.Anomalies["openButResolved"]++
		}
		if o.ReceivedSec > H {
			x.AfterHorizon++
			continue
		}
		if o.WantedSec > H-7*86400 && o.WantedSec <= H && o.ReceivedSec > t0 {
			x.NearHorizon[out]++
		}
		if o.ReceivedSec <= t0 {
			continue
		}
		q := o.Priority
		if x.Received[q] == nil {
			x.Received[q], x.LossCause[q] = map[string]int64{}, map[string]int64{}
		}
		x.Received[q][out]++
		if o.WalkIn {
			x.WalkIns[q]++
		}
		if len(o.DisplacementSec) > 0 {
			x.Displaced[q]++
		}
		if out == eval.OutLost {
			c := cause[o.ID]
			if c == "" {
				c = "(not diagnosed: lost after the first offer)"
			}
			x.LossCause[q][c]++
			if o.ResolvedSec-o.ReceivedSec < 60 {
				x.LostFirstOffer[q]++
			} else {
				x.LostLater[q]++
			}
		}
		if out == eval.OutServed && o.FirstApptSec != 0 && o.ResolvedSec > o.FirstApptSec+3600 {
			x.Rescheduled[q]++
		}
		if q == "urgent" {
			if out == eval.OutServed {
				w := float64(o.ResolvedSec-o.WantedSec) / 3600
				s := float64(serviceSec(date, t0, o.WantedSec, o.ResolvedSec)) / 3600
				cal, svc = append(cal, w), append(svc, s)
				nServed++
				if w <= 24 {
					n24++
				}
				if w <= 48 {
					n48++
				}
				if s <= 12 {
					s12++
				}
				if s <= 24 {
					s24++
				}
			}
			mid0 := t0 - t0%86400
			d := floorDivDays(o.WantedSec-mid0, 86400)
			if wd := (int(date.AddDays(d).Weekday()) + 6) % 7; wd == 5 && (o.WantedSec-mid0)%86400 >= 12*3600 {
				x.UrgentSaturdayPM[out]++
			}
		}
	}
	x.UrgentCalendarH, x.UrgentServiceH = pctF(cal, 0, 50, 90, 99, 100), pctF(svc, 0, 50, 90, 99, 100)
	if nServed > 0 {
		x.UrgentWithin = map[string]float64{"calendar24h": n24 / nServed, "calendar48h": n48 / nServed, "open12h": s12 / nServed, "open24h": s24 / nServed}
	}
	for _, s := range rec.Studies {
		if s.AcquiredSec > t0 && s.AcquiredSec <= H {
			x.Acquired[s.Priority]++
		}
	}
	// The v5 evaluator's cohorts under each limit of the grid.
	for _, q := range eval.Priorities {
		for _, lim := range grid[q] {
			c := *cand.Clinical
			c.AccessLimitMin = maps.Clone(c.AccessLimitMin)
			c.AccessLimitMin[q] = lim
			if !c.MaxWaitReportOnly {
				c.MaxWaitMin = maps.Clone(c.MaxWaitMin)
				c.MaxWaitMin[q] = max(c.MaxWaitMin[q], lim)
			}
			p := *cand
			p.Clinical = &c
			m := eval.Measure(&p, &rec)
			a := m.Clinical.Access[eval.WinGated][q]
			cell := accessCell{LimitMin: lim, Wanted: a.Wanted, Timely: a.Timely, Untimely: a.Untimely, Withdrawn: a.Withdrawn, Undecided: a.Undecided, Lost: a.Lost,
				Fraction: float64(a.Timely) / float64(max(1, a.Timely+a.Untimely)), UntimelyBy: map[string]int64{}}
			// The untimely orders by outcome (the evaluator's rule restated:
			// served late, open past the limit, patient after the limit, lost,
			// left).
			for i, o := range rec.Orders {
				if o.Priority != q || o.ReceivedSec <= t0 || o.WantedSec <= t0 || o.WantedSec > H {
					continue
				}
				out, l := end[i], lim*60
				switch {
				case out == eval.OutServed && o.ResolvedSec-o.WantedSec > l:
					cell.UntimelyBy["servedLate"]++
				case out == eval.OutOpen && E-o.WantedSec > l:
					cell.UntimelyBy["openPastLimit"]++
				case out == eval.OutPatient && cmpOr(o.ResolvedSec, E) >= o.WantedSec+l:
					cell.UntimelyBy["patientAfterLimit"]++
				case (out == eval.OutLost || out == eval.OutLeft) && cand.Clinical.LostDemand != nil:
					// counted apart (Lost), gated against the baseline
				case out == eval.OutLost:
					cell.UntimelyBy["lost"]++
				case out == eval.OutLeft:
					cell.UntimelyBy["left"]++
				}
			}
			var by int64
			for _, v := range cell.UntimelyBy {
				by += v
			}
			if by != cell.Untimely {
				t.Errorf("seed %d %s %d: untimely %d, by outcome %d", rec.Seed, q, lim, cell.Untimely, by)
			}
			x.Access[q] = append(x.Access[q], cell)
		}
	}
	m := eval.Measure(cand, &rec)
	for _, q := range eval.Priorities {
		a := m.Clinical.InheritedAccess[q]
		x.Inherited[q] = accessCell{LimitMin: cand.Clinical.AccessLimitMin[q], Wanted: a.Wanted, Timely: a.Timely, Untimely: a.Untimely,
			Withdrawn: a.Withdrawn, Undecided: a.Undecided, Fraction: float64(a.Timely) / float64(max(1, a.Timely+a.Untimely))}
		g := m.Clinical.Access[eval.WinGated][q]
		x.CandidateTimely[q] = float64(g.Timely) / float64(max(1, g.Timely+g.Untimely))
		x.WorstWeekSevere[q] = m.Clinical.WorstWeekSevere[q]
		x.SevereRate[q] = map[string]float64{}
		for _, w := range cand.Service.PerSeedWindows[q] {
			dc := m.Clinical.Delay[w][q]
			x.SevereRate[q][w] = float64(dc.Severe) / float64(max(1, dc.Acquired))
		}
	}
	x.WeeklySevere = m.Clinical.WeeklySevere
	for _, a := range m.Clinical.Absolute {
		if a.Limit == "pastDue" {
			x.AbsoluteViolations[a.Priority], x.WorstPastDueMin[a.Priority] = a.Violations, a.WorstMin
		}
	}
	return x
}

func cmpOr(a, b int64) int64 {
	if a != 0 {
		return a
	}
	return b
}
