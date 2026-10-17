package session

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// 0.2.0-dev.10 (Slice 5C.0a, evaluation policy v0.2-eval-2) changes no
// engine behaviour; it registers 0.2.0-dev.9, whose records replay with
// this build's action set and rules (batch mode v0.2_dev9), every digest
// verified.

// A contract on Saturdays and Sundays for urgent studies from Oct 31
// (accepted, T00001); its cancellation with too little notice
// (notice_too_short); then with end Dec 20 (accepted); 45 days.
const dev9Fixture = "../../testdata/sessions/mini-seed31-dev9.session.json"

func TestDev9RecordReplays(t *testing.T) {
	_, rep, err := Load(dev9Fixture, ReplayOptions{})
	if err != nil || !rep.OK || !rep.Earlier || rep.Compatibility() != "earlier_build_exact" || rep.Replayed != rep.Recorded {
		t.Fatalf("%v %s", err, repString(rep))
	}
	for _, step := range []simtime.Duration{29 * simtime.Minute, 13 * simtime.Hour} {
		if _, rep, err := Load(dev9Fixture, ReplayOptions{Step: step}); err != nil || rep.Replayed != rep.Recorded {
			t.Fatalf("step %s: %v %s", step, err, repString(rep))
		}
	}
	raw, _ := os.ReadFile(dev9Fixture)
	r, err := DecodeRecord(raw)
	if err != nil {
		t.Fatal(err)
	}
	if r.Engine != "0.2.0-dev.9" || r.Batches[0].Results[0].ContractID != "T00001" || r.Batches[1].Results[0].Code != "notice_too_short" ||
		r.Batches[2].Results[0].Code != api.ActionAccepted {
		t.Fatalf("fixture: %s %+v", r.Engine, r.Batches)
	}
}

// A loaded dev.9 session continues under this build: it saves with the
// record embedded verbatim (batch mode v0.2_dev9) and reloads exactly.
func TestDev9RecordContinues(t *testing.T) {
	l, _, err := Load(dev9Fixture, ReplayOptions{})
	if err != nil {
		t.Fatal(err)
	}
	at(t, l, 50, "09:00")
	if res := l.Apply(teleStart("tele-c", "weeknights", "2026-12-28")); res[0].Code != api.ActionAccepted || res[0].ContractID != "T00002" {
		t.Fatalf("%+v", res)
	}
	at(t, l, 80, "06:00")
	p := saveTemp(t, l)
	raw, _ := os.ReadFile(p)
	cont, err := DecodeRecord(raw)
	if err != nil || cont.Legacy == nil || cont.Legacy.Engine != "0.2.0-dev.9" || cont.Legacy.BatchMode != "v0.2_dev9" ||
		cont.Legacy.Outcome != CompatExact || cont.Legacy.State != CompatExact || len(cont.Batches) != 1 {
		t.Fatalf("continued record: %v %+v", err, cont.Legacy)
	}
	fixture, _ := os.ReadFile(dev9Fixture)
	var a, b bytes.Buffer
	json.Compact(&a, fixture)
	json.Compact(&b, cont.Legacy.Record)
	if a.String() != b.String() {
		t.Fatal("the dev.9 record is not embedded verbatim")
	}
	for _, step := range []simtime.Duration{0, 53 * simtime.Minute} {
		l2, rep, err := Load(p, ReplayOptions{Step: step})
		if err != nil || rep.Compatibility() != "prefix:earlier_build_exact+native_exact" || l2.Digests() != l.Digests() {
			t.Fatalf("continued load (step %s): %v %s", step, err, repString(rep))
		}
	}
}
