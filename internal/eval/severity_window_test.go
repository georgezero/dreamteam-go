package eval

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// Slice 5C.4f: volume-aware severity windows (docs/calibration/
// clinical-5c4f/severity-windows.md): no urgent weekly gate; soon in
// fixed four-week calendar blocks (minimum 100); routine in fixed
// calendar weeks (minimum 100). Synthetic witnesses only.

// The block calendar of these witnesses: takeover Wednesday 2001-09-12
// 06:00, so the anchor is Monday 2001-09-17; horizon 63 days later,
// Wednesday 2001-11-14 06:00. Soon windows: the leading partial week
// (2001-09-10 to 09-16), two complete blocks (09-17 to 10-14, 10-15 to
// 11-11) and the trailing partial block from 11-12.
var (
	bwMon    = int64(11575) * daySec // Monday 2001-09-10
	bwTake   = bwMon + 2*daySec + 6*hour
	bwAnchor = bwMon + 7*daySec
	bwHorz   = bwTake + 63*daySec
)

// blockRecord is a record with, for each window start in at, n[i]
// studies of priority q acquired every 30 minutes from an hour after it,
// the first sev[i] severe (one hour beyond the priority's severe delay),
// the rest read within the hour.
func blockRecord(q string, at []int64, n, sev []int) *Record {
	r := &Record{TakeoverSec: bwTake, TakeoverDate: "2001-09-12", HorizonEndSec: bwHorz, TATTargetMin: testTargets}
	severe := map[string]int64{"urgent": 13 * hour, "soon": 25 * hour, "routine": 73 * hour}
	for i, from := range at {
		for j := range n[i] {
			acq := from + hour + int64(j)*1800
			fin := acq + hour
			if j < sev[i] {
				fin = acq + testTargets[q]*60 + severe[q]
			}
			r.Studies = append(r.Studies, Study{Priority: q, AcquiredSec: acq, FinalSec: fin})
		}
	}
	return r
}

// soonWindows: the leading partial week, the two complete blocks, the
// trailing partial block.
var soonStarts = []int64{bwTake, bwAnchor, bwAnchor + 28*daySec, bwAnchor + 56*daySec}

// soonBlocks judges them under the 5C.4f candidate (the 5C.4h count
// ceiling on partial windows: boundary_safety_test.go).
func soonBlocks(t *testing.T, n, sev []int) *WeeklySevere {
	t.Helper()
	p, _ := candidate5c4fPolicy(t)
	r := blockRecord("soon", soonStarts, n, sev)
	return weeklySevereCalendar(p.Clinical, r, bwHorz+40*daySec)["soon"]
}

// The soon blocks' layout: fixed 28-day windows from the anchor, exact
// dates, complete or partial, never slid.
func TestSoonBlockLayout(t *testing.T) {
	ws := soonBlocks(t, []int{0, 100, 100, 0}, []int{0, 0, 0, 0})
	type w struct {
		from, to string
		weeks    int64
		complete bool
	}
	var got []w
	for _, x := range ws.Weeks {
		got = append(got, w{x.From, x.To, x.Weeks, x.Complete})
	}
	want := []w{{"2001-09-10", "2001-09-16", 1, false}, {"2001-09-17", "2001-10-14", 4, true}, {"2001-10-15", "2001-11-11", 4, true},
		{"2001-11-12", "2001-12-09", 4, false}}
	if !slices.Equal(got, want) || ws.WindowWeeks != 4 || ws.PartialWeeks != 2 {
		t.Fatalf("%+v", got)
	}
	// The takeover on a Monday at 00:00: the anchor is the takeover, no
	// leading partial window.
	p, _ := candidatePolicy(t)
	r := blockRecord("soon", []int64{bwAnchor}, []int{100}, []int{0})
	r.TakeoverSec, r.TakeoverDate = bwAnchor, "2001-09-17"
	if ws := weeklySevereCalendar(p.Clinical, r, bwHorz+40*daySec)["soon"]; ws.Weeks[0].From != "2001-09-17" || !ws.Weeks[0].Complete || ws.Weeks[0].Acquired != 100 {
		t.Fatalf("Monday takeover: %+v", ws.Weeks[0])
	}
}

