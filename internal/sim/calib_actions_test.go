//go:build calibration

package sim_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/scenario"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// What can the three V0.1 actions do for the established practice? Each
// strategy is decided at the takeover for the next 30 days (inside the
// booking horizon, so extensions approved ahead can be booked), then the
// practice runs; nine seeds, same demand across strategies.
//
//	go test -tags calibration ./internal/sim -run CalibrationActions -v

type strategy struct {
	name string
	plan func(w *sim.World, days int) []api.Action
}

// extendTo extends the named employees to the given close on every
// weekday they work in the next days.
func extendTo(w *sim.World, days int, names []string, until simtime.Duration) []api.Action {
	cal := w.Calendar()
	start := cal.DayIndex(w.StartTime())
	var as []api.Action
	for d := int64(0); d < int64(days); d++ {
		for _, e := range w.Roster(start + d) {
			if !contains(names, e.Name) {
				continue
			}
			end, _ := simtime.ParseClock(e.RosterEnd)
			cap, _ := simtime.ParseClock(e.CapAt)
			if gap := (min(until, cap) - end) / simtime.Minute; gap > 0 && gap <= 240 {
				as = append(as, api.Action{Type: api.ActExtendShift, Employee: e.ID, Minutes: int64(gap),
					Date: cal.Date(cal.StartOfDay(start + d)).String()})
			}
		}
	}
	return as
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}

var strategies = []strategy{
	{"A no actions", func(*sim.World, int) []api.Action { return nil }},
	{"B US+MR to close, walk-ins until 19:00", func(w *sim.World, days int) []api.Action {
		return append(extendTo(w, days, []string{"Morgan", "Casey"}, 20*simtime.Hour),
			api.Action{Type: api.ActSetWalkInPolicy, Policy: api.WalkInAcceptUntil, Cutoff: "19:00"})
	}},
	{"C 2nd XR/CT tech evenings, accept all walk-ins", func(w *sim.World, days int) []api.Action {
		return append(extendTo(w, days, []string{"Riley"}, 20*simtime.Hour),
			api.Action{Type: api.ActSetWalkInPolicy, Policy: api.WalkInAcceptAlways})
	}},
	{"D all three extensions + accept all", func(w *sim.World, days int) []api.Action {
		return append(extendTo(w, days, []string{"Morgan", "Casey", "Riley"}, 20*simtime.Hour),
			api.Action{Type: api.ActSetWalkInPolicy, Policy: api.WalkInAcceptAlways})
	}},
	{"E decline walk-ins, FIFO worklist", func(*sim.World, int) []api.Action {
		return []api.Action{{Type: api.ActSetWalkInPolicy, Policy: api.WalkInScheduleLater},
			{Type: api.ActSetWorklistPolicy, Policy: api.WorklistFIFO}}
	}},
	{"F shortest-first worklist", func(*sim.World, int) []api.Action {
		return []api.Action{{Type: api.ActSetWorklistPolicy, Policy: api.WorklistShortestFirst}}
	}},
}

func TestCalibrationActions(t *testing.T) {
	cfg, err := scenario.Resolve("established")
	if err != nil {
		t.Fatal(err)
	}
	const days = 30
	type row struct {
		exams, lost, left, declined, tatP50, tatP90, overTarget, otH, apprH, rev, opex, profit, margin, rejected []float64
	}
	rows := make([]row, len(strategies))
	for _, seed := range seedsFromEnv() {
		var demand []sim.OrderInfo
		for k, s := range strategies {
			w := sim.New(cfg, rng.SeedFromInt(seed))
			w.Engine().RunUntil(w.StartTime(), nil)
			if as := s.plan(w, days); len(as) > 0 {
				w.Apply(as)
			}
			w.Engine().RunUntil(w.StartTime().Add(days*simtime.Day), nil)
			if k == 0 {
				demand = exogenous(w.Orders())
			} else if !reflect.DeepEqual(demand, exogenous(w.Orders())) {
				t.Fatalf("seed %d, %s: demand differs from no actions", seed, s.name)
			}
			ev := w.Evaluation()
			o, m := ev.Ops, ev.Money
			r := &rows[k]
			f := func(xs *[]float64, x float64) { *xs = append(*xs, x) }
			f(&r.exams, float64(o.Exams))
			f(&r.lost, float64(o.Demand.Lost))
			f(&r.left, float64(o.WalkIns.Left))
			f(&r.declined, float64(o.WalkIns.Declined))
			f(&r.tatP50, float64(o.Reporting.TAT.P50Sec/60))
			f(&r.tatP90, float64(o.Reporting.TAT.P90Sec/60))
			f(&r.overTarget, pct(o.Reporting.OverTarget, o.Reporting.Acquired))
			f(&r.otH, float64(o.Workforce.OvertimeSec)/3600)
			f(&r.apprH, float64(o.Workforce.ApprovedOvertimeSec)/3600)
			f(&r.rev, float64(m.NetRevenueCents)/100_000)
			f(&r.opex, float64(m.OperatingExpenseCents)/100_000)
			f(&r.profit, float64(m.OperatingProfitCents)/100_000)
			f(&r.margin, float64(m.Margin.BP())/100)
			f(&r.rejected, float64(o.Actions.Rejected))
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "30 days from takeover, %d seeds: mean (min–max)\n", len(seedsFromEnv()))
	fmt.Fprintf(&b, "| strategy | exams | lost orders | walk-ins left / declined | TAT p50 / p90 min | over target %% | overtime h (approved) | net revenue $k | opex $k | profit $k | margin %% |\n|---|---|---|---|---|---|---|---|---|---|---|\n")
	mr := func(xs []float64) string {
		s := stats(xs)
		return fmt.Sprintf("%.0f (%.0f–%.0f)", s.mean, s.min, s.max)
	}
	m1 := func(xs []float64) string {
		s := stats(xs)
		return fmt.Sprintf("%.1f (%.1f–%.1f)", s.mean, s.min, s.max)
	}
	for k, s := range strategies {
		r := rows[k]
		fmt.Fprintf(&b, "| %s | %s | %s | %.0f / %.0f | %.0f / %.0f | %.1f | %.0f (%.0f) | %s | %s | %s | %s |\n", s.name,
			mr(r.exams), mr(r.lost), stats(r.left).mean, stats(r.declined).mean, stats(r.tatP50).mean, stats(r.tatP90).mean,
			stats(r.overTarget).mean, stats(r.otH).mean, stats(r.apprH).mean, m1(r.rev), m1(r.opex), m1(r.profit), m1(r.margin))
	}
	t.Log("\n" + b.String())
}
