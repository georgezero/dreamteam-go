package scenario

import (
	"encoding/json"
	"errors"
	"flag"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/georgezero/dreamteam-go/internal/simtime"
	"github.com/georgezero/dreamteam-go/scenarios"
)

var update = flag.Bool("update", false, "rewrite golden files")

func TestBundledScenariosCompile(t *testing.T) {
	ids := List()
	if !slices.Equal(ids, []string{"established", "mini"}) || !slices.Contains(ids, Default) {
		t.Fatalf("List() = %v, default %q", ids, Default)
	}
	for _, id := range ids {
		if _, err := Resolve(id); err != nil {
			t.Errorf("%s: %v", id, err)
		}
	}
	// The base file is not playable but must be valid on its own.
	f, err := Load(scenarios.FS, "base/established.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Compile(f); err != nil {
		t.Fatal(err)
	}
}

func TestMiniInheritsBase(t *testing.T) {
	cfg, err := Resolve("mini")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ID != "mini" || cfg.WarmupDays != 0 || cfg.Name != "DreamTeam Imaging Center" ||
		len(cfg.Rooms) != 4 || len(cfg.Employees) != 8 || cfg.Start.String() != "2026-10-13" {
		t.Fatalf("mini = %+v", cfg)
	}
}

func TestMerge(t *testing.T) {
	var b, o, want any
	must := func(s string, v *any) {
		if err := json.Unmarshal([]byte(s), v); err != nil {
			t.Fatal(err)
		}
	}
	must(`{"a": 1, "b": {"x": 1, "y": 2}, "drop": 5,
	       "list": [1, 2], "emps": [{"id": "E1", "n": 1, "k": 1}, {"id": "E2", "n": 2}]}`, &b)
	must(`{"a": 2, "b": {"y": 3, "z": null}, "drop": null,
	       "list": [3], "emps": [{"id": "E2", "n": 20}, {"id": "E3", "n": 3, "q": null}]}`, &o)
	must(`{"a": 2, "b": {"x": 1, "y": 3},
	       "list": [3], "emps": [{"id": "E1", "n": 1, "k": 1}, {"id": "E2", "n": 20}, {"id": "E3", "n": 3}]}`, &want)
	if got := Merge(b, o); !reflect.DeepEqual(got, want) {
		t.Fatalf("Merge =\n%v\nwant\n%v", got, want)
	}
}

func TestLoadExtendsAndPreset(t *testing.T) {
	base, _ := scenarios.FS.ReadFile("base/established.json")
	fsys := fstest.MapFS{
		"base/established.json": {Data: base},
		"child.json": {Data: []byte(`{"extends": "base/established.json", "id": "child", "preset": "lean",
			"employees": [{"id": "E03", "pay": {"type": "hourly", "rateCents": 3800}}]}`)},
		"presets/lean.json": {Data: []byte(`{"openingCashCents": 5000000, "policies": {"graceMin": 10}}`)},
		"loop1.json":        {Data: []byte(`{"extends": "loop2.json"}`)},
		"loop2.json":        {Data: []byte(`{"extends": "loop1.json"}`)},
		"typo.json":         {Data: []byte(`{"extends": "base/established.json", "warmupDay": 3}`)},
	}
	f, err := Load(fsys, "child.json")
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := Compile(f)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ID != "child" || cfg.OpeningCashCents != 5000000 || cfg.Policies.Grace != 600 ||
		cfg.Employees[2].RateCents != 3800 || cfg.Employees[2].Name != "Riley" || len(cfg.Employees) != 8 {
		t.Fatalf("merged config wrong: id=%s cash=%d grace=%d riley=%+v", cfg.ID, cfg.OpeningCashCents, cfg.Policies.Grace, cfg.Employees[2])
	}
	if _, err := Load(fsys, "loop1.json"); err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Errorf("cycle: %v", err)
	}
	if _, err := Load(fsys, "typo.json"); err == nil || !strings.Contains(err.Error(), "warmupDay") {
		t.Errorf("unknown field: %v", err)
	}
}

