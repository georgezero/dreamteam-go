package sim

import (
	"fmt"

	"github.com/georgezero/dreamteam-go/internal/api"
)

// Test-only access for the slots-closed report (V0.13b, E4).

// BookableByCoverage lists the template slots of the materialized days
// from day on that coverage makes bookable now, as "date room slot" keys.
func (w *World) BookableByCoverage(day int64) map[string]string {
	st := w.st
	out := map[string]string{}
	for d := max(day, st.day+1); d < int64(len(st.dem.days)); d++ {
		ds := st.dem.days[d]
		for r := range ds.rooms {
			for k, s := range ds.rooms[r] {
				if s.why == slotOK {
					out[fmt.Sprintf("%s %d %d", st.cal.Date(st.cal.StartOfDay(d)), r, k)] = st.cfg.Rooms[r].Modality
				}
			}
		}
	}
	return out
}

// CoverageClosedAll runs coverageClosed for any batch the engine accepts,
// planning every materialized day after today under its proposed
// registry, and returns the total by modality.
func (w *World) CoverageClosedAll(actions []api.Action) map[string]int {
	st := w.st
	prev := st.now
	st.now = w.eng.Now()
	defer func() { st.now = prev }()
	_, p := st.staffBatch(w.eng, actions, false, v2Actions)
	h := newHypo(st, p)
	n := int64(len(st.dem.days))
	plans := map[int64]DayPlan{}
	for d := st.day + 1; d < n; d++ {
		plans[d] = st.planHypo(d, h)
	}
	byDay, _ := coverageClosed(st, plans, st.day+1, n)
	out := map[string]int{}
	for _, m := range byDay {
		for k, c := range m {
			out[k] += c
		}
	}
	return out
}
