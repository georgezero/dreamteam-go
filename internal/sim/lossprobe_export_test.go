package sim

import "github.com/georgezero/dreamteam-go/internal/simtime"

// Booking-loss diagnosis (V0.2 Step 0B). When an order is lost, the probe
// examines every template slot that could have taken it — same category,
// any room — on each day of the patient's acceptance window: from the
// earliest wanted day (not before today) to earliest + tolerance, capped at
// the booking horizon. FirstFit has already rejected all of them; the
// diagnosis records why, without changing anything.
//
// Each such slot has exactly one blocking reason, the first check FirstFit
// fails, in its own order: booked (an appointment holds it), unstaffed (no
// qualified technologist rostered for the whole slot), no supervision (a
// contrast slot without an on-site radiologist), notice (too soon today),
// shared (free and staffed, but its technologists are needed by
// appointments already booked in another room: §5.3 shared capacity).
//
// One loss can have several causes; Primary picks one by a fixed
// precedence, the cause whose removal alone would have booked the patient
// without moving anyone, most specific first:
//
//  1. shared_tech      a free, staffed slot blocked only by shared staff
//  2. not_rostered     a free slot with no qualified technologist rostered
//  3. no_supervision   a free contrast slot without supervision
//  4. notice           a free bookable slot inside the booking notice
//  5. template_category every compatible slot booked, while slots of another
//     category in the modality's rooms were free and bookable (the template's
//     category mix binds, not room time)
//  6. category_full    every compatible slot booked and no other free slot:
//     the template is full
//  7. no_template_slot no compatible slot on any day of the window
//
// The patient's request bounds the window; the loss is always "the first
// slot came after the patient's tolerance", so BeyondDays (how far after)
// and ToleranceDays describe the request side. NoSlotInHorizon marks losses
// with no compatible slot at all within the booking horizon.

// LossDiagnosis is one lost order's booking-failure record.
type LossDiagnosis struct {
	Order           string
	Received        simtime.Time
	WalkIn          bool
	Modality        string
	Category        string
	Priority        string
	ToleranceDays   int64
	WindowDays      int   // days examined
	NoSlotInHorizon bool  // FirstFit found no compatible slot in the horizon
	BeyondDays      int64 // first compatible slot's day − last acceptable day; −1 if none
	Slots           int   // compatible slots in the window
	Booked          int
	Unstaffed       int
	NoSupervision   int
	Notice          int
	SharedConflict  int
	SharedAt        []simtime.Time // start of each shared-conflict slot (Step 0C)
	Unexplained     int            // free and bookable: must stay 0 (FirstFit would have booked it)
	OtherFree       int            // free bookable slots of other categories in the modality's rooms
	OtherFreeMin    int64
	Primary         string
}

// SetLossProbe calls f with the diagnosis of every lost order from now on.
func (w *World) SetLossProbe(f func(LossDiagnosis)) {
	st := w.st
	st.lossProbe = func(now simtime.Time, oi int, earliest int64, hasAny bool, anyDay int64) {
		f(st.diagnoseLoss(now, oi, earliest, hasAny, anyDay))
	}
}

func (st *state) diagnoseLoss(now simtime.Time, oi int, earliest int64, hasAny bool, anyDay int64) LossDiagnosis {
	o := &st.dem.orders[oi]
	ex := st.cfg.Catalog[o.exam]
	d := LossDiagnosis{Order: orderID(oi), Received: o.received, WalkIn: o.walkIn, Modality: ex.Modality,
		Category: st.cfg.Categories[ex.Category].Name, Priority: o.priority.String(), ToleranceDays: o.tolerance,
		NoSlotInHorizon: !hasAny, BeyondDays: -1}
	if hasAny {
		d.BeyondDays = anyDay - (earliest + o.tolerance)
	}
	notBefore := now.Add(st.cfg.Policies.BookingNotice)
	last := min(earliest+o.tolerance, st.day+st.cfg.Policies.BookingHorizonDays)
	for day := max(earliest, st.day); day <= last; day++ {
		d.WindowDays++
		ds := st.dem.days[day]
		for r := range ds.rooms {
			if st.cfg.Rooms[r].Modality != ex.Modality {
				continue
			}
			for i := range ds.rooms[r] {
				s := &ds.rooms[r][i]
				free := s.appt < 0 && s.why == slotOK && st.cal.At(day, s.From) >= notBefore
				if s.Category != ex.Category {
					if free && sharedFree(ds, r, s) {
						d.OtherFree++
						d.OtherFreeMin += int64((s.To - s.From) / simtime.Minute)
					}
					continue
				}
				d.Slots++
				switch {
				case s.appt >= 0:
					d.Booked++
				case s.why == slotUnstaffed:
					d.Unstaffed++
				case s.why == slotNoSupervision:
					d.NoSupervision++
				case st.cal.At(day, s.From) < notBefore:
					d.Notice++
				case !sharedFree(ds, r, s):
					d.SharedConflict++
					d.SharedAt = append(d.SharedAt, st.cal.At(day, s.From))
				default:
					d.Unexplained++
				}
			}
		}
	}
	switch {
	case d.SharedConflict > 0:
		d.Primary = "shared_tech"
	case d.Unstaffed > 0:
		d.Primary = "not_rostered"
	case d.NoSupervision > 0:
		d.Primary = "no_supervision"
	case d.Notice > 0:
		d.Primary = "notice"
	case d.Slots > 0 && d.OtherFree > 0:
		d.Primary = "template_category"
	case d.Slots > 0:
		d.Primary = "category_full"
	default:
		d.Primary = "no_template_slot"
	}
	if d.Unexplained > 0 {
		d.Primary = "unexplained"
	}
	return d
}
