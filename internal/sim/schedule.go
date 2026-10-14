package sim

import (
	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

var slotWhy = [...]string{slotUnstaffed: api.SlotUnstaffed, slotNoSupervision: api.SlotNoSupervision}

// observeDemand fills the demand summary.
func (st *state) observeDemand(o *api.Observation, now simtime.Time) {
	d := &st.dem
	o.Demand.RecallList = len(d.recall)
	for _, m := range st.modalities() {
		md := api.ModalityDemand{Modality: m, Today: counts(d.today[m]), SinceStart: counts(d.total[m]), NextOpenDays: -1}
		notBefore := now.Add(st.cfg.Policies.BookingNotice)
		today := st.cal.DayIndex(now)
		if today < int64(len(d.days)) {
			for r, room := range st.cfg.Rooms {
				if room.Modality != m {
					continue
				}
				for _, s := range d.days[today].rooms[r] {
					switch {
					case s.appt >= 0:
						md.ApptsToday++
					case s.why == slotOK && st.cal.At(today, s.From) >= notBefore && sharedFree(d.days[today], r, &s):
						md.OpenToday++
					}
				}
			}
		}
		if day, ok := st.nextOpen(m, today, notBefore); ok {
			md.NextOpen = st.cal.Date(st.cal.StartOfDay(day)).String()
			md.NextOpenDays = int(day - today)
		}
		o.Demand.Modalities = append(o.Demand.Modalities, md)
	}
}

func counts(c *modalityCounts) api.DemandCounts {
	return api.DemandCounts{Received: c.received, Booked: c.booked, Recall: c.recall, Lost: c.lost, Cancelled: c.cancelled}
}

// nextOpen finds the first day with a free bookable slot of modality m.
func (st *state) nextOpen(m string, today int64, notBefore simtime.Time) (int64, bool) {
	last := min(today+st.cfg.Policies.BookingHorizonDays, int64(len(st.dem.days))-1)
	for day := max(today, 0); day <= last; day++ {
		for r, room := range st.cfg.Rooms {
			if room.Modality != m {
				continue
			}
			for _, s := range st.dem.days[day].rooms[r] {
				if s.appt < 0 && s.why == slotOK && st.cal.At(day, s.From) >= notBefore && sharedFree(st.dem.days[day], r, &s) {
					return day, true
				}
			}
		}
	}
	return 0, false
}

// Schedule projects the schedule of day index day: planned staffing per
// room, every template slot with its status, and the appointments booked.
// Days outside the materialized range show their template without bookings.
func (w *World) Schedule(day int64) api.DaySchedule {
	st := w.st
	now := w.eng.Now()
	date := st.cal.Date(st.cal.StartOfDay(day))
	out := api.DaySchedule{Date: date.String(), Weekday: date.Weekday().String(), Offset: day - st.cal.DayIndex(now)}
	ds := (*daySlots)(nil)
	if day >= 0 && day < int64(len(st.dem.days)) {
		ds = st.dem.days[day]
	} else {
		ds = st.buildDay(day)
	}
	plan := st.cfg.Plan(date)
	if plan.Hours == nil {
		return out
	}
	out.Hours = &api.Span{From: plan.Hours.From, To: plan.Hours.To}
	cov := plan.Coverage()
	notBefore := now.Add(st.cfg.Policies.BookingNotice)
	for r, room := range st.cfg.Rooms {
		rs := api.RoomSchedule{Room: room.ID, Name: room.Name, Modality: room.Modality}
		for _, seg := range cov.Segments {
			if !seg.Open || len(seg.Rooms[r]) == 0 {
				continue
			}
			if n := len(rs.Staffed); n > 0 && rs.Staffed[n-1].To == seg.From {
				rs.Staffed[n-1].To = seg.To
			} else {
				rs.Staffed = append(rs.Staffed, api.Span{From: seg.From, To: seg.To})
			}
		}
		for _, b := range cov.OnBreak(r) {
			rs.Breaks = append(rs.Breaks, api.Span{From: b.From, To: b.To})
		}
		for k, s := range ds.rooms[r] {
			x := api.Slot{Span: api.Span{From: s.From, To: s.To}, Category: st.cfg.Categories[s.Category].Name}
			switch {
			case s.appt >= 0:
				x.Status = api.SlotBooked
				a := st.apptView(s.appt)
				x.Appt = &a
			case s.why != slotOK:
				x.Status = slotWhy[s.why]
			case st.cal.At(day, s.From) < notBefore:
				x.Status = api.SlotPassed
			case !sharedFree(ds, r, &ds.rooms[r][k]):
				x.Status = api.SlotSharedBusy
			default:
				x.Status = api.SlotOpen
			}
			rs.Slots = append(rs.Slots, x)
		}
		out.Rooms = append(out.Rooms, rs)
	}
	return out
}

func (st *state) apptView(ai int) api.Appointment {
	a := &st.dem.appts[ai]
	o := &st.dem.orders[a.order]
	x := &st.cfg.Catalog[o.exam]
	src := "—"
	if o.source >= 0 {
		src = st.cfg.Sources[o.source].Name
	}
	return api.Appointment{ID: apptID(ai), Order: orderID(a.order), Patient: patientID(o.patient),
		Exam: x.Code, ExamName: x.Name, Priority: o.priority.String(), Source: src, BookedAt: st.cal.ISO(a.bookedAt)}
}

// DayIndex returns the day index of today.
func (w *World) DayIndex() int64 { return w.st.cal.DayIndex(w.eng.Now()) }
