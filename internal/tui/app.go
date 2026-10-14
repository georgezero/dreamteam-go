// Package tui is the Bubble Tea v2 terminal interface. It is a client of
// session and api only: it never imports internal/sim (enforced by a test),
// and nothing it does affects simulation outcomes, only when they are shown.
package tui

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/session"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// speed is simulated time per real second; zero means Max.
type speed struct {
	label  string
	perSec simtime.Duration
}

var speeds = []speed{
	{"1 min/s", simtime.Minute},
	{"5 min/s", 5 * simtime.Minute},
	{"15 min/s", 15 * simtime.Minute},
	{"1 h/s", simtime.Hour},
	{"Max", 0},
}

const (
	defaultSpeed = 2 // 15 min/s
	frame        = 100 * time.Millisecond
	// maxFrameGap caps one frame's wall time so a suspended terminal does
	// not produce a huge jump on resume.
	maxFrameGap = 500 * time.Millisecond
	// maxEventsPerFrame bounds Max-speed work per frame; the budget is only
	// checked between timestamp transactions.
	maxEventsPerFrame = 20_000
	recentLog         = 200  // entries observed for the dashboard
	fullLog           = 2000 // entries observed for the log screen (the retained log)
	dashboardLog      = 8    // most entries the dashboard shows
)

// screen is the active view.
type screen int

const (
	screenDashboard screen = iota
	screenSchedule
	screenLog
	screenPatients
	screenWorklist
)

// Patient list filters, cycled with tab.
var patientFilters = []string{"all", "on site", "missed / no-show", "left / rescheduled", "walk-ins"}

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(frame, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// Model is the root TUI model.
type Model struct {
	sess      *session.Session
	obs       api.Observation
	running   bool
	speed     int
	autoPause bool
	lastTick  time.Time
	carryMs   int64 // sub-second simulated remainder, in sim-ms
	width     int
	height    int
	status    string
	quitting  bool

	screen screen
	// Schedule screen: day offset from today, selected room, list scroll.
	schedDay    int64
	schedRoom   int
	schedScroll int
	sched       api.DaySchedule
	// Log screen: entries scrolled back from the newest.
	logScroll int
	// Patients screen: day offset, filter, selected row and the patient
	// whose journey is open ("" for the list).
	patDay    int64
	patFilter int
	patSel    int
	patScroll int
	patients  []api.PatientRow
	journeyID string
	journey   api.Journey
	// Worklist screen: selected row and scroll.
	wlSel, wlScroll int
}

// New returns a paused model on s.
func New(s *session.Session) *Model {
	m := &Model{sess: s, speed: defaultSpeed, autoPause: true, width: 80, height: 24}
	m.status = "Paused. Press space to start the day."
	m.refresh()
	return m
}

func (m *Model) refresh() {
	n := recentLog
	if m.screen == screenLog {
		n = fullLog
	}
	m.obs = m.sess.Observe(n)
	switch m.screen {
	case screenSchedule:
		m.sched = m.sess.Schedule(m.schedDay)
	case screenPatients:
		m.patients = filterPatients(m.sess.Patients(m.patDay), m.patFilter)
		m.patSel = max(0, min(m.patSel, len(m.patients)-1))
		if m.journeyID != "" {
			m.journey, _ = m.sess.Journey(m.journeyID)
		}
	}
}

func filterPatients(rows []api.PatientRow, f int) []api.PatientRow {
	if f == 0 {
		return rows
	}
	var out []api.PatientRow
	for _, r := range rows {
		keep := false
		switch patientFilters[f] {
		case "on site":
			keep = r.Status == api.PatientArrived || r.Status == api.PatientRegistering ||
				r.Status == api.PatientWaiting || r.Status == api.PatientInExam
		case "missed / no-show":
			keep = r.Status == api.PatientMissed || r.Status == api.PatientNoShow
		case "left / rescheduled":
			keep = r.Status == api.PatientLeft || r.Status == api.PatientRescheduled
		case "walk-ins":
			keep = r.WalkIn
		}
		if keep {
			out = append(out, r)
		}
	}
	return out
}

// Init starts the frame clock.
func (m *Model) Init() tea.Cmd { return tick() }

// Update handles messages.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tickMsg:
		m.frame(time.Time(msg))
		return m, tick()
	case tea.KeyPressMsg:
		return m, m.key(msg.String())
	}
	return m, nil
}

func (m *Model) key(k string) tea.Cmd {
	if m.screenKey(k) {
		m.refresh()
		return nil
	}
	switch k {
	case "q", "ctrl+c":
		m.quitting = true
		return tea.Quit
	case "space":
		m.running = !m.running
		m.lastTick = time.Time{}
		if m.running {
			m.status = ""
		} else {
			m.status = "Paused."
		}
	case "1", "2", "3", "4", "5":
		m.speed = int(k[0] - '1')
		m.carryMs = 0
		m.status = "Speed " + speeds[m.speed].label
	case "n":
		m.running = false
		if m.sess.StepVisible() == session.Horizon {
			m.status = "Nothing observable in the next 7 days."
		} else {
			m.status = "Stepped to the next event."
		}
		m.refresh()
	case "a":
		m.autoPause = !m.autoPause
		if m.autoPause {
			m.status = "Auto-pause on alerts: on"
		} else {
			m.status = "Auto-pause on alerts: off"
		}
	}
	return nil
}

