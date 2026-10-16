package tui

import (
	"fmt"
	"strings"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// The staff screen (V0.2 Slices 2–3): the staff registry with each
// employee's lifecycle, contract and pending roster changes, a hire form,
// termination rows and roster rows (enter opens the roster editor,
// roster.go). Like the actions screen it only submits through the
// session's Apply; the row under the cursor is previewed with
// PreviewStaffing (read-only: it changes nothing, allocates no ID and
// reveals no hidden patient outcome).

// offerChoice is one recruitable offer (the fixed V0.2 table, as the
// simulation publishes it in its validation messages; mirrored here).
type offerChoice struct {
	role                  string
	skills                []string
	std, prem             int64 // cents an hour
	stdLead, premLead     int64 // days
	label, modality, room string
}

var offerChoices = []offerChoice{
	{role: "technologist", skills: []string{"XR"}, std: 3600, prem: 4000, stdLead: 28, premLead: 21, label: "XR technologist", modality: "XR"},
	{role: "technologist", skills: []string{"CT"}, std: 4000, prem: 4400, stdLead: 28, premLead: 21, label: "CT technologist", modality: "CT"},
	{role: "technologist", skills: []string{"XR", "CT"}, std: 3800, prem: 4200, stdLead: 28, premLead: 21, label: "XR+CT technologist", modality: "CT"},
	{role: "technologist", skills: []string{"US"}, std: 4400, prem: 4800, stdLead: 28, premLead: 21, label: "US technologist", modality: "US"},
	{role: "technologist", skills: []string{"MR"}, std: 4500, prem: 4900, stdLead: 28, premLead: 21, label: "MR technologist", modality: "MR"},
	{role: "front_desk", skills: []string{"registration"}, std: 2100, prem: 2400, stdLead: 14, premLead: 7, label: "front desk", modality: ""},
}

// rosterChoice is a weekly pattern offered for a hire, the same in both
// rota weeks.
type rosterChoice struct {
	label        string
	days         []int // 0 = Mon … 6 = Sun
	start, end   string
	brk          [2]string // empty: no break
	assignRoom   bool      // assigned to the offer's room (else the desk)
	netMinWeekly int64
}

var rosterChoices = []rosterChoice{
	{label: "Mon–Fri 07:00–15:30 (break 11:00)", days: []int{0, 1, 2, 3, 4}, start: "07:00", end: "15:30", brk: [2]string{"11:00", "11:30"}, assignRoom: true, netMinWeekly: 2400},
	{label: "Mon–Fri 11:30–20:00 (break 15:30)", days: []int{0, 1, 2, 3, 4}, start: "11:30", end: "20:00", brk: [2]string{"15:30", "16:00"}, assignRoom: true, netMinWeekly: 2400},
	{label: "Tue–Sat 11:30–20:00 (break 15:30)", days: []int{1, 2, 3, 4, 5}, start: "11:30", end: "20:00", brk: [2]string{"15:30", "16:00"}, assignRoom: true, netMinWeekly: 2400},
	{label: "Mon–Fri 16:00–20:00 evenings", days: []int{0, 1, 2, 3, 4}, start: "16:00", end: "20:00", assignRoom: true, netMinWeekly: 1200},
	{label: "Mon–Fri 07:45–16:15 desk (break 12:00)", days: []int{0, 1, 2, 3, 4}, start: "07:45", end: "16:15", brk: [2]string{"12:00", "12:30"}, netMinWeekly: 2400},
}

// Hire form rows.
const (
	hireOffer = iota
	hireTier
	hireRoster
	hireContract
	hireStart
	hireRows
)

// staffForm is the hire form and termination choices.
type staffForm struct {
	sel               int
	offer, tier, rost int
	contractMin       int64 // 0: the roster's
	startShift        int64 // days after the earliest start
	lastShift         map[string]int64
	result            *api.ActionResult
	edit              *rosterEditor // the open roster editor, if any
}

// rosterFor lists the patterns that suit an offer (desk for front desk,
// room shifts for technologists).
func rosterFor(o offerChoice) []int {
	var out []int
	for i, r := range rosterChoices {
		if r.assignRoom == (o.role == "technologist") {
			out = append(out, i)
		}
	}
	return out
}

func (m *Model) hireOfferRoom(o offerChoice) string {
	for _, r := range m.obs.Rooms {
		if r.Modality == o.modality {
			return r.ID
		}
	}
	return ""
}

// todayDate is the observation's date.
func (m *Model) todayDate() simtime.Date {
	d, _ := simtime.ParseDate(m.obs.Date)
	return d
}

// hireAction is the hire the form describes.
func (m *Model) hireAction() api.Action {
	f := &m.staff
	o := offerChoices[f.offer]
	opts := rosterFor(o)
	rc := rosterChoices[opts[f.rost%len(opts)]]
	rate, lead := o.std, o.stdLead
	if f.tier == 1 {
		rate, lead = o.prem, o.premLead
	}
	contract := f.contractMin
	if contract == 0 {
		contract = rc.netMinWeekly
	}
	shift := func() *api.Shift {
		s := &api.Shift{Start: rc.start, End: rc.end, Breaks: [][2]string{}, Assignment: "desk", Covers: []api.Cover{}}
		if rc.brk[0] != "" {
			s.Breaks = [][2]string{rc.brk}
		}
		if rc.assignRoom {
			s.Assignment = m.hireOfferRoom(o)
		}
		return s
	}
	var wk api.Week
	days := []**api.Shift{&wk.Mon, &wk.Tue, &wk.Wed, &wk.Thu, &wk.Fri, &wk.Sat, &wk.Sun}
	for _, d := range rc.days {
		*days[d] = shift()
	}
	start := m.todayDate().AddDays(lead + f.startShift).String()
	return api.Action{Type: api.ActHire, RequestID: fmt.Sprintf("tui-%s-%s-%d", m.obs.Date, strings.ReplaceAll(m.obs.Clock, ":", ""), len(m.obs.Staff)),
		Role: o.role, Skills: o.skills, RateCents: rate, ContractedWeeklyMinutes: contract, Roster: &api.Roster{A: wk, B: wk}, StartDate: start}
}

// terminable lists the employees who can be given notice: hourly
// technologists and front desk staff not yet leaving.
func (m *Model) terminable() []api.Employee {
	var out []api.Employee
	for _, e := range m.obs.Staff {
		if (e.Role == "technologist" || e.Role == "front_desk") && e.HourlyCents > 0 && e.Leaving == nil &&
			e.Employment != api.EmploymentTerminated {
			out = append(out, e)
		}
	}
	return out
}

// terminateAction is the termination row i describes: the earliest lawful
// last day (14 days after today and the start date) plus the chosen shift.
func (m *Model) terminateAction(e api.Employee) api.Action {
	earliest := m.todayDate().AddDays(14)
	if e.Hired != nil {
		if s, err := simtime.ParseDate(e.Hired.StartDate); err == nil && s.AddDays(14).Ordinal() > earliest.Ordinal() {
			earliest = s.AddDays(14)
		}
	}
	return api.Action{Type: api.ActTerminate, Employee: e.ID, LastDay: earliest.AddDays(m.staff.lastShift[e.ID]).String()}
}

// staffSelected is the cursor row's action (a roster row has none: enter
// opens the editor).
func (m *Model) staffSelected() (api.Action, bool) {
	f := &m.staff
	if f.sel < hireRows {
		return m.hireAction(), true
	}
	if k := f.sel - hireRows; k < len(m.terminable()) {
		return m.terminateAction(m.terminable()[k]), true
	}
	return api.Action{}, false
}

// rosterRow is the employee of the cursor's roster row, if it is one.
func (m *Model) rosterRow() (api.Employee, bool) {
	k := m.staff.sel - hireRows - len(m.terminable())
	if r := m.rosterable(); k >= 0 && k < len(r) {
		return r[k], true
	}
	return api.Employee{}, false
}

// staffKey handles the staff screen's keys; it reports whether k was one.
func (m *Model) staffKey(k string) bool {
	f := &m.staff
	if f.edit != nil {
		return m.rosterKey(k)
	}
	rows := hireRows + len(m.terminable()) + len(m.rosterable())
	switch k {
	case "up", "k":
		f.sel = max(0, f.sel-1)
	case "down", "j":
		f.sel = min(rows-1, f.sel+1)
	case "left", "right":
		step := 1
		if k == "left" {
			step = -1
		}
		switch f.sel {
		case hireOffer:
			f.offer = (f.offer + step + len(offerChoices)) % len(offerChoices)
			f.rost, f.contractMin = 0, 0
		case hireTier:
			f.tier = 1 - f.tier
		case hireRoster:
			n := len(rosterFor(offerChoices[f.offer]))
			f.rost, f.contractMin = (f.rost+step+n)%n, 0
		case hireContract:
			cur := f.contractMin
			if cur == 0 {
				cur = m.hireAction().ContractedWeeklyMinutes
			}
			f.contractMin = min(2400, max(30, cur+int64(step*30)))
		case hireStart:
			f.startShift = max(0, f.startShift+int64(step))
		default:
			if e := m.terminable(); f.sel-hireRows < len(e) {
				id := e[f.sel-hireRows].ID
				f.lastShift[id] = max(0, f.lastShift[id]+int64(step))
			}
		}
	case "enter":
		if e, ok := m.rosterRow(); ok {
			m.openRoster(e.ID)
			f.result = nil
			return true
		}
		if a, ok := m.staffSelected(); ok {
			r := m.sess.Apply(a)[0]
			f.result = &r
			m.status = resultText(r)
		}
	default:
		return false
	}
	return true
}

// staffBody renders the staff screen.
func (m *Model) staffBody(w, n int) []string {
	o := m.obs
	f := &m.staff
	if f.edit != nil {
		return m.rosterBody(w, n)
	}
	lines := []string{titleStyle.Render(section(fmt.Sprintf("STAFF · %s %s %s", o.Weekday, shortDate(o.Date), o.Clock), w))}
	curLine := 0
	cursor := func(i int) string {
		if i == f.sel {
			curLine = len(lines) // the line about to be appended
			return titleStyle.Render(">")
		}
		return " "
	}
	a := m.hireAction()
	oc := offerChoices[f.offer]
	tier := fmt.Sprintf("standard $%d.%02d/h, %d-day lead", oc.std/100, oc.std%100, oc.stdLead)
	if f.tier == 1 {
		tier = fmt.Sprintf("premium $%d.%02d/h, %d-day lead", oc.prem/100, oc.prem%100, oc.premLead)
	}
	opts := rosterFor(oc)
	lines = append(lines, "", titleStyle.Render(" Hire")+dimStyle.Render(" — recruiting fee now; paid onboarding (no patient capacity) until training is complete"))
	for i, row := range [][2]string{
		{"offer   ", oc.label},
		{"pay     ", tier},
		{"roster  ", rosterChoices[opts[f.rost%len(opts)]].label + " (both rota weeks)"},
		{"contract", minutes(a.ContractedWeeklyMinutes) + " a week (the roster may not exceed it)"},
		{"start   ", a.StartDate},
	} {
		lines = append(lines, fmt.Sprintf(" %s %s ‹ %s ›", cursor(i), row[0], row[1]))
	}
	lines = append(lines, "", titleStyle.Render(" Terminate")+dimStyle.Render(" — notice of at least 14 days with contract pay; two weeks' severance; bookings that lose coverage are rebooked or lost"))
	for k, e := range m.terminable() {
		t := m.terminateAction(e)
		lines = append(lines, fmt.Sprintf(" %s %-7s %-12s last day ‹ %s ›  severance %s", cursor(hireRows+k), trim(e.ID, 7), trim(e.Name, 12),
			t.LastDay, dollars(e.SeveranceCents)))
	}
	lines = append(lines, "")
	if r := f.result; r != nil {
		mark := openStyle.Render(" Done ✓ ")
		if !r.OK {
			mark = alertStyle.Render(" Refused ✗ ")
		}
		lines = append(lines, wrapAfter(mark, r.Message, w, 2)...)
	}
	if sel, ok := m.staffSelected(); ok {
		lines = append(lines, m.previewLines(m.sess.PreviewStaffing(sel), w, n-len(lines))...)
	}
	lines = append(lines, "", titleStyle.Render(" Rosters")+dimStyle.Render(" — enter edits: dated from a Monday ≥ 14 days ahead; the contract is paid whatever is rostered"))
	base := hireRows + len(m.terminable())
	for k, e := range m.rosterable() {
		pend := ""
		for _, c := range e.RosterChanges {
			pend += fmt.Sprintf(" · from %s %s", shortDate(c.Effective), minutes(c.NetMinA))
		}
		today := e.Shift
		if today == "" {
			today = "off today"
		}
		lines = append(lines, fmt.Sprintf(" %s %-7s %-12s contract %-7s %s%s", cursor(base+k), trim(e.ID, 7), trim(e.Name, 12),
			minutes(e.ContractMin), today, dimStyle.Render(pend)))
	}
	lines = append(lines, "")
	lines = append(lines, m.staffTable()...)
	// Keep the cursor row in view on a short terminal: scroll the form.
	if cut := curLine - (n - 4); cut > 0 {
		lines = append(lines[:2:2], lines[2+cut:]...)
	}
	if len(lines) > n {
		lines = lines[:n]
	}
	return lines
}

// previewLines renders a staffing preview: the verdict, guaranteed and
// forecast money apart, hires' training, roster changes, coverage
// differences and the bookings that would lose coverage.
func (m *Model) previewLines(p api.StaffingPreview, w, n int) []string {
	var lines []string
	{
		mark := dimStyle.Render(" If submitted: ") + openStyle.Render("✓ ")
		if !p.OK {
			mark = dimStyle.Render(" If submitted: ") + alertStyle.Render("✗ ")
		}
		msg := ""
		for _, r := range p.Results {
			if !r.OK || msg == "" {
				msg = r.Message
			}
			if !r.OK && r.Code != "batch_rejected" {
				break
			}
		}
		lines = append(lines, wrapAfter(mark, msg, w, 3)...)
		if p.OK {
			lines = append(lines, dimStyle.Render(" Guaranteed (contractual):"))
			for _, g := range p.Guaranteed {
				lines = append(lines, fmt.Sprintf("   %-58s %10s  %s", trim(g.Label, 58), dollars(g.Cents), dimStyle.Render(g.When)))
			}
			if len(p.Forecast) > 0 {
				lines = append(lines, dimStyle.Render(" Forecast (estimates, not obligations):"))
				for _, g := range p.Forecast {
					amt := dollars(g.Cents)
					if g.Cents == 0 {
						amt = ""
					}
					lines = append(lines, fmt.Sprintf("   %-58s %10s  %s", trim(g.Label, 58), amt, dimStyle.Render(trim(g.Basis, max(10, w-76)))))
				}
			}
			for _, h := range p.Hires {
				var opens []string
				for _, c := range h.OpensSlots {
					opens = append(opens, fmt.Sprintf("%d %s", c.Count, c.Modality))
				}
				lines = append(lines, dimStyle.Render(fmt.Sprintf(" Training %s from %s; independent from %s; opens %s slots in the booking horizon.",
					minutes(h.TrainingMin), shortDate(h.StartDate), shortDate(h.ProductiveDate), strings.Join(append(opens, "no other")[:max(1, len(opens))], ", "))))
			}
			for _, r := range p.Rosters {
				line := fmt.Sprintf(" %s from %s: rostered %s/%s → %s/%s (A/B) · contract %s", r.Employee, shortDate(r.Effective),
					minutes(r.BeforeMinA), minutes(r.BeforeMinB), minutes(r.AfterMinA), minutes(r.AfterMinB), minutes(r.ContractMin))
				if r.ProductiveBefore != "" {
					line += fmt.Sprintf(" · independent %s → %s", shortDate(r.ProductiveBefore), shortDate(r.ProductiveAfter))
				}
				for _, x := range r.Extensions {
					line += fmt.Sprintf(" · extension %s +%dm kept", shortDate(x.Date), x.Minutes)
				}
				lines = append(lines, dimStyle.Render(line))
			}
			if len(p.Rosters) > 0 {
				count := func(cs []api.ModalityCount) string {
					var out []string
					for _, c := range cs {
						out = append(out, fmt.Sprintf("%d %s", c.Count, c.Modality))
					}
					if len(out) == 0 {
						return "none"
					}
					return strings.Join(out, ", ")
				}
				lines = append(lines, dimStyle.Render(fmt.Sprintf(" Coverage in the booking horizon: slots opened %s · closed %s", count(p.SlotsOpened), count(p.SlotsClosed))))
			}
			if len(p.Displacements) > 0 {
				lines = append(lines, warnStyle.Render(fmt.Sprintf(" %d booked appointments lose coverage:", len(p.Displacements))))
				for _, d := range p.Displacements {
					if len(lines) >= n-2 {
						break
					}
					repl := "no feasible slot in the horizon"
					if d.ProvisionalDate != "" {
						repl = "earliest feasible " + shortDate(d.ProvisionalDate) + " " + d.ProvisionalStart
					}
					lines = append(lines, dimStyle.Render(fmt.Sprintf("   %s %s %s %s %s · %s · %s", d.Appointment, shortDate(d.Date), d.Start, d.Room,
						trim(d.Exam, 18), d.Reason, repl)))
				}
			}
		}
	}
	return lines
}

// staffTable lists every employee with stage, contract and dates.
func (m *Model) staffTable() []string {
	o := m.obs
	lines := []string{titleStyle.Render(" Staff"), dimStyle.Render("   id      name         role          stage        contract  rate     dates")}
	for _, e := range o.Staff {
		contract, rate := "—", "salary"
		if e.ContractMin > 0 {
			contract = minutes(e.ContractMin)
		}
		if e.HourlyCents > 0 {
			rate = fmt.Sprintf("$%d.%02d/h", e.HourlyCents/100, e.HourlyCents%100)
		}
		dates := ""
		if h := e.Hired; h != nil {
			dates = fmt.Sprintf("start %s · independent %s · trained %s/%s", shortDate(h.StartDate), shortDate(h.ProductiveDate),
				minutes(h.TrainedMin), minutes(h.TrainingMin))
		}
		if l := e.Leaving; l != nil {
			paid := "due " + shortDate(l.SeverancePayday)
			if l.SeverancePaid {
				paid = "paid"
			}
			dates = strings.TrimPrefix(dates+" · ", " · ") + fmt.Sprintf("last day %s · severance %s %s", shortDate(l.LastDay), dollars(l.SeveranceCents), paid)
		}
		for _, c := range e.RosterChanges {
			dates = strings.TrimPrefix(dates+" · ", " · ") + fmt.Sprintf("roster from %s", shortDate(c.Effective))
		}
		lines = append(lines, fmt.Sprintf("   %-7s %-12s %-13s %-12s %-9s %-8s %s", trim(e.ID, 7), trim(e.Name, 12),
			trim(strings.ReplaceAll(e.Role, "_", " "), 13), e.Employment, contract, rate, dates))
	}
	return lines
}

// dollars formats cents as "$1,234".
func dollars(c int64) string {
	neg := c < 0
	if neg {
		c = -c
	}
	s := fmt.Sprint((c + 50) / 100)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	if neg {
		return "−$" + s
	}
	return "$" + s
}
