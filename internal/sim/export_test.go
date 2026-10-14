package sim

import "github.com/georgezero/dreamteam-go/internal/simtime"

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
}

var orderStateNames = [...]string{orderBooked: "booked", orderRecall: "recall", orderLost: "lost",
	orderCancelled: "cancelled", orderVisit: "visit", orderNoShow: "no-show", orderLeft: "left", orderImaged: "imaged"}

// Orders lists every disclosed order (opening schedule included).
func (w *World) Orders() []OrderInfo {
	var out []OrderInfo
	for i, o := range w.st.dem.orders {
		out = append(out, OrderInfo{ID: orderID(i), Origin: [4]int64{o.key.day, o.key.src, o.key.block, o.key.k},
			Received: o.received, Exam: w.st.cfg.Catalog[o.exam].Code, Priority: o.priority.String(),
			Earliest: o.earliest, State: orderStateNames[o.state]})
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
}

// Appointments lists every appointment ever booked.
func (w *World) Appointments() []ApptInfo {
	var out []ApptInfo
	for i, a := range w.st.dem.appts {
		s := w.st.dem.days[a.day].rooms[a.room][a.slot]
		out = append(out, ApptInfo{ID: apptID(i), Order: orderID(a.order), Exam: w.st.cfg.Catalog[w.st.dem.orders[a.order].exam].Code,
			Room: a.room, Day: a.day, Span: s.Span, Cancelled: a.state == apptCancelled,
			State: apptStateNames[a.state], BookedAt: a.bookedAt, EndedAt: a.endedAt, Visit: a.visit})
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
			Reason: v.reason})
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
