package bench

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/georgezero/dreamteam-go/internal/agent"
	"github.com/georgezero/dreamteam-go/internal/session"
)

// fakePractice answers conservative-v2's requests from synthetic public
// observations (V0.6c qualification): one decision point a day at 07:00,
// the urgent and soon breaches logged since the previous one, cash, the
// trailing 30 days' profit, two radiologists' allocations, and validate's
// and preview's answers. It applies what the policy submits as the
// simulator would: pending until its effective date, then in effect. It
// records every operation the policy calls.
type fakePractice struct {
	t           *testing.T
	day         int
	seq         uint64
	log         []logEntry
	cash        int64
	profit      int64
	alloc       map[string]int64 // in effect
	scenario    map[string]int64
	pending     map[string][2]int64 // employee → (effective day, bp)
	valid       bool
	displace    bool
	added       int64 // preview's added salary per 30 days, per 5 points
	ops         []string
	observeArgs []map[string]any
}

func newFake(t *testing.T) *fakePractice {
	return &fakePractice{t: t, cash: 10_000_000, profit: 2_000_000, valid: true, added: 250_000,
		alloc: map[string]int64{"E07": 6000, "E08": 5000}, scenario: map[string]int64{"E07": 6000, "E08": 5000},
		pending: map[string][2]int64{}}
}

var fakeStart = time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)

func (f *fakePractice) date(day int) string { return fakeStart.AddDate(0, 0, day).Format("2006-01-02") }

// breaches logs u urgent and s soon breaches during the day before
// decision point day.
func (f *fakePractice) breaches(day, u, s int) {
	at := fakeStart.AddDate(0, 0, day-1).Add(15 * time.Hour).Format("2006-01-02T15:04:05")
	for i := 0; i < u+s; i++ {
		pr := "soon"
		if i < u {
			pr = "urgent"
		}
		f.seq++
		f.log = append(f.log, logEntry{Seq: f.seq, Time: at, Severity: "warn", Category: "reading",
			Text: fmt.Sprintf("S%05d %s report overdue: CT Head not final after 4h0m0s (unread)", f.seq, pr)})
		f.seq++ // an unrelated entry between breaches
		f.log = append(f.log, logEntry{Seq: f.seq, Time: at, Severity: "info", Category: "reading", Text: "S00001 CT Head on the worklist (soon)"})
	}
}

func (f *fakePractice) Handle(line []byte) []byte {
	var req struct {
		ID   any            `json:"id"`
		Op   string         `json:"op"`
		Args map[string]any `json:"args"`
	}
	json.Unmarshal(line, &req)
	f.ops = append(f.ops, req.Op)
	var res any
	switch req.Op {
	case "observe":
		f.observeArgs = append(f.observeArgs, req.Args)
		log := f.log
		if n := 2000; len(log) > n {
			log = log[len(log)-n:]
		}
		res = map[string]any{"observation": map[string]any{
			"date": f.date(f.day), "time": f.date(f.day) + "T07:00:00", "recentLog": log,
			"finance": map[string]any{"cashCents": f.cash, "trailing30": map[string]any{"operatingProfitCents": f.profit}}}}
	case "actions":
		var ts []map[string]any
		for _, e := range []string{"E07", "E08"} {
			versions := []map[string]any{{"effective": "", "siteBP": f.scenario[e], "pending": false}}
			if p, ok := f.pending[e]; ok {
				versions = append(versions, map[string]any{"effective": f.date(int(p[0])), "siteBP": p[1], "pending": true})
			}
			ts = append(ts, map[string]any{"employee": e, "inEffectBP": f.alloc[e], "editable": true, "maxBP": 10000,
				"earliestEffective": f.date(f.day + 14), "versions": versions})
		}
		raw, _ := json.Marshal(ts)
		res = map[string]any{"types": []map[string]any{{"type": "set_reading_allocation", "targets": json.RawMessage(raw)}}}
	case "validate":
		res = map[string]any{"valid": f.valid}
	case "preview":
		var d []any
		if f.displace {
			d = []any{map[string]any{"appointment": "A1"}}
		}
		res = map[string]any{"preview": map[string]any{"ok": true, "displacements": d,
			"allocations": []map[string]any{{"beforeSalary30Cents": 1_000_000, "afterSalary30Cents": 1_000_000 + f.added}}}}
	default:
		raw, _ := json.Marshal(map[string]any{"id": req.ID, "ok": false, "error": map[string]any{"code": "not_permitted", "message": req.Op}})
		return raw
	}
	raw, _ := json.Marshal(map[string]any{"id": req.ID, "ok": true, "result": res})
	return raw
}