// 28-day boundaries: a study on the block's last second and one on the
// next block's first second fall in different blocks.
func TestSoonBlockBoundaries(t *testing.T) {
	p, _ := candidatePolicy(t)
	r := blockRecord("soon", soonStarts, []int{0, 100, 100, 0}, []int{0, 0, 0, 0})
	edge := bwAnchor + 28*daySec
	sev := testTargets["soon"]*60 + 25*hour
	r.Studies = append(r.Studies, Study{Priority: "soon", AcquiredSec: edge - 1, FinalSec: edge - 1 + sev},
		Study{Priority: "soon", AcquiredSec: edge, FinalSec: edge + sev}, Study{Priority: "soon", AcquiredSec: edge + 1, FinalSec: edge + 1 + sev})
	ws := weeklySevereCalendar(p.Clinical, r, bwHorz+40*daySec)["soon"]
	if a, b := ws.Weeks[1], ws.Weeks[2]; a.Acquired != 101 || a.Severe != 1 || b.Acquired != 102 || b.Severe != 2 || a.To != "2001-10-14" || b.From != "2001-10-15" {
		t.Fatalf("%+v / %+v", a, b)
	}
}

// The soon minimum and rate, exactly: 100 acquisitions judged, 99
// insufficient (a complete block); 5 of 100 (exactly 5%) passes, one more
// severe report (6 of 100) fails; blocks never pooled (99 then 101).
func TestSoonBlockMinimumAndRate(t *testing.T) {
	for _, x := range []struct {
		name   string
		n, sev []int
		want   string
	}{
		{"exactly 100, 5%", []int{0, 100, 100, 0}, []int{0, 5, 0, 0}, "not_judged pass pass not_judged"},
		{"one more severe", []int{0, 100, 100, 0}, []int{0, 6, 0, 0}, "not_judged fail pass not_judged"},
		{"99", []int{0, 99, 100, 0}, []int{0, 0, 0, 0}, "not_judged insufficient pass not_judged"},
		{"99 then 101, not pooled", []int{0, 99, 101, 0}, []int{0, 6, 0, 0}, "not_judged insufficient pass not_judged"},
		{"sparse: an empty complete block", []int{0, 0, 100, 0}, []int{0, 0, 0, 0}, "not_judged insufficient pass not_judged"},
	} {
		if got := statuses(soonBlocks(t, x.n, x.sev)); got != x.want {
			t.Errorf("%s: %s", x.name, got)
		}
	}
}

// Partial windows at both ends: judged when they reach the minimum (and
// failing above the rate), NOT_JUDGED below it, never a pass and never
// insufficient; studies after the horizon not counted.
func TestSoonBlockPartials(t *testing.T) {
	ws := soonBlocks(t, []int{100, 100, 100, 50}, []int{6, 0, 0, 10})
	if statuses(ws) != "fail pass pass not_judged" || ws.Insufficient || ws.Failing != 1 || ws.NotJudgedWeeks != 1 || ws.Weeks[3].Severe != 10 {
		t.Fatalf("%s %+v", statuses(ws), ws)
	}
	p, _ := candidate5c4fPolicy(t)
	r := blockRecord("soon", soonStarts, []int{0, 100, 100, 100}, []int{0, 0, 0, 6})
	r.Studies = append(r.Studies, Study{Priority: "soon", AcquiredSec: bwHorz + hour, FinalSec: bwHorz + 40*hour})
	ws = weeklySevereCalendar(p.Clinical, r, bwHorz+40*daySec)["soon"]
	if statuses(ws) != "not_judged pass pass fail" || ws.Acquired != 300 || ws.Weeks[3].Complete {
		t.Fatalf("horizon: %s %+v", statuses(ws), ws)
	}
	// The truncated block's studies before the horizon only: 100 of them
	// would need 50 hours; the horizon is 54 hours after its Monday.
	r = blockRecord("soon", soonStarts, []int{0, 100, 100, 120}, []int{0, 0, 0, 0})
	if ws := weeklySevereCalendar(p.Clinical, r, bwHorz+40*daySec)["soon"]; ws.Weeks[3].Acquired != 107 {
		t.Fatalf("truncated: %+v", ws.Weeks[3])
	}
}

