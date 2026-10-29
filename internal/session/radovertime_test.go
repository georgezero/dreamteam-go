package session

import (
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/scenario"
)

// V0.7a: a session under the radiologist overtime policy "salary_rate"
// embeds the policy in its scenario, saves and replays exactly, and differs
// from the same run without it only in money. "unpaid" spelled out is
// today's behaviour: the same log, ledger, actions and state as the policy
// omitted (only the scenario fingerprint, and so the run digest, differ).
func TestRadOvertimeSessionReplays(t *testing.T) {
	policy := func(p string) func(*scenario.File) {
		return func(f *scenario.File) { f.Policies.RadiologistOvertime = p }
	}
	run := func(s *Session) {
		at(t, s, 0, "07:00")
		if r := s.Apply(api.Action{Type: api.ActExtendShift, Employee: "E07", Minutes: 120}); !r[0].OK {
			t.Fatal(r[0].Message)
		}
		at(t, s, 3, "12:00")
	}
	omitted, unpaid, paid := openFile(t, "mini", 4, nil), openFile(t, "mini", 4, policy("unpaid")), openFile(t, "mini", 4, policy("salary_rate"))
	for _, s := range []*Session{omitted, unpaid, paid} {
		run(s)
	}
	a, b, c := omitted.Digests(), unpaid.Digests(), paid.Digests()
	if a.Log != b.Log || a.Ledger != b.Ledger || a.Actions != b.Actions || a.State != b.State || a.Run == b.Run {
		t.Fatalf("unpaid differs from omitted: %+v %+v", a, b)
	}
	// The acceptance message, part of the action log, carries the
	// estimate, so the action digest differs too.
	if c.Ledger == a.Ledger || c.State == a.State || c.Actions == a.Actions {
		t.Fatalf("salary_rate: %+v", c)
	}
	p := saveTemp(t, paid)
	l, rep, err := Load(p, ReplayOptions{})
	if err != nil || !rep.OK || l.Digests() != c {
		t.Fatalf("%v %s", err, repString(rep))
	}
	// Continued, the loaded session and the original stay identical.
	for _, s := range []*Session{paid, l} {
		at(t, s, 5, "08:00")
	}
	if paid.Digests() != l.Digests() {
		t.Fatal("the loaded session diverged")
	}
}
