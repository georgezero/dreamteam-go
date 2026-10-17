package eval

import (
	"fmt"
	"slices"
	"sort"
)

// compare pairs a candidate's seeds with another plan's. The seed sets,
// takeover, engine and scenario must be identical, or the comparison is
// invalid (it is not evidence).
func compare(p *Policy, cand, ag *PlanReport, cr, ar []Record) Comparison {
	c := Comparison{Candidate: cand.Plan, Against: ag.Plan, Valid: true, Integrity: []string{},
		Profit: map[string]Interval{}, Revenue: map[string]Interval{}, Reports: map[string]Interval{}, CashAtEnd: map[string]Interval{},
		BreachDelta: map[string]map[string]Interval{}, TATP90Delta: map[string]map[string]Interval{},
		WorstBreach: map[string]map[string][2]float64{}, RemoteCost: map[string]Interval{}, UnbilledDelta: map[string]Interval{}}
	bad := func(format string, args ...any) { c.Integrity = append(c.Integrity, fmt.Sprintf(format, args...)) }
	if !slices.Equal(cand.Seeds, ag.Seeds) {
		bad("seed sets differ: %v and %v", cand.Seeds, ag.Seeds)
	}
	ri := map[int64]*Record{}
	for i := range ar {
		ri[ar[i].Seed] = &ar[i]
	}
	for i := range cr {
		a, b := &cr[i], ri[cr[i].Seed]
		if b == nil {
			continue
		}
		if a.Engine != b.Engine || a.ScenarioFingerprint != b.ScenarioFingerprint || a.TakeoverSec != b.TakeoverSec {
			bad("seed %d: engine, scenario or takeover differ", a.Seed)
		}
	}
	if !cand.Valid || !ag.Valid {
		bad("a plan's records do not check out")
	}
	c.Valid = len(c.Integrity) == 0
	type pair struct{ a, b *SeedResult }
	var pairs []pair
	bySeed := map[int64]*SeedResult{}
	for i := range ag.PerSeed {
		bySeed[ag.PerSeed[i].Seed] = &ag.PerSeed[i]
	}
	for i := range cand.PerSeed {
		if b := bySeed[cand.PerSeed[i].Seed]; b != nil {
			pairs = append(pairs, pair{&cand.PerSeed[i], b})
		}
	}
	for _, w := range []string{WinT30, WinT90, WinOps, WinTail, WinT180} {
		var prof, rev, reps, cash, cost, unb []float64
		dq := map[string][]float64{}
		tq := map[string][]float64{}
		worst := map[string][2]float64{}
		complete := true
		for _, x := range pairs {
			wa, wb := x.a.Measures.Windows[w], x.b.Measures.Windows[w]
			if wa == nil || wb == nil {
				complete = false
				break
			}
			prof = append(prof, float64(wa.Money.OperatingProfitCents-wb.Money.OperatingProfitCents))
			rev = append(rev, float64(wa.Money.NetRevenueCents-wb.Money.NetRevenueCents))
			reps = append(reps, float64(wa.Remote.LocalReports+wa.Remote.RemoteReports-wb.Remote.LocalReports-wb.Remote.RemoteReports))
			cash = append(cash, float64(wa.Money.End.CashCents-wb.Money.End.CashCents))
			cost = append(cost, float64(wa.Remote.TotalCostCents-wb.Remote.TotalCostCents))
			unb = append(unb, float64(wa.Money.End.UnbilledCompletedCents-wb.Money.End.UnbilledCompletedCents))
			for _, q := range Priorities {
				ca, cb := wa.Cohorts[q], wb.Cohorts[q]
				if ca.BreachRate.Defined && cb.BreachRate.Defined {
					dq[q] = append(dq[q], ca.BreachRate.Value-cb.BreachRate.Value)
				}
				if ca.CompletedTATMin.Defined && cb.CompletedTATMin.Defined {
					tq[q] = append(tq[q], float64(ca.CompletedTATMin.P90-cb.CompletedTATMin.P90))
				}
				v := worst[q]
				v[0], v[1] = max(v[0], ca.BreachRate.Value), max(v[1], cb.BreachRate.Value)
				worst[q] = v
			}
		}
		if !complete || len(pairs) == 0 {
			continue
		}
		c.Profit[w], c.Revenue[w], c.Reports[w], c.CashAtEnd[w] = pairedT(prof), pairedT(rev), pairedT(reps), pairedT(cash)
		c.RemoteCost[w], c.UnbilledDelta[w] = pairedT(cost), pairedT(unb)
		c.BreachDelta[w], c.TATP90Delta[w], c.WorstBreach[w] = map[string]Interval{}, map[string]Interval{}, worst
		for _, q := range Priorities {
			if len(dq[q]) == len(pairs) {
				c.BreachDelta[w][q] = pairedT(dq[q])
			}
			if len(tq[q]) == len(pairs) {
				c.TATP90Delta[w][q] = pairedT(tq[q])
			}
		}
	}
	var mc []float64
	for _, x := range pairs {
		mc = append(mc, float64(x.a.Measures.MinCash-x.b.Measures.MinCash))
	}
	c.MinCashDelta = pairedT(mc)
	for _, h := range p.Payback.Horizons {
		c.AccrualPayback = append(c.AccrualPayback, payback(p, h, cr, ri, func(s Sample) int64 { return s.ProfitCents }))
		c.CashPayback = append(c.CashPayback, payback(p, h, cr, ri, func(s Sample) int64 { return s.CashCents }))
	}
	c.CaseSelection, c.CaseSelectionStatus = caseSelection(p, cand, ag)
	return c
}

