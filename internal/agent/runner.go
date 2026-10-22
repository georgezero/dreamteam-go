package agent

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Client drives a Handler through the protocol's bytes only, as an
// external agent would: it builds requests, numbers them, and reads the
// responses. A transcript, when set, receives every request and response
// line.
type Client struct {
	H          *Handler
	Transcript io.Writer
	next       int
}

// Call sends one request and decodes the result into out (when not nil).
func (c *Client) Call(op string, args any, out any) error {
	c.next++
	req := map[string]any{"id": c.next, "op": op}
	if args != nil {
		req["args"] = args
	}
	line, err := json.Marshal(req)
	if err != nil {
		return err
	}
	resp := c.H.Handle(line)
	if c.Transcript != nil {
		fmt.Fprintf(c.Transcript, "%s\n%s\n", line, resp)
	}
	var r struct {
		OK     bool            `json:"ok"`
		Result json.RawMessage `json:"result"`
		Error  *Error          `json:"error"`
	}
	if err := json.Unmarshal(resp, &r); err != nil {
		return err
	}
	if !r.OK {
		return r.Error
	}
	if out != nil {
		return json.Unmarshal(r.Result, out)
	}
	return nil
}

// A Policy decides each morning what to submit, from what the protocol
// shows it. It returns the batches to apply (none: no decision).
type Policy interface {
	Name() string
	Decide(c *Client) ([][]json.RawMessage, error)
}

// Noop never decides.
type Noop struct{}

func (Noop) Name() string                                { return "noop" }
func (Noop) Decide(*Client) ([][]json.RawMessage, error) { return nil, nil }

// Greedy is a deliberately simple policy for exercising the interface,
// not a strategy: when the oldest unread study is more than a day old, it
// raises the site reading allocation of the first radiologist with room
// to grow and no pending change by 5 points, from the earliest date. It
// reads only the observation's worklist and the action catalog.
type Greedy struct{}

func (Greedy) Name() string { return "greedy" }

func (Greedy) Decide(c *Client) ([][]json.RawMessage, error) {
	var obs struct {
		Observation struct {
			Worklist struct {
				OldestMin int64 `json:"oldestMin"`
			} `json:"worklist"`
		} `json:"observation"`
	}
	if err := c.Call("observe", map[string]any{"recentLog": 0}, &obs); err != nil {
		return nil, err
	}
	if obs.Observation.Worklist.OldestMin <= 24*60 {
		return nil, nil
	}
	var cat struct {
		Types []struct {
			Type    string          `json:"type"`
			Targets json.RawMessage `json:"targets"`
		} `json:"types"`
	}
	if err := c.Call("actions", nil, &cat); err != nil {
		return nil, err
	}
	for _, t := range cat.Types {
		if t.Type != "set_reading_allocation" {
			continue
		}
		var targets []struct {
			Employee          string `json:"employee"`
			InEffectBP        int64  `json:"inEffectBP"`
			EarliestEffective string `json:"earliestEffective"`
			MaxBP             int64  `json:"maxBP"`
			Versions          []struct {
				Pending bool `json:"pending"`
			} `json:"versions"`
		}
		if err := json.Unmarshal(t.Targets, &targets); err != nil {
			return nil, err
		}
		for _, a := range targets {
			pending := false
			for _, v := range a.Versions {
				pending = pending || v.Pending
			}
			if pending || a.InEffectBP >= a.MaxBP {
				continue
			}
			act, _ := json.Marshal(map[string]any{"type": "set_reading_allocation", "op": "set", "employee": a.Employee,
				"effective": a.EarliestEffective, "siteAllocationBP": min(a.MaxBP, a.InEffectBP+500)})
			return [][]json.RawMessage{{act}}, nil
		}
	}
	return nil, nil
}

// RunSummary is a scripted run's outcome.
type RunSummary struct {
	Policy   string `json:"policy"`
	Days     int    `json:"days"`
	Requests int    `json:"requests"`
	Applied  int    `json:"applied"` // batches applied
	Refused  int    `json:"refused"` // batches validated and not applied
	Clock    Clock  `json:"clock"`
}

// Run plays days days with p through the protocol: hello, then the
// policy's decisions at the start and each following day at 07:00
// practice time (each batch applied only if valid), the clock moving to
// the next day's 07:00 in between.
func Run(c *Client, p Policy, days int) (RunSummary, error) {
	sum := RunSummary{Policy: p.Name(), Days: days}
	var hello Hello
	if err := c.Call("hello", nil, &hello); err != nil {
		return sum, err
	}
	date := hello.Clock.Time[:10]
	for d := 0; d < days; d++ {
		bs, err := p.Decide(c)
		if err != nil {
			return sum, err
		}
		for _, b := range bs {
			var r Applied
			if err := c.Call("apply", map[string]any{"actions": b}, &r); err != nil {
				return sum, err
			}
			if r.Applied {
				sum.Applied++
			} else {
				sum.Refused++
			}
		}
		next, err := nextDate(date)
		if err != nil {
			return sum, err
		}
		var a Advanced
		if err := c.Call("advance", map[string]any{"to": next + "T07:00:00"}, &a); err != nil {
			return sum, err
		}
		date = next
		sum.Clock = a.Clock
	}
	sum.Requests = c.next
	return sum, nil
}

// nextDate is the calendar day after date (YYYY-MM-DD).
func nextDate(date string) (string, error) {
	d, err := simtime.ParseDate(date)
	if err != nil {
		return "", err
	}
	return d.AddDays(1).String(), nil
}
