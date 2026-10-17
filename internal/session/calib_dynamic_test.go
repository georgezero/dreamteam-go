//go:build calibration

package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/eval"
	"github.com/georgezero/dreamteam-go/internal/scenario"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// The dynamic staffing benchmark's plans (docs/calibration/dynamic-staffing)
// as saved sessions: each plan played through a Session on the sustainable
// starting practice, saved, then replayed from the record — authoritative
// replay, whole and in chunks — to the same digests.
//
//	go test -tags calibration ./internal/session -run DynamicPlansReplay -v
//	DT_DYN_SEEDS=901,1001   the seeds (default 901)
//	DT_DYN_DAYS=180         how far to play (default 120)
//	DT_DYN_SESSIONS=dir     also keep the session files there

const dynDir = "../../docs/calibration/dynamic-staffing"

func TestDynamicPlansReplay(t *testing.T) {
	var bm struct {
		Scenario string `json:"scenario"`
	}
	readDyn(t, filepath.Join(dynDir, "benchmark.json"), &bm)
	var man struct {
		Scenario string          `json:"scenario"`
		Overlay  json.RawMessage `json:"overlay"`
	}
	readDyn(t, filepath.Join(dynDir, bm.Scenario), &man)
	f, err := scenario.ResolveFile(man.Scenario)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := scenario.Canonical(f)
	var base, over any
	json.Unmarshal(raw, &base)
	json.Unmarshal(man.Overlay, &over)
	merged, _ := json.Marshal(scenario.Merge(base, over))
	if f, err = scenario.Decode(merged); err != nil {
		t.Fatal(err)
	}
	days := int64(120)
	if v := os.Getenv("DT_DYN_DAYS"); v != "" {
		json.Unmarshal([]byte(v), &days)
	}
	seeds := []int64{901}
	if v := os.Getenv("DT_DYN_SEEDS"); v != "" {
		seeds = nil
		for _, x := range strings.Split(v, ",") {
			var n int64
			json.Unmarshal([]byte(x), &n)
			seeds = append(seeds, n)
		}
	}
	plans, _ := filepath.Glob(filepath.Join(dynDir, "plans", "*.json"))
	for _, path := range plans {
		var p struct {
			Name    string `json:"name"`
			Batches []struct {
				Time    string       `json:"time"`
				Actions []api.Action `json:"actions"`
			} `json:"batches"`
		}
		readDyn(t, path, &p)
		if err := eval.GuardRepo("../..", seeds); err != nil {
			t.Fatal(err)
		}
		for _, seed := range seeds {
			s, err := FromFile(f, seed)
			if err != nil {
				t.Fatal(err)
			}
			cal := s.Calendar()
			for _, b := range p.Batches {
				at, err := cal.ParseISO(b.Time)
				if err != nil {
					t.Fatal(err)
				}
				s.AdvanceTo(at)
				for i, res := range s.Apply(b.Actions...) {
					if !res.OK || res.Code != api.ActionAccepted {
						t.Fatalf("%s seed %d: %s %+v", p.Name, seed, res.Code, b.Actions[i])
					}
				}
			}
			s.AdvanceTo(s.w.StartTime().Add(simtime.Duration(days) * simtime.Day))
			file := filepath.Join(t.TempDir(), p.Name+".session.json")
			if dir := os.Getenv("DT_DYN_SESSIONS"); dir != "" {
				file = filepath.Join(dir, p.Name+"-seed"+jsonInt(seed)+".session.json")
			}
			if err := s.Save(file); err != nil {
				t.Fatal(err)
			}
			for _, step := range []simtime.Duration{0, 13 * simtime.Hour} {
				l, rep, err := Load(file, ReplayOptions{Step: step})
				if err != nil || !rep.OK || rep.Compatibility() != "native_exact" || l.Digests() != s.Digests() {
					t.Fatalf("%s seed %d step %s: %v %s", p.Name, seed, step, err, repString(rep))
				}
			}
			t.Logf("%s seed %d: %d batches replayed to day %d, run digest %s", p.Name, seed, len(p.Batches), days, s.Digests().Run[:16])
		}
	}
}

func jsonInt(n int64) string {
	b, _ := json.Marshal(n)
	return string(b)
}

func readDyn(t *testing.T, path string, v any) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, v); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
}