// payback finds each seed's sustained break-even day within horizon h
// (days 1..h of the daily samples). Seeds without samples through h are
// left out; a seed still below zero at h is censored. The median is the
// smallest day by which at least half the seeds have repaid, and is not
// reported when fewer than half have.
func payback(p *Policy, h int, cr []Record, ri map[int64]*Record, val func(Sample) int64) Payback {
	pb := Payback{Horizon: h, Days: []int{}}
	recs := slices.Clone(cr)
	sort.Slice(recs, func(i, j int) bool { return recs[i].Seed < recs[j].Seed })
	for i := range recs {
		a, b := &recs[i], ri[recs[i].Seed]
		if b == nil {
			continue
		}
		inc := make([]int64, h+1)
		ok := true
		for d := 1; d <= h; d++ {
			sa, sb := sampleDay(a, d), sampleDay(b, d)
			if sa == nil || sb == nil {
				ok = false
				break
			}
			inc[d] = val(*sa) - val(*sb)
		}
		if !ok {
			continue
		}
		day := 0
		if inc[h] >= 0 {
			day = h
			for day > 1 && inc[day-1] >= 0 {
				day--
			}
		}
		pb.Days = append(pb.Days, day)
	}
	var repaid []int
	for _, d := range pb.Days {
		if d > 0 {
			repaid = append(repaid, d)
		}
	}
	slices.Sort(repaid)
	n := len(pb.Days)
	pb.Repaid, pb.Censored = len(repaid), n-len(repaid)
	pb.RepaidFrac = wilson(pb.Repaid, n)
	if len(repaid) > 0 {
		pb.Earliest, pb.Latest = repaid[0], repaid[len(repaid)-1]
	}
	if n > 0 && float64(len(repaid)) >= p.Payback.MedianMinRepaidFrac*float64(n) {
		k := (n + 1) / 2 // the smallest day by which ⌈n/2⌉ seeds have repaid
		pb.Median, pb.MedianDefined = repaid[k-1], true
	}
	return pb
}

func sampleDay(r *Record, d int) *Sample {
	if d-1 < len(r.Samples) && r.Samples[d-1].Day == d {
		return &r.Samples[d-1]
	}
	for i := range r.Samples {
		if r.Samples[i].Day == d {
			return &r.Samples[i]
		}
	}
	return nil
}

