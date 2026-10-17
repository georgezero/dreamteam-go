package eval

import (
	"fmt"
	"slices"

	"github.com/georgezero/dreamteam-go/internal/simtime"
)

const daySec = 86400

// Cohort is one priority's studies acquired in a window (a, b]: every one
// of them, finished or not. Breach status is decided at each study's due
// time (acquisition + target), observed through the closure after b, so
// it does not depend on where the window ends; completed-only turnaround
// is reported beside it and never decides a gate.
type Cohort struct {
	Acquired           int64 `json:"acquired"`
	Finalized          int64 `json:"finalized"`  // by b
	Unfinished         int64 `json:"unfinished"` // at b
	CompletionFraction Ratio `json:"completionFraction"`
	CompletedTATMin    Dist  `json:"completedTatMin"`     // final by b
	UnfinishedAgeMin   Dist  `json:"unfinishedAgeMin"`    // at b
	OverdueUnfinished  int64 `json:"overdueUnfinished"`   // at b, past target
	OverdueStudyMin    int64 `json:"overdueStudyMinutes"` // time past due within the window, all studies
	Breaches           int64 `json:"breaches"`
	Unresolved         int64 `json:"unresolved"` // due after the observation ended, not final: undecided
	BreachRate         Ratio `json:"breachRate"` // breaches ÷ acquired; undefined if empty or unresolved
	// The rule of the historical reports (calib_test.go tatByPriority): as
	// of b, completed over target plus unfinished past it, ÷ acquired.
	LegacyBreaches   int64 `json:"legacyBreaches"`
	LegacyBreachRate Ratio `json:"legacyBreachRate"`
}

// Access is the demand received in a window, its outcomes at the window's
// end. Due demand has its original wanted time at or before the end.
type Access struct {
	Received        int64            `json:"received"`
	Due             int64            `json:"due"`
	NotYetDue       int64            `json:"notYetDue"`
	Served          int64            `json:"served"` // of due
	Lost            int64            `json:"lostReferrals"`
	WalkIns         int64            `json:"walkIns"`
	WalkInsLeft     int64            `json:"walkInsLeft"`
	PatientCaused   int64            `json:"patientCaused"`
	UnresolvedDue   int64            `json:"unresolvedDue"`
	DisplacedOrders int64            `json:"displacedOrders"`
	Displacements   int64            `json:"displacementEvents"`
	ServedFraction  Ratio            `json:"servedFraction"`
	LossRate        Ratio            `json:"lostReferralRate"`  // ÷ referrals received
	AbandonRate     Ratio            `json:"walkInAbandonRate"` // ÷ walk-ins received
	BookingLeadHr   Dist             `json:"bookingLeadHours"`  // received → first appointment
	DueByPriority   map[string]int64 `json:"dueByPriority"`
	ServedByPri     map[string]int64 `json:"servedByPriority"`
	DueByModality   map[string]int64 `json:"dueByModality"`
	ServedByMod     map[string]int64 `json:"servedByModality"`
}

// Remote is the reading mix of reports finalized in the window.
type Remote struct {
	LocalReports      int64 `json:"localReports"`
	RemoteReports     int64 `json:"remoteReports"`
	RemoteClaims      int64 `json:"remoteClaims"`
	HandoffWaitMin    Dist  `json:"handoffWaitMin"`
	RemoteTATMin      Dist  `json:"remoteTatMin"`
	LocalTATMin       Dist  `json:"localTatMin"`
	RetainerCents     int64 `json:"retainerCents"`
	ReadFeeCents      int64 `json:"readFeeCents"`  // ledger
	ClaimFeeCents     int64 `json:"claimFeeCents"` // Σ locked fees of the remote reports
	TotalCostCents    int64 `json:"totalCostCents"`
	CostPerRemoteCent Ratio `json:"costPerRemoteReportCents"`
}

