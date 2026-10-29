package sim

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// The ledger (SIMULATION_DESIGN §2.13, §8): a double-entry journal in
// integer cents. Every entry balances (Σ debits = Σ credits), carries a
// stable idempotency key built from simulation identifiers (study, claim,
// employee and day indices, never wall-clock or processing order), and is
// never edited after posting: a correction is an explicit reversing entry.
// Operational events and journal entries stay separate; an entry's key
// names the operational record it comes from.

type account uint8

const (
	acctCash account = iota
	acctAR
	acctRevenue  // patient service revenue at expected net
	acctVariance // collection variance (contra-revenue)
	acctWages    // payroll expense
	acctBenefits // benefits and payroll tax expense
	acctFacility
	acctEquipment
	acctSupplies
	acctIT
	acctBilling
	acctAccruedPayroll
	acctAP
	acctEquity
	// V0.2 (appended; V0.1 accounts keep their numbers): recruiting fees,
	// severance (with its benefits) and severance owed until its payday.
	acctRecruiting
	acctSeverance
	acctSeverancePayable
	// V0.2 Slice 5B (appended): teleradiology retainers and remote read
	// fees, both owed to the vendor through accounts payable.
	acctRemoteRetainer
	acctRemoteReads
	// V0.7b (appended): the credit line's drawn balance and its interest,
	// both only under a scenario's finance.solvency.
	acctCreditLine
	acctInterest
	numAccounts
)

// numV3Accounts is the chart the state/v3 encoding was defined with:
// accounts appended after it are encoded only when non-zero (digest.go).
const numV3Accounts = acctRemoteRetainer

// numV5Accounts is the chart before V0.7b: the accounts appended after it
// are encoded, after their own omitTag, only when non-zero (digest.go).
const numV5Accounts = acctCreditLine

// numLegacyAccounts is the V0.1 chart of accounts: its accounts are
// always listed; a V0.2 account only once it has a balance (so a run
// without V0.2 actions lists exactly the V0.1 chart).
const numLegacyAccounts = acctRecruiting

type acctKind uint8

const (
	kindAsset acctKind = iota
	kindLiability
	kindEquity
	kindRevenue
	kindContra
	kindExpense
)

var accountInfo = [numAccounts]struct {
	id, name string
	kind     acctKind
}{
	acctCash:             {"cash", "Cash", kindAsset},
	acctAR:               {"accounts_receivable", "Accounts receivable", kindAsset},
	acctRevenue:          {"patient_service_revenue", "Patient service revenue (net expected)", kindRevenue},
	acctVariance:         {"collection_variance", "Collection variance", kindContra},
	acctWages:            {"payroll", "Payroll expense", kindExpense},
	acctBenefits:         {"benefits", "Benefits & payroll tax expense", kindExpense},
	acctFacility:         {"facility", "Rent & facility", kindExpense},
	acctEquipment:        {"equipment", "Equipment service & leases", kindExpense},
	acctSupplies:         {"supplies", "Medical supplies & contrast", kindExpense},
	acctIT:               {"it", "IT / PACS", kindExpense},
	acctBilling:          {"billing", "Billing service fees", kindExpense},
	acctAccruedPayroll:   {"accrued_payroll", "Accrued payroll", kindLiability},
	acctAP:               {"accounts_payable", "Accounts payable", kindLiability},
	acctEquity:           {"opening_equity", "Opening equity", kindEquity},
	acctRecruiting:       {"recruiting", "Recruiting fees", kindExpense},
	acctSeverance:        {"severance", "Severance (with benefits)", kindExpense},
	acctSeverancePayable: {"severance_payable", "Severance payable", kindLiability},
	acctRemoteRetainer:   {"remote_retainer", "Teleradiology retainers", kindExpense},
	acctRemoteReads:      {"remote_reads", "Teleradiology read fees", kindExpense},
	acctCreditLine:       {"credit_line", "Credit line drawn", kindLiability},
	acctInterest:         {"interest", "Interest on the credit line", kindExpense},
}

