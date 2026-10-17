package session

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// 0.2.0-dev.8 (Slice 5B) adds contract_teleradiology. A 0.2.0-dev.7
// record (and dev.5's and dev.6's) was written when the type was an
// unknown action: alone it was refused as unknown_action, beside a
// staffing action as mixed_batch with the Slice 4 message. Such records
// replay with Slice 4's action set (batch modes v0.2_slice4, v0.2_dev6,
// v0.2_dev7) and this build's rules (dev.6 and dev.7), every digest
// verified.

// A contract_teleradiology alone (unknown_action); one beside Dr. Avery
// at 80% (mixed_batch, the batch refused); Dr. Blake at 70% from Oct 30;
// 25 days.
const dev7Fixture = "../../testdata/sessions/mini-seed27-dev7.session.json"

func teleStart(req, windows, effective string) api.Action {
	return api.Action{Type: api.ActContractTeleradiology, Op: api.TeleOpStart, RequestID: req, Plan: "standard-v1",
		Windows: []string{windows}, Scope: "all", Effective: effective}
}

// The record replays exactly, whole and chunked. Before 0.2.0-dev.7 was
// registered it was refused as an engine mismatch.
func TestDev7RecordReplays(t *testing.T) {
	_, rep, err := Load(dev7Fixture, ReplayOptions{})
	if err != nil || !rep.OK || !rep.Earlier || rep.Compatibility() != "earlier_build_exact" || rep.Replayed != rep.Recorded {
		t.Fatalf("%v %s", err, repString(rep))
	}
	for _, step := range []simtime.Duration{29 * simtime.Minute, 7 * simtime.Hour} {
		if _, rep, err := Load(dev7Fixture, ReplayOptions{Step: step}); err != nil || rep.Replayed != rep.Recorded {
			t.Fatalf("step %s: %v %s", step, err, repString(rep))
		}
	}
	for _, f := range []string{dev6Fixture, "../../testdata/sessions/mini-seed24-dev5.session.json"} {
		if _, rep, err := Load(f, ReplayOptions{}); err != nil || rep.Compatibility() != "earlier_build_exact" {
			t.Fatalf("%s: %v %s", f, err, repString(rep))
		}
	}
}

// Replayed by hand, the record's results reproduce with Slice 4's action
// set and not with this build's, which accepts the type (and refuses the
// recorded item, which has no plan, as invalid_schema).
func TestDev7RecordNeedsSlice4Actions(t *testing.T) {
	raw, err := os.ReadFile(dev7Fixture)
	if err != nil {
		t.Fatal(err)
	}
	r, err := DecodeRecord(raw)
	if err != nil {
		t.Fatal(err)
	}
	if r.Batches[0].Results[0].Code != "unknown_action" || r.Batches[1].Results[1].Code != "mixed_batch" {
		t.Fatalf("fixture: %+v", r.Batches[:2])
	}
	for _, slice4 := range []bool{true, false} {
		s := openFile(t, "mini", r.Seed, nil)
		same := true
		for _, b := range r.Batches {
			s.w.Engine().RunUntil(simtime.Time(b.AtSec), nil)
			var got []api.ActionResult
			if slice4 {
				got = s.w.ApplySlice4(b.Actions)
			} else {
				got = s.w.Apply(b.Actions)
			}
			if !reflect.DeepEqual(got, b.Results) {
				same = false
				if !slice4 && got[len(got)-1].Code != "invalid_schema" {
					t.Fatalf("this build: %+v", got)
				}
			}
		}
		s.w.Engine().RunUntil(simtime.Time(r.UntilSec), nil)
		switch {
		case slice4 && (!same || s.Digests() != r.Digests):
			t.Fatalf("Slice 4's actions: results same %v, digests %+v, recorded %+v", same, s.Digests(), r.Digests)
		case !slice4 && same:
			t.Fatal("this build's action set reproduces the dev.7 record: the fixture does not bind the old meaning")
		}
	}
}