// caseSelection guards against passing service gates by serving fewer
// or easier cases: in every gated window, each priority's due-demand
// served fraction, per seed and pooled, each modality's pooled, each
// priority's acquisitions, and the loss and abandonment rates, against
// the paired plan.
func caseSelection(p *Policy, cand, ag *PlanReport) ([]GateSummary, string) {
	cs := p.CaseSelection
	bySeed := map[int64]*SeedResult{}
	for i := range ag.PerSeed {
		bySeed[ag.PerSeed[i].Seed] = &ag.PerSeed[i]
	}
	var out []GateSummary
	status := Pass
	add := func(g GateSummary) {
		out = append(out, g)
		if g.Status == Fail {
			status = Fail
		} else if g.Status == Insufficient && status == Pass {
			status = Insufficient
		}
	}
	for _, w := range p.Service.Gated {
		type tally struct{ dueA, srvA, dueB, srvB int64 }
		pri, mod := map[string]*tally{}, map[string]*tally{}
		acq := map[string][2]int64{}
		var lostA, refA, lostB, refB, leftA, wiA, leftB, wiB int64
		perSeed := map[string]*GateSummary{}
		n := 0
		for i := range cand.PerSeed {
			a := &cand.PerSeed[i]
			b := bySeed[a.Seed]
			if b == nil {
				continue
			}
			wa, wb := a.Measures.Windows[w], b.Measures.Windows[w]
			if wa == nil || wb == nil {
				add(GateSummary{Gate: "case_selection", Window: w, Role: "gated", Status: Insufficient})
				return out, status
			}
			n++
			xa, xb := wa.Access, wb.Access
			for _, q := range Priorities {
				t := pri[q]
				if t == nil {
					t = &tally{}
					pri[q] = t
				}
				t.dueA += xa.DueByPriority[q]
				t.srvA += xa.ServedByPri[q]
				t.dueB += xb.DueByPriority[q]
				t.srvB += xb.ServedByPri[q]
				v := acq[q]
				v[0] += wa.Cohorts[q].Acquired
				v[1] += wb.Cohorts[q].Acquired
				acq[q] = v
				g := perSeed[q]
				if g == nil {
					g = &GateSummary{Gate: "served_fraction_drop_per_seed", Window: w, Stratum: q, Role: "gated", Threshold: cs.PerSeedMaxServedDrop}
					perSeed[q] = g
				}
				g.Seeds++
				if xb.DueByPriority[q] < cs.MinStratumDue {
					g.InsufficientSeeds++ // too few due orders to judge a drop on this seed: pooled only
					continue
				}
				drop := ratio(xb.ServedByPri[q], xb.DueByPriority[q]).Value - ratio(xa.ServedByPri[q], xa.DueByPriority[q]).Value
				if drop > g.Worst || g.Passing+g.Failing == 0 {
					g.Worst, g.WorstSeed = drop, a.Seed
				}
				if drop > cs.PerSeedMaxServedDrop {
					g.Failing++
				} else {
					g.Passing++
				}
			}
			for md, d := range xb.DueByModality {
				t := mod[md]
				if t == nil {
					t = &tally{}
					mod[md] = t
				}
				t.dueB += d
				t.srvB += xb.ServedByMod[md]
			}
			for md, d := range xa.DueByModality {
				t := mod[md]
				if t == nil {
					t = &tally{}
					mod[md] = t
				}
				t.dueA += d
				t.srvA += xa.ServedByMod[md]
			}
			lostA += xa.LossRate.Num
			refA += xa.LossRate.Den
			lostB += xb.LossRate.Num
			refB += xb.LossRate.Den
			leftA += xa.AbandonRate.Num
			wiA += xa.AbandonRate.Den
			leftB += xb.AbandonRate.Num
			wiB += xb.AbandonRate.Den
		}
		lim := failLimit(p.Service.MaxFailingSeedFraction, n)
		for _, q := range Priorities {
			g := perSeed[q]
			if g == nil {
				continue
			}
			g.FailLimit = lim
			g.FailingFraction = wilson(g.Failing, g.Seeds)
			g.Status = Pass
			if g.Failing > lim {
				g.Status = Fail
			}
			add(*g)
		}
		drop := func(gate, stratum string, t *tally, thr float64) {
			g := GateSummary{Gate: gate, Window: w, Stratum: stratum, Role: "gated", Threshold: thr, Seeds: n}
			ra, rb := ratio(t.srvA, t.dueA), ratio(t.srvB, t.dueB)
			g.Pooled = ra
			g.Worst = rb.Value - ra.Value // the drop
			switch {
			case t.dueB < cs.MinStratumDue:
				g.Status = Pass // too rare to judge pooled; reported
				if t.dueB > 0 && t.dueA == 0 {
					g.Status = Fail // a stratum served before, now absent: dropped
				}
			case !ra.Defined:
				g.Status = Fail // demand that is due but no longer reaches the practice
			case g.Worst > thr:
				g.Status = Fail
			default:
				g.Status = Pass
			}
			add(g)
		}
		for _, q := range Priorities {
			drop("served_fraction_drop_pooled", q, pri[q], cs.PooledMaxServedDrop)
		}
		mods := make([]string, 0, len(mod))
		for md := range mod {
			mods = append(mods, md)
		}
		sort.Strings(mods)
		for _, md := range mods {
			drop("served_fraction_drop_pooled", md, mod[md], cs.PooledMaxServedDrop)
		}
		for _, q := range Priorities {
			v := acq[q]
			g := GateSummary{Gate: "acquired_drop_pooled", Window: w, Stratum: q, Role: "gated", Threshold: cs.MaxAcquiredDrop, Seeds: n}
			g.Pooled = ratio(v[0], v[1])
			g.Worst = 1 - g.Pooled.Value
			g.Status = Pass
			if !g.Pooled.Defined {
				g.Status = Insufficient
			} else if g.Worst > cs.MaxAcquiredDrop {
				g.Status = Fail
			}
			add(g)
		}
		rise := func(gate string, na, da, nb, db int64) {
			g := GateSummary{Gate: gate, Window: w, Role: "gated", Threshold: cs.MaxLossRateIncrease, Seeds: n}
			ra, rb := ratio(na, da), ratio(nb, db)
			g.Pooled = ra
			g.Worst = ra.Value - rb.Value
			g.Status = Pass
			if !ra.Defined && rb.Defined {
				g.Status = Insufficient
			} else if g.Worst > cs.MaxLossRateIncrease {
				g.Status = Fail
			}
			add(g)
		}
		rise("lost_referral_rate_increase", lostA, refA, lostB, refB)
		rise("walkin_abandon_rate_increase", leftA, wiA, leftB, wiB)
	}
	return out, status
}

