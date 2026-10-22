package tui

import (
	"errors"
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/georgezero/dreamteam-go/internal/eval"
	"github.com/georgezero/dreamteam-go/internal/session"
)

// The T90 view (summary screen, tab): the management evaluation of the
// player's plan under the evaluation policy in force (session.T90). It
// runs in the background on a snapshot of the plan; play may go on.

// t90Msg delivers a finished evaluation.
type t90Msg struct {
	res *session.T90
	err error
}

// t90Point is the session's time and action count when an evaluation was
// taken: a later point makes the shown result stale.
type t90Point struct {
	at      string
	actions int
}

func (m *Model) t90Now() t90Point {
	return t90Point{at: m.sess.Calendar().ISO(m.sess.Now()), actions: len(m.sess.ActionLog())}
}

// t90Key handles the T90 view's keys: enter evaluates, ↑↓ scroll.
func (m *Model) t90Key(k string) (tea.Cmd, bool) {
	switch k {
	case "enter":
		if m.t90Busy {
			return nil, true
		}
		job, err := m.sess.T90Job()
		if err != nil {
			m.t90, m.t90Err = nil, err
			return nil, true
		}
		m.t90Busy, m.t90Err, m.t90Taken = true, nil, m.t90Now()
		return func() tea.Msg {
			res, err := job.Run()
			return t90Msg{res, err}
		}, true
	case "up", "k":
		m.t90Scroll = max(0, m.t90Scroll-1)
		return nil, true
	case "down", "j":
		m.t90Scroll++
		return nil, true
	}
	return nil, false
}

// t90Body renders the T90 view.
func (m *Model) t90Body(w, n int) []string {
	lines := []string{titleStyle.Render(section("T90 EVALUATION · your plan against no action, by the evaluation policy in force", w))}
	add := func(s string) { lines = append(lines, s) }
	switch {
	case m.t90Busy:
		add(" Evaluating: your plan and the no-action baseline run from the takeover to the policy's observation")
		add(" end (about 40 s on a Raspberry Pi). Play may go on meanwhile.")
		return lines
	case m.t90Err != nil:
		add(" " + t90ErrText(m.t90Err))
		return lines
	case m.t90 == nil:
		for _, s := range []string{
			" The T90 evaluation judges your decisions before day 90 against doing nothing, on the same scenario",
			" and seed, by the frozen evaluation policy (v0.2-eval-4): the metric vector (profit, cash, served",
			" demand, report breaches by priority), the service gates and the policy's verdict.",
			" Days you have not played yet are projected with your plan as it stands and no further decision;",
			" after day 90 both plans run with no decision (the evaluation tail).",
			" It is defined for the sustainable scenario (dreamteam -scenario sustainable).",
			"",
			" enter: evaluate · ↑↓ scroll · tab: back to the day",
		} {
			add(s)
		}
		return lines
	}
	if p := m.t90Now(); p != m.t90Taken {
		add(alertStyle.Render(fmt.Sprintf(" Stale: evaluated at %s with %d actions; press enter to evaluate the plan as it is now.",
			isoShort(m.t90Taken.at), m.t90Taken.actions)))
	}
	body := m.t90.Lines(w - 1)
	m.t90Scroll = min(m.t90Scroll, max(0, len(body)-(n-len(lines))))
	for _, l := range body[m.t90Scroll:] {
		if len(lines) >= n {
			break
		}
		add(" " + l)
	}
	return lines
}

func t90ErrText(err error) string {
	switch {
	case errors.Is(err, session.ErrT90Scenario):
		return "Not available here: the evaluation policy is defined for the sustainable scenario (dreamteam -scenario sustainable)."
	case errors.Is(err, session.ErrT90Started):
		return "Nothing to evaluate yet: start play (space) first."
	case errors.Is(err, eval.ErrSealed):
		return "Not available for this seed: it belongs to the sealed evaluation holdout."
	case errors.Is(err, session.ErrUnsavable):
		return "Not available: this session has no scenario file."
	}
	return "The evaluation failed: " + err.Error()
}
