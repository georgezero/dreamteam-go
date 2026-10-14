package sim_test

import (
	"slices"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

func TestZZReadSmoke(t *testing.T) {
	cfg := miniConfig(t)
	w := sim.New(cfg, rng.SeedFromInt(1))
	cal := w.Calendar()
	for d := int64(0); d < 21; d++ {
		w.Engine().RunUntil(cal.StartOfDay(d).Add(7*simtime.Hour+30*simtime.Minute), nil)
		backlog := 0
		for _, s := range w.Studies() {
			if s.State == "unread" || s.State == "reading" {
				backlog++
			}
		}
		w.Engine().RunUntil(cal.StartOfDay(d+1)-1, nil)
		var tats []int
		acq, fin := 0, 0
		for _, s := range w.Studies() {
			if cal.DayIndex(s.Acquired) == d {
				acq++
				end := s.Final
				if s.State != "final" {
					end = cal.StartOfDay(d + 1)
				} else {
					fin++
				}
				tats = append(tats, int(end.Sub(s.Acquired)/simtime.Minute))
			}
		}
		slices.Sort(tats)
		med, p90 := 0, 0
		if len(tats) > 0 {
			med, p90 = tats[len(tats)/2], tats[len(tats)*9/10]
		}
		f := w.Observe(0).Flow.Today
		t.Logf("%s backlog@07:30 %d | acquired %d final(same) %d | finalToday %d breaches %d | TAT med %dm p90 %dm",
			cal.Short(cal.StartOfDay(d))[:10], backlog, acq, fin, f.ExamsDone, 0, med, p90)
	}
}
