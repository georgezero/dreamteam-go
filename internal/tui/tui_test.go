package tui

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/session"
	"github.com/georgezero/dreamteam-go/internal/simtime"
	"github.com/georgezero/dreamteam-go/internal/version"
)

// The TUI may see the world only through session and api.
func TestImportBoundary(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		af, err := parser.ParseFile(token.NewFileSet(), f, src, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range af.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			if p == "github.com/georgezero/dreamteam-go/internal/sim" || strings.HasPrefix(p, "github.com/georgezero/dreamteam-go/internal/sim/") {
				t.Errorf("%s imports %s", f, p)
			}
		}
	}
}

func openMini(t testing.TB, seed int64) *session.Session {
	t.Helper()
	s, err := session.Open("mini", seed)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func press(m *Model, k string) {
	switch k {
	case "space":
		m.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	default:
		m.Update(tea.KeyPressMsg{Code: rune(k[0]), Text: k})
	}
}

func TestKeys(t *testing.T) {
	m := New(openMini(t, 1))
	if m.running {
		t.Fatal("starts running")
	}
	press(m, "space")
	if !m.running {
		t.Fatal("space did not start")
	}
	press(m, "4")
	if m.speed != 3 {
		t.Fatalf("speed %d", m.speed)
	}
	before := m.obs.LogTotal
	press(m, "n")
	if m.running || m.obs.LogTotal <= before {
		t.Fatal("n should pause and step to a new entry")
	}
	press(m, "a")
	if m.autoPause {
		t.Fatal("a did not toggle auto-pause")
	}
	if _, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"}); cmd == nil {
		t.Fatal("q did not quit")
	}
}

// Frame rate and speed must not change outcomes: two TUIs ticking at
// different rates and speeds, then brought to the same time, match a
// headless run.
func TestPacingDoesNotAffectOutcome(t *testing.T) {
	const seed = 11
	runTUI := func(fps int, speedPlan []int) *session.Session {
		s := openMini(t, seed)
		m := New(s)
		press(m, "space")
		wall := time.Unix(1_000_000, 0)
		step := time.Second / time.Duration(fps)
		for i := 0; i < 600; i++ {
			if i%100 == 0 {
				m.key(strconv.Itoa(speedPlan[(i/100)%len(speedPlan)] + 1))
			}
			// Irregular frames: jitter between 0.5× and 1.5× the period.
			wall = wall.Add(step/2 + time.Duration(i%7)*step/6)
			m.Update(tickMsg(wall))
		}
		return s
	}
	a := runTUI(10, []int{2, 3, 4, 0})
	b := runTUI(3, []int{4, 1, 3})
	target := max(a.Now(), b.Now()).Add(simtime.Hour)
	a.Advance(target.Sub(a.Now()), session.Options{})
	b.Advance(target.Sub(b.Now()), session.Options{})

	h := openMini(t, seed)
	h.Advance(target.Sub(h.Now()), session.Options{})
	if a.Digest() != h.Digest() || b.Digest() != h.Digest() {
		t.Fatal("TUI pacing changed the simulation outcome")
	}
}

func TestRenderFitsTerminal(t *testing.T) {
	m := New(openMini(t, 2))
	m.sess.Advance(5*simtime.Hour, session.Options{})
	m.refresh()
	for _, scr := range []string{"d", "s", "l", "p", "w"} {
		m.key(scr)
		for _, size := range [][2]int{{80, 24}, {120, 36}, {160, 48}, {60, 16}} {
			m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
			out := m.render()
			lines := strings.Split(out, "\n")
			if len(lines) != size[1] {
				t.Errorf("%s %v: %d lines", scr, size, len(lines))
			}
			for i, l := range lines {
				if w := ansi.StringWidth(l); w > size[0] {
					t.Errorf("%s %v: line %d width %d", scr, size, i, w)
				}
			}
		}
	}
	m.key("d")
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	out := ansi.Strip(m.render())
	for _, want := range []string{"DREAMTEAM v" + version.Number, "ROOMS", "XR ", "US ", "CT ", "MR ",
		"Techs 3/4", "Desk ● Jordan", "PATIENT FLOW today", "On site ", "in exam", "walk-ins", "Unread ", "final today", "DEMAND ", "next open slot", "EVENT LOG"} {
		if !strings.Contains(out, want) {
			t.Errorf("80×24 dashboard lacks %q:\n%s", want, out)
		}
	}
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	if out := ansi.Strip(m.render()); !strings.Contains(out, "Dr. Avery") {
		t.Errorf("120×36 dashboard lacks the staff list:\n%s", out)
	}
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 10})
	if !strings.Contains(m.render(), "too small") {
		t.Error("no size message at 40×10")
	}
}

// The dashboard log stays compact; the log screen holds the full log.
func TestDashboardLogCompact(t *testing.T) {
	m := New(openMini(t, 2))
	m.sess.Advance(10*simtime.Hour, session.Options{})
	m.Update(tea.WindowSizeMsg{Width: 160, Height: 60})
	m.refresh()
	entries := func(out string) int {
		n := 0
		for _, l := range strings.Split(ansi.Strip(out), "\n") {
			if len(l) > 7 && l[0] == ' ' && l[3] == ':' && strings.Contains(l, "  ") && !strings.Contains(l, "on duty until") {
				n++
			}
		}
		return n
	}
	if n := entries(m.render()); n < 5 || n > dashboardLog {
		t.Fatalf("dashboard shows %d log entries", n)
	}
	m.key("l")
	if n := strings.Count(ansi.Strip(m.render()), "Oct 13 "); n < 40 {
		t.Fatalf("log screen shows only %d entries", n)
	}
}

