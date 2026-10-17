package session

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// 0.2.0-dev.6 changed outcomes, not just encodings: the reading scheduler
// (touching blocks, overrun debt through other duties) and the minimum
// allocation commitment. A 0.2.0-dev.5 record therefore replays in batch
// mode v0.2_slice4: this build's action set under dev.5's rules
// (sim.SetDev5Rules), with every digest verified; a session continuing it
// runs this build's rules from the record's end.

const (
	// Dr. Avery at 100% and Dr. Blake at 95% from Oct 27; Sunday-and-
	// weekend reductions (Blake's later withdrawn, leaving 95% on Oct 27
	// and Nov 2; Avery's 30% on Sat Nov 7, 100% from Mon Nov 9); 30 days.
	dev5Fixture = "../../testdata/sessions/mini-seed24-dev5.session.json"
	// Only the 100%/95% batch; 20 days.
	dev5ReadingFixture = "../../testdata/sessions/mini-seed25-dev5-reading.session.json"
)

// Both dev.5 records replay exactly in their own batch mode, in any
// chunking. Before 0.2.0-dev.5 was registered they were refused as an
// engine mismatch; registered without dev.5's rules, dev5Fixture's second
// batch (accepted by dev.5) is refused.
func TestDev5RecordsReplay(t *testing.T) {
	for _, f := range []string{dev5Fixture, dev5ReadingFixture} {
		_, rep, err := Load(f, ReplayOptions{})
		if err != nil || !rep.OK || !rep.Earlier || rep.Compatibility() != "earlier_build_exact" || rep.Replayed != rep.Recorded {
			t.Fatalf("%s: %v %s", f, err, repString(rep))
		}
		for _, step := range []simtime.Duration{37 * simtime.Minute, 9 * simtime.Hour} {
			if _, rep, err := Load(f, ReplayOptions{Step: step}); err != nil || rep.Replayed != rep.Recorded {
				t.Fatalf("%s step %s: %v %s", f, step, err, repString(rep))
			}
		}
	}
}

