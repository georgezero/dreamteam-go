package session

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// 0.2.0-dev.13 (Slice 5C.2, evaluation policy v0.2-eval-3) changes no
// engine behaviour; it registers 0.2.0-dev.12, whose records replay with
// this build's action set and rules (batch mode v0.2_dev12), every digest
// verified.

// An urgent-only weeknight contract from Oct 27 (accepted, T00001); on
// Nov 2 its cancellation with end Dec 3 and a weekend successor from Dec 3
// in one batch (both accepted, T00002); on Nov 5 another end for T00001
// (already_cancelled); 45 days.
const dev12Fixture = "../../testdata/sessions/mini-seed37-dev12.session.json"

func TestDev12RecordReplays(t *testing.T) {
	_, rep, err := Load(dev12Fixture, ReplayOptions{})
	if err != nil || !rep.OK || !rep.Earlier || rep.Compatibility() != "earlier_build_exact" || rep.Replayed != rep.Recorded {
		t.Fatalf("%v %s", err, repString(rep))
	}
	for _, step := range []simtime.Duration{37 * simtime.Minute, 7 * simtime.Hour} {
		if _, rep, err := Load(dev12Fixture, ReplayOptions{Step: step}); err != nil || rep.Replayed != rep.Recorded {
			t.Fatalf("step %s: %v %s", step, err, repString(rep))
		}
	}
	raw, _ := os.ReadFile(dev12Fixture)
	r, err := DecodeRecord(raw)
	if err != nil {
		t.Fatal(err)
	}
	if r.Engine != "0.2.0-dev.12" || r.Batches[0].Results[0].ContractID != "T00001" || r.Batches[1].Results[1].ContractID != "T00002" ||
		r.Batches[2].Results[0].Code != "already_cancelled" {
		t.Fatalf("fixture: %s %+v", r.Engine, r.Batches)
	}
}

// A loaded dev.12 session continues under this build: it saves with the
// record embedded verbatim (batch mode v0.2_dev12) and reloads exactly.
func TestDev12RecordContinues(t *testing.T) {
	l, _, err := Load(dev12Fixture, ReplayOptions{})
	if err != nil {
		t.Fatal(err)
	}
	at(t, l, 50, "09:00")
	if res := l.Apply(teleStart("tele-c", "saturday", "2026-12-28")); res[0].Code != "contract_overlap" {
		t.Fatalf("%+v", res)
	}
	at(t, l, 80, "06:00")
	p := saveTemp(t, l)
	raw, _ := os.ReadFile(p)
	cont, err := DecodeRecord(raw)
	if err != nil || cont.Legacy == nil || cont.Legacy.Engine != "0.2.0-dev.12" || cont.Legacy.BatchMode != "v0.2_dev12" ||
		cont.Legacy.Outcome != CompatExact || cont.Legacy.State != CompatExact || len(cont.Batches) != 1 {
		t.Fatalf("continued record: %v %+v", err, cont.Legacy)
	}
	fixture, _ := os.ReadFile(dev12Fixture)
	var a, b bytes.Buffer
	json.Compact(&a, fixture)
	json.Compact(&b, cont.Legacy.Record)
	if a.String() != b.String() {
		t.Fatal("the dev.12 record is not embedded verbatim")
	}
	for _, step := range []simtime.Duration{0, 41 * simtime.Minute} {
		l2, rep, err := Load(p, ReplayOptions{Step: step})
		if err != nil || rep.Compatibility() != "prefix:earlier_build_exact+native_exact" || l2.Digests() != l.Digests() {
			t.Fatalf("continued load (step %s): %v %s", step, err, repString(rep))
		}
	}
}
