package session

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// 0.4.0-dev.1 (V0.4a, the agent evaluation harness) changes no engine
// behaviour; it registers 0.3.0 (the V0.3 release), whose records replay
// with this build's action set and rules (batch mode v0.3.0), every digest
// verified.

// A Saturday and Sunday contract for all priorities from Nov 1
// (accepted, T00001); on Oct 26 the cancellation of a contract that does
// not exist (unknown_contract); on Nov 8 T00001's cancellation with end
// Jan 31 (accepted); 45 days, seed 52.
const v030Fixture = "../../testdata/sessions/mini-seed52-v030.session.json"

func TestV030RecordReplays(t *testing.T) {
	_, rep, err := Load(v030Fixture, ReplayOptions{})
	if err != nil || !rep.OK || !rep.Earlier || rep.Compatibility() != "earlier_build_exact" || rep.Replayed != rep.Recorded {
		t.Fatalf("%v %s", err, repString(rep))
	}
	for _, step := range []simtime.Duration{37 * simtime.Minute, 7 * simtime.Hour} {
		if _, rep, err := Load(v030Fixture, ReplayOptions{Step: step}); err != nil || rep.Replayed != rep.Recorded {
			t.Fatalf("step %s: %v %s", step, err, repString(rep))
		}
	}
	raw, _ := os.ReadFile(v030Fixture)
	r, err := DecodeRecord(raw)
	if err != nil {
		t.Fatal(err)
	}
	if r.Engine != "0.3.0" || r.Batches[0].Results[0].ContractID != "T00001" || r.Batches[1].Results[0].Code != "unknown_contract" ||
		r.Batches[2].Results[0].Code != "accepted" {
		t.Fatalf("fixture: %s %+v", r.Engine, r.Batches)
	}
}

// A loaded 0.3.0 session continues under this build: it saves with the
// record embedded verbatim (batch mode v0.3.0) and reloads exactly.
func TestV030RecordContinues(t *testing.T) {
	l, _, err := Load(v030Fixture, ReplayOptions{})
	if err != nil {
		t.Fatal(err)
	}
	at(t, l, 50, "09:00")
	if res := l.Apply(teleStart("tele-c", "sunday", "2027-01-31")); !res[0].OK || res[0].ContractID != "T00002" {
		t.Fatalf("%+v", res)
	}
	at(t, l, 80, "06:00")
	p := saveTemp(t, l)
	raw, _ := os.ReadFile(p)
	cont, err := DecodeRecord(raw)
	if err != nil || cont.Legacy == nil || cont.Legacy.Engine != "0.3.0" || cont.Legacy.BatchMode != "v0.3.0" ||
		cont.Legacy.Outcome != CompatExact || cont.Legacy.State != CompatExact || len(cont.Batches) != 1 {
		t.Fatalf("continued record: %v %+v", err, cont.Legacy)
	}
	fixture, _ := os.ReadFile(v030Fixture)
	var a, b bytes.Buffer
	json.Compact(&a, fixture)
	json.Compact(&b, cont.Legacy.Record)
	if a.String() != b.String() {
		t.Fatal("the 0.3.0 record is not embedded verbatim")
	}
	for _, step := range []simtime.Duration{0, 41 * simtime.Minute} {
		l2, rep, err := Load(p, ReplayOptions{Step: step})
		if err != nil || rep.Compatibility() != "prefix:earlier_build_exact+native_exact" || l2.Digests() != l.Digests() {
			t.Fatalf("continued load (step %s): %v %s", step, err, repString(rep))
		}
	}
}
