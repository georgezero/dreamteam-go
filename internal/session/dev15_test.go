package session

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// 0.2.0-dev.16 (Slice 5C.4b, the v0.2-eval-5 clinical evaluator) changes
// no engine behaviour; it registers 0.2.0-dev.15, whose records replay
// with this build's action set and rules (batch mode v0.2_dev15), every
// digest verified.

// An urgent weekend contract from Oct 30 (accepted, T00001); on Oct 21 the
// cancellation of a contract that does not exist (unknown_contract); on
// Nov 4 T00001's cancellation with end Dec 5 (accepted); 45 days.
const dev15Fixture = "../../testdata/sessions/mini-seed43-dev15.session.json"

func TestDev15RecordReplays(t *testing.T) {
	_, rep, err := Load(dev15Fixture, ReplayOptions{})
	if err != nil || !rep.OK || !rep.Earlier || rep.Compatibility() != "earlier_build_exact" || rep.Replayed != rep.Recorded {
		t.Fatalf("%v %s", err, repString(rep))
	}
	for _, step := range []simtime.Duration{37 * simtime.Minute, 7 * simtime.Hour} {
		if _, rep, err := Load(dev15Fixture, ReplayOptions{Step: step}); err != nil || rep.Replayed != rep.Recorded {
			t.Fatalf("step %s: %v %s", step, err, repString(rep))
		}
	}
	raw, _ := os.ReadFile(dev15Fixture)
	r, err := DecodeRecord(raw)
	if err != nil {
		t.Fatal(err)
	}
	if r.Engine != "0.2.0-dev.15" || r.Batches[0].Results[0].ContractID != "T00001" || r.Batches[1].Results[0].Code != "unknown_contract" ||
		r.Batches[2].Results[0].Code != "accepted" {
		t.Fatalf("fixture: %s %+v", r.Engine, r.Batches)
	}
}

// A loaded dev.15 session continues under this build: it saves with the
// record embedded verbatim (batch mode v0.2_dev15) and reloads exactly.
func TestDev15RecordContinues(t *testing.T) {
	l, _, err := Load(dev15Fixture, ReplayOptions{})
	if err != nil {
		t.Fatal(err)
	}
	at(t, l, 50, "09:00")
	if res := l.Apply(teleStart("tele-c", "sunday", "2026-12-16")); !res[0].OK || res[0].ContractID != "T00002" {
		t.Fatalf("%+v", res)
	}
	at(t, l, 80, "06:00")
	p := saveTemp(t, l)
	raw, _ := os.ReadFile(p)
	cont, err := DecodeRecord(raw)
	if err != nil || cont.Legacy == nil || cont.Legacy.Engine != "0.2.0-dev.15" || cont.Legacy.BatchMode != "v0.2_dev15" ||
		cont.Legacy.Outcome != CompatExact || cont.Legacy.State != CompatExact || len(cont.Batches) != 1 {
		t.Fatalf("continued record: %v %+v", err, cont.Legacy)
	}
	fixture, _ := os.ReadFile(dev15Fixture)
	var a, b bytes.Buffer
	json.Compact(&a, fixture)
	json.Compact(&b, cont.Legacy.Record)
	if a.String() != b.String() {
		t.Fatal("the dev.15 record is not embedded verbatim")
	}
	for _, step := range []simtime.Duration{0, 41 * simtime.Minute} {
		l2, rep, err := Load(p, ReplayOptions{Step: step})
		if err != nil || rep.Compatibility() != "prefix:earlier_build_exact+native_exact" || l2.Digests() != l.Digests() {
			t.Fatalf("continued load (step %s): %v %s", step, err, repString(rep))
		}
	}
}
