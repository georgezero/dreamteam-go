package sim

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"math"
	"reflect"
	"slices"
	"sync"
)

// Verification digests (SIMULATION_DESIGN §4.5, as built in step 14).
//
//   - Log: the running hash of every observable log entry (log.go).
//   - Ledger: every journal entry in posting order (ledger.go).
//   - Actions: the canonical JSON of the action record, results included.
//   - State: a canonical walk of the authoritative simulation state,
//     hidden fields included — every order, appointment, visit, study,
//     claim (with its latent payer draw), employee, room, the ledger's
//     balances and day books, metric tallies, alert and evaluation state,
//     policies and approved extensions — plus the pending events that are
//     not stale, by (time, class, kind, generation, arguments).
//
// The state walk skips what is not state: the configuration (fingerprinted
// with the scenario), caches that depend on what has been read (draw memo,
// completed-day summaries, Poisson tables, the ledger's key index and
// reconciliation shadow), scratch buffers, test hooks, the log (hashed on
// its own) and the action record (its own digest). It leaves out event
// sequence numbers and engine statistics, which count processing, not
// outcomes. Integers are 64-bit big-endian, strings length-prefixed, map
// entries sorted by their encoding: the digest is the same on every
// platform. A digest is never part of an observation.

// StateSchema names the state digest's encoding. v2 (V0.2 Slice 1) adds
// the employment registry: identity, lifecycle, contracts and the week's
// guaranteed-pay counters. v3 (V0.2 Slice 2) adds hires (terms, offer,
// training), terminations (last day, severance), the productive date,
// growable employee sets in slot staffing, practice-caused displacement
// records and their counters, the new ledger accounts, and the tolerance
// anchor, placement and booking-lead fields of orders and appointments.
// Outcomes of a V0.1 scenario without V0.2 actions are unchanged; only
// state and run digests differ.
//
// V0.2 Slice 3 extends v3 without changing any existing encoding: an
// employee's roster versions and roster log (set_roster) are encoded only
// when non-empty, introduced by a tag no length prefix can equal, so a
// state without them encodes exactly as Slice 2's did (omitEmpty). Slice
// 4 adds a radiologist's allocation versions and log
// (set_reading_allocation) the same way, and Slice 5A (0.2.0-dev.7) the
// remote reading state (contracts, lanes and claims, state.tele), a
// pointer encoded only when non-nil: without a teleradiology contract a
// state encodes exactly as before. A study's reader keeps its single
// integer encoding (reading.go).
const StateSchema = "dreamteam.state/v3"

// omitEmpty names slice and pointer fields, by "Type.field", that are
// encoded only when non-empty (non-nil), after omitTag: fields added to
// the v3 encoding.
var omitEmpty = map[string]bool{"employment.Rosters": true, "employment.RosterLog": true,
	"employment.Allocs": true, "employment.AllocLog": true, "state.tele": true,
	"finState.credit": true}

// acctArrays names account-indexed array fields, by "Type.field". Their
// first numV3Accounts elements encode as the array did under v3; the
// accounts appended since (Slice 5B's teleradiology accounts) follow,
// after omitTag, only when one is non-zero.
var acctArrays = map[string]bool{"ledger.bal": true, "dayBook.net": true, "dayBook.byMod": true}

// omitTag introduces a present omitEmpty field. It has the top bit set,
// which no length prefix (string, slice or map) can.
const omitTag = 0xD7A1_0000_0000_0003

// skipFields names struct fields the state walk leaves out, by
// "Type.field".
var skipFields = map[string]bool{
	"state.cfg": true, "state.log": true, "state.drawMemo": true, "state.staffBuf": true,
	"state.dropOrder": true, "state.forceNoShow": true, "state.lossProbe": true, "state.actions": true,
	"state.now":            true, // the time of the last transaction: when processing happened, not what
	"state.dev5":           true, // replay configuration (SetDev5Rules), not state
	"metricsState.done":    true,
	"finState.forceDenied": true,
	"ledger.keys":          true, "ledger.verified": true, "ledger.shadow": true, "ledger.cal": true,
	"demandState.poisson": true, "demandState.catIdx": true,
	"employee.cfg": true, "employee.terms": true, "room.cfg": true,
	"DayPlan.cfg": true, "DayPlan.emps": true, "Coverage.Plan": true,
	"employment.trainView": true, // derived from the hire's terms
	"alertState.cfg":       true, "alertState.wakeGen": true, "alertState.wakeAt": true,
	"metricsState.cum":        true, // digested as a snapshot at the current time
	"tally.actions":           true, // the action record's length: the action digest's
	"rosterVersion.trainView": true, // derived from the version's terms
	"allocVersion.view":       true, // derived from the radiologist's terms
	"DayPlan.cutReading":      true, // derived from the plan and the scenario
}

// stateHasher writes the canonical encoding into a buffer flushed to the
// hash in large blocks.
type stateHasher struct {
	h   hash.Hash
	buf []byte
}

func (d *stateHasher) flush() {
	d.h.Write(d.buf)
	d.buf = d.buf[:0]
}

func (d *stateHasher) u64(v uint64) {
	d.buf = binary.BigEndian.AppendUint64(d.buf, v)
	if d.h != nil && len(d.buf) >= 1<<16 {
		d.flush()
	}
}

func (d *stateHasher) str(s string) {
	d.u64(uint64(len(s)))
	d.buf = append(d.buf, s...)
}

