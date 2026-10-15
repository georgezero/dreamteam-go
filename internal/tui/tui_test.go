package tui

import (
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"math/rand/v2"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
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
	press(m, "A")
	if m.autoPause {
		t.Fatal("A did not toggle auto-pause")
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
	fits := func(scr string) {
		for _, size := range [][2]int{{80, 24}, {140, 40}, {60, 16}} {
			m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
			lines := strings.Split(m.render(), "\n")
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
	defer func() {
		m.key("e") // the end-of-day summary (finishes the day, so last)
		fits("e")
	}()
	for _, scr := range []string{"d", "s", "l", "p", "w", "f", "h", "b", "a", "?"} {
		if m.screen == screenHelp {
			m.key("?")
		}
		m.key(scr)
		for _, size := range [][2]int{{80, 24}, {140, 40}, {120, 36}, {160, 48}, {60, 16}} {
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
		"Techs 3/4", "Desk ● Jordan", "PATIENT FLOW today", "On site ", "in exam", "walk-ins", "Unread ", "final today", "Today: busy of staffed", "ALERTS", "Demand today", "next open slot", "EVENT LOG"} {
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
		!strings.Contains(out, "not bookable: no technologist rostered (15 slots)") {
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

func TestFinanceScreen(t *testing.T) {
	m := New(openMini(t, 1))
	m.sess.Advance(45*simtime.Day+5*simtime.Hour, session.Options{}) // past the first payer remittances
	m.Update(tea.WindowSizeMsg{Width: 140, Height: 44})
	dash := ansi.Strip(m.render())
	for _, want := range []string{"Cash $", "Revenue today", "Op. profit today", "A/R $"} {
		if !strings.Contains(dash, want) {
			t.Errorf("dashboard lacks %q:\n%s", want, dash)
		}
	}
	m.key("f")
	out := ansi.Strip(m.render())
	for _, want := range []string{"FINANCE · ", "[overview]", "TODAY", "MONTH TO DATE", "TRAILING 30D", "SINCE DAY 1",
		"Revenue recognized", "collected vs expected", "Net revenue", "BALANCE", "equity = opening capital", "Operating profit", "margin", "Cash collected", "CASH ", "days in A/R", "denial rate",
		"A/R aging", "BY MODALITY", "Cash (actual), last 30 days"} {
		if !strings.Contains(out, want) {
			t.Errorf("finance overview lacks %q:\n%s", want, out)
		}
	}
	if testing.Verbose() {
		t.Log("\n" + out)
	}
	m.key("tab")
	out = ansi.Strip(m.render())
	if !strings.Contains(out, "[claims]") || !strings.Contains(out, "Open claims") || len(m.obs.Finance.Claims.OpenClaims) == 0 {
		t.Errorf("claims view:\n%s", out)
	}
	m.key("tab")
	out = ansi.Strip(m.render())
	if !strings.Contains(out, "[journal]") || !strings.Contains(out, " Dr ") || !strings.Contains(out, " Cr ") {
		t.Errorf("journal view:\n%s", out)
	}
	if testing.Verbose() {
		t.Log("\n" + out)
	}
}

// The established practice opens on the takeover briefing, built from the
// warm-up history; enter goes to the dashboard and h shows the history.
func TestOpeningBriefing(t *testing.T) {
	s, err := session.Open("established", 1)
	if err != nil {
		t.Fatal(err)
	}
	m := New(s)
	m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	if m.screen != screenBriefing {
		t.Fatalf("opened on screen %d", m.screen)
	}
	out := ansi.Strip(m.render())
	for _, want := range []string{"TAKING OVER", "operated for 180 days", "Opening cash $150,000", "owner capital",
		"Last 30 days", "receivables", "Room utilization", "Booked ahead", "Worklist now", "financing shortfall"} {
		if !strings.Contains(out, want) {
			t.Errorf("briefing lacks %q:\n%s", want, out)
		}
	}
	if testing.Verbose() {
		t.Log("\n" + out)
	}
	m.key("h")
	out = ansi.Strip(m.render())
	if !strings.Contains(out, "DAILY HISTORY") || len(m.hist) != historyDays || !m.hist[0].BeforeStart {
		t.Errorf("history: %d days\n%s", len(m.hist), out)
	}
	if testing.Verbose() {
		t.Log("\n" + out)
	}
	m.key("b")
	m.key("enter")
	if m.screen != screenDashboard {
		t.Fatalf("enter left screen %d", m.screen)
	}
	// The mini scenario has no history and opens on the dashboard.
	if m := New(openMini(t, 1)); m.screen != screenDashboard {
		t.Fatalf("mini opened on screen %d", m.screen)
	}
}

// Step 11 review finding 5: the balance-sheet equation is shown in exact
// cents, so it holds as printed (whole-dollar rounding of each term made
// $428,154 = $88,071 + $340,082).
func TestRegress11BalanceEquationExact(t *testing.T) {
	f := api.Finance{BalanceSheet: api.BalanceSheet{CashCents: 15_000_000, ReceivablesCents: 27_815_355, AssetsCents: 42_815_355,
		PayablesCents: 4_893_284, AccruedPayrollCents: 3_913_859, LiabilitiesCents: 8_807_143, OpeningCapitalCents: 56_882_864,
		HistoryEarningsCents: -22_375_815, EarningsSinceStartCents: -498_837, EquityCents: 34_008_212}}
	out := ansi.Strip(strings.Join(overviewLines(f, 140), "\n"))
	for _, want := range []string{"$428,153.55", "$88,071.43", "$340,082.12"} {
		if !strings.Contains(out, want) {
			t.Errorf("balance line lacks %s:\n%s", want, out)
		}
	}
}

// Step 11 review finding 2: history and the cash chart show actual cash
// and name the takeover capital; nothing presents pro forma cash as
// actual.
func TestRegress11CashHistoryLabelled(t *testing.T) {
	s, err := session.Open("established", 1)
	if err != nil {
		t.Fatal(err)
	}
	m := New(s)
	m.Update(tea.WindowSizeMsg{Width: 140, Height: 44})
	m.key("h")
	out := ansi.Strip(m.render())
	if !strings.Contains(out, "ACTUAL CASH") || !strings.Contains(out, "takeover: owner capital +$568,829") {
		t.Errorf("history screen:\n%s", out)
	}
	m.key("f")
	out = ansi.Strip(m.render())
	if !strings.Contains(out, "Cash (actual), last") || !strings.Contains(out, "owner capital +$568,829 at takeover") {
		t.Errorf("finance screen:\n%s", out)
	}
}

// `e` completes today's operations and shows the end-of-day summary; tab
// shows the window since takeover; both fit 80×24 and 120×36.
func TestEndOfDaySummary(t *testing.T) {
	m := New(openMini(t, 3))
	m.key("e")
	if m.screen != screenSummary || m.obs.Practice.OperationalClose == "" || m.running {
		t.Fatalf("after e: screen %d, operational close %q", m.screen, m.obs.Practice.OperationalClose)
	}
	for _, size := range [][2]int{{80, 24}, {120, 36}} {
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		for _, view := range []int{sumDay, sumWindow} {
			m.sumView = view
			m.refresh()
			out := m.render()
			for i, l := range strings.Split(out, "\n") {
				if w := ansi.StringWidth(l); w > size[0] {
					t.Errorf("%v view %d: line %d width %d", size, view, i, w)
				}
			}
			plain := ansi.Strip(out)
			want := []string{"END OF DAY", "Patients", "Rooms", "busy", "Reporting", "Money", "Cash", "Alerts", "Takeover day"}
			if view == sumWindow {
				want = []string{"SINCE TAKEOVER", "Inherited", "Adjusted", "Ledger", "Cash flow", "cash bridge reconciles", "baseline"}
			}
			for _, s := range want {
				if !strings.Contains(plain, s) {
					t.Errorf("%v view %d lacks %q:\n%s", size, view, s, plain)
				}
			}
		}
	}
	// Earlier days page with ←; today with t.
	m.sumView = sumDay
	m.key("left")
	m.refresh()
	if m.sumOffset != 0 {
		t.Fatal("paged before the first day of a scenario without history")
	}
	m.key("enter")
	if m.screen != screenDashboard {
		t.Fatal("enter did not return to the dashboard")
	}
}

// The dashboard's alert panel shows active alerts with severity and the
// first one's explanation, and "nothing needs attention" otherwise.
func TestDashboardAlerts(t *testing.T) {
	m := New(openMini(t, 2))
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	out := ansi.Strip(m.render())
	if !strings.Contains(out, "ALERTS · 0 active") || !strings.Contains(out, "nothing needs attention") {
		t.Fatalf("no empty alert panel:\n%s", out)
	}
	for i := 0; i < 40 && len(m.obs.Alerts.Active) == 0; i++ {
		m.sess.Advance(15*simtime.Minute, session.Options{})
		m.refresh()
	}
	if len(m.obs.Alerts.Active) == 0 {
		t.Fatal("no alert in 10 hours of the mini practice")
	}
	a := m.obs.Alerts.Active[0]
	out = ansi.Strip(m.render())
	if !strings.Contains(out, a.Text[:min(30, len(a.Text))]) || !strings.Contains(out, "why: ") {
		t.Fatalf("alert %+v not shown with its explanation:\n%s", a, out)
	}
}

// The actions screen submits through the session (the same Apply an agent
// uses), previews the cursor row, shows accepted and rejected results, and
// fits 80×24 and 120×36.
func TestActionsScreen(t *testing.T) {
	m := New(openMini(t, 2))
	m.key("a")
	if m.screen != screenActions {
		t.Fatal("a did not open actions")
	}
	staff := m.actionStaff()
	idx := func(name string) int {
		for i, e := range staff {
			if e.Name == name {
				return i
			}
		}
		t.Fatalf("no %s", name)
		return 0
	}
	m.actSel = idx("Morgan")
	m.key("right") // +75m
	out := ansi.Strip(m.render())
	if !strings.Contains(out, "If submitted: ✓ Shift extended: Morgan today until 19:15") {
		t.Fatalf("no preview:\n%s", out)
	}
	m.key("enter")
	if m.actResult == nil || !m.actResult.OK || m.obs.Actions.Accepted != 1 {
		t.Fatalf("result %+v", m.actResult)
	}
	m.actSel = idx("Sam")
	m.key("enter") // Sam already works to close
	if m.actResult.OK || m.actResult.Code != "after_close" || !strings.Contains(m.status, "✗") {
		t.Fatalf("result %+v status %q", m.actResult, m.status)
	}
	m.actSel = len(staff) // walk-in policy row
	for m.selected().Policy != api.WalkInScheduleLater {
		m.key("right")
	}
	m.key("enter")
	if m.obs.Policies.WalkIn != api.WalkInScheduleLater {
		t.Fatalf("policies %+v", m.obs.Policies)
	}
	for _, size := range [][2]int{{80, 24}, {120, 36}} {
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		for i, l := range strings.Split(m.render(), "\n") {
			if w := ansi.StringWidth(l); w > size[0] {
				t.Errorf("%v: line %d width %d", size, i, w)
			}
		}
	}
	// Ahead: next Thursday (Oct 22), approved now.
	m.actSel = 0
	for i := 0; i < 9; i++ {
		m.key("]")
	}
	m.actSel = idx("Morgan")
	m.key("enter")
	if !m.actResult.OK || len(m.obs.Extensions) != 2 || m.obs.Extensions[1].Date != "2026-10-22" {
		t.Fatalf("ahead: %+v, extensions %+v", m.actResult, m.obs.Extensions)
	}
	m.key("d")
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 36})
	out = ansi.Strip(m.render())
	if !strings.Contains(out, "+1h 15m approved") || !strings.Contains(out, "walk-ins: schedule_later") {
		t.Fatalf("dashboard lacks the extension or the policy:\n%s", out)
	}
	if len(m.sess.ActionLog()) != 4 {
		t.Fatalf("action log %+v", m.sess.ActionLog())
	}
}

// ctrl+s saves the session (scenario, seed, actions), the top bar shows
// unsaved changes, ctrl+o reloads the saved file by replaying it, and
// ctrl+v verifies the run by replaying it from scratch.
func TestSessionKeys(t *testing.T) {
	m := New(openMini(t, 2))
	p := filepath.Join(t.TempDir(), "s.json")
	m.SetSavePath(p)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.sess.Advance(4*simtime.Hour, session.Options{})
	m.key("a")
	m.key("enter") // extend the first rostered employee by 60m
	m.refresh()
	if top := ansi.Strip(strings.Split(m.render(), "\n")[0]); !strings.Contains(top, "●") || !strings.Contains(top, "PAUSED · 15 min/s") {
		t.Fatalf("top bar %q", top)
	}
	m.key("ctrl+s")
	if !strings.HasPrefix(m.status, "✓ Saved "+p) || m.sess.Unsaved() {
		t.Fatalf("save: %q", m.status)
	}
	saved := m.sess.Now()
	digest := m.sess.Digest()
	m.sess.Advance(2*simtime.Hour, session.Options{})
	m.refresh()
	m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	if !strings.Contains(ansi.Strip(m.render()), "seed 2 ● unsaved") {
		t.Fatal("no unsaved mark after playing on")
	}
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.key("ctrl+o")
	if !strings.HasPrefix(m.status, "✓ Loaded") || m.sess.Now() != saved || m.sess.Digest() != digest {
		t.Fatalf("reload: %q", m.status)
	}
	m.key("ctrl+v")
	if !strings.HasPrefix(m.status, "✓ Replay verified: 1 batches") {
		t.Fatalf("verify: %q", m.status)
	}
}

// Acceptance: ? shows the keys from any screen and returns to it; the log
// screen filters by category (tab) and severity (!).
func TestHelpAndLogFilters(t *testing.T) {
	m := New(openMini(t, 2))
	m.sess.Advance(10*simtime.Hour, session.Options{})
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.key("w")
	m.key("?")
	out := ansi.Strip(m.render())
	for _, want := range []string{"HELP · keys", "space", "1 2 3 4 5", "ctrl+v", "Urgent studies are always read first"} {
		if !strings.Contains(out, want) {
			t.Errorf("help lacks %q:\n%s", want, out)
		}
	}
	m.key("esc")
	if m.screen != screenWorklist {
		t.Fatalf("esc from help went to screen %d, not the worklist", m.screen)
	}

	m.key("l")
	all := len(m.obs.RecentLog)
	for m.logCat = 0; logCategories[m.logCat] != "patients"; {
		m.key("tab")
	}
	out = ansi.Strip(m.render())
	if !strings.Contains(out, "EVENT LOG · patients · all") {
		t.Fatalf("no category title:\n%s", out)
	}
	n := len(filterLog(m.obs.RecentLog, m.logCat, 0))
	if n == 0 || n >= all {
		t.Fatalf("patients filter keeps %d of %d", n, all)
	}
	for _, e := range filterLog(m.obs.RecentLog, m.logCat, 0) {
		if e.Category != "patients" {
			t.Fatalf("filtered entry %+v", e)
		}
	}
	m.key("tab")
	m.logCat = 0
	m.key("!")
	for _, e := range filterLog(m.obs.RecentLog, 0, m.logSev) {
		if e.Severity == api.Info {
			t.Fatalf("severity filter kept %+v", e)
		}
	}
	m.key("!")
	m.key("!")
	if m.logSev != 0 || len(filterLog(m.obs.RecentLog, 0, 0)) != all {
		t.Fatal("severity filter does not cycle back to all")
	}
}

// Acceptance: NO_COLOR. The program's renderer downsamples through the
// color profile detected from the environment (as Bubble Tea does); with
// NO_COLOR set no color sequence survives, and severity stays readable
// from its marks (⚠, ‼), not only from color.
func TestNoColor(t *testing.T) {
	m := New(openMini(t, 2))
	m.sess.Advance(10*simtime.Hour, session.Options{})
	m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	m.key("l")
	m.key("!") // warnings and alerts only: every line carries a mark
	raw := m.render()
	if !strings.Contains(raw, "\x1b[") {
		t.Fatal("the raw render has no styling at all: the test is vacuous")
	}
	env := []string{"TERM=xterm-256color", "COLORTERM=truecolor", "NO_COLOR=1"}
	if p := colorprofile.Env(env); p > colorprofile.Ascii {
		t.Fatalf("NO_COLOR detected as profile %v", p)
	}
	var buf strings.Builder
	w := &colorprofile.Writer{Forward: &buf, Profile: colorprofile.Env(env)}
	if _, err := w.Write([]byte(raw)); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if regexp.MustCompile(`\x1b\[[0-9;]*(3[0-9]|4[0-9]|9[0-7]|10[0-7])(;[0-9;]*)?m`).MatchString(out) {
		t.Errorf("a color sequence survived NO_COLOR: %q", out)
	}
	if !strings.Contains(out, "⚠") && !strings.Contains(out, "‼") {
		t.Errorf("no severity marks in the filtered log:\n%s", ansi.Strip(out))
	}
}

// Acceptance criterion 8: a run played in the TUI — random speeds, frame
// gaps, pauses, steps, alerts and actions submitted from the actions
// screen — is reproduced exactly by a headless session given only its
// action record, and by replaying its saved record. All five digests.
func TestTUIRunReplaysHeadless(t *testing.T) {
	for trial := uint64(1); trial <= 3; trial++ {
		r := rand.New(rand.NewPCG(trial, 18))
		m := New(openMini(t, 4))
		m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
		m.key("space")
		start := m.sess.Now()
		wall := time.Unix(0, 0)
		submitted := 0
		for frames := 0; frames < 4000 && m.sess.Now() < start.Add(2*simtime.Day); frames++ {
			switch x := r.IntN(100); {
			case x < 8:
				m.key(strconv.Itoa(1 + r.IntN(5)))
			case x < 11:
				m.key("space") // pause …
				m.key("space") // … and resume
			case x < 13:
				m.key("n")
				m.key("space")
			case x < 15 && submitted < 6:
				// From the actions screen: a worklist policy change, or a
				// shift extension on today's first row.
				m.key("a")
				if r.IntN(2) == 0 {
					for range 20 {
						m.key("down")
					}
					m.key("right")
				}
				m.key("enter")
				m.key("d")
				submitted++
			}
			if !m.running {
				m.key("space") // auto-paused on an alert, or by a step
			}
			wall = wall.Add(time.Duration(10+r.IntN(600)) * time.Millisecond)
			m.frame(wall)
		}
		rec, err := m.sess.Record()
		if err != nil {
			t.Fatal(err)
		}
		if len(rec.Batches) == 0 {
			t.Fatalf("trial %d: no actions were submitted", trial)
		}
		head := openMini(t, 4)
		for _, b := range rec.Batches {
			head.AdvanceTo(simtime.Time(b.AtSec))
			head.Apply(b.Actions...)
		}
		head.AdvanceTo(m.sess.Now())
		if got, want := head.Digests(), m.sess.Digests(); got != want {
			t.Fatalf("trial %d: headless %+v, TUI %+v", trial, got, want)
		}
		re, rep, err := session.Replay(rec, session.ReplayOptions{})
		if err != nil {
			t.Fatalf("trial %d: replay: %v %+v", trial, err, rep)
		}
		if re.Digests() != m.sess.Digests() {
			t.Fatalf("trial %d: replayed digests differ", trial)
		}
		t.Logf("trial %d: %d batches, ended %s, run digest %.12s…", trial, len(rec.Batches), m.sess.Calendar().ISO(m.sess.Now()), m.sess.Digests().Run)
	}
}

var updateScreens = flag.Bool("update-screens", false, "rewrite the golden screen snapshots in testdata/screens")

// Golden screen snapshots (ROADMAP step 16): every screen at 80×24,
// 120×36 and 160×48, text only, at a fixed point of a fixed run. A layout
// change shows up as a diff; regenerate deliberately with
//
//	go test ./internal/tui -run ScreenSnapshots -update-screens
func TestScreenSnapshots(t *testing.T) {
	m := New(openMini(t, 2))
	m.sess.Advance(5*simtime.Hour+17*simtime.Minute, session.Options{})
	m.refresh()
	dir := filepath.Join("..", "..", "testdata", "screens")
	for _, scr := range []struct{ key, name string }{
		{"d", "dashboard"}, {"s", "schedule"}, {"l", "log"}, {"p", "patients"}, {"w", "worklist"},
		{"f", "finance"}, {"h", "history"}, {"b", "briefing"}, {"a", "actions"}, {"?", "help"},
	} {
		if m.screen == screenHelp {
			m.key("?")
		}
		m.key(scr.key)
		for _, size := range [][2]int{{80, 24}, {120, 36}, {160, 48}} {
			m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
			got := snapshotText(m.render())
			path := filepath.Join(dir, fmt.Sprintf("%s-%dx%d.txt", scr.name, size[0], size[1]))
			if *updateScreens {
				os.MkdirAll(dir, 0o755)
				if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				continue
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("%v (generate with -update-screens)", err)
			}
			if string(want) != got {
				t.Errorf("%s differs from its snapshot:\n%s", path, got)
			}
		}
	}
}

// snapshotText is the rendered screen as text. The top bar keeps only its
// right side (day, clock, run state): its left side holds the version,
// whose length decides how much of the seed and practice name fits, so a
// version bump alone must not change a snapshot (TestRenderFitsTerminal
// checks the full bar).
func snapshotText(r string) string {
	lines := strings.Split(ansi.Strip(r), "\n")
	if i := regexp.MustCompile(`(Mon|Tue|Wed|Thu|Fri|Sat|Sun) `).FindStringIndex(lines[0]); i != nil {
		lines[0] = strings.Join(strings.Fields(lines[0][i[0]:]), " ")
	}
	return strings.Join(lines, "\n") + "\n"
}

// Max-speed responsiveness (criterion 16): a key is handled between
// frames, so the longest frame (simulation step plus render) bounds the
// delay before space takes effect. Reports p50, p99 and max frame time
// over 20 simulated days of the established practice at Max.
func BenchmarkMaxSpeedFrames(b *testing.B) {
	for range b.N {
		b.StopTimer()
		s, err := session.Open("established", 1)
		if err != nil {
			b.Fatal(err)
		}
		m := New(s)
		m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
		m.key("space")
		m.key("A") // no auto-pause: keep running
		m.key("5")
		end := s.Now().Add(20 * simtime.Day)
		var ds []time.Duration
		wall := time.Unix(0, 0)
		b.StartTimer()
		for s.Now() < end {
			t0 := time.Now()
			wall = wall.Add(frame)
			m.frame(wall)
			_ = m.render()
			ds = append(ds, time.Since(t0))
		}
		b.StopTimer()
		slices.Sort(ds)
		b.ReportMetric(float64(ds[len(ds)/2].Microseconds())/1000, "p50-ms")
		b.ReportMetric(float64(ds[len(ds)*99/100].Microseconds())/1000, "p99-ms")
		b.ReportMetric(float64(ds[len(ds)-1].Microseconds())/1000, "max-ms")
		b.ReportMetric(float64(len(ds)), "frames")
		b.StartTimer()
	}
}