// A loaded dev.7 session continues under this build, where the type is a
// contract: it saves with the record embedded verbatim (batch mode
// v0.2_dev7) and reloads exactly.
func TestDev7RecordContinues(t *testing.T) {
	l, _, err := Load(dev7Fixture, ReplayOptions{})
	if err != nil {
		t.Fatal(err)
	}
	at(t, l, 25, "09:00") // Sat Nov 7
	if res := l.Apply(teleStart("tele-3", "weeknights", "2026-11-23")); res[0].Code != api.ActionAccepted || res[0].ContractID != "T00001" {
		t.Fatalf("%+v", res)
	}
	at(t, l, 45, "06:00")
	if v := l.Teleradiology(); v.RemoteReports == 0 || v.RetainerCents == 0 {
		t.Fatalf("%+v", v)
	}
	p := saveTemp(t, l)
	raw, _ := os.ReadFile(p)
	cont, err := DecodeRecord(raw)
	if err != nil || cont.Legacy == nil || cont.Legacy.Engine != "0.2.0-dev.7" || cont.Legacy.BatchMode != "v0.2_dev7" ||
		cont.Legacy.Outcome != CompatExact || cont.Legacy.State != CompatExact || len(cont.Batches) != 1 {
		t.Fatalf("continued record: %v %+v", err, cont.Legacy)
	}
	fixture, _ := os.ReadFile(dev7Fixture)
	var a, b bytes.Buffer
	json.Compact(&a, fixture)
	json.Compact(&b, cont.Legacy.Record)
	if a.String() != b.String() {
		t.Fatal("the dev.7 record is not embedded verbatim")
	}
	for _, step := range []simtime.Duration{0, 53 * simtime.Minute} {
		l2, rep, err := Load(p, ReplayOptions{Step: step})
		if err != nil || rep.Compatibility() != "prefix:earlier_build_exact+native_exact" || l2.Digests() != l.Digests() {
			t.Fatalf("continued load (step %s): %v %s", step, err, repString(rep))
		}
	}
}

// 26. Save and load while a remote lane is in its handoff and while it
// reads: the load (a replay) reaches the same state, and both continue
// identically, whole and chunked.
func TestTeleSaveLoadMidHandoffAndRead(t *testing.T) {
	s := openFile(t, "mini", 28, nil)
	at(t, s, 0, "13:00")
	if res := s.Apply(teleStart("tele-1", "weeknights", "2026-10-27"), teleStart("tele-0", "sunday", "2026-11-01")); res[0].Code != "batch_rejected" ||
		res[1].Code != "contract_overlap" || !strings.Contains(res[1].Message, `new contract "tele-1" (from 2026-10-27, open-ended) and new contract "tele-0"`) {
		t.Fatalf("%+v", res)
	}
	if res := s.Apply(teleStart("tele-1", "weeknights", "2026-10-27")); res[0].Code != api.ActionAccepted {
		t.Fatalf("%+v", res)
	}
	phase := func(want string) bool {
		for _, l := range s.Teleradiology().Lanes {
			if l.Phase == want {
				return true
			}
		}
		return false
	}
	for _, want := range []string{"handoff", "reading"} {
		for !phase(want) {
			s.w.Engine().RunUntil(s.Now().Add(simtime.Minute), nil)
			if s.Now() > s.w.StartTime().Add(40*simtime.Day) {
				t.Fatalf("no lane in %s", want)
			}
		}
		p := saveTemp(t, s)
		later := s.Now().Add(26 * simtime.Hour)
		var cont []Digests
		for _, step := range []simtime.Duration{0, 17 * simtime.Minute} {
			l, rep, err := Load(p, ReplayOptions{Step: step})
			if err != nil || !rep.OK || l.Digests() != s.Digests() || !reflect.DeepEqual(l.Teleradiology(), s.Teleradiology()) {
				t.Fatalf("%s, step %s: %v %s", want, step, err, repString(rep))
			}
			l.w.Engine().RunUntil(later, nil)
			cont = append(cont, l.Digests())
		}
		s.w.Engine().RunUntil(later, nil)
		if cont[0] != s.Digests() || cont[1] != s.Digests() {
			t.Fatalf("continued after a save in %s: digests differ", want)
		}
	}
}
