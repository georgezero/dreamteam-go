package tui

import (
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
