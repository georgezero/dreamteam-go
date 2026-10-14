package sim_test

import (
	"testing"

	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/scenario"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// The calibrated established practice (step 18) is a struggling,
// capacity-constrained practice. Broad bands, from the nine-seed
// measurements in docs/calibration, checked on three seeds over 30 days
// from the takeover: wide enough for honest variability, narrow enough to
// catch a change of character. The full nine-seed report is the tagged
// calibration suite (go test -tags calibration ./internal/sim).
func TestEstablishedCharacterization(t *testing.T) {
	if testing.Short() {
		t.Skip("three 30-day established runs")
	}
	cfg, err := scenario.Resolve("established")
	if err != nil {
		t.Fatal(err)
	}
	for _, seed := range []int64{1, 4, 7} {
		w := sim.New(cfg, rng.SeedFromInt(seed))
		start := w.StartTime()
		var backlog []int
		for d := int64(1); d < 30; d++ {
			at := w.Calendar().StartOfDay(w.Calendar().DayIndex(start) + d).Add(7*simtime.Hour + 30*simtime.Minute)
			w.Engine().RunUntil(at, nil)
			wl := w.Observe(0).Worklist
			backlog = append(backlog, wl.Unread+wl.Reading)
		}
		w.Engine().RunUntil(start.Add(30*simtime.Day), nil)
		ev := w.Evaluation()
		o, m := ev.Ops, ev.Money
		demand := o.Demand.Received + o.WalkIns.Arrived
		var us, xr int
		for _, md := range o.Demand.ByModality {
			switch md.Modality {
			case "US":
				us = md.Lost
			case "XR":
				xr = md.Lost
			}
		}
		maxBacklog := 0
		for _, b := range backlog {
			maxBacklog = max(maxBacklog, b)
		}
		checks := []struct {
			what   string
			ok     bool
			detail any
		}{
			{"demand ≈ 1,650 a month (orders + walk-ins)", demand >= 1450 && demand <= 1850, demand},
			{"exams well below demand: 1,250–1,500 a month", o.Exams >= 1250 && o.Exams <= 1500, o.Exams},
			{"struggling: margin −20% to −6%, never in the 6–12% band", m.Margin.Defined() && m.Margin.BP() >= -2000 && m.Margin.BP() <= -600, m.Margin.BP()},
			{"capacity-constrained: 25–65% of walk-ins leave", o.WalkIns.Left*100 >= 25*o.WalkIns.Arrived && o.WalkIns.Left*100 <= 65*o.WalkIns.Arrived, o.WalkIns},
			{"US loses the most demand", us > xr, [2]int{us, xr}},
			{"reporting keeps up: TAT p50 under 2 h, morning backlog under 40", o.Reporting.TAT.P50Sec < 7200 && maxBacklog < 40, [2]int64{o.Reporting.TAT.P50Sec, int64(maxBacklog)}},
			{"books reconcile", m.Reconciled, m.Reconciled},
		}
		for _, c := range checks {
			if !c.ok {
				t.Errorf("seed %d: %s: got %v", seed, c.what, c.detail)
			}
		}
	}
}
