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

var orderStateNames = [...]string{orderBooked: "booked", orderRecall: "recall", orderLost: "lost", orderCancelled: "cancelled"}

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
}

// Appointments lists every appointment ever booked.
func (w *World) Appointments() []ApptInfo {
	var out []ApptInfo
	for i, a := range w.st.dem.appts {
		s := w.st.dem.days[a.day].rooms[a.room][a.slot]
		out = append(out, ApptInfo{ID: apptID(i), Order: orderID(a.order), Exam: w.st.cfg.Catalog[w.st.dem.orders[a.order].exam].Code,
			Room: a.room, Day: a.day, Span: s.Span, Cancelled: a.state == apptCancelled})
	}
	return out
}
