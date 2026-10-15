package sim

import (
	"testing"

	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// V0.2 Slice 1: guaranteed contract pay (V0.2_DESIGN §5.2, §5.4). One
// hourly employee is driven day by day through the same pay paths the
// engine uses (worked pay when going home, the guarantee at day close),
// so the design's worked examples are checked to the cent.

// monday is a Monday; day index 0 of the test calendar.
var monday = simtime.Date{Year: 2026, Month: 10, Day: 12}

// payrollState is a minimal state with one hourly employee at rate cents
// an hour, contracted for weekly, rostered 8 hours Monday to Friday in
// both rota weeks.
func payrollState(t *testing.T, rate int64, weekly simtime.Duration) *state {
	t.Helper()
	e := EmployeeConfig{ID: "E1", Name: "Pat", Role: Technologist, Skills: []string{"XR"}, Pay: Hourly, RateCents: rate}
	for wd := simtime.Monday; wd <= simtime.Friday; wd++ {
		e.Roster[wd] = &ShiftTemplate{Span: Span{8 * simtime.Hour, 16 * simtime.Hour}, Assignment: "R1"}
	}
	e.Contract = InferContract(&e)
	e.Contract.Weekly, e.Contract.Inferred = weekly, false
	cfg := &Config{Start: monday, RotaWeekA: monday, Employees: []EmployeeConfig{e},
		Rooms:    []RoomConfig{{ID: "R1", Name: "X-ray", Modality: "XR"}},
		Policies: Policies{WeeklyHours: 40 * simtime.Hour, OvertimeBP: 15_000},
		Finance:  FinanceConfig{BenefitsBP: 2500, PayrollAnchor: monday.AddDays(4), PayrollEveryDays: 14}}
	st := &state{cfg: cfg, cal: simtime.NewCalendar(monday)}
	st.emps = []employee{{cfg: &cfg.Employees[0], shift: -1}}
	st.initDemand()
	st.initFinance()
	st.initMetrics()
	st.initStaff()
	return st
}

// work pays day's worked time (going home at 20:00, as goOff does) and
// closes the day (23:59:59).
func (st *state) work(day int64, paid simtime.Duration) {
	if st.cal.Weekday(st.cal.StartOfDay(day)) == simtime.Monday {
		st.emps[0].weekPaid = 0
	}
	st.day = day
	if paid > 0 {
		st.now = st.cal.At(day, 20*simtime.Hour)
		st.payWages(0, paid)
		st.emps[0].weekPaid += paid
	}
	st.now = st.cal.At(day, simtime.Day-1)
	st.guaranteeClose()
}

// expense sums the payroll and benefits expense posted for the week
// starting on day from.
func (st *state) labor(from int64) (wages, benefits int64) {
	for d := from; d < from+7 && d < int64(len(st.fin.led.days)); d++ {
		wages += st.fin.led.days[d].net[acctWages]
		benefits += st.fin.led.days[d].net[acctBenefits]
	}
	return wages, benefits
}

func TestGuaranteeWorkedExamples(t *testing.T) {
	const h = simtime.Hour
	for _, c := range []struct {
		name           string
		contract       simtime.Duration
		days           []simtime.Duration // Monday onwards
		wages, benefit int64
	}{
		// §5.4: C=40 h, r=$40.
		{"no work", 40 * h, nil, 160_000, 40_000},
		{"32 of 40", 40 * h, []simtime.Duration{8 * h, 8 * h, 8 * h, 8 * h}, 160_000, 40_000},
		// Base 45 h and the 0.5× premium on five hours; no top-up.
		{"45 h", 40 * h, []simtime.Duration{9 * h, 9 * h, 9 * h, 9 * h, 9 * h}, 190_000, 47_500},
		// Worked hours above a smaller contract: no top-up, no premium.
		{"28 of 20", 20 * h, []simtime.Duration{7 * h, 7 * h, 7 * h, 7 * h}, 112_000, 28_000},
		// Exactly the contract: worked pay replaces the guarantee entirely.
		{"40 of 40", 40 * h, []simtime.Duration{8 * h, 8 * h, 8 * h, 8 * h, 8 * h}, 160_000, 40_000},
	} {
		t.Run(c.name, func(t *testing.T) {
			st := payrollState(t, 4000, c.contract)
			for d := int64(0); d < 7; d++ {
				var paid simtime.Duration
				if int(d) < len(c.days) {
					paid = c.days[d]
				}
				st.work(d, paid)
			}
			w, b := st.labor(0)
			if w != c.wages || b != c.benefit {
				t.Fatalf("week: wages %d benefits %d, want %d and %d", w, b, c.wages, c.benefit)
			}
			// The guarantee is never worked time: weekly hours (and so
			// overtime) count only what was worked.
			var worked simtime.Duration
			for _, d := range c.days {
				worked += d
			}
			if st.emps[0].weekPaid != worked {
				t.Fatalf("week paid time %s, worked %s", st.emps[0].weekPaid, worked)
			}
			// Week closed: everything owed is ordinary accrued payroll,
			// and nothing is left provisional.
			if open := st.openGuarantee(); open != 0 {
				t.Fatalf("open guarantee %d after the week closed", open)
			}
			if got := -st.fin.led.bal[acctAccruedPayroll]; got != c.wages+c.benefit {
				t.Fatalf("accrued payroll %d, want %d", got, c.wages+c.benefit)
			}
			if err := st.checkFinance(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// The provision follows the cumulative shortfall day by day: posted when
// the guarantee runs ahead of worked pay, reversed (signed) as worked pay
// catches up, never a second payment for the same hours.
func TestGuaranteeProvisionReverses(t *testing.T) {
	const h = simtime.Hour
	st := payrollState(t, 4000, 40*h)
	st.work(0, 0) // Monday off: guarantee 8 h ahead
	if got := st.openGuarantee(); got != 32_000+8_000 {
		t.Fatalf("Monday: open guarantee %d, want 40000", got)
	}
	st.work(1, 16*h) // Tuesday double: caught up
	if got := st.openGuarantee(); got != 0 {
		t.Fatalf("Tuesday: open guarantee %d, want 0", got)
	}
	var signs []int64
	for _, en := range st.fin.led.entries {
		if en.key.kind == entGuarantee {
			signs = append(signs, en.lines[0].amt)
		}
	}
	if len(signs) != 2 || signs[0] != 32_000 || signs[1] != -32_000 {
		t.Fatalf("guarantee postings %v, want [32000 -32000]", signs)
	}
	for d := int64(2); d < 7; d++ {
		var paid simtime.Duration
		if d <= 4 {
			paid = 8 * h // Wednesday to Friday
		}
		st.work(d, paid)
	}
	// 40 h worked in the week, unevenly: exactly r×C, nothing on top.
	if w, b := st.labor(0); w != 160_000 || b != 40_000 {
		t.Fatalf("40 h worked: wages %d benefits %d", w, b)
	}
	// A zero change posts nothing: still only Monday's and Tuesday's.
	n := 0
	for _, en := range st.fin.led.entries {
		if en.key.kind == entGuarantee {
			n++
		}
	}
	if n != 2 {
		t.Fatalf("%d guarantee entries, want 2", n)
	}
}

// A partial first week guarantees only the employed days, by the
// contract's frozen weights; a full week after it guarantees C.
func TestGuaranteePartialWeek(t *testing.T) {
	const h = simtime.Hour
	st := payrollState(t, 4000, 40*h)
	st.staff[0].From = 2 // employed from Wednesday
	for d := int64(2); d < 14; d++ {
		var paid simtime.Duration
		if d == 2 || d == 3 {
			paid = 8 * h
		}
		st.work(d, paid)
	}
	// Wednesday to Friday: 24 h guaranteed, 16 worked.
	if w, b := st.labor(0); w != 96_000 || b != 24_000 {
		t.Fatalf("partial week: wages %d benefits %d, want 96000 and 24000", w, b)
	}
	if w, b := st.labor(7); w != 160_000 || b != 40_000 {
		t.Fatalf("full week without work: wages %d benefits %d", w, b)
	}
}

// Rounding: the guarantee is truncated per contract day, like legacy day
// wages, so it never exceeds r×C and falls short of it by at most a cent
// a contract day.
func TestGuaranteeRounding(t *testing.T) {
	const h = simtime.Hour
	st := payrollState(t, 3333, 39*h+30*simtime.Minute)
	for d := int64(0); d < 7; d++ {
		st.work(d, 0)
	}
	exact := 3333 * int64(39*h+30*simtime.Minute) / 3600
	w, _ := st.labor(0)
	if w > exact || exact-w > 5 {
		t.Fatalf("week guarantee %d, exact r×C %d", w, exact)
	}
}

// Inferred contracts: the smaller net rota week (breaks unpaid), with the
// original net rostered time per rota week and weekday as weights.
func TestInferContract(t *testing.T) {
	const h, m = simtime.Hour, simtime.Minute
	e := EmployeeConfig{Pay: Hourly}
	day := func(weeks int) *ShiftTemplate {
		return &ShiftTemplate{Span: Span{9*h + 30*m, 18 * h}, Weeks: weeks, Breaks: []Span{{13*h + 30*m, 14 * h}}}
	}
	e.Roster[simtime.Monday] = day(WeekA)
	for wd := simtime.Tuesday; wd <= simtime.Friday; wd++ {
		e.Roster[wd] = day(WeekBoth)
	}
	e.Roster[simtime.Saturday] = &ShiftTemplate{Span: Span{10 * h, 18 * h}, Weeks: WeekB, Breaks: []Span{{13*h + 30*m, 14 * h}}}
	c := InferContract(&e)
	if c.Weekly != 39*h+30*m || !c.Inferred {
		t.Fatalf("contract %s inferred %v, want 39h30m", c.Weekly, c.Inferred)
	}
	if c.Weights[0][simtime.Monday] != 8*h || c.Weights[1][simtime.Monday] != 0 ||
		c.Weights[1][simtime.Saturday] != 7*h+30*m || c.Weights[0][simtime.Saturday] != 0 {
		t.Fatalf("weights %v", c.Weights)
	}
	if s := InferContract(&EmployeeConfig{Pay: Salaried, Roster: e.Roster}); s != (Contract{}) {
		t.Fatalf("salaried contract %+v, want none", s)
	}
}

// The employment registry is state: the digest sees every term.
func TestEmploymentInStateDigest(t *testing.T) {
	st := payrollState(t, 4000, 40*simtime.Hour)
	w := &World{st: st, eng: NewEngine(st, 0)}
	before := w.StateDigest()
	st.staff[0].Contract.Weekly++
	if w.StateDigest() == before {
		t.Fatal("contract terms are not in the state digest")
	}
	st.staff[0].Contract.Weekly--
	st.staff[0].weekB++
	if w.StateDigest() == before {
		t.Fatal("week pay counters are not in the state digest")
	}
}
