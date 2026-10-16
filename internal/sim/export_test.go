package sim

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"

	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Test-only access to hidden state.

// DropOrdersFrom suppresses every exogenous order arriving at or after t.
// It must be called before the day containing t is scheduled.
func (w *World) DropOrdersFrom(t simtime.Time) {
	w.st.dropOrder = func(at simtime.Time) bool { return at >= t }
}

// OrderInfo is a disclosed order's exogenous attributes.
type OrderInfo struct {
	ID       string
	Origin   [4]int64
	Received simtime.Time
	Exam     string
	Priority string
	Earliest int64
	State    string
	WalkIn   bool
	LostWhy  string
}

var orderStateNames = [...]string{orderBooked: "booked", orderRecall: "recall", orderLost: "lost",
	orderCancelled: "cancelled", orderVisit: "visit", orderNoShow: "no-show", orderLeft: "left", orderImaged: "imaged",
	orderReported: "reported", orderDisplacedLost: "displaced-lost"}

// Orders lists every disclosed order (opening schedule included).
func (w *World) Orders() []OrderInfo {
	var out []OrderInfo
	for i, o := range w.st.dem.orders {
		out = append(out, OrderInfo{ID: orderID(i), Origin: [4]int64{o.key.day, o.key.src, o.key.block, o.key.k},
			Received: o.received, Exam: w.st.cfg.Catalog[o.exam].Code, Priority: o.priority.String(),
			Earliest: o.earliest, State: orderStateNames[o.state], WalkIn: o.walkIn, LostWhy: o.lostWhy})
	}
	return out
}

// ApptInfo is one appointment, active or cancelled.
type ApptInfo struct {
	ID, Order, Exam string
	Room            int
	Day             int64
	Span            Span
	Cancelled       bool
	State           string
	BookedAt        simtime.Time
	EndedAt         simtime.Time
	Visit           int
	// V0.2: behavioral attempt, replacement placement, the no-show lead
	// basis and the confirmation flag.
	Attempt, Placement, NoShowLead int64
	Confirmed                      bool
}

// Appointments lists every appointment ever booked.
func (w *World) Appointments() []ApptInfo {
	var out []ApptInfo
	for i, a := range w.st.dem.appts {
		s := w.st.dem.days[a.day].rooms[a.room][a.slot]
		out = append(out, ApptInfo{ID: apptID(i), Order: orderID(a.order), Exam: w.st.cfg.Catalog[w.st.dem.orders[a.order].exam].Code,
			Room: a.room, Day: a.day, Span: s.Span, Cancelled: a.state == apptCancelled,
			State: apptStateNames[a.state], BookedAt: a.bookedAt, EndedAt: a.endedAt, Visit: a.visit,
			Attempt: a.attempt, Placement: a.placement, NoShowLead: a.noShowLead, Confirmed: a.confirmed})
	}
	return out
}

// VisitInfo is one visit.
type VisitInfo struct {
	ID, Order  string
	Appt       int
	WalkIn     bool
	Exam       string
	Arrived    simtime.Time
	Late       simtime.Duration
	AfterGrace bool
	State      string
	EndedAt    simtime.Time
	ExamStart  simtime.Time
	ExamEnd    simtime.Time
	Reason     string
	Supervisor string // radiologist supervising a contrast exam
	Registered bool
	RegStart   simtime.Time
}

var visitStateNames = [...]string{visitArrived: "arrived", visitRegistering: "registering", visitWaiting: "waiting",
	visitInExam: "in exam", visitDone: "done", visitLeft: "left", visitRescheduled: "rescheduled"}

// Visits lists every visit.
func (w *World) Visits() []VisitInfo {
	var out []VisitInfo
	for i, v := range w.st.dem.visits {
		o := &w.st.dem.orders[v.order]
		out = append(out, VisitInfo{ID: visitID(i), Order: orderID(v.order), Appt: v.appt, WalkIn: o.walkIn,
			Exam: w.st.cfg.Catalog[o.exam].Code, Arrived: v.arrived, Late: v.late, AfterGrace: v.afterGrace,
			State: visitStateNames[v.state], EndedAt: v.endedAt, ExamStart: v.examStart, ExamEnd: v.examEnd,
			Reason: v.reason, Supervisor: w.supervisorName(v.supervisor), Registered: v.regBy >= 0, RegStart: v.regStart})
	}
	return out
}