// listed reports whether account a appears in a chart listing.
func (l *ledger) listed(a account) bool { return a < numLegacyAccounts || l.bal[a] != 0 }

// maxMods bounds the modality dimension (XR, US, CT, MR).
const maxMods = 4

// line is one posting: amt > 0 debits the account, amt < 0 credits it.
// mod tags the line with a modality (index into state.mods), or -1; item
// with the fixed cost it accrues (index into FinanceConfig.Fixed), or -1,
// so costs that share an account (rent, utilities and insurance under
// Rent & facility) stay separable.
type line struct {
	acct account
	mod  int8
	amt  int64
	item int16
}

// entryKind names the business transaction an entry records.
type entryKind uint8

const (
	entOpening      entryKind = iota + 1 // opening cash (a: 0)
	entRevenue                           // report final (a: study)
	entStudyCost                         // supplies and per-study IT at exam end (a: study)
	entPayment                           // payment received (a: claim, b: payment number)
	entSettle                            // claim settles: residual to collection variance (a: claim)
	entWages                             // hourly pay for a worked day (a: employee, b: day)
	entSalary                            // salaried daily accrual (a: employee, b: day)
	entFixed                             // daily fixed-cost accrual (a: day)
	entPayroll                           // payroll paid (a: day)
	entPayAP                             // accounts payable paid (a: day)
	entReversal                          // reverses entry a
	entGuarantee                         // guaranteed-pay provision change at day close (a: employee, b: day)
	entRecruiting                        // recruiting fee at hire acceptance (a: employee)
	entSeverance                         // severance recognized at termination acceptance (a: employee)
	entSeverancePay                      // severance paid on its payday (a: employee)
	entRetainer                          // teleradiology retainer for an active day (a: contract×3 + window, b: day)
	entRemoteRead                        // teleradiology read fee at report final (a: study, b: remote claim)
	entRadOvertime                       // a salaried radiologist's overtime for a worked day, policy salary_rate (a: employee, b: day)
	entCreditDraw                        // credit line drawn after a disbursement (a: day, b: 0 payroll, 1 payables, 2+employee severance)
	entCreditRepay                       // credit line repaid at 00:00 (a: day)
	entInterest                          // a day's interest on the credit line (a: day)
)

// entryKey is an entry's idempotency key.
type entryKey struct {
	kind entryKind
	a, b int64
}

type entry struct {
	at       simtime.Time
	key      entryKey
	lines    []line
	reversed int // index of the entry reversing this one, -1 if none
}

func journalID(i int) string { return pubID('J', int64(i+1), 6) }

// dayBook holds one day's postings by account and modality, and the day's
// gross cash movements, so period statements need no journal scan.
type dayBook struct {
	net      [numAccounts]int64
	byMod    [numAccounts][maxMods]int64
	receipts int64   // cash in from payments
	paidOut  int64   // cash out: payroll and payables (credit draws and repayments are neither)
	fixed    int64   // expense posted by fixed-cost accruals (and their reversals)
	items    []int64 // fixed-cost expense by item
}

type ledger struct {
	entries []entry
	keys    map[entryKey]int
	bal     [numAccounts]int64 // debit-positive running balances
	days    []dayBook          // by day index
	cal     simtime.Calendar
	// verified counts the entries the incremental reconciliation
	// (check) has re-summed; shadow is its independent running balance.
	verified int
	shadow   [numAccounts]int64
	// win, once opened at the takeover, books every entry posted after it
	// up to winEnd (0: open-ended): the evaluation window (evaluation.go).
	win    *dayBook
	winEnd simtime.Time
}

// openWindow starts the evaluation window's book: entries posted from now
// on (the takeover's own are already in) up to end (0: no end).
func (l *ledger) openWindow(end simtime.Time) {
	l.win, l.winEnd = &dayBook{}, end
}

