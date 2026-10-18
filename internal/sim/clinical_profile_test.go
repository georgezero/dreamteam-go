package sim_test

import (
	"cmp"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/eval"
	"github.com/georgezero/dreamteam-go/internal/scenario"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// The simulator profile of Slice 5C.4b's final calibration correction
// (docs/calibration/clinical-5c4b/fidelity-protocol.md): the evaluation
// scenario's own practice, with no action, on the development seeds,
// recorded as the evaluation records it, and measured for what the
// synthetic clinical calibration model must reproduce (weekday and
// priority demand, acquisition hours, modalities, read and documentation
// times, reading blocks) and what it is validated against (utilization,
// queue age, delay tails, the weekend). No strategy runs: the world
// takes no action.

const profileManifest = "docs/calibration/feasibility/step0c/experiments/W130-SHR-MRft-s20.json"

// profileConfig compiles the evaluation scenario (the development sets'
// scenario manifest), with an open-ended evaluation window.
func profileConfig(t *testing.T) *sim.Config {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("../..", profileManifest))
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		Scenario string          `json:"scenario"`
		Overlay  json.RawMessage `json:"overlay"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	f, err := scenario.ResolveFile(m.Scenario)
	if err != nil {
		t.Fatal(err)
	}
	canon, err := scenario.Canonical(f)
	if err != nil {
		t.Fatal(err)
	}
	var b, o any
	json.Unmarshal(canon, &b)
	if err := json.Unmarshal(m.Overlay, &o); err != nil {
		t.Fatal(err)
	}
	merged, _ := json.Marshal(scenario.Merge(b, o))
	if f, err = scenario.Decode(merged); err != nil {
		t.Fatal(err)
	}
	cfg, err := scenario.Compile(f)
	if err != nil {
		t.Fatal(err)
	}
	cfg.EvaluationDays = 0
	return cfg
}

// recordBaseline runs seed with no action to the policy's observation
// end, recorded as the evaluation records a plan.
func recordBaseline(p *eval.Policy, cfg *sim.Config, seed int64) eval.Record {
	return recordBaselineWith(p, cfg, seed, nil)
}

// recordBaselineWith is recordBaseline with observe called on the new
// world before it runs (an observer that does not change the run).
func recordBaselineWith(p *eval.Policy, cfg *sim.Config, seed int64, observe func(*sim.World)) eval.Record {
	w := sim.NewEvalWorld(cfg, seed)
	if observe != nil {
		observe(w)
	}
	start := w.StartTime()
	w.Engine().RunUntil(start, nil)
	rec := w.NewEvalRecorder("baseline", "", "", seed, p.Finance.EmployeeCommitmentDays)
	horizon := eval.TailEnd(p, int64(start), 0, 0)
	h := p.Horizons
	day := func(n int) simtime.Time { return start.Add(simtime.Duration(n) * simtime.Day) }
	end := simtime.Time(eval.ObservationEnd(p, int64(start), w.Calendar().Date(start).String(), horizon))
	type mark struct {
		at   simtime.Time
		day  int
		name string
	}
	var marks []mark
	for d := 1; day(d) <= end; d++ {
		marks = append(marks, mark{at: day(d), day: d})
	}
	for _, c := range []mark{{at: day(h.FeedbackDays), name: "T30"}, {at: day(h.PrimaryDays), name: "T90"}, {at: simtime.Time(horizon), name: "tailEnd"},
		{at: day(h.ValidationDays), name: "T180"}, {at: end, name: "end"}} {
		marks = append(marks, c)
	}
	slices.SortStableFunc(marks, func(x, y mark) int { return cmp.Compare(x.at, y.at) })
	for _, m := range marks {
		w.Engine().RunUntil(m.at, nil)
		if m.name == "" {
			rec.Sample(m.day)
		} else {
			rec.Capture(m.name)
		}
	}
	if err := w.Check(); err != nil {
		panic(err)
	}
	return rec.Finish(0, w.LatestEffect(), horizon, "")
}

// The profile (dreamteam.simprofile/v1). Counts are totals over the
// seeds; per-seed measures are listed by seed.
type simProfile struct {
	Schema   string  `json:"schema"`
	Source   string  `json:"source"`
	Manifest string  `json:"scenarioManifest"`
	Seeds    []int64 `json:"seeds"`
	Window   string  `json:"window"`
	// Weekday (Monday 0) by priority: orders received, studies acquired,
	// and the number of such days in the window (all seeds).
	Days     [7]int             `json:"daysByWeekday"`
	Orders   map[string][7]int  `json:"ordersByWeekday"`
	WalkIns  map[string][7]int  `json:"walkInOrdersByWeekday"`
	Acquired map[string][7]int  `json:"acquiredByWeekday"`
	Hours    map[string][24]int `json:"acquisitionsByHour"` // "weekday" (Mon–Fri), "saturday"
	// By priority: acquisitions by hour (Mon–Fri), the wait from worklist
	// entry to claim (percentiles, seconds), and the breach share by
	// acquisition hour (Mon–Fri).
	HoursByPriority map[string][24]int     `json:"weekdayAcquisitionsByHourPriority"`
	ClaimWait       map[string][]int64     `json:"claimWaitSecPercentiles"`
	BreachByHour    map[string][24]float64 `json:"weekdayBreachShareByHourPriority"`
	// Studies acquired a day, Mon–Fri and Saturday: percentiles 0..100.
	DailyAcquired map[string][]int64        `json:"dailyAcquiredPercentiles"`
	Modality      map[string]map[string]int `json:"acquiredByPriorityModality"`
	// Order outcomes at the end of observation, orders received in the
	// window, by priority.
	Outcomes map[string]map[string]int `json:"orderOutcomes"`
	// Wanted-to-acquisition lead of served orders (days, by priority):
	// percentiles 0..100.
	ReadSec  map[string][]int64 `json:"readSecPercentiles"`  // by modality: final − read start
	EntrySec map[string][]int64 `json:"entrySecPercentiles"` // by modality: entered − acquired
	// Reading blocks: site reading minutes in a week, by weekday, for rota
	// weeks A and B, and each radiologist's shifts and allocation.
	Readers []profileReader  `json:"readers"`
	PerSeed []profileSeed    `json:"perSeed"`
	Targets map[string]int64 `json:"tatTargetMin"`
}

type profileReader struct {
	Name          string         `json:"name"`
	AllocationBP  int64          `json:"siteAllocationBP"`
	Shifts        [2][7][2]int64 `json:"shiftsMin"` // [week A, B][weekday]: from, to (minutes after midnight; 0, 0: none)
	Remote        [2][7]bool     `json:"remote"`
	BlockMinWeekA int64          `json:"blockMinutesWeekA"`
	BlockMinWeekB int64          `json:"blockMinutesWeekB"`
}

type profileSeed struct {
	Seed            int64              `json:"seed"`
	Acquired        int                `json:"acquiredInWindow"`
	Utilization     float64            `json:"readingUtilization"` // read time of the window's acquisitions over the window's block time
	UtilizationLate float64            `json:"readingUtilizationLast30Days"`
	WorstPastDueMin map[string]int64   `json:"worstPastDueMin"`
	SevereShare     map[string]float64 `json:"severeShare"` // more than 1,440 min past due
	BreachShare     map[string]float64 `json:"breachShare"`
	SatPMUrgentMin  int64              `json:"urgentAcquiredSaturdayAfter14WorstPastDueMin"`
	AgeMaxMin       []int64            `json:"sampleAgeMaxMinPercentiles"` // 0, 10, 50, 90, 100 of the daily samples
	UnreadP         []int64            `json:"sampleUnreadPercentiles"`
	TakeoverUnread  int                `json:"takeoverUnread"`
	TakeoverOverdue int                `json:"takeoverOverdue"`
	TakeoverOrders  int                `json:"takeoverOpenOrders"`
}

func percentiles(xs []int64) []int64 {
	if len(xs) == 0 {
		return nil
	}
	s := slices.Clone(xs)
	slices.Sort(s)
	out := make([]int64, 101)
	for p := range out {
		out[p] = s[min(len(s)-1, p*(len(s)-1)/100)]
	}
	return out
}

func TestSimulatorClinicalProfile(t *testing.T) {
	out := os.Getenv("DT_SIM_PROFILE")
	if out == "" {
		t.Skip("set DT_SIM_PROFILE=<file> for the simulator profile (docs/calibration/clinical-5c4b/fidelity-protocol.md)")
	}
	p, _, err := eval.LoadPolicy("../../" + eval.PolicyPath)
	if err != nil {
		t.Fatal(err)
	}
	seeds, err := eval.DevelopmentSeeds(p)
	if err != nil {
		t.Fatal(err)
	}
	cfg := profileConfig(t)
	recs := make([]eval.Record, len(seeds))
	var wg sync.WaitGroup
	for i, s := range seeds {
		wg.Add(1)
		go func() {
			defer wg.Done()
			recs[i] = recordBaseline(p, cfg, s)
		}()
	}
	wg.Wait()
	pr := simProfile{Schema: "dreamteam.simprofile/v1", Source: "the evaluation scenario's practice with no action, development seeds, recorded by the evaluation recorder",
		Manifest: profileManifest, Seeds: seeds, Window: "acquired (or received) after the takeover and by the horizon",
		Orders: map[string][7]int{}, WalkIns: map[string][7]int{}, Acquired: map[string][7]int{}, Hours: map[string][24]int{},
		HoursByPriority: map[string][24]int{}, ClaimWait: map[string][]int64{}, BreachByHour: map[string][24]float64{}, DailyAcquired: map[string][]int64{},
		Modality: map[string]map[string]int{}, Outcomes: map[string]map[string]int{}, ReadSec: map[string][]int64{}, EntrySec: map[string][]int64{},
		Targets: recs[0].TATTargetMin}
	readers, weekBlocks := profileReaders(cfg)
	pr.Readers = readers
	reads, entries := map[string][]int64{}, map[string][]int64{}
	waits, daily := map[string][]int64{}, map[string][]int64{}
	var brN, brK [3][24]float64
	for _, r := range recs {
		t0, H := r.TakeoverSec, r.HorizonEndSec
		date, _ := simtime.ParseDate(r.TakeoverDate)
		// Day index from the takeover date's midnight (local seconds:
		// the takeover is at its date's clock time).
		mid0 := t0 - (t0 % 86400)
		dayOf := func(sec int64) int { return int((sec - mid0) / 86400) }
		wdOf := func(d int) int { return (int(date.AddDays(int64(d)).Weekday()) + 6) % 7 }
		clockMin := func(sec int64) int64 { return (sec - mid0) % 86400 / 60 }
		// The window's days: the takeover's (from 06:00) to the horizon's
		// eve (the horizon is at the same clock time 150 days on).
		for d := 0; mid0+int64(d+1)*86400 <= H; d++ {
			pr.Days[wdOf(d)]++
		}
		end := &r.Captures[len(r.Captures)-1]
		inherited := 0
		for i, o := range r.Orders {
			if o.ReceivedSec <= t0 {
				inherited++
				continue
			}
			if o.ReceivedSec > H {
				continue
			}
			wd := wdOf(dayOf(o.ReceivedSec))
			c := pr.Orders[o.Priority]
			c[wd]++
			pr.Orders[o.Priority] = c
			if o.WalkIn {
				c := pr.WalkIns[o.Priority]
				c[wd]++
				pr.WalkIns[o.Priority] = c
			}
			if pr.Outcomes[o.Priority] == nil {
				pr.Outcomes[o.Priority] = map[string]int{}
			}
			pr.Outcomes[o.Priority][end.OrderOutcomes[i]]++
		}
		ps := profileSeed{Seed: r.Seed, WorstPastDueMin: map[string]int64{}, SevereShare: map[string]float64{}, BreachShare: map[string]float64{},
			TakeoverOrders: inherited}
		n, sev, br := map[string]int{}, map[string]int{}, map[string]int{}
		perDay := map[int]int{}
		var readSec, readLate float64
		late := H - 30*86400
		E := r.ObservedUntilSec
		for _, s := range r.Studies {
			tgt := r.TATTargetMin[s.Priority] * 60
			if s.AcquiredSec <= t0 {
				ps.TakeoverUnread++
				if t0-s.AcquiredSec > tgt {
					ps.TakeoverOverdue++
				}
				continue
			}
			if s.AcquiredSec > H {
				continue
			}
			ps.Acquired++
			d := dayOf(s.AcquiredSec)
			wd := wdOf(d)
			perDay[d]++
			k := slices.Index([]string{"urgent", "soon", "routine"}, s.Priority)
			if s.ClaimedSec != 0 {
				waits[s.Priority] = append(waits[s.Priority], s.ClaimedSec-s.EnteredSec)
			}
			if wd < 5 {
				hb := pr.HoursByPriority[s.Priority]
				hb[clockMin(s.AcquiredSec)/60]++
				pr.HoursByPriority[s.Priority] = hb
				fin := s.FinalSec
				if fin == 0 {
					fin = r.ObservedUntilSec
				}
				brN[k][clockMin(s.AcquiredSec)/60]++
				if fin-s.AcquiredSec > tgt {
					brK[k][clockMin(s.AcquiredSec)/60]++
				}
			}
			a := pr.Acquired[s.Priority]
			a[wd]++
			pr.Acquired[s.Priority] = a
			switch {
			case wd < 5:
				h := pr.Hours["weekday"]
				h[clockMin(s.AcquiredSec)/60]++
				pr.Hours["weekday"] = h
			case wd == 5:
				h := pr.Hours["saturday"]
				h[clockMin(s.AcquiredSec)/60]++
				pr.Hours["saturday"] = h
			}
			if pr.Modality[s.Priority] == nil {
				pr.Modality[s.Priority] = map[string]int{}
			}
			pr.Modality[s.Priority][s.Modality]++
			entries[s.Modality] = append(entries[s.Modality], s.EnteredSec-s.AcquiredSec)
			if s.FinalSec != 0 && s.ReadStartSec != 0 {
				reads[s.Modality] = append(reads[s.Modality], s.FinalSec-s.ReadStartSec)
				readSec += float64(s.FinalSec - s.ReadStartSec)
				if s.AcquiredSec > late {
					readLate += float64(s.FinalSec - s.ReadStartSec)
				}
			}
			fin := s.FinalSec
			if fin == 0 || fin > E {
				fin = E
			}
			past := fin - (s.AcquiredSec + tgt)
			n[s.Priority]++
			ps.WorstPastDueMin[s.Priority] = max(ps.WorstPastDueMin[s.Priority], past/60)
			if past > 1440*60 {
				sev[s.Priority]++
			}
			if past > 0 {
				br[s.Priority]++
			}
			if s.Priority == "urgent" && wd == 5 && clockMin(s.AcquiredSec) >= 14*60 {
				ps.SatPMUrgentMin = max(ps.SatPMUrgentMin, past/60)
			}
		}
		for d := 0; mid0+int64(d+1)*86400 <= H; d++ {
			switch wd := wdOf(d); {
			case wd < 5:
				daily["weekday"] = append(daily["weekday"], int64(perDay[d]))
			case wd == 5:
				daily["saturday"] = append(daily["saturday"], int64(perDay[d]))
			}
		}
		for q, k := range n {
			ps.SevereShare[q], ps.BreachShare[q] = float64(sev[q])/float64(k), float64(br[q])/float64(k)
		}
		// Block time in the window, and in its last 30 days.
		var block, blockLate float64
		for d := 0; mid0+int64(d)*86400 < H; d++ {
			wk := rotaWeek(cfg.RotaWeekA, date.AddDays(int64(d)))
			for _, rd := range weekBlocks {
				for _, b := range rd[wk][wdOf(d)] {
					from, to := mid0+int64(d)*86400+b[0]*60, mid0+int64(d)*86400+b[1]*60
					from, to = max(from, t0), min(to, H)
					if to > from {
						block += float64(to - from)
						if lf := max(from, late); to > lf {
							blockLate += float64(to - lf)
						}
					}
				}
			}
		}
		ps.Utilization, ps.UtilizationLate = readSec/block, readLate/blockLate
		var ages, unread []int64
		for _, s := range r.Samples {
			if s.AtSec <= H {
				ages, unread = append(ages, s.AgeMaxMin), append(unread, int64(s.Unread))
			}
		}
		pa, pu := percentiles(ages), percentiles(unread)
		ps.AgeMaxMin = []int64{pa[0], pa[10], pa[50], pa[90], pa[100]}
		ps.UnreadP = []int64{pu[0], pu[10], pu[50], pu[90], pu[100]}
		pr.PerSeed = append(pr.PerSeed, ps)
	}
	for m, xs := range reads {
		pr.ReadSec[m] = percentiles(xs)
	}
	for m, xs := range entries {
		pr.EntrySec[m] = percentiles(xs)
	}
	for q, xs := range waits {
		pr.ClaimWait[q] = percentiles(xs)
	}
	for k, xs := range daily {
		pr.DailyAcquired[k] = percentiles(xs)
	}
	for k, q := range []string{"urgent", "soon", "routine"} {
		var b [24]float64
		for h := range 24 {
			if brN[k][h] > 0 {
				b[h] = brK[k][h] / brN[k][h]
			}
		}
		pr.BreachByHour[q] = b
	}
	raw, err := json.MarshalIndent(pr, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

// rotaWeek: 0 in rota week A (the week of a, and every second week from
// it; weeks run Monday to Sunday), 1 in week B.
func rotaWeek(a, d simtime.Date) int {
	mon := a.AddDays(-int64((int(a.Weekday()) + 6) % 7))
	return int(floorMod(floorDivDays(d.DaysSince(mon), 7), 2))
}

func floorDivDays(a, b int64) int64 {
	q := a / b
	if a%b != 0 && a < 0 {
		q--
	}
	return q
}

func floorMod(a, b int64) int64 { return ((a % b) + b) % b }

// profileReaders are the radiologists' rostered shifts and their site
// reading blocks (sim.ReadingBlocks), in minutes after midnight, by rota
// week and weekday (Monday 0).
func profileReaders(cfg *sim.Config) ([]profileReader, [][2][7][][2]int64) {
	var out []profileReader
	var blocks [][2][7][][2]int64
	for _, e := range cfg.Employees {
		if e.Role != sim.Radiologist {
			continue
		}
		pr := profileReader{Name: e.Name, AllocationBP: e.SiteAllocationBP}
		var bs [2][7][][2]int64
		for wk, week := range []int{sim.WeekA, sim.WeekB} {
			for wd := range 7 {
				sw := simtime.Weekday((wd + 1) % 7) // simtime: Sunday 0
				st := e.ShiftOn(sw, week)
				if st == nil {
					continue
				}
				pr.Shifts[wk][wd] = [2]int64{int64(st.From / simtime.Minute), int64(st.To / simtime.Minute)}
				pr.Remote[wk][wd] = st.Remote
				for _, b := range sim.ReadingBlocks(st.Span, e.SiteAllocationBP) {
					m := [2]int64{int64(b.From / simtime.Minute), int64(b.To / simtime.Minute)}
					bs[wk][wd] = append(bs[wk][wd], m)
					if wk == 0 {
						pr.BlockMinWeekA += m[1] - m[0]
					} else {
						pr.BlockMinWeekB += m[1] - m[0]
					}
				}
			}
		}
		out = append(out, pr)
		blocks = append(blocks, bs)
	}
	return out, blocks
}