// screenKey handles navigation keys; it reports whether k was one.
func (m *Model) screenKey(k string) bool {
	if m.screen == screenPatients && m.patientKey(k) {
		return true
	}
	switch k {
	case "d", "esc":
		m.screen = screenDashboard
		return true
	case "p":
		m.screen, m.journeyID = screenPatients, ""
		return true
	case "s":
		m.screen, m.schedDay, m.schedScroll = screenSchedule, 0, 0
		return true
	case "l":
		m.screen, m.logScroll = screenLog, 0
		return true
	case "w":
		m.screen = screenWorklist
		return true
	}
	switch m.screen {
	case screenWorklist:
		switch k {
		case "up", "k":
			m.wlSel = max(0, m.wlSel-1)
		case "down", "j":
			m.wlSel++
		case "enter":
			if items := m.obs.Worklist.Items; m.wlSel < len(items) {
				m.screen, m.journeyID = screenPatients, items[m.wlSel].Patient
			}
		default:
			return false
		}
		return true
	case screenSchedule:
		switch k {
		case "up":
			m.schedRoom = max(0, m.schedRoom-1)
			m.schedScroll = 0
		case "down":
			m.schedRoom = min(max(0, len(m.obs.Rooms)-1), m.schedRoom+1)
			m.schedScroll = 0
		case "left":
			m.schedDay = max(m.obs.Practice.StartOffset, m.schedDay-1)
			m.schedScroll = 0
		case "right":
			m.schedDay = min(m.obs.Practice.BookingHorizonDays, m.schedDay+1)
			m.schedScroll = 0
		case "t":
			m.schedDay, m.schedScroll = 0, 0
		case "j":
			m.schedScroll++
		case "k":
			m.schedScroll = max(0, m.schedScroll-1)
		default:
			return false
		}
		return true
	case screenLog:
		page := max(1, m.height-4)
		switch k {
		case "up", "k":
			m.logScroll++
		case "down", "j":
			m.logScroll = max(0, m.logScroll-1)
		case "pgup":
			m.logScroll += page
		case "pgdown":
			m.logScroll = max(0, m.logScroll-page)
		case "g":
			m.logScroll = len(m.obs.RecentLog)
		case "G":
			m.logScroll = 0
		default:
			return false
		}
		return true
	}
	return false
}

// patientKey handles the patients screen: list navigation, day paging,
// filters, and opening or closing a journey.
func (m *Model) patientKey(k string) bool {
	if m.journeyID != "" {
		if k == "esc" || k == "backspace" {
			m.journeyID = ""
			return true
		}
		return false
	}
	switch k {
	case "up", "k":
		m.patSel = max(0, m.patSel-1)
	case "down", "j":
		m.patSel = min(max(0, len(m.patients)-1), m.patSel+1)
	case "left":
		m.patDay = max(m.obs.Practice.StartOffset, m.patDay-1)
		m.patSel, m.patScroll = 0, 0
	case "right":
		m.patDay = min(m.obs.Practice.BookingHorizonDays, m.patDay+1)
		m.patSel, m.patScroll = 0, 0
	case "t":
		m.patDay, m.patSel, m.patScroll = 0, 0, 0
	case "tab":
		m.patFilter = (m.patFilter + 1) % len(patientFilters)
		m.patSel, m.patScroll = 0, 0
	case "enter":
		if m.patSel < len(m.patients) {
			m.journeyID = m.patients[m.patSel].Patient
		}
	default:
		return false
	}
	return true
}

// frame advances the simulation for one tick of wall time.
func (m *Model) frame(now time.Time) {
	if !m.running {
		m.lastTick = now
		return
	}
	elapsed := frame
	if !m.lastTick.IsZero() {
		elapsed = min(now.Sub(m.lastTick), maxFrameGap)
	}
	m.lastTick = now
	opts := session.Options{StopOnAlert: m.autoPause}
	sp := speeds[m.speed]

	var why session.Reason
	switch {
	case !m.obs.Practice.Open && sp.perSec != 0:
		// Nobody watches an empty building: jump to the next observable
		// entry (normally "Practice opens").
		why = m.sess.StepVisible()
	case sp.perSec == 0:
		opts.EventBudget = maxEventsPerFrame
		why = m.sess.Advance(simtime.Day, opts)
	default:
		simMs := int64(sp.perSec)*elapsed.Milliseconds() + m.carryMs
		m.carryMs = simMs % 1000
		why = m.sess.Advance(simtime.Duration(simMs/1000), opts)
	}
	if why == session.Alert {
		m.running = false
		m.status = "Auto-paused on an alert."
	}
	m.refresh()
}

// View renders the screen.
func (m *Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	return v
}
