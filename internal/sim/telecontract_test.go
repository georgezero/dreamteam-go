package sim_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// contract_teleradiology (Slice 5B, V0.2_DESIGN §6.3, §8.2, §8.4, §9).
//
// The fixture is micro-tele (remote_test.go): the run starts Mon Oct 12
// and its 24 X-ray studies, imaged Tue Oct 13 with no radiologist on
// site, wait. A contract applied on Oct 12 activates Mon Oct 26 at the
// earliest, so the backlog is claimed from the Monday-night window (18:00
// Oct 26), urgent studies first. With longRead (60-hour) reads two lanes
// are still reading at Wed Nov 25 00:00, the earliest end of a contract
// activated Oct 26, with six studies waiting. standard-v1: XR $12 a read,
// $14.40 claimed at night (≥ 22:00 or < 08:00); retainers weeknights
// $1,500, saturday $800, sunday $800 a month.

const (
	xrFee    = 1_200
	xrNight  = 1_440
	longRead = 3_600
)

func startAct(req, windows, scope, effective string) api.Action {
	return api.Action{Type: api.ActContractTeleradiology, Op: api.TeleOpStart, RequestID: req, Plan: "standard-v1",
		Windows: strings.Split(windows, ","), Scope: scope, Effective: effective}
}

func cancelAct(id, end string) api.Action {
	return api.Action{Type: api.ActContractTeleradiology, Op: api.TeleOpCancel, Contract: id, EndDate: end}
}

// econWorld is a micro-tele world with fixed read minutes, run to Mon Oct
// 12 09:00; day is Oct 12 00:00.
func econWorld(t *testing.T, fixture string, read int64) (*sim.World, simtime.Time) {
	t.Helper()
	w, tue := teleWorld(t, fixture, read)
	mon := tue.Add(-simtime.Day)
	mustRun(t, w, clk(mon, "09:00"))
	return w, mon
}

// on is a clock on the date n days after day.
func on(day simtime.Time, n int, hhmm string) simtime.Time {
	return clk(day.Add(simtime.Duration(n)*simtime.Day), hhmm)
}

func codeList(res []api.ActionResult) string {
	var c []string
	for _, r := range res {
		c = append(c, r.Code)
	}
	return strings.Join(c, ",")
}

// accrual is the documented retainer proration: floor(M×d/n) −
// floor(M×(d−1)/n) on day d of an n-day month.
func accrual(m int64, d, n int) int64 { return m*int64(d)/int64(n) - m*int64(d-1)/int64(n) }

// retainerFor sums the retainer of monthly m over [from, to).
func retainerFor(w *sim.World, m int64, from, to string) int64 {
	var sum int64
	for d := dayOf(w, from); d < dayOf(w, to); d++ {
		date := w.Calendar().Date(w.Calendar().StartOfDay(d))
		sum += accrual(m, date.Day, date.DaysInMonth())
	}
	return sum
}

func entriesOf(w *sim.World, kind string) []sim.EntryInfo {
	var out []sim.EntryInfo
	for _, e := range w.Journal() {
		if e.Kind == kind {
			out = append(out, e)
		}
	}
	return out
}

func lineOf(e sim.EntryInfo, acct string) int64 {
	var n int64
	for _, l := range e.Lines {
		if l.Account == acct {
			n += l.Amt
		}
	}
	return n
}

// 1. A valid start: accepted with its ID and activation, frozen terms in
// the view, nothing accrued or claimable before the activation.
func TestTeleStartAccepted(t *testing.T) {
	w, mon := econWorld(t, "micro-tele.json", 30)
	if w.HasTele() || w.Observe(0).Teleradiology != nil {
		t.Fatal("remote state before any contract")
	}
	r := mustApply(t, w, startAct("tele-1", "weeknights", "all", "2026-10-26"))[0]
	if r.Code != api.ActionAccepted || r.ContractID != "T00001" || r.EffectiveAt != "2026-10-26" ||
		!strings.Contains(r.Message, "Teleradiology contract T00001 (standard-v1): weeknights · all priorities · from 2026-10-26") {
		t.Fatalf("%+v", r)
	}
	v := w.Observe(0).Teleradiology
	if v == nil || len(v.Contracts) != 1 {
		t.Fatalf("%+v", v)
	}
	c := v.Contracts[0]
	if c.Status != "pending" || c.Effective != "2026-10-26" || c.EndDate != "" || c.EarliestEndDate != "2026-11-25" ||
		c.MinimumTermEnd != "2026-11-25" || c.MonthlyRetainerCents != 150_000 || c.ReadFees[0] != (api.ModalityPrice{Modality: "XR", Cents: xrFee}) ||
		c.NightSurchargeBP != 2_000 || c.Action != r.ID || c.RetainerCents != 0 {
		t.Fatalf("%+v", c)
	}
	want := retainerFor(w, 150_000, "2026-10-26", "2026-11-25")
	if len(v.Guaranteed) != 1 || v.Guaranteed[0].Cents != want || want != 29_033+120_000 {
		t.Fatalf("guaranteed %+v, want %d", v.Guaranteed, want)
	}
	mustRun(t, w, on(mon, 14, "12:00")) // Mon Oct 26: active, but no coverage until 18:00
	if ret, _, _ := w.TeleTotals(); ret != accrual(150_000, 26, 31) || len(w.TeleClaims()) != 0 {
		t.Fatalf("Oct 26 retainer %d, %d claims", ret, len(w.TeleClaims()))
	}
}