// The reading fixture's outcomes depend on the scheduler: its batch is
// accepted under either build's rules, and replayed by hand it reproduces
// the recorded digests only under dev.5's rules. The change is to outcomes
// (the log), not only to the state encoding.
func TestDev5ReadingOutcomesNeedDev5Rules(t *testing.T) {
	raw, err := os.ReadFile(dev5ReadingFixture)
	if err != nil {
		t.Fatal(err)
	}
	r, err := DecodeRecord(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, dev5 := range []bool{true, false} {
		s := openFile(t, "mini", r.Seed, nil)
		s.w.SetDev5Rules(dev5)
		for _, b := range r.Batches {
			s.w.Engine().RunUntil(simtime.Time(b.AtSec), nil)
			for _, res := range s.Apply(b.Actions...) {
				if res.Code != api.ActionAccepted {
					t.Fatalf("dev5 %v: %+v", dev5, res)
				}
			}
		}
		s.w.Engine().RunUntil(simtime.Time(r.UntilSec), nil)
		got := s.Digests()
		switch {
		case dev5 && got != r.Digests:
			t.Fatalf("dev.5 rules: %+v, recorded %+v", got, r.Digests)
		case !dev5 && (got.Log == r.Digests.Log || got.Run == r.Digests.Run):
			t.Fatalf("this build's rules reproduce dev.5's outcomes: the fixture does not exercise the scheduler change")
		}
	}
}

// A loaded dev.5 session continues under this build's rules: the
// commitment applies to new changes (old short pairs are history), and Dr.
// Avery at 100% reads through his block boundaries (dev.5's rules would
// have left him in other duties after 09:10). It saves with the dev.5
// record as its prefix, verbatim, and reloads exactly.
func TestDev5RecordContinues(t *testing.T) {
	l, _, err := Load(dev5Fixture, ReplayOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if v := l.ReadingAllocations()[0].Versions; len(v) != 4 || v[2].Effective != "2026-11-07" || v[3].Effective != "2026-11-09" {
		t.Fatalf("Avery's dev.5 timeline: %+v", v)
	}
	at(t, l, 30, "10:00") // Thu Nov 12
	status := func() string {
		for _, x := range l.Observe(0).Staff {
			if x.ID == "E07" {
				return x.Status
			}
		}
		return ""
	}
	if s := status(); s != api.StaffReading && s != api.StaffReadingFree {
		t.Fatalf("Avery at 100%%, 10:00: %s", s)
	}
	at(t, l, 30, "12:00")
	res := l.Apply(allocAct("E07", "2026-11-26", 6000), allocAct("E08", "2026-11-30", 5000), allocAct("E08", "2026-12-03", 6000))
	if res[0].Code != "batch_rejected" || res[1].Code != "commitment_too_short" {
		t.Fatalf("short pair after a dev.5 record: %+v", res)
	}
	if res := l.Apply(allocAct("E07", "2026-11-26", 6000), allocAct("E08", "2026-11-30", 5000)); res[0].Code != api.ActionAccepted || res[1].Code != api.ActionAccepted {
		t.Fatalf("%+v", res)
	}
	at(t, l, 32, "21:00")
	p := saveTemp(t, l)
	raw, _ := os.ReadFile(p)
	cont, err := DecodeRecord(raw)
	if err != nil || cont.Legacy == nil || cont.Legacy.Engine != "0.2.0-dev.5" || cont.Legacy.BatchMode != "v0.2_slice4" ||
		cont.Legacy.Outcome != CompatExact || cont.Legacy.State != CompatExact || len(cont.Batches) != 2 {
		t.Fatalf("continued record: %v %+v", err, cont.Legacy)
	}
	fixture, _ := os.ReadFile(dev5Fixture)
	var a, b bytes.Buffer
	json.Compact(&a, fixture)
	json.Compact(&b, cont.Legacy.Record)
	if a.String() != b.String() {
		t.Fatal("the dev.5 record is not embedded verbatim")
	}
	for _, step := range []simtime.Duration{0, 41 * simtime.Minute} {
		l2, rep, err := Load(p, ReplayOptions{Step: step})
		if err != nil || rep.Compatibility() != "prefix:earlier_build_exact+native_exact" || l2.Digests() != l.Digests() {
			t.Fatalf("continued load (step %s): %v %s", step, err, repString(rep))
		}
	}
	q := copyFile(t, p)
	rewrite(t, q, func(r *Record) { r.Legacy.BatchMode = batchModeSlice3 })
	if _, _, err := Load(q, ReplayOptions{}); !errors.Is(err, ErrIncompatible) && !errors.Is(err, ErrMismatch) {
		t.Errorf("batch mode tampered: %v", err)
	}

	// Under dev.5's rules the same continuation would differ: Avery in
	// other duties after his first block.
	d, _, err := Load(dev5Fixture, ReplayOptions{})
	if err != nil {
		t.Fatal(err)
	}
	d.w.SetDev5Rules(true)
	at(t, d, 30, "10:00")
	for _, x := range d.Observe(0).Staff {
		if x.ID == "E07" && x.Status != api.StaffOtherDuties {
			t.Fatalf("dev.5 rules, Avery at 10:00: %s", x.Status)
		}
	}
}

// A record that claims 0.2.0-dev.5 but whose batches only replay under
// this build's rules is refused, not silently accepted: dev.5's rules are
// applied to every dev.5 record.
func TestDev5RecordWithDev6OutcomesRefused(t *testing.T) {
	s := openFile(t, "mini", 24, nil)
	at(t, s, 0, "13:00")
	if res := s.Apply(allocAct("E08", "2026-10-31", 3000), allocAct("E08", "2026-11-02", 5000)); res[0].Code != "commitment_too_short" {
		t.Fatalf("%+v", res)
	}
	at(t, s, 2, "06:00")
	p := saveTemp(t, s)
	rewrite(t, p, func(r *Record) { r.Engine = "0.2.0-dev.5" })
	_, rep, err := Load(p, ReplayOptions{})
	if !errors.Is(err, ErrMismatch) || !strings.Contains(repString(rep), "commitment_too_short") {
		t.Fatalf("%v %s", err, repString(rep))
	}
}
