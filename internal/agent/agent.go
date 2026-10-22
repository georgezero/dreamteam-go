// Package agent is DreamTeam's agent protocol (V0.3a): a deterministic
// JSON request/response contract over one session, independent of its
// transport. Handle takes one request and returns one response, as
// bytes; stdio.go carries them one per line over standard input and
// output, and another transport (HTTP, WebSocket) would carry the same
// bytes without changing what they mean.
//
// The protocol adds no simulation behaviour. Every operation is one of the
// session API's (Observe, ValidateBatch, PreviewStaffing, Apply, Advance,
// AdvanceTo, StepVisible, EndOfDay, Save, Digests) or a read of the public
// action contract (the offer table, roster plans, reading allocations,
// teleradiology plans). It exposes nothing the session does not already
// show a player. docs/AGENT_PROTOCOL.md is the specification.
package agent

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/session"
	"github.com/georgezero/dreamteam-go/internal/simtime"
	"github.com/georgezero/dreamteam-go/internal/version"
)

// Protocol is the protocol's name and version, in every response.
const Protocol = "dreamteam.agent/v1"

// Error codes (stable; docs/AGENT_PROTOCOL.md §5). An action's own refusal
// is not a protocol error: it is a result, with the simulator's code.
const (
	ErrInvalidJSON    = "invalid_json"    // the request is not one JSON object
	ErrInvalidRequest = "invalid_request" // unknown fields, a missing op, an id that is not a string or number
	ErrUnknownOp      = "unknown_op"
	ErrInvalidArgs    = "invalid_args" // the op's arguments are missing, unknown or out of range
	ErrDuplicate      = "duplicate_target"
	ErrNotPreviewable = "not_previewable" // preview covers the V0.2 staffing actions only
	ErrSaveFailed     = "save_failed"
	ErrUnsavable      = "unsavable"
	ErrDiverged       = "apply_diverged" // never expected: a validated batch was refused on apply
)

// Request is one request: an id the response echoes (a string or a
// number), the operation and its arguments.
type Request struct {
	ID   json.RawMessage `json:"id,omitempty"`
	Op   string          `json:"op"`
	Args json.RawMessage `json:"args,omitempty"`
}

// Response is one response: the protocol, the request's id (null when the
// request had none or could not be read), whether it succeeded, and its
// result or error.
type Response struct {
	Protocol string          `json:"protocol"`
	ID       json.RawMessage `json:"id"`
	OK       bool            `json:"ok"`
	Result   any             `json:"result,omitempty"`
	Error    *Error          `json:"error,omitempty"`
}

// Error is a protocol error: a stable code, a message, and where it is
// (an argument field, or an action's index in a batch).
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
	Index   *int   `json:"index,omitempty"`
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

func argErr(field, f string, a ...any) *Error {
	return &Error{Code: ErrInvalidArgs, Message: fmt.Sprintf(f, a...), Field: field}
}

// Ops lists the operations, in the order of the specification.
var Ops = []string{"hello", "observe", "actions", "validate", "preview", "apply", "advance", "save", "digests"}

// Handler serves one session. Requests are handled one at a time, in
// order; a Handler is not safe for concurrent use.
type Handler struct {
	s *session.Session
	// Ref is the scenario reference the session was opened with (a
	// bundled ID or a file), reported by hello.
	Ref string
}

// New returns a handler for s.
func New(s *session.Session) *Handler { return &Handler{s: s} }

// Handle answers one request with one response. It never fails: a request
// that cannot be read, or an operation that fails, gets an error response,
// and a failed operation changes nothing.
func (h *Handler) Handle(line []byte) []byte {
	resp := Response{Protocol: Protocol, ID: json.RawMessage("null")}
	var req Request
	dec := json.NewDecoder(bytes.NewReader(line))
	dec.DisallowUnknownFields()
	switch {
	case !json.Valid(line): // also: anything after the request's value
		resp.Error = &Error{Code: ErrInvalidJSON, Message: "the request is not one JSON value"}
	default:
		if err := dec.Decode(&req); err != nil {
			resp.Error = &Error{Code: ErrInvalidRequest, Message: err.Error()}
			// Echo the id of an object whose id can be read.
			var loose struct {
				ID json.RawMessage `json:"id"`
			}
			if json.Unmarshal(line, &loose) == nil {
				if id, ok := normalizeID(loose.ID); ok {
					resp.ID = id
				}
			}
			break
		}
		id, ok := normalizeID(req.ID)
		if !ok {
			resp.Error = &Error{Code: ErrInvalidRequest, Message: "id must be a string or a number", Field: "id"}
			break
		}
		resp.ID = id
		resp.Result, resp.Error = h.dispatch(req)
	}
	resp.OK = resp.Error == nil
	if !resp.OK {
		resp.Result = nil
	}
	out, err := json.Marshal(resp)
	if err != nil { // a result that cannot be encoded (never expected)
		out, _ = json.Marshal(Response{Protocol: Protocol, ID: resp.ID, Error: &Error{Code: "internal_error", Message: err.Error()}})
	}
	return out
}

