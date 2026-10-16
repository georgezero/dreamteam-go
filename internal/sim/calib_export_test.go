//go:build calibration

package sim

import (
	"slices"

	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// LaborPosting is one posted labor entry, for the dynamic staffing
// benchmark's cost decomposition (calib_dynamic_test.go): the expense it
// booked to payroll, benefits, recruiting or severance. Read-only.
type LaborPosting struct {
	Kind     string // "worked", "salary", "guarantee", "recruiting", "severance"
	Employee string // employee ID
	Day      int64  // the day it is for (worked, salary, guarantee), else the posting day
	At       simtime.Time
	Cents    int64
}

// LaborPostings lists every labor posting from from on, a reversal
// attributed (negated) to the entry it reverses.
func (w *World) LaborPostings(from simtime.Time) []LaborPosting {
	st := w.st
	led := &st.fin.led
	kinds := map[entryKind]string{entWages: "worked", entSalary: "salary", entGuarantee: "guarantee",
		entRecruiting: "recruiting", entSeverance: "severance"}
	var out []LaborPosting
	for _, e := range led.entries {
		if e.at < from {
			continue
		}
		key := e.key
		if key.kind == entReversal {
			key = led.entries[key.a].key
		}
		kind, ok := kinds[key.kind]
		if !ok {
			continue
		}
		var c int64
		for _, ln := range e.lines {
			switch ln.acct {
			case acctWages, acctBenefits, acctRecruiting, acctSeverance:
				c += ln.amt
			}
		}
		day := st.cal.DayIndex(e.at)
		if key.kind == entWages || key.kind == entSalary || key.kind == entGuarantee {
			day = key.b
		}
		out = append(out, LaborPosting{Kind: kind, Employee: st.emps[key.a].terms.ID, Day: day, At: e.at, Cents: c})
	}
	return out
}

// Backlog is the reporting backlog now (unread and reading studies): its
// count, the net revenue it will recognize when reported (expected net,
// at each order's payer), and the mean and p90 age since imaging.
func (w *World) Backlog() (n int, revenueCents int64, meanAge, p90Age simtime.Duration) {
	st := w.st
	var ages []simtime.Duration
	var total simtime.Duration
	for si := range st.dem.studies {
		s := &st.dem.studies[si]
		if s.state != studyUnread && s.state != studyReading {
			continue
		}
		o := &st.dem.orders[s.order]
		revenueCents += st.cfg.Finance.ExpectedNetCents(&st.cfg.Catalog[s.exam], o.payer)
		a := st.now.Sub(s.acquiredAt)
		ages = append(ages, a)
		total += a
	}
	if len(ages) == 0 {
		return 0, 0, 0, 0
	}
	slices.Sort(ages)
	return len(ages), revenueCents, total / simtime.Duration(len(ages)), ages[(len(ages)*9+9)/10-1]
}
