package bench

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// A Metric is one figure of a run's canonical result (dreamteam.result/v1,
// session.Result): its name in the benchmark, its path in result.json and
// its kind. The value is the result's own, copied without conversion;
// docs/BENCHMARK.md §6 gives each definition's source. Kind "int" is an
// integer (cents, seconds or a count); "ratio" a {num, den} pair, kept as
// a pair. Absent is the value when the result omits the path (the
// result's own convention for nothing to report), or "" when it never
// does.
type Metric struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Group  string `json:"group"`
	Absent string `json:"absent,omitempty"`
}

// Metrics are the benchmark's figures, in report order. Money is the
// evaluation window's (the exact takeover to the end of the run; baseline
// adjusted, as the result defines it); operations are the window's
// operational metrics.
var Metrics = []Metric{
	{"netRevenueCents", "window.money.netRevenueCents", "int", "finance", ""},
	{"operatingExpenseCents", "window.money.operatingExpenseCents", "int", "finance", ""},
	{"operatingProfitCents", "window.money.operatingProfitCents", "int", "finance", ""},
	{"margin", "window.money.margin", "ratio", "finance", ""},
	{"operatingCashFlowCents", "window.money.operatingCashFlowCents", "int", "finance", ""},
	{"minCashCents", "window.money.minCashCents", "int", "finance", ""},
	{"shortfallCents", "window.money.shortfallCents", "int", "finance", ""},
	{"ordersReceived", "window.ops.demand.received", "int", "demand", ""},
	{"ordersLost", "window.ops.demand.lost", "int", "demand", ""},
	{"lostRate", "window.ops.demand.lostRate", "ratio", "demand", ""},
	{"examsCompleted", "window.ops.exams", "int", "demand", ""},
	{"reportsFinal", "window.ops.reporting.final", "int", "reporting", ""},
	{"tatP50Sec", "window.ops.reporting.tat.p50Sec", "int", "reporting", ""},
	{"tatP90Sec", "window.ops.reporting.tat.p90Sec", "int", "reporting", ""},
	{"tatOverTarget", "window.ops.reporting.overTarget", "int", "reporting", ""},
	{"tatBreaches", "window.ops.reporting.breaches", "int", "reporting", ""},
	{"backlogEnd", "window.ops.reporting.backlogEnd", "int", "reporting", ""},
	{"waitToExamP50Sec", "window.ops.waits.toExam.p50Sec", "int", "service", ""},
	{"waitToExamP90Sec", "window.ops.waits.toExam.p90Sec", "int", "service", ""},
	{"waitsOver30", "window.ops.waits.over30", "int", "service", ""},
	{"displacedAppointments", "window.ops.displacement.events", "int", "service", "none displaced"},
	{"overtimeSec", "window.ops.workforce.overtimeSec", "int", "staffing", ""},
	{"heldOver", "window.ops.workforce.heldOver", "int", "staffing", ""},
	{"skippedBreaks", "window.ops.workforce.skippedBreaks", "int", "staffing", ""},
	{"noCoverage", "window.ops.workforce.noCoverage", "int", "staffing", ""},
	{"alertsRaisedWarn", "alerts.raisedWarn", "int", "safety", ""},
	{"alertsRaisedAlert", "alerts.raisedAlert", "int", "safety", ""},
	{"alertsActiveAtEnd", "alerts.activeAtEnd", "int", "safety", ""},
	{"daysBelowZero", "balances.daysBelowZero", "int", "safety", ""},
	{"actionsAttempted", "actions.attempted", "int", "actions", ""},
	{"actionsAccepted", "actions.accepted", "int", "actions", ""},
	{"actionsRejected", "actions.rejected", "int", "actions", ""},
}

// Value is one metric's value in a run: the result's JSON value (null
// when the result omits it).
type Value struct {
	Name  string          `json:"name"`
	Value json.RawMessage `json:"value"`
}

// Extract reads every metric from a result's bytes.
func Extract(result []byte) ([]Value, error) {
	var root any
	dec := json.NewDecoder(bytes.NewReader(result))
	dec.UseNumber()
	if err := dec.Decode(&root); err != nil {
		return nil, err
	}
	out := make([]Value, len(Metrics))
	for i, m := range Metrics {
		v, ok := lookup(root, m.Path)
		if !ok {
			if m.Absent == "" {
				return nil, fmt.Errorf("the result has no %s", m.Path)
			}
			out[i] = Value{m.Name, json.RawMessage("null")}
			continue
		}
		raw, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		if err := checkKind(m, raw); err != nil {
			return nil, err
		}
		out[i] = Value{m.Name, raw}
	}
	return out, nil
}

func lookup(v any, path string) (any, bool) {
	for _, k := range strings.Split(path, ".") {
		o, ok := v.(map[string]any)
		if !ok {
			return nil, false
		}
		if v, ok = o[k]; !ok {
			return nil, false
		}
	}
	return v, true
}

func checkKind(m Metric, raw []byte) error {
	switch m.Kind {
	case "int":
		var n int64
		if json.Unmarshal(raw, &n) != nil {
			return fmt.Errorf("%s (%s) is not an integer: %s", m.Name, m.Path, raw)
		}
	case "ratio":
		var r struct{ Num, Den *int64 }
		if json.Unmarshal(raw, &r) != nil || r.Num == nil || r.Den == nil {
			return fmt.Errorf("%s (%s) is not a ratio: %s", m.Name, m.Path, raw)
		}
	}
	return nil
}

// intValue is an int metric's value; ok is false when it is absent.
func intValue(v Value) (int64, bool) {
	var n *int64
	if json.Unmarshal(v.Value, &n) != nil || n == nil {
		return 0, false
	}
	return *n, true
}