// ForceNoShowsFrom makes every patient whose appointment starts at or after
// t a no-show. Call before the day containing t starts.
func (w *World) ForceNoShowsFrom(t simtime.Time) {
	w.st.forceNoShow = func(start simtime.Time) bool { return start >= t }
}

// PushForTest queues a raw event (for same-timestamp ordering tests).
func (w *World) PushForTest(at simtime.Time, class Class, kind string, a int64) {
	k := map[string]Kind{"arrive": kindArrive, "grace": kindGrace, "cancel": kindCancel, "patience": kindPatience}[kind]
	w.eng.Push(Event{At: at, Class: class, Kind: k, A: a})
}

// TryBook books a test order (exam code, preferred window, earliest day
// index) through FirstFit and returns the room and slot start it got.
func (w *World) TryBook(exam string, window Window, earliest int64) (room int, from simtime.Duration, ok bool) {
	_, room, from, ok = w.TryBookDay(exam, window, earliest)
	return room, from, ok
}

// TryBookDay is TryBook, also returning the day index booked.
func (w *World) TryBookDay(exam string, window Window, earliest int64) (day int64, room int, from simtime.Duration, ok bool) {
	st := w.st
	xi := -1
	for i, x := range st.cfg.Catalog {
		if x.Code == exam {
			xi = i
		}
	}
	st.dem.patients++
	oi := len(st.dem.orders)
	st.dem.orders = append(st.dem.orders, order{key: originKey{-2, -2, -2, int64(oi)}, patient: st.dem.patients,
		source: -1, exam: xi, window: window, received: w.eng.Now(), earliest: earliest, tolerance: 1000, appt: -1, visit: -1})
	pref, any, hasPref, hasAny := st.firstFit(&st.dem.orders[oi], earliest, w.eng.Now())
	pick := any
	switch {
	case hasPref:
		pick = pref
	case !hasAny:
		return 0, 0, 0, false
	}
	st.book(w.eng, oi, pick, true)
	return pick.day, pick.room, st.slotAt(pick).From, true
}

// StaffInfo is an employee's day so far.
type StaffInfo struct {
	Name       string
	OnDuty     bool
	Held       bool
	BreakTaken simtime.Duration
	Overtime   simtime.Duration // booked when they went home
}

// Staff lists every employee.
func (w *World) Staff() []StaffInfo {
	var out []StaffInfo
	for _, e := range w.st.emps {
		out = append(out, StaffInfo{Name: e.cfg.Name, OnDuty: e.status != offDuty, Held: e.held,
			BreakTaken: e.breakTaken, Overtime: e.otToday})
	}
	return out
}

// PushLiveNoise queues an event that is handled (so dispatch runs) but
// changes nothing: a roster edge for employee 0 in their current
// generation. Outcomes must not depend on such unrelated events.
func (w *World) PushLiveNoise(at simtime.Time) {
	w.eng.Push(Event{At: at, Class: ClassCapacityUp, Kind: kindRosterEdge, Gen: w.st.emps[0].gen, A: 0})
}

// PushStaleNoise queues an event that is stale on arrival.
func (w *World) PushStaleNoise(at simtime.Time) {
	w.eng.Push(Event{At: at, Class: ClassDeadline, Kind: kindWake, Gen: 1 << 31, A: 0})
}

// StudyInfo is one study.
type StudyInfo struct {
	ID, Visit, Exam, Priority         string
	State                             string
	Acquired, Entered, Claimed, Final simtime.Time
	By                                string
	ReadTime                          simtime.Duration // final − claimed, once final
}

var studyStateNames = [...]string{studyAcquired: "acquired", studyUnread: "unread", studyReading: "reading", studyFinal: "final"}

// Studies lists every study.
func (w *World) Studies() []StudyInfo {
	var out []StudyInfo
	for i, s := range w.st.dem.studies {
		x := StudyInfo{ID: studyID(i), Visit: visitID(s.visit), Exam: w.st.cfg.Catalog[s.exam].Code,
			Priority: s.priority.String(), State: studyStateNames[s.state], Acquired: s.acquiredAt,
			Entered: s.enteredAt, Claimed: s.claimedAt, Final: s.finalAt}
		if s.by >= 0 {
			x.By = w.st.emps[s.by].cfg.Name
		}
		if s.state == studyFinal {
			x.ReadTime = s.finalAt.Sub(s.claimedAt)
		}
		out = append(out, x)
	}
	return out
}