// 2. Activation dates: malformed, not after today, beyond the booking
// horizon; and the other start fields.
func TestTeleStartValidation(t *testing.T) {
	w, _ := econWorld(t, "micro-tele.json", 30)
	bad := func(code string, a api.Action) {
		t.Helper()
		if r := w.Apply([]api.Action{a})[0]; r.OK || r.Code != code {
			t.Fatalf("%+v: %+v, want %s", a, r, code)
		}
	}
	bad("invalid_date", startAct("a", "weeknights", "all", "2026-10-32"))
	bad("past_date", startAct("a", "weeknights", "all", "2026-10-12"))
	bad("beyond_horizon", startAct("a", "weeknights", "all", "2026-11-12"))
	bad("invalid_window", startAct("a", "weeknights,weeknights", "all", "2026-10-26"))
	bad("invalid_window", startAct("a", "evenings", "all", "2026-10-26"))
	bad("invalid_scope", startAct("a", "sunday", "routine", "2026-10-26"))
	x := startAct("a", "sunday", "all", "2026-10-26")
	x.Plan = "premium-v2"
	bad("unknown_plan", x)
	x = startAct("a", "sunday", "all", "2026-10-26")
	x.Employee = "E01"
	bad("invalid_schema", x)
	x = startAct("a", "sunday", "all", "2026-10-26")
	x.EndDate = "2026-12-01"
	bad("invalid_schema", x)
	bad("invalid_schema", api.Action{Type: api.ActContractTeleradiology, Op: "renew"})
	if w.HasTele() {
		t.Fatal("a refused start left remote state")
	}
	if r := mustApply(t, w, startAct("a", "sunday", "all", "2026-11-11"))[0]; r.EffectiveAt != "2026-11-11" { // the horizon's last day
		t.Fatalf("%+v", r)
	}
}

// 3. Notice: activation at least 14 days ahead.
func TestTeleNoticePeriod(t *testing.T) {
	w, _ := econWorld(t, "micro-tele.json", 30)
	if r := w.Apply([]api.Action{startAct("a", "weeknights", "all", "2026-10-25")})[0]; r.Code != "notice_too_short" ||
		!strings.Contains(r.Message, "the earliest activation is 2026-10-26") {
		t.Fatalf("%+v", r)
	}
	mustApply(t, w, startAct("a", "weeknights", "all", "2026-10-26"))
}

// 4. The minimum term: an end at least 30 days after activation (and 30
// days ahead).
func TestTeleMinimumTerm(t *testing.T) {
	w, _ := econWorld(t, "micro-tele.json", 30)
	mustApply(t, w, startAct("a", "weeknights", "all", "2026-10-26"))
	if r := w.Apply([]api.Action{cancelAct("T00001", "2026-11-24")})[0]; r.Code != "minimum_term" ||
		!strings.Contains(r.Message, "the earliest end is 2026-11-25") {
		t.Fatalf("%+v", r)
	}
	if r := w.Apply([]api.Action{cancelAct("T00009", "2026-11-25")})[0]; r.Code != "unknown_contract" {
		t.Fatalf("%+v", r)
	}
	r := mustApply(t, w, cancelAct("T00001", "2026-11-25"))[0]
	if r.ContractID != "T00001" || r.EffectiveAt != "2026-11-25" {
		t.Fatalf("%+v", r)
	}
}

// 5. A cancellation before activation fixes the end; the minimum term is
// still owed in full with no read at all.
func TestTeleCancelBeforeActivation(t *testing.T) {
	w, mon := econWorld(t, "micro-tele.json", 30)
	mustApply(t, w, startAct("a", "weeknights", "urgent", "2026-10-26"))
	mustApply(t, w, cancelAct("T00001", "2026-11-25"))
	mustRun(t, w, on(mon, 13, "23:00"))
	if ret, _, _ := w.TeleTotals(); ret != 0 {
		t.Fatalf("retainer %d before activation", ret)
	}
	runChecked(t, w, on(mon, 50, "00:00")) // Dec 1
	ret, reads, paid := w.TeleTotals()
	want := retainerFor(w, 150_000, "2026-10-26", "2026-11-25")
	if ret != want || want != 149_033 || w.Balance("remote_retainer") != want {
		t.Fatalf("retainer %d (account %d), want %d", ret, w.Balance("remote_retainer"), want)
	}
	if n := len(entriesOf(w, "retainer")); n != 30 {
		t.Fatalf("%d retainer entries, want 30", n)
	}
	if reads != 3*xrFee { // the three urgent studies, Mon Oct 26 18:00–18:40
		t.Fatalf("reads %d", reads)
	}
	if paid != ret+reads {
		t.Fatalf("paid %d of %d on Dec 1", paid, ret+reads)
	}
}

