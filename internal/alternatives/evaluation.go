package alternatives

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/georgezero/dreamteam-go/internal/session"
)

// Evaluation is the evaluation audience: the agent audience exactly as
// shown, then the class B projections of every alternative under one
// declared continuation, and paired differences. It is written after the
// decision and never shown to the agent at the decision point.
type Evaluation struct {
	Schema             string               `json:"schema"`
	Audience           string               `json:"audience"`
	Agent              Agent                `json:"agent"`
	Source             EvalSource           `json:"source"`
	Continuation       session.Continuation `json:"continuation"`
	ContinuationSha256 string               `json:"continuationSha256"`
	ContinuationNote   string               `json:"continuationNote"`
	HorizonSec         int64                `json:"horizonSec"`
	Horizon            string               `json:"horizon"`
	Branches           []EvalBranch         `json:"branches"`
	Differences        []Difference         `json:"differences"`
	Sha256             string               `json:"sha256"`
}

// EvalSource is the decision point in full (evaluation only).
type EvalSource struct {
	Seed           int64           `json:"seed"`
	RecordChecksum string          `json:"recordChecksum"`
	RecordedBy     string          `json:"recordedBy"`
	Compatibility  string          `json:"compatibility"`
	Digests        session.Digests `json:"digests"`
}

// EvalBranch is one alternative's branch.
type EvalBranch struct {
	ID          string                 `json:"id"`
	Identity    session.BranchIdentity `json:"identity"`
	Verified    bool                   `json:"replayVerified"`
	AtDecision  session.Digests        `json:"atDecision"`
	AtHorizon   session.Digests        `json:"atHorizon"`
	Projected   []Value                `json:"projected"`
	Unavailable []Unavailable          `json:"unavailable"`
}

// Difference is one projected metric across branches: each branch's value
// and its difference from the reference branch, under the continuation and
// horizon named in Label. It is not a gain, a loss or a verdict.
type Difference struct {
	Key            string           `json:"key"`
	Unit           string           `json:"unit"`
	Reference      string           `json:"reference"`
	Values         map[string]int64 `json:"values"`
	MinusReference map[string]int64 `json:"minusReference"`
	Label          string           `json:"label"`
}

const continuationNote = "Every projection is exact for this seed, this engine and this continuation only: no management action of any kind " +
	"after the decision instant. It uses the simulator's realized future, which no manager knows at the decision point. It is not a " +
	"prediction, not a counterfactual truth outside the simulator, and not evidence of a good or bad decision."

// BuildEvaluation branches every alternative of the agent comparison from
// the decision-point record, verifies the set, and reads the projections.
// The record must be the decision point the comparison was built at.
func BuildEvaluation(r *session.Record, a *Agent, opts session.BranchOptions) (*Evaluation, error) {
	if err := a.check(); err != nil {
		return nil, err
	}
	if r.Until != a.DecisionPoint.Time || r.Scenario.Fingerprint != a.DecisionPoint.Scenario || r.Digests.State != a.DecisionPoint.StateDigest {
		return nil, fmt.Errorf("%w: the record is not the comparison's decision point", ErrContract)
	}
	var alts []session.Alternative
	for _, x := range a.Alternatives {
		alts = append(alts, session.Alternative{ID: x.ID, Actions: x.Actions})
	}
	set, err := session.Branch(r, alts, opts)
	if err != nil {
		return nil, err
	}
	if err := session.VerifyBranchSet(r, set); err != nil {
		return nil, err
	}
	e := &Evaluation{Schema: ContractSchema, Audience: AudienceEvaluation, Agent: *a,
		Source:       EvalSource{Seed: r.Seed, RecordChecksum: r.Checksum, RecordedBy: r.Engine, Compatibility: set.Source.Compatibility, Digests: set.Source.Digests},
		Continuation: set.Continuation, ContinuationSha256: set.Continuation.Sha256(), ContinuationNote: continuationNote,
		HorizonSec: set.HorizonSec, Horizon: set.Horizon, Differences: []Difference{}}
	for _, b := range set.Branches {
		eb := EvalBranch{ID: b.ID, Identity: b.Identity, Verified: b.Verified, AtDecision: b.AtDecision, AtHorizon: b.AtHorizon,
			Projected: []Value{}, Unavailable: []Unavailable{}}
		project(&eb, b.Start, b.End, set.HorizonSec)
		e.Branches = append(e.Branches, eb)
	}
	e.Differences = differences(e)
	e.Sha256 = e.digest()
	if err := e.check(); err != nil {
		return nil, err
	}
	return e, nil
}

