package tui

import (
	"slices"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/session"
)

// V0.2 Slice 2: the staff screen submits hire and terminate through the
// session's Apply, previews the cursor row read-only, and fits 80×24.
func TestStaffScreen(t *testing.T) {
	s, err := session.Open("mini", 1)
	if err != nil {
		t.Fatal(err)
	}
	m := New(s)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.key("m")
	before := s.Digests()
	for range 3 {
		_ = m.render() // each render previews the cursor row
	}
	if s.Digests() != before {
		t.Fatal("rendering the staff screen changed the run")
	}
	screen := ansi.Strip(m.render())
	for _, want := range []string{"STAFF", "XR technologist", "standard $36.00/h, 28-day lead", "If submitted: ✓", "Guaranteed"} {
		if !strings.Contains(screen, want) {
			t.Fatalf("staff screen lacks %q:\n%s", want, screen)
		}
	}
	if lines := strings.Split(screen, "\n"); len(lines) != 24 {
		t.Fatalf("%d lines at 80×24", len(lines))
	}
	m.key("right") // CT technologist
	m.key("enter")
	log := s.ActionLog()
	if len(log) != 1 || log[0].Action.Type != api.ActHire || !log[0].OK || log[0].EmployeeID != "H00001" {
		t.Fatalf("hire from the staff screen: %+v", log)
	}
	// The first termination row: the earliest lawful last day.
	for range hireRows {
		m.key("down")
	}
	m.key("enter")
	if log = s.ActionLog(); len(log) != 2 || log[1].Action.Type != api.ActTerminate || !log[1].OK {
		t.Fatalf("termination from the staff screen: %+v", log[1:])
	}
	if !strings.Contains(ansi.Strip(m.render()), "Done ✓") {
		t.Fatal("no result shown")
	}
}

// Rendering the staff screen (with its live preview) stays within a frame
// budget on a Pi 4 at the established practice.
func TestStaffScreenRenderTime(t *testing.T) {
	if testing.Short() {
		t.Skip("timing")
	}
	s, err := session.Open("established", 1)
	if err != nil {
		t.Fatal(err)
	}
	m := New(s)
	m.Update(tea.WindowSizeMsg{Width: 160, Height: 48})
	m.key("m")
	for range hireRows + 2 {
		m.key("down") // a termination row: its preview reconciles the whole horizon
	}
	t0 := time.Now()
	const n = 5
	for range n {
		_ = m.render()
	}
	per := time.Since(t0) / n
	t.Logf("staff screen render with a termination preview: %s", per)
	if per > 500*time.Millisecond {
		t.Fatalf("render %s", per)
	}
}

// openRosterEditor opens the staff screen's roster editor for employee id.
func openRosterEditor(t *testing.T, m *Model, id string) {
	t.Helper()
	m.key("m")
	k := slices.IndexFunc(m.rosterable(), func(e api.Employee) bool { return e.ID == id })
	if k < 0 {
		t.Fatalf("%s has no roster row", id)
	}
	for range hireRows + len(m.terminable()) + len(m.readers()) + k {
		m.key("down")
	}
	m.key("enter")
	if m.staff.edit == nil || m.staff.edit.emp != id {
		t.Fatal("the roster editor did not open")
	}
}

// V0.2 Slice 3: the roster editor edits a weekly or fortnightly pattern
// for a chosen Monday, previews it read-only, submits set_roster through
// the session's Apply and withdraws a pending change; a refused edit
// shows why. It fits 80×24.
func TestRosterEditor(t *testing.T) {
	s, err := session.Open("mini", 1)
	if err != nil {
		t.Fatal(err)
	}
	m := New(s)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	openRosterEditor(t, m, "E03")
	before := s.Digests()
	for range 3 {
		_ = m.render()
	}
	if s.Digests() != before {
		t.Fatal("rendering the roster editor changed the run")
	}
	screen := ansi.Strip(m.render())
	for _, want := range []string{"ROSTER · Riley (E03)", "effective ‹ Nov 2 ›", "weekly", "Mon", "If submitted: ✓", "nothing changed"} {
		if !strings.Contains(screen, want) {
			t.Fatalf("editor lacks %q:\n%s", want, screen)
		}
	}
	if lines := strings.Split(screen, "\n"); len(lines) != 24 {
		t.Fatalf("%d lines at 80×24", len(lines))
	}
	// Weekly: Fridays off from Nov 2.
	for range 4 {
		m.key("right")
	}
	m.key("o")
	if screen := ansi.Strip(m.render()); !strings.Contains(screen, "rostered 32h 00m (A) · 32h 00m (B)") || !strings.Contains(screen, "Guaranteed") {
		t.Fatalf("after Fri off:\n%s", screen)
	}
	m.key("enter")
	log := s.ActionLog()
	if n := len(log); n != 1 || log[0].Action.Type != api.ActSetRoster || !log[0].OK || log[0].Action.Effective != "2026-11-02" ||
		log[0].Action.Roster.A.Fri != nil || log[0].Action.Roster.B.Fri != nil || log[0].Action.Roster.A.Thu == nil {
		t.Fatalf("set_roster from the editor: %+v", log)
	}
	// Fortnightly, a week later: week B's Monday off too.
	m.key("w")
	m.key(".")
	for range 3 {
		m.key("right") // A Fri → A Sat, A Sun, B Mon
	}
	m.key("o")
	m.key("enter")
	if log = s.ActionLog(); len(log) != 2 || !log[1].OK || log[1].Action.Effective != "2026-11-09" ||
		log[1].Action.Roster.B.Mon != nil || log[1].Action.Roster.A.Mon == nil {
		t.Fatalf("fortnightly: %+v", log[1:])
	}
	// Withdraw it.
	m.key("x")
	m.key("enter")
	if log = s.ActionLog(); len(log) != 3 || !log[2].OK || log[2].Action.Op != api.RosterOpRemove || log[2].Action.Effective != "2026-11-09" {
		t.Fatalf("withdraw: %+v", log[2:])
	}
	// A roster past the contract: previewed ✗, refused with the reason.
	for range 3 {
		m.key("left") // B Mon → A Sun, A Sat, A Fri
	}
	m.key("o")    // A Fri back on: 40 h in week A
	m.key("down") // the end field
	m.key("+")    // 40 h 15 min
	if screen := ansi.Strip(m.render()); !strings.Contains(screen, "If submitted: ✗") {
		t.Fatalf("over-contract preview:\n%s", screen)
	}
	m.key("enter")
	if r := s.ActionLog()[3]; r.OK || r.Code != "invalid_roster" || !strings.Contains(ansi.Strip(m.render()), "Refused ✗") {
		t.Fatalf("over contract: %+v", r)
	}
	m.key("esc")
	if m.staff.edit != nil || !strings.Contains(ansi.Strip(m.render()), "from Nov 2 32h 00m") {
		t.Fatalf("staff screen after the editor:\n%s", ansi.Strip(m.render()))
	}
}

