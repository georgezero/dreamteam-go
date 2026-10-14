package sim_test

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Step 18 review finding 4: the report mixed flows (a "330" whose parts
// summed to about 372). An order cohort reconciles exactly: every order
// received in the window is, at its end, completed (imaged, reported or
// not), ended without an exam (lost, cancelled, no-show, left), or still
// outstanding (booked, on recall, on site). Small enough to check by hand
// from the log: mini scenario, seed 1, one week.
func TestRegress18OrderCohortReconciles(t *testing.T) {
	w, d0 := miniDay(t, 1)
	start := w.StartTime()
	end := d0.Add(7 * simtime.Day)
	if err := runTo(w, end); err != nil {
		t.Fatal(err)
	}
	class := map[string]string{
		"imaged": "completed", "reported": "completed",
		"lost": "ended", "cancelled": "ended", "no-show": "ended", "left": "ended",
		"booked": "outstanding", "recall": "outstanding", "visit": "outstanding",
	}
	exams := map[string]int{}
	for _, v := range w.Visits() {
		if v.ExamStart != 0 {
			exams[v.Order]++
		}
	}
	type key struct{ origin, class, state string }
	n := map[key]int{}
	arrived := map[string]int{}
	for _, o := range w.Orders() {
		if o.Received < start || o.Received >= end {
			continue
		}
		origin := "referral"
		if o.WalkIn {
			origin = "walk-in"
		}
		c, ok := class[o.State]
		if !ok {
			t.Fatalf("order %s in unclassified state %q", o.ID, o.State)
		}
		// Completed orders have exactly one exam; ended ones none (an
		// outstanding order on site may be in its exam).
		if c == "completed" && exams[o.ID] != 1 || c == "ended" && exams[o.ID] != 0 || exams[o.ID] > 1 {
			t.Errorf("order %s (%s): %d exams", o.ID, o.State, exams[o.ID])
		}
		arrived[origin]++
		n[key{origin, c, o.State}]++
	}
	var lines []string
	for origin, a := range arrived {
		parts := map[string]int{}
		for k, v := range n {
			if k.origin == origin {
				parts[k.class] += v
				lines = append(lines, fmt.Sprintf("%-8s %-11s %-9s %4d", origin, k.class, k.state, v))
			}
		}
		if parts["completed"]+parts["ended"]+parts["outstanding"] != a {
			t.Errorf("%s: %d arrived ≠ %d completed + %d ended + %d outstanding", origin, a, parts["completed"], parts["ended"], parts["outstanding"])
		}
		lines = append(lines, fmt.Sprintf("%-8s arrived %d = completed %d + ended %d + outstanding %d", origin, a,
			parts["completed"], parts["ended"], parts["outstanding"]))
	}
	sort.Strings(lines)
	if arrived["referral"] == 0 || arrived["walk-in"] == 0 {
		t.Fatalf("empty cohort: %v", arrived)
	}
	t.Log("orders received " + w.Calendar().ISO(start) + " → " + w.Calendar().ISO(end) + ", as of the end:\n" + strings.Join(lines, "\n"))
}