// fieldsOf lists a struct type's encoded fields (cached).
var (
	fieldMu    sync.Mutex
	fieldCache = map[reflect.Type][]field{}
)

// field is an encoded struct field: its index, and whether it is omitted
// when empty.
type field struct {
	i          int
	omit, acct bool
}

func fieldsOf(t reflect.Type) []field {
	fieldMu.Lock()
	defer fieldMu.Unlock()
	if fs, ok := fieldCache[t]; ok {
		return fs
	}
	var fs []field
	for i := 0; i < t.NumField(); i++ {
		switch t.Field(i).Type.Kind() {
		case reflect.Func, reflect.Interface, reflect.Chan, reflect.UnsafePointer:
			continue // not state: hooks, hash states (digested on their own)
		}
		name := t.Name() + "." + t.Field(i).Name
		if !skipFields[name] {
			fs = append(fs, field{i, omitEmpty[name], acctArrays[name]})
		}
	}
	fieldCache[t] = fs
	return fs
}

func (d *stateHasher) value(v reflect.Value) {
	switch v.Kind() {
	case reflect.Bool:
		if v.Bool() {
			d.u64(1)
		} else {
			d.u64(0)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		d.u64(uint64(v.Int()))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		d.u64(v.Uint())
	case reflect.Float32, reflect.Float64:
		d.u64(math.Float64bits(v.Float()))
	case reflect.String:
		d.str(v.String())
	case reflect.Pointer:
		if v.IsNil() {
			d.u64(0)
			return
		}
		d.u64(1)
		d.value(v.Elem())
	case reflect.Slice, reflect.Array:
		n := v.Len()
		d.u64(uint64(n))
		for i := 0; i < n; i++ {
			d.value(v.Index(i))
		}
	case reflect.Map:
		type kv struct{ k, v []byte }
		var es []kv
		for it := v.MapRange(); it.Next(); {
			es = append(es, kv{sub(it.Key()), sub(it.Value())})
		}
		slices.SortFunc(es, func(a, b kv) int { return bytes.Compare(a.k, b.k) })
		d.u64(uint64(len(es)))
		for _, e := range es {
			d.buf = append(append(d.buf, e.k...), e.v...)
		}
	case reflect.Struct:
		for _, f := range fieldsOf(v.Type()) {
			x := v.Field(f.i)
			if f.omit {
				if x.Kind() == reflect.Pointer && x.IsNil() || x.Kind() != reflect.Pointer && x.Len() == 0 {
					continue
				}
				d.u64(omitTag)
			}
			if f.acct {
				d.accounts(x)
				continue
			}
			d.value(x)
		}
	case reflect.Func, reflect.Interface, reflect.Chan, reflect.UnsafePointer:
	default:
		panic(fmt.Sprintf("sim: state digest cannot encode %s", v.Kind()))
	}
}

// accounts encodes an account-indexed array (acctArrays): the v3 chart,
// then each group appended since (Slice 5B's, V0.7b's) after an omitTag,
// only when one of its accounts is non-zero.
func (d *stateHasher) accounts(x reflect.Value) {
	n := int(numV3Accounts)
	d.u64(uint64(n))
	for i := 0; i < n; i++ {
		d.value(x.Index(i))
	}
	for _, g := range [][2]int{{int(numV3Accounts), int(numV5Accounts)}, {int(numV5Accounts), x.Len()}} {
		for i := g[0]; i < g[1]; i++ {
			if !x.Index(i).IsZero() {
				d.u64(omitTag)
				d.u64(uint64(g[1] - g[0]))
				for k := g[0]; k < g[1]; k++ {
					d.value(x.Index(k))
				}
				break
			}
		}
	}
}

// sub encodes v on its own (for sorting map entries).
func sub(v reflect.Value) []byte {
	d := &stateHasher{} // no hash: the whole encoding stays in the buffer
	d.value(v)
	return d.buf
}

// StateDigest hashes the authoritative state and the live pending events.
func (w *World) StateDigest() string {
	d := &stateHasher{h: sha256.New()}
	d.str(StateSchema)
	d.u64(uint64(w.eng.Now()))
	d.value(reflect.ValueOf(w.st).Elem())
	// The interval tally as of now: where it was last accrued is an
	// artifact of when transactions happened, its totals are not.
	snap := w.st.snapshot(w.eng.Now())
	d.value(reflect.ValueOf(&snap).Elem())
	// Live pending events; passive ones (alert wakes, the evaluation end)
	// are bookkeeping derived from the state and change no outcome.
	evs := w.eng.PendingEvents()
	var live []Event
	for _, ev := range evs {
		if !w.st.IsStale(ev) && !w.st.Passive(ev) {
			live = append(live, ev)
		}
	}
	d.u64(uint64(len(live)))
	for _, ev := range live {
		d.u64(uint64(ev.At))
		d.u64(uint64(ev.Class))
		d.u64(uint64(ev.Kind))
		d.u64(uint64(ev.Gen))
		d.u64(uint64(ev.A))
		d.u64(uint64(ev.B))
	}
	d.flush()
	return hex.EncodeToString(d.h.Sum(nil))
}

// ActionDigest hashes the canonical JSON of the action record.
func (w *World) ActionDigest() string {
	raw, err := json.Marshal(w.ActionLog())
	if err != nil {
		panic(err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
