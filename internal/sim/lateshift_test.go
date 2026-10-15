package sim_test

import (
	"encoding/json"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/scenario"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

func withOverlay(t *testing.T, overlay string) *sim.Config {
	t.Helper()
	f, err := scenario.ResolveFile("established")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := scenario.Canonical(f)
	if err != nil {
		t.Fatal(err)
	}
	var base, over any
	json.Unmarshal(raw, &base)
	if err := json.Unmarshal([]byte(overlay), &over); err != nil {
		t.Fatal(err)
	}
	merged, _ := json.Marshal(scenario.Merge(base, over))
	if f, err = scenario.Decode(merged); err != nil {
		t.Fatal(err)
	}
	cfg, err := scenario.Compile(f)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

// V0.2 Step 0C: short shifts starting after noon run. Their cap is
// accounting close, not start + maxShiftHours (which would cross midnight);
// they staff their room and the engine's invariants hold every day.
func TestShortLateShiftsRun(t *testing.T) {
	cfg := withOverlay(t, `{"employees": [
		{"id": "E09", "name": "Jamie", "role": "technologist", "skills": ["XR", "CT"], "pay": {"type": "hourly", "rateCents": 3800},
		 "roster": {"Tue": {"start": "16:00", "end": "20:00", "assignment": "R1"}, "Wed": {"start": "16:00", "end": "20:00", "assignment": "R1"}}},
		{"id": "E10", "name": "Drew", "role": "technologist", "skills": ["MR"], "pay": {"type": "hourly", "rateCents": 4500},
		 "roster": {"Tue": {"start": "17:00", "end": "20:00", "assignment": "R4"}}}]}`)
	w := sim.New(cfg, rng.SeedFromInt(5))
	w.Engine().RunUntil(w.StartTime(), nil)
	start := w.StartTime()
	seen := map[string]bool{}
	for d := int64(0); d < 14; d++ {
		day := w.DayIndex()
		for _, r := range w.Roster(day) {
			if (r.ID == "E09" || r.ID == "E10") && r.CapAt != "23:59" {
				t.Fatalf("%s cap %s, want 23:59", r.ID, r.CapAt)
			}
		}
		w.Engine().RunUntil(w.Calendar().At(day, 18*simtime.Hour), nil)
		for _, e := range w.Observe(0).Staff {
			if (e.ID == "E09" || e.ID == "E10") && e.Shift != "" {
				if e.CapAt != "23:59" || len(e.Rooms) == 0 {
					t.Fatalf("day %d 18:00: %s cap %q rooms %v", d, e.ID, e.CapAt, e.Rooms)
				}
				seen[e.ID] = true
			}
		}
		w.Engine().RunUntil(start.Add(simtime.Duration(d+1)*simtime.Day), nil)
		if err := w.Check(); err != nil {
			t.Fatal(err)
		}
	}
	if !seen["E09"] || !seen["E10"] {
		t.Fatalf("late shifts never on duty: %v", seen)
	}
}
