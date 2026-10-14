package sim

import (
	"testing"

	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Step 18 review risk 6: under shortest_first and fifo urgency is not a
// sort key, only urgent_then_fifo puts urgent and overdue studies first.
// A controlled micro case through the real worklistOrder: one radiologist;
// a routine XR (expected read 150 s) arrives every 140 s from 08:00 to
// 16:00; one urgent MR (expected read 960 s) arrives at 09:00. The
// radiologist always takes the first study offered and reads it for its
// expected time. This pins the trade-off; it changes no policy.
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

	def, defBefore := run("urgent_then_fifo")
	fifo, fifoBefore := run("fifo")
	short, shortBefore := run("shortest_first")
	t.Logf("urgent MR wait: urgent_then_fifo %v (%d routine first), fifo %v (%d), shortest_first %v (%d)",
		def, defBefore, fifo, fifoBefore, short, shortBefore)
	if def < 0 || def > 150*simtime.Second {
		t.Errorf("urgent_then_fifo: the urgent study waited %v; it should be next after the read in progress", def)
	}
	if fifo <= def || fifo < 0 {
		t.Errorf("fifo: the urgent study waited %v, behind the backlog that arrived before it", fifo)
	}
	if short != -1 {
		t.Errorf("shortest_first: the urgent study was read after %v; with short routine studies always waiting it is expected to starve", short)
	}
}
