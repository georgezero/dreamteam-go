package sim_test

import (
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Step 18 review finding 3: walk-in arrival, same-visit admission,
// conversion to a scheduled appointment and the eventual exam are separate
// concepts. A walk-in returning on a later day for a booked exam is not a
// walk-in imaged that day; walk-in losses are not referral demand losses;
// an offer is not a booking.

// wiTruth classifies the walk-ins who arrived in [from, to) as of to,
// independently of the engine: from visits (the first exam start and
// whether it was on the walk-in visit) and appointments (the first
// booking).
func wiTruth(w *sim.World, from, to simtime.Time) api.WalkInCohort {
	exam := map[string]sim.VisitInfo{}
	for _, v := range w.Visits() {
		if _, ok := exam[v.Order]; !ok && v.ExamStart != 0 {
			exam[v.Order] = v
		}
	}
	booked := map[string]simtime.Time{}
	for _, a := range w.Appointments() {
		if _, ok := booked[a.Order]; !ok {
			booked[a.Order] = a.BookedAt
		}
	}
	ended := map[string]simtime.Time{} // left: the walk-in visit's end
	for _, v := range w.Visits() {
		if v.WalkIn && v.State == "left" {
			ended[v.Order] = v.EndedAt
		}
	}
	var c api.WalkInCohort
	for _, o := range w.Orders() {
		if !o.WalkIn || o.Received < from || o.Received >= to {
			continue
		}
		c.Arrived++
		v, examined := exam[o.ID]
		examined = examined && v.ExamStart < to
		b, hasBooking := booked[o.ID]
		switch {
		case examined && v.Appt < 0:
			c.ImagedSameVisit++
		case hasBooking && b < to:
			c.Converted++
			if examined {
				c.ConvertedImaged++
			}
		case o.State == "left" && ended[o.ID] < to:
			c.Left++
		case o.State == "lost" && o.Received < to && booked[o.ID] == 0:
			c.Lost++ // lost on arrival: the offer found no acceptable slot
		default:
			c.OnSite++
		}
	}
	return c
}

func checkWalkIns(t *testing.T, what string, m api.WalkInMetrics, truth api.WalkInCohort) {
	t.Helper()
	c := m.Cohort
	if c != truth {
		t.Errorf("%s: cohort %+v, independent classification %+v", what, c, truth)
	}
	if c.ImagedSameVisit+c.Converted+c.Left+c.Lost+c.OnSite != c.Arrived || c.ConvertedImaged > c.Converted {
		t.Errorf("%s: cohort does not partition the arrivals: %+v", what, c)
	}
	if c.Arrived != m.Arrived || m.Admitted+m.Offered != m.Arrived || m.Declined+m.Unservable != m.Offered ||
		m.OfferBooked+m.OfferFailed != m.Offered {
		t.Errorf("%s: activity does not add up: %+v", what, m)
	}
	if m.Acceptance.Num > m.Acceptance.Den || m.Abandonment.Num > m.Abandonment.Den ||
		m.Acceptance.Num+m.Abandonment.Num > int64(c.Arrived) {
		t.Errorf("%s: rates above 100%%: acceptance %v, abandonment %v", what, m.Acceptance, m.Abandonment)
	}
}

func TestRegress18WalkInCohorts(t *testing.T) {
	cfg := established(t)
	w := sim.New(cfg, rng.SeedFromInt(1))
	cal := w.Calendar()
	start := cal.DayIndex(w.StartTime())
	const days = 21
	// No new orders or walk-ins for the last week: returns and pending
	// bookings keep completing with no arrivals.
	stop := cal.StartOfDay(start + days - 7)
	w.DropOrdersFrom(stop)
	end := cal.StartOfDay(start + days)

	// Mid-day: walk-ins still on site are pending, not anything else.
	pendingSeen := false
	for d := start + 1; d < start+4; d++ {
		noon := cal.At(d, 11*simtime.Hour)
		if err := runTo(w, noon); err != nil {
			t.Fatal(err)
		}
		s := day(t, w, d)
		checkWalkIns(t, "partial "+s.Date, s.Ops.WalkIns, wiTruth(w, cal.StartOfDay(d), noon))
		pendingSeen = pendingSeen || s.Ops.WalkIns.Cohort.OnSite > 0
	}
	if err := runTo(w, end); err != nil {
		t.Fatal(err)
	}

	visits := w.Visits()
	isWI := map[string]bool{}
	for _, o := range w.Orders() {
		isWI[o.ID] = o.WalkIn
	}
	var returns, failed, quiet, sameTotal, returnTotal int
	for d := start; d < start+days; d++ {
		s := day(t, w, d)
		m := s.Ops.WalkIns
		checkWalkIns(t, s.Date, m, wiTruth(w, cal.StartOfDay(d), cal.StartOfDay(d+1)))
		// Exams of walk-in orders started that day, from the visits.
		var same, ret int
		for _, v := range visits {
			if !v.WalkIn || v.ExamStart < cal.StartOfDay(d) || v.ExamStart >= cal.StartOfDay(d+1) {
				continue
			}
			if v.Appt < 0 {
				same++
			} else {
				ret++
			}
		}
		if m.SameVisitExams != same || m.ReturnExams != ret {
			t.Errorf("%s: same-visit / return exams %d / %d, visits say %d / %d", s.Date, m.SameVisitExams, m.ReturnExams, same, ret)
		}
		sameTotal += same
		returnTotal += ret
		returns += m.ReturnExams
		failed += m.OfferFailed
		if m.Arrived == 0 && m.ReturnExams > 0 {
			quiet++
			if m.Acceptance.Defined() {
				t.Errorf("%s: no walk-ins, yet acceptance %v", s.Date, m.Acceptance)
			}
		}
	}
	if returns == 0 || failed == 0 || quiet == 0 || !pendingSeen {
		t.Fatalf("scenario does not exercise the cases: returns %d, failed offers %d, quiet days with returns %d, pending %v",
			returns, failed, quiet, pendingSeen)
	}

	// The evaluation window: walk-ins on the takeover day before the
	// takeover, and orders inherited with the opening schedule, are not
	// in its cohort.
	ev := w.Evaluation().Ops
	checkWalkIns(t, "evaluation", ev.WalkIns, wiTruth(w, w.StartTime(), end))
	if ev.WalkIns.SameVisitExams+ev.WalkIns.ReturnExams > sameTotal+returnTotal {
		t.Errorf("evaluation counts %d walk-in exams, the days %d", ev.WalkIns.SameVisitExams+ev.WalkIns.ReturnExams, sameTotal+returnTotal)
	}

	// Referral demand excludes walk-in-origin booking events; those are
	// counted apart, every booking of a walk-in order once (offers and
	// rebookings).
	var wiBooked, refBooked, wiLost int
	for _, a := range w.Appointments() {
		if a.BookedAt < w.StartTime() || a.BookedAt >= end {
			continue
		}
		if isWI[a.Order] {
			wiBooked++
		} else {
			refBooked++
		}
	}
	for _, o := range w.Orders() {
		if o.WalkIn && o.State == "lost" && o.Received >= w.StartTime() {
			wiLost++
		}
	}
	dm := ev.Demand
	if dm.WalkInOrigin.Booked != wiBooked || dm.Booked != refBooked {
		t.Errorf("bookings: referral %d (appointments %d), walk-in origin %d (appointments %d)", dm.Booked, refBooked, dm.WalkInOrigin.Booked, wiBooked)
	}
	if dm.WalkInOrigin.Lost < wiLost || dm.WalkInOrigin.Booked < ev.WalkIns.OfferBooked {
		t.Errorf("walk-in-origin demand %+v: %d walk-in orders lost, %d offers booked", dm.WalkInOrigin, wiLost, ev.WalkIns.OfferBooked)
	}
	if dm.Lost > dm.Received {
		t.Errorf("referral lost %d of %d received", dm.Lost, dm.Received)
	}

	// Each completed order has exactly one exam, one study, and (once
	// final) one claim.
	exams, studies, claims := map[string]int{}, map[string]int{}, map[string]int{}
	for _, v := range visits {
		if v.ExamStart != 0 {
			exams[v.Order]++
		}
	}
	for _, o := range w.StudyOrders() {
		studies[o]++
	}
	for _, o := range w.ClaimOrders() {
		claims[o]++
	}
	for _, o := range w.Orders() {
		switch o.State {
		case "imaged", "reported":
			if exams[o.ID] != 1 || studies[o.ID] != 1 || o.State == "reported" && claims[o.ID] != 1 {
				t.Errorf("order %s (%s, walk-in %v): %d exams, %d studies, %d claims", o.ID, o.State, o.WalkIn, exams[o.ID], studies[o.ID], claims[o.ID])
			}
		default:
			if studies[o.ID]+claims[o.ID] != 0 {
				t.Errorf("order %s (%s): %d studies, %d claims", o.ID, o.State, studies[o.ID], claims[o.ID])
			}
		}
	}
}
