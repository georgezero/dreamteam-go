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

// Access is a window's demand, every order followed to its outcome at the
// end of observation (the same instant for every plan of an evaluation),
// so an order is judged wherever it was received and however long it
// stays open.
//
// Due demand is every new order (received after the takeover) whose
// original wanted time is in (a, b], received in the window or before it;
// the served fraction is served ÷ due, an order still open at the end of
// observation counting as not served. Received demand (the loss and
// abandonment rates) is every order received in (a, b].
type Access struct {
	Due                int64            `json:"due"`
	DueReceivedEarlier int64            `json:"dueReceivedBeforeWindow"` // carried across the window's start
	Served             int64            `json:"served"`                  // of due, imaged by the end of observation
	ServedByWindowEnd  int64            `json:"servedByWindowEnd"`       // of due, imaged by b
	LostDue            int64            `json:"lostDue"`
	PatientCaused      int64            `json:"patientCaused"`
	UnresolvedDue      int64            `json:"unresolvedDue"` // open at the end of observation
	ServedFraction     Ratio            `json:"servedFraction"`
	DueBy              map[string]int64 `json:"dueByStratum"` // stratum key (strata.go) → due
	ServedBy           map[string]int64 `json:"servedByStratum"`
	// CarriedIn: due by a and still open at a; where they ended.
	CarriedIn          int64 `json:"carriedIn"`
	CarriedInServed    int64 `json:"carriedInServed"`
	CarriedInOpenAtEnd int64 `json:"carriedInOpenAtEnd"`

	Received        int64 `json:"received"`
	Referrals       int64 `json:"referrals"`
	Lost            int64 `json:"lostReferrals"`
	WalkIns         int64 `json:"walkIns"`
	WalkInsLeft     int64 `json:"walkInsLeft"`
	LossRate        Ratio `json:"lostReferralRate"`  // ÷ referrals received
	AbandonRate     Ratio `json:"walkInAbandonRate"` // ÷ walk-ins received
	NotDueByHorizon int64 `json:"receivedNotDueByHorizon"`
	// Displacements in (a, b], by when they happened.
	DisplacedOrders int64 `json:"displacedOrders"`
	Displacements   int64 `json:"displacementEvents"`
	BookingLeadHr   Dist  `json:"bookingLeadHours"` // received → first appointment
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
	// The obligation bridge at the end (Money.Obligations) and its excess
	// over cash on hand.
	ObligationsCents int64 `json:"obligationsCents"`
	FundingGapCents  int64 `json:"fundingGapCents"`
}

// WindowMeasures is one window of one seed.
type WindowMeasures struct {
	From       int64              `json:"fromSec"`
	To         int64              `json:"toSec"`
	Cohorts    map[string]*Cohort `json:"cohorts"`
	AcquiredBy map[string]int64   `json:"acquiredByStratum"`
	Access     Access             `json:"access"`
	Remote     Remote             `json:"remote"`
	Money      WindowMoney        `json:"money"`
}

// Inherited is the backlog and orders outstanding at the takeover,
// separate from new demand: never in a new-demand cohort or denominator,
// followed to the end of observation, and guarded as its own stratum.
type Inherited struct {
	Studies             map[string]*Cohort `json:"studies"` // as of the horizon, breach decided at due
	UnfinishedAtT90     int64              `json:"unfinishedAtT90"`
	UnfinishedAtHorizon int64              `json:"unfinishedAtHorizon"`
	LastFinalSec        int64              `json:"lastFinalSec"`
	Orders              int64              `json:"orders"`
	OpenAtT90           int64              `json:"ordersOpenAtT90"`
	// Of the orders wanted by the horizon, outcomes at the end of
	// observation, and the wait past the wanted time (to the outcome, or
	// to the end of observation while open).
	Due              int64 `json:"due"`
	Served           int64 `json:"served"`
	Lost             int64 `json:"lost"`
	PatientCaused    int64 `json:"patientCaused"`
	OpenAtEnd        int64 `json:"openAtEnd"`
	WaitPastWantedHr Dist  `json:"waitPastWantedHours"`
}