// Routine is unchanged from 5C.4e: the same fixed weeks, minimum 100,
// 5%: identical results under both candidates.
func TestRoutineWeeksUnchanged(t *testing.T) {
	p, _ := candidate5c4fPolicy(t)
	old, _ := candidate5c4ePolicy(t)
	var at []int64
	for k := range 9 {
		at = append(at, bwMon+int64(k)*7*daySec)
	}
	at[0] = bwTake
	r := blockRecord("routine", at, []int{40, 100, 99, 100, 120, 100, 100, 100, 100}, []int{0, 5, 0, 6, 0, 0, 0, 0, 2})
	a := weeklySevereCalendar(p.Clinical, r, bwHorz+40*daySec)["routine"]
	b := weeklySevereCalendar(old.Clinical, r, bwHorz+40*daySec)["routine"]
	if !reflect.DeepEqual(a, b) || statuses(a) != "not_judged pass insufficient fail pass pass pass pass pass not_judged" {
		t.Fatalf("%s\n%+v\n%+v", statuses(a), a, b)
	}
}

// Urgent: no weekly gate (no window measured, no seed gate), while the
// per-seed severe rate and the absolute 24-hour limit stay. On the
// synthetic fixture: urgent reports 13 hours past due on 10 of the 150
// urgent studies of a seed fail its severe rate (above 5%); six urgent
// reports eight days late in one week break the absolute limit: NO GO
// for clinical safety, as under 5C.4e, now without a weekly gate.
func TestUrgentWeeklyGateDisabled(t *testing.T) {
	p, d := candidatePolicy(t)
	r := blockRecord("urgent", soonStarts, []int{10, 30, 30, 5}, []int{0, 10, 0, 0})
	if ws := weeklySevereCalendar(p.Clinical, r, bwHorz+40*daySec); ws["urgent"] != nil || ws["soon"] == nil || ws["routine"] == nil {
		t.Fatalf("urgent measured: %+v", ws)
	}
	late := func(n int, after int64) func(b synth, r *Record) {
		return func(b synth, r *Record) {
			for _, i := range studiesOf(r, "urgent", 60, 150)[:n] {
				finishAt(r, i, r.Studies[i].AcquiredSec+testTargets["urgent"]*60+after)
			}
		}
	}
	pr, c := clinicalCaseAB(t, p, d, nil, late(10, 13*hour))
	if gateStatus(pr, 1, GateSevereWeek, WinGated, "urgent") != "missing" || gateStatus(pr, 1, GateSevereWeek, WinGated, "soon") == "missing" ||
		gateStatus(pr, 1, GateSevereRate, WinGated, "urgent") != Fail || pr.ClinicalSafety.Status != Pass || c.Verdict != NOGO {
		t.Fatalf("severe rate: %s %s %v", gateStatus(pr, 1, GateSevereRate, WinGated, "urgent"), c.Verdict, c.Reasons)
	}
	if ws := pr.PerSeed[0].Measures.Clinical.WeeklySevere; ws["urgent"] != nil {
		t.Fatalf("urgent weekly measured: %+v", ws["urgent"])
	}
	pr, c = clinicalCaseAB(t, p, d, nil, late(6, 8*daySec))
	if pr.ClinicalSafety.Status != Fail || len(pr.ClinicalSafety.Seeds) != 40 || !hasReason(c, "absolute pastDue limit of 1440 min") {
		t.Fatalf("absolute: %+v %v", pr.ClinicalSafety, c.Reasons)
	}
	// One urgent report 13 hours late: within the 5% rate, no gate fails
	// for it, and no weekly gate exists to fail.
	pr, _ = clinicalCaseAB(t, p, d, nil, late(1, 13*hour))
	if gateStatus(pr, 1, GateSevereRate, WinGated, "urgent") != Pass {
		t.Fatalf("one: %s", gateStatus(pr, 1, GateSevereRate, WinGated, "urgent"))
	}
}