// WindowMoney is the accrual P&L of the window and the position at its end.
type WindowMoney struct {
	NetRevenueCents      int64 `json:"netRevenueCents"`
	OperatingProfitCents int64 `json:"operatingProfitCents"`
	Margin               Ratio `json:"margin"` // profit ÷ net revenue (cents)
	End                  Money `json:"end"`
	// Obligations at the end beyond cash on hand: liabilities recognized,
	// wages earned but not posted, remote fees committed and contract
	// minimums still owed. FundingGap = max(0, obligations − cash).
	ObligationsCents int64 `json:"obligationsCents"`
	FundingGapCents  int64 `json:"fundingGapCents"`
}

// WindowMeasures is one window of one seed.
type WindowMeasures struct {
	From    int64              `json:"fromSec"`
	To      int64              `json:"toSec"`
	Cohorts map[string]*Cohort `json:"cohorts"`
	Access  Access             `json:"access"`
	Remote  Remote             `json:"remote"`
	Money   WindowMoney        `json:"money"`
}

// Inherited is the backlog and orders outstanding at the takeover,
// separate from new demand.
type Inherited struct {
	Studies         map[string]*Cohort `json:"studies"` // as of T90, breach decided at due
	Orders          int64              `json:"orders"`
	OrdersServedT90 int64              `json:"ordersServedT90"`
	OrdersLostT90   int64              `json:"ordersLostT90"`
	OrdersOpenT90   int64              `json:"ordersOpenT90"`
	LastFinalSec    int64              `json:"lastFinalSec"`
	UnfinishedAtT90 int64              `json:"unfinishedAtT90"`
}

// Block is one rota-aligned block of daily samples.
type Block struct {
	FirstDay   int     `json:"firstDay"`
	LastDay    int     `json:"lastDay"`
	FirstDate  string  `json:"firstDate"`
	Mean       float64 `json:"meanBacklog"`
	StageMeans struct {
		AwaitingDoc   float64 `json:"awaitingDocumentation"`
		Unread        float64 `json:"unread"`
		RemoteHandoff float64 `json:"remoteHandoff"`
		LocalReading  float64 `json:"localReading"`
		RemoteReading float64 `json:"remoteReading"`
	} `json:"stageMeans"`
	OverdueFraction Ratio   `json:"overdueFraction"` // Σ overdue ÷ Σ backlog over the block's samples
	AgeP50MinMean   float64 `json:"ageP50MinMean"`
	AgeP90MinMean   float64 `json:"ageP90MinMean"`
	AgeMaxMin       int64   `json:"ageMaxMin"`
	Acquired        int64   `json:"acquired"`
	Finalized       int64   `json:"finalized"`
}

// BacklogMeasures is the rota-aligned stability test of one seed.
type BacklogMeasures struct {
	Status      string  `json:"status"` // stable, unstable, insufficient
	Reason      string  `json:"reason,omitempty"`
	Reference   Block   `json:"reference"` // the last whole block ending by the primary endpoint
	Final       Block   `json:"final"`     // the last whole block ending by the tail's end
	Growth      float64 `json:"growth"`    // final − reference mean
	Tolerance   float64 `json:"tolerance"`
	SlopePerDay float64 `json:"slopePerDay"` // least squares over the reference start → final end
	// RollingMeans: the 28-day trailing mean at each block boundary.
	RollingMeans []float64 `json:"rollingMeansAtBlockEnds"`
	// Legacy: the historical single-sample rule (backlog at day 180 ≤
	// 1.1 × day 90), kept for comparison only.
	LegacyDay90   int  `json:"legacyDay90"`
	LegacyDay180  int  `json:"legacyDay180"`
	LegacyHolds   bool `json:"legacyHolds"`
	LegacyDefined bool `json:"legacyDefined"`
}

