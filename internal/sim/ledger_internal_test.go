package sim

import (
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/simtime"
)

func TestLedgerPostingRules(t *testing.T) {
	cal := simtime.NewCalendar(simtime.Date{Year: 2026, Month: 10, Day: 13})
	l := newLedger(cal)
	k := func(a int64) entryKey { return entryKey{kind: entRevenue, a: a} }

	if _, err := l.post(10, k(1), []line{{acctAR, 0, 500}, {acctRevenue, 0, -500}}); err != nil {
		t.Fatal(err)
	}
	for name, c := range map[string]struct {
		at    simtime.Time
		key   entryKey
		lines []line
		want  string
	}{
		"unbalanced": {20, k(2), []line{{acctAR, 0, 500}, {acctRevenue, 0, -499}}, "does not balance"},
		"empty":      {20, k(3), []line{{acctAR, 0, 0}, {acctRevenue, 0, 0}}, "empty"},
		"duplicate":  {20, k(1), []line{{acctAR, 0, 1}, {acctRevenue, 0, -1}}, "duplicate"},
		"backwards":  {5, k(4), []line{{acctAR, 0, 1}, {acctRevenue, 0, -1}}, "before the previous"},
		"bad mod":    {20, k(5), []line{{acctAR, maxMods, 1}, {acctRevenue, 0, -1}}, "bad line"},
	} {
		n := len(l.entries)
		_, err := l.post(c.at, c.key, c.lines)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err %v, want %q", name, err, c.want)
		}
		if len(l.entries) != n || l.bal[acctAR] != 500 {
			t.Errorf("%s: a rejected entry changed the ledger", name)
		}
	}

	// Corrections are explicit reversing entries; history is unchanged.
	before := l.entries[0]
	r, err := l.reverse(30, 0)
	if err != nil {
		t.Fatal(err)
	}
	if l.bal[acctAR] != 0 || l.bal[acctRevenue] != 0 || l.entries[0].lines[0] != before.lines[0] {
		t.Fatalf("reversal: balances %v, original %v", l.bal, l.entries[0])
	}
	if l.entries[r].key != (entryKey{kind: entReversal, a: 0}) || l.entries[0].reversed != r {
		t.Fatalf("reversal not linked: %+v", l.entries[r])
	}
	if _, err := l.reverse(31, 0); err == nil {
		t.Error("an entry was reversed twice")
	}
	if _, err := l.reverse(31, r); err == nil {
		t.Error("a reversal was reversed")
	}
	if err := l.check(); err != nil {
		t.Fatal(err)
	}
	if err := l.checkDays(0); err != nil {
		t.Fatal(err)
	}
	h := l.hash()
	l.mustPost(40, k(9), line{acctCash, -1, 100}, line{acctAR, 0, -100})
	if l.hash() == h {
		t.Error("hash ignores a new entry")
	}
	if l.days[0].receipts != 100 {
		t.Errorf("receipts %d", l.days[0].receipts)
	}
	// The incremental reconciliation catches a tampered balance.
	l.bal[acctCash]++
	if err := l.check(); err == nil {
		t.Error("check missed a balance that disagrees with the journal")
	}
}
