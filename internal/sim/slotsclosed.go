package sim

import (
	"slices"

	"github.com/georgezero/dreamteam-go/internal/api"
)

// Slots closed by a termination (V0.13b, E4): an opt-in, read-only count
// of the scheduled capacity a termination-only batch removes. It is a
// separate request: PreviewStaffing and its answer are unchanged.
//
// What is counted: template appointment slots of the days already in the
// booking horizon (materialized), from the first day the batch changes
// anything to the last materialized day, whose room coverage makes them
// bookable now and would not under the proposed registry — the batch's
// ends set, nothing else changed. A slot is counted once, booked or not;
// BookedNow is how many of them hold a booked appointment now. Nothing
// beyond the booking horizon is counted (its slots do not exist yet), and
// nothing is said of demand: a closed slot is capacity, not an order, a
// patient or revenue.

// SlotsClosedSchema identifies the report.
const SlotsClosedSchema = "dreamteam.slots-closed/v1"

// Reasons a report is not available.
const (
	SlotsClosedNotTerminationOnly = "not_termination_only" // empty, or another action type in the batch
	SlotsClosedInvalid            = "invalid_batch"        // the engine refuses the batch
)

// SlotsClosedReport is the count, by modality and by day.
type SlotsClosedReport struct {
	Schema    string `json:"schema"`
	AsOf      string `json:"asOf"`
	Available bool   `json:"available"`
	Reason    string `json:"reason,omitempty"`
	// From and To are the days counted, inclusive (empty when no
	// materialized day is affected); Days how many.
	From       string               `json:"from,omitempty"`
	To         string               `json:"to,omitempty"`
	Days       int64                `json:"days"`
	Total      int                  `json:"total"`
	BookedNow  int                  `json:"bookedNow"`
	ByModality []api.ModalityCount  `json:"byModality"`
	ByDay      []SlotsClosedDayLine `json:"byDay"`
}

// SlotsClosedDayLine is one day's count for one modality.
type SlotsClosedDayLine struct {
	Date      string `json:"date"`
	Modality  string `json:"modality"`
	Count     int    `json:"count"`
	BookedNow int    `json:"bookedNow"`
}

// PreviewSlotsClosed counts the slots a termination-only batch closes
// (see SlotsClosedReport). It changes nothing.
func (w *World) PreviewSlotsClosed(actions []api.Action) SlotsClosedReport {
	st := w.st
	prev := st.now
	st.now = w.eng.Now()
	defer func() { st.now = prev }()
	out := SlotsClosedReport{Schema: SlotsClosedSchema, AsOf: st.cal.ISO(st.now), ByModality: []api.ModalityCount{}, ByDay: []SlotsClosedDayLine{}}
	if len(actions) == 0 || slices.ContainsFunc(actions, func(a api.Action) bool { return a.Type != api.ActTerminate }) {
		out.Reason = SlotsClosedNotTerminationOnly
		return out
	}
	for _, r := range w.ValidateBatch(actions) {
		if !r.OK {
			out.Reason = SlotsClosedInvalid
			return out
		}
	}
	_, p := st.staffBatch(w.eng, actions, false, v2Actions)
	h := newHypo(st, p)
	first := noEnd
	for _, u := range h.until {
		first = min(first, u)
	}
	out.Available = true
	n := int64(len(st.dem.days))
	from := max(first, st.day+1)
	if from >= n {
		return out // every end date is past the materialized days
	}
	date := func(d int64) string { return st.cal.Date(st.cal.StartOfDay(d)).String() }
	out.From, out.To, out.Days = date(from), date(n-1), n-from
	plans := map[int64]DayPlan{}
	for d := from; d < n; d++ {
		plans[d] = st.planHypo(d, h)
	}
	byDay, byDayBooked := coverageClosed(st, plans, from, n)
	total, booked := map[string]int{}, map[string]int{}
	for d := from; d < n; d++ {
		for _, m := range st.modalities() {
			if c := byDay[d][m]; c > 0 {
				out.ByDay = append(out.ByDay, SlotsClosedDayLine{Date: date(d), Modality: m, Count: c, BookedNow: byDayBooked[d][m]})
				total[m] += c
				booked[m] += byDayBooked[d][m]
			}
		}
	}
	for _, m := range st.modalities() {
		if total[m] > 0 {
			out.ByModality = append(out.ByModality, api.ModalityCount{Modality: m, Count: total[m]})
			out.Total += total[m]
			out.BookedNow += booked[m]
		}
	}
	return out
}

// coverageClosed counts, for each day in [from, to) with a plan, the
// template slots bookable now by coverage that plans would not cover, by
// modality, and how many of them hold a booked appointment now. It is the
// counting rule of previewRosters' SlotsClosed (a test checks they agree).
func coverageClosed(st *state, plans map[int64]DayPlan, from, to int64) (map[int64]map[string]int, map[int64]map[string]int) {
	closed, booked := map[int64]map[string]int{}, map[int64]map[string]int{}
	for d := from; d < to; d++ {
		pl, ok := plans[d]
		if !ok || pl.Hours == nil {
			continue
		}
		cov := pl.Coverage()
		ds := st.dem.days[d]
		closed[d], booked[d] = map[string]int{}, map[string]int{}
		for r := range ds.rooms {
			for _, s := range ds.rooms[r] {
				x := s
				st.coverSlot(&cov, r, &x)
				if s.why == slotOK && x.why != slotOK {
					m := st.cfg.Rooms[r].Modality
					closed[d][m]++
					if s.appt >= 0 && st.dem.appts[s.appt].state == apptBooked {
						booked[d][m]++
					}
				}
			}
		}
	}
	return closed, booked
}