// Each invalid fixture is the base scenario with one overlay; the error
// must name the JSON path.
func TestValidationErrors(t *testing.T) {
	base, _ := scenarios.FS.ReadFile("base/established.json")
	cases := []struct{ overlay, path, msg string }{
		{`{"schema": "v0"}`, "schema", "is not"},
		{`{"evaluationDays": -1}`, "evaluationDays", "outside"},
		{`{"alerts": {"waitWarnMin": 30, "waitAlertMin": 20, "capWarnMin": 30, "closingWarnMin": 30, "closingAlertMin": 60, "clearAfterMin": 10}}`, "alerts.waitAlertMin", "below waitWarnMin"},
		{`{"alerts": {"waitWarnMin": 0, "waitAlertMin": 60, "capWarnMin": 30, "closingWarnMin": 30, "closingAlertMin": 60, "clearAfterMin": 10}}`, "alerts.waitWarnMin", "outside"},
		{`{"alerts": {"waitWarnMin": 30, "waitAlertMin": 60, "capWarnMin": 30, "closingWarnMin": 30, "closingAlertMin": 60, "clearAfterMin": 10, "lowCashCents": -1}}`, "alerts.lowCashCents", "negative"},
		{`{"startDate": "2026-02-30"}`, "startDate", "not a date"},
		{`{"hours": {"Mon": ["20:00", "08:00"]}}`, "hours.Mon", "not before"},
		{`{"hours": {"Monday": ["08:00", "20:00"]}}`, "hours.Monday", "not a weekday"},
		{`{"rooms": [{"id": "R1", "modality": "PET"}]}`, "rooms[0].modality", "not one of"},
		{`{"employees": [{"id": "E01", "roster": {"Tue": {"breaks": [["07:00", "07:30"]]}}}]}`, "employees[0].roster.Tue.breaks[0]", "strictly inside"},
		{`{"employees": [{"id": "E05", "roster": {"Tue": {"assignment": "R1"}}}]}`, "employees[4].roster.Tue.assignment", "lacks the XR skill"},
		{`{"employees": [{"id": "E03", "roster": {"Tue": {"covers": [{"room": "R4"}]}}}]}`, "employees[2].roster.Tue.covers[0].room", "lacks the MR skill"},
		{`{"employees": [{"id": "E03", "roster": {"Tue": {"covers": [{"room": "R3", "until": "17:00"}]}}}]}`, "employees[2].roster.Tue.covers[0]", "inside the shift"},
		{`{"employees": [{"id": "E04", "roster": {"Tue": {"start": "06:00", "end": "20:00"}}}]}`, "employees[3].roster.Tue", "longer than maxShiftHours"},
		{`{"employees": [{"id": "E06", "roster": {"Sat": {"weeks": "C"}}}]}`, "employees[5].roster.Sat.weeks", "not A or B"},
		{`{"employees": [{"id": "E03", "roster": {"Tue": {"location": "remote"}}}]}`, "employees[2].roster.Tue.location", "only radiologists"},
		{`{"employees": [{"id": "E07", "siteAllocationBP": 0}]}`, "employees[6].siteAllocationBP", "outside"},
		{`{"employees": [{"id": "E01", "skills": []}]}`, "employees[0].skills", "need \"registration\""},
		{`{"policies": {"worklist": "random"}}`, "policies.worklist", "not one of"},
		{`{"policies": {"contrastSupervision": "virtual"}}`, "policies.contrastSupervision", "not one of"},
		{`{"policies": {"bookingHorizonDays": 0}}`, "policies.bookingHorizonDays", "outside"},
		{`{"policies": {"overtimeFactorBP": 5000}}`, "policies.overtimeFactorBP", "outside"},
		{`{"employees": [{"id": "E08", "roster": {"Tue": {"start": "14:00", "end": "24:00"}}}]}`, "employees[7].roster.Tue", "accounting close"},
		{`{"behavior": {"registration": {"selfRegFactorBP": 1}}}`, "behavior.registration", "must be 10000–30000"},
		{`{"catalog": [{"code": "XR_X", "name": "x", "modality": "XR", "category": "XR_15", "techExtraSec": {"p0": 1, "p100": 2}}]}`, "catalog[0].roomTimeSec", "required"},
		{`{"behavior": {"arrivalClasses": [{"weightBP": 5000, "fromMin": -10, "toMin": 0}]}}`, "behavior.arrivalClasses", "sum to 5000"},
		{`{"behavior": {"arrivalClasses": [{"weightBP": 10000, "fromMin": 10, "toMin": 0}]}}`, "behavior.arrivalClasses[0]", "fromMin ≤ toMin"},
		{`{"behavior": {"noShow": {"basePPM": {"PET": 1}}}}`, "behavior.noShow.basePPM.PET", "not a modality"},
		{`{"behavior": {"noShow": {"rebookPPM": 2000000}}}`, "behavior.noShow.rebookPPM", "outside"},
		{`{"behavior": {"walkInPatienceMin": {"p0": 0}}}`, "behavior.walkInPatienceMin.p0", "below 1"},
		{`{"walkIns": {"examMix": {"MR_KNEE_WO": 5}}}`, "walkIns.examMix.MR_KNEE_WO", "not walk-in eligible"},
		{`{"walkIns": {"hourProfileBP": {"08": 0}}}`, "walkIns.hourProfileBP", "sums to 9500"},
		{`{"templates": {"R4": [{"days": ["Tue"], "slots": [{"start": "19:30", "lenMin": 45, "category": "MR_45"}]}]}}`, "templates.R4[0].slots", "outside open hours"},
		{`{"templates": {"R4": [{"days": ["Tue"], "slots": [{"start": "08:00", "lenMin": 45, "category": "MR_45"}, {"start": "08:30", "lenMin": 45, "category": "MR_45"}]}]}}`, "templates.R4[0].slots", "overlap"},
		{`{"templates": {"R4": [{"days": ["Tue"], "slots": [{"start": "08:00", "lenMin": 15, "category": "XR_15"}]}]}}`, "templates.R4[0].slots[0].category", "room R4 is MR"},
		{`{"templates": {"R4": [{"days": ["Tue"], "slots": []}, {"days": ["Tue"], "slots": []}]}}`, "templates.R4[1].days[0]", "already has"},
		{`{"templates": {"R9": []}}`, "templates.R9", "not a room"},
		{`{"referralSources": [{"id": "SRC1", "examMix": {"XR_FOOT": 5}}]}`, "referralSources[0].examMix.XR_FOOT", "not an exam code"},
		{`{"referralSources": [{"id": "SRC2", "priorityMixBP": {"routine": 5000}}]}`, "referralSources[1].priorityMixBP", "sums to 6000"},
		{`{"demand": {"hourProfileBP": {"07": 0}}}`, "demand.hourProfileBP", "sums to 9600"},
		{`{"demand": {"leadTimeDays": {"p100": null}}}`, "demand.leadTimeDays", "needs p0 and p100"},
		{`{"demand": {"weekdayFactorBP": {"Sun": null}}}`, "demand.weekdayFactorBP", "all seven"},
		{`{"behavior": {"waitToleranceDays": {"stat": {"p0": 0, "p100": 1}}}}`, "behavior.waitToleranceDays.stat", "not a priority"},
		{`{"catalog": [{"code": "X", "name": "x", "modality": "XR", "category": "MR_45"}, {"code": "Y", "name": "y", "modality": "MR", "category": "MR_45"}]}`, "catalog[1].category", "mixes modalities"},
		{`{"openingSchedule": [{"time": "19:00", "room": "R2", "exam": "US_PELVIS", "source": "SRC3"}]}`, "openingSchedule[0]", "no R2 template slot"},
		{`{"openingSchedule": [{"day": 6, "time": "09:30", "room": "R2", "exam": "US_PELVIS", "source": "SRC3"}]}`, "openingSchedule[0]", "not bookable (unstaffed)"},
		{`{"openingSchedule": [{"time": "09:30", "room": "R2", "exam": "US_THYROID", "source": "SRC3"}]}`, "openingSchedule[0]", "slot is US_45"},
		{`{"catalog": [{"code": "XR_X", "name": "x", "modality": "XR", "category": "XR_15", "roomTimeSec": {"p0": 60, "p100": 60}, "readSec": {"p0": 60, "p100": 60}, "techExtraSec": {"p0": 1, "p100": 1}}]}`, "catalog[0].baseRateCents", "outside"},
		{`{"finance": {"payers": [{"id": "medicare", "mixBP": 3000}]}}`, "finance.payers", "mixBP sums to 9500"},
		{`{"finance": {"payers": [{"id": "commercial", "deniedPPM": 990000}]}}`, "finance.payers[0]", "exceed 1000000"},
		{`{"finance": {"payers": [{"id": "self_pay", "recoveredPPM": 1}]}}`, "finance.payers[3].recoveredPPM", "not reworked"},
		{`{"finance": {"payers": [{"id": "medicaid", "delayDays": [40, 30]}]}}`, "finance.payers[2].delayDays", "increasing range"},
		{`{"finance": {"reworkDays": [2, 60]}}`, "finance.reworkDays", "within 7–365"},
		{`{"finance": {"payrollAnchor": "2026-13-01"}}`, "finance.payrollAnchor", "not a date"},
		{`{"finance": {"fixedMonthly": [{"id": "rent", "account": "marketing"}]}}`, "finance.fixedMonthly[0].account", "not one of"},
		{`{"finance": {"fixedMonthly": [{"id": "lease_ct", "modality": "PET"}]}}`, "finance.fixedMonthly[3].modality", "not a room modality"},
		{`{"finance": {"submitTime": "25:00"}}`, "finance.submitTime", ""},
		// Step 11 review finding 3: point-in-time financial clocks are before 24:00.
		{`{"finance": {"submitTime": "24:00"}}`, "finance.submitTime", "before 24:00"},
		{`{"finance": {"paymentTime": "24:00"}}`, "finance.paymentTime", "before 24:00"},
	}
	for _, tc := range cases {
		fsys := fstest.MapFS{
			"base/established.json": {Data: base},
			"bad.json":              {Data: []byte(`{"extends": "base/established.json", ` + strings.TrimPrefix(tc.overlay, "{"))},
		}
		f, err := Load(fsys, "bad.json")
		if err != nil {
			t.Errorf("%s: load: %v", tc.overlay, err)
			continue
		}
		_, err = Compile(f)
		var ve *ValidationError
		if !errors.As(err, &ve) {
			t.Errorf("%s: want validation error, got %v", tc.overlay, err)
			continue
		}
		found := false
		for _, is := range ve.Issues {
			if is.Path == tc.path && strings.Contains(is.Msg, tc.msg) {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: no %q at %s in\n%v", tc.overlay, tc.msg, tc.path, err)
		}
	}
}

// Duplicates cannot come from an id-merged overlay, so test them directly.
func TestDuplicateIDs(t *testing.T) {
	f, err := Load(scenarios.FS, "base/established.json")
	if err != nil {
		t.Fatal(err)
	}
	f.Rooms = append(f.Rooms, f.Rooms[0])
	f.Employees = append(f.Employees, f.Employees[0])
	_, err = Compile(f)
	if err == nil || !strings.Contains(err.Error(), `rooms[4].id: duplicate "R1"`) ||
		!strings.Contains(err.Error(), `employees[8].id: duplicate "E01"`) {
		t.Fatalf("got %v", err)
	}
}

// Coverage timelines for Mon, Tue and Sat in both rota weeks
// (SIMULATION_DESIGN §6.3).
func TestCoverageTimelines(t *testing.T) {
	cfg, err := Resolve("mini")
	if err != nil {
		t.Fatal(err)
	}
	days := Fortnight(cfg)
	want := map[string]string{
		"A Mon": "XR 11:30–12:00, 16:15–20:00 | US 08:00–09:30, 13:30–14:00, 18:00–20:00 | CT 11:30–12:00, 16:15–20:00 | MR 12:30–13:00, 16:30–20:00 | desk 12:00–12:30, 16:15–20:00 | sup 12h",
		"B Mon": "XR 11:30–12:00, 16:15–20:00 | US 08:00–20:00 | CT 11:30–12:00, 16:15–20:00 | MR 12:30–13:00, 16:30–20:00 | desk 12:00–12:30, 16:15–20:00 | sup 12h",
		"A Tue": "XR 11:30–12:00 | US 08:00–09:30, 13:30–14:00, 18:00–20:00 | CT 15:30–16:00 | MR 12:30–13:00, 16:30–20:00 | desk 12:00–12:30, 16:15–20:00 | sup 12h",
		"B Tue": "XR 11:30–12:00 | US 08:00–09:30, 13:30–14:00, 18:00–20:00 | CT 15:30–16:00 | MR 12:30–13:00, 16:30–20:00 | desk 12:00–12:30, 16:15–20:00 | sup 12h",
		"A Sat": "XR 13:30–14:00 | US 10:00–18:00 | CT 13:30–14:00 | MR 10:00–18:00 | desk 10:00–18:00 | sup 0h",
		"B Sat": "XR 13:30–14:00 | US 13:30–14:00 | CT 13:30–14:00 | MR 10:00–18:00 | desk 10:00–18:00 | sup 0h",
	}
	for i, d := range days {
		key := []string{"A", "B"}[i/7] + " " + d.Weekday().String()
		exp, ok := want[key]
		if !ok {
			continue
		}
		p := cfg.Plan(d)
		cov := p.Coverage()
		rooms, desk := Gaps(&cov, len(cfg.Rooms))
		var parts []string
		for r, g := range rooms {
			parts = append(parts, cfg.Rooms[r].Modality+" "+spans(g))
		}
		parts = append(parts, "desk "+spans(desk))
		var sup simtime.Duration
		for _, s := range cov.Segments {
			if s.Open && s.Supervision {
				sup += s.To - s.From
			}
		}
		parts = append(parts, "sup "+hours(sup)+"h")
		if got := strings.Join(parts, " | "); got != exp {
			t.Errorf("%s %s:\n got %s\nwant %s", key, d, got, exp)
		}
	}
}

func TestCoverageReportGolden(t *testing.T) {
	cfg, err := Resolve("mini")
	if err != nil {
		t.Fatal(err)
	}
	got := Report(cfg)
	const path = "testdata/coverage-mini.txt"
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Errorf("coverage report changed (run with -update to accept):\n%s", got)
	}
}

// A resolved scenario survives its canonical JSON: decoding it compiles to
// the same configuration, and the fingerprint is stable.
func TestCanonicalRoundTrip(t *testing.T) {
	for _, id := range []string{"established", "mini"} {
		f, err := ResolveFile(id)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := Canonical(f)
		if err != nil {
			t.Fatal(err)
		}
		g, err := Decode(raw)
		if err != nil {
			t.Fatal(err)
		}
		a, err := Compile(f)
		if err != nil {
			t.Fatal(err)
		}
		b, err := Compile(g)
		if err != nil {
			t.Fatal(err)
		}
		fa, _ := Fingerprint(f)
		fb, _ := Fingerprint(g)
		if !reflect.DeepEqual(a, b) || fa != fb || len(fa) != 64 {
			t.Fatalf("%s: canonical round trip changed the scenario", id)
		}
	}
}