// Block is one rota-aligned block of daily samples. Sums are of the daily
// values, kept so that tolerances compare exactly.
type Block struct {
	FirstDay   int     `json:"firstDay"`
	LastDay    int     `json:"lastDay"`
	FirstDate  string  `json:"firstDate"`
	Mean       float64 `json:"meanBacklog"`
	Max        int64   `json:"maxBacklog"`
	StageMeans struct {
		AwaitingDoc   float64 `json:"awaitingDocumentation"`
		Unread        float64 `json:"unread"`
		RemoteHandoff float64 `json:"remoteHandoff"`
		LocalReading  float64 `json:"localReading"`
		RemoteReading float64 `json:"remoteReading"`
	} `json:"stageMeans"`
	OverdueMean     float64 `json:"overdueMean"`        // unfinished past due
	StaleMean       float64 `json:"staleMean"`          // unfinished more than staleAfterDueMin past due
	ExposureMeanMin float64 `json:"overdueExposureMin"` // Σ minutes past due of the unfinished, per sample
	PersistentMax   int64   `json:"persistentMax"`      // the most persistent studies at a sample
	OverdueFraction Ratio   `json:"overdueFraction"`    // Σ overdue ÷ Σ backlog over the block's samples
	AgeP50MinMean   float64 `json:"ageP50MinMean"`
	AgeP90MinMean   float64 `json:"ageP90MinMean"`
	AgeMaxMin       int64   `json:"ageMaxMin"`
	Acquired        int64   `json:"acquired"`
	Finalized       int64   `json:"finalized"`

	n, sum, overdue, stale, exposure int64
}

// BlockCheck is one backlog check of a later block against the reference.
type BlockCheck struct {
	FirstDay  int     `json:"firstDay"`
	Check     string  `json:"check"` // mean, max, overdue, stale, exposure
	Value     float64 `json:"value"` // the block's
	Reference float64 `json:"reference"`
	Tolerance float64 `json:"tolerance"`
	Pass      bool    `json:"pass"`
}

// BacklogMeasures is the backlog test of one seed (README §4): every whole
// rota-aligned block after the reference, through the one covering the
// horizon, against the reference block, and no persistent study.
type BacklogMeasures struct {
	Status    string       `json:"status"` // stable, unstable, insufficient
	Reasons   []string     `json:"reasons,omitempty"`
	Reference Block        `json:"reference"` // the last whole block ending by the primary endpoint
	Later     []Block      `json:"later"`     // the last covers the horizon
	Checks    []BlockCheck `json:"checks"`
	// Persistent: studies unfinished at a sample more than the persistent
	// limit past due, through the covering block (inherited included).
	Persistent     int64   `json:"persistentStudies"`
	WorstPastDueHr float64 `json:"worstPastDueHours"`
	SlopePerDay    float64 `json:"slopePerDay"` // least squares over the reference start → covering end
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
	TailEnd   int64                      `json:"tailEndSec"`         // the common horizon
	Required  int64                      `json:"requiredTailEndSec"` // this record's own tail rule
	Windows   map[string]*WindowMeasures `json:"windows"`
	Inherited Inherited                  `json:"inherited"`
	Backlog   BacklogMeasures            `json:"backlog"`
	MinCash   int64                      `json:"minCashCents"` // through the end of observation
	MinCashAt int64                      `json:"minCashSec"`
	PastDue   int64                      `json:"pastDueCents"` // the most at any capture
	// FundingGap: the largest excess of obligations over cash at any
	// capture or daily sample through the end of observation, and when.
	FundingGap   int64    `json:"fundingGapCents"`
	FundingGapAt int64    `json:"fundingGapSec"`
	Integrity    []string `json:"integrity"` // failed checks: the record is not evidence
}