type change struct {
	Day       int
	Employee  string
	BP        int64
	Effective string
}

// run plays days decision points; perDay sets each day's breaches and
// conditions before the policy decides. It applies what the policy
// submits, and returns the changes.
func (f *fakePractice) run(p *ConservativeV2, days int, perDay func(day int)) []change {
	f.t.Helper()
	c := &agent.Client{H: f}
	var out []change
	for f.day = 1; f.day <= days; f.day++ {
		for e, pd := range f.pending {
			if int64(f.day) >= pd[0] {
				f.alloc[e] = pd[1]
				delete(f.pending, e)
			}
		}
		perDay(f.day)
		bs, err := p.Decide(c)
		if err != nil {
			f.t.Fatal(err)
		}
		if len(bs) > 1 || (len(bs) == 1 && len(bs[0]) != 1) {
			f.t.Fatalf("day %d: %d batches", f.day, len(bs))
		}
		for _, b := range bs {
			var a struct {
				Type, Employee, Effective, Op string
				BP                            int64 `json:"siteAllocationBP"`
			}
			json.Unmarshal(b[0], &a)
			if a.Type != "set_reading_allocation" || a.Op != "set" {
				f.t.Fatalf("day %d: %s", f.day, b[0])
			}
			if a.Effective != f.date(f.day+14) {
				f.t.Fatalf("day %d: effective %s, the earliest lawful date is %s", f.day, a.Effective, f.date(f.day+14))
			}
			eff, _ := time.Parse("2006-01-02", a.Effective)
			f.pending[a.Employee] = [2]int64{int64(eff.Sub(fakeStart).Hours() / 24), a.BP}
			out = append(out, change{f.day, a.Employee, a.BP, a.Effective})
		}
	}
	return out
}

