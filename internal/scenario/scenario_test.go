package scenario

import (
	"encoding/json"
	"errors"
	"flag"
	"os"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/georgezero/dreamteam-go/internal/simtime"
	"github.com/georgezero/dreamteam-go/scenarios"
)

var update = flag.Bool("update", false, "rewrite golden files")

func TestBundledScenariosCompile(t *testing.T) {
	ids := List()
	if len(ids) == 0 || ids[0] != "mini" {
		t.Fatalf("List() = %v", ids)
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
