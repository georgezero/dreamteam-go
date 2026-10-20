package eval

import (
	"cmp"
	"fmt"
	"slices"
	"sort"
	"strings"
)

// compare pairs a candidate's seeds with another plan's. The seed sets,
// takeover, engine and scenario must be identical, or the comparison is
// invalid (it is not evidence); so it is with any evaluation-wide failure
// in global (Evaluate requires one horizon and observation end for every
// record).
func compare(p *Policy, cand, ag *PlanReport, cr, ar []Record, global []string) Comparison {
	c := Comparison{Candidate: cand.Plan, Against: ag.Plan, Valid: true, Integrity: []string{},
		Profit: map[string]Interval{}, WorstProfit: map[string]WorstSeed{}, Revenue: map[string]Interval{}, Reports: map[string]Interval{},
		CashAtEnd: map[string]Interval{}, BreachDelta: map[string]map[string]Interval{}, TATP90Delta: map[string]map[string]Interval{},
		WorstBreach: map[string]map[string]BreachExtremes{}, RemoteCost: map[string]Interval{}, UnbilledDelta: map[string]Interval{}}
	bad := func(format string, args ...any) { c.Integrity = append(c.Integrity, fmt.Sprintf(format, args...)) }
	c.Integrity = append(c.Integrity, global...)
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
		if provenanceApplies(p) {
			// Order provenance (Slice 5C.4l): every order of either record
			// in the other by origin, with the same immutable request.
			for _, f := range orderProvenance(a, b) {
				bad("seed %d: order provenance: %s", a.Seed, f)
			}
		}
	}
	for _, pr := range []*PlanReport{cand, ag} {
		if !pr.Valid {
			bad("plan %s's records do not check out", pr.Plan)
		}
		for _, f := range pr.Integrity {
			bad("plan %s: %s", pr.Plan, f)
		}
	}
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
		ext := map[string]BreachExtremes{}
		var worst WorstSeed
		complete := true
		for _, x := range pairs {
			wa, wb := x.a.Measures.Windows[w], x.b.Measures.Windows[w]
			if wa == nil || wb == nil {
				complete = false
				break
			}
			d := float64(wa.Money.OperatingProfitCents - wb.Money.OperatingProfitCents)
			prof = append(prof, d)
			worst.note(d, x.a.Seed, false)
			rev = append(rev, float64(wa.Money.NetRevenueCents-wb.Money.NetRevenueCents))
			reps = append(reps, float64(wa.Remote.LocalReports+wa.Remote.RemoteReports-wb.Remote.LocalReports-wb.Remote.RemoteReports))
			cash = append(cash, float64(wa.Money.End.CashCents-wb.Money.End.CashCents))
			cost = append(cost, float64(wa.Remote.TotalCostCents-wb.Remote.TotalCostCents))
			unb = append(unb, float64(wa.Money.End.UnbilledCompletedCents-wb.Money.End.UnbilledCompletedCents))
			for _, q := range Priorities {
				ca, cb := wa.Cohorts[q], wb.Cohorts[q]
				e := ext[q]
				if ca.BreachRate.Defined {
					e.Candidate.note(ca.BreachRate.Value, x.a.Seed, true)
				}
				if cb.BreachRate.Defined {
					e.Against.note(cb.BreachRate.Value, x.a.Seed, true)
				}
				if ca.BreachRate.Defined && cb.BreachRate.Defined {
					dq[q] = append(dq[q], ca.BreachRate.Value-cb.BreachRate.Value)
					e.LargestIncrease.note(ca.BreachRate.Value-cb.BreachRate.Value, x.a.Seed, true)
				}
				ext[q] = e
				if ca.CompletedTATMin.Defined && cb.CompletedTATMin.Defined {
					tq[q] = append(tq[q], float64(ca.CompletedTATMin.P90-cb.CompletedTATMin.P90))
				}
			}
		}
		if !complete || len(pairs) == 0 {
			continue
		}
		c.Profit[w], c.Revenue[w], c.Reports[w], c.CashAtEnd[w] = pairedT(prof), pairedT(rev), pairedT(reps), pairedT(cash)
		c.WorstProfit[w], c.RemoteCost[w], c.UnbilledDelta[w] = worst, pairedT(cost), pairedT(unb)
		c.BreachDelta[w], c.TATP90Delta[w], c.WorstBreach[w] = map[string]Interval{}, map[string]Interval{}, ext
		for _, q := range Priorities {
			// A paired interval only over every seed: a seed whose rate is
			// undefined leaves the interval out rather than shrinking n.
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
		for _, k := range []struct {
			name string
			val  func(Sample) int64
			out  *[]Payback
		}{{"accrual", func(s Sample) int64 { return s.ProfitCents }, &c.AccrualPayback}, {"cash", func(s Sample) int64 { return s.CashCents }, &c.CashPayback}} {
			pb, missing := payback(p, h, cr, ri, k.val)
			*k.out = append(*k.out, pb)
			if len(missing) > 0 {
				bad("%s payback to day %d: seeds %v lack daily samples", k.name, h, missing)
			}
		}
	}
	var obl []float64
	for _, x := range pairs {
		if wm := x.a.Measures.Windows[WinT90]; wm != nil {
			obl = append(obl, float64(wm.Money.End.SeverancePayableCents+wm.Money.End.ContractMinimumCents))
		}
	}
	c.Financial = Financial{T90Absolute: cand.OperatingProfit[WinT90], T90Increment: c.Profit[WinT90],
		OperatingAbsolute: cand.OperatingProfit[WinOps], OperatingIncrement: c.Profit[WinOps],
		T180Absolute: cand.OperatingProfit[WinT180], T180Increment: c.Profit[WinT180], TransitionObligationsT90: pairedT(obl)}
	c.CaseSelection, c.CaseSelectionStatus = caseSelection(p, cand, ag)
	if p.Clinical != nil && p.Clinical.LostDemand != nil {
		c.LostDemand = lostDemandGates(p, cr, ar)
		c.Access = accessStatements(cand, c.LostDemand)
		if p.Clinical.LostDemand.Rule == RuleEmpirical {
			c.AccessClaims = accessClaims(c.LostDemand)
			c.ClinicalJoint = seedSafeguardJoint(p, cand, c.LostDemand)
		}
	}
	slices.Sort(c.Integrity)
	c.Integrity = slices.Compact(c.Integrity)
	c.Valid = len(c.Integrity) == 0
	return c
}