func (w *World) supervisorName(i int) string {
	if i < 0 {
		return ""
	}
	return w.st.emps[i].cfg.Name
}

// EntryInfo is one journal entry with its key spelled out.
type EntryInfo struct {
	Index    int
	At       simtime.Time
	Kind     string
	A, B     int64
	Lines    []LineInfo
	Reversed bool
}

// LineInfo is one posting; Amt > 0 is a debit.
type LineInfo struct {
	Account  string
	Modality string
	Amt      int64
}

var entryKindNames = map[entryKind]string{entOpening: "opening", entRevenue: "revenue", entStudyCost: "study_cost",
	entPayment: "payment", entSettle: "settle", entWages: "wages", entSalary: "salary", entFixed: "fixed",
	entPayroll: "payroll", entPayAP: "pay_ap", entReversal: "reversal", entGuarantee: "guarantee",
	entRecruiting: "recruiting", entSeverance: "severance", entSeverancePay: "severance_pay"}

// OpenGuarantee is the guaranteed pay provisioned this week and not yet
// final (wages and benefits).
func (w *World) OpenGuarantee() int64 { return w.st.openGuarantee() }

// Journal lists every posted entry.
func (w *World) Journal() []EntryInfo {
	var out []EntryInfo
	for i, e := range w.st.fin.led.entries {
		ei := EntryInfo{Index: i, At: e.at, Kind: entryKindNames[e.key.kind], A: e.key.a, B: e.key.b, Reversed: e.reversed >= 0}
		for _, ln := range e.lines {
			li := LineInfo{Account: accountInfo[ln.acct].id, Amt: ln.amt}
			if ln.mod >= 0 {
				li.Modality = w.st.mods[ln.mod]
			}
			ei.Lines = append(ei.Lines, li)
		}
		out = append(out, ei)
	}
	return out
}

// Balance returns an account's debit-positive balance.
func (w *World) Balance(id string) int64 {
	for a := range numAccounts {
		if accountInfo[a].id == id {
			return w.st.fin.led.bal[a]
		}
	}
	panic("no account " + id)
}

// ClaimInfo is a claim with its timestamps.
type ClaimInfo struct {
	ID, Study, Exam, Payer, State                 string
	Origin                                        [4]int64
	Allowed, Expected, Paid                       int64
	Final, Submitted, Denied, Resubmitted, Closed simtime.Time
	WasDenied                                     bool
}

// Claims lists every claim.
func (w *World) Claims() []ClaimInfo {
	st := w.st
	var out []ClaimInfo
	for ci, c := range st.fin.claims {
		o := &st.dem.orders[c.order]
		out = append(out, ClaimInfo{ID: claimID(ci), Study: studyID(c.study), Exam: st.cfg.Catalog[o.exam].Code,
			Payer: st.cfg.Finance.Payers[c.payer].ID, State: claimStateNames[c.state],
			Origin: [4]int64{o.key.day, o.key.src, o.key.block, o.key.k}, Allowed: c.allowed, Expected: c.expected,
			Paid: c.paid, Final: c.finalAt, Submitted: c.submittedAt, Denied: c.deniedAt, Resubmitted: c.resubmittedAt,
			Closed: c.closedAt, WasDenied: c.denied})
	}
	return out
}

// DrawClaim returns the latent payer draw for an origin key and payer.
func (w *World) DrawClaim(origin [4]int64, payer int) (outcome int, delayDays, fracBP, reworkDays int64) {
	d := w.st.claimDraw(originKey{origin[0], origin[1], origin[2], origin[3]}, payer)
	return d.outcome, d.delayDays, d.fracBP, d.reworkDays
}

// Claim outcomes for DrawClaim.
const (
	OutPaid       = outPaid
	OutShort      = outShort
	OutRecovered  = outRecovered
	OutWrittenOff = outWrittenOff
)