// The roster editor's live preview reconciles the whole horizon; it stays
// within a frame budget on a Pi 4 at the established practice.
func TestRosterEditorRenderTime(t *testing.T) {
	if testing.Short() {
		t.Skip("timing")
	}
	s, err := session.Open("established", 1)
	if err != nil {
		t.Fatal(err)
	}
	m := New(s)
	m.Update(tea.WindowSizeMsg{Width: 160, Height: 48})
	openRosterEditor(t, m, "E04")
	m.key("right")
	m.key("o") // Sam's Tuesdays off: dozens of bookings lose coverage
	t0 := time.Now()
	const n = 5
	for range n {
		_ = m.render()
	}
	per := time.Since(t0) / n
	t.Logf("roster editor render with a displacing preview: %s", per)
	if per > 500*time.Millisecond {
		t.Fatalf("render %s", per)
	}
}

// V0.2 Slice 4: the reading allocation rows show each radiologist's share,
// site reading time and pending changes with the reporting backlog,
// preview the change read-only (salary per 30 days, site reading time,
// withdrawal date), submit set_reading_allocation through the session's
// Apply, withdraw a pending change, and fit 80×24.
func TestReadingAllocationRows(t *testing.T) {
	s, err := session.Open("mini", 1)
	if err != nil {
		t.Fatal(err)
	}
	m := New(s)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.key("m")
	if len(m.readers()) != 2 {
		t.Fatalf("readers: %+v", m.readers())
	}
	for range hireRows + len(m.terminable()) {
		m.key("down") // Dr. Avery's row
	}
	m.key("right") // 60% → 62.5%
	before := s.Digests()
	for range 3 {
		_ = m.render()
	}
	if s.Digests() != before {
		t.Fatal("rendering the allocation preview changed the run")
	}
	screen := ansi.Strip(m.render())
	for _, want := range []string{"Reading allocation", "reporting backlog now", "‹ 62.5% › from ‹ Oct 27 ›", "If submitted: ✓"} {
		if !strings.Contains(screen, want) {
			t.Fatalf("staff screen lacks %q:\n%s", want, screen)
		}
	}
	if lines := strings.Split(screen, "\n"); len(lines) != 24 {
		t.Fatalf("%d lines at 80×24", len(lines))
	}
	m.Update(tea.WindowSizeMsg{Width: 160, Height: 48})
	screen = ansi.Strip(m.render())
	for _, want := range []string{"Salary allocation change · Dr. Avery (E07)", "Site reading capacity", "site share 60% → 62.5%", "withdrawable until Oct 13"} {
		if !strings.Contains(screen, want) {
			t.Fatalf("160×48 staff screen lacks %q:\n%s", want, screen)
		}
	}
	m.key(".") // a day later
	m.key("enter")
	log := s.ActionLog()
	if len(log) != 1 || log[0].Action.Type != api.ActSetReadingAllocation || !log[0].OK || log[0].Action.Effective != "2026-10-28" ||
		log[0].Action.SiteAllocationBP != 6250 {
		t.Fatalf("allocation from the staff screen: %+v", log)
	}
	if !strings.Contains(ansi.Strip(m.render()), "62.5% from Oct 28") {
		t.Fatalf("pending change not shown:\n%s", ansi.Strip(m.render()))
	}
	m.key("x")
	m.key("enter")
	if log = s.ActionLog(); len(log) != 2 || !log[1].OK || log[1].Action.Op != api.RosterOpRemove {
		t.Fatalf("withdraw: %+v", log[1:])
	}
	// Past the bounds: the share stops at 100%.
	for range 20 {
		m.key("right")
	}
	if a, _ := m.staffSelected(); a.SiteAllocationBP != 10000 {
		t.Fatalf("share %d", a.SiteAllocationBP)
	}
}