// SeedMeasures is everything measured of one record.
type SeedMeasures struct {
	Seed      int64                      `json:"seed"`
	TailEnd   int64                      `json:"tailEndSec"`
	Windows   map[string]*WindowMeasures `json:"windows"`
	Inherited Inherited                  `json:"inherited"`
	Backlog   BacklogMeasures            `json:"backlog"`
	MinCash   int64                      `json:"minCashCents"` // through the end of observation
	MinCashAt int64                      `json:"minCashSec"`
	PastDue   int64                      `json:"pastDueCents"` // the most at any capture
	Integrity []string                   `json:"integrity"`    // failed checks: the record is not evidence
}

// tailEnd is the frozen-policy tail's end (V0.2_DESIGN §13.1).
func tailEnd(p *Policy, r *Record) int64 {
	t90 := r.TakeoverSec + int64(p.Horizons.PrimaryDays)*daySec
	end := t90 + int64(p.Horizons.Tail.MinDaysAfterPrimary)*daySec
	if r.LatestEffectSec > 0 {
		end = max(end, r.LatestEffectSec+int64(p.Horizons.Tail.DaysAfterEffect)*daySec)
	}
	return end
}

// windows are the window bounds, and the capture at each end.
func windows(p *Policy, r *Record) map[string][3]int64 {
	t0 := r.TakeoverSec
	d := func(n int) int64 { return t0 + int64(n)*daySec }
	h := p.Horizons
	return map[string][3]int64{
		WinT30:  {t0, d(h.FeedbackDays)},
		WinT90:  {t0, d(h.PrimaryDays)},
		WinOps:  {d(h.OperatingFrom), d(h.PrimaryDays)},
		WinTail: {d(h.PrimaryDays), tailEnd(p, r)},
		WinT180: {t0, d(h.ValidationDays)},
	}
}

func captureName(w string) string {
	switch w {
	case WinOps:
		return WinT90
	case WinTail:
		return "tailEnd"
	}
	return w
}

// Measure computes a record's measures. It never fails: a record that
// does not check out carries its failures in Integrity.
func Measure(p *Policy, r *Record) *SeedMeasures {
	m := &SeedMeasures{Seed: r.Seed, TailEnd: tailEnd(p, r), Windows: map[string]*WindowMeasures{}, Integrity: []string{}}
	bad := func(format string, args ...any) { m.Integrity = append(m.Integrity, fmt.Sprintf(format, args...)) }
	caps := map[string]*Capture{}
	for i := range r.Captures {
		c := &r.Captures[i]
		caps[c.Name] = c
		if len(c.OrderOutcomes) != len(r.Orders) {
			bad("capture %s: %d order outcomes for %d orders", c.Name, len(c.OrderOutcomes), len(r.Orders))
		}
		if !c.Money.CashBridgeReconciles {
			bad("capture %s: the cash bridge does not reconcile", c.Name)
		}
		m.PastDue = max(m.PastDue, c.Money.PastDueCents)
	}
	switch {
	case r.Schema != RecordSchema:
		bad("schema %q", r.Schema)
	case p.Scenario.Fingerprint != "" && r.ScenarioFingerprint != p.Scenario.Fingerprint:
		bad("scenario %s is not the policy's", r.ScenarioFingerprint)
	}
	for _, q := range Priorities {
		if r.TATTargetMin[q] != p.Scenario.TATTargetMin[q] {
			bad("%s target %d min, the policy freezes %d", q, r.TATTargetMin[q], p.Scenario.TATTargetMin[q])
		}
	}
	ws := windows(p, r)
	t90 := ws[WinT90][1]
	if r.LastActionSec >= t90 {
		bad("an action at %d is not before the primary endpoint %d", r.LastActionSec, t90)
	}
	if c := caps["tailEnd"]; c != nil && c.AtSec != m.TailEnd {
		bad("tailEnd captured at %d, the rule gives %d", c.AtSec, m.TailEnd)
	}
	end := caps["end"]
	if end == nil || end.AtSec != r.ObservedUntilSec {
		bad("no capture at the end of observation")
	} else {
		m.MinCash, m.MinCashAt = end.Money.MinCashCents, end.Money.MinCashSec
	}
	closure := p.Horizons.ClosureMin * 60
	for name, b := range ws {
		c := caps[captureName(name)]
		if c == nil || c.AtSec != b[1] {
			if slices.Contains(p.Service.Gated, name) {
				bad("window %s: no capture at its end", name)
			}
			continue
		}
		if slices.Contains(p.Service.Gated, name) && r.ObservedUntilSec < b[1]+closure {
			bad("window %s: observed to %d, its cohorts close at %d", name, r.ObservedUntilSec, b[1]+closure)
		}
		var from *Capture
		if b[0] != r.TakeoverSec {
			for i := range r.Captures {
				if r.Captures[i].AtSec == b[0] {
					from = &r.Captures[i]
				}
			}
			if from == nil {
				bad("window %s: no capture at its start", name)
				continue
			}
		}
		m.Windows[name] = measureWindow(r, b[0], b[1], from, c)
	}
	m.Inherited = measureInherited(r, t90, caps[WinT90])
	m.Backlog = measureBacklog(p, r, t90, m.TailEnd)
	reconcile(r, m, bad)
	slices.Sort(m.Integrity)
	return m
}