// 6. A cancellation during the term: 30 days' notice; claims stop at the
// end.
func TestTeleCancelDuringTerm(t *testing.T) {
	w, mon := econWorld(t, "micro-tele.json", 30)
	mustApply(t, w, startAct("a", "weeknights", "all", "2026-10-26"))
	mustRun(t, w, on(mon, 14, "19:00")) // Mon Oct 26: reading
	if r := w.Apply([]api.Action{cancelAct("T00001", "2026-11-24")})[0]; r.Code != "notice_too_short" ||
		!strings.Contains(r.Message, "the earliest end is 2026-11-25") {
		t.Fatalf("%+v", r)
	}
	before := w.TeleFees()
	mustApply(t, w, cancelAct("T00001", "2026-11-25"))
	if !slices.Equal(before, w.TeleFees()[:len(before)]) {
		t.Fatal("cancellation repriced claims")
	}
	if c := w.Observe(0).Teleradiology.Contracts[0]; c.EndDate != "2026-11-25" || c.Status != "active" || c.CancelAction == "" {
		t.Fatalf("%+v", c)
	}
}

// 7. Expiry: the end is exclusive. With backlog waiting and the Tuesday
// night window open, nothing is claimed at the end instant or after; the
// retainer stops; the contract shows ended.
func TestTeleContractExpiry(t *testing.T) {
	w, mon := econWorld(t, "micro-tele.json", longRead)
	mustApply(t, w, startAct("a", "weeknights", "all", "2026-10-26"))
	mustApply(t, w, cancelAct("T00001", "2026-11-25"))
	runChecked(t, w, on(mon, 44, "06:00")) // Wed Nov 25 06:00, inside Tuesday's night window
	claims := w.TeleClaims()
	for _, c := range claims {
		if c.Claimed >= on(mon, 44, "00:00") {
			t.Fatalf("claim at or after the end: %+v", c)
		}
	}
	if len(claims) != 18 || w.Observe(0).Teleradiology.EligibleWaiting != 0 || len(w.Observe(0).Worklist.Items) == 0 {
		t.Fatalf("%d claims", len(claims))
	}
	if w.TeleCovering(on(mon, 44, "00:00")) != "" || w.TeleCovering(on(mon, 43, "23:59")) != "T00001" {
		t.Fatal("coverage at the end")
	}
	if ret, _, _ := w.TeleTotals(); ret != retainerFor(w, 150_000, "2026-10-26", "2026-11-25") {
		t.Fatalf("retainer %d", ret)
	}
	v := w.Observe(0).Teleradiology
	if v.Contracts[0].Status != "ended" || v.EligibleWaiting != 0 || v.Covering != "" || w.Observe(0).Worklist.Unread != 6 {
		t.Fatalf("%+v", v.Contracts[0])
	}
}

// 8. A successor: cancel and start in one batch, in either order, judged
// on the final timeline; the successor starts at the predecessor's end.
func TestTeleSuccessor(t *testing.T) {
	for _, reversed := range []bool{false, true} {
		w, mon := econWorld(t, "micro-tele.json", longRead)
		mustApply(t, w, startAct("a", "weeknights", "all", "2026-10-26"))
		mustRun(t, w, on(mon, 14, "09:00")) // Mon Oct 26
		batch := []api.Action{startAct("b", "weeknights,saturday,sunday", "urgent", "2026-11-25"), cancelAct("T00001", "2026-11-25")}
		if reversed {
			batch[0], batch[1] = batch[1], batch[0]
		}
		res := mustApply(t, w, batch...)
		k := 0
		if reversed {
			k = 1
		}
		if res[k].ContractID != "T00002" || res[1-k].ContractID != "T00001" {
			t.Fatalf("%+v", res)
		}
		v := w.Observe(0).Teleradiology
		if v.Contracts[0].EndDate != "2026-11-25" || v.Contracts[1].Effective != "2026-11-25" || v.Contracts[1].MonthlyRetainerCents != 310_000 {
			t.Fatalf("%+v", v.Contracts)
		}
	}
}

// 9. Overlap: a second contract while the first is open-ended, or one
// starting before the first's end.
func TestTeleOverlapRefused(t *testing.T) {
	w, mon := econWorld(t, "micro-tele.json", 30)
	mustApply(t, w, startAct("a", "weeknights", "all", "2026-10-26"))
	if r := w.Apply([]api.Action{startAct("b", "sunday", "all", "2026-11-08")})[0]; r.Code != "contract_overlap" {
		t.Fatalf("%+v", r)
	}
	mustRun(t, w, on(mon, 14, "09:00"))
	res := w.Apply([]api.Action{cancelAct("T00001", "2026-11-25"), startAct("b", "sunday", "all", "2026-11-24")})
	if codeList(res) != "batch_rejected,contract_overlap" {
		t.Fatalf("%+v", res)
	}
	if c := w.Observe(0).Teleradiology.Contracts; len(c) != 1 || c[0].EndDate != "" {
		t.Fatalf("the refused batch changed the timeline: %+v", c)
	}
}