// TailEnd is a plan's own tail end (V0.2_DESIGN §13.1, policy
// horizons.tail): max(T90 + min tail, latest effect + days after it,
// latest commitment end). An evaluation's common horizon is the latest of
// its plans' (a harness computes it before running, from the plans'
// accepted actions).
func TailEnd(p *Policy, takeoverSec, latestEffectSec, latestCommitmentSec int64) int64 {
	t90 := takeoverSec + int64(p.Horizons.PrimaryDays)*daySec
	end := t90 + int64(p.Horizons.Tail.MinDaysAfterPrimary)*daySec
	if latestEffectSec > 0 {
		end = max(end, latestEffectSec+int64(p.Horizons.Tail.DaysAfterEffect)*daySec)
	}
	return max(end, latestCommitmentSec)
}

func requiredTail(p *Policy, r *Record) int64 {
	return TailEnd(p, r.TakeoverSec, r.LatestEffectSec, r.LatestCommitmentSec)
}

// ObservationEnd is how long a harness must observe every plan of an
// evaluation with the common horizon (observationNeeded).
func ObservationEnd(p *Policy, takeoverSec int64, takeoverDate string, horizonEndSec int64) int64 {
	return observationNeeded(p, &Record{TakeoverSec: takeoverSec, TakeoverDate: takeoverDate, HorizonEndSec: horizonEndSec})
}

// windows are the window bounds, from the record's common horizon.
func windows(p *Policy, r *Record) map[string][2]int64 {
	t0 := r.TakeoverSec
	d := func(n int) int64 { return t0 + int64(n)*daySec }
	h := p.Horizons
	return map[string][2]int64{
		WinT30:   {t0, d(h.FeedbackDays)},
		WinT90:   {t0, d(h.PrimaryDays)},
		WinOps:   {d(h.OperatingFrom), d(h.PrimaryDays)},
		WinTail:  {d(h.PrimaryDays), r.HorizonEndSec},
		WinGated: {t0, r.HorizonEndSec},
		WinT180:  {t0, d(h.ValidationDays)},
	}
}

// Captures a record must have, each exactly once.
var captureNames = []string{"T30", "T90", "tailEnd", "T180", "end"}

func captureName(w string) string {
	switch w {
	case WinOps:
		return WinT90
	case WinTail, WinGated:
		return "tailEnd"
	}
	return w
}

// observationNeeded is how far a record must be observed: every cohort
// closed (the horizon and T180, plus the closure) and the backlog block
// covering the horizon sampled.
func observationNeeded(p *Policy, r *Record) int64 {
	t180 := r.TakeoverSec + int64(p.Horizons.ValidationDays)*daySec
	need := max(r.HorizonEndSec, t180) + p.Horizons.ClosureMin*60
	if day, ok := coverDay(p, r); ok {
		need = max(need, r.TakeoverSec+int64(day)*daySec)
	}
	return need
}

