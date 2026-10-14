package sim

import (
	"fmt"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

func TestLedgerPostingRules(t *testing.T) {
	cal := simtime.NewCalendar(simtime.Date{Year: 2026, Month: 10, Day: 13})
	l := newLedger(cal)
	k := func(a int64) entryKey { return entryKey{kind: entRevenue, a: a} }

	if _, err := l.post(10, k(1), []line{{acctAR, 0, 500, -1}, {acctRevenue, 0, -500, -1}}); err != nil {
		t.Fatal(err)
	}
	for name, c := range map[string]struct {
		at    simtime.Time
		key   entryKey
		lines []line
		want  string
	}{
		"unbalanced": {20, k(2), []line{{acctAR, 0, 500, -1}, {acctRevenue, 0, -499, -1}}, "does not balance"},
		"empty":      {20, k(3), []line{{acctAR, 0, 0, -1}, {acctRevenue, 0, 0, -1}}, "empty"},
		"duplicate":  {20, k(1), []line{{acctAR, 0, 1, -1}, {acctRevenue, 0, -1, -1}}, "duplicate"},
		"backwards":  {5, k(4), []line{{acctAR, 0, 1, -1}, {acctRevenue, 0, -1, -1}}, "before the previous"},
		"bad mod":    {20, k(5), []line{{acctAR, maxMods, 1, -1}, {acctRevenue, 0, -1, -1}}, "bad line"},
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
	l.mustPost(40, k(9), line{acctCash, -1, 100, -1}, line{acctAR, 0, -100, -1})
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

func TestPubIDMatchesFmt(t *testing.T) {
	for _, n := range []int64{0, 1, 9, 10, 99999, 100000, 1234567, 9_999_999_999} {
		for _, w := range []int{5, 6} {
			if got, want := pubID('S', n, w), fmt.Sprintf("S%0*d", w, n); got != want {
				t.Fatalf("pubID(%d, %d) = %q, want %q", n, w, got, want)
			}
		}
	}
}

// Step 11 review finding 4: every line field, including the fixed-cost
// item, is part of the ledger hash.
func TestRegress11LedgerHashCoversItem(t *testing.T) {
	cal := simtime.NewCalendar(simtime.Date{Year: 2026, Month: 10, Day: 13})
	a, b := newLedger(cal), newLedger(cal)
	a.mustPost(0, entryKey{kind: entFixed}, line{acctFacility, -1, 100, 0}, line{acctAP, -1, -100, -1})
	b.mustPost(0, entryKey{kind: entFixed}, line{acctFacility, -1, 100, 1}, line{acctAP, -1, -100, -1})
	if a.hash() == b.hash() {
		t.Fatal("entries differing only in the fixed-cost item hash the same")
	}
}

// Review test gap: memoized draws equal fresh hashes, each call gets its
// own lane cursors, and the memo resets.
func TestDrawMemoEqualsFreshHash(t *testing.T) {
	st := &state{seed: rng.SeedFromInt(9)}
	for k := int64(0); k < 50; k++ {
		fresh := rng.Hash(st.seed, "attempt", 1, 2, 3, k, 0).Lanes()
		first := st.hash("attempt", 1, 2, 3, k, 0)
		_ = first.Uint64() // consume a cursor on the first copy
		second := st.hash("attempt", 1, 2, 3, k, 0)
		if first.Lanes() != fresh || second.Lanes() != fresh || second.Uint64() != fresh[0] {
			t.Fatalf("key %d: memoized draw differs from a fresh hash", k)
		}
	}
	if len(st.drawMemo) != 50 {
		t.Fatalf("memo holds %d", len(st.drawMemo))
	}
	clear(st.drawMemo)
	if st.hash("attempt", 1).Lanes() != rng.Hash(st.seed, "attempt", 1).Lanes() {
		t.Fatal("after reset")
	}
}

// Review test gap: the streamed log hash covers every entry, regardless of
// ring eviction or digests read in between.
func TestLogDigestIndependentOfReadsAndEviction(t *testing.T) {
	var a, b eventLog
	for i := 0; i < logRetention+500; i++ {
		text := fmt.Sprintf("entry %d", i)
		a.emit(simtime.Time(i), "info", "test", text)
		b.emit(simtime.Time(i), "info", "test", text)
		if i%97 == 0 {
			_ = b.digest()
		}
	}
	if string(a.digest()) != string(b.digest()) || len(a.ring) != logRetention {
		t.Fatal("reading the digest or evicting entries changed the hash")
	}
	cal := simtime.NewCalendar(simtime.Date{Year: 2026, Month: 10, Day: 13})
	r := a.recent(cal, 3)
	if r[0].Text != fmt.Sprintf("entry %d", logRetention+497) || r[2].Seq != uint64(logRetention+500) {
		t.Fatalf("retained order: %+v", r)
	}
}
