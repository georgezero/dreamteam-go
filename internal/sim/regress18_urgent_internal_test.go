package sim

import (
	"testing"

	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Step 18 review risk 6, and the owner's rule (0.1.0-dev.18.2): urgent
// studies come first under every worklist policy. A controlled micro case
// through the real worklistOrder: one radiologist; a routine XR (expected
// read 150 s) arrives every 140 s from 08:00 to 16:00, so short routine
// work never runs out; one urgent MR (expected read 960 s) arrives at
// 09:00. The radiologist always takes the first study offered and reads it
// for its expected time; the read in progress is never interrupted. Before
// the rule, shortest_first never offered the MR in 7 hours and fifo after
// 5 minutes.
func TestRegress18UrgentLongStudyVsShortRoutine(t *testing.T) {
	q := func(p50 int64) rng.Quantile {
		r, err := rng.NewQuantile([]rng.QuantilePoint{{P: 0, V: p50 / 2}, {P: 500_000, V: p50}, {P: rng.OneMillion, V: 2 * p50}})
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	const xr, mr = 0, 1
	cfg := &Config{Catalog: []ExamConfig{{Code: "XR", ReadTime: q(150)}, {Code: "MR", ReadTime: q(960)}}}
	cfg.Policies.TATTarget = [numPriorities]simtime.Duration{Routine: 24 * simtime.Hour, Soon: 4 * simtime.Hour, Urgent: simtime.Hour}
	day := func(h int) simtime.Time { return simtime.Time(h) * simtime.Time(simtime.Hour) }

	// run returns the urgent study's wait (acquisition to read start), or
	// -1 if it was never read before 16:00, and the routine reads before it.
	run := func(policy string) (wait simtime.Duration, before int) {
		st := &state{cfg: cfg}
		st.pol.Worklist = policy
		next := day(8)     // next routine arrival
		urgentAt := day(9) // urgent arrival
		free := day(8)     // radiologist free from
		for st.now = day(8); st.now < day(16); {
			// Arrivals up to now, in time order.
			for next <= st.now || urgentAt <= st.now {
				s := study{exam: xr, priority: Routine, acquiredAt: next, enteredAt: next, by: -1, claim: -1}
				if urgentAt <= next {
					s = study{exam: mr, priority: Urgent, acquiredAt: urgentAt, enteredAt: urgentAt, by: -1, claim: -1}
					urgentAt = simtime.Time(1 << 62)
				} else {
					next = next.Add(140 * simtime.Second)
				}
				s.state = studyUnread
				st.dem.studies = append(st.dem.studies, s)
				st.dem.worklist = append(st.dem.worklist, len(st.dem.studies)-1)
			}
			if st.now >= free {
				if order := st.worklistOrder(); len(order) > 0 {
					si := order[0]
					s := &st.dem.studies[si]
					if s.priority == Urgent {
						return st.now.Sub(s.acquiredAt), before
					}
					before++
					s.state = studyFinal
					free = st.now.Add(simtime.Duration(cfg.Catalog[s.exam].ReadTime.At(500_000)) * simtime.Second)
				}
			}
			at := min(next, urgentAt)
			if free > st.now {
				at = min(at, free)
			}
			st.now = at
		}
		return -1, before
	}

	for _, policy := range []string{"urgent_then_fifo", "fifo", "oldest_first", "shortest_first"} {
		wait, before := run(policy)
		t.Logf("%s: urgent MR waited %v (%d routine reads first)", policy, wait, before)
		// Next after the read in progress.
		if wait < 0 || wait > 150*simtime.Second {
			t.Errorf("%s: the urgent MR waited %v (%d routine reads first); it must be next after the read in progress", policy, wait, before)
		}
	}
}

// Every policy offers urgent studies before routine ones and orders each
// class by its own key; with no urgent study waiting, shortest_first is
// still shortest first. Deterministic: ties fall to the study index.
func TestRegress18WorklistUrgentFirst(t *testing.T) {
	q := func(p50 int64) rng.Quantile {
		r, _ := rng.NewQuantile([]rng.QuantilePoint{{P: 0, V: p50 / 2}, {P: 500_000, V: p50}, {P: rng.OneMillion, V: 2 * p50}})
		return r
	}
	const xr, ct, mr = 0, 1, 2
	cfg := &Config{Catalog: []ExamConfig{{Code: "XR", ReadTime: q(150)}, {Code: "CT", ReadTime: q(540)}, {Code: "MR", ReadTime: q(960)}}}
	cfg.Policies.TATTarget = [numPriorities]simtime.Duration{Routine: 24 * simtime.Hour, Soon: 4 * simtime.Hour, Urgent: simtime.Hour}
	type st0 struct {
		exam     int
		pri      Priority
		acquired simtime.Duration // minutes after 08:00
		entered  simtime.Duration
	}
	// Entered order differs from acquisition order for the last two.
	studies := []st0{
		{xr, Routine, 0, 5},   // 0
		{mr, Urgent, 10, 15},  // 1
		{ct, Routine, 20, 25}, // 2
		{xr, Urgent, 30, 35},  // 3
		{mr, Routine, 40, 60}, // 4: entered after 5
		{xr, Routine, 45, 50}, // 5
		{ct, Soon, 50, 55},    // 6: soon is not urgent
	}
	build := func(policy string, only func(st0) bool) []int {
		st := &state{cfg: cfg}
		st.pol.Worklist = policy
		st.now = simtime.Time(9 * simtime.Hour)
		for _, x := range studies {
			if only != nil && !only(x) {
				x.pri = Routine
			}
			at := simtime.Time(8*simtime.Hour + x.acquired*simtime.Minute)
			en := simtime.Time(8*simtime.Hour + x.entered*simtime.Minute)
			st.dem.studies = append(st.dem.studies, study{exam: x.exam, priority: x.pri, acquiredAt: at, enteredAt: en,
				state: studyUnread, by: -1, claim: -1})
			st.dem.worklist = append(st.dem.worklist, len(st.dem.studies)-1)
		}
		return st.worklistOrder()
	}
	for _, c := range []struct {
		policy string
		want   []int
	}{
		{"urgent_then_fifo", []int{1, 3, 0, 2, 5, 6, 4}},
		{"fifo", []int{1, 3, 0, 2, 5, 6, 4}},
		{"oldest_first", []int{1, 3, 0, 2, 4, 5, 6}},
		{"shortest_first", []int{3, 1, 0, 5, 2, 6, 4}},
	} {
		got := build(c.policy, nil)
		if !slicesEqual(got, c.want) {
			t.Errorf("%s: order %v, want %v", c.policy, got, c.want)
		}
		if again := build(c.policy, nil); !slicesEqual(again, got) {
			t.Errorf("%s: not deterministic: %v then %v", c.policy, got, again)
		}
	}
	// No urgent study waiting: shortest_first is purely by expected read
	// time (ties by index), as before the rule.
	none := func(st0) bool { return false }
	if got, want := build("shortest_first", none), []int{0, 3, 5, 2, 6, 1, 4}; !slicesEqual(got, want) {
		t.Errorf("shortest_first without urgent studies: %v, want %v", got, want)
	}
}

func slicesEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
