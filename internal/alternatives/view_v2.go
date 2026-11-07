package alternatives

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/session"
	"github.com/georgezero/dreamteam-go/internal/sim"
)

const ViewV2Schema = "dreamteam.alternatives-view/v2"

// All new quantities retain their native unit and explicit horizon. A nil
// quantity is unavailable, never an inferred zero.
type QuantityV2 struct {
	Number *int64 `json:"number"`
	Unit   string `json:"unit"`
	From   string `json:"from"`
	To     string `json:"to"`
}

type CommitmentV2 struct {
	Ref                 string `json:"ref"`
	PublicPosition      int    `json:"publicPosition"`
	BaselineGuarantee   *int64 `json:"baselineGuarantee"`
	ContinuingGuarantee *int64 `json:"continuingGuarantee"`
	GuaranteeDelta      *int64 `json:"guaranteeDelta"`
	BaselineExposure    *int64 `json:"baselineExposure"`
	ConditionalExposure *int64 `json:"conditionalExposure"`
	ExposureDelta       *int64 `json:"exposureDelta"`
	Through             string `json:"through,omitempty"`
	Terminating         bool   `json:"terminating"`
}

type CapacityV2 struct {
	Closed           QuantityV2          `json:"closed"`
	Opened           QuantityV2          `json:"opened"`
	Displaced        QuantityV2          `json:"displaced"`
	Replacements     QuantityV2          `json:"replacements"`
	ClosedByModality []api.ModalityCount `json:"closedByModality"`
}

type ReversalV2 struct {
	Action   int    `json:"action"`
	Kind     string `json:"kind"`
	Deadline string `json:"deadline,omitempty"`
}

type ProxyV2 struct {
	Ref           string   `json:"ref"`
	Cents         int64    `json:"cents"`
	Studies       int      `json:"studies"`
	LookbackDays  int64    `json:"lookbackDays"`
	AvailableFrom string   `json:"availableFrom"`
	AvailableTo   string   `json:"availableTo"`
	Incomplete    bool     `json:"incomplete"`
	Windows       []string `json:"windows"`
	Scope         string   `json:"scope"`
}

type ViewAltV2 struct {
	ViewAlt
	Commitments []CommitmentV2 `json:"commitments"`
	Capacity    CapacityV2     `json:"capacity"`
	Reversals   []ReversalV2   `json:"reversals"`
	Proxies     []ProxyV2      `json:"proxies"`
}

type ViewV2 struct {
	Schema       string      `json:"schema"`
	Time         string      `json:"time"`
	Refs         []Ref       `json:"refs"`
	Operations   int         `json:"operations"`
	Alternatives []ViewAltV2 `json:"alternatives"`
}