func target(r *Record, q string) int64 { return r.TATTargetMin[q] * 60 }

// cohort measures studies acquired in (a, b] of one priority.
func cohort(r *Record, xs []*Study, b int64) *Cohort {
	c := &Cohort{}
	var tats, ages []int64
	for _, s := range xs {
		tg := target(r, s.Priority)
		due := s.AcquiredSec + tg
		c.Acquired++
		done := s.FinalSec != 0 && s.FinalSec <= b
		if done {
			c.Finalized++
			tats = append(tats, (s.FinalSec-s.AcquiredSec)/60)
			if s.FinalSec-s.AcquiredSec > tg {
				c.LegacyBreaches++
			}
		} else {
			c.Unfinished++
			age := b - s.AcquiredSec
			ages = append(ages, age/60)
			if age > tg {
				c.OverdueUnfinished++
				c.LegacyBreaches++
			}
		}
		switch {
		case s.FinalSec != 0 && s.FinalSec <= due:
		case s.FinalSec != 0 || due <= r.ObservedUntilSec:
			c.Breaches++
		default:
			c.Unresolved++
		}
		stop := b
		if s.FinalSec != 0 {
			stop = min(stop, s.FinalSec)
		}
		if stop > due {
			c.OverdueStudyMin += (stop - due) / 60
		}
	}
	c.CompletionFraction = ratio(c.Finalized, c.Acquired)
	c.CompletedTATMin, c.UnfinishedAgeMin = dist(tats), dist(ages)
	if c.Unresolved == 0 {
		c.BreachRate = ratio(c.Breaches, c.Acquired)
	} else {
		c.BreachRate = Ratio{Num: c.Breaches, Den: c.Acquired}
	}
	c.LegacyBreachRate = ratio(c.LegacyBreaches, c.Acquired)
	return c
}