// 1. A sustained problem triggers an action; 5. at the earliest lawful
// date (checked in run); 8. through permitted operations only, with no
// seed asked for.
func TestConservativeV2SustainedActs(t *testing.T) {
	for name, perDay := range map[string]func(*fakePractice, int){
		"soon on every day":             func(f *fakePractice, d int) { f.breaches(d, 0, 2) },
		"urgent on 3 days of 7":         func(f *fakePractice, d int) { f.breaches(d, b2i(d%7 < 3), 0) },
		"soon on 5 days of 7 (Mon–Fri)": func(f *fakePractice, d int) { f.breaches(d, 0, b2i(d%7 < 5)) },
	} {
		f := newFake(t)
		ch := f.run(&ConservativeV2{}, 10, func(d int) { perDay(f, d) })
		if len(ch) != 1 || ch[0].Day != 7 || ch[0].Employee != "E08" || ch[0].BP != 5500 {
			t.Errorf("%s: changes %+v", name, ch)
		}
		for _, op := range f.ops {
			if !slices.Contains([]string{"observe", "actions", "validate", "preview"}, op) {
				t.Errorf("%s: called %s", name, op)
			}
		}
		for _, a := range f.observeArgs {
			if fmt.Sprint(a["sections"]) != "[finance recentLog]" || a["compact"] != true {
				t.Errorf("%s: observe %v", name, a)
			}
		}
	}
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// 2. A transient problem does not: a weekend's breaches (two days of
// seven), a single bad day, urgent breaches on two days of seven.
func TestConservativeV2TransientDoesNot(t *testing.T) {
	for name, perDay := range map[string]func(*fakePractice, int){
		"soon on two days of seven":   func(f *fakePractice, d int) { f.breaches(d, 0, 3*b2i(d%7 >= 5)) },
		"one bad day":                 func(f *fakePractice, d int) { f.breaches(d, 2*b2i(d == 4), 9*b2i(d == 4)) },
		"urgent on two days of seven": func(f *fakePractice, d int) { f.breaches(d, b2i(d%7 < 2), 0) },
		"soon on four days of seven":  func(f *fakePractice, d int) { f.breaches(d, 0, b2i(d%7 < 4)) },
	} {
		f := newFake(t)
		if ch := f.run(&ConservativeV2{}, 35, func(d int) { perDay(f, d) }); len(ch) != 0 {
			t.Errorf("%s: changes %+v", name, ch)
		}
	}
}

// 3. Financial guardrails prevent spending: a trailing-30-day loss, a raise
// whose cost would make one, cash below $50,000.
func TestConservativeV2FinancialGuardrails(t *testing.T) {
	for name, set := range map[string]func(*fakePractice){
		"losing money":               func(f *fakePractice) { f.profit = -1 },
		"the raise would lose money": func(f *fakePractice) { f.profit = 249_999 },
		"cash below the floor":       func(f *fakePractice) { f.cash = 4_999_999 },
	} {
		f := newFake(t)
		set(f)
		if ch := f.run(&ConservativeV2{}, 35, func(d int) { f.breaches(d, 1, 3) }); len(ch) != 0 {
			t.Errorf("%s: changes %+v", name, ch)
		}
	}
	f := newFake(t)
	f.profit = 250_000 // exactly affordable
	if ch := f.run(&ConservativeV2{}, 8, func(d int) { f.breaches(d, 1, 3) }); len(ch) != 1 {
		t.Errorf("affordable: changes %+v", ch)
	}
}

// 4. Cooldowns and the review: after a raise, nothing until 14 days after
// it takes effect; a raise that cut breaches by a quarter allows another
// (to the 10-point cap); one that did not is stepped back, and no raise
// follows.
func TestConservativeV2CooldownAndReview(t *testing.T) {
	// Breaches fall by more than a quarter once the first raise takes
	// effect (day 21): the review on day 35 finds it helped, and a second
	// raise follows (E08 again, still the lowest in effect). Its own review
	// (day 63) finds no further fall, so it is stepped back and no raise
	// follows.
	f := newFake(t)
	ch := f.run(&ConservativeV2{}, 90, func(d int) {
		if d < 21 {
			f.breaches(d, 1, 3)
		} else {
			f.breaches(d, b2i(d%2 == 0), 1)
		}
	})
	want := []change{{7, "E08", 5500, "2026-11-03"}, {35, "E08", 6000, "2026-12-01"}, {63, "E08", 5500, "2026-12-29"}}
	if fmt.Sprint(ch) != fmt.Sprint(want) {
		t.Fatalf("helped once: changes %+v", ch)
	}
	// Breaches keep falling with each raise: raises continue to the
	// 10-point cap above the scenario levels in all, and no further.
	f = newFake(t)
	ch = f.run(&ConservativeV2{}, 120, func(d int) {
		above := f.alloc["E07"] - 6000 + f.alloc["E08"] - 5000
		f.breaches(d, 1, max(1, 12-int(above)/50))
	})
	if len(ch) != 2 || ch[0].BP != 5500 || ch[1].BP != 6000 {
		t.Fatalf("cap: changes %+v", ch)
	}
	// Breaches unchanged: the review on day 35 steps the raise back (E08
	// to 5000), and nothing else follows although the problem persists.
	f = newFake(t)
	ch = f.run(&ConservativeV2{}, 90, func(d int) { f.breaches(d, 1, 3) })
	if len(ch) != 2 || ch[1].Day != 35 || ch[1].Employee != "E08" || ch[1].BP != 5000 {
		t.Fatalf("did not help: changes %+v", ch)
	}
	// A pending change (made elsewhere) means wait.
	f = newFake(t)
	f.pending["E07"] = [2]int64{60, 6500}
	if ch := f.run(&ConservativeV2{}, 40, func(d int) { f.breaches(d, 1, 3) }); len(ch) != 0 {
		t.Fatalf("pending: changes %+v", ch)
	}
}

// 6. Stable conditions: no breach, or breaches on at most one day a week,
// allow no action; and a raise is stepped back once service has been
// clear for a week, or when the practice starts losing money.
func TestConservativeV2Stable(t *testing.T) {
	f := newFake(t)
	if ch := f.run(&ConservativeV2{}, 60, func(d int) { f.breaches(d, 0, b2i(d%7 == 0)) }); len(ch) != 0 {
		t.Fatalf("stable: changes %+v", ch)
	}
	f = newFake(t)
	ch := f.run(&ConservativeV2{}, 60, func(d int) {
		if d < 14 {
			f.breaches(d, 1, 3)
		}
	})
	if len(ch) != 2 || ch[1].Day != 35 || ch[1].BP != 5000 {
		t.Fatalf("cleared: changes %+v", ch)
	}
	f = newFake(t)
	ch = f.run(&ConservativeV2{}, 60, func(d int) {
		f.breaches(d, 1, 3)
		if d >= 25 {
			f.profit = -1
		}
	})
	if len(ch) != 2 || ch[1].Day != 35 || ch[1].BP != 5000 {
		t.Fatalf("losing money: changes %+v", ch)
	}
}

// 7. Invalid changes never apply: validate refusing, or the preview
// displacing a booked patient, means no submission.
func TestConservativeV2InvalidNeverSubmitted(t *testing.T) {
	for name, set := range map[string]func(*fakePractice){
		"validate refuses":  func(f *fakePractice) { f.valid = false },
		"preview displaces": func(f *fakePractice) { f.displace = true },
	} {
		f := newFake(t)
		set(f)
		if ch := f.run(&ConservativeV2{}, 30, func(d int) { f.breaches(d, 1, 3) }); len(ch) != 0 {
			t.Errorf("%s: changes %+v", name, ch)
		}
		if !slices.Contains(f.ops, "validate") {
			t.Errorf("%s: never validated", name)
		}
	}
}

// The log is read once: breaches are counted from the entries since the
// last decision point, not twice; at the first, from the last 24 hours.
func TestConservativeV2Counting(t *testing.T) {
	f := newFake(t)
	f.day = 0
	f.breaches(-3, 0, 5) // three days before the first decision: not counted
	p := &ConservativeV2{}
	c := &agent.Client{H: f}
	f.day = 1
	f.breaches(1, 1, 2)
	p.Decide(c)
	f.day = 2
	f.breaches(2, 0, 1)
	p.Decide(c)
	if len(p.days) != 2 || p.days[0].Urgent != 1 || p.days[0].Soon != 2 || p.days[1].Urgent != 0 || p.days[1].Soon != 1 {
		t.Fatalf("%+v", p.days)
	}
	if u, s := breach(logEntry{Category: "reading", Text: "S00001 routine report overdue: x"}); u || s {
		t.Error("routine counted")
	}
}

// The breach signal is the simulator's own: real "report overdue" entries
// in the public log, read through observe as the policy reads them, parse
// as urgent or soon breaches (test seed 7, 8 days of the sustainable
// practice).
func TestConservativeV2ReadsTheRealLog(t *testing.T) {
	s, err := session.Open("sustainable", 7)
	if err != nil {
		t.Fatal(err)
	}
	c := &agent.Client{H: agent.New(s)}
	if err := c.Call("advance", map[string]any{"to": "2026-10-21T07:00:00"}, nil); err != nil {
		t.Fatal(err)
	}
	var obs struct {
		Observation struct {
			RecentLog []logEntry `json:"recentLog"`
		} `json:"observation"`
	}
	if err := c.Call("observe", map[string]any{"sections": []string{"finance", "recentLog"}, "compact": true, "recentLog": 2000}, &obs); err != nil {
		t.Fatal(err)
	}
	// One entry per breached study ("S08624 soon report overdue: …"); the
	// alert summaries ("1 urgent report overdue (target 1h) · …",
	// "Resolved: …") are not breaches and are not counted.
	var u, so, studies, summaries int
	for _, e := range obs.Observation.RecentLog {
		a, b := breach(e)
		u, so = u+b2i(a), so+b2i(b)
		switch {
		case strings.HasPrefix(e.Text, "S") && strings.Contains(e.Text, " report overdue: "):
			studies++
		case strings.Contains(e.Text, "report overdue"):
			summaries++
			if a || b {
				t.Errorf("alert summary counted: %q", e.Text)
			}
		}
	}
	if studies == 0 || summaries == 0 || u+so != studies || so == 0 {
		t.Fatalf("%d breach entries, %d summaries, %d urgent and %d soon breaches parsed", studies, summaries, u, so)
	}
	t.Logf("%d log entries: %d urgent, %d soon breaches", len(obs.Observation.RecentLog), u, so)
}
