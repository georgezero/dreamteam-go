package sim

import (
	"fmt"
	"strings"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Remote reading: the teleradiology engine (V0.2_DESIGN §8.2–8.3, §10;
// Slice 5A, 0.2.0-dev.7). The player action that creates contracts
// (contract_teleradiology) and their economics arrive in Slice 5B; until
// then a contract exists only through addTeleContract, which no action
// reaches.
//
//   - Lanes. The practice has exactly two remote lanes, 0 then 1, for all
//     of its contracts together. They are persistent state, outside the
//     employees' daily materialization: midnight, operational close and
//     contract expiry never reset them. A lane is occupied from claim to
//     final report, handoff included, and released only by its read's end.
//   - Coverage. A contract covers the half-open recurring windows it
//     selects (weeknights Mon–Fri [18:00, next day 08:00), saturday
//     [14:00, 24:00), sunday [00:00, 24:00)), united, then intersected with
//     [activation 00:00, end 00:00). Contracts never overlap (a successor
//     starts at or after its predecessor's end), so at most one covers an
//     instant, and coverage is a pure function of time: windows that touch
//     or overlap add no capacity (it is always the two lanes).
//   - Claims. At each transaction's dispatch, after every on-site pass and
//     employed reading, each idle lane in lane order claims the first
//     eligible unread study in the worklist policy's order, if a contract
//     covers that instant. Eligibility is the contract's scope on the
//     study's original priority (an overdue study promoted in the worklist
//     order is not urgent). No p100-by-shift-end rule applies: a lane has
//     no shift. Coverage governs claims only: a claimed study finishes
//     after its window, its contract's end and midnight.
//   - Handoff and read. A claim occupies its lane for the ten-minute
//     handoff, then the study's own keyed read duration (the same draw an
//     employed radiologist would make). Both are generation-checked
//     completions.
//   - Finalization is the shared path (finalize): one final report, one
//     revenue recognition, turnaround and breach once.
//
// The claim keeps what 5B's settlement needs at the locked terms: the
// contract, lane, claim and read-start times and the night classification
// (claimed at ≥ 22:00 or < 08:00 local).

const (
	teleLanes   = 2
	teleHandoff = 10 * simtime.Minute
)

// teleWindows is a set of recurring coverage windows, in catalog order.
type teleWindows uint8

const (
	teleWeeknights teleWindows = 1 << iota
	teleSaturday
	teleSunday
	teleAllWindows = teleWeeknights | teleSaturday | teleSunday
)

var teleWindowNames = [...]string{"weeknights", "saturday", "sunday"}

// teleScope is which priorities a contract reads.
type teleScope uint8

const (
	teleScopeAll teleScope = iota
	teleScopeUrgent
)

// teleContract is a contract's coverage terms. Days are day indices:
// coverage from from's 00:00 to until's 00:00 (exclusive; −1 while no end
// is fixed).
type teleContract struct {
	windows     teleWindows
	scope       teleScope
	from, until int64
}

// teleLane is a remote lane: the claim it holds (−1 idle) and the
// generation that makes its completions stale once released.
type teleLane struct {
	claim int
	gen   uint32
}

// teleClaim is one remote read claim, in claim order: a study's single
// remote owner, from claim to final.
type teleClaim struct {
	study, contract, lane int
	claimedAt             simtime.Time // handoff starts
	readAt                simtime.Time // handoff done, read starts (0 before)
	night                 bool         // claimed at ≥ 22:00 or < 08:00 local
}

// teleState is the practice's remote reading. It exists from the first
// contract on; state.tele is nil (and not encoded) before.
type teleState struct {
	contracts []teleContract
	lanes     [teleLanes]teleLane
	claims    []teleClaim
	edgeGen   uint32 // the pending coverage-edge wake
}

func teleID(c int) string { return pubID('T', int64(c+1), 5) }

func laneName(l int) string { return fmt.Sprintf("Teleradiology lane %d", l+1) }

func (w teleWindows) String() string {
	var out []string
	for k, n := range teleWindowNames {
		if w&(1<<k) != 0 {
			out = append(out, n)
		}
	}
	return strings.Join(out, ", ")
}

func (s teleScope) String() string {
	if s == teleScopeUrgent {
		return "urgent only"
	}
	return "all priorities"
}

// contain reports whether the windows cover a local weekday and clock.
// Friday's night runs into Saturday before 08:00; Sunday's window ends at
// Monday 00:00, so Monday 00:00–08:00 is never covered.
func (w teleWindows) contain(wd simtime.Weekday, c simtime.Duration) bool {
	const h = simtime.Hour
	if w&teleWeeknights != 0 &&
		(wd >= simtime.Monday && wd <= simtime.Friday && c >= 18*h || wd >= simtime.Tuesday && wd <= simtime.Saturday && c < 8*h) {
		return true
	}
	if w&teleSaturday != 0 && wd == simtime.Saturday && c >= 14*h {
		return true
	}
	return w&teleSunday != 0 && wd == simtime.Sunday
}

// teleEdgeClocks are the only clocks at which coverage can change: window
// boundaries and contract dates (00:00).
var teleEdgeClocks = [...]simtime.Duration{0, 8 * simtime.Hour, 14 * simtime.Hour, 18 * simtime.Hour}

// teleCovers reports whether contract c covers instant t.
func (st *state) teleCovers(c int, t simtime.Time) bool {
	k := &st.tele.contracts[c]
	d := st.cal.DayIndex(t)
	if d < k.from || (k.until >= 0 && d >= k.until) {
		return false
	}
	return k.windows.contain(st.cal.Weekday(t), simtime.ClockOf(t))
}

// teleCovering is the contract covering instant t, or −1.
func (st *state) teleCovering(t simtime.Time) int {
	if st.tele == nil {
		return -1
	}
	for c := range st.tele.contracts {
		if st.teleCovers(c, t) {
			return c
		}
	}
	return -1
}

// teleEligible reports whether contract c may claim study si: its scope
// on the study's original priority.
func (st *state) teleEligible(c, si int) bool {
	return st.tele.contracts[c].scope == teleScopeAll || st.dem.studies[si].priority == Urgent
}

// nextTeleEdge is the first instant after t at which the covering contract
// changes (coverage opens, closes or passes to a successor), and the
// contract covering from then (−1: none); −1 if coverage never changes
// again. Coverage repeats weekly after the last contract date, so a week
// past it bounds the search.
func (st *state) nextTeleEdge(t simtime.Time) (simtime.Time, int) {
	cur := st.teleCovering(t)
	last := st.cal.DayIndex(t)
	for _, k := range st.tele.contracts {
		last = max(last, k.from, k.until)
	}
	for d := st.cal.DayIndex(t); d <= last+7; d++ {
		for _, c := range teleEdgeClocks {
			x := st.cal.At(d, c)
			if x <= t {
				continue
			}
			if n := st.teleCovering(x); n != cur {
				return x, n
			}
		}
	}
	return -1, -1
}

// scheduleTeleEdge replaces the pending coverage-edge wake with one at the
// next change after now. An edge changes eligibility only: it claims
// nothing itself, but the dispatch after it may (an opening wakes
// ClassCapacityUp, a close ClassCapacityDown).
func (st *state) scheduleTeleEdge(e *Engine) {
	t := st.tele
	t.edgeGen++
	at, c := st.nextTeleEdge(e.Now())
	if at < 0 {
		return
	}
	class := ClassCapacityUp
	if c < 0 {
		class = ClassCapacityDown
	}
	e.Push(Event{At: at, Class: class, Kind: kindTeleEdge, Gen: t.edgeGen, B: int64(c)})
}

// teleEdge logs a coverage change and schedules the next.
func (st *state) teleEdge(e *Engine, c int) {
	if c < 0 {
		st.emit(e.Now(), api.Info, "reading", "Teleradiology coverage closes")
	} else {
		k := &st.tele.contracts[c]
		st.emit(e.Now(), api.Info, "reading", fmt.Sprintf("Teleradiology coverage opens: %s (%s, %s)", teleID(c), k.windows, k.scope))
	}
	st.scheduleTeleEdge(e)
}

// addTeleContract records a contract's coverage terms (Slice 5B's
// contract_teleradiology validates notice, plan and economics before it):
// at least one window, an activation after today, an end (if fixed) after
// the activation, and no overlap with any other contract, which must
// therefore end at or before this one starts. Lanes are never added.
func (st *state) addTeleContract(e *Engine, k teleContract) (int, error) {
	today := st.cal.DayIndex(e.Now())
	switch {
	case k.windows == 0 || k.windows&^teleAllWindows != 0:
		return -1, fmt.Errorf("windows %d: select weeknights, saturday or sunday", k.windows)
	case k.scope > teleScopeUrgent:
		return -1, fmt.Errorf("scope %d", k.scope)
	case k.from <= today:
		return -1, fmt.Errorf("activation day %d is not after today (%d)", k.from, today)
	case k.until >= 0 && k.until <= k.from:
		return -1, fmt.Errorf("end day %d is not after the activation (%d)", k.until, k.from)
	}
	if st.tele == nil {
		st.tele = &teleState{}
		for l := range st.tele.lanes {
			st.tele.lanes[l].claim = -1
		}
	}
	for c, p := range st.tele.contracts {
		if p.until < 0 || p.until > k.from {
			if k.until < 0 || k.until > p.from {
				return -1, fmt.Errorf("overlaps %s", teleID(c))
			}
		}
	}
	st.tele.contracts = append(st.tele.contracts, k)
	st.scheduleTeleEdge(e)
	return len(st.tele.contracts) - 1, nil
}

// endTeleContract fixes contract c's exclusive end day: after today and
// the activation, and not past a successor's start. Claims already made
// are untouched.
func (st *state) endTeleContract(e *Engine, c int, until int64) error {
	if st.tele == nil || c < 0 || c >= len(st.tele.contracts) {
		return fmt.Errorf("no contract %d", c)
	}
	k := &st.tele.contracts[c]
	today := st.cal.DayIndex(e.Now())
	switch {
	case until <= today:
		return fmt.Errorf("end day %d is not after today (%d)", until, today)
	case until <= k.from:
		return fmt.Errorf("end day %d is not after the activation (%d)", until, k.from)
	}
	for d, p := range st.tele.contracts {
		if d != c && p.from >= k.from && until > p.from {
			return fmt.Errorf("end day %d is past %s's start", until, teleID(d))
		}
	}
	k.until = until
	st.scheduleTeleEdge(e)
	return nil
}

// matchRemote is dispatch's last pass: each idle lane, in lane order,
// claims the first eligible unread study in worklist order, while a
// contract covers this instant.
func (st *state) matchRemote(e *Engine) {
	t := st.tele
	if t == nil || len(st.dem.worklist) == 0 {
		return
	}
	c := st.teleCovering(st.now)
	if c < 0 {
		return
	}
	var order []int
	for l := range t.lanes {
		if t.lanes[l].claim >= 0 {
			continue
		}
		if order == nil {
			order = st.worklistOrder()
		}
		k := -1
		for j, si := range order {
			if st.teleEligible(c, si) {
				k = j
				break
			}
		}
		if k < 0 {
			return
		}
		st.teleClaim(e, l, order[k], c)
		order = append(order[:k], order[k+1:]...)
	}
}

// teleClaim gives study si to lane l under contract c: the study leaves
// the ready set, the lane is occupied, and the handoff starts.
func (st *state) teleClaim(e *Engine, l, si, c int) {
	t := st.tele
	s := &st.dem.studies[si]
	x := &st.cfg.Catalog[s.exam]
	clock := simtime.ClockOf(e.Now())
	rc := len(t.claims)
	t.claims = append(t.claims, teleClaim{study: si, contract: c, lane: l, claimedAt: e.Now(),
		night: clock >= 22*simtime.Hour || clock < 8*simtime.Hour})
	ln := &t.lanes[l]
	ln.claim = rc
	ln.gen++
	s.state, s.claimedAt, s.by = studyReading, e.Now(), remoteReader(rc)
	st.emit(e.Now(), api.Info, "reading", fmt.Sprintf("%s %s %s (%s) sent to teleradiology: %s, %s · handoff %s",
		studyID(si), x.Modality, x.Name, s.priority, teleID(c), laneName(l), teleHandoff))
	e.Push(Event{At: e.Now().Add(teleHandoff), Class: ClassCompletion, Kind: kindTeleHandoff, Gen: ln.gen, A: int64(l)})
}

// teleHandoffEnd starts lane l's read: the study's own keyed duration.
func (st *state) teleHandoffEnd(e *Engine, l int) {
	ln := &st.tele.lanes[l]
	cl := &st.tele.claims[ln.claim]
	cl.readAt = e.Now()
	s := &st.dem.studies[cl.study]
	o := &st.dem.orders[s.order]
	x := &st.cfg.Catalog[s.exam]
	dur := simtime.Duration(x.ReadTime.Sample(st.hash("read", o.key.keys()...).Lane(0)))
	e.Push(Event{At: e.Now().Add(max(1, dur)), Class: ClassCompletion, Kind: kindTeleReadEnd, Gen: ln.gen, A: int64(l)})
}

// teleReadEnd releases lane l and finalizes its study's report.
func (st *state) teleReadEnd(e *Engine, l int) {
	ln := &st.tele.lanes[l]
	si := st.tele.claims[ln.claim].study
	ln.claim = -1
	ln.gen++
	st.finalize(e, si)
}

// checkTele verifies remote ownership: each occupied lane holds a claim on
// it whose study is being read by that claim; each claim is its study's
// only owner, made while its contract covered the instant and within its
// scope, and is either in progress on its lane or final.
func (st *state) checkTele() error {
	t := st.tele
	remote := 0
	for si := range st.dem.studies {
		if _, ok := st.dem.studies[si].by.remote(); ok {
			remote++
		}
	}
	if t == nil {
		if remote > 0 {
			return fmt.Errorf("%d studies read remotely with no remote state", remote)
		}
		return nil
	}
	if remote != len(t.claims) {
		return fmt.Errorf("%d remote claims, %d studies read remotely", len(t.claims), remote)
	}
	for l, ln := range t.lanes {
		if ln.claim < 0 {
			continue
		}
		if ln.claim >= len(t.claims) || t.claims[ln.claim].lane != l {
			return fmt.Errorf("%s holds claim %d of another lane", laneName(l), ln.claim)
		}
		if s := &st.dem.studies[t.claims[ln.claim].study]; s.state != studyReading {
			return fmt.Errorf("%s holds %s in state %d", laneName(l), studyID(t.claims[ln.claim].study), s.state)
		}
	}
	for c, cl := range t.claims {
		s := &st.dem.studies[cl.study]
		switch {
		case s.by != remoteReader(c):
			return fmt.Errorf("remote claim %d: %s has another reader", c, studyID(cl.study))
		case s.claimedAt != cl.claimedAt:
			return fmt.Errorf("remote claim %d: claim times differ", c)
		case !st.teleCovers(cl.contract, cl.claimedAt) || !st.teleEligible(cl.contract, cl.study):
			return fmt.Errorf("remote claim %d of %s outside %s's coverage or scope", c, studyID(cl.study), teleID(cl.contract))
		case s.state == studyReading && t.lanes[cl.lane].claim != c:
			return fmt.Errorf("remote claim %d of %s in progress off its lane", c, studyID(cl.study))
		case s.state != studyReading && s.state != studyFinal:
			return fmt.Errorf("remote claim %d of %s in state %d", c, studyID(cl.study), s.state)
		}
	}
	return nil
}