func measureWindow(r *Record, a, b int64, from, to *Capture) *WindowMeasures {
	w := &WindowMeasures{From: a, To: b, Cohorts: map[string]*Cohort{}}
	by := map[string][]*Study{}
	var rtat, ltat, wait []int64
	for i := range r.Studies {
		s := &r.Studies[i]
		if s.AcquiredSec > a && s.AcquiredSec <= b {
			by[s.Priority] = append(by[s.Priority], s)
		}
		if s.Remote && s.ClaimedSec > a && s.ClaimedSec <= b {
			w.Remote.RemoteClaims++
			if s.ReadStartSec != 0 {
				wait = append(wait, (s.ReadStartSec-s.ClaimedSec)/60)
			}
		}
		if s.FinalSec > a && s.FinalSec <= b {
			tat := (s.FinalSec - s.AcquiredSec) / 60
			if s.Remote {
				w.Remote.RemoteReports++
				w.Remote.ClaimFeeCents += s.FeeCents
				rtat = append(rtat, tat)
			} else {
				w.Remote.LocalReports++
				ltat = append(ltat, tat)
			}
		}
	}
	for _, q := range Priorities {
		w.Cohorts[q] = cohort(r, by[q], b)
	}
	rm := &w.Remote
	rm.HandoffWaitMin, rm.RemoteTATMin, rm.LocalTATMin = dist(wait), dist(rtat), dist(ltat)
	var m0 Money
	if from != nil {
		m0 = from.Money
	}
	m1 := to.Money
	rm.RetainerCents = m1.RemoteRetainerCents - m0.RemoteRetainerCents
	rm.ReadFeeCents = m1.RemoteReadsCents - m0.RemoteReadsCents
	rm.TotalCostCents = rm.RetainerCents + rm.ReadFeeCents
	rm.CostPerRemoteCent = ratio(rm.TotalCostCents, rm.RemoteReports)
	wm := &w.Money
	wm.NetRevenueCents = m1.NetRevenueCents - m0.NetRevenueCents
	wm.OperatingProfitCents = m1.OperatingProfitCents - m0.OperatingProfitCents
	wm.Margin = ratio(wm.OperatingProfitCents, wm.NetRevenueCents)
	if wm.NetRevenueCents <= 0 {
		wm.Margin = Ratio{Num: wm.OperatingProfitCents, Den: wm.NetRevenueCents}
	}
	wm.End = m1
	wm.ObligationsCents = m1.PayablesCents + m1.AccruedPayrollCents + m1.SeverancePayableCents +
		m1.EarnedUnpostedWagesCents + m1.RemoteInFlightCents + m1.ContractMinimumCents + m1.PastDueCents
	wm.FundingGapCents = max(0, wm.ObligationsCents-m1.CashCents)
	w.Access = access(r, a, b, to)
	return w
}

func access(r *Record, a, b int64, c *Capture) Access {
	x := Access{DueByPriority: map[string]int64{}, ServedByPri: map[string]int64{}, DueByModality: map[string]int64{}, ServedByMod: map[string]int64{}}
	var lead []int64
	var referrals int64
	for i := range r.Orders {
		o := &r.Orders[i]
		if o.ReceivedSec <= a || o.ReceivedSec > b || i >= len(c.OrderOutcomes) {
			continue
		}
		out := c.OrderOutcomes[i]
		x.Received++
		if o.WalkIn {
			x.WalkIns++
			if out == OutLeft {
				x.WalkInsLeft++
			}
		} else {
			referrals++
			if out == OutLost {
				x.Lost++
			}
		}
		if o.Displacements > 0 {
			x.DisplacedOrders++
			x.Displacements += int64(o.Displacements)
		}
		if o.FirstApptSec != 0 {
			lead = append(lead, (o.FirstApptSec-o.ReceivedSec)/3600)
		}
		if o.WantedSec > b {
			x.NotYetDue++
			continue
		}
		x.Due++
		x.DueByPriority[o.Priority]++
		x.DueByModality[o.Modality]++
		switch out {
		case OutServed:
			x.Served++
			x.ServedByPri[o.Priority]++
			x.ServedByMod[o.Modality]++
		case OutPatient:
			x.PatientCaused++
		case OutOpen:
			x.UnresolvedDue++
		}
	}
	x.ServedFraction = ratio(x.Served, x.Due)
	x.LossRate = ratio(x.Lost, referrals)
	x.AbandonRate = ratio(x.WalkInsLeft, x.WalkIns)
	x.BookingLeadHr = dist(lead)
	return x
}