// 10. Mixed batches are atomic: a refused item refuses the batch, a V0.1
// action cannot join it, and nothing changes.
func TestTeleMixedBatchAtomic(t *testing.T) {
	w := atTakeover(t, `{}`, 31)
	before := digestsOf(w)
	res := w.Apply([]api.Action{setAlloc("E07", "2026-10-28", 8_000), startAct("a", "weeknights", "all", "2026-10-28"),
		startAct("b", "sunday", "nightly", "2026-10-28")})
	if codeList(res) != "batch_rejected,batch_rejected,invalid_scope" {
		t.Fatalf("%+v", res)
	}
	res = w.Apply([]api.Action{{Type: api.ActSetWorklistPolicy, Policy: api.WorklistFIFO}, startAct("a", "weeknights", "all", "2026-10-28")})
	if res[0].Code != "mixed_batch" || !strings.Contains(res[0].Message, "set_roster, contract_teleradiology)") || res[1].Code != "batch_rejected" {
		t.Fatalf("%+v", res)
	}
	if digestsOf(w) != before || w.HasTele() || len(w.ReadingAllocations()[0].Versions) != 1 {
		t.Fatal("a refused batch changed the state")
	}
	res = mustApply(t, w, setAlloc("E07", "2026-10-28", 8_000), startAct("a", "weeknights", "all", "2026-10-28"))
	if res[1].ContractID != "T00001" || len(w.ReadingAllocations()[0].Versions) != 2 {
		t.Fatalf("%+v", res)
	}
}

// 11. Retainers: each selected window accrues its own entry every active
// day at 00:00; a full month adds to the monthly retainers to the cent.
func TestTeleRetainerAccrual(t *testing.T) {
	w, mon := econWorld(t, "micro-tele.json", 30)
	mustApply(t, w, startAct("a", "weeknights,saturday,sunday", "all", "2026-10-26"))
	runChecked(t, w, on(mon, 50, "00:00")) // Dec 1
	var nov, n int64
	for _, e := range entriesOf(w, "retainer") {
		date := w.Calendar().Date(e.At)
		if simtime.ClockOf(e.At) != 0 || lineOf(e, "remote_retainer") != -lineOf(e, "accounts_payable") {
			t.Fatalf("%+v", e)
		}
		if date.Month == 11 {
			nov += lineOf(e, "remote_retainer")
			n++
		}
	}
	if nov != 150_000+80_000+80_000 || n != 90 {
		t.Fatalf("November retainers %d in %d entries", nov, n)
	}
}

// 12. Partial periods: an activation and an end mid-month prorate by
// active days, window by window, to the cent.
func TestTelePartialPeriod(t *testing.T) {
	w, mon := econWorld(t, "micro-tele.json", 30)
	mustApply(t, w, startAct("a", "saturday,sunday", "all", "2026-10-26"))
	mustApply(t, w, cancelAct("T00001", "2026-11-25"))
	runChecked(t, w, on(mon, 50, "00:00"))
	want := 2 * retainerFor(w, 80_000, "2026-10-26", "2026-11-25")
	if want != 2*(15_484+64_000) {
		t.Fatalf("expected arithmetic %d", want)
	}
	var got int64
	first, last := simtime.Time(1<<62), simtime.Time(0)
	for _, e := range entriesOf(w, "retainer") {
		got += lineOf(e, "remote_retainer")
		first, last = min(first, e.At), max(last, e.At)
	}
	if got != want || w.Calendar().Date(first).String() != "2026-10-26" || w.Calendar().Date(last).String() != "2026-11-24" {
		t.Fatalf("retainer %d (want %d), %s to %s", got, want, w.Calendar().ISO(first), w.Calendar().ISO(last))
	}
}

// 13. A claim locks its fee: a cancellation while it is in flight changes
// nothing, and the fee is recognized once, at the report's final (not at
// claim or handoff).
func TestTeleFeeFrozenAtClaim(t *testing.T) {
	w, mon := econWorld(t, "micro-tele.json", 30)
	mustApply(t, w, startAct("a", "weeknights", "all", "2026-10-26"))
	mustRun(t, w, on(mon, 14, "18:15")) // both lanes reading since 18:10
	if _, reads, _ := w.TeleTotals(); reads != 0 || len(entriesOf(w, "remote_read")) != 0 {
		t.Fatal("a fee recognized before final")
	}
	if v := w.Observe(0).Teleradiology; v.InFlightCents != 2*xrFee || v.Lanes[0].FeeCents != xrFee || v.Lanes[0].Phase != "reading" {
		t.Fatalf("%+v", v)
	}
	mustApply(t, w, cancelAct("T00001", "2026-11-26"))
	mustRun(t, w, on(mon, 14, "18:40"))
	reads := entriesOf(w, "remote_read")
	if len(reads) != 2 || reads[0].At != on(mon, 14, "18:40") || lineOf(reads[0], "remote_reads") != xrFee || lineOf(reads[1], "remote_reads") != xrFee {
		t.Fatalf("%+v", reads)
	}
}