// normalizeID accepts an absent id, a string or a number, compacted.
func normalizeID(raw json.RawMessage) (json.RawMessage, bool) {
	if len(raw) == 0 {
		return json.RawMessage("null"), true
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, false
	}
	switch v.(type) {
	case string, float64, nil:
		var b bytes.Buffer
		json.Compact(&b, raw)
		return b.Bytes(), true
	}
	return nil, false
}

// args decodes an op's arguments strictly (absent: an empty object).
func args(raw json.RawMessage, v any) *Error {
	if len(raw) == 0 || string(raw) == "null" {
		raw = []byte("{}")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return &Error{Code: ErrInvalidArgs, Message: err.Error()}
	}
	return nil
}

func (h *Handler) dispatch(req Request) (any, *Error) {
	switch req.Op {
	case "":
		return nil, &Error{Code: ErrInvalidRequest, Message: "op is required", Field: "op"}
	case "hello":
		if e := args(req.Args, &struct{}{}); e != nil {
			return nil, e
		}
		return h.hello(), nil
	case "observe":
		return h.observe(req.Args)
	case "actions":
		if e := args(req.Args, &struct{}{}); e != nil {
			return nil, e
		}
		return catalog(h.s), nil
	case "validate":
		return h.validate(req.Args)
	case "preview":
		return h.preview(req.Args)
	case "apply":
		return h.apply(req.Args)
	case "advance":
		return h.advance(req.Args)
	case "save":
		return h.save(req.Args)
	case "digests":
		if e := args(req.Args, &struct{}{}); e != nil {
			return nil, e
		}
		return h.clock(), nil
	}
	return nil, &Error{Code: ErrUnknownOp, Message: fmt.Sprintf("unknown op %q; the ops are %v", req.Op, Ops), Field: "op"}
}

// Clock is where the session is: its time, day, the log count and the
// five digests.
type Clock struct {
	Time     string          `json:"time"`
	Day      int64           `json:"day"`
	LogTotal uint64          `json:"logTotal"`
	Digests  session.Digests `json:"digests"`
}

func (h *Handler) clock() Clock {
	o := h.s.Observe(0)
	return Clock{Time: o.Time, Day: o.Day, LogTotal: o.LogTotal, Digests: h.s.Digests()}
}

// Hello identifies the protocol, the engine and the session.
type Hello struct {
	Protocol string `json:"protocol"`
	Engine   string `json:"engine"`
	Scenario struct {
		Ref         string `json:"ref,omitempty"`
		ID          string `json:"id"`
		Name        string `json:"name"`
		Fingerprint string `json:"fingerprint"`
	} `json:"scenario"`
	Seed  int64    `json:"seed"`
	Ops   []string `json:"ops"`
	Clock Clock    `json:"clock"`
}

func (h *Hello) setScenario(s *session.Session) {
	h.Scenario.ID, h.Scenario.Name, h.Scenario.Fingerprint = s.ScenarioID(), s.Observe(0).Practice.Name, s.Fingerprint()
}

func (h *Handler) hello() Hello {
	r := Hello{Protocol: Protocol, Engine: version.Number, Seed: h.s.Seed(), Ops: Ops, Clock: h.clock()}
	r.setScenario(h.s)
	r.Scenario.Ref = h.Ref
	return r
}