func newLedger(cal simtime.Calendar) ledger {
	return ledger{keys: map[entryKey]int{}, cal: cal}
}

// post records a balanced entry. A duplicate key, an unbalanced entry or
// an empty one is an error and nothing is posted.
func (l *ledger) post(at simtime.Time, key entryKey, lines []line) (int, error) {
	if _, dup := l.keys[key]; dup {
		return -1, fmt.Errorf("ledger: duplicate posting %v", key)
	}
	var sum int64
	nonzero := false
	for _, ln := range lines {
		sum += ln.amt
		nonzero = nonzero || ln.amt != 0
		if ln.acct >= numAccounts || ln.mod >= maxMods {
			return -1, fmt.Errorf("ledger: bad line %+v", ln)
		}
	}
	if sum != 0 || !nonzero {
		return -1, fmt.Errorf("ledger: entry %v does not balance (Σ = %d) or is empty", key, sum)
	}
	if key.kind == entReversal && (key.a < 0 || key.a >= int64(len(l.entries))) {
		return -1, fmt.Errorf("ledger: reversal of a missing entry %d", key.a)
	}
	day := l.cal.DayIndex(at)
	if n := len(l.entries); n > 0 && at < l.entries[n-1].at {
		return -1, fmt.Errorf("ledger: entry %v posted at %d before the previous entry", key, at)
	}
	for int64(len(l.days)) <= day {
		l.days = append(l.days, dayBook{})
	}
	kind := key.kind
	if kind == entReversal {
		kind = l.entries[key.a].key.kind
	}
	l.book(&l.days[day], key, kind, lines)
	if l.win != nil && (l.winEnd == 0 || at <= l.winEnd) {
		l.book(l.win, key, kind, lines)
	}
	for _, ln := range lines {
		l.bal[ln.acct] += ln.amt
	}
	i := len(l.entries)
	l.entries = append(l.entries, entry{at: at, key: key, lines: lines, reversed: -1})
	l.keys[key] = i
	return i, nil
}

// book adds an entry's lines to a day book.
func (l *ledger) book(b *dayBook, key entryKey, kind entryKind, lines []line) {
	for _, ln := range lines {
		if kind == entFixed && accountInfo[ln.acct].kind == kindExpense {
			b.fixed += ln.amt
		}
		if ln.item >= 0 {
			for len(b.items) <= int(ln.item) {
				b.items = append(b.items, 0)
			}
			b.items[ln.item] += ln.amt
		}
		b.net[ln.acct] += ln.amt
		if ln.mod >= 0 {
			b.byMod[ln.acct][ln.mod] += ln.amt
		}
		if ln.acct == acctCash && key.kind != entOpening && kind != entCreditDraw && kind != entCreditRepay {
			if ln.amt > 0 {
				b.receipts += ln.amt
			} else {
				b.paidOut -= ln.amt
			}
		}
	}
}

// mustPost posts and panics on error: the simulation never builds a bad
// entry, so one is a bug.
func (l *ledger) mustPost(at simtime.Time, key entryKey, lines ...line) {
	if _, err := l.post(at, key, lines); err != nil {
		panic(err)
	}
}

// reverse posts the mirror image of entry i at time at. History is never
// rewritten; an entry can be reversed once.
func (l *ledger) reverse(at simtime.Time, i int) (int, error) {
	if i < 0 || i >= len(l.entries) {
		return -1, fmt.Errorf("ledger: no entry %d", i)
	}
	orig := &l.entries[i]
	if orig.key.kind == entReversal {
		return -1, fmt.Errorf("ledger: %s is itself a reversal", journalID(i))
	}
	if orig.reversed >= 0 {
		return -1, fmt.Errorf("ledger: %s already reversed by %s", journalID(i), journalID(orig.reversed))
	}
	mirror := make([]line, len(orig.lines))
	for k, ln := range orig.lines {
		mirror[k] = line{acct: ln.acct, mod: ln.mod, amt: -ln.amt, item: ln.item}
	}
	r, err := l.post(at, entryKey{kind: entReversal, a: int64(i)}, mirror)
	if err == nil {
		l.entries[i].reversed = r
	}
	return r, err
}