// Measure computes a record's measures. It never fails: a record that
// does not check out carries its failures in Integrity.
func Measure(p *Policy, r *Record) *SeedMeasures {
	m := &SeedMeasures{Seed: r.Seed, TailEnd: r.HorizonEndSec, Required: requiredTail(p, r), Windows: map[string]*WindowMeasures{}, Integrity: []string{}}
	bad := func(format string, args ...any) { m.Integrity = append(m.Integrity, fmt.Sprintf(format, args...)) }
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
	if r.HorizonEndSec < m.Required {
		bad("the common horizon %d does not cover this record's tail end %d", r.HorizonEndSec, m.Required)
	}
	if need := observationNeeded(p, r); r.ObservedUntilSec < need {
		bad("observed to %d, the horizon's cohorts and backlog blocks need %d", r.ObservedUntilSec, need)
	}
	caps := checkCaptures(p, r, ws, bad)
	checkSamples(r, bad)
	checkStudies(r, bad)
	checkOrders(r, caps, bad)
	if end := caps["end"]; end != nil {
		m.MinCash, m.MinCashAt = end.Money.MinCashCents, end.Money.MinCashSec
		if m.MinCashAt < r.TakeoverSec || m.MinCashAt > r.ObservedUntilSec {
			bad("minimum cash at %d, outside the observation", m.MinCashAt)
		}
	}
	gap := func(at, owed, cash int64) {
		if g := owed - cash; g > m.FundingGap {
			m.FundingGap, m.FundingGapAt = g, at
		}
	}
	for _, c := range r.Captures {
		m.PastDue = max(m.PastDue, c.Money.PastDueCents)
		gap(c.AtSec, c.Money.Obligations(), c.Money.CashCents)
		if caps["end"] != nil && c.Money.CashCents < m.MinCash {
			bad("capture %s: cash %d below the recorded minimum %d", c.Name, c.Money.CashCents, m.MinCash)
		}
	}
	for _, s := range r.Samples {
		gap(s.AtSec, s.ObligationsCents, s.CashCents)
		if caps["end"] != nil && s.CashCents < m.MinCash {
			bad("day %d: cash %d below the recorded minimum %d", s.Day, s.CashCents, m.MinCash)
		}
	}
	end := caps["end"]
	for name, b := range ws {
		c := caps[captureName(name)]
		if c == nil || end == nil {
			continue
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
		m.Windows[name] = measureWindow(r, b[0], b[1], from, c, end)
	}
	if end != nil {
		m.Inherited = measureInherited(r, t90, caps["T90"], end)
	}
	m.Backlog = measureBacklog(p, r, t90)
	reconcile(r, m, bad)
	slices.Sort(m.Integrity)
	return m
}

// checkCaptures requires each capture exactly once at its instant.
func checkCaptures(p *Policy, r *Record, ws map[string][2]int64, bad func(string, ...any)) map[string]*Capture {
	at := map[string]int64{"T30": ws[WinT30][1], "T90": ws[WinT90][1], "tailEnd": r.HorizonEndSec, "T180": ws[WinT180][1], "end": r.ObservedUntilSec}
	caps := map[string]*Capture{}
	for i := range r.Captures {
		c := &r.Captures[i]
		switch want, ok := at[c.Name]; {
		case !ok:
			bad("capture %q is not a horizon", c.Name)
			continue
		case caps[c.Name] != nil:
			bad("capture %s twice", c.Name)
			continue
		case c.AtSec != want:
			bad("capture %s at %d, its instant is %d", c.Name, c.AtSec, want)
			continue
		}
		caps[c.Name] = c
		if len(c.OrderOutcomes) != len(r.Orders) {
			bad("capture %s: %d order outcomes for %d orders", c.Name, len(c.OrderOutcomes), len(r.Orders))
			delete(caps, c.Name)
			continue
		}
		if !c.Money.CashBridgeReconciles {
			bad("capture %s: the cash bridge does not reconcile", c.Name)
		}
	}
	for _, n := range captureNames {
		if caps[n] == nil {
			bad("no capture %s", n)
		}
	}
	return caps
}

// checkSamples requires one sample a day, days 1 … through the end of
// observation, each at the takeover's clock time on its date.
func checkSamples(r *Record, bad func(string, ...any)) {
	date, err := simtime.ParseDate(r.TakeoverDate)
	if err != nil {
		bad("takeover date %q", r.TakeoverDate)
		return
	}
	days := int((r.ObservedUntilSec - r.TakeoverSec) / daySec)
	if len(r.Samples) != days {
		bad("%d daily samples, the observation has %d days", len(r.Samples), days)
	}
	for i := range min(len(r.Samples), days) {
		s := &r.Samples[i]
		d := i + 1
		if s.Day != d || s.AtSec != r.TakeoverSec+int64(d)*daySec || s.Date != date.AddDays(int64(d)).String() {
			bad("sample %d is day %d at %d (%s), want day %d at %d (%s)", i, s.Day, s.AtSec, s.Date, d, r.TakeoverSec+int64(d)*daySec, date.AddDays(int64(d)))
			return
		}
	}
}

// checkStudies requires every study's timestamps in order and inside the
// observation, and an inherited study unfinished at the takeover.
func checkStudies(r *Record, bad func(string, ...any)) {
	for i := range r.Studies {
		s := &r.Studies[i]
		ok := slices.Contains(Priorities, s.Priority) && s.AcquiredSec > 0 && s.AcquiredSec <= r.ObservedUntilSec &&
			(s.EnteredSec == 0 || s.EnteredSec >= s.AcquiredSec) && s.FinalSec <= r.ObservedUntilSec &&
			(s.ClaimedSec == 0 || (s.EnteredSec != 0 && s.ClaimedSec >= s.EnteredSec)) &&
			(s.ReadStartSec == 0 || (s.ClaimedSec != 0 && s.ReadStartSec >= s.ClaimedSec)) &&
			(s.FinalSec == 0 || (s.ReadStartSec != 0 && s.FinalSec >= s.ReadStartSec)) &&
			(s.AcquiredSec > r.TakeoverSec || s.FinalSec == 0 || s.FinalSec > r.TakeoverSec)
		if !ok {
			bad("study %s: timestamps out of order or outside the observation", s.ID)
			return
		}
	}
}

// checkOrders requires every order's outcome at each capture to agree
// with its resolution time: undisclosed before receipt, open until
// resolved, then its outcome for good. The one outcome that can be
// undone is a no-show (patient), whose patient may call for a new date:
// it may precede a later resolution.
func checkOrders(r *Record, caps map[string]*Capture, bad func(string, ...any)) {
	end := caps["end"]
	for i := range r.Orders {
		o := &r.Orders[i]
		if o.ReceivedSec > r.ObservedUntilSec || o.WantedSec < o.ReceivedSec || (o.ResolvedSec != 0 && (o.ResolvedSec < o.ReceivedSec || o.ResolvedSec > r.ObservedUntilSec)) ||
			!slices.IsSorted(o.DisplacementSec) || slices.ContainsFunc(o.DisplacementSec, func(t int64) bool { return t < o.ReceivedSec || t > r.ObservedUntilSec }) {
			bad("order %s: timestamps out of order or outside the observation", o.ID)
			return
		}
		for _, n := range captureNames {
			c := caps[n]
			if c == nil {
				continue
			}
			out := c.OrderOutcomes[i]
			var ok bool
			switch {
			case o.ReceivedSec > c.AtSec:
				ok = out == OutNotDisclosed
			case o.ResolvedSec == 0 || o.ResolvedSec > c.AtSec:
				ok = out == OutOpen || out == OutPatient
			default:
				ok = out != OutOpen && out != OutNotDisclosed && (end == nil || out == end.OrderOutcomes[i])
			}
			if !ok {
				bad("order %s: outcome %q at %s disagrees with its resolution at %d", o.ID, out, n, o.ResolvedSec)
				return
			}
		}
	}
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

func measureWindow(r *Record, a, b int64, from, to, end *Capture) *WindowMeasures {
	w := &WindowMeasures{From: a, To: b, Cohorts: map[string]*Cohort{}, AcquiredBy: map[string]int64{}}
	by := map[string][]*Study{}
	var rtat, ltat, wait []int64
	for i := range r.Studies {
		s := &r.Studies[i]
		if s.AcquiredSec > a && s.AcquiredSec <= b {
			by[s.Priority] = append(by[s.Priority], s)
			for _, k := range strataOf(s.Priority, s.Modality) {
				w.AcquiredBy[k]++
			}
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
	wm.ObligationsCents = m1.Obligations()
	wm.FundingGapCents = max(0, wm.ObligationsCents-m1.CashCents)
	w.Access = access(r, a, b, from, to, end)
	return w
}

// access measures the window's due and received demand, each order's
// outcome taken at the end of observation (end); at b (to) for the
// served-by-window-end count, and at a (from) for what was carried in.
func access(r *Record, a, b int64, from, to, end *Capture) Access {
	x := Access{DueBy: map[string]int64{}, ServedBy: map[string]int64{}}
	var lead []int64
	for i := range r.Orders {
		o := &r.Orders[i]
		if o.ReceivedSec <= r.TakeoverSec {
			continue // inherited: its own population
		}
		out := end.OrderOutcomes[i]
		if o.WantedSec > a && o.WantedSec <= b {
			x.Due++
			if o.ReceivedSec <= a {
				x.DueReceivedEarlier++
			}
			ks := strataOf(o.Priority, o.Modality)
			for _, k := range ks {
				x.DueBy[k]++
			}
			switch out {
			case OutServed:
				x.Served++
				for _, k := range ks {
					x.ServedBy[k]++
				}
				if to.OrderOutcomes[i] == OutServed {
					x.ServedByWindowEnd++
				}
			case OutLost, OutLeft:
				x.LostDue++
			case OutPatient:
				x.PatientCaused++
			case OutOpen:
				x.UnresolvedDue++
			}
		}
		if from != nil && o.WantedSec <= a && from.OrderOutcomes[i] == OutOpen {
			x.CarriedIn++
			switch out {
			case OutServed:
				x.CarriedInServed++
			case OutOpen:
				x.CarriedInOpenAtEnd++
			}
		}
		for _, t := range o.DisplacementSec {
			if t > a && t <= b {
				x.Displacements++
			}
		}
		if slices.ContainsFunc(o.DisplacementSec, func(t int64) bool { return t > a && t <= b }) {
			x.DisplacedOrders++
		}
		if o.ReceivedSec <= a || o.ReceivedSec > b {
			continue
		}
		x.Received++
		if o.WalkIn {
			x.WalkIns++
			if out == OutLeft {
				x.WalkInsLeft++
			}
		} else {
			x.Referrals++
			if out == OutLost {
				x.Lost++
			}
		}
		if o.FirstApptSec != 0 {
			lead = append(lead, (o.FirstApptSec-o.ReceivedSec)/3600)
		}
		if o.WantedSec > r.HorizonEndSec {
			x.NotDueByHorizon++
		}
	}
	x.ServedFraction = ratio(x.Served, x.Due)
	x.LossRate = ratio(x.Lost, x.Referrals)
	x.AbandonRate = ratio(x.WalkInsLeft, x.WalkIns)
	x.BookingLeadHr = dist(lead)
	return x
}

func measureInherited(r *Record, t90 int64, c90, end *Capture) Inherited {
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
			if s.FinalSec == 0 || s.FinalSec > r.HorizonEndSec {
				in.UnfinishedAtHorizon++
			}
		}
	}
	for _, q := range Priorities {
		in.Studies[q] = cohort(r, by[q], r.HorizonEndSec)
	}
	var wait []int64
	for i := range r.Orders {
		o := &r.Orders[i]
		if o.ReceivedSec > r.TakeoverSec {
			continue
		}
		in.Orders++
		if c90 != nil && c90.OrderOutcomes[i] == OutOpen {
			in.OpenAtT90++
		}
		if o.WantedSec > r.HorizonEndSec {
			continue
		}
		in.Due++
		stop := o.ResolvedSec
		switch end.OrderOutcomes[i] {
		case OutServed:
			in.Served++
		case OutLost, OutLeft:
			in.Lost++
		case OutPatient:
			in.PatientCaused++
		case OutOpen:
			in.OpenAtEnd++
			stop = end.AtSec
		}
		wait = append(wait, max(0, stop-o.WantedSec)/3600)
	}
	in.WaitPastWantedHr = dist(wait)
	return in
}

// coverDay is the last day of the backlog block covering the horizon: the
// first whole rota-aligned block after the reference block that ends on
// or after the horizon's day.
func coverDay(p *Policy, r *Record) (int, bool) {
	blocks, ok := blockStarts(p, r)
	if !ok || len(blocks) < 2 {
		return 0, false
	}
	return blocks[len(blocks)-1] + p.Backlog.BlockDays - 1, true
}

// blockStarts are the first days of the reference block and every later
// whole block through the covering one (the blocks follow each other).
func blockStarts(p *Policy, r *Record) ([]int, bool) {
	anchor, err1 := simtime.ParseDate(p.Scenario.RotaWeekA)
	date, err2 := simtime.ParseDate(r.TakeoverDate)
	if err1 != nil || err2 != nil {
		return nil, false
	}
	n, cycle := p.Backlog.BlockDays, int64(p.Backlog.RotaCycleDays)
	ref := 0
	for d := 1; d+n-1 <= p.Horizons.PrimaryDays; d++ {
		if mod(date.AddDays(int64(d)).Ordinal()-anchor.Ordinal(), cycle) == 0 {
			ref = d
		}
	}
	if ref == 0 {
		return nil, false
	}
	hday := int((r.HorizonEndSec - r.TakeoverSec + daySec - 1) / daySec)
	out := []int{ref}
	for s := ref + n; ; s += n {
		out = append(out, s)
		if s+n-1 >= hday {
			return out, true
		}
	}
}

// pastDue is a sample's unfinished studies by lateness: overdue (past
// due), stale and persistent counts, and Σ minutes past due.
type pastDue struct{ overdue, stale, persistent, exposureMin, worstMin int64 }

func samplePastDue(p *Policy, r *Record, at int64) pastDue {
	var x pastDue
	stale, persistent := p.Backlog.StaleAfterDueMin*60, int64(p.Backlog.PersistentAfterDueDays)*daySec
	for i := range r.Studies {
		s := &r.Studies[i]
		if s.AcquiredSec > at || (s.FinalSec != 0 && s.FinalSec <= at) {
			continue
		}
		late := at - s.AcquiredSec - target(r, s.Priority)
		if late <= 0 {
			continue
		}
		x.overdue++
		x.exposureMin += late / 60
		x.worstMin = max(x.worstMin, late/60)
		if late > stale {
			x.stale++
		}
		if late > persistent {
			x.persistent++
		}
	}
	return x
}

// measureBacklog applies the backlog rule (README §4): whole blocks of
// BlockDays daily samples, each starting on a rota cycle boundary (a
// week-A Monday). The reference block is the last one ending by the
// primary endpoint; every later block, through the first that ends on or
// after the horizon's day, must stay within tolerance of it in its mean
// and peak backlog, its mean overdue and stale counts and its overdue
// exposure; and no study may be persistent at any sample through then.
func measureBacklog(p *Policy, r *Record, t90 int64) BacklogMeasures {
	bm := BacklogMeasures{Status: "insufficient"}
	starts, ok := blockStarts(p, r)
	if !ok {
		bm.Reasons = []string{"no whole rota-aligned reference block"}
		return bm
	}
	n := p.Backlog.BlockDays
	last := starts[len(starts)-1] + n - 1
	if last > len(r.Samples) {
		bm.Reasons = []string{fmt.Sprintf("samples end before day %d, the end of the block covering the horizon", last)}
		return bm
	}
	pd := make([]pastDue, last+1)
	for d := 1; d <= last; d++ {
		pd[d] = samplePastDue(p, r, r.Samples[d-1].AtSec)
		bm.Persistent = max(bm.Persistent, pd[d].persistent)
		bm.WorstPastDueHr = max(bm.WorstPastDueHr, float64(pd[d].worstMin)/60)
	}
	blocks := make([]Block, len(starts))
	for i, s := range starts {
		blocks[i] = block(r, r.Samples[s-1:s-1+n], pd[s:s+n])
	}
	ref := blocks[0]
	bm.Reference, bm.Later = ref, blocks[1:]
	bm.Status = "stable"
	fr, fl := bp(p.Backlog.Growth), p.Backlog.GrowthStudies
	check := func(b Block, name string, sum, rsum, floor int64, nn int64) {
		c := BlockCheck{FirstDay: b.FirstDay, Check: name, Value: float64(sum) / float64(nn), Reference: float64(rsum) / float64(nn),
			Tolerance: max(float64(fr)*float64(rsum)/10_000/float64(nn), float64(floor)), Pass: !growthAbove(sum, rsum, nn, fr, floor)}
		bm.Checks = append(bm.Checks, c)
		if !c.Pass {
			bm.Status = "unstable"
			bm.Reasons = append(bm.Reasons, fmt.Sprintf("block from day %d: %s %.2f against %.2f, tolerance %.2f", b.FirstDay, name, c.Value, c.Reference, c.Tolerance))
		}
	}
	for _, b := range bm.Later {
		check(b, "mean", b.sum, ref.sum, fl, b.n)
		check(b, "max", b.Max, ref.Max, fl, 1)
		check(b, "overdue", b.overdue, ref.overdue, fl, b.n)
		check(b, "stale", b.stale, ref.stale, fl, b.n)
		check(b, "exposure", b.exposure, ref.exposure, p.Backlog.ExposureFloorMin, b.n)
	}
	if bm.Persistent > 0 {
		bm.Status = "unstable"
		bm.Reasons = append(bm.Reasons, fmt.Sprintf("%d studies unfinished more than %d days past due (worst %.1f h)", bm.Persistent, p.Backlog.PersistentAfterDueDays, bm.WorstPastDueHr))
	}
	var xs, ys []float64
	for _, s := range r.Samples[starts[0]-1 : last] {
		xs, ys = append(xs, float64(s.Day)), append(ys, float64(s.Backlog()))
	}
	bm.SlopePerDay = slope(xs, ys)
	anchor, _ := simtime.ParseDate(p.Scenario.RotaWeekA)
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

func block(r *Record, ss []Sample, pd []pastDue) Block {
	b := Block{FirstDay: ss[0].Day, LastDay: ss[len(ss)-1].Day, FirstDate: ss[0].Date, n: int64(len(ss))}
	n := float64(len(ss))
	for i, s := range ss {
		b.sum += int64(s.Backlog())
		b.Max = max(b.Max, int64(s.Backlog()))
		b.StageMeans.AwaitingDoc += float64(s.AwaitingDoc) / n
		b.StageMeans.Unread += float64(s.Unread) / n
		b.StageMeans.RemoteHandoff += float64(s.RemoteHandoff) / n
		b.StageMeans.LocalReading += float64(s.LocalReading) / n
		b.StageMeans.RemoteReading += float64(s.RemoteReading) / n
		b.AgeP50MinMean += float64(s.AgeP50Min) / n
		b.AgeP90MinMean += float64(s.AgeP90Min) / n
		b.AgeMaxMin = max(b.AgeMaxMin, s.AgeMaxMin)
		b.overdue += pd[i].overdue
		b.stale += pd[i].stale
		b.exposure += pd[i].exposureMin
		b.PersistentMax = max(b.PersistentMax, pd[i].persistent)
	}
	b.Mean, b.OverdueMean = float64(b.sum)/n, float64(b.overdue)/n
	b.StaleMean, b.ExposureMeanMin = float64(b.stale)/n, float64(b.exposure)/n
	b.OverdueFraction = ratio(b.overdue, b.sum)
	// Flow over the block: from the sample before it (the takeover's
	// zero if it is the first) to its last sample.
	var a0, f0 int
	if d := ss[0].Day; d > 1 {
		a0, f0 = r.Samples[d-2].AcquiredCum, r.Samples[d-2].FinalCum
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
// the study timestamps (backlog, overdue and flow), and the remote
// reports' locked fees against the ledger's read-fee expense.
func reconcile(r *Record, m *SeedMeasures, bad func(string, ...any)) {
	for i := range r.Samples {
		s := &r.Samples[i]
		var open, acq, fin, over int
		for j := range r.Studies {
			x := &r.Studies[j]
			if x.AcquiredSec <= s.AtSec && (x.FinalSec == 0 || x.FinalSec > s.AtSec) {
				open++
				if s.AtSec-x.AcquiredSec > target(r, x.Priority) {
					over++
				}
			}
			if x.AcquiredSec > r.TakeoverSec && x.AcquiredSec <= s.AtSec {
				acq++
			}
			if x.FinalSec > r.TakeoverSec && x.FinalSec <= s.AtSec {
				fin++
			}
		}
		if open != s.Backlog() || acq != s.AcquiredCum || fin != s.FinalCum || over != s.Overdue {
			bad("day %d: sample backlog %d, overdue %d, acquired %d, final %d; studies give %d, %d, %d, %d",
				s.Day, s.Backlog(), s.Overdue, s.AcquiredCum, s.FinalCum, open, over, acq, fin)
			return
		}
	}
	for name, w := range m.Windows {
		if w.Remote.ClaimFeeCents != w.Remote.ReadFeeCents {
			bad("window %s: remote reports' fees %d, ledger read fees %d", name, w.Remote.ClaimFeeCents, w.Remote.ReadFeeCents)
		}
	}
}
