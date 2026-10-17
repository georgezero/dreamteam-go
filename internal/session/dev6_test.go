package session

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// 0.2.0-dev.7 (Slice 5A, the remote reading engine) changes no outcome
// without a teleradiology contract, which no action can create yet. A
// 0.2.0-dev.6 record therefore replays in batch mode v0.2_dev6: this
// build's action set under this build's rules, every digest verified. It
// must not replay under dev.5's rules (as every earlier mode does): its
// batches depend on dev.6's commitment and reading rules.

// Dr. Avery at 100% and Dr. Blake at 95% from Oct 27; Dr. Blake's Sunday
// reduction refused (commitment_too_short); Dr. Avery back to 60% from
// Nov 10; 25 days.
const dev6Fixture = "../../testdata/sessions/mini-seed26-dev6.session.json"

// The record replays exactly, whole and in any chunking. Before 0.2.0-dev.6
// was registered it was refused as an engine mismatch.
func TestDev6RecordReplays(t *testing.T) {
	_, rep, err := Load(dev6Fixture, ReplayOptions{})
	if err != nil || !rep.OK || !rep.Earlier || rep.Compatibility() != "earlier_build_exact" || rep.Replayed != rep.Recorded {
		t.Fatalf("%v %s", err, repString(rep))
	}
	for _, step := range []simtime.Duration{29 * simtime.Minute, 7 * simtime.Hour} {
		if _, rep, err := Load(dev6Fixture, ReplayOptions{Step: step}); err != nil || rep.Replayed != rep.Recorded {
			t.Fatalf("step %s: %v %s", step, err, repString(rep))
		}
	}
}

// Replayed by hand, the record's results and digests are reproduced under
// this build's rules and not under dev.5's: its second batch would be
// accepted there, and Dr. Avery at 100% would read differently.
func TestDev6RecordNeedsDev6Rules(t *testing.T) {
	raw, err := os.ReadFile(dev6Fixture)
	if err != nil {
		t.Fatal(err)
	}
	r, err := DecodeRecord(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, dev5 := range []bool{false, true} {
		s := openFile(t, "mini", r.Seed, nil)
		s.w.SetDev5Rules(dev5)
		same := true
		for _, b := range r.Batches {
			s.w.Engine().RunUntil(simtime.Time(b.AtSec), nil)
			if got := s.Apply(b.Actions...); !reflect.DeepEqual(got, b.Results) {
				same = false
			}
		}
		s.w.Engine().RunUntil(simtime.Time(r.UntilSec), nil)
		switch {
		case !dev5 && (!same || s.Digests() != r.Digests):
			t.Fatalf("this build's rules: results same %v, digests %+v, recorded %+v", same, s.Digests(), r.Digests)
		case dev5 && same:
			t.Fatal("dev.5's rules reproduce the dev.6 record's results: the fixture does not bind dev.6's rules")
		}
	}
}

// A loaded dev.6 session continues under this build's rules, saves with
// the dev.6 record embedded verbatim as its prefix (batch mode v0.2_dev6)
// and reloads exactly, whole and chunked.
func TestDev6RecordContinues(t *testing.T) {
	l, _, err := Load(dev6Fixture, ReplayOptions{})
	if err != nil {
		t.Fatal(err)
	}
	at(t, l, 25, "12:00") // Sat Nov 7
	if res := l.Apply(allocAct("E08", "2026-11-24", 5000)); res[0].Code != api.ActionAccepted {
		t.Fatalf("%+v", res)
	}
	at(t, l, 27, "21:00")
	p := saveTemp(t, l)
	raw, _ := os.ReadFile(p)
	cont, err := DecodeRecord(raw)
	if err != nil || cont.Legacy == nil || cont.Legacy.Engine != "0.2.0-dev.6" || cont.Legacy.BatchMode != "v0.2_dev6" ||
		cont.Legacy.Outcome != CompatExact || cont.Legacy.State != CompatExact || len(cont.Batches) != 1 {
		t.Fatalf("continued record: %v %+v", err, cont.Legacy)
	}
	fixture, _ := os.ReadFile(dev6Fixture)
	var a, b bytes.Buffer
	json.Compact(&a, fixture)
	json.Compact(&b, cont.Legacy.Record)
	if a.String() != b.String() {
		t.Fatal("the dev.6 record is not embedded verbatim")
	}
	for _, step := range []simtime.Duration{0, 53 * simtime.Minute} {
		l2, rep, err := Load(p, ReplayOptions{Step: step})
		if err != nil || rep.Compatibility() != "prefix:earlier_build_exact+native_exact" || l2.Digests() != l.Digests() {
			t.Fatalf("continued load (step %s): %v %s", step, err, repString(rep))
		}
	}
}
