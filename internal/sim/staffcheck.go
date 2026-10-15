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
	appt        int
}

// CheckStaffing reports the booked appointments that a staffing change
// would leave without qualified coverage, day by day in survivor order.
// With no change it reports none: every booking was made against the
// roster in force (and a removal of coverage is reconciled when it is
// committed, displace.go).
func (w *World) CheckStaffing(ch StaffingChange) ([]StaffingConflict, error) {
	st := w.st
	if today := st.cal.DayIndex(w.eng.Now()); ch.From <= today {
		return nil, fmt.Errorf("a staffing change starts after today (day %d), not day %d", today, ch.From)
	}
	over := map[int]*EmployeeConfig{}
	for _, id := range sortedKeys(ch.Rosters) {
		i := st.findEmployee(id)
		if i < 0 {
			return nil, fmt.Errorf("unknown employee %q", id)
		}
		t := *st.emps[i].terms
		ro := ch.Rosters[id]
		if err := st.checkRoster(&t, ro); err != nil {
			return nil, fmt.Errorf("%s: %w", id, err)
		}
		t.Roster, t.Alt = ro, [7]*ShiftTemplate{}
		over[i] = &t
	}
	var out []StaffingConflict
	for day := ch.From; day < int64(len(st.dem.days)); day++ {
		emps := st.dayTerms(day)
		for i, t := range over {
			if emps[i] == nil {
				continue // not employed that day
			}
			if emps[i].Role == Trainee {
				v := *emps[i]
				v.Roster, v.Alt = t.Roster, t.Alt
				emps[i] = &v
				continue
			}
			emps[i] = t
		}
		plan := st.cfg.planOf(st.cal.Date(st.cal.StartOfDay(day)), emps)
		for i, d := range st.ext[day] {
			if k := plan.ByEmp[i]; k >= 0 && over[i] == nil {
				plan.Shifts[k].To += d // approved extensions of unchanged shifts stand
			}
		}
		out = append(out, st.survivors(day, plan)...)
	}
	return out, nil
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