// 14. Night pricing by claim time: 22:00 is night, 21:20 day; claims
// after midnight night.
func TestTeleNightClassification(t *testing.T) {
	w, mon := econWorld(t, "micro-tele.json", 30)
	mustApply(t, w, startAct("a", "weeknights", "all", "2026-10-26"))
	runChecked(t, w, on(mon, 16, "00:00"))
	claims, fees := w.TeleClaims(), w.TeleFees()
	if len(claims) != 24 {
		t.Fatalf("%d claims", len(claims))
	}
	for k, c := range claims {
		clock := simtime.ClockOf(c.Claimed)
		night := clock >= 22*simtime.Hour || clock < 8*simtime.Hour
		want := int64(xrFee)
		if night {
			want = xrNight
		}
		if fees[k].Night != night || fees[k].Fee+fees[k].Surcharge != want {
			t.Fatalf("claim at %s: %+v", w.Calendar().ISO(c.Claimed), fees[k])
		}
	}
	if simtime.ClockOf(claims[10].Claimed) != 21*simtime.Hour+20*simtime.Minute || fees[10].Night ||
		simtime.ClockOf(claims[12].Claimed) != 22*simtime.Hour || !fees[12].Night {
		t.Fatal("boundary claims")
	}
	_, reads, _ := w.TeleTotals()
	if reads != 12*xrFee+12*xrNight || w.Balance("remote_reads") != reads {
		t.Fatalf("reads %d", reads)
	}
}

// 15. A read across midnight finishes the next day at its locked fee,
// after that day's retainer; its revenue and fee belong to the final
// date.
func TestTeleReadCrossesMidnight(t *testing.T) {
	w, mon := econWorld(t, "micro-tele.json", 45)
	mustApply(t, w, startAct("a", "weeknights", "all", "2026-10-26"))
	runChecked(t, w, on(mon, 15, "06:00"))
	var cross []int
	for k, c := range w.TeleClaims() {
		if s := studyByID(t, w, c.Study); w.Calendar().DayIndex(c.Claimed) != w.Calendar().DayIndex(s.Final) {
			cross = append(cross, k)
		}
	}
	if len(cross) != 2 {
		t.Fatalf("%d reads across midnight", len(cross))
	}
	j := w.Journal()
	ret := slices.IndexFunc(j, func(e sim.EntryInfo) bool { return e.Kind == "retainer" && e.At == on(mon, 15, "00:00") })
	for _, k := range cross {
		c := w.TeleClaims()[k]
		s := studyByID(t, w, c.Study)
		fee := slices.IndexFunc(j, func(e sim.EntryInfo) bool { return e.Kind == "remote_read" && fmt.Sprintf("S%05d", e.A+1) == c.Study })
		rev := slices.IndexFunc(j, func(e sim.EntryInfo) bool { return e.Kind == "revenue" && fmt.Sprintf("S%05d", e.A+1) == c.Study })
		if ret < 0 || fee < ret || j[fee].At != s.Final || j[rev].At != s.Final || lineOf(j[fee], "remote_reads") != xrNight {
			t.Fatalf("claim %+v: retainer %d, fee %+v, revenue %+v", c, ret, j[fee], j[rev])
		}
	}
}

// 16. Reads across the contract's end complete after it at their locked
// fees, recognized at their finals; no retainer accrues after the end.
func TestTeleReadCrossesExpiry(t *testing.T) {
	w, mon := econWorld(t, "micro-tele.json", longRead)
	mustApply(t, w, startAct("a", "weeknights", "all", "2026-10-26"))
	mustApply(t, w, cancelAct("T00001", "2026-11-25"))
	runChecked(t, w, on(mon, 47, "12:00"))
	end := on(mon, 44, "00:00")
	var after int
	for _, f := range w.TeleFees() {
		s := studyByID(t, w, f.Study)
		if s.Final <= end {
			continue
		}
		after++
		want := int64(xrFee)
		if f.Night {
			want = xrNight
		}
		fee := slices.IndexFunc(entriesOf(w, "remote_read"), func(e sim.EntryInfo) bool { return sid(int(e.A)+1) == f.Study })
		if f.Contract != "T00001" || f.Fee+f.Surcharge != want || fee < 0 || entriesOf(w, "remote_read")[fee].At != s.Final {
			t.Fatalf("%+v", f)
		}
	}
	if after != 2 {
		t.Fatalf("%d reads finished after the end", after)
	}
	for _, e := range entriesOf(w, "retainer") {
		if e.At >= end {
			t.Fatalf("retainer after the end: %+v", e)
		}
	}
}

// 17. Settlement: each first-of-month A/P payment pays every vendor bill
// accrued before it (the 1st's own retainer waits a month), including,
// on Dec 1, the fees of reads finished after the contract's end; an
// expiry erases nothing.
func TestTeleSettlementAfterExpiry(t *testing.T) {
	w, mon := econWorld(t, "micro-tele.json", longRead)
	mustApply(t, w, startAct("a", "weeknights", "all", "2026-10-26"))
	mustApply(t, w, cancelAct("T00001", "2026-11-25"))
	billed := func(before simtime.Time) int64 {
		var n int64
		for _, e := range w.Journal() {
			if (e.Kind == "retainer" || e.Kind == "remote_read") && e.At < before {
				n += lineOf(e, "remote_retainer") + lineOf(e, "remote_reads")
			}
		}
		return n
	}
	nov1, dec1 := on(mon, 20, "00:00"), on(mon, 50, "00:00")
	runChecked(t, w, nov1.Add(simtime.Minute))
	if _, _, paid := w.TeleTotals(); paid != billed(nov1) || paid == 0 {
		t.Fatalf("Nov 1: paid %d, October bills %d", paid, billed(nov1))
	}
	runChecked(t, w, dec1.Add(-simtime.Hour))
	ret, reads, paid := w.TeleTotals()
	v := w.Observe(0).Teleradiology
	if v.UnpaidCents != ret+reads-paid || v.UnpaidCents == 0 || v.PayableDate != "2026-12-01" || v.Contracts[0].Status != "ended" {
		t.Fatalf("Nov 30: %+v", v)
	}
	runChecked(t, w, dec1.Add(simtime.Minute))
	ret2, reads2, paid2 := w.TeleTotals()
	late := slices.ContainsFunc(entriesOf(w, "remote_read"), func(e sim.EntryInfo) bool { return e.At > on(mon, 44, "00:00") && e.At < dec1 })
	if ret2 != ret || reads2 != reads || paid2 != billed(dec1) || paid2 != ret+reads || !late || w.Observe(0).Teleradiology.UnpaidCents != 0 {
		t.Fatalf("Dec 1: retainer %d, reads %d, paid %d (bills %d), a fee after the end %v", ret2, reads2, paid2, billed(dec1), late)
	}
}