func TestScheduleNavigation(t *testing.T) {
	m := New(openMini(t, 2))
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m.key("s")
	out := ansi.Strip(m.render())
	for _, want := range []string{"SCHEDULE · Tue Oct 13 (today)", "XR ▶", "A00001", "Chest 2V", "booked"} {
		if !strings.Contains(out, want) {
			t.Errorf("schedule lacks %q:\n%s", want, out)
		}
	}
	m.key("down")
	m.key("right")
	m.key("right")
	m.key("right")
	m.key("right")
	m.key("right")
	m.key("right") // Monday Oct 19: rota week B, no US technologist
	out = ansi.Strip(m.render())
	if !strings.Contains(out, "Mon Oct 19 (in 6 days)") || !strings.Contains(out, "US ▶") ||
		!strings.Contains(out, "not bookable: no technologist rostered (12 slots)") {
		t.Errorf("week B Monday US:\n%s", out)
	}
	m.key("left")
	m.key("t")
	if m.schedDay != 0 {
		t.Fatal("t did not return to today")
	}
	for i := 0; i < 10; i++ {
		m.key("left")
	}
	if m.schedDay != 0 {
		t.Fatalf("paged before the start date: %d", m.schedDay)
	}
	m.key("esc")
	if m.screen != screenDashboard {
		t.Fatal("esc did not return to the dashboard")
	}
}

func TestPatientsScreen(t *testing.T) {
	m := New(openMini(t, 1))
	m.sess.Advance(7*simtime.Hour, session.Options{}) // 13:00
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m.key("p")
	out := ansi.Strip(m.render())
	for _, want := range []string{"PATIENTS · Tue Oct 13 (today) · all", "P00001", "report final", "08:00"} {
		if !strings.Contains(out, want) {
			t.Errorf("patient list lacks %q:\n%s", want, out)
		}
	}
	m.key("tab") // on site
	if len(m.patients) == 0 {
		t.Fatal("nobody on site at 13:00")
	}
	for _, r := range m.patients {
		switch r.Status {
		case api.PatientArrived, api.PatientRegistering, api.PatientWaiting, api.PatientInExam:
		default:
			t.Fatalf("on-site filter shows %s", r.Status)
		}
	}
	m.key("enter")
	if m.journeyID == "" {
		t.Fatal("enter did not open a journey")
	}
	out = ansi.Strip(m.render())
	for _, want := range []string{"PATIENT " + m.journeyID, "Booked", "Arrived", "so far"} {
		if !strings.Contains(out, want) {
			t.Errorf("journey lacks %q:\n%s", want, out)
		}
	}
	m.key("esc")
	if m.journeyID != "" || m.screen != screenPatients {
		t.Fatal("esc did not return to the list")
	}
	m.key("esc")
	if m.screen != screenDashboard {
		t.Fatal("esc did not return to the dashboard")
	}
}

// The practice line shows the closing phase: patients remaining and
// overtime after scheduled close, then when operations actually closed.
func TestClosingStatusLine(t *testing.T) {
	o := api.Observation{}
	o.Practice = api.Practice{Phase: api.PhaseClosing, OnSite: 3, OvertimeMin: 42, OvertimeCostCents: 3800,
		ScheduledClose: "20:00", Hours: "08:00–20:00", HeldOver: []string{"Sam", "Dr. Blake"}}
	got := ansi.Strip(practiceStatus(o))
	if want := "◐ CLOSING — 3 patients remaining · OT +42m ($38) after 20:00 · held over: Sam, Dr. Blake"; got != want {
		t.Errorf("closing:\n got %q\nwant %q", got, want)
	}
	o.Practice.Phase, o.Practice.OperationalClose, o.Practice.HeldOver = api.PhaseClosed, "20:47", nil
	got = ansi.Strip(practiceStatus(o))
	if want := "○ CLOSED · operations closed 20:47 (scheduled 20:00) · OT +42m ($38)"; got != want {
		t.Errorf("closed:\n got %q\nwant %q", got, want)
	}
}

func TestWorklistScreen(t *testing.T) {
	m := New(openMini(t, 1))
	m.sess.Advance(6*simtime.Day+3*simtime.Hour, session.Options{}) // Monday 09:00: weekend backlog
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	m.key("w")
	out := ansi.Strip(m.render())
	for _, want := range []string{"WORKLIST · ", "policy: urgent then FIFO", "SINCE IMAGING", "LATEST REPORTS", "TAT today median", "Backlog by modality"} {
		if !strings.Contains(out, want) {
			t.Errorf("worklist lacks %q:\n%s", want, out)
		}
	}
	if len(m.obs.Worklist.Items) == 0 {
		t.Fatal("empty worklist on Monday morning")
	}
	m.key("enter")
	if m.screen != screenPatients || m.journeyID != m.obs.Worklist.Items[0].Patient {
		t.Fatalf("enter did not open the patient: screen %d journey %q", m.screen, m.journeyID)
	}
	if out := ansi.Strip(m.render()); !strings.Contains(out, "On worklist") {
		t.Errorf("journey lacks the worklist step:\n%s", out)
	}
}
