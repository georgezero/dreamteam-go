package sim

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Teleradiology contracts (V0.2 Slice 5B: contract_teleradiology;
// V0.2_DESIGN §6.3, §8.2, §8.4, §9, §5.13). The remote reading engine is
// Slice 5A's (remote.go); this file is the action, the contract terms and
// the money.
//
//   - Catalog. One plan, standard-v1 (§8.2): monthly retainers weeknights
//     $1,500, saturday $800, sunday $800 (the selected windows add); per
//     finalized read XR $12, US $30, CT $45, MR $60; +20% for a study
//     claimed at ≥ 22:00 or < 08:00. Terms are copied into the contract at
//     acceptance and never change.
//   - op start: a known plan, one to three windows, scope urgent or all,
//     activation at least 14 days ahead (within the booking horizon). The
//     contract is open-ended. Same requestId and content: unchanged;
//     other content: request_conflict.
//   - op cancel: fixes an open-ended contract's exclusive end, at least 30
//     days ahead and at least 30 days after activation (the minimum term,
//     payable even if the cancellation comes before activation). The same
//     end again: unchanged; another: already_cancelled. Claims already
//     made complete at their locked fees.
//   - Contracts never overlap (a successor starts at or after its
//     predecessor's end), checked on the timeline the whole batch leaves;
//     they share the practice's two lanes, which no contract adds to.
//   - Money (§8.4, §9). Each active date at 00:00, after the day's
//     payables are paid and the fixed costs accrue, each selected window's
//     retainer accrues floor(M×d/n) − floor(M×(d−1)/n) (day d of an n-day
//     month): debit remote retainer, credit A/P. A claim locks its fee;
//     the fee is recognized at the report's final: debit remote reads,
//     credit A/P. Both are paid by the existing first-of-month A/P
//     payment, which pays everything accrued before it: an expiry never
//     erases an unpaid bill. Report revenue stays at final (finalize).

// teleMods are the catalog modalities, in fee order.
var teleMods = [...]string{"XR", "US", "CT", "MR"}

// Contract rules (§6.3, §8.2).
const (
	teleNoticeDays  = 14 // activation at least this far ahead
	teleMinTermDays = 30 // end at least this long after activation
	teleCancelDays  = 30 // end at least this far ahead
	teleTrailDays   = 28 // the previews' trailing observed demand
)

// telePlan is a catalog plan's terms.
type telePlan struct {
	id       string
	retainer [3]int64 // monthly cents by window, catalog order
	fees     [len(teleMods)]int64
	nightBP  int64
}

var telePlans = []telePlan{
	{id: "standard-v1", retainer: [3]int64{150_000, 80_000, 80_000}, fees: [len(teleMods)]int64{1_200, 3_000, 4_500, 6_000}, nightBP: 2_000},
}

var teleWindowHours = [...]string{"Mon–Fri 18:00 to 08:00 next day", "Sat 14:00–24:00", "Sun 00:00–24:00"}

// teleNight reports whether a claim at local clock c pays the night
// surcharge.
func teleNight(c simtime.Duration) bool { return c >= 22*simtime.Hour || c < 8*simtime.Hour }

// teleMod is exam x's catalog modality index, or −1 (not readable
// remotely).
func (st *state) teleMod(x int) int {
	return slices.Index(teleMods[:], st.cfg.Catalog[x].Modality)
}

// withTerms is k with plan p's terms for its windows.
func withTerms(k teleContract, p *telePlan) teleContract {
	k.plan, k.fees, k.nightBP = p.id, p.fees, p.nightBP
	for w := range k.retainer {
		k.retainer[w] = 0
		if k.windows&(1<<w) != 0 {
			k.retainer[w] = p.retainer[w]
		}
	}
	return k
}

func (k *teleContract) monthly() int64 { return k.retainer[0] + k.retainer[1] + k.retainer[2] }

// active reports whether contract k is in force on day.
func (k *teleContract) active(day int64) bool { return day >= k.from && (k.until < 0 || day < k.until) }

// retainerOn is contract k's retainer accrual on day (0 if not active).
func (st *state) retainerOn(k *teleContract, day int64) int64 {
	if !k.active(day) {
		return 0
	}
	d := st.cal.Date(st.cal.StartOfDay(day))
	var n int64
	for w := range k.retainer {
		n += DailyAccrual(k.retainer[w], d.Day, d.DaysInMonth())
	}
	return n
}

// retainerBetween is contract k's retainer accrued on days [from, to).
func (st *state) retainerBetween(k *teleContract, from, to int64) int64 {
	var n int64
	for d := from; d < to; d++ {
		n += st.retainerOn(k, d)
	}
	return n
}

// teleEntry is one accepted contract operation, in order.
type teleEntry struct {
	op       string
	contract int
	day      int64 // start: the activation; cancel: the exclusive end
	action   int
	batch    int64
	at       simtime.Time
}

// teleSpec is a validated contract_teleradiology item, not yet committed.
type teleSpec struct {
	op       string
	contract int          // the contract (start: the index it will get)
	k        teleContract // start: the terms
	until    int64        // cancel: the end
	prevEnd  int64
}

// telesOf is the contract timeline as the batch so far proposes it.
func (p *proposal) telesOf(st *state) []teleContract {
	if p.tcons == nil && st.tele != nil {
		p.tcons = slices.Clone(st.tele.contracts)
	}
	return p.tcons
}

// teleCanon is a start request's canonical content: windows in catalog
// order.
func teleCanon(a api.Action, w teleWindows) string {
	c := api.Action{Type: a.Type, Op: a.Op, RequestID: a.RequestID, Plan: a.Plan, Windows: strings.Split(w.String(), ", "), Scope: a.Scope, Effective: a.Effective}
	raw, err := json.Marshal(c)
	if err != nil {
		panic(err)
	}
	return string(raw)
}

// parseTeleWindows parses a start's windows: one to three distinct
// catalog windows.
func parseTeleWindows(ws []string) (teleWindows, bool) {
	var w teleWindows
	for _, n := range ws {
		k := slices.Index(teleWindowNames[:], n)
		if k < 0 || w&(1<<k) != 0 {
			return 0, false
		}
		w |= 1 << k
	}
	return w, w != 0
}

// teleFields reports whether a sets a contract_teleradiology field.
func teleFields(a api.Action) bool {
	return a.Plan != "" || a.Windows != nil || a.Scope != "" || a.Contract != "" || a.EndDate != ""
}

// findTele is the contract with public ID id, or −1.
func (st *state) findTele(id string) int {
	if st.tele == nil {
		return -1
	}
	for c := range st.tele.contracts {
		if teleID(c) == id {
			return c
		}
	}
	return -1
}

// checkTeleAction validates one contract_teleradiology item against the
// state and the batch's earlier items.
func (st *state) checkTeleAction(a api.Action, p *proposal) staffOutcome {
	bad := func(code, format string, args ...any) staffOutcome {
		return staffOutcome{code: code, msg: fmt.Sprintf(format, args...)}
	}
	switch {
	case a.Employee != "" || a.Minutes != 0 || a.Date != "" || a.Policy != "" || a.Cutoff != "" || a.Role != "" || a.Skills != nil ||
		a.RateCents != 0 || a.ContractedWeeklyMinutes != 0 || a.Roster != nil || a.StartDate != "" || a.LastDay != "" || a.SiteAllocationBP != 0:
		return bad("invalid_schema", "contract_teleradiology takes op and, op start, requestId, plan, windows, scope and effective, or, op cancel, contract and endDate.")
	case a.Op == api.TeleOpStart:
		if a.Contract != "" || a.EndDate != "" {
			return bad("invalid_schema", "contract_teleradiology op start takes requestId, plan, windows, scope and effective only.")
		}
		return st.checkTeleStart(a, p)
	case a.Op == api.TeleOpCancel:
		if a.RequestID != "" || a.Plan != "" || a.Windows != nil || a.Scope != "" || a.Effective != "" {
			return bad("invalid_schema", "contract_teleradiology op cancel takes contract and endDate only.")
		}
		return st.checkTeleCancel(a, p)
	}
	return bad("invalid_schema", "contract_teleradiology needs op %q or %q.", api.TeleOpStart, api.TeleOpCancel)
}

func (st *state) checkTeleStart(a api.Action, p *proposal) staffOutcome {
	bad := func(code, format string, args ...any) staffOutcome {
		return staffOutcome{code: code, msg: fmt.Sprintf(format, args...)}
	}
	switch {
	case a.RequestID == "" || len(a.RequestID) > 128:
		return bad("invalid_schema", "contract_teleradiology op start needs a requestId of 1–128 characters.")
	case a.Plan == "" || a.Windows == nil || a.Scope == "" || a.Effective == "":
		return bad("invalid_schema", "contract_teleradiology op start needs plan, windows, scope and effective.")
	case p.treqs[a.RequestID]:
		return bad("invalid_parameters", "requestId %q appears twice in this batch.", a.RequestID)
	}
	w, wok := parseTeleWindows(a.Windows)
	canon := ""
	if wok {
		canon = teleCanon(a, w)
	}
	if st.tele != nil {
		for c := range st.tele.contracts {
			k := &st.tele.contracts[c]
			if k.req != a.RequestID {
				continue
			}
			if k.canon == canon {
				return staffOutcome{ok: true, code: api.ActionUnchanged, contract: teleID(c),
					msg: fmt.Sprintf("Teleradiology contract %q was already accepted as %s: nothing changed.", a.RequestID, teleID(c))}
			}
			return bad("request_conflict", "requestId %q was already used for %s with different terms.", a.RequestID, teleID(c))
		}
	}
	pi := slices.IndexFunc(telePlans, func(x telePlan) bool { return x.id == a.Plan })
	if pi < 0 {
		var ids []string
		for _, x := range telePlans {
			ids = append(ids, x.id)
		}
		return bad("unknown_plan", "No teleradiology plan %q: the catalog has %s.", a.Plan, strings.Join(ids, ", "))
	}
	if !wok {
		return bad("invalid_window", "windows must be one to three distinct windows of %s, not %v.", strings.Join(teleWindowNames[:], ", "), a.Windows)
	}
	var scope teleScope
	switch a.Scope {
	case "all":
		scope = teleScopeAll
	case "urgent":
		scope = teleScopeUrgent
	default:
		return bad("invalid_scope", "scope must be \"urgent\" or \"all\", not %q.", a.Scope)
	}
	date, err := simtime.ParseDate(a.Effective)
	if err != nil {
		return bad("invalid_date", "effective %q is not YYYY-MM-DD.", a.Effective)
	}
	from := st.cal.DayIndex(st.cal.StartOfDate(date))
	switch {
	case from <= st.day:
		return bad("past_date", "effective %s is not after today.", a.Effective)
	case from < st.day+teleNoticeDays:
		return bad("notice_too_short", "A teleradiology contract needs %d days' notice: the earliest activation is %s.", teleNoticeDays,
			st.cal.Date(st.cal.StartOfDay(st.day+teleNoticeDays)))
	case from > st.day+st.cfg.Policies.BookingHorizonDays:
		return bad("beyond_horizon", "effective %s is beyond the %d-day booking horizon.", a.Effective, st.cfg.Policies.BookingHorizonDays)
	}
	k := withTerms(teleContract{windows: w, scope: scope, from: from, until: -1, req: a.RequestID, canon: canon, action: -1, endAction: -1}, &telePlans[pi])
	cons := p.telesOf(st)
	ts := &teleSpec{op: api.TeleOpStart, contract: len(cons), k: k}
	p.tcons = append(cons, k)
	p.treqs[a.RequestID] = true
	p.teles = append(p.teles, ts)
	return staffOutcome{ok: true, code: api.ActionAccepted, tele: ts, effective: date.String(), msg: st.teleText(ts, false)}
}

func (st *state) checkTeleCancel(a api.Action, p *proposal) staffOutcome {
	bad := func(code, format string, args ...any) staffOutcome {
		return staffOutcome{code: code, msg: fmt.Sprintf(format, args...)}
	}
	if a.Contract == "" || a.EndDate == "" {
		return bad("invalid_schema", "contract_teleradiology op cancel needs contract and endDate.")
	}
	c := st.findTele(a.Contract)
	if c < 0 {
		return bad("unknown_contract", "No teleradiology contract %q.", a.Contract)
	}
	if p.tcancel[c] {
		return bad("invalid_parameters", "%s is cancelled twice in this batch.", a.Contract)
	}
	date, err := simtime.ParseDate(a.EndDate)
	if err != nil {
		return bad("invalid_date", "endDate %q is not YYYY-MM-DD.", a.EndDate)
	}
	until := st.cal.DayIndex(st.cal.StartOfDate(date))
	k := &st.tele.contracts[c]
	if k.until >= 0 {
		end := st.cal.Date(st.cal.StartOfDay(k.until))
		if k.until == until {
			return staffOutcome{ok: true, code: api.ActionUnchanged, contract: a.Contract,
				msg: fmt.Sprintf("%s already ends %s: nothing changed.", a.Contract, end)}
		}
		return bad("already_cancelled", "%s already ends %s: a fixed end cannot be moved or withdrawn.", a.Contract, end)
	}
	earliest := max(st.day+teleCancelDays, k.from+teleMinTermDays)
	switch {
	case until <= st.day:
		return bad("past_date", "endDate %s is not after today.", a.EndDate)
	case until < st.day+teleCancelDays:
		return bad("notice_too_short", "Ending a teleradiology contract needs %d days' notice: the earliest end is %s.", teleCancelDays,
			st.cal.Date(st.cal.StartOfDay(earliest)))
	case until < k.from+teleMinTermDays:
		return bad("minimum_term", "%s runs at least %d days from its activation on %s: the earliest end is %s.", a.Contract, teleMinTermDays,
			st.cal.Date(st.cal.StartOfDay(k.from)), st.cal.Date(st.cal.StartOfDay(earliest)))
	}
	cons := p.telesOf(st)
	ts := &teleSpec{op: api.TeleOpCancel, contract: c, k: cons[c], until: until, prevEnd: cons[c].until}
	cons[c].until = until
	ts.k.until = until
	p.tcancel[c] = true
	p.teles = append(p.teles, ts)
	return staffOutcome{ok: true, code: api.ActionAccepted, tele: ts, contract: a.Contract, effective: date.String(), msg: st.teleText(ts, false)}
}

// checkTeleTimeline checks the contract timeline the whole batch leaves:
// no two contracts overlap. Only pairs the batch touches are checked. It
// returns the refused item (the batch's first item on either contract of
// the overlapping pair) and its outcome, or −1.
func (st *state) checkTeleTimeline(p *proposal, out []staffOutcome) (int, staffOutcome) {
	cons := p.tcons
	touched := map[int]bool{}
	for _, ts := range p.teles {
		touched[ts.contract] = true
	}
	end := func(k *teleContract) int64 {
		if k.until < 0 {
			return noEnd
		}
		return k.until
	}
	for i := range cons {
		for j := i + 1; j < len(cons); j++ {
			a, b := &cons[i], &cons[j]
			if !touched[i] && !touched[j] || a.from >= end(b) || b.from >= end(a) {
				continue
			}
			first, second := a, b
			fi, si := i, j
			if b.from < a.from {
				first, second, fi, si = b, a, j, i
			}
			// The refused item: the batch's item on the later contract,
			// else on the earlier one.
			at := slices.IndexFunc(out, func(o staffOutcome) bool { return o.tele != nil && o.tele.contract == si })
			if at < 0 {
				at = slices.IndexFunc(out, func(o staffOutcome) bool { return o.tele != nil && o.tele.contract == fi })
			}
			name := func(k *teleContract) string {
				if c := slices.IndexFunc(cons, func(x teleContract) bool { return x.req == k.req }); st.tele != nil && c < len(st.tele.contracts) {
					return teleID(c)
				}
				return fmt.Sprintf("new contract %q", k.req)
			}
			until := "open-ended"
			if first.until >= 0 {
				until = "to " + st.cal.Date(st.cal.StartOfDay(first.until)).String()
			}
			return at, staffOutcome{code: "contract_overlap", msg: fmt.Sprintf("Teleradiology contracts cannot overlap: %s (from %s, %s) and %s (from %s). A successor starts on or after its predecessor's end: cancel the predecessor first or in the same batch.",
				name(first), st.cal.Date(st.cal.StartOfDay(first.from)), until, name(second), st.cal.Date(st.cal.StartOfDay(second.from)))}
		}
	}
	return -1, staffOutcome{}
}

// commitTele records a contract start or cancellation. The coverage-edge
// wake is rescheduled once for the batch (staffBatch).
func (st *state) commitTele(ts *teleSpec, action int) string {
	if st.tele == nil {
		st.tele = &teleState{}
		for l := range st.tele.lanes {
			st.tele.lanes[l].claim = -1
		}
	}
	t := st.tele
	day := ts.until
	if ts.op == api.TeleOpStart {
		k := ts.k
		k.action = action
		ts.contract = len(t.contracts)
		t.contracts = append(t.contracts, k)
		day = k.from
	} else {
		t.contracts[ts.contract].until = ts.until
		t.contracts[ts.contract].endAction = action
	}
	t.log = append(t.log, teleEntry{op: ts.op, contract: ts.contract, day: day, action: action, batch: st.actions.batches, at: st.now})
	st.emit(st.now, api.Info, "actions", st.teleText(ts, true))
	return teleID(ts.contract)
}

// teleText describes a contract item (done: once committed).
func (st *state) teleText(ts *teleSpec, done bool) string {
	k := &ts.k
	date := func(d int64) string { return st.cal.Date(st.cal.StartOfDay(d)).String() }
	if ts.op == api.TeleOpCancel {
		head := "Teleradiology contract end: "
		if done {
			head = "Teleradiology contract ending: "
		}
		g := st.retainerBetween(k, max(st.day+1, k.from), ts.until)
		return fmt.Sprintf("%s%s ends %s (last covered day %s) · retainer %s guaranteed from tomorrow to the end · claims in flight complete at their locked fees.",
			head, teleID(ts.contract), date(ts.until), date(ts.until-1), money(g))
	}
	who := "Teleradiology contract"
	if done {
		who += " " + teleID(ts.contract)
	}
	var fees []string
	for m, f := range k.fees {
		fees = append(fees, fmt.Sprintf("%s %s", teleMods[m], rate(f)))
	}
	return fmt.Sprintf("%s (%s): %s · %s · from %s, open-ended (earliest end %s) · retainer %s a month, accrued daily · per read %s, +%s claimed 22:00–08:00 · two lanes for all contracts, %s handoff · payable on the 1st.",
		who, k.plan, k.windows, k.scope, date(k.from), date(max(st.day+teleCancelDays, k.from+teleMinTermDays)), money(k.monthly()),
		strings.Join(fees, ", "), pct(k.nightBP), teleHandoff)
}

// accrueRetainers posts each active contract's retainer for day, one entry
// per selected window (financeDayStart, after the fixed costs), and logs
// activations and expiries.
func (st *state) accrueRetainers(now simtime.Time, day int64) {
	t := st.tele
	if t == nil {
		return
	}
	d := st.cal.Date(now)
	for c := range t.contracts {
		k := &t.contracts[c]
		switch day {
		case k.from:
			st.emit(now, api.Info, "finance", fmt.Sprintf("Teleradiology contract %s in force from today (%s, %s)", teleID(c), k.windows, k.scope))
		case k.until:
			st.emit(now, api.Info, "finance", fmt.Sprintf("Teleradiology contract %s ended at midnight: no new claims; claims in flight complete", teleID(c)))
		}
		if !k.active(day) {
			continue
		}
		for w := range k.retainer {
			amt := DailyAccrual(k.retainer[w], d.Day, d.DaysInMonth())
			if amt == 0 {
				continue
			}
			st.fin.led.mustPost(now, entryKey{kind: entRetainer, a: int64(c)*3 + int64(w), b: day},
				line{acctRemoteRetainer, -1, amt, -1}, line{acctAP, -1, -amt, -1})
			st.fin.apAccrued += amt
			t.retainer += amt
		}
	}
}

// recognizeRemoteRead posts remote claim rc's locked fee at its report's
// final.
func (st *state) recognizeRemoteRead(e *Engine, rc int) {
	t := st.tele
	cl := &t.claims[rc]
	fee := cl.fee + cl.surcharge
	if fee == 0 {
		return
	}
	s := &st.dem.studies[cl.study]
	st.fin.led.mustPost(e.Now(), entryKey{kind: entRemoteRead, a: int64(cl.study), b: int64(rc)},
		line{acctRemoteReads, st.modIndex(st.cfg.Catalog[s.exam].Modality), fee, -1}, line{acctAP, -1, -fee, -1})
	st.fin.apAccrued += fee
	t.reads += fee
}

// paidTele records that an A/P payment paid every vendor bill accrued
// before it.
func (st *state) paidTele() {
	if t := st.tele; t != nil {
		t.paid = t.retainer + t.reads
	}
}

// inFlight is the fees committed by remote claims not yet final.
func (st *state) inFlight() int64 {
	var n int64
	for _, ln := range st.tele.lanes {
		if ln.claim >= 0 {
			cl := &st.tele.claims[ln.claim]
			n += cl.fee + cl.surcharge
		}
	}
	return n
}

// checkTeleMoney verifies the vendor accounts: retainers and read fees
// recognized equal their accounts, read fees equal the locked fees of
// final remote reports, and what is unpaid is still in accounts payable.
func (st *state) checkTeleMoney() error {
	led := &st.fin.led
	t := st.tele
	if t == nil {
		if led.bal[acctRemoteRetainer] != 0 || led.bal[acctRemoteReads] != 0 {
			return fmt.Errorf("teleradiology expense with no contract")
		}
		return nil
	}
	var fees int64
	for _, cl := range t.claims {
		if st.dem.studies[cl.study].state == studyFinal {
			fees += cl.fee + cl.surcharge
		}
		c := &t.contracts[cl.contract]
		var night int64
		if cl.night {
			night = cl.fee * c.nightBP / 10_000
		}
		if cl.fee != c.fees[st.teleMod(st.dem.studies[cl.study].exam)] || cl.surcharge != night || cl.night != teleNight(simtime.ClockOf(cl.claimedAt)) {
			return fmt.Errorf("remote claim of %s not at its contract's locked price", studyID(cl.study))
		}
	}
	switch {
	case led.bal[acctRemoteRetainer] != t.retainer:
		return fmt.Errorf("remote retainer %d, recognized %d", led.bal[acctRemoteRetainer], t.retainer)
	case led.bal[acctRemoteReads] != t.reads || t.reads != fees:
		return fmt.Errorf("remote read fees %d, recognized %d, final claims %d", led.bal[acctRemoteReads], t.reads, fees)
	case t.paid > t.retainer+t.reads || t.retainer+t.reads-t.paid > -led.bal[acctAP]:
		return fmt.Errorf("vendor paid %d of %d, payables %d", t.paid, t.retainer+t.reads, -led.bal[acctAP])
	}
	return nil
}

// --- views -----------------------------------------------------------------

func (p *telePlan) view() api.TelePlan {
	x := api.TelePlan{ID: p.id, NightSurchargeBP: p.nightBP, NightFrom: "22:00", NightUntil: "08:00", NoticeDays: teleNoticeDays,
		MinTermDays: teleMinTermDays, CancelNoticeDays: teleCancelDays, Lanes: teleLanes, HandoffMin: int64(teleHandoff / simtime.Minute),
		Scopes: []string{"urgent", "all"}}
	for w, n := range teleWindowNames {
		x.Retainers = append(x.Retainers, api.WindowPrice{Window: n, Hours: teleWindowHours[w], MonthlyCents: p.retainer[w]})
	}
	x.ReadFees = feeList(p.fees)
	return x
}

func feeList(f [len(teleMods)]int64) []api.ModalityPrice {
	var out []api.ModalityPrice
	for m, c := range f {
		out = append(out, api.ModalityPrice{Modality: teleMods[m], Cents: c})
	}
	return out
}

func scopeName(s teleScope) string {
	if s == teleScopeUrgent {
		return "urgent"
	}
	return "all"
}

func windowList(w teleWindows) []string { return strings.Split(w.String(), ", ") }

// Teleradiology is the remote reading view (api.Teleradiology): the
// catalog always, contracts, lanes and money once a contract exists.
func (w *World) Teleradiology() api.Teleradiology { return w.st.teleView(w.eng.Now()) }

func (st *state) teleView(now simtime.Time) api.Teleradiology {
	var v api.Teleradiology
	for k := range telePlans {
		v.Plans = append(v.Plans, telePlans[k].view())
	}
	t := st.tele
	if t == nil {
		return v
	}
	today := st.cal.DayIndex(now)
	date := func(d int64) string { return st.cal.Date(st.cal.StartOfDay(d)).String() }
	claims := make([]int, len(t.contracts))
	for _, cl := range t.claims {
		claims[cl.contract]++
	}
	for c := range t.contracts {
		k := &t.contracts[c]
		x := api.TeleContract{ID: teleID(c), Plan: k.plan, RequestID: k.req, Windows: windowList(k.windows), Scope: scopeName(k.scope),
			Effective: date(k.from), MinimumTermEnd: date(k.from + teleMinTermDays), MonthlyRetainerCents: k.monthly(),
			ReadFees: feeList(k.fees), NightSurchargeBP: k.nightBP, Claims: claims[c], Action: actionIDOr(k.action), CancelAction: actionIDOr(k.endAction)}
		switch {
		case today < k.from:
			x.Status = "pending"
		case k.active(today):
			x.Status = "active"
		default:
			x.Status = "ended"
		}
		end := k.until
		if k.until >= 0 {
			x.EndDate = date(k.until)
		} else {
			end = max(today+teleCancelDays, k.from+teleMinTermDays)
			x.EarliestEndDate = date(end)
		}
		x.RetainerCents = st.retainerBetween(k, k.from, min(today+1, end))
		v.Contracts = append(v.Contracts, x)
		if g := st.retainerBetween(k, max(today+1, k.from), end); g > 0 {
			when := "accrued daily to " + date(end-1)
			label := "Retainer to the contract's end · " + teleID(c)
			if k.until < 0 {
				label = "Retainer to the earliest end · " + teleID(c)
			}
			v.Guaranteed = append(v.Guaranteed, api.MoneyLine{Label: label, Cents: g, When: when, Basis: "contract: minimum term and notice"})
		}
		if k.until < 0 {
			v.Forecast = append(v.Forecast, api.MoneyLine{Label: "Retainer run rate while open-ended · " + teleID(c), Cents: k.monthly(),
				When: "per month after " + date(end-1), Basis: "until a cancellation fixes the end"})
		}
	}
	for c := range v.Contracts {
		var fees int64
		for _, cl := range t.claims {
			if cl.contract == c && st.dem.studies[cl.study].state == studyFinal {
				fees += cl.fee + cl.surcharge
			}
		}
		v.Contracts[c].ReadFeesCents = fees
	}
	cov := st.teleCovering(now)
	if cov >= 0 {
		v.Covering = teleID(cov)
		for _, si := range st.dem.worklist {
			if st.dem.studies[si].state == studyUnread && st.teleEligible(cov, si) {
				v.EligibleWaiting++
			}
		}
	}
	for l, ln := range t.lanes {
		x := api.TeleLane{Lane: l + 1, Phase: "idle"}
		if ln.claim >= 0 {
			cl := &t.claims[ln.claim]
			s := &st.dem.studies[cl.study]
			x.Phase, x.Study, x.Modality, x.Priority = "handoff", studyID(cl.study), st.cfg.Catalog[s.exam].Modality, s.priority.String()
			if cl.readAt > 0 {
				x.Phase = "reading"
				v.InRemoteRead++
			} else {
				v.InHandoff++
			}
			x.Contract, x.Since, x.FeeCents, x.Night = teleID(cl.contract), st.cal.ISO(cl.claimedAt), cl.fee+cl.surcharge, cl.night
		}
		v.Lanes = append(v.Lanes, x)
	}
	var remoteTAT, localTAT int64
	start := st.cal.StartOfDay(st.startDay)
	for _, si := range st.dem.finals {
		s := &st.dem.studies[si]
		if s.finalAt < start {
			continue
		}
		tat := int64(s.finalAt.Sub(s.acquiredAt) / simtime.Minute)
		if c, ok := s.by.remote(); ok {
			v.RemoteReports++
			remoteTAT += tat
			if t.claims[c].night {
				v.NightReads++
			}
		} else {
			v.LocalReports++
			localTAT += tat
		}
	}
	if v.RemoteReports > 0 {
		v.RemoteTATMeanMin = remoteTAT / int64(v.RemoteReports)
	}
	if v.LocalReports > 0 {
		v.LocalTATMeanMin = localTAT / int64(v.LocalReports)
	}
	v.RetainerCents, v.ReadFeesCents, v.PaidCents = t.retainer, t.reads, t.paid
	v.UnpaidCents, v.InFlightCents = t.retainer+t.reads-t.paid, st.inFlight()
	d := st.cal.Date(st.cal.StartOfDay(today))
	v.PayableDate = d.AddDays(int64(d.DaysInMonth() - d.Day + 1)).String()
	head := []api.MoneyLine{}
	if v.UnpaidCents > 0 {
		head = append(head, api.MoneyLine{Label: "Vendor bills unpaid (accounts payable)", Cents: v.UnpaidCents, When: "paid " + v.PayableDate,
			Basis: "recognized retainers and read fees"})
	}
	if v.InFlightCents > 0 {
		head = append(head, api.MoneyLine{Label: "Read fees committed by claims in flight", Cents: v.InFlightCents, When: "recognized at report final",
			Basis: "locked at claim"})
	}
	v.Guaranteed = append(head, v.Guaranteed...)
	return v
}

func actionIDOr(i int) string {
	if i < 0 {
		return ""
	}
	return actionID(i)
}

// trailingDemand counts the studies that waited unread on the worklist,
// in the 28 days before now, at an instant windows w cover, within scope
// (original priority, catalog modality), and prices them at plan p's fees
// at the first such instant. It reads only the past: no draw, no hidden
// outcome.
func (st *state) trailingDemand(w teleWindows, scope teleScope, fees [len(teleMods)]int64, nightBP int64) (int, int64) {
	now := st.now
	from := now.Add(-teleTrailDays * simtime.Day)
	covered := func(x simtime.Time) bool { return w.contain(st.cal.Weekday(x), simtime.ClockOf(x)) }
	var n int
	var cents int64
	for si := range st.dem.studies {
		s := &st.dem.studies[si]
		if s.state < studyUnread {
			continue
		}
		a, b := max(s.enteredAt, from), now
		if s.state >= studyReading {
			b = s.claimedAt
		}
		m := st.teleMod(s.exam)
		if a >= b || m < 0 || scope == teleScopeUrgent && s.priority != Urgent {
			continue
		}
		at := simtime.Time(-1)
		if covered(a) {
			at = a
		} else {
		edges:
			for d := st.cal.DayIndex(a); d <= st.cal.DayIndex(b); d++ {
				for _, c := range teleEdgeClocks {
					if x := st.cal.At(d, c); x > a && x < b && covered(x) {
						at = x
						break edges
					}
				}
			}
		}
		if at < 0 {
			continue
		}
		n++
		cents += fees[m]
		if teleNight(simtime.ClockOf(at)) {
			cents += fees[m] * nightBP / 10_000
		}
	}
	return n, cents
}

// telePreview previews a valid contract item (PreviewStaffing).
func (st *state) telePreview(ts *teleSpec, out *api.StaffingPreview) {
	k := &ts.k
	date := func(d int64) string { return st.cal.Date(st.cal.StartOfDay(d)).String() }
	x := api.TelePreview{Op: ts.op, Plan: k.plan, Windows: windowList(k.windows), Scope: scopeName(k.scope), Effective: date(k.from),
		MinimumTermEnd: date(k.from + teleMinTermDays), MonthlyRetainerCents: k.monthly(), ReadFees: feeList(k.fees), NightSurchargeBP: k.nightBP,
		TrailingDays: teleTrailDays}
	end := max(st.day+teleCancelDays, k.from+teleMinTermDays)
	if ts.op == api.TeleOpCancel {
		x.Contract, x.EndDate, end = teleID(ts.contract), date(ts.until), ts.until
	}
	x.GuaranteedThrough = date(end)
	x.GuaranteedCents = st.retainerBetween(k, max(st.day+1, k.from), end)
	x.TrailingStudies, x.TrailingFeeCents = st.trailingDemand(k.windows, k.scope, k.fees, k.nightBP)
	out.Teleradiology = append(out.Teleradiology, x)
	who := teleID(ts.contract)
	if ts.op == api.TeleOpStart {
		who = "new " + k.windows.String() + " contract"
		out.Guaranteed = append(out.Guaranteed, api.MoneyLine{Label: "Teleradiology retainer to the earliest end · " + who, Cents: x.GuaranteedCents,
			When: "accrued daily " + date(k.from) + " to " + date(end-1) + ", paid on the 1st", Basis: "contract: minimum term and notice, even with no reads"})
		out.Forecast = append(out.Forecast, api.MoneyLine{Label: "Retainer run rate while open-ended · " + who, Cents: k.monthly(),
			When: "per month after " + date(end-1), Basis: "until a cancellation fixes the end"})
	} else {
		out.Guaranteed = append(out.Guaranteed, api.MoneyLine{Label: "Teleradiology retainer to the end · " + who, Cents: x.GuaranteedCents,
			When: "accrued daily to " + date(end-1) + ", paid on the 1st", Basis: "contract: the fixed end"})
	}
	out.Forecast = append(out.Forecast, api.MoneyLine{Label: fmt.Sprintf("Remote read fees at trailing demand · %s (%d studies)", who, x.TrailingStudies),
		Cents: x.TrailingFeeCents, When: fmt.Sprintf("per %d days", teleTrailDays),
		Basis: "studies that waited unread in these windows and scope in the last 28 days: an upper bound (two lanes; employed readers claim first)"})
}
