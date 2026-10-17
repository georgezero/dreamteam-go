package session

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// 0.2.0-dev.9 (Slice 5C.0, the frozen evaluation) changes no engine
// behaviour; it registers 0.2.0-dev.8, whose records replay with this
// build's action set and rules (batch mode v0.2_dev8), every digest
// verified.

// A contract on weeknights and Sundays from Oct 27 (accepted, T00001); a
// second, overlapping one (contract_overlap); T00001 cancelled on Nov 9
// with end Dec 9; 45 days.
const dev8Fixture = "../../testdata/sessions/mini-seed29-dev8.session.json"

func TestDev8RecordReplays(t *testing.T) {
	_, rep, err := Load(dev8Fixture, ReplayOptions{})
	if err != nil || !rep.OK || !rep.Earlier || rep.Compatibility() != "earlier_build_exact" || rep.Replayed != rep.Recorded {
		t.Fatalf("%v %s", err, repString(rep))
	}
	for _, step := range []simtime.Duration{31 * simtime.Minute, 11 * simtime.Hour} {
		if _, rep, err := Load(dev8Fixture, ReplayOptions{Step: step}); err != nil || rep.Replayed != rep.Recorded {
			t.Fatalf("step %s: %v %s", step, err, repString(rep))
		}
	}
	raw, _ := os.ReadFile(dev8Fixture)
	r, err := DecodeRecord(raw)
	if err != nil {
		t.Fatal(err)
	}
	if r.Engine != "0.2.0-dev.8" || r.Batches[0].Results[0].ContractID != "T00001" || r.Batches[1].Results[0].Code != "contract_overlap" ||
		r.Batches[2].Results[0].Code != api.ActionAccepted {
		t.Fatalf("fixture: %s %+v", r.Engine, r.Batches)
	}
}

// A loaded dev.8 session continues under this build: it saves with the
// record embedded verbatim (batch mode v0.2_dev8) and reloads exactly.
func TestDev8RecordContinues(t *testing.T) {
	l, _, err := Load(dev8Fixture, ReplayOptions{})
	if err != nil {
		t.Fatal(err)
	}
	at(t, l, 50, "09:00")
	if res := l.Apply(teleStart("tele-c", "saturday", "2026-12-26")); res[0].Code != api.ActionAccepted || res[0].ContractID != "T00002" {
		t.Fatalf("%+v", res)
	}
	at(t, l, 80, "06:00")
	p := saveTemp(t, l)
	raw, _ := os.ReadFile(p)
	cont, err := DecodeRecord(raw)
	if err != nil || cont.Legacy == nil || cont.Legacy.Engine != "0.2.0-dev.8" || cont.Legacy.BatchMode != "v0.2_dev8" ||
		cont.Legacy.Outcome != CompatExact || cont.Legacy.State != CompatExact || len(cont.Batches) != 1 {
		t.Fatalf("continued record: %v %+v", err, cont.Legacy)
	}
	fixture, _ := os.ReadFile(dev8Fixture)
	var a, b bytes.Buffer
	json.Compact(&a, fixture)
	json.Compact(&b, cont.Legacy.Record)
	if a.String() != b.String() {
		t.Fatal("the dev.8 record is not embedded verbatim")
	}
	for _, step := range []simtime.Duration{0, 47 * simtime.Minute} {
		l2, rep, err := Load(p, ReplayOptions{Step: step})
		if err != nil || rep.Compatibility() != "prefix:earlier_build_exact+native_exact" || l2.Digests() != l.Digests() {
			t.Fatalf("continued load (step %s): %v %s", step, err, repString(rep))
		}
	}
}