// BuildViewV2 first verifies and maps the historical comparison grammar,
// then maps additional read-only observation and preview fields. No candidate
// is added or changed. The session and simulator are never modified here.
func BuildViewV2(s *session.Session, a *Agent, pos []int, ops int) (*ViewV2, *ViewMap, error) {
	before := s.Digests()
	base, m, err := BuildView(s, a, pos, ops)
	if err != nil {
		return nil, nil, err
	}
	o := s.Observe(0)
	v := &ViewV2{Schema: ViewV2Schema, Time: base.Time, Refs: base.Refs, Operations: ops, Alternatives: []ViewAltV2{}}
	prop := a.Alternatives[0].Actions
	pp := s.PreviewStaffing(prop...)
	guarantees := map[string]int64{}
	// These are the engine's exact full-week figures; observation.severance/2
	// is not used as an exact guarantee because benefits round differently.
	for _, e := range o.Staff {
		who := e.Name + " (" + e.ID + ")"
		hiredWho := ""
		if e.Hired != nil {
			hiredWho = e.ID + " (" + offerLabelOf(e) + ")"
		}
		for _, ln := range append(slices.Clone(pp.Guaranteed), pp.Forecast...) {
			if ln.Label == "Contract cost no longer owed · "+who || ln.Label == "Contract guarantee per full week · "+who || (hiredWho != "" && (ln.Label == "Contract cost no longer owed · "+hiredWho || ln.Label == "Contract guarantee per full week · "+hiredWho)) {
				guarantees[e.ID] = ln.Cents
			}
		}
	}
	for i, alt := range a.Alternatives {
		x := ViewAltV2{ViewAlt: base.Alternatives[i], Commitments: []CommitmentV2{}, Reversals: []ReversalV2{}, Proxies: []ProxyV2{}}
		// Replace the legacy obligation and capacity fragments, so a commitment
		// is shown once even when several actions touch the same employee.
		from, to := shiftDateV2(o.Date, 1), shiftDateV2(o.Date, o.Practice.BookingHorizonDays)
		x.Capacity = CapacityV2{Closed: QuantityV2{Unit: "slots", From: from, To: to}, Opened: QuantityV2{Unit: "slots", From: from, To: to}, Displaced: QuantityV2{Unit: "bookings", From: from, To: to}, Replacements: QuantityV2{Unit: "bookings", From: from, To: to}, ClosedByModality: []api.ModalityCount{}}
		keep := func(it ViewItem) bool { return activeItemV2(it.Kind) }
		x.Facts = slices.DeleteFunc(slices.Clone(x.Facts), func(it ViewItem) bool { return !keep(it) })
		x.Estimates = slices.DeleteFunc(slices.Clone(x.Estimates), func(it ViewItem) bool { return !keep(it) })
		x.Obligations = slices.DeleteFunc(slices.Clone(x.Obligations), func(it ViewItem) bool { return !keep(it) })
		x.Unavailable = slices.DeleteFunc(slices.Clone(x.Unavailable), func(it ViewItem) bool {
			return strings.HasPrefix(it.Kind, "weekly_guarantee") || it.Kind == "unmodeled"
		})
		if alt.Validation.OK {
			p := s.PreviewStaffing(alt.Actions...)
			for j, e := range o.Staff {
				if e.Employment == api.EmploymentTerminated {
					continue
				}
				ref := "S" + fmt.Sprint(j+1)
				for r, id := range m.Refs {
					if strings.HasPrefix(r, "E") && id == e.ID {
						ref = r
						break
					}
				}
				c := CommitmentV2{Ref: ref, PublicPosition: j + 1}
				if g, ok := guarantees[e.ID]; ok {
					c.BaselineGuarantee = num(g)
					c.ContinuingGuarantee = num(g)
					c.GuaranteeDelta = num(0)
				}
				// Hourly conditional severance is explicitly exposed. Salaried terms
				// have no hourly guarantee or terminable severance in this engine.
				if e.HourlyCents > 0 && e.Leaving == nil {
					c.BaselineExposure = num(e.SeveranceCents)
					c.ConditionalExposure = num(e.SeveranceCents)
					c.ExposureDelta = num(0)
				}
				if e.Leaving != nil {
					c.Through = e.Leaving.LastDay
				}
				for _, t := range p.Terminations {
					if t.Employee == e.ID {
						c.Terminating = true
						c.Through = t.LastDay
						if c.BaselineExposure != nil {
							c.ConditionalExposure = num(0)
							c.ExposureDelta = num(-*c.BaselineExposure)
						}
					}
				}
				x.Commitments = append(x.Commitments, c)
			}
			x.Capacity = capacityV2(s, o, alt.Actions, p)
			for _, mc := range x.Capacity.ClosedByModality {
				if !slices.Contains(viewModalities, mc.Modality) || mc.Count < 0 {
					return nil, nil, vf("capacity modality or count")
				}
			}
			used := map[int]bool{}
			for _, act := range alt.Actions {
				k := -1
				for j, pr := range prop {
					if !used[j] && pr.Type == act.Type && pr.Employee == act.Employee && pr.RequestID == act.RequestID && pr.Contract == act.Contract {
						k = j
						used[j] = true
						break
					}
				}
				if k < 0 {
					return nil, nil, vf("successor action reference")
				}
				r := ReversalV2{Action: pos[k]}
				switch act.Type {
				case api.ActTerminate:
					r.Kind = "termination"
				case api.ActHire:
					r.Kind = "hire"
				case api.ActSetRoster:
					r.Kind = "roster"
					if act.Op == api.RosterOpRemove {
						r.Kind = "roster_remove"
					}
					r.Deadline = shiftDateV2(act.Effective, -sim.NoticeDays)
				case api.ActSetReadingAllocation:
					r.Kind = "allocation"
					for _, ap := range p.Allocations {
						if ap.Employee == act.Employee && ap.Effective == act.Effective {
							r.Deadline = ap.WithdrawBy
						}
					}
				case api.ActContractTeleradiology:
					r.Kind = "tele_start"
					if act.Op == api.TeleOpCancel {
						r.Kind = "tele_cancel"
					}
				}
				x.Reversals = append(x.Reversals, r)
			}
			proxyRefs := []string{}
			for _, it := range base.Alternatives[i].Estimates {
				if strings.HasPrefix(it.Kind, "remote_read_fees") {
					proxyRefs = append(proxyRefs, it.Ref)
				}
			}
			for ti, tp := range p.Teleradiology {
				pr := ProxyV2{Cents: tp.TrailingFeeCents, Studies: tp.TrailingStudies, LookbackDays: tp.TrailingDays, AvailableTo: o.Time,
					Windows: tp.Windows, Scope: tp.Scope}
				for _, wn := range pr.Windows {
					if !slices.Contains(viewWindows, wn) {
						return nil, nil, vf("proxy window")
					}
				}
				if !slices.Contains(viewScopes, pr.Scope) {
					return nil, nil, vf("proxy scope")
				}
				now, er := time.Parse("2006-01-02T15:04:05", o.Time)
				if er != nil {
					return nil, nil, er
				}
				start, er := time.Parse("2006-01-02", shiftDateV2(o.Date, o.Practice.StartOffset))
				if er != nil {
					return nil, nil, er
				}
				cutoff := now.AddDate(0, 0, -int(tp.TrailingDays))
				pr.Incomplete = start.After(cutoff)
				if start.Before(cutoff) {
					start = cutoff
				}
				pr.AvailableFrom = start.Format("2006-01-02T15:04:05")
				// Preview entries and their fee lines have the same engine order.
				if ti >= len(proxyRefs) {
					return nil, nil, vf("proxy reference")
				}
				pr.Ref = proxyRefs[ti]
				x.Proxies = append(x.Proxies, pr)
			}
		}
		v.Alternatives = append(v.Alternatives, x)
	}
	if err := digestsUnchanged(s, before); err != nil {
		return nil, nil, err
	}
	return v, m, nil
}