// observe: the session's observation, with up to recentLog log entries
// (default 20, at most 2,000).
func (h *Handler) observe(raw json.RawMessage) (any, *Error) {
	a := struct {
		RecentLog *int `json:"recentLog"`
	}{}
	if e := args(raw, &a); e != nil {
		return nil, e
	}
	n := 20
	if a.RecentLog != nil {
		if *a.RecentLog < 0 || *a.RecentLog > 2000 {
			return nil, argErr("recentLog", "recentLog must be 0–2000, not %d", *a.RecentLog)
		}
		n = *a.RecentLog
	}
	return struct {
		Observation api.Observation `json:"observation"`
	}{h.s.Observe(n)}, nil
}

// batch reads a batch: one to 64 actions, decoded strictly.
func batch(raw json.RawMessage) ([]api.Action, *Error) {
	a := struct {
		Actions []json.RawMessage `json:"actions"`
	}{}
	if e := args(raw, &a); e != nil {
		return nil, e
	}
	if len(a.Actions) == 0 || len(a.Actions) > 64 {
		return nil, argErr("actions", "a batch has 1–64 actions, not %d", len(a.Actions))
	}
	out := make([]api.Action, len(a.Actions))
	for i, r := range a.Actions {
		dec := json.NewDecoder(bytes.NewReader(r))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&out[i]); err != nil {
			return nil, &Error{Code: ErrInvalidArgs, Message: err.Error(), Field: "actions", Index: &i}
		}
	}
	return out, nil
}

// Check is a batch's validation: whether every action is valid now, and
// each action's result (code and message from the simulator).
type Check struct {
	Valid   bool               `json:"valid"`
	Results []api.ActionResult `json:"results"`
}

// check validates a batch without changing anything. A batch of V0.1
// actions is checked action by action, as the session checks it; two
// shift extensions for the same employee and day would be checked apart
// but applied in turn, so the protocol refuses that batch (split it).
func (h *Handler) check(actions []api.Action) (Check, *Error) {
	seen := map[[2]string]bool{}
	today := h.s.Calendar().Date(h.s.Now()).String()
	for i, a := range actions {
		if a.Type != api.ActExtendShift {
			continue
		}
		d := a.Date
		if d == "" {
			d = today
		}
		k := [2]string{a.Employee, d}
		if seen[k] {
			i := i
			return Check{}, &Error{Code: ErrDuplicate, Message: fmt.Sprintf("two shift extensions for %s on %s in one batch: send them in separate batches", a.Employee, d), Field: "actions", Index: &i}
		}
		seen[k] = true
	}
	res := h.s.ValidateBatch(actions...)
	return Check{Valid: !slices.ContainsFunc(res, func(r api.ActionResult) bool { return !r.OK }), Results: res}, nil
}

func (h *Handler) validate(raw json.RawMessage) (any, *Error) {
	actions, e := batch(raw)
	if e != nil {
		return nil, e
	}
	c, e := h.check(actions)
	if e != nil {
		return nil, e
	}
	return c, nil
}

// preview: the session's preview of a batch of V0.2 staffing actions.
func (h *Handler) preview(raw json.RawMessage) (any, *Error) {
	actions, e := batch(raw)
	if e != nil {
		return nil, e
	}
	for i, a := range actions {
		if !slices.Contains(staffingTypes, a.Type) {
			return nil, &Error{Code: ErrNotPreviewable, Message: fmt.Sprintf("%q has no preview: preview covers %v (validate checks any action)", a.Type, staffingTypes), Field: "actions", Index: &i}
		}
	}
	return struct {
		Preview api.StaffingPreview `json:"preview"`
	}{h.s.PreviewStaffing(actions...)}, nil
}

// staffingTypes are the V0.2 staffing actions, which have previews.
var staffingTypes = []string{api.ActHire, api.ActTerminate, api.ActSetRoster, api.ActSetReadingAllocation, api.ActContractTeleradiology}

// Applied is an apply's outcome: whether the batch was applied (only a
// fully valid batch is), each action's result, and the clock after.
type Applied struct {
	Applied bool               `json:"applied"`
	Results []api.ActionResult `json:"results"`
	Clock   Clock              `json:"clock"`
}

// apply validates the batch and applies it only if every action is valid:
// a refused batch changes nothing, not even the action record.
func (h *Handler) apply(raw json.RawMessage) (any, *Error) {
	actions, e := batch(raw)
	if e != nil {
		return nil, e
	}
	c, e := h.check(actions)
	if e != nil {
		return nil, e
	}
	if !c.Valid {
		return Applied{Results: c.Results, Clock: h.clock()}, nil
	}
	res := h.s.Apply(actions...)
	if slices.ContainsFunc(res, func(r api.ActionResult) bool { return !r.OK }) {
		return nil, &Error{Code: ErrDiverged, Message: "a validated batch was refused on apply"}
	}
	return Applied{Applied: true, Results: res, Clock: h.clock()}, nil
}