// verdict applies the frozen decision rules (policy "verdict"): integrity,
// then the hard gates, then the financial test. Profit never compensates
// for a failed gate.
func verdict(p *Policy, c *Comparison, cand, base *PlanReport) {
	if !c.Valid {
		c.Verdict, c.Reasons = INVALID, append([]string{}, c.Integrity...)
		return
	}
	var why []string
	for _, g := range cand.ServiceGates {
		if g.Role == "gated" && g.Status != Pass {
			why = append(why, fmt.Sprintf("service %s %s %s: %s (pooled %.4f, %d failing and %d insufficient of %d, limit %d)",
				g.Gate, g.Window, g.Stratum, g.Status, g.Pooled.Value, g.Failing, g.InsufficientSeeds, g.Seeds, g.FailLimit))
		}
	}
	for _, g := range c.CaseSelection {
		if g.Status != Pass {
			why = append(why, fmt.Sprintf("case selection %s %s %s: %s (%.4f, threshold %.4f)", g.Gate, g.Window, g.Stratum, g.Status, g.Worst, g.Threshold))
		}
	}
	if cand.BacklogStatus != Pass {
		why = append(why, fmt.Sprintf("backlog: %d of %d seeds stable", cand.BacklogStable, len(cand.PerSeed)))
	}
	if cand.SolvencyStatus != Pass {
		why = append(why, fmt.Sprintf("solvency: cash below the floor on seeds %v", cand.InsolventSeeds))
	}
	if len(why) > 0 {
		c.Verdict, c.Reasons = NOGO, why
		return
	}
	t90, t180 := c.Profit[WinT90], c.Profit[WinT180]
	benefit := t90.Defined && t90.Lo > 0
	validated := cand.ValidationPass && t180.Defined && t180.Mean > 0
	complete := len(cand.FundingGapSeeds) == 0
	switch {
	case benefit && validated && complete:
		c.Verdict = GO
		return
	case t90.Mean > 0:
		c.Verdict = CONDITIONAL
		if !benefit {
			why = append(why, fmt.Sprintf("T90 profit over %s not significant: %.0f [%.0f, %.0f] cents", c.Against, t90.Mean, t90.Lo, t90.Hi))
		}
	case !base.Eligible:
		c.Verdict = CONDITIONAL
		why = append(why, fmt.Sprintf("meets the gates %s does not, at a T90 cost of %.0f [%.0f, %.0f] cents", c.Against, -t90.Mean, -t90.Hi, -t90.Lo))
	default:
		c.Verdict = NOGO
		why = append(why, fmt.Sprintf("no T90 profit over %s (%.0f cents), which already meets the gates", c.Against, t90.Mean))
	}
	if !validated {
		why = append(why, "T180 validation not met (service or profit)")
	}
	if !complete {
		why = append(why, fmt.Sprintf("obligations exceed cash at the tail's end on seeds %v", cand.FundingGapSeeds))
	}
	c.Reasons = why
}