func shiftDateV2(date string, days int64) string {
	d, err := time.Parse("2006-01-02", date)
	if err != nil {
		return ""
	}
	return d.AddDate(0, 0, int(days)).Format("2006-01-02")
}

func capacityV2(s *session.Session, o api.Observation, acts []api.Action, p api.StaffingPreview) CapacityV2 {
	from, to := shiftDateV2(o.Date, 1), shiftDateV2(o.Date, o.Practice.BookingHorizonDays)
	q := func(n *int64, unit string) QuantityV2 { return QuantityV2{n, unit, from, to} }
	c := CapacityV2{Closed: q(nil, "slots"), Opened: q(nil, "slots"), Displaced: q(num(int64(len(p.Displacements))), "bookings"), Replacements: q(num(0), "bookings"), ClosedByModality: []api.ModalityCount{}}
	n := int64(0)
	for _, d := range p.Displacements {
		if d.ProvisionalDate != "" {
			n++
		}
	}
	c.Replacements.Number = num(n)
	if len(acts) == 0 {
		c.Closed.Number = num(0)
		c.Opened.Number = num(0)
		return c
	}
	termsOnly := !slices.ContainsFunc(acts, func(a api.Action) bool { return a.Type != api.ActTerminate })
	switch {
	case termsOnly:
		r := s.PreviewSlotsClosed(acts...)
		if r.Available {
			c.Closed.Number = num(int64(r.Total))
			if r.Days > 0 {
				c.Closed.From, c.Closed.To = r.From, r.To
			}
			c.ClosedByModality = r.ByModality
		}
		c.Opened.Number = num(0)
	case len(p.Rosters) > 0:
		a, b := int64(0), int64(0)
		for _, m := range p.SlotsClosed {
			a += int64(m.Count)
		}
		for _, m := range p.SlotsOpened {
			b += int64(m.Count)
		}
		c.Closed.Number, c.Opened.Number = num(a), num(b)
		c.ClosedByModality = p.SlotsClosed
	case len(acts) == 1 && len(p.Hires) == 1:
		n := int64(0)
		for _, m := range p.Hires[0].OpensSlots {
			n += int64(m.Count)
		}
		c.Opened.Number = num(n)
		c.Opened.From = p.Hires[0].ProductiveDate
		if c.Opened.From > c.Opened.To {
			c.Opened.From = from
		}
	}
	return c
}

func quantityTextV2(q QuantityV2) string {
	s := "unavailable"
	if q.Number != nil {
		s = count(*q.Number) + " " + q.Unit
	}
	return s + " · " + q.From + " to " + q.To + " (current booking horizon)"
}
func centsTextV2(n *int64) string {
	if n == nil {
		return "unavailable"
	}
	return money(*n)
}

// ViewV2TableText pins the successor's additional fixed vocabulary together
// with the unchanged grammar it uses for the remaining legacy facts.
func activeItemV2(kind string) bool {
	return kind != "weekly_guarantee" && kind != "weekly_guarantee_roster" && kind != "severance_if_terminated_later" &&
		!strings.HasPrefix(kind, "remote_read_fees") && !strings.HasPrefix(kind, "slots_closed") && kind != "displaced_bookings" && kind != "displaced_replacement"
}

func ViewV2TableText() string {
	var b strings.Builder
	b.WriteString(ViewV2Schema + "\n\nInherited typed grammar: each row is labelled by its rendering status.\n")
	for _, name := range viewKindNames {
		k := viewKinds[name]
		status := "ACTIVE"
		if !activeItemV2(name) {
			status = "LEGACY-INACTIVE (internal binding only)"
		}
		fmt.Fprintf(&b, "%s | %s | label %q | ref %q | unit %s | basis %q | basisNoWhen %q | forms", status, name, k.label, k.ref, k.unit, k.basis, k.basisNoWhen)
		for _, f := range k.when {
			fmt.Fprintf(&b, " %q", f.text)
		}
		fmt.Fprintf(&b, " | empty %v\n", k.empty)
	}
	b.WriteString("\nACTIVE successor detail, capacity, commitment and proxy templates:\n")
	b.Write(presentationV3Text)
	b.WriteString("\nACTIVE action-specific reversibility:\n")
	keys := make([]string, 0, len(reversalTextsV2))
	for key := range reversalTextsV2 {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		fmt.Fprintf(&b, "%s | %q\n", key, reversalTextsV2[key])
	}
	return b.String()
}
