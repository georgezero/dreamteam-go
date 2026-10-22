package tui

import (
	"encoding/json"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/georgezero/dreamteam-go/internal/session"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// The summary screen's T90 view evaluates in the background and shows the
// same evaluation as the session (and so as the CLI), line for line; a
// later point in play marks it stale.
func TestT90View(t *testing.T) {
	if testing.Short() {
		t.Skip("runs four 200-day worlds")
	}
	s, err := session.Open("sustainable", 907)
	if err != nil {
		t.Fatal(err)
	}
	s.AdvanceTo(s.Now().Add(20 * simtime.Day))
	m := New(s)
	m.Update(tea.WindowSizeMsg{Width: 160, Height: 48})
	m.key("e")
	m.key("tab")
	m.key("tab")
	if m.screen != screenSummary || m.sumView != sumT90 {
		t.Fatalf("screen %d view %d", m.screen, m.sumView)
	}
	if body := strings.Join(m.t90Body(159, 100), "\n"); !strings.Contains(body, "enter: evaluate") {
		t.Fatalf("before evaluating:\n%s", body)
	}
	cmd := m.key("enter")
	if cmd == nil || !m.t90Busy {
		t.Fatal("enter did not start the evaluation")
	}
	if body := strings.Join(m.t90Body(159, 100), "\n"); !strings.Contains(body, "Evaluating") {
		t.Fatalf("while evaluating:\n%s", body)
	}
	m.Update(cmd())
	if m.t90Busy || m.t90Err != nil || m.t90 == nil {
		t.Fatalf("busy %v, err %v", m.t90Busy, m.t90Err)
	}
	want, err := s.T90()
	if err != nil {
		t.Fatal(err)
	}
	a, _ := json.Marshal(m.t90)
	b, _ := json.Marshal(want)
	if string(a) != string(b) {
		t.Fatal("the view's evaluation differs from the session's")
	}
	body := m.t90Body(159, 1000)
	lines := want.Lines(158)
	if len(body) != len(lines)+1 {
		t.Fatalf("%d lines, want the title and %d", len(body), len(lines))
	}
	for i, l := range lines {
		if body[i+1] != " "+l {
			t.Fatalf("line %d: %q, want %q", i+1, body[i+1], " "+l)
		}
	}
	s.AdvanceTo(s.Now().Add(simtime.Day))
	if body := strings.Join(m.t90Body(159, 100), "\n"); !strings.Contains(body, "Stale") {
		t.Fatal("a later point is not marked stale")
	}
	m.key("tab")
	if m.sumView != sumDay {
		t.Fatal("tab does not return to the day")
	}
}

// Outside the sustainable scenario the view says where the evaluation is
// defined, without evaluating.
func TestT90ViewElsewhere(t *testing.T) {
	m := New(openMini(t, 3))
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m.key("e")
	m.key("tab")
	m.key("tab")
	if cmd := m.key("enter"); cmd != nil || m.t90Busy {
		t.Fatal("evaluated a scenario the policy does not cover")
	}
	if body := strings.Join(m.t90Body(119, 40), "\n"); !strings.Contains(body, "dreamteam -scenario sustainable") {
		t.Fatalf("no explanation:\n%s", body)
	}
}
