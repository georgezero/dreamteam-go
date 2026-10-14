package scenario

import (
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

var (
	priorities = []string{"routine", "soon", "urgent"}
	windows    = []string{"any", "morning", "afternoon", "evening"} // sim.Window order
	booking    = []string{"first_fit"}
)

// maxBlockMilli caps one source's hourly rate so a Poisson draw fits in one
// hash (at most four parts of λ ≤ 16).
const maxBlockMilli = 64_000

// quantile parses a {"p0": v, …, "p100": v} table.
func (c *checker) quantile(p string, q Quantile, minV int64) rng.Quantile {
	type kv struct{ pct, v int64 }
	var pts []kv
	for k, v := range q {
		n, err := strconv.ParseInt(strings.TrimPrefix(k, "p"), 10, 64)
		if !strings.HasPrefix(k, "p") || err != nil || n < 0 || n > 100 {
			c.fail(p+"."+k, "percentile keys are p0 … p100")
			continue
		}
		if v < minV {
			c.fail(p+"."+k, "%d is below %d", v, minV)
		}
		pts = append(pts, kv{n, v})
	}
	sort.Slice(pts, func(i, j int) bool { return pts[i].pct < pts[j].pct })
	var qp []rng.QuantilePoint
	for _, x := range pts {
		qp = append(qp, rng.QuantilePoint{P: rng.PPM(x.pct * 10_000), V: x.v})
	}
	out, err := rng.NewQuantile(qp)
	if err != nil && len(pts) > 0 {
		c.fail(p, "needs p0 and p100 with non-decreasing values (%v)", err)
	} else if len(pts) == 0 {
		c.fail(p, "required")
	}
	return out
}

// mixBP parses a {name: bp} map over a fixed name list that must sum to
// 10,000; the result is in list order.
func (c *checker) mixBP(p string, m map[string]int64, names []string) []uint64 {
	out := make([]uint64, len(names))
	var sum int64
	for k, v := range m {
		i := slices.Index(names, k)
		if i < 0 {
			c.fail(p+"."+k, "not one of %s", strings.Join(names, ", "))
			continue
		}
		if v < 0 {
			c.fail(p+"."+k, "negative")
			continue
		}
		out[i] = uint64(v)
		sum += v
	}
	if sum != 10_000 {
		c.fail(p, "sums to %d, not 10000", sum)
	}
	return out
}

// demand validates and compiles the catalog, templates, referral sources,
// demand, behavior, booking policy and opening schedule.
func (c *checker) demand(f *File, cfg *sim.Config) {
	pol := f.Policies
	c.oneOf("policies.booking", pol.Booking, booking)
	if pol.BookingHorizonDays < 1 || pol.BookingHorizonDays > 366 {
		c.fail("policies.bookingHorizonDays", "%d is outside 1–366", pol.BookingHorizonDays)
	}
	if pol.BookingNoticeMin < 0 || pol.BookingNoticeMin > 1440 {
		c.fail("policies.bookingNoticeMin", "%d is outside 0–1440", pol.BookingNoticeMin)
	}
	cfg.Policies.Booking = pol.Booking
	cfg.Policies.BookingHorizonDays = pol.BookingHorizonDays
	cfg.Policies.BookingNotice = simtime.Duration(pol.BookingNoticeMin) * simtime.Minute
	if pol.EarlyStartMin < 0 || pol.EarlyStartMin > 120 {
		c.fail("policies.earlyStartMin", "%d is outside 0–120", pol.EarlyStartMin)
	}
	if pol.BreakDeferMaxMin < 0 || pol.BreakDeferMaxMin > 240 {
		c.fail("policies.breakDeferMaxMin", "%d is outside 0–240", pol.BreakDeferMaxMin)
	}
	if pol.WeeklyOvertimeHours < 0 || pol.WeeklyOvertimeHours > 80 {
		c.fail("policies.weeklyOvertimeHours", "%d is outside 0–80", pol.WeeklyOvertimeHours)
	}
	if pol.OvertimeFactorBP < 10_000 || pol.OvertimeFactorBP > 30_000 {
		c.fail("policies.overtimeFactorBP", "%d is outside 10000–30000", pol.OvertimeFactorBP)
	}
	cfg.Policies.EarlyStart = simtime.Duration(pol.EarlyStartMin) * simtime.Minute
	cfg.Policies.StartEarly = pol.StartEarly
	cfg.Policies.BreakDeferMax = simtime.Duration(pol.BreakDeferMaxMin) * simtime.Minute
	cfg.Policies.WeeklyHours = simtime.Duration(pol.WeeklyOvertimeHours) * simtime.Hour
	cfg.Policies.OvertimeBP = pol.OvertimeFactorBP
	for i, name := range priorities {
		v, ok := pol.TATTargetMin[name]
		if !ok || v < 1 || v > 7*24*60 {
			c.fail("policies.tatTargetMin."+name, "required, 1–10080 minutes")
		}
		cfg.Policies.TATTarget[i] = simtime.Duration(v) * simtime.Minute
	}
	for k := range pol.TATTargetMin {
		if !slices.Contains(priorities, k) {
			c.fail("policies.tatTargetMin."+k, "not a priority")
		}
	}

	// Catalog and slot categories.
	if len(f.Catalog) == 0 {
		c.fail("catalog", "at least one exam is required")
	}
	exams := map[string]int{}
	cats := map[string]int{}
	for i, x := range f.Catalog {
		xp := fmt.Sprintf("catalog[%d]", i)
		if x.Code == "" {
			c.fail(xp+".code", "required")
		} else if _, dup := exams[x.Code]; dup {
			c.fail(xp+".code", "duplicate %q", x.Code)
		}
		exams[x.Code] = i
		if x.Name == "" {
			c.fail(xp+".name", "required")
		}
		c.oneOf(xp+".modality", x.Modality, modalities)
		if x.Category == "" {
			c.fail(xp+".category", "required")
		}
		ci, ok := cats[x.Category]
		if !ok {
			ci = len(cfg.Categories)
			cats[x.Category] = ci
			cfg.Categories = append(cfg.Categories, sim.CategoryConfig{Name: x.Category, Modality: x.Modality, Contrast: x.Contrast})
		} else if cc := cfg.Categories[ci]; cc.Modality != x.Modality || cc.Contrast != x.Contrast {
			c.fail(xp+".category", "%q mixes modalities or contrast with an earlier exam", x.Category)
		}
		cfg.Catalog = append(cfg.Catalog, sim.ExamConfig{Code: x.Code, Name: x.Name, Modality: x.Modality,
			Category: ci, Contrast: x.Contrast, WalkInEligible: x.WalkInEligible,
			RoomTime:      c.quantile(xp+".roomTimeSec", x.RoomTimeSec, 60),
			TechExtra:     c.quantile(xp+".techExtraSec", x.TechExtraSec, 1),
			ReadTime:      c.quantile(xp+".readSec", x.ReadSec, 30),
			BaseRateCents: x.BaseRateCents, SupplyCostCents: x.SupplyCostCents})
	}

	// Templates.
	cfg.Templates = make([][7][]sim.SlotTemplate, len(cfg.Rooms))
	roomIdx := map[string]int{}
	for i, r := range cfg.Rooms {
		roomIdx[r.ID] = i
	}
	roomIDs := make([]string, 0, len(f.Templates))
	for id := range f.Templates {
		roomIDs = append(roomIDs, id)
	}
	sort.Strings(roomIDs)
	for _, id := range roomIDs {
		tp := "templates." + id
		ri, ok := roomIdx[id]
		if !ok {
			c.fail(tp, "%q is not a room", id)
			continue
		}
		room := cfg.Rooms[ri]
		var assigned [7]bool
		for gi, g := range f.Templates[id] {
			gp := fmt.Sprintf("%s[%d]", tp, gi)
			var slots []sim.SlotTemplate
			for si, sp := range g.Slots {
				spp := fmt.Sprintf("%s.slots[%d]", gp, si)
				start := c.clock(spp+".start", sp.Start)
				n := sp.Count
				if n == 0 {
					n = 1
				}
				if sp.LenMin < 5 || sp.LenMin > 240 || n < 1 || n > 200 {
					c.fail(spp, "lenMin must be 5–240 and count 1–200")
					continue
				}
				ci, ok := cats[sp.Category]
				if !ok {
					c.fail(spp+".category", "%q is not a category in the catalog", sp.Category)
					continue
				}
				if cfg.Categories[ci].Modality != room.Modality {
					c.fail(spp+".category", "%q is %s, room %s is %s", sp.Category, cfg.Categories[ci].Modality, id, room.Modality)
				}
				l := simtime.Duration(sp.LenMin) * simtime.Minute
				for k := int64(0); k < n; k++ {
					from := start + simtime.Duration(k)*l
					slots = append(slots, sim.SlotTemplate{Span: sim.Span{From: from, To: from + l}, Category: ci})
				}
			}
			slices.SortStableFunc(slots, func(a, b sim.SlotTemplate) int { return int(a.From - b.From) })
			for k := 1; k < len(slots); k++ {
				if slots[k].From < slots[k-1].To {
					c.fail(gp+".slots", "slots overlap at %s", simtime.FormatClock(slots[k].From))
					break
				}
			}
			if len(g.Days) == 0 {
				c.fail(gp+".days", "required")
			}
			for di, ds := range g.Days {
				wd, err := simtime.ParseWeekday(ds)
				dp := fmt.Sprintf("%s.days[%d]", gp, di)
				if err != nil || ds != wd.String() {
					c.fail(dp, "not a weekday (use Mon, Tue, …)")
					continue
				}
				if assigned[wd] {
					c.fail(dp, "%s already has a template for room %s", ds, id)
					continue
				}
				assigned[wd] = true
				h := cfg.Hours[wd]
				for _, s := range slots {
					if h == nil || s.From < h.From || s.To > h.To {
						c.fail(gp+".slots", "slot %s–%s on %s is outside open hours", simtime.FormatClock(s.From), simtime.FormatClock(s.To), ds)
						break
					}
				}
				cfg.Templates[ri][wd] = slots
			}
		}
	}

	// Referral sources.
	if len(f.Sources) == 0 {
		c.fail("referralSources", "at least one source is required")
	}
	srcIdx := map[string]int{}
	for i, s := range f.Sources {
		sp := fmt.Sprintf("referralSources[%d]", i)
		if s.ID == "" {
			c.fail(sp+".id", "required")
		} else if _, dup := srcIdx[s.ID]; dup {
			c.fail(sp+".id", "duplicate %q", s.ID)
		}
		srcIdx[s.ID] = i
		if s.Name == "" {
			c.fail(sp+".name", "required")
		}
		if s.BaseDailyMilliOrders < 0 || s.BaseDailyMilliOrders > 1_000_000 {
			c.fail(sp+".baseDailyMilliOrders", "%d is outside 0–1000000", s.BaseDailyMilliOrders)
		}
		sc := sim.SourceConfig{ID: s.ID, Name: s.Name, Kind: s.Kind, BaseDailyMilli: s.BaseDailyMilliOrders}
		copy(sc.PriorityBP[:], c.mixBP(sp+".priorityMixBP", s.PriorityMixBP, priorities))
		// Exam mix in catalog order, so map order never matters.
		for code := range s.ExamMix {
			if _, ok := exams[code]; !ok {
				c.fail(sp+".examMix."+code, "not an exam code in the catalog")
			}
		}
		for xi, x := range f.Catalog {
			if w, ok := s.ExamMix[x.Code]; ok {
				if w <= 0 {
					c.fail(sp+".examMix."+x.Code, "weight must be positive")
					continue
				}
				sc.Exams = append(sc.Exams, xi)
				sc.ExamWeights = append(sc.ExamWeights, uint64(w))
			}
		}
		if len(sc.Exams) == 0 {
			c.fail(sp+".examMix", "must not be empty")
		}
		cfg.Sources = append(cfg.Sources, sc)
	}

	// Demand shape.
	d := f.Demand
	dc := &cfg.Demand
	var hourSum int64
	for k, v := range d.HourProfileBP {
		h, err := strconv.Atoi(k)
		if err != nil || k != fmt.Sprintf("%02d", h) || h < 0 || h > 23 || v < 0 {
			c.fail("demand.hourProfileBP."+k, "keys are hours \"00\" … \"23\" with non-negative shares")
			continue
		}
		dc.HourBP[h] = v
		hourSum += v
	}
	if hourSum != 10_000 {
		c.fail("demand.hourProfileBP", "sums to %d, not 10000", hourSum)
	}
	if len(d.WeekdayFactorBP) != 7 {
		c.fail("demand.weekdayFactorBP", "needs all seven weekdays")
	}
	for _, wd := range weekdays(c, "demand.weekdayFactorBP", d.WeekdayFactorBP) {
		v := d.WeekdayFactorBP[wd.String()]
		if v < 0 || v > 30_000 {
			c.fail("demand.weekdayFactorBP."+wd.String(), "%d is outside 0–30000", v)
		}
		dc.WeekdayBP[wd] = v
	}
	if len(d.SeasonFactorBP) != 12 {
		c.fail("demand.seasonFactorBP", "needs 12 months")
	}
	for i, v := range d.SeasonFactorBP {
		if i < 12 {
			if v < 0 || v > 30_000 {
				c.fail(fmt.Sprintf("demand.seasonFactorBP[%d]", i), "%d is outside 0–30000", v)
			}
			dc.SeasonBP[i] = v
		}
	}
	dc.LeadDays = c.quantile("demand.leadTimeDays", d.LeadTimeDays, 0)
	copy(dc.WindowBP[:], c.mixBP("demand.windowMixBP", d.WindowMixBP, windows))
	for i, s := range cfg.Sources {
		for h := range dc.HourBP {
			for wd := range dc.WeekdayBP {
				for _, sf := range dc.SeasonBP {
					if s.BaseDailyMilli*dc.HourBP[h]/10_000*dc.WeekdayBP[wd]/10_000*sf/10_000 > maxBlockMilli {
						c.fail(fmt.Sprintf("referralSources[%d]", i), "hourly rate above %d milli-orders", maxBlockMilli)
						return
					}
				}
			}
		}
	}

	// Behavior.
	b := f.Behavior
	for i, name := range priorities {
		q, ok := b.WaitToleranceDays[name]
		if !ok {
			c.fail("behavior.waitToleranceDays."+name, "required")
			continue
		}
		cfg.Behavior.ToleranceDays[i] = c.quantile("behavior.waitToleranceDays."+name, q, 0)
	}
	for k := range b.WaitToleranceDays {
		if !slices.Contains(priorities, k) {
			c.fail("behavior.waitToleranceDays."+k, "not a priority")
		}
	}
	cn := b.Cancellation
	if cn.PPM < 0 || cn.PPM > 1_000_000 || cn.RebookPPM < 0 || cn.RebookPPM > 1_000_000 {
		c.fail("behavior.cancellation", "ppm and rebookPPM must be 0–1000000")
	}
	cfg.Behavior.CancelPPM = rng.PPM(cn.PPM)
	cfg.Behavior.RebookPPM = rng.PPM(cn.RebookPPM)
	cfg.Behavior.CancelLeadHours = c.quantile("behavior.cancellation.leadHours", cn.LeadHours, 0)

	c.arrivals(f, cfg)

	// Opening schedule: references only here; slot checks need the plan.
	for i, op := range f.OpeningSchedule {
		op_ := fmt.Sprintf("openingSchedule[%d]", i)
		oa := sim.OpeningAppt{Day: op.Day, Start: c.clock(op_+".time", op.Time)}
		var ok bool
		if oa.Room, ok = roomIdx[op.Room]; !ok {
			c.fail(op_+".room", "%q is not a room", op.Room)
		}
		if oa.Exam, ok = exams[op.Exam]; !ok {
			c.fail(op_+".exam", "%q is not an exam code", op.Exam)
		}
		if oa.Source, ok = srcIdx[op.Source]; !ok {
			c.fail(op_+".source", "%q is not a referral source", op.Source)
		}
		pr := op.Priority
		if pr == "" {
			pr = "routine"
		}
		c.oneOf(op_+".priority", pr, priorities)
		oa.Priority = sim.Priority(max(0, slices.Index(priorities, pr)))
		if op.Day < 0 || op.Day > 6 {
			c.fail(op_+".day", "must be 0–6")
		}
		cfg.Opening = append(cfg.Opening, oa)
	}
}

// openingSlots checks that each opening appointment sits on its own
// bookable template slot of the right category. It runs on a compiled
// config, after every other check passed.
func (c *checker) openingSlots(cfg *sim.Config) {
	type key struct {
		day   int64
		room  int
		start simtime.Duration
	}
	used := map[key]bool{}
	covs := map[int64]*sim.Coverage{}
	for i, op := range cfg.Opening {
		p := fmt.Sprintf("openingSchedule[%d]", i)
		date := cfg.Start.AddDays(op.Day)
		var tpl *sim.SlotTemplate
		for k, s := range cfg.Templates[op.Room][date.Weekday()] {
			if s.From == op.Start {
				tpl = &cfg.Templates[op.Room][date.Weekday()][k]
			}
		}
		x := cfg.Catalog[op.Exam]
		switch {
		case tpl == nil:
			c.fail(p, "no %s template slot at %s on %s", cfg.Rooms[op.Room].ID, simtime.FormatClock(op.Start), date)
			continue
		case tpl.Category != x.Category:
			c.fail(p, "slot is %s, exam %s needs %s", cfg.Categories[tpl.Category].Name, x.Code, cfg.Categories[x.Category].Name)
			continue
		}
		k := key{op.Day, op.Room, op.Start}
		if used[k] {
			c.fail(p, "slot already used by another opening appointment")
		}
		used[k] = true
		cov, ok := covs[op.Day]
		if !ok {
			plan := cfg.Plan(date)
			cv := plan.Coverage()
			cov = &cv
			covs[op.Day] = cov
		}
		if ok, why := cov.SlotBookable(op.Room, tpl.Span, x.Contrast); !ok {
			c.fail(p, "slot is not bookable (%s)", why)
		}
	}
	if len(c.issues) > 0 {
		return
	}
	// The same booking rules as any appointment, in order: shared
	// technologists can't be double-booked.
	problems := sim.CheckOpening(cfg)
	for i := range cfg.Opening {
		if why, ok := problems[i]; ok {
			c.fail(fmt.Sprintf("openingSchedule[%d]", i), "%s", why)
		}
	}
}

// arrivals validates no-shows, arrival classes, walk-in patience and the
// walk-in stream (SIMULATION_DESIGN §5.2, §5.6, §5.8).
func (c *checker) arrivals(f *File, cfg *sim.Config) {
	b := &cfg.Behavior
	ns := f.Behavior.NoShow
	b.NoShowPPM = map[string]rng.PPM{}
	for _, m := range modalities {
		v, ok := ns.BasePPM[m]
		if !ok {
			c.fail("behavior.noShow.basePPM."+m, "required")
		}
		b.NoShowPPM[m] = c.ppm("behavior.noShow.basePPM."+m, v)
	}
	for k := range ns.BasePPM {
		if !slices.Contains(modalities, k) {
			c.fail("behavior.noShow.basePPM."+k, "not a modality")
		}
	}
	b.NoShowPerTenDaysPPM = c.ppm("behavior.noShow.perTenDaysLeadPPM", ns.PerTenDaysLeadPPM)
	b.NoShowMaxLeadPPM = c.ppm("behavior.noShow.maxLeadPPM", ns.MaxLeadPPM)
	b.NoShowRebookPPM = c.ppm("behavior.noShow.rebookPPM", ns.RebookPPM)

	var wsum int64
	for i, a := range f.Behavior.Arrival {
		ap := fmt.Sprintf("behavior.arrivalClasses[%d]", i)
		if a.WeightBP < 0 {
			c.fail(ap+".weightBP", "negative")
		}
		if a.FromMin > a.ToMin || a.FromMin < -120 || a.ToMin > 180 {
			c.fail(ap, "need -120 ≤ fromMin ≤ toMin ≤ 180")
		}
		wsum += a.WeightBP
		b.Arrival = append(b.Arrival, sim.ArrivalClass{WeightBP: uint64(max(0, a.WeightBP)),
			From: simtime.Duration(a.FromMin) * simtime.Minute, To: simtime.Duration(a.ToMin) * simtime.Minute})
	}
	if wsum != 10_000 {
		c.fail("behavior.arrivalClasses", "weights sum to %d, not 10000", wsum)
	}
	b.WalkInPatienceMin = c.quantile("behavior.walkInPatienceMin", f.Behavior.WalkInPatienceMin, 1)
	rg := f.Behavior.Registration
	b.RegScheduled = c.quantile("behavior.registration.scheduledSec", rg.ScheduledSec, 30)
	b.RegWalkIn = c.quantile("behavior.registration.walkInSec", rg.WalkInSec, 30)
	if rg.OpsFactorBP < 10_000 || rg.OpsFactorBP > 30_000 || rg.SelfRegFactorBP < 10_000 || rg.SelfRegFactorBP > 30_000 {
		c.fail("behavior.registration", "opsFactorBP and selfRegFactorBP must be 10000–30000")
	}
	b.RegOpsBP, b.RegSelfBP = rg.OpsFactorBP, rg.SelfRegFactorBP

	w := f.WalkIns
	wc := &cfg.WalkIns
	if w.DailyMilli < 0 || w.DailyMilli > 200_000 {
		c.fail("walkIns.dailyMilli", "%d is outside 0–200000", w.DailyMilli)
	}
	wc.DailyMilli = w.DailyMilli
	var hsum int64
	for k, v := range w.HourProfileBP {
		h, err := strconv.Atoi(k)
		if err != nil || k != fmt.Sprintf("%02d", h) || h < 0 || h > 23 || v < 0 {
			c.fail("walkIns.hourProfileBP."+k, "keys are hours \"00\" … \"23\" with non-negative shares")
			continue
		}
		wc.HourBP[h] = v
		hsum += v
	}
	if hsum != 10_000 {
		c.fail("walkIns.hourProfileBP", "sums to %d, not 10000", hsum)
	}
	if len(w.WeekdayFactorBP) != 7 {
		c.fail("walkIns.weekdayFactorBP", "needs all seven weekdays")
	}
	for _, wd := range weekdays(c, "walkIns.weekdayFactorBP", w.WeekdayFactorBP) {
		v := w.WeekdayFactorBP[wd.String()]
		if v < 0 || v > 30_000 {
			c.fail("walkIns.weekdayFactorBP."+wd.String(), "%d is outside 0–30000", v)
		}
		wc.WeekdayBP[wd] = v
	}
	copy(wc.PriorityBP[:], c.mixBP("walkIns.priorityMixBP", w.PriorityMixBP, priorities))
	for code := range w.ExamMix {
		found := false
		for _, x := range cfg.Catalog {
			if x.Code == code {
				found = true
				if !x.WalkInEligible {
					c.fail("walkIns.examMix."+code, "exam is not walk-in eligible")
				}
			}
		}
		if !found {
			c.fail("walkIns.examMix."+code, "not an exam code in the catalog")
		}
	}
	for xi, x := range cfg.Catalog {
		if v, ok := w.ExamMix[x.Code]; ok && v > 0 && x.WalkInEligible {
			wc.Exams = append(wc.Exams, xi)
			wc.ExamWeights = append(wc.ExamWeights, uint64(v))
		}
	}
	if len(wc.Exams) == 0 && w.DailyMilli > 0 {
		c.fail("walkIns.examMix", "must not be empty")
	}
	for h := range wc.HourBP {
		for wd := range wc.WeekdayBP {
			if wc.DailyMilli*wc.HourBP[h]/10_000*wc.WeekdayBP[wd]/10_000 > maxBlockMilli {
				c.fail("walkIns", "hourly rate above %d milli-patients", maxBlockMilli)
				return
			}
		}
	}
}

func (c *checker) ppm(p string, v int64) rng.PPM {
	if v < 0 || v > 1_000_000 {
		c.fail(p, "%d is outside 0–1000000", v)
		return 0
	}
	return rng.PPM(v)
}