// Missing or contradictory records stay INVALID under the windows (the
// F3 checks): a soon study of unknown priority, a soon order resolved
// with an empty outcome; no panic in the calendar measure.
func TestSeverityWindowsInvalidRecords(t *testing.T) {
	p, _ := candidatePolicy(t)
	for name, change := range map[string]func(r *Record){
		"unknown soon study priority": func(r *Record) {
			for i := range r.Studies {
				if r.Studies[i].Priority == "soon" && r.Studies[i].AcquiredSec > t0 {
					r.Studies[i].Priority = ""
					return
				}
			}
		},
		"empty soon outcome": func(r *Record) {
			for i, o := range r.Orders {
				if o.Priority == "soon" && o.ResolvedSec > t0 {
					for k := range r.Captures {
						if r.Captures[k].AtSec >= o.ResolvedSec {
							r.Captures[k].OrderOutcomes[i] = ""
						}
					}
					return
				}
			}
		},
	} {
		b := base("B", 1)
		r := b.record()
		change(&r)
		if m := Measure(p, &r); len(m.Integrity) == 0 || m.Clinical != nil {
			t.Errorf("%s: valid %v", name, m.Integrity)
		}
	}
}

// The policy's windows validate: explicit for every priority; the
// calendar partition only; at most 13 weeks; a disabled priority has no
// rate or minimum, an enabled one both.
func TestSeverityWindowValidation(t *testing.T) {
	p, _ := candidatePolicy(t)
	if err := p.validate(); err != nil || p.Clinical.weeklyWeeks("urgent") != 0 || p.Clinical.weeklyWeeks("soon") != 4 || p.Clinical.weeklyWeeks("routine") != 1 {
		t.Fatalf("%v %v", err, p.Clinical.WeeklyWindowWeeks)
	}
	for name, change := range map[string]func(c *Clinical){
		"pooled with windows": func(c *Clinical) { c.WeeklyPartition = "" },
		"missing priority":    func(c *Clinical) { c.WeeklyWindowWeeks = map[string]int64{"soon": 4, "routine": 1} },
		"too long":            func(c *Clinical) { c.WeeklyWindowWeeks = map[string]int64{"urgent": 0, "soon": 14, "routine": 1} },
		"negative":            func(c *Clinical) { c.WeeklyWindowWeeks = map[string]int64{"urgent": -1, "soon": 4, "routine": 1} },
		"disabled with a rate": func(c *Clinical) {
			c.MaxWeeklySevereRate = map[string]float64{"urgent": 0.05, "soon": 0.05, "routine": 0.05}
		},
		"enabled without minimum": func(c *Clinical) { c.MinWeeklyCohort = map[string]int64{"routine": 100} },
	} {
		q := *p
		c := *p.Clinical
		change(&c)
		q.Clinical = &c
		if err := q.validate(); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

// Old v5 policies: the 5C.4e candidate's exact bytes load and keep
// calendar weeks for every priority (its urgent weekly gate present and,
// on the fixture, insufficient); its JSON has no window fields.
func TestOld5C4ECompatibility(t *testing.T) {
	_, digest, err := LoadPolicy(candidate5c4ePath)
	if err != nil || digest != "802cca4fd8e9aa33d9f1aaf6d80f495a9b941b04a1fbb1775fd4239694560907" {
		t.Fatalf("%v %s", err, digest)
	}
	p, d := candidate5c4ePolicy(t)
	pr, _ := clinicalCaseAB(t, p, d, nil, nil)
	ws := pr.PerSeed[0].Measures.Clinical.WeeklySevere
	if ws["urgent"] == nil || ws["soon"].WindowWeeks != 0 || gateStatus(pr, 1, GateSevereWeek, WinGated, "urgent") != Insufficient {
		t.Fatalf("%+v", ws)
	}
	raw, _ := json.Marshal(ws)
	if strings.Contains(string(raw), `"windowWeeks"`) || strings.Contains(string(raw), `"weeks":4`) {
		t.Fatalf("window fields in a 5C.4e result: %s", raw)
	}
}