func measureInherited(r *Record, t90 int64, c *Capture) Inherited {
	in := Inherited{Studies: map[string]*Cohort{}}
	by := map[string][]*Study{}
	for i := range r.Studies {
		s := &r.Studies[i]
		if s.AcquiredSec <= r.TakeoverSec {
			by[s.Priority] = append(by[s.Priority], s)
			in.LastFinalSec = max(in.LastFinalSec, s.FinalSec)
			if s.FinalSec == 0 || s.FinalSec > t90 {
				in.UnfinishedAtT90++
			}
		}
	}
	for _, q := range Priorities {
		in.Studies[q] = cohort(r, by[q], t90)
	}
	for i := range r.Orders {
		if r.Orders[i].ReceivedSec > r.TakeoverSec {
			continue
		}
		in.Orders++
		if c == nil || i >= len(c.OrderOutcomes) {
			continue
		}
		switch c.OrderOutcomes[i] {
		case OutServed:
			in.OrdersServedT90++
		case OutLost, OutLeft:
			in.OrdersLostT90++
		case OutOpen:
			in.OrdersOpenT90++
		}
	}
	return in
}

// measureBacklog applies the rota-aligned rule: whole blocks of
// BlockDays daily samples, each starting on a rota cycle boundary (a
// week-A Monday); the reference block is the last one ending by the
// primary endpoint, the final block the last one ending by the tail's end.
func measureBacklog(p *Policy, r *Record, t90, tail int64) BacklogMeasures {
	bm := BacklogMeasures{Status: "insufficient"}
	anchor, err := simtime.ParseDate(p.Scenario.RotaWeekA)
	if err != nil {
		bm.Reason = "no rota anchor"
		return bm
	}
	byDate := map[int64]*Sample{}
	for i := range r.Samples {
		s := &r.Samples[i]
		d, err := simtime.ParseDate(s.Date)
		if err != nil {
			bm.Reason = "sample date " + s.Date
			return bm
		}
		byDate[d.Ordinal()] = s
	}
	last := func(until int64) (Block, []*Sample, bool) {
		var best []*Sample
		for i := range r.Samples {
			s := &r.Samples[i]
			d, _ := simtime.ParseDate(s.Date)
			if mod(d.Ordinal()-anchor.Ordinal(), int64(p.Backlog.RotaCycleDays)) != 0 {
				continue
			}
			var blk []*Sample
			for k := range int64(p.Backlog.BlockDays) {
				x := byDate[d.Ordinal()+k]
				if x == nil || x.AtSec > until {
					blk = nil
					break
				}
				blk = append(blk, x)
			}
			if blk != nil {
				best = blk
			}
		}
		if best == nil {
			return Block{}, nil, false
		}
		return block(r, best, byDate), best, true
	}
	ref, rs, ok1 := last(t90)
	fin, fs, ok2 := last(tail)
	switch {
	case !ok1 || !ok2:
		bm.Reason = "no whole rota-aligned block"
		return bm
	case fs[0].Day <= rs[len(rs)-1].Day:
		bm.Reason = "the final block overlaps the reference block"
		return bm
	}
	bm.Reference, bm.Final = ref, fin
	bm.Growth = fin.Mean - ref.Mean
	bm.Tolerance = max(p.Backlog.GrowthFraction*ref.Mean, p.Backlog.GrowthStudies)
	bm.Status = "stable"
	if bm.Growth > bm.Tolerance {
		bm.Status = "unstable"
	}
	var xs, ys []float64
	for i := range r.Samples {
		s := &r.Samples[i]
		if s.Day >= rs[0].Day && s.Day <= fs[len(fs)-1].Day {
			xs, ys = append(xs, float64(s.Day)), append(ys, float64(s.Backlog()))
		}
	}
	bm.SlopePerDay = slope(xs, ys)
	n := p.Backlog.BlockDays
	for i := range r.Samples {
		s := &r.Samples[i]
		d, _ := simtime.ParseDate(s.Date)
		if i+1 >= n && mod(d.Ordinal()+1-anchor.Ordinal(), int64(p.Backlog.RotaCycleDays)) == 0 {
			var sum float64
			for _, x := range r.Samples[i+1-n : i+1] {
				sum += float64(x.Backlog())
			}
			bm.RollingMeans = append(bm.RollingMeans, sum/float64(n))
		}
	}
	for i := range r.Samples {
		switch s := &r.Samples[i]; s.Day {
		case 90:
			bm.LegacyDay90 = s.Backlog()
		case 180:
			bm.LegacyDay180 = s.Backlog()
			bm.LegacyDefined = true
		}
	}
	bm.LegacyHolds = bm.LegacyDefined && float64(bm.LegacyDay180) <= 1.1*float64(bm.LegacyDay90)
	return bm
}

