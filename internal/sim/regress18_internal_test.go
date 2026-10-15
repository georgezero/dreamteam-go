package sim

import (
	"reflect"
	"testing"
	"unsafe"

	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Step 18 review finding 2: booking with shared technologists accepts a
// slot exactly when every overlapping booked appointment, and the new one,
// can each be given a distinct qualified technologist (handovers only at
// coverage boundaries), whatever order they were booked in.

const (
	tRiley = 0 // XR (and CT where the roster says so)
	tSam   = 1 // CT, and XR where the roster says so
	rXR    = 0
	rCT    = 1
)

func clock(h, m int) simtime.Duration {
	return simtime.Duration(h)*simtime.Hour + simtime.Duration(m)*simtime.Minute
}

// testDay builds a day of XR 15-min and CT 30-min slots from 16:00 to 18:00
// over the given coverage segments, with slot staff sets and sharing as
// coverSlot computes them.
func testDay(segs []Segment) *daySlots {
	ds := &daySlots{segs: segs, rooms: make([][]slot, 2)}
	add := func(r int, len simtime.Duration) {
		for at := clock(16, 0); at+len <= clock(18, 0); at += len {
			s := slot{SlotTemplate: SlotTemplate{Span: Span{at, at + len}}, appt: -1}
			for _, seg := range segs {
				if seg.To <= s.From || seg.From >= s.To {
					continue
				}
				var here empSet
				for _, e := range seg.Rooms[r] {
					here = here.with(e)
				}
				s.staff = s.staff.union(here)
				for r2 := range seg.Rooms {
					for _, e := range seg.Rooms[r2] {
						if r2 != r && here.has(e) {
							s.shared = true
						}
					}
				}
			}
			ds.rooms[r] = append(ds.rooms[r], s)
		}
	}
	add(rXR, 15*simtime.Minute)
	add(rCT, 30*simtime.Minute)
	return ds
}

func slotAtClock(ds *daySlots, r int, from simtime.Duration) *slot {
	for k := range ds.rooms[r] {
		if ds.rooms[r][k].From == from {
			return &ds.rooms[r][k]
		}
	}
	panic("no slot")
}

// book places an appointment if the slot is free by the shared-staff rule.
func book(ds *daySlots, r int, from simtime.Duration) bool {
	s := slotAtClock(ds, r, from)
	if !sharedFree(ds, r, s) {
		return false
	}
	s.appt = 1
	return true
}

func seg(from, to simtime.Duration, xr, ct []int) Segment {
	return Segment{Span: Span{from, to}, Open: true, Rooms: [][]int{xr, ct}}
}

func TestRegress18SharedStaffFeasibility(t *testing.T) {
	// Riley extended past 16:15: XR by Riley or Sam, CT by Sam only.
	both := []Segment{seg(clock(16, 0), clock(18, 0), []int{tRiley, tSam}, []int{tSam})}
	// One technologist for both rooms.
	alone := []Segment{seg(clock(16, 0), clock(18, 0), []int{tSam}, []int{tSam})}
	// Handover at 17:00: Riley alone on XR and CT, then Riley XR and Sam CT
	// (Sam also XR).
	handover := []Segment{seg(clock(16, 0), clock(17, 0), []int{tRiley}, []int{tRiley}),
		seg(clock(17, 0), clock(18, 0), []int{tRiley, tSam}, []int{tSam})}

	type step struct {
		room int
		from simtime.Duration
		ok   bool
	}
	for _, c := range []struct {
		name  string
		segs  []Segment
		steps []step
	}{
		{"XR then CT", both, []step{{rXR, clock(16, 0), true}, {rCT, clock(16, 0), true}}},
		{"CT then XR", both, []step{{rCT, clock(16, 0), true}, {rXR, clock(16, 0), true}}},
		{"two XR then CT", both, []step{{rXR, clock(16, 0), true}, {rXR, clock(16, 15), true}, {rCT, clock(16, 0), true}}},
		{"CT then two XR", both, []step{{rCT, clock(16, 0), true}, {rXR, clock(16, 0), true}, {rXR, clock(16, 15), true}}},
		{"one technologist", alone, []step{{rXR, clock(16, 0), true}, {rCT, clock(16, 0), false}, {rXR, clock(16, 30), true}, {rCT, clock(16, 30), false}}},
		{"one technologist, CT first", alone, []step{{rCT, clock(16, 0), true}, {rXR, clock(16, 0), false}, {rXR, clock(16, 15), false}, {rXR, clock(16, 30), true}}},
		{"handover: before 17:00 one tech", handover, []step{{rXR, clock(16, 30), true}, {rCT, clock(16, 30), false}}},
		{"handover: after 17:00 two", handover, []step{{rXR, clock(17, 0), true}, {rCT, clock(17, 0), true}, {rXR, clock(17, 15), true}}},
		{"handover: CT across the change", []Segment{seg(clock(16, 0), clock(16, 45), []int{tRiley}, []int{tRiley}),
			seg(clock(16, 45), clock(18, 0), []int{tRiley, tSam}, []int{tSam})},
			// XR 16:30 (Riley) and CT 16:30–17:00 (Riley until 16:45, then Sam) cannot
			// both hold: Riley would be in both rooms until 16:45.
			[]step{{rXR, clock(16, 30), true}, {rCT, clock(16, 30), false}, {rXR, clock(16, 45), true}}},
		{"XR swap would be needed", both,
			// CT 16:00 takes Sam; XR 16:00 and 16:15 Riley; nothing else fits 16:00–16:30.
			[]step{{rCT, clock(16, 0), true}, {rXR, clock(16, 0), true}, {rXR, clock(16, 15), true}, {rCT, clock(16, 30), true}, {rXR, clock(16, 30), true}}},
	} {
		ds := testDay(c.segs)
		for i, st := range c.steps {
			if got := book(ds, st.room, st.from); got != st.ok {
				t.Errorf("%s, step %d (%s %s): booked %v, want %v", c.name, i+1, []string{"XR", "CT"}[st.room],
					simtime.FormatClock(st.from), got, st.ok)
			}
		}
	}
}

// Every flowCounts field is folded by add and negated by scale: a counter
// missing from either would silently vanish from closed days or periods.
func TestRegress18FlowCountsFieldsFolded(t *testing.T) {
	// Integer fields, and each element of array fields (V0.2: the
	// displacement counters), get distinct values.
	var g flowCounts
	v := reflect.ValueOf(&g).Elem()
	n := int64(0)
	for i := 0; i < v.NumField(); i++ {
		f := reflect.NewAt(v.Field(i).Type(), unsafe.Pointer(v.Field(i).UnsafeAddr())).Elem()
		if f.Kind() == reflect.Array {
			for k := 0; k < f.Len(); k++ {
				n++
				f.Index(k).SetInt(n)
			}
			continue
		}
		n++
		f.SetInt(n)
	}
	var f flowCounts
	f.add(&g)
	f.add(&g)
	f.sub(&g)
	if f != g {
		t.Errorf("flowCounts add/sub: got %+v, want %+v", f, g)
	}
}