// 18. At an endpoint mid-read the fee is committed, not recognized: it is
// in flight in the obligations, not in expense or payables.
func TestTeleOutstandingAtEndpoint(t *testing.T) {
	w, mon := econWorld(t, "micro-tele.json", 30)
	mustApply(t, w, startAct("a", "weeknights", "all", "2026-10-26"))
	runChecked(t, w, on(mon, 14, "22:30"))
	_, reads, _ := w.TeleTotals()
	v := w.Observe(0).Teleradiology
	var inflight int64
	for _, f := range w.TeleFees() {
		if !f.Final {
			inflight += f.Fee + f.Surcharge
		}
	}
	if inflight != xrNight*2 || v.InFlightCents != inflight || w.Balance("remote_reads") != reads ||
		!slices.ContainsFunc(v.Guaranteed, func(m api.MoneyLine) bool { return m.Cents == inflight && strings.Contains(m.Label, "in flight") }) {
		t.Fatalf("in flight %d: %+v", inflight, v)
	}
}

// 19. One claim, one final report, one revenue entry, one payer claim and
// one fee per remote study; the fees add up to the account.
func TestTeleOneReportOneFee(t *testing.T) {
	w, mon := econWorld(t, "micro-tele.json", 30)
	mustApply(t, w, startAct("a", "weeknights", "all", "2026-10-26"))
	runChecked(t, w, on(mon, 16, "00:00"))
	count := func(kind string) map[int64]int {
		m := map[int64]int{}
		for _, e := range entriesOf(w, kind) {
			m[e.A]++
		}
		return m
	}
	rev, fee := count("revenue"), count("remote_read")
	payer := map[string]int{}
	for _, c := range w.Claims() {
		payer[c.Study]++
	}
	var sum int64
	for _, f := range w.TeleFees() {
		sum += f.Fee + f.Surcharge
	}
	for n := 1; n <= 24; n++ {
		if rev[int64(n-1)] != 1 || fee[int64(n-1)] != 1 || payer[sid(n)] != 1 {
			t.Fatalf("%s: %d revenue, %d fees, %d payer claims", sid(n), rev[int64(n-1)], fee[int64(n-1)], payer[sid(n)])
		}
	}
	if w.Balance("remote_reads") != sum {
		t.Fatalf("fees %d, account %d", sum, w.Balance("remote_reads"))
	}
	if v := w.Observe(0).Teleradiology; v.RemoteReports != 24 || v.LocalReports != 0 || v.RemoteTATMeanMin == 0 {
		t.Fatalf("%+v", v)
	}
}

// 20. An urgent-only contract claims by original priority.
func TestTeleUrgentOnlyByOriginalPriority(t *testing.T) {
	w, mon := econWorld(t, "micro-tele.json", 30)
	mustApply(t, w, startAct("a", "weeknights", "urgent", "2026-10-26"))
	runChecked(t, w, on(mon, 16, "00:00"))
	var got []string
	for _, c := range w.TeleClaims() {
		got = append(got, c.Study)
	}
	if want := []string{sid(4), sid(10), sid(15)}; !slices.Equal(got, want) {
		t.Fatalf("claims %v, want %v", got, want)
	}
}

// 21. Two lanes for the practice: a third eligible study waits through
// both handoffs and reads.
func TestTeleTwoLanes(t *testing.T) {
	w, mon := econWorld(t, "micro-tele.json", 30)
	mustApply(t, w, startAct("a", "weeknights,saturday,sunday", "all", "2026-10-26"))
	mustRun(t, w, on(mon, 14, "17:00"))
	w.PushLiveNoise(on(mon, 14, "18:05")) // a dispatch mid-handoff
	mustRun(t, w, on(mon, 14, "18:05"))
	v := w.Observe(0).Teleradiology
	if len(v.Lanes) != 2 || v.InHandoff != 2 || v.EligibleWaiting != 22 || len(w.TeleClaims()) != 2 {
		t.Fatalf("%+v", v)
	}
	mustRun(t, w, on(mon, 14, "18:39"))
	if len(w.TeleClaims()) != 2 {
		t.Fatal("a third claim while both lanes read")
	}
}