func block(r *Record, ss []*Sample, byDate map[int64]*Sample) Block {
	b := Block{FirstDay: ss[0].Day, LastDay: ss[len(ss)-1].Day, FirstDate: ss[0].Date}
	var over, tot int64
	n := float64(len(ss))
	for _, s := range ss {
		b.Mean += float64(s.Backlog()) / n
		b.StageMeans.AwaitingDoc += float64(s.AwaitingDoc) / n
		b.StageMeans.Unread += float64(s.Unread) / n
		b.StageMeans.RemoteHandoff += float64(s.RemoteHandoff) / n
		b.StageMeans.LocalReading += float64(s.LocalReading) / n
		b.StageMeans.RemoteReading += float64(s.RemoteReading) / n
		b.AgeP50MinMean += float64(s.AgeP50Min) / n
		b.AgeP90MinMean += float64(s.AgeP90Min) / n
		b.AgeMaxMin = max(b.AgeMaxMin, s.AgeMaxMin)
		over += int64(s.Overdue)
		tot += int64(s.Backlog())
	}
	b.OverdueFraction = ratio(over, tot)
	// Flow over the block: from the sample before it (the takeover's
	// zero if it is the first) to its last sample.
	var a0, f0 int
	for i := range r.Samples {
		if r.Samples[i].Day == ss[0].Day-1 {
			a0, f0 = r.Samples[i].AcquiredCum, r.Samples[i].FinalCum
		}
	}
	b.Acquired = int64(ss[len(ss)-1].AcquiredCum - a0)
	b.Finalized = int64(ss[len(ss)-1].FinalCum - f0)
	return b
}

func mod(a, m int64) int64 { return (a%m + m) % m }

func slope(xs, ys []float64) float64 {
	n := float64(len(xs))
	if n < 2 {
		return 0
	}
	var sx, sy, sxx, sxy float64
	for i := range xs {
		sx += xs[i]
		sy += ys[i]
		sxx += xs[i] * xs[i]
		sxy += xs[i] * ys[i]
	}
	den := n*sxx - sx*sx
	if den == 0 {
		return 0
	}
	return (n*sxy - sx*sy) / den
}

// reconcile cross-checks the record's sources: the daily samples against
// the study timestamps (backlog and flow), and the remote reports'
// locked fees against the ledger's read-fee expense.
func reconcile(r *Record, m *SeedMeasures, bad func(string, ...any)) {
	for i := range r.Samples {
		s := &r.Samples[i]
		var open, acq, fin int
		for j := range r.Studies {
			x := &r.Studies[j]
			if x.AcquiredSec <= s.AtSec && (x.FinalSec == 0 || x.FinalSec > s.AtSec) {
				open++
			}
			if x.AcquiredSec > r.TakeoverSec && x.AcquiredSec <= s.AtSec {
				acq++
			}
			if x.FinalSec > r.TakeoverSec && x.FinalSec <= s.AtSec {
				fin++
			}
		}
		if open != s.Backlog() || acq != s.AcquiredCum || fin != s.FinalCum {
			bad("day %d: sample backlog %d, acquired %d, final %d; studies give %d, %d, %d", s.Day, s.Backlog(), s.AcquiredCum, s.FinalCum, open, acq, fin)
			return
		}
	}
	for name, w := range m.Windows {
		if w.Remote.ClaimFeeCents != w.Remote.ReadFeeCents {
			bad("window %s: remote reports' fees %d, ledger read fees %d", name, w.Remote.ClaimFeeCents, w.Remote.ReadFeeCents)
		}
	}
}