// ForceDeniedClaims makes every claim denied and written off: a change to
// hidden latent payer outcomes.
func (w *World) ForceDeniedClaims() { w.st.fin.forceDenied = func(int) bool { return true } }

// OverheadForTest exposes the overtime cost recorded so far (all days).
func (w *World) OvertimeCostTotal() int64 { return w.st.dem.flowTotal.otCost }

// Check runs the state invariants now.
func (w *World) Check() error { return w.st.CheckInvariants() }

// DrawChargeLag returns the latent charge lag (business days) for an origin key.
func (w *World) DrawChargeLag(origin [4]int64) int64 {
	key := originKey{origin[0], origin[1], origin[2], origin[3]}
	return uniformIn(rng.Hash(w.st.seed, "claim.lag", key.keys()...).Lane(0), w.st.cfg.Finance.ChargeLagDays)
}

// WeekPaid returns an employee's paid time so far this week.
func (w *World) WeekPaid(name string) simtime.Duration {
	for i := range w.st.emps {
		if w.st.emps[i].cfg.Name == name {
			return w.st.emps[i].weekPaid
		}
	}
	panic("no employee " + name)
}

// ForceDeniedClaimsFrom makes every claim adjudicated at or after t fail:
// a change to hidden payer outcomes from t on.
func (w *World) ForceDeniedClaimsFrom(t simtime.Time) {
	w.st.fin.forceDenied = func(int) bool { return w.st.now >= t }
}

// Config returns the world's configuration.
func (w *World) Config() *Config { return w.st.cfg }

// StateDigestParts digests each top-level state field on its own (to find
// what differs).
func (w *World) StateDigestParts() map[string]string {
	out := map[string]string{}
	v := reflect.ValueOf(w.st).Elem()
	for _, f := range fieldsOf(v.Type()) {
		d := &stateHasher{h: sha256.New()}
		d.value(v.Field(f.i))
		d.flush()
		out[v.Type().Field(f.i).Name] = hex.EncodeToString(d.h.Sum(nil))[:12]
	}
	return out
}

// StudyOrders lists the order of every study; ClaimOrders that of every
// claim.
func (w *World) StudyOrders() []string {
	var out []string
	for _, s := range w.st.dem.studies {
		out = append(out, orderID(s.order))
	}
	return out
}

func (w *World) ClaimOrders() []string {
	var out []string
	for _, c := range w.st.fin.claims {
		out = append(out, orderID(c.order))
	}
	return out
}

// DisplacementInfo is one practice-caused displacement record.
type DisplacementInfo struct {
	Appt, Order, Cause, Reason, Resolution, Replacement string
	Actions                                             []int
	At                                                  simtime.Time
}

// Displacements lists the displacement records in order.
func (w *World) Displacements() []DisplacementInfo {
	var out []DisplacementInfo
	for _, x := range w.st.dem.displaced {
		d := DisplacementInfo{Appt: apptID(x.Appt), Order: orderID(x.Order), Cause: x.Cause, Reason: x.Reason,
			Resolution: x.Resolution, Actions: x.Actions, At: x.At}
		if x.Replacement >= 0 {
			d.Replacement = apptID(x.Replacement)
		}
		out = append(out, d)
	}
	return out
}

// OrderAnchor is an order's tolerance anchor (day index) and tolerance.
func (w *World) OrderAnchor(id string) (anchor, tolerance int64) {
	for i := range w.st.dem.orders {
		if orderID(i) == id {
			return w.st.dem.orders[i].anchor, w.st.dem.orders[i].tolerance
		}
	}
	panic("no order " + id)
}

// RegistryLen is the number of employment records.
func (w *World) RegistryLen() int { return len(w.st.staff) }

// SlotStaffWords is the longest slot staff set (in 64-bit words, the
// inline one included) over the materialized days.
func (w *World) SlotStaffWords() int {
	n := 0
	for _, ds := range w.st.dem.days {
		for r := range ds.rooms {
			for _, s := range ds.rooms[r] {
				n = max(n, 1+len(s.staff.hi))
			}
		}
	}
	return n
}

// RosterLogLen is the number of operations in employee id's roster log.
func (w *World) RosterLogLen(id string) int { return len(w.st.staff[w.st.findEmployee(id)].RosterLog) }