// project reads the class B metrics of one branch: flows over the window
// from the decision instant (horizon minus decision, the window metrics
// being cumulative from the takeover), waits over the takeover-to-horizon
// window (percentiles do not subtract), and balances at the horizon.
func project(b *EvalBranch, start, end *session.Result, horizon int64) {
	p := func(key, label, unit, source string, n int64) {
		b.Projected = append(b.Projected, Value{Key: key, Label: label, Class: ClassProjected, Unit: unit, Number: num(n), Source: source})
	}
	if end.Window.Complete && end.Window.To.Sec < horizon {
		b.Unavailable = append(b.Unavailable, Unavailable{"window", ClassUnavailable, "the scenario's evaluation window ended before the horizon"})
	} else {
		w0, w1 := start.Window, end.Window
		d := func(key, label, unit, source string, x0, x1 int64) {
			p(key, label+" (decision to horizon)", unit, "end."+source+" - start."+source, x1-x0)
		}
		d("orders_received", "Orders received", UnitCount, "window.ops.demand.received", int64(w0.Ops.Demand.Received), int64(w1.Ops.Demand.Received))
		d("orders_booked", "Orders booked", UnitCount, "window.ops.demand.booked", int64(w0.Ops.Demand.Booked), int64(w1.Ops.Demand.Booked))
		d("orders_lost", "Orders lost", UnitCount, "window.ops.demand.lost", int64(w0.Ops.Demand.Lost), int64(w1.Ops.Demand.Lost))
		d("appointments_cancelled", "Appointments cancelled by patients", UnitCount, "window.ops.demand.cancelled", int64(w0.Ops.Demand.Cancelled), int64(w1.Ops.Demand.Cancelled))
		d("walkins_arrived", "Walk-ins arrived", UnitCount, "window.ops.walkIns.arrived", int64(w0.Ops.WalkIns.Arrived), int64(w1.Ops.WalkIns.Arrived))
		d("exams", "Exams performed", UnitCount, "window.ops.exams", int64(w0.Ops.Exams), int64(w1.Ops.Exams))
		d("reports_final", "Reports finalized", UnitCount, "window.ops.reporting.final", int64(w0.Ops.Reporting.Final), int64(w1.Ops.Reporting.Final))
		d("net_revenue", "Net revenue (window convention)", UnitCents, "window.money.netRevenueCents", w0.Money.NetRevenueCents, w1.Money.NetRevenueCents)
		d("wages", "Wages", UnitCents, "window.money.ledger.wagesCents", w0.Money.Ledger.WagesCents, w1.Money.Ledger.WagesCents)
		d("benefits", "Benefits", UnitCents, "window.money.ledger.benefitsCents", w0.Money.Ledger.BenefitsCents, w1.Money.Ledger.BenefitsCents)
		d("severance", "Severance recognized", UnitCents, "window.money.ledger.severanceCents", w0.Money.Ledger.SeveranceCents, w1.Money.Ledger.SeveranceCents)
		d("operating_expense", "Operating expense (window convention)", UnitCents, "window.money.operatingExpenseCents", w0.Money.OperatingExpenseCents, w1.Money.OperatingExpenseCents)
		d("operating_profit", "Operating profit (window convention)", UnitCents, "window.money.operatingProfitCents", w0.Money.OperatingProfitCents, w1.Money.OperatingProfitCents)
		p("wait_to_exam_p50", "Wait to exam, 50th percentile (takeover to horizon)", UnitSec, "end.window.ops.waits.toExam.p50Sec", end.Window.Ops.Waits.ToExam.P50Sec)
		p("wait_to_exam_p90", "Wait to exam, 90th percentile (takeover to horizon)", UnitSec, "end.window.ops.waits.toExam.p90Sec", end.Window.Ops.Waits.ToExam.P90Sec)
	}
	bal, ob := end.Balances, end.Obligations
	p("cash_at_horizon", "Cash at the horizon", UnitCents, "end.balances.cashCents", bal.CashCents)
	p("lowest_cash", "Lowest cash since day 1", UnitCents, "end.balances.minCashCents", bal.MinCashCents)
	p("days_below_zero", "Days below zero cash since day 1", UnitDays, "end.balances.daysBelowZero", bal.DaysBelowZero)
	p("payables_at_horizon", "Accounts payable at the horizon", UnitCents, "end.balances.payablesCents", bal.PayablesCents)
	p("severance_payable_at_horizon", "Severance payable at the horizon", UnitCents, "end.balances.severancePayableCents", bal.SeverancePayableCents)
	p("booked_ahead_at_horizon", "Appointments booked ahead at the horizon", UnitCount, "end.obligations.bookedAhead", int64(ob.BookedAhead))
	p("unread_at_horizon", "Studies unread at the horizon", UnitCount, "end.obligations.unread", int64(ob.Unread))
}

