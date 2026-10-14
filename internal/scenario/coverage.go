package scenario

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Fortnight returns the 14 dates of one full rota cycle, starting on the
// Monday of week A.
func Fortnight(cfg *sim.Config) []simtime.Date {
	a := cfg.RotaWeekA
	monday := a.AddDays(-int64((a.Weekday() + 6) % 7))
	out := make([]simtime.Date, 14)
	for i := range out {
		out[i] = monday.AddDays(int64(i))
	}
	return out
}

// Gaps returns, per room, the open-hours intervals with nobody staffing it,
// plus intervals where the desk is not covered by front-desk staff.
func Gaps(cov *sim.Coverage, nRooms int) (rooms [][]sim.Span, desk []sim.Span) {
	rooms = make([][]sim.Span, nRooms)
	add := func(xs []sim.Span, s sim.Span) []sim.Span {
		if n := len(xs); n > 0 && xs[n-1].To == s.From {
			xs[n-1].To = s.To
			return xs
		}
		return append(xs, s)
	}
	for _, s := range cov.Segments {
		if !s.Open {
			continue
		}
		for r := range s.Rooms {
			if len(s.Rooms[r]) == 0 {
				rooms[r] = add(rooms[r], s.Span)
			}
		}
		if s.Desk != sim.DeskStaffed {
			desk = add(desk, s.Span)
		}
	}
	return rooms, desk
}

// Report is the human-readable coverage report (PLAN §8.3): staffed versus
// open hours per room for every day of the fortnight rota, with the gaps.
// It is advice, not validation: gaps are the intended first problem.
func Report(cfg *sim.Config) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Coverage · %s · %s\n", cfg.ID, cfg.Name)
	b.WriteString("Staffed / open hours per room. Desk = front desk (+ops) hours. Sup = contrast supervision (radiologist on site).\n")
	var totStaffed, totOpen simtime.Duration
	for i, d := range Fortnight(cfg) {
		if i%7 == 0 {
			week := "A"
			if i >= 7 {
				week = "B"
			}
			fmt.Fprintf(&b, "\nWeek %s      %-12s", week, "Open")
			for _, r := range cfg.Rooms {
				fmt.Fprintf(&b, " %-7s", r.Modality)
			}
			b.WriteString(" Desk     Sup\n")
		}
		p := cfg.Plan(d)
		if p.Hours == nil {
			fmt.Fprintf(&b, "  %s        closed\n", d.Weekday())
			continue
		}
		cov := p.Coverage()
		staffed, open := cov.StaffedSeconds()
		var desk, ops, sup simtime.Duration
		for _, s := range cov.Segments {
			if !s.Open {
				continue
			}
			switch s.Desk {
			case sim.DeskStaffed:
				desk += s.To - s.From
			case sim.DeskOps:
				ops += s.To - s.From
			}
			if s.Supervision {
				sup += s.To - s.From
			}
		}
		fmt.Fprintf(&b, "  %s        %-12s", d.Weekday(), simtime.FormatClock(p.Hours.From)+"–"+simtime.FormatClock(p.Hours.To))
		for r := range cfg.Rooms {
			fmt.Fprintf(&b, " %-7s", hours(staffed[r])+"/"+hours(open))
			totStaffed += staffed[r]
			totOpen += open
		}
		deskCol := hours(desk)
		if ops > 0 {
			deskCol += "+" + hours(ops)
		}
		fmt.Fprintf(&b, " %-8s %s\n", deskCol, hours(sup))

		roomGaps, deskGaps := Gaps(&cov, len(cfg.Rooms))
		var notes []string
		for r, g := range roomGaps {
			if len(g) > 0 {
				notes = append(notes, cfg.Rooms[r].Modality+" "+spans(g))
			}
		}
		if len(deskGaps) > 0 {
			notes = append(notes, "no front desk "+spans(deskGaps))
		}
		if len(notes) > 0 {
			b.WriteString("               unstaffed: " + strings.Join(notes, "; ") + "\n")
		}
		var slotNotes []string
		for r := range cfg.Rooms {
			var total, unstaffed, unsupervised int
			for _, t := range cfg.Templates[r][d.Weekday()] {
				total++
				if ok, why := cov.SlotBookable(r, t.Span, cfg.Categories[t.Category].Contrast); !ok {
					if why == "no_supervision" {
						unsupervised++
					} else {
						unstaffed++
					}
				}
			}
			if unstaffed+unsupervised == 0 {
				continue
			}
			n := fmt.Sprintf("%s %d of %d", cfg.Rooms[r].Modality, unstaffed+unsupervised, total)
			if unsupervised > 0 {
				n += fmt.Sprintf(" (%d contrast, no radiologist on site)", unsupervised)
			}
			slotNotes = append(slotNotes, n)
		}
		if len(slotNotes) > 0 {
			b.WriteString("               template slots not bookable: " + strings.Join(slotNotes, "; ") + "\n")
		}
	}
	if totOpen > 0 {
		fmt.Fprintf(&b, "\nFortnight: %s of %s room-hours staffed (%d%%).\n",
			hours(totStaffed), hours(totOpen), int64(totStaffed)*100/int64(totOpen))
	}
	return b.String()
}

// hours formats a duration as decimal hours: "8.5", "12", "0.25".
func hours(d simtime.Duration) string {
	h := math.Round(float64(d)/float64(simtime.Hour)*100) / 100
	return strconv.FormatFloat(h, 'f', -1, 64)
}

func spans(xs []sim.Span) string {
	var parts []string
	for _, s := range xs {
		parts = append(parts, simtime.FormatClock(s.From)+"–"+simtime.FormatClock(s.To))
	}
	return strings.Join(parts, ", ")
}