// payback finds each seed's sustained break-even day within horizon h
// (days 1..h of the daily samples): a seed still below zero at h is
// censored. The median is the smallest day by which at least half the
// seeds have repaid, and is not reported when fewer than half have. A
// seed without samples through h is not left out silently: it is
// returned as missing, which invalidates the comparison.
func payback(p *Policy, h int, cr []Record, ri map[int64]*Record, val func(Sample) int64) (Payback, []int64) {
	pb := Payback{Horizon: h, Days: []int{}}
	var missing []int64
	recs := slices.Clone(cr)
	sort.Slice(recs, func(i, j int) bool { return recs[i].Seed < recs[j].Seed })
	for i := range recs {
		a, b := &recs[i], ri[recs[i].Seed]
		if b == nil {
			continue
		}
		pb.Seeds++
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
			missing = append(missing, a.Seed)
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
	if n > 0 && len(missing) == 0 && float64(len(repaid)) >= p.Payback.MedianMinRepaidFrac*float64(n) {
		k := (n + 1) / 2 // the smallest day by which ⌈n/2⌉ seeds have repaid
		pb.Median, pb.MedianDefined = repaid[k-1], true
	}
	return pb, missing
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

// tally is one stratum's paired counts: candidate (a) and baseline (b).
type tally struct{ dueA, srvA, dueB, srvB int64 }

// caseSelection guards against passing the service gates by serving
// fewer or easier cases (README §8), against the paired plan and decided
// exactly: in every gated window, the due-demand served fraction per seed
// (each priority and modality) and pooled (each priority, modality and
// their intersection), the acquisitions pooled (each priority and
// modality), the loss and abandonment rates pooled; and the inherited
// orders' served fraction, per seed and pooled. A stratum with fewer due
// (or acquired, or received) than minStratumDue on the baseline is
// insufficient: never a pass; it fails if the candidate drops it
// entirely; its lack of evidence blocks eligibility when its kind is in
// blockingStrata.
func caseSelection(p *Policy, cand, ag *PlanReport) ([]GateSummary, string) {
	cs := p.CaseSelection
	blocking := func(key string) bool { return slices.Contains(cs.BlockingStrata, kindOf(key)) }
	bySeed := map[int64]*SeedResult{}
	for i := range ag.PerSeed {
		bySeed[ag.PerSeed[i].Seed] = &ag.PerSeed[i]
	}
	var out []GateSummary
	status := Pass
	add := func(g GateSummary) {
		out = append(out, g)
		switch {
		case g.Status == Fail:
			status = Fail
		case g.Status == Insufficient && g.Blocking && status == Pass:
			status = Insufficient
		}
	}
	type seedTally struct {
		seed int64
		t    map[string]*tally
	}
	judge := func(window string, seeds []seedTally, pooled map[string]*tally) {
		n := len(seeds)
		lim := failLimit(bp(p.Service.MaxFailingSeedFraction), n)
		keys := make([]string, 0, len(pooled))
		for k := range pooled {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if kd := kindOf(k); kd == KindPriority || kd == KindModality || kd == KindInherited {
				g := GateSummary{Gate: "served_fraction_drop_per_seed", Window: window, Stratum: k, Role: "gated", Threshold: cs.PerSeedMaxServedDrop,
					Seeds: n, FailLimit: lim, Blocking: blocking(k)}
				for _, s := range seeds {
					t := s.t[k]
					if t == nil || t.dueB < cs.MinStratumDue {
						g.InsufficientSeeds++ // too few due on this baseline seed to judge a drop: pooled only
						continue
					}
					g.DefinedSeeds++
					g.Worst.note(ratio(t.srvB, t.dueB).Value-ratio(t.srvA, t.dueA).Value, s.seed, true)
					if t.dueA == 0 || dropAbove(t.srvB, t.dueB, t.srvA, t.dueA, bp(cs.PerSeedMaxServedDrop)) {
						g.Failing++
					} else {
						g.Passing++
					}
				}
				switch {
				case g.Failing > lim:
					g.Status = Fail
				case pooled[k].dueA == 0 && pooled[k].dueB == 0:
					g.Status = Empty
				case g.DefinedSeeds == 0:
					g.Status = Insufficient
				default:
					g.Status = Pass
				}
				add(g)
			}
			t := pooled[k]
			g := GateSummary{Gate: "served_fraction_drop_pooled", Window: window, Stratum: k, Role: "gated", Threshold: cs.PooledMaxServedDrop, Seeds: n, Blocking: blocking(k)}
			g.Pooled = ratio(t.srvA, t.dueA)
			if t.dueB > 0 && t.dueA > 0 {
				g.Worst = WorstSeed{Value: ratio(t.srvB, t.dueB).Value - ratio(t.srvA, t.dueA).Value, Defined: true}
			}
			switch {
			case t.dueA == 0 && t.dueB == 0:
				g.Status = Empty
			case t.srvB > 0 && t.srvA == 0:
				g.Status = Fail // a stratum served before, no longer served at all
			case t.dueB < cs.MinStratumDue:
				g.Status = Insufficient // too rare to judge a drop: reported, never a pass
			case t.dueA == 0 || dropAbove(t.srvB, t.dueB, t.srvA, t.dueA, bp(cs.PooledMaxServedDrop)):
				g.Status = Fail
			default:
				g.Status = Pass
			}
			add(g)
		}
	}
	for _, w := range p.Service.Gated {
		pooled, acq := map[string]*tally{}, map[string]*tally{}
		var seeds []seedTally
		var lostA, refA, lostB, refB, leftA, wiA, leftB, wiB int64
		missing := false
		for i := range cand.PerSeed {
			a := &cand.PerSeed[i]
			b := bySeed[a.Seed]
			if b == nil {
				continue
			}
			wa, wb := a.Measures.Windows[w], b.Measures.Windows[w]
			if wa == nil || wb == nil {
				missing = true
				break
			}
			xa, xb := wa.Access, wb.Access
			st := seedTally{seed: a.Seed, t: map[string]*tally{}}
			keys := map[string]bool{}
			for k := range xa.DueBy {
				keys[k] = true
			}
			for k := range xb.DueBy {
				keys[k] = true
			}
			for k := range keys {
				t := &tally{dueA: xa.DueBy[k], srvA: xa.ServedBy[k], dueB: xb.DueBy[k], srvB: xb.ServedBy[k]}
				st.t[k] = t
				pt := pooled[k]
				if pt == nil {
					pt = &tally{}
					pooled[k] = pt
				}
				pt.dueA, pt.srvA, pt.dueB, pt.srvB = pt.dueA+t.dueA, pt.srvA+t.srvA, pt.dueB+t.dueB, pt.srvB+t.srvB
			}
			seeds = append(seeds, st)
			for k, v := range wa.AcquiredBy {
				if kd := kindOf(k); kd == KindPriority || kd == KindModality {
					if acq[k] == nil {
						acq[k] = &tally{}
					}
					acq[k].dueA += v
				}
			}
			for k, v := range wb.AcquiredBy {
				if kd := kindOf(k); kd == KindPriority || kd == KindModality {
					if acq[k] == nil {
						acq[k] = &tally{}
					}
					acq[k].dueB += v
				}
			}
			lostA, refA, lostB, refB = lostA+xa.Lost, refA+xa.Referrals, lostB+xb.Lost, refB+xb.Referrals
			leftA, wiA, leftB, wiB = leftA+xa.WalkInsLeft, wiA+xa.WalkIns, leftB+xb.WalkInsLeft, wiB+xb.WalkIns
		}
		if missing {
			add(GateSummary{Gate: "case_selection", Window: w, Role: "gated", Status: Insufficient, Blocking: true})
			continue
		}
		judge(w, seeds, pooled)
		keys := make([]string, 0, len(acq))
		for k := range acq {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			t := acq[k]
			g := GateSummary{Gate: "acquired_drop_pooled", Window: w, Stratum: k, Role: "gated", Threshold: cs.MaxAcquiredDrop, Seeds: len(seeds), Blocking: blocking(k)}
			g.Pooled = ratio(t.dueA, t.dueB)
			if g.Pooled.Defined {
				g.Worst = WorstSeed{Value: 1 - g.Pooled.Value, Defined: true}
			}
			switch {
			case t.dueB > 0 && t.dueA == 0:
				g.Status = Fail
			case t.dueB < cs.MinStratumDue:
				g.Status = Insufficient
			case relDropAbove(t.dueA, t.dueB, bp(cs.MaxAcquiredDrop)):
				g.Status = Fail
			default:
				g.Status = Pass
			}
			add(g)
		}
		rise := func(gate string, na, da, nb, db int64) {
			g := GateSummary{Gate: gate, Window: w, Role: "gated", Threshold: cs.MaxLossRateIncrease, Seeds: len(seeds), Blocking: false}
			g.Pooled = ratio(na, da)
			if da > 0 && db > 0 {
				g.Worst = WorstSeed{Value: ratio(na, da).Value - ratio(nb, db).Value, Defined: true}
			}
			switch {
			case da == 0 && db == 0:
				g.Status = Empty
			case db < cs.MinStratumDue || da == 0:
				g.Status = Insufficient // too few received to judge a rate; reported
			case dropAbove(na, da, nb, db, bp(cs.MaxLossRateIncrease)):
				g.Status = Fail
			default:
				g.Status = Pass
			}
			add(g)
		}
		rise("lost_referral_rate_increase", lostA, refA, lostB, refB)
		rise("walkin_abandon_rate_increase", leftA, wiA, leftB, wiB)
	}
	// The inherited orders, followed to the end of observation.
	var seeds []seedTally
	pooled := map[string]*tally{InheritedStratum: {}}
	for i := range cand.PerSeed {
		a := &cand.PerSeed[i]
		b := bySeed[a.Seed]
		if b == nil {
			continue
		}
		ia, ib := a.Measures.Inherited, b.Measures.Inherited
		t := &tally{dueA: ia.Due, srvA: ia.Served, dueB: ib.Due, srvB: ib.Served}
		seeds = append(seeds, seedTally{seed: a.Seed, t: map[string]*tally{InheritedStratum: t}})
		pt := pooled[InheritedStratum]
		pt.dueA, pt.srvA, pt.dueB, pt.srvB = pt.dueA+t.dueA, pt.srvA+t.srvA, pt.dueB+t.dueB, pt.srvB+t.srvB
	}
	judge("inherited", seeds, pooled)
	return out, status
}

// verdict applies the frozen decision rules (policy "verdict"): integrity,
// then the hard gates, then the financial classification. Profit never
// compensates for a failed gate.
func verdict(p *Policy, c *Comparison, cand, base *PlanReport) {
	c.Classification = ClassNotJudged
	// A fact about the pair, whatever the verdict: the baseline fails the
	// service gates the candidate passes.
	c.ServiceRestored = base.Valid && base.ServiceStatus != Pass && cand.ServiceStatus == Pass
	v5 := p.Clinical != nil
	if v5 {
		// v0.2-eval-5 (audit M3): only a demonstrated failure is restored,
		// never a baseline that merely lacks evidence; and the candidate
		// must pass the whole clinical rule.
		c.ServiceRestored = base.Valid && base.ServiceStatus == Fail && cand.ClinicalStatus == Pass &&
			cand.ClinicalSafety != nil && cand.ClinicalSafety.Status == Pass
	}
	if !c.Valid {
		c.Verdict, c.Reasons = INVALID, append([]string{}, c.Integrity...)
		return
	}
	var why []string
	if v5 {
		why = clinicalReasons(cand, cmp.Or(c.ClinicalJoint, cand.ClinicalJoint))
		// Lost demand is its own gate: a timeliness gain never offsets it.
		for _, g := range c.LostDemand {
			if g.Status != Pass {
				why = append(why, fmt.Sprintf("lost demand %s: %s (%s)", g.Priority, g.Status, g.Reason))
			}
		}
	}
	if j := cand.ServiceJoint; !v5 && j.Status != Pass {
		why = append(why, fmt.Sprintf("service seed reliability: %s (%d failing %v and %d insufficient %v of %d seeds, limit %d)",
			j.Status, len(j.FailingSeeds), j.FailingSeeds, len(j.InsufficientSeeds), j.InsufficientSeeds, j.Seeds, j.FailLimit))
	}
	for _, g := range cand.ServiceGates {
		if !v5 && g.Role == "gated" && (g.Status == Fail || g.Status == Insufficient) {
			why = append(why, fmt.Sprintf("service %s %s %s pooled: %s (%d/%d)", g.Gate, g.Window, g.Stratum, g.Status, g.Pooled.Num, g.Pooled.Den))
		}
	}
	var notJudged []string
	for _, g := range c.CaseSelection {
		switch {
		case g.Status == Fail || (g.Status == Insufficient && g.Blocking):
			why = append(why, fmt.Sprintf("case selection %s %s %s: %s (%.4f, threshold %.4f)", g.Gate, g.Window, g.Stratum, g.Status, g.Worst.Value, g.Threshold))
		case g.Status == Insufficient:
			notJudged = append(notJudged, g.Window+" "+g.Stratum)
		}
	}
	if !v5 && cand.BacklogStatus != Pass {
		why = append(why, fmt.Sprintf("backlog: %d of %d seeds stable", cand.BacklogStable, len(cand.PerSeed)))
	}
	if cand.SolvencyStatus != Pass {
		why = append(why, fmt.Sprintf("solvency: cash below the floor on seeds %v", cand.InsolventSeeds))
	}
	if cand.FundingStatus != Pass {
		why = append(why, fmt.Sprintf("funding: committed liabilities exceed cash at the end of observation on seeds %v", cand.FundingGapSeeds))
	}
	if len(why) > 0 {
		c.Verdict, c.Reasons = NOGO, why
		return
	}
	if len(notJudged) > 0 {
		why = append(why, fmt.Sprintf("rare strata reported, not judged (insufficient evidence; none dropped): %s", strings.Join(notJudged, ", ")))
	}
	t90, t180 := c.Profit[WinT90], c.Profit[WinT180]
	a90, a180 := cand.OperatingProfit[WinT90], cand.OperatingProfit[WinT180]
	benefit := t90.Defined && t90.Lo > 0
	validated := cand.ValidationPass && t180.N > 0 && t180.Mean > 0
	profitable := a90.N > 0 && a90.Mean > 0 && a180.N > 0 && a180.Mean > 0
	restored := base.Valid && base.ServiceStatus != Pass
	if v5 {
		restored = base.Valid && base.ServiceStatus == Fail
	}
	ops := c.Profit[WinOps]
	transition := ops.N > 0 && ops.Mean > 0
	switch {
	case benefit && validated && profitable:
		c.Verdict, c.Classification = GO, ClassFinancial
	case t90.Mean > 0:
		c.Verdict, c.Classification = CONDITIONAL, ClassRelative
		if !benefit {
			why = append(why, fmt.Sprintf("T90 profit over %s not significant: %.0f [%.0f, %.0f] cents", c.Against, t90.Mean, t90.Lo, t90.Hi))
		}
		if !validated {
			why = append(why, "T180 validation not met (service or incremental profit)")
		}
		if !profitable {
			why = append(why, fmt.Sprintf("not profitable itself: mean operating profit %.0f cents at T90, %.0f at T180", a90.Mean, a180.Mean))
		}
	case transition:
		// Not repaid by T90, ahead after the transition: an unrepaid
		// investment, never financial GO whatever T180 shows.
		c.Verdict, c.Classification = CONDITIONAL, ClassTransition
		why = append(why, fmt.Sprintf("transition not yet repaid: T90 increment %.0f cents, days %d–%d increment %.0f [%.0f, %.0f] cents; T180 increment %.0f cents (reported, not promoted)",
			t90.Mean, p.Horizons.OperatingFrom+1, p.Horizons.PrimaryDays, ops.Mean, ops.Lo, ops.Hi, t180.Mean))
		if restored {
			why = append(why, fmt.Sprintf(reasonServiceRestored, c.Against, base.ServiceStatus, -t90.Mean, -t90.Hi, -t90.Lo))
		}
		if !validated {
			why = append(why, "T180 validation not met (service or incremental profit)")
		}
	case restored:
		c.Verdict, c.Classification = CONDITIONAL, ClassServicePurchase
		why = append(why, fmt.Sprintf(reasonServiceRestored, c.Against, base.ServiceStatus, -t90.Mean, -t90.Hi, -t90.Lo))
		if !validated {
			why = append(why, "T180 validation not met (service or incremental profit)")
		}
	case v5 && base.Valid && base.ServiceStatus == Insufficient:
		c.Verdict, c.Classification = NOGO, ClassNoBenefit
		why = append(why, fmt.Sprintf("no T90 profit over %s (%.0f cents), whose service is not shown to fail (insufficient evidence): no restoration is claimed", c.Against, t90.Mean))
	default:
		c.Verdict, c.Classification = NOGO, ClassNoBenefit
		why = append(why, fmt.Sprintf("no T90 profit over %s (%.0f cents), whose service already passes", c.Against, t90.Mean))
	}
	c.Reasons = why
	if v5 && p.Clinical.LostDemand != nil && (c.Verdict == GO || c.Verdict == CONDITIONAL) {
		c.AccessLimitation = AccessLimitation
		lead := []string{AccessLimitation}
		if c.AccessClaims != nil {
			lead = append(lead, EmpiricalScope)
		}
		c.Reasons = append(lead, c.Reasons...)
	}
}