// check re-sums entries posted since the last check, independently of
// post's bookkeeping, and compares: every entry balances, the shadow
// balances equal the running ones, the day books add up to them, and the
// accounts sum to zero.
func (l *ledger) check() error {
	for ; l.verified < len(l.entries); l.verified++ {
		e := &l.entries[l.verified]
		var sum int64
		for _, ln := range e.lines {
			sum += ln.amt
			l.shadow[ln.acct] += ln.amt
		}
		if sum != 0 {
			return fmt.Errorf("ledger: %s unbalanced by %d", journalID(l.verified), sum)
		}
		if k, ok := l.keys[e.key]; !ok || k != l.verified {
			return fmt.Errorf("ledger: %s key index broken", journalID(l.verified))
		}
	}
	var total int64
	for a := range numAccounts {
		if l.shadow[a] != l.bal[a] {
			return fmt.Errorf("ledger: %s balance %d, journal sums to %d", accountInfo[a].id, l.bal[a], l.shadow[a])
		}
		total += l.bal[a]
	}
	if total != 0 {
		return fmt.Errorf("ledger: accounts sum to %d", total)
	}
	return nil
}

// checkDays verifies that the day books add up to the balances and that
// cash equals opening cash plus receipts minus disbursements (O(days)).
func (l *ledger) checkDays(opening int64) error {
	var sum [numAccounts]int64
	cash := opening
	for d := range l.days {
		for a := range numAccounts {
			sum[a] += l.days[d].net[a]
		}
		cash += l.days[d].receipts - l.days[d].paidOut - l.days[d].net[acctCreditLine]
	}
	if sum != l.bal {
		return fmt.Errorf("ledger: day books %v, balances %v", sum, l.bal)
	}
	if cash != l.bal[acctCash] {
		return fmt.Errorf("ledger: cash %d, opening + receipts − disbursements %d", l.bal[acctCash], cash)
	}
	return nil
}

// hash digests the journal in posting order (time, key, lines).
func (l *ledger) hash() string {
	h := sha256.New()
	var buf [8]byte
	put := func(v int64) {
		binary.BigEndian.PutUint64(buf[:], uint64(v))
		h.Write(buf[:])
	}
	for i := range l.entries {
		e := &l.entries[i]
		put(int64(e.at))
		put(int64(e.key.kind))
		put(e.key.a)
		put(e.key.b)
		for _, ln := range e.lines {
			put(int64(ln.acct))
			put(int64(ln.mod))
			put(ln.amt)
			put(int64(ln.item))
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}

// sumDays adds the day books for days [from, to] (clipped to what exists).
func (l *ledger) sumDays(from, to int64) dayBook {
	var out dayBook
	from = max(from, 0)
	to = min(to, int64(len(l.days))-1)
	for d := from; d <= to; d++ {
		b := &l.days[d]
		for a := range numAccounts {
			out.net[a] += b.net[a]
			for k := range maxMods {
				out.byMod[a][k] += b.byMod[a][k]
			}
		}
		out.receipts += b.receipts
		out.paidOut += b.paidOut
		out.fixed += b.fixed
		for len(out.items) < len(b.items) {
			out.items = append(out.items, 0)
		}
		for k, v := range b.items {
			out.items[k] += v
		}
	}
	return out
}

// cashAtEnd is the cash balance at the end of day d.
func (l *ledger) cashAtEnd(d int64) int64 {
	var c int64
	for k := int64(0); k <= d && k < int64(len(l.days)); k++ {
		c += l.days[k].net[acctCash]
	}
	return c
}