// 22. A successor brings no lanes: at its start both lanes still hold
// its predecessor's reads, so it claims nothing until one finishes, then
// only into the freed lanes; no instant ever has three claims in progress.
func TestTeleNoNewLanesOnRenewal(t *testing.T) {
	w, mon := econWorld(t, "micro-tele.json", longRead)
	mustApply(t, w, startAct("a", "weeknights", "all", "2026-10-26"))
	mustRun(t, w, on(mon, 14, "09:00"))
	mustApply(t, w, startAct("b", "weeknights,saturday,sunday", "all", "2026-11-25"), cancelAct("T00001", "2026-11-25"))
	runChecked(t, w, on(mon, 50, "00:00"))
	end := on(mon, 44, "00:00")
	type span struct{ from, to simtime.Time }
	var spans []span
	first, freed := simtime.Time(1<<62), simtime.Time(1<<62)
	for _, c := range w.TeleClaims() {
		s := studyByID(t, w, c.Study)
		to := s.Final
		if s.State != "final" {
			to = 1 << 62
		}
		spans = append(spans, span{c.Claimed, to})
		switch {
		case c.Claimed >= end && c.Contract != "T00002", c.Claimed < end && c.Contract != "T00001":
			t.Fatalf("%+v", c)
		case c.Claimed >= end:
			first = min(first, c.Claimed)
		case to > end:
			freed = min(freed, to)
		}
	}
	for _, a := range spans {
		n := 0
		for _, b := range spans {
			if b.from <= a.from && a.from < b.to {
				n++
			}
		}
		if n > 2 {
			t.Fatalf("%d claims in progress at %s", n, w.Calendar().ISO(a.from))
		}
	}
	if w.TeleCovering(end) != "T00002" || w.TeleCovering(freed) != "T00002" || first != freed {
		t.Fatalf("successor's first claim %s, first lane freed %s", w.Calendar().ISO(first), w.Calendar().ISO(freed))
	}
}

// 23. A contract started by the action changes no booking, supervision or
// plan (established practice, three weeks of remote reading).
func TestTeleBookingUnchanged(t *testing.T) {
	plain, tele := atTakeover(t, `{}`, 31), atTakeover(t, `{}`, 31)
	mustApply(t, tele, startAct("a", "weeknights,saturday,sunday", "all", "2026-10-28"))
	end := plain.Calendar().StartOfDay(dayOf(plain, "2026-11-18"))
	runChecked(t, plain, end)
	runChecked(t, tele, end)
	if n := len(tele.TeleClaims()); n < 50 {
		t.Fatalf("%d remote claims", n)
	}
	if !slices.Equal(plain.Appointments(), tele.Appointments()) {
		t.Fatal("appointments differ")
	}
	if !slices.EqualFunc(plain.Visits(), tele.Visits(), func(a, b sim.VisitInfo) bool { return fmt.Sprint(a) == fmt.Sprint(b) }) {
		t.Fatal("visits differ")
	}
	for d := dayOf(plain, "2026-10-13"); d < plain.DaysBuilt(); d++ {
		if plain.BookingView(d) != tele.BookingView(d) || !slices.Equal(plain.SupervisionOn(d), tele.SupervisionOn(d)) ||
			planText(plain.PlanOn(d)) != planText(tele.PlanOn(d)) {
			t.Fatalf("day %d: booking, supervision or plan differs", d)
		}
	}
}

// 24. No contract, no change: refused contract actions and previews leave
// the run exactly as a run without them, with no remote account listed.
func TestTeleNoContractNeutral(t *testing.T) {
	plain, other := atTakeover(t, `{}`, 32), atTakeover(t, `{}`, 32)
	other.PreviewStaffing([]api.Action{startAct("a", "weeknights", "all", "2026-10-28")})
	other.Apply([]api.Action{startAct("a", "weeknights", "all", "2026-10-14")})
	other.ValidateBatch([]api.Action{startAct("a", "sunday", "all", "2026-10-28")})
	end := plain.Calendar().StartOfDay(dayOf(plain, "2026-11-04"))
	runChecked(t, plain, end)
	runChecked(t, other, end)
	if digestsOf(plain) != digestsOf(other) || other.HasTele() {
		t.Fatal("the runs differ")
	}
	f := other.Observe(0).Finance
	for _, a := range f.Accounts {
		if strings.HasPrefix(a.ID, "remote_") {
			t.Fatalf("%+v listed", a)
		}
	}
	if f.SinceStart.RemoteRetainerCents != 0 || f.SinceStart.RemoteReadsCents != 0 {
		t.Fatal("remote expense")
	}
}

