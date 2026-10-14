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

	"github.com/georgezero/dreamteam-go/internal/session"
	"github.com/georgezero/dreamteam-go/internal/simtime"
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
	for _, size := range [][2]int{{80, 24}, {120, 36}, {160, 48}, {60, 16}} {
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		out := m.render()
		lines := strings.Split(out, "\n")
		if len(lines) != size[1] {
			t.Errorf("%v: %d lines", size, len(lines))
		}
		for i, l := range lines {
			if w := ansi.StringWidth(l); w > size[0] {
				t.Errorf("%v: line %d width %d", size, i, w)
			}
		}
	}
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	out := ansi.Strip(m.render())
	for _, want := range []string{"ROOMS", "STAFF", "XR ○ Idle       Riley", "US ○ Idle       Morgan", "Dr. Avery", "Desk ● Jordan"} {
		if !strings.Contains(out, want) {
			t.Errorf("80×24 render lacks %q:\n%s", want, out)
		}
	}
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 10})
	if !strings.Contains(m.render(), "too small") {
		t.Error("no size message at 40×10")
	}
}