// differences pairs every projected number with the reference branch's:
// no_action when listed and accepted, else the proposal.
func differences(e *Evaluation) []Difference {
	out := []Difference{}
	ref := IDProposal
	if slices.ContainsFunc(e.Branches, func(b EvalBranch) bool { return b.ID == IDNoAction }) {
		ref = IDNoAction
	}
	ri := slices.IndexFunc(e.Branches, func(b EvalBranch) bool { return b.ID == ref })
	label := "difference from " + ref + " under " + e.Continuation.ID + " to " + e.Horizon + "; not a gain, a loss or a verdict"
	for _, v := range e.Branches[ri].Projected {
		d := Difference{Key: v.Key, Unit: v.Unit, Reference: ref, Values: map[string]int64{}, MinusReference: map[string]int64{}, Label: label}
		for _, b := range e.Branches {
			i := slices.IndexFunc(b.Projected, func(x Value) bool { return x.Key == v.Key })
			if i < 0 {
				d.Values = nil
				break
			}
			d.Values[b.ID] = *b.Projected[i].Number
			d.MinusReference[b.ID] = *b.Projected[i].Number - *v.Number
		}
		if d.Values != nil {
			out = append(out, d)
		}
	}
	return out
}

func (e *Evaluation) digest() string {
	c := *e
	c.Sha256 = ""
	return sha(&c)
}

// Encode is the evaluation's canonical bytes.
func (e *Evaluation) Encode() ([]byte, error) {
	raw, err := json.MarshalIndent(e, "", " ")
	return append(raw, '\n'), err
}

func (e *Evaluation) check() error {
	switch {
	case e.Schema != ContractSchema || e.Audience != AudienceEvaluation:
		return fmt.Errorf("%w: schema %q audience %q", ErrContract, e.Schema, e.Audience)
	case e.Sha256 != e.digest():
		return fmt.Errorf("%w: digest", ErrContract)
	case e.ContinuationSha256 != e.Continuation.Sha256() || e.ContinuationNote != continuationNote || e.Continuation.ID != session.ContinuationNoFurtherAction:
		return fmt.Errorf("%w: continuation", ErrContract)
	case len(e.Branches) != len(e.Agent.Alternatives):
		return fmt.Errorf("%w: %d branches for %d alternatives", ErrContract, len(e.Branches), len(e.Agent.Alternatives))
	}
	if err := e.Agent.check(); err != nil {
		return err
	}
	for i, b := range e.Branches {
		alt := e.Agent.Alternatives[i]
		if b.ID != alt.ID || !jsonEqual(b.Identity.Actions, canon(alt.Actions)) || !b.Verified || b.Identity.HorizonSec != e.HorizonSec ||
			b.Identity.Continuation != e.Continuation || b.Identity.SourceChecksum != e.Source.RecordChecksum {
			return fmt.Errorf("%w: branch %s does not match its alternative, horizon, continuation or source", ErrContract, b.ID)
		}
		for _, v := range b.Projected {
			if err := checkValue(v, ClassProjected); err != nil {
				return fmt.Errorf("%s: %w", b.ID, err)
			}
		}
		for _, u := range b.Unavailable {
			if u.Class != ClassUnavailable || u.Reason == "" {
				return fmt.Errorf("%w: %s: unavailable item %q", ErrContract, b.ID, u.Key)
			}
		}
	}
	if !jsonEqual(differences(e), e.Differences) {
		return fmt.Errorf("%w: the differences do not follow from the branches", ErrContract)
	}
	return nil
}

// DecodeEvaluation reads an evaluation-audience comparison and checks
// every rule.
func DecodeEvaluation(raw []byte) (*Evaluation, error) {
	var e Evaluation
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&e); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrContract, err)
	}
	if dec.More() {
		return nil, fmt.Errorf("%w: trailing data", ErrContract)
	}
	if err := e.check(); err != nil {
		return nil, err
	}
	return &e, nil
}

// VerifyEvaluation rebuilds the evaluation from the decision-point record
// and requires it equal, byte for byte.
func VerifyEvaluation(r *session.Record, e *Evaluation) error {
	if err := e.check(); err != nil {
		return err
	}
	again, err := BuildEvaluation(r, &e.Agent, session.BranchOptions{Continuation: e.Continuation})
	if err != nil {
		return err
	}
	x, _ := again.Encode()
	y, _ := e.Encode()
	if !bytes.Equal(x, y) {
		return fmt.Errorf("%w: the evaluation does not rebuild", ErrContract)
	}
	return nil
}