// 25. Replay: the same actions at the same instants reach the same state
// whatever the chunking.
func TestTeleReplayAndChunking(t *testing.T) {
	run := func(step simtime.Duration) [3]string {
		w, mon := econWorld(t, "micro-tele.json", longRead)
		acts := []struct {
			at simtime.Time
			a  []api.Action
		}{
			{on(mon, 0, "09:00"), []api.Action{startAct("a", "weeknights", "all", "2026-10-26")}},
			{on(mon, 14, "09:00"), []api.Action{startAct("b", "weeknights,sunday", "all", "2026-11-25"), cancelAct("T00001", "2026-11-25")}},
		}
		end := on(mon, 45, "12:00")
		for _, x := range acts {
			for w.Engine().Now() < x.at {
				mustRun(t, w, min(x.at, w.Engine().Now().Add(step)))
			}
			mustApply(t, w, x.a...)
		}
		for w.Engine().Now() < end {
			mustRun(t, w, min(end, w.Engine().Now().Add(step)))
		}
		return digestsOf(w)
	}
	whole := run(1000 * simtime.Day)
	for _, step := range []simtime.Duration{29 * simtime.Minute, 7 * simtime.Hour} {
		if got := run(step); got != whole {
			t.Fatalf("step %s: %v, whole %v", step, got, whole)
		}
	}
}

// 27. Previews are read-only: any number of previews and validations
// leave every digest as it was, and show the guaranteed retainer.
func TestTelePreviewReadOnly(t *testing.T) {
	w := atTakeover(t, `{}`, 31)
	runChecked(t, w, w.Calendar().StartOfDay(dayOf(w, "2026-10-20")).Add(9*simtime.Hour))
	before, acts := digestsOf(w), w.ActionDigest()
	a := startAct("a", "weeknights,sunday", "all", "2026-11-03")
	var p api.StaffingPreview
	for range 3 {
		p = w.PreviewStaffing([]api.Action{a})
		w.ValidateBatch([]api.Action{a})
	}
	if digestsOf(w) != before || w.ActionDigest() != acts || w.HasTele() {
		t.Fatal("previews changed the run")
	}
	want := retainerFor(w, 150_000, "2026-11-03", "2026-12-03") + retainerFor(w, 80_000, "2026-11-03", "2026-12-03")
	if !p.OK || len(p.Teleradiology) != 1 || p.Teleradiology[0].GuaranteedCents != want || p.Teleradiology[0].GuaranteedThrough != "2026-12-03" ||
		p.Teleradiology[0].TrailingStudies == 0 || p.Teleradiology[0].TrailingFeeCents == 0 || p.Results[0].ContractID != "" {
		t.Fatalf("%+v (want %d)", p.Teleradiology, want)
	}
	if !slices.ContainsFunc(p.Guaranteed, func(m api.MoneyLine) bool { return m.Cents == want }) || len(p.Forecast) != 2 {
		t.Fatalf("%+v %+v", p.Guaranteed, p.Forecast)
	}
}

// 28. Refusals and repeats change nothing: every refusal leaves every
// digest; the same start or cancellation again is unchanged; a reused
// requestId with other terms and a moved end are refused.
func TestTeleRefusalUnchanged(t *testing.T) {
	w, mon := econWorld(t, "micro-tele.json", 30)
	mustApply(t, w, startAct("a", "weeknights", "all", "2026-10-26"))
	mustRun(t, w, on(mon, 1, "09:00"))
	before := digestsOf(w)
	for _, tc := range []struct {
		code string
		a    api.Action
	}{
		{"request_conflict", startAct("a", "sunday", "all", "2026-10-26")},
		{"contract_overlap", startAct("b", "sunday", "all", "2026-11-11")},
		{"notice_too_short", cancelAct("T00001", "2026-11-11")},
		{"minimum_term", cancelAct("T00001", "2026-11-24")},
		{"unknown_contract", cancelAct("T1", "2026-12-01")},
		{"invalid_date", cancelAct("T00001", "soon")},
	} {
		if r := w.Apply([]api.Action{tc.a})[0]; r.Code != tc.code {
			t.Fatalf("%+v: %+v, want %s", tc.a, r, tc.code)
		}
		if digestsOf(w) != before {
			t.Fatalf("%s changed the state", tc.code)
		}
	}
	if r := w.Apply([]api.Action{startAct("a", "weeknights", "all", "2026-10-26")})[0]; r.Code != api.ActionUnchanged || r.ContractID != "T00001" {
		t.Fatalf("%+v", r)
	}
	if res := w.Apply([]api.Action{startAct("c", "sunday", "all", "2026-11-11"), startAct("c", "sunday", "all", "2026-11-11")}); codeList(res) != "batch_rejected,invalid_parameters" {
		t.Fatalf("%+v", res)
	}
	if digestsOf(w) != before {
		t.Fatal("a repeat changed the state")
	}
	mustApply(t, w, cancelAct("T00001", "2026-11-25"))
	after := digestsOf(w)
	if r := w.Apply([]api.Action{cancelAct("T00001", "2026-11-25")})[0]; r.Code != api.ActionUnchanged {
		t.Fatalf("%+v", r)
	}
	if r := w.Apply([]api.Action{cancelAct("T00001", "2026-12-01")})[0]; r.Code != "already_cancelled" {
		t.Fatalf("%+v", r)
	}
	if res := w.Apply([]api.Action{cancelAct("T00001", "2026-12-01"), cancelAct("T00001", "2026-12-01")}); codeList(res) != "already_cancelled,batch_rejected" {
		t.Fatalf("%+v", res)
	}
	if digestsOf(w) != after || len(w.Observe(0).Teleradiology.Contracts) != 1 {
		t.Fatal("repeats changed the state")
	}
}