// Advanced is an advance's outcome: why it stopped and the clock after.
type Advanced struct {
	Reason string `json:"reason"`
	Clock  Clock  `json:"clock"`
}

// maxAdvance bounds one advance.
const maxAdvance = 366 * simtime.Day

// advance moves time by the session's own semantics, exactly one of:
// minutes (Advance, optionally stopping on an alert), to (AdvanceTo, a
// later practice-local time), or next ("visible": StepVisible; "end_of_day":
// EndOfDay).
func (h *Handler) advance(raw json.RawMessage) (any, *Error) {
	a := struct {
		Minutes     *int64  `json:"minutes"`
		To          *string `json:"to"`
		Next        *string `json:"next"`
		StopOnAlert bool    `json:"stopOnAlert"`
	}{}
	if e := args(raw, &a); e != nil {
		return nil, e
	}
	n := 0
	for _, set := range []bool{a.Minutes != nil, a.To != nil, a.Next != nil} {
		if set {
			n++
		}
	}
	if n != 1 {
		return nil, argErr("", "give exactly one of minutes, to or next")
	}
	if a.StopOnAlert && a.Minutes == nil {
		return nil, argErr("stopOnAlert", "stopOnAlert applies to minutes only")
	}
	now := h.s.Now()
	var reason string
	switch {
	case a.Minutes != nil:
		d := simtime.Duration(*a.Minutes) * simtime.Minute
		if *a.Minutes <= 0 || d > maxAdvance {
			return nil, argErr("minutes", "minutes must be 1–%d, not %d", int64(maxAdvance/simtime.Minute), *a.Minutes)
		}
		reason = reasonName(h.s.Advance(d, session.Options{StopOnAlert: a.StopOnAlert}))
	case a.To != nil:
		t, err := h.s.Calendar().ParseISO(*a.To)
		switch {
		case err != nil:
			return nil, argErr("to", "to must be a practice-local time YYYY-MM-DDTHH:MM:SS: %v", err)
		case t <= now:
			return nil, argErr("to", "to %s is not after now (%s)", *a.To, h.s.Calendar().ISO(now))
		case t > now.Add(maxAdvance):
			return nil, argErr("to", "to %s is more than 366 days ahead", *a.To)
		}
		h.s.AdvanceTo(t)
		reason = "reached"
	case *a.Next == "visible":
		reason = reasonName(h.s.StepVisible())
	case *a.Next == "end_of_day":
		reason = reasonName(h.s.EndOfDay())
	default:
		return nil, argErr("next", "next must be \"visible\" or \"end_of_day\", not %q", *a.Next)
	}
	return Advanced{Reason: reason, Clock: h.clock()}, nil
}

func reasonName(r session.Reason) string {
	switch r {
	case session.Reached:
		return "reached"
	case session.Budget:
		return "budget"
	case session.Alert:
		return "alert"
	case session.Visible:
		return "visible"
	case session.Horizon:
		return "horizon"
	}
	return "unknown"
}

// Saved is a save's outcome: the file and the clock it records.
type Saved struct {
	Path    string `json:"path"`
	Batches int    `json:"batches"`
	Clock   Clock  `json:"clock"`
}

// save writes the session to path (the session's own save: scenario,
// seed and action batches with the five digests).
func (h *Handler) save(raw json.RawMessage) (any, *Error) {
	a := struct {
		Path string `json:"path"`
	}{}
	if e := args(raw, &a); e != nil {
		return nil, e
	}
	if a.Path == "" {
		return nil, argErr("path", "path is required")
	}
	if err := h.s.Save(a.Path); err != nil {
		if errors.Is(err, session.ErrUnsavable) {
			return nil, &Error{Code: ErrUnsavable, Message: err.Error()}
		}
		return nil, &Error{Code: ErrSaveFailed, Message: err.Error(), Field: "path"}
	}
	r, err := h.s.Record()
	if err != nil {
		return nil, &Error{Code: ErrSaveFailed, Message: err.Error()}
	}
	return Saved{Path: a.Path, Batches: len(r.Batches), Clock: h.clock()}, nil
}
