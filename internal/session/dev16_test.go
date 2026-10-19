package session

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// 0.2.0-dev.17 (Slice 5C.4c, the v5 clinical access and weekly severity
// gates) changes no engine behaviour; it registers 0.2.0-dev.16, whose
// records replay with this build's action set and rules (batch mode
// v0.2_dev16), every digest verified.

// An urgent weeknight contract from Oct 28 (accepted, T00001); on Oct 20
// the cancellation of a contract that does not exist (unknown_contract);
// on Nov 6 T00001's cancellation with end Dec 3 (notice_too_short); 45
// days.
const dev16Fixture = "../../testdata/sessions/mini-seed45-dev16.session.json"

func TestDev16RecordReplays(t *testing.T) {
	_, rep, err := Load(dev16Fixture, ReplayOptions{})
	if err != nil || !rep.OK || !rep.Earlier || rep.Compatibility() != "earlier_build_exact" || rep.Replayed != rep.Recorded {
		t.Fatalf("%v %s", err, repString(rep))
	}
	for _, step := range []simtime.Duration{37 * simtime.Minute, 7 * simtime.Hour} {
		if _, rep, err := Load(dev16Fixture, ReplayOptions{Step: step}); err != nil || rep.Replayed != rep.Recorded {
			t.Fatalf("step %s: %v %s", step, err, repString(rep))
		}
	}
	raw, _ := os.ReadFile(dev16Fixture)
	r, err := DecodeRecord(raw)
	if err != nil {
		t.Fatal(err)
	}
	if r.Engine != "0.2.0-dev.16" || r.Batches[0].Results[0].ContractID != "T00001" || r.Batches[1].Results[0].Code != "unknown_contract" ||
		r.Batches[2].Results[0].Code != "notice_too_short" {
		t.Fatalf("fixture: %s %+v", r.Engine, r.Batches)
	}
}

// A loaded dev.16 session continues under this build: it saves with the
// record embedded verbatim (batch mode v0.2_dev16) and reloads exactly.
func TestDev16RecordContinues(t *testing.T) {
	l, _, err := Load(dev16Fixture, ReplayOptions{})
	if err != nil {
		t.Fatal(err)
	}
	at(t, l, 50, "09:00")
	cancel := api.Action{Type: api.ActContractTeleradiology, Op: api.TeleOpCancel, Contract: "T00001", EndDate: "2027-01-31"}
	if res := l.Apply(cancel); !res[0].OK {
		t.Fatalf("%+v", res)
	}
	at(t, l, 80, "06:00")
	p := saveTemp(t, l)
	raw, _ := os.ReadFile(p)
	cont, err := DecodeRecord(raw)
	if err != nil || cont.Legacy == nil || cont.Legacy.Engine != "0.2.0-dev.16" || cont.Legacy.BatchMode != "v0.2_dev16" ||
		cont.Legacy.Outcome != CompatExact || cont.Legacy.State != CompatExact || len(cont.Batches) != 1 {
		t.Fatalf("continued record: %v %+v", err, cont.Legacy)
	}
	fixture, _ := os.ReadFile(dev16Fixture)
	var a, b bytes.Buffer
	json.Compact(&a, fixture)
	json.Compact(&b, cont.Legacy.Record)
	if a.String() != b.String() {
		t.Fatal("the dev.16 record is not embedded verbatim")
	}
	for _, step := range []simtime.Duration{0, 41 * simtime.Minute} {
		l2, rep, err := Load(p, ReplayOptions{Step: step})
		if err != nil || rep.Compatibility() != "prefix:earlier_build_exact+native_exact" || l2.Digests() != l.Digests() {
			t.Fatalf("continued load (step %s): %v %s", step, err, repString(rep))
		}
	}
}
