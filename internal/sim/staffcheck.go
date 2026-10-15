package sim

import (
	"fmt"
	"slices"

	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Whole-schedule staffing feasibility (V0.2_DESIGN §7.1), read-only.
//
// A future staffing change (a roster edit, a departure) can take qualified
// coverage away from appointments already booked. CheckStaffing rebuilds,
// for a hypothetical change, every materialized day from the change on:
// the day's planned coverage under the new rosters, its slots' staffing
// and supervision, then its booked appointments re-inserted one by one in
// survivor order — priority (urgent first), appointment start,
// confirmation time, appointment — each against only the appointments
// already retained, with the booking rule's own predicates (SlotBookable
// for the whole slot, sharedFree for shared technologists). sharedFree
// searches technologist assignments over the whole conflict cluster with
// backtracking, so no appointment is dropped for a greedy choice. An
// appointment that cannot be retained is a conflict. Nothing is changed:
// no appointment is cancelled, moved or displaced (Slice 1), no random
// draw is made and no log entry is written.

// StaffingChange is a hypothetical change of who works when: replacement
// weekly rosters by employee ID (a weekday's nil entry is a day off; an
// all-nil roster takes the employee off entirely), from day index From,
// which must be after today, on.
type StaffingChange struct {
	From    int64
	Rosters map[string][7]*ShiftTemplate
}

// Conflict reasons: the slot would be left without a qualified
// technologist for part of it, a contrast slot without an on-site
// radiologist, or the slot is staffed but its technologists would be
// needed by retained appointments in other rooms.
const (
	ConflictUnstaffed     = "unstaffed"
	ConflictNoSupervision = "no_supervision"
	ConflictShared        = "shared_capacity"
)

// StaffingConflict is a booked appointment that would lose its coverage.
type StaffingConflict struct {
	Appointment string
	Day         int64
	Room        string
	Start       simtime.Duration
	Priority    Priority
	Reason      string
}

// CheckStaffing reports the booked appointments that a staffing change
// would leave without qualified coverage, day by day in survivor order.
// With no change it reports none: every booking was made against the
// roster in force (V0.1 never takes capacity away).
func (w *World) CheckStaffing(ch StaffingChange) ([]StaffingConflict, error) {
	st := w.st
	if today := st.cal.DayIndex(w.eng.Now()); ch.From <= today {
		return nil, fmt.Errorf("a staffing change starts after today (day %d), not day %d", today, ch.From)
	}
	cfg := *st.cfg
	cfg.Employees = slices.Clone(st.cfg.Employees)
	changed := map[int]bool{}
	for _, id := range sortedKeys(ch.Rosters) {
		i := slices.IndexFunc(cfg.Employees, func(e EmployeeConfig) bool { return e.ID == id })
		if i < 0 {
			return nil, fmt.Errorf("unknown employee %q", id)
		}
		ro := ch.Rosters[id]
		if err := st.checkRoster(&cfg.Employees[i], ro); err != nil {
			return nil, fmt.Errorf("%s: %w", id, err)
		}
		cfg.Employees[i].Roster, changed[i] = ro, true
	}
	var out []StaffingConflict
	for day := ch.From; day < int64(len(st.dem.days)); day++ {
		out = append(out, st.checkDay(&cfg, changed, day)...)
	}
	return out, nil
}

// checkDay re-books day's appointments under cfg's rosters.
func (st *state) checkDay(cfg *Config, changed map[int]bool, day int64) []StaffingConflict {
	ds := st.dem.days[day]
	type cand struct{ room, k, appt int }
	var cands []cand
	for r := range ds.rooms {
		for k := range ds.rooms[r] {
			if a := ds.rooms[r][k].appt; a >= 0 {
				cands = append(cands, cand{r, k, a})
			}
		}
	}
	if len(cands) == 0 {
		return nil
	}
	plan := cfg.Plan(st.cal.Date(st.cal.StartOfDay(day)))
	for i, d := range st.ext[day] {
		if k := plan.ByEmp[i]; k >= 0 && !changed[i] {
			plan.Shifts[k].To += d // approved extensions of unchanged shifts stand
		}
	}
	cov := plan.Coverage()
	tmp := &daySlots{segs: cov.Segments, rooms: make([][]slot, len(ds.rooms))}
	for r := range ds.rooms {
		tmp.rooms[r] = slices.Clone(ds.rooms[r])
		for k := range tmp.rooms[r] {
			tmp.rooms[r][k].appt = -1
			st.coverSlot(&cov, r, &tmp.rooms[r][k])
		}
	}
	slices.SortFunc(cands, func(a, b cand) int {
		pa, pb := st.dem.orders[st.dem.appts[a.appt].order].priority, st.dem.orders[st.dem.appts[b.appt].order].priority
		switch {
		case pa != pb:
			return int(pb) - int(pa)
		case ds.rooms[a.room][a.k].From != ds.rooms[b.room][b.k].From:
			return int(ds.rooms[a.room][a.k].From - ds.rooms[b.room][b.k].From)
		case st.dem.appts[a.appt].bookedAt != st.dem.appts[b.appt].bookedAt:
			return int(st.dem.appts[a.appt].bookedAt - st.dem.appts[b.appt].bookedAt)
		}
		return a.appt - b.appt
	})
	var out []StaffingConflict
	for _, c := range cands {
		s := &tmp.rooms[c.room][c.k]
		why := ""
		switch {
		case s.why == slotUnstaffed:
			why = ConflictUnstaffed
		case s.why == slotNoSupervision:
			why = ConflictNoSupervision
		case !sharedFree(tmp, c.room, s):
			why = ConflictShared
		default:
			s.appt = c.appt
			continue
		}
		out = append(out, StaffingConflict{Appointment: apptID(c.appt), Day: day, Room: st.cfg.Rooms[c.room].ID,
			Start: s.From, Priority: st.dem.orders[st.dem.appts[c.appt].order].priority, Reason: why})
	}
	return out
}

// checkRoster makes the structural checks a hypothetical roster needs to
// be planned: shifts inside the day and the shift cap, breaks and cover
// windows inside the shift, known rooms the employee is qualified for.
// (Full roster validation comes with the set_roster action.)
func (st *state) checkRoster(e *EmployeeConfig, ro [7]*ShiftTemplate) error {
	for wd, t := range ro {
		if t == nil {
			continue
		}
		where := simtime.Weekday(wd).String()
		switch {
		case t.From < 0 || t.From >= t.To || t.To > st.cfg.Policies.ShiftCap(t.From):
			return fmt.Errorf("%s: shift %s is outside the day or the shift cap", where, span(t.Span))
		case t.Weeks < WeekBoth || t.Weeks > WeekB:
			return fmt.Errorf("%s: unknown rota week %d", where, t.Weeks)
		}
		for _, b := range t.Breaks {
			if b.From < t.From || b.To > t.To || b.From >= b.To {
				return fmt.Errorf("%s: break %s outside the shift", where, span(b))
			}
		}
		rooms := []string{t.Assignment}
		switch t.Assignment {
		case AssignDesk, AssignReading, AssignAdmin:
			rooms = nil
		}
		for _, cv := range t.Covers {
			if cv.From < t.From || cv.To > t.To || cv.From >= cv.To {
				return fmt.Errorf("%s: cover %s outside the shift", where, span(cv.Span))
			}
			rooms = append(rooms, cv.Room)
		}
		for _, id := range rooms {
			r := slices.IndexFunc(st.cfg.Rooms, func(rc RoomConfig) bool { return rc.ID == id })
			switch {
			case r < 0:
				return fmt.Errorf("%s: unknown room %q", where, id)
			case e.Role != Technologist || !e.HasSkill(st.cfg.Rooms[r].Modality):
				return fmt.Errorf("%s: not qualified for room %s", where, id)
			}
		}
	}
	return nil
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
