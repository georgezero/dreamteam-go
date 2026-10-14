package session

import (
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/simtime"
)

func mini(t testing.TB, seed int64) *Session {
	t.Helper()
	s, err := Open("mini", seed)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestStartsTuesdayMorning(t *testing.T) {
	s := mini(t, 1)
	o := s.Observe(10)
	if o.Weekday != "Tue" || o.Clock != "06:00" || o.Day != 1 || o.Date != "2026-10-13" {
		t.Fatalf("start = %s %s day %d %s", o.Weekday, o.Clock, o.Day, o.Date)
	}
	if o.Practice.Open {
		t.Fatal("practice open at 06:00")
	}
}

func TestStepVisibleLandsOnNewEntry(t *testing.T) {
	s := mini(t, 1)
	for i := 0; i < 50; i++ {
		before := s.Observe(1).LogTotal
		if s.StepVisible() != Visible {
			t.Fatal("no visible entry within horizon")
		}
		o := s.Observe(1)
		if o.LogTotal <= before {
			t.Fatal("StepVisible did not add an entry")
		}
		if last := o.RecentLog[len(o.RecentLog)-1]; last.At != s.Now() {
			t.Fatalf("clock %d not at entry time %d", s.Now(), last.At)
		}
	}
	first := mini(t, 1)
	// The 47 opening appointments hold O00001–O00047; the first order of
	// the day arrives after 06:00 and is the first observable entry.
	if first.StepVisible() != Visible || first.Observe(1).RecentLog[0].Category != "orders" ||
		!strings.HasPrefix(first.Observe(1).RecentLog[0].Text, "O00048 ") {
		t.Errorf("first visible entry = %+v", first.Observe(1).RecentLog)
	}
}

func TestBudgetStopsAtTransactionAndResumes(t *testing.T) {
	a := mini(t, 4)
	a.Advance(3*simtime.Day, Options{})

	b := mini(t, 4)
	target := b.Now().Add(3 * simtime.Day)
	stops := 0
	for b.Now() < target {
		if b.Advance(target.Sub(b.Now()), Options{EventBudget: 7}) == Budget {
			stops++
		}
	}
	if stops == 0 {
		t.Fatal("budget never applied")
	}
	if a.Digest() != b.Digest() {
		t.Fatal("budgeted advance changed the outcome")
	}
}
