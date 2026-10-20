package eval

import (
	"fmt"
	"maps"
	"slices"
)

// Order provenance (Slice 5C.4l, audit 5C.4k K1 and K2; owner decision
// of 2026-10-20).
//
// A record's own fields cannot establish an order's identity: K1 relinked
// a violating study to another order of the same request and re-timed
// that order consistently, and K2 deleted, relabelled or re-timed
// inherited work, each leaving one record internally consistent. Under
// paired exogenous demand both plans of a seed receive the same orders:
// the same origins (the demand draw that created each order), with the
// same requests, and the same inherited orders open at the takeover (the
// takeover state precedes every plan). So the paired plan's record is the
// authority for a candidate's orders: every order of either record must
// have its origin in the other, with the same immutable request. Outcomes
// (bookings, resolutions, losses, cancellations, studies, reports) may
// differ and are never compared.
//
// What this does not establish, stated rather than assumed:
//
//   - An order present in one record only has no trusted provenance: no
//     plan creates demand in the simulator or the fidelity model, and no
//     generation record outside the records exists, so such an order is
//     a provenance failure (INVALID), never accepted.
//   - The paired record is trusted as produced by the harness (in process
//     for the reserved run, by committed record digests for a file-based
//     evaluation): editing both records of a pair alike is not
//     detectable here (audit 5C.4k K3).
//   - A study's link to its order is an outcome of the candidate's record
//     and is not reconciled: see the protocol's limitations.

// provenanceRequest is an order's immutable request.
type provenanceRequest struct {
	id, priority, modality string
	received, wanted       int64
	walkIn                 bool
}

// orderProvenance reconciles the orders of record a (the candidate) with
// those of record b (the paired plan) of the same seed; each failure is a
// message, the first few in order of origin, then their number.
func orderProvenance(a, b *Record) []string {
	index := func(r *Record) (map[string]provenanceRequest, []string) {
		m := map[string]provenanceRequest{}
		var bad []string
		for _, o := range r.Orders {
			if o.Origin == "" {
				bad = append(bad, fmt.Sprintf("order %s of plan %s has no origin", o.ID, r.Plan))
				continue
			}
			if _, dup := m[o.Origin]; dup {
				bad = append(bad, fmt.Sprintf("origin %s repeats in plan %s", o.Origin, r.Plan))
				continue
			}
			m[o.Origin] = provenanceRequest{o.ID, o.Priority, o.Modality, o.ReceivedSec, o.WantedSec, o.WalkIn}
		}
		return m, bad
	}
	ma, bad := index(a)
	mb, badB := index(b)
	bad = append(bad, badB...)
	for _, k := range slices.Sorted(maps.Keys(ma)) {
		y, ok := mb[k]
		switch x := ma[k]; {
		case !ok:
			bad = append(bad, fmt.Sprintf("order %s (origin %s) is in plan %s only: no trusted provenance", x.id, k, a.Plan))
		case x != y:
			bad = append(bad, fmt.Sprintf("order %s (origin %s): request %s, the paired plan's %s", x.id, k, x, y))
		}
	}
	for _, k := range slices.Sorted(maps.Keys(mb)) {
		if _, ok := ma[k]; !ok {
			bad = append(bad, fmt.Sprintf("order %s (origin %s) of plan %s is missing from plan %s", mb[k].id, k, b.Plan, a.Plan))
		}
	}
	if len(bad) > 5 {
		bad = append(bad[:5], fmt.Sprintf("%d provenance failures in all", len(bad)))
	}
	return bad
}

func (x provenanceRequest) String() string {
	return fmt.Sprintf("{id %s, %s %s, received %d, wanted %d, walk-in %v}", x.id, x.priority, x.modality, x.received, x.wanted, x.walkIn)
}

// provenanceApplies: the policy reconciles orders by origin.
func provenanceApplies(p *Policy) bool {
	return p.Clinical != nil && p.Clinical.OrderProvenance == ProvenancePairedOrigin
}
