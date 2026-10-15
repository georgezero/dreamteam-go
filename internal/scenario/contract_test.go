package scenario

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// V0.2 Slice 1: contracted weekly hours (V0.2_DESIGN §5.2). A scenario
// may state an hourly employee's contract; without it the contract is
// inferred from the roster at load (the smaller net rota week) and marked
// inferred. The scenario file itself is never rewritten.

func contractOf(t *testing.T, cfg *sim.Config, id string) sim.Contract {
	t.Helper()
	for _, e := range cfg.Employees {
		if e.ID == id {
			return e.Contract
		}
	}
	t.Fatalf("no employee %s", id)
	return sim.Contract{}
}

func TestContractedWeeklyMinutes(t *testing.T) {
	const h, m = simtime.Hour, simtime.Minute
	f, err := ResolveFile("established")
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := Compile(f)
	if err != nil {
		t.Fatal(err)
	}
	for id, want := range map[string]simtime.Duration{"E01": 40 * h, "E03": 40 * h, "E04": 39*h + 30*m, "E05": 40 * h, "E06": 39*h + 30*m} {
		if c := contractOf(t, cfg, id); c.Weekly != want || !c.Inferred {
			t.Errorf("%s: contract %s inferred %v, want %s inferred", id, c.Weekly, c.Inferred, want)
		}
	}
	for _, id := range []string{"E02", "E07", "E08"} {
		if c := contractOf(t, cfg, id); c != (sim.Contract{}) {
			t.Errorf("%s (salaried): contract %+v", id, c)
		}
	}

	cfg, err = compileOverlay(t, `{"employees": [{"id": "E04", "contractedWeeklyMinutes": 2400}]}`)
	if err != nil {
		t.Fatal(err)
	}
	if c := contractOf(t, cfg, "E04"); c.Weekly != 40*h || c.Inferred || c.Weights[0][simtime.Tuesday] != 8*h {
		t.Fatalf("explicit contract %+v", c)
	}

	for _, c := range []struct{ overlay, path, msg string }{
		{`{"employees": [{"id": "E04", "contractedWeeklyMinutes": 0}]}`, "employees[3].contractedWeeklyMinutes", "1–2400"},
		{`{"employees": [{"id": "E04", "contractedWeeklyMinutes": 2401}]}`, "employees[3].contractedWeeklyMinutes", "1–2400"},
		{`{"employees": [{"id": "E07", "contractedWeeklyMinutes": 2400}]}`, "employees[6].contractedWeeklyMinutes", "hourly"},
	} {
		_, err := compileOverlay(t, c.overlay)
		var ve *ValidationError
		if !errors.As(err, &ve) {
			t.Errorf("%s: want a validation error, got %v", c.overlay, err)
			continue
		}
		found := false
		for _, is := range ve.Issues {
			found = found || (is.Path == c.path && strings.Contains(is.Msg, c.msg))
		}
		if !found {
			t.Errorf("%s: issues %v, want %s: %s", c.overlay, ve.Issues, c.path, c.msg)
		}
	}
}

// The sustainable scenario's starting state (owner decision, Slice 1):
// shared XR/CT staffing at ×1.30 demand with the full-time evening MR
// technologist, the Step 0C control. It is built from existing scenario
// inputs only. The Step 0C lead candidate compiles from the same inputs;
// against the starting contracts, its part-time evenings fall short of
// Sam's and Alex's contracted hours, so reaching it by roster changes
// alone keeps paying those hours: savings need termination and hiring.
func TestSustainableStartAndLeadContracts(t *testing.T) {
	const h, m = simtime.Hour, simtime.Minute
	load := func(name string) *sim.Config {
		raw, err := os.ReadFile("../../docs/calibration/feasibility/step0c/experiments/" + name + ".json")
		if err != nil {
			t.Fatal(err)
		}
		var man struct {
			Scenario string          `json:"scenario"`
			Overlay  json.RawMessage `json:"overlay"`
		}
		if err := json.Unmarshal(raw, &man); err != nil || man.Scenario != "established" {
			t.Fatalf("%s: %v (scenario %q)", name, err, man.Scenario)
		}
		cfg, err := compileOverlay(t, string(man.Overlay))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		return cfg
	}
	weekNet := func(cfg *sim.Config, id string) [2]simtime.Duration {
		c := contractOf(t, cfg, id)
		var w [2]simtime.Duration
		for k := range c.Weights {
			for _, d := range c.Weights[k] {
				w[k] += d
			}
		}
		return w
	}
	start := load("W130-SHR-MRft-s20")
	techs := 0
	for _, e := range start.Employees {
		if e.Role == sim.Technologist {
			techs++
		}
	}
	if techs != 5 {
		t.Fatalf("starting practice: %d technologists, want 5", techs)
	}
	for id, want := range map[string]simtime.Duration{"E03": 40 * h, "E04": 39*h + 30*m, "E11": 40 * h} {
		if c := contractOf(t, start, id); c.Weekly != want {
			t.Errorf("start %s: contract %s, want %s", id, c.Weekly, want)
		}
	}
	lead := load("W130-DED-MRpt-A62-s20")
	for id, want := range map[string]simtime.Duration{"E03": 40 * h, "E04": 27*h + 30*m, "E09": 40 * h, "E11": 20 * h} {
		if w := weekNet(lead, id); w[0] != want || w[1] != want {
			t.Errorf("lead %s: rostered %v a week, want %s", id, w, want)
		}
	}
}
