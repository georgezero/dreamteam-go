package bench

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/georgezero/dreamteam-go/internal/agent"
	"github.com/georgezero/dreamteam-go/internal/session"
)

// privateSeed is a seed whose decimal digits appear nowhere else: a
// policy that saw them would have been shown the seed.
const privateSeed = 739182645

// helperArgs runs this test binary as an external policy (TestHelperPolicy)
// in the given mode.
func helperArgs(mode string) []string {
	return []string{os.Args[0], "-test.run=^TestHelperPolicy$", "--", mode}
}

func helperPolicy(id, mode string, timeout int) ExternalPolicy {
	return ExternalPolicy{ID: id, Version: "test", Description: "test helper " + mode, Command: helperArgs(mode), TimeoutSeconds: timeout}
}

// TestHelperPolicy is not a test: run by the tests as a subprocess, it is
// an external policy speaking the policy protocol. Its mode:
//
//	ok        checks hello (no seed), its arguments, environment and
//	          working directory (no seed), tries the harness's operations
//	          (denied) and a line that is not JSON (an error response); at
//	          point 0 submits a change, then nothing; exits 0 after "end".
//	crash     exits with status 2 at point 1.
//	hang      stops answering at point 1.
//	malformed answers point 1 with a decision that cannot be read.
//	linger    does not exit after "end".
//	fail      exits with status 1 after "end".
func TestHelperPolicy(t *testing.T) {
	i := slices.Index(os.Args, "--")
	if i < 0 || i+1 >= len(os.Args) {
		t.Skip("run by the external policy tests")
	}
	os.Exit(helper(os.Args[i+1]))
}

func helper(mode string) int {
	in := bufio.NewScanner(os.Stdin)
	in.Buffer(nil, 64<<20)
	out := bufio.NewWriter(os.Stdout)
	n := 0
	call := func(line string) map[string]any {
		fmt.Fprintln(out, line)
		out.Flush()
		if !in.Scan() {
			os.Exit(5)
		}
		var r map[string]any
		json.Unmarshal(in.Bytes(), &r)
		return r
	}
	req := func(op string, args any) map[string]any {
		n++
		b, _ := json.Marshal(map[string]any{"id": fmt.Sprintf("x%d", n), "op": op, "args": args})
		return call(string(b))
	}
	seed := strconv.Itoa(privateSeed)
	wd, _ := os.Getwd()
	if strings.Contains(strings.Join(os.Args, " "), seed) || strings.Contains(strings.Join(os.Environ(), " "), seed) || strings.Contains(wd, seed) {
		return 3
	}
	for in.Scan() {
		var t struct {
			Protocol string `json:"protocol"`
			Turn     string `json:"turn"`
			Point    int    `json:"point"`
		}
		if json.Unmarshal(in.Bytes(), &t) != nil || t.Protocol != PolicyProtocol {
			return 4
		}
		if t.Turn == "end" {
			switch mode {
			case "linger":
				time.Sleep(time.Hour)
			case "fail":
				return 1
			}
			return 0
		}
		decision := `{"decision":{"batches":[]}}`
		switch {
		case t.Point == 0:
			hello := req("hello", nil)
			res, _ := hello["result"].(map[string]any)
			if _, ok := res["seed"]; ok || res["benchmark"] == nil {
				return 6
			}
			req("observe", map[string]any{"compact": true})
			for _, op := range []string{"apply", "advance", "save", "t90"} {
				if r := req(op, map[string]any{}); r["ok"] != false {
					return 7
				}
			}
			call("this is not JSON")
			decision = `{"decision":{"batches":[[{"type":"set_worklist_policy","policy":"fifo"}]]}}`
		case t.Point == 1 && mode == "crash":
			return 2
		case t.Point == 1 && mode == "hang":
			time.Sleep(time.Hour)
		case t.Point == 1 && mode == "malformed":
			decision = `{"decision":{"batch":[]}}`
		}
		fmt.Fprintln(out, decision)
		out.Flush()
	}
	return 0
}

func externalConfig(t *testing.T, ext ...ExternalPolicy) *Config {
	t.Setenv("TMPDIR", t.TempDir()) // where the policies' working directories go
	c := testConfig("noop")
	c.Seeds = []int64{privateSeed}
	for _, x := range ext {
		c.Policies = append(c.Policies, x.ID)
	}
	c.External = ext
	return c
}

// An external policy process plays through the gate like a built-in one:
// its runs are deterministic and verify; it cannot reach the harness's
// operations; what it sends that is not a request is answered as the
// protocol answers it; its working directory is removed.
func TestExternalPolicy(t *testing.T) {
	c := externalConfig(t, helperPolicy("ext", "ok", 60))
	a, b := runMatrix(t, c), runMatrix(t, c)
	for _, f := range []string{FileManifest, FileReportJSON, FileReportText} {
		x, _ := os.ReadFile(filepath.Join(a, f))
		y, _ := os.ReadFile(filepath.Join(b, f))
		if !bytes.Equal(x, y) {
			t.Errorf("%s differs between identical runs", f)
		}
	}
	m, _, recs, err := LoadResults(a)
	if err != nil {
		t.Fatal(err)
	}
	ext := recs[1]
	want := Counts{DecisionPoints: 3, Batches: 1, Actions: 1, Applied: 1, PolicyRequests: 7, Denied: 4, RequestErrors: 1}
	if ext.Status != StatusCompleted || ext.Counts != want || !ext.Replay.Verified || !ext.ActionsMatch {
		t.Fatalf("%s: %s %+v %+v", ext.ID, ext.Status, ext.Counts, ext.Failure)
	}
	if p := m.Policies[1]; p.External == nil || p.External.TimeoutSeconds != 60 || !slices.Equal(p.External.Command, helperArgs("ok")) {
		t.Errorf("manifest policy %+v", p)
	}
	if !slices.Contains(m.Nondeterministic, "runs/"+ext.ID+"/"+FileStderr) {
		t.Errorf("nondeterministic %v", m.Nondeterministic)
	}
	for _, ck := range mustVerify(t, a) {
		if !ck.OK {
			t.Errorf("verify: %s: %s", ck.What, ck.Note)
		}
	}
	if left, _ := filepath.Glob(filepath.Join(os.Getenv("TMPDIR"), policyWorkPrefix+"*")); len(left) > 0 {
		t.Errorf("working directories left: %v", left)
	}
}

func mustVerify(t *testing.T, dir string) []Check {
	t.Helper()
	checks, err := Verify(dir, VerifyOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return checks
}

// Acceptance (V0.4b): a failing external policy is classified (crash,
// timeout, a decision that cannot be read, no exit, a failed exit), its
// process is stopped, its run's files verify as far as they go, and the
// other runs are unaffected.
func TestExternalPolicyFailures(t *testing.T) {
	c := externalConfig(t,
		helperPolicy("crash", "crash", 60),
		helperPolicy("hang", "hang", 2),
		helperPolicy("malformed", "malformed", 60),
		helperPolicy("linger", "linger", 2),
		helperPolicy("fail", "fail", 60),
		ExternalPolicy{ID: "gone", Version: "test", Description: "a program that is not there", Command: []string{"./no-such-policy"}, TimeoutSeconds: 5})
	if _, err := c.prepare(); err == nil || !strings.Contains(err.Error(), "no-such-policy") {
		t.Fatalf("a missing policy file: %v", err)
	}
	c.External = c.External[:5]
	c.Policies = c.Policies[:6]
	alone := runMatrix(t, externalConfig(t))
	dir := runMatrix(t, c)
	_, _, recs, err := LoadResults(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]struct {
		status, msg string
		point       int
	}{
		"noop":      {StatusCompleted, "", 0},
		"crash":     {StatusCrash, "ended at decision point 1 (exit status 2)", 1},
		"hang":      {StatusTimeout, "no decision within 2 s at decision point 1", 1},
		"malformed": {StatusPolicy, "decision point 1: the decision cannot be read", 1},
		"linger":    {StatusTimeout, "did not exit within 2 s after the last decision", 3},
		"fail":      {StatusCrash, "exited after the last decision with exit status 1", 3},
	}
	for _, r := range recs {
		w := want[r.Policy.ID]
		got := ""
		if r.Failure != nil {
			got = r.Failure.Message
			if r.Failure.Point != w.point {
				t.Errorf("%s: decision point %d", r.ID, r.Failure.Point)
			}
		}
		if r.Status != w.status || !strings.Contains(got, w.msg) || !r.Replay.Verified || !r.ActionsMatch {
			t.Errorf("%s: %s %q, replay %+v", r.ID, r.Status, got, r.Replay)
		}
	}
	for _, ck := range mustVerify(t, dir) {
		if !ck.OK {
			t.Errorf("verify: %s: %s", ck.What, ck.Note)
		}
	}
	id := runID("mini", privateSeed, "noop")
	for _, f := range runFiles {
		x, err := os.ReadFile(filepath.Join(alone, "runs", id, f))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if y, _ := os.ReadFile(filepath.Join(dir, "runs", id, f)); !bytes.Equal(x, y) {
			t.Errorf("%s of the noop run differs beside failing policies", f)
		}
	}
	if left, _ := filepath.Glob(filepath.Join(os.Getenv("TMPDIR"), policyWorkPrefix+"*")); len(left) > 0 {
		t.Errorf("working directories left: %v", left)
	}
}

// Seed privacy (V0.4b): nothing a policy is answered carries the seed,
// whichever permitted operation it calls and however it fails; hello says
// the seed is withheld. Outside the benchmark, hello still reports it.
func TestSeedIsWithheld(t *testing.T) {
	probe := func(c *agent.Client, point int) ([][]json.RawMessage, error) {
		if point > 0 {
			return nil, nil
		}
		bad := []json.RawMessage{json.RawMessage(`{"type":"set_reading_allocation","op":"remove","employee":"nobody","effective":"2099-01-01"}`)}
		for _, x := range []struct {
			op   string
			args any
		}{
			{"hello", nil}, {"observe", map[string]any{"recentLog": 2000}}, {"observe", map[string]any{"compact": true}},
			{"observe", map[string]any{"sections": []string{"nowhere"}}}, {"sections", nil}, {"actions", nil},
			{"validate", map[string]any{"actions": bad}}, {"preview", map[string]any{"actions": bad}},
			{"validate", map[string]any{"actions": []string{}}}, {"digests", nil}, {"digests", map[string]any{"x": 1}},
			{"t90", nil}, {"save", map[string]any{"path": "x.json"}},
		} {
			c.Call(x.op, x.args, nil)
		}
		return nil, nil
	}
	withPolicies(t, spec("test-probe", probe))
	c := externalConfig(t, helperPolicy("ext", "ok", 60))
	c.Scenarios = []string{"mini", "sustainable"}
	c.Days = 1
	c.Policies = []string{"test-probe", "ext"}
	c.Baseline = "test-probe"
	dir := runMatrix(t, c)
	seed := []byte(strconv.Itoa(privateSeed))
	seen := 0
	for _, scen := range c.Scenarios {
		for _, p := range c.Policies {
			raw, err := os.ReadFile(filepath.Join(dir, "runs", runID(scen, privateSeed, p), FileTranscript))
			if err != nil {
				t.Fatal(err)
			}
			lines := bytes.Split(bytes.TrimSpace(raw), []byte("\n"))
			for i := 0; i+1 < len(lines); i += 2 {
				var req struct {
					ID string `json:"id"`
					Op string `json:"op"`
				}
				json.Unmarshal(lines[i], &req)
				if strings.HasPrefix(req.ID, "h") {
					continue // the harness's own request
				}
				seen++
				if bytes.Contains(lines[i+1], seed) {
					t.Errorf("%s %s: the response to %s carries the seed", scen, p, lines[i])
				}
				if req.Op == "hello" && (bytes.Contains(lines[i+1], []byte(`"seed"`)) || !bytes.Contains(lines[i+1], []byte(`"seedWithheld":true`))) {
					t.Errorf("hello: %s", lines[i+1])
				}
			}
		}
	}
	if seen < 2*(13+7) {
		t.Fatalf("%d policy responses", seen)
	}
	// The seed is kept where the harness keeps its evidence.
	var rec RunRecord
	if err := readJSON(filepath.Join(dir, "runs", runID("mini", privateSeed, "ext"), FileRun), &rec); err != nil || rec.Seed != privateSeed {
		t.Fatalf("run.json seed %d: %v", rec.Seed, err)
	}
	// An ordinary agent session still reports it.
	h := agent.New(mustOpen(t, "mini", privateSeed))
	if out := h.Handle([]byte(`{"id":1,"op":"hello"}`)); !bytes.Contains(out, []byte(`"seed":`+string(seed))) || bytes.Contains(out, []byte("benchmark")) {
		t.Fatalf("hello outside the benchmark: %s", out)
	}
}

// The reference Python policy (clients/python/reference_policy.py) plays
// as an external process: it completes, its run verifies, and two runs are
// identical. Skipped without python3.
func TestPythonReferencePolicy(t *testing.T) {
	if testing.Short() {
		t.Skip("runs the sustainable scenario four times")
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not found")
	}
	wd, _ := os.Getwd()
	t.Chdir(filepath.Join(wd, "..", ".."))
	py := ExternalPolicy{ID: "python-ref", Version: "1", Description: "the reference external policy",
		Command: []string{"python3", "-I", "-B", "./clients/python/reference_policy.py"}, TimeoutSeconds: 120}
	c := externalConfig(t, py)
	c.Scenarios, c.Seeds, c.Days, c.Evaluation.T90 = []string{"sustainable"}, []int64{77}, 8, false
	a, b := runMatrix(t, c), runMatrix(t, c)
	x, _ := os.ReadFile(filepath.Join(a, FileManifest))
	y, _ := os.ReadFile(filepath.Join(b, FileManifest))
	if !bytes.Equal(x, y) {
		t.Fatal("two runs of the Python policy differ")
	}
	m, _, recs, err := LoadResults(a)
	if err != nil {
		t.Fatal(err)
	}
	r := recs[1]
	if r.Status != StatusCompleted || r.Counts.DecisionPoints != 8 || r.Counts.PolicyRequests == 0 || r.Counts.RequestErrors+r.Counts.Denied != 0 {
		t.Fatalf("%s: %s %+v %+v", r.ID, r.Status, r.Counts, r.Failure)
	}
	if f := m.Policies[1].External.Files; len(f) != 1 || f[0].Path != "./clients/python/reference_policy.py" || len(f[0].SHA256) != 64 {
		t.Fatalf("policy files %+v", f)
	}
	for _, ck := range mustVerify(t, a) {
		if !ck.OK {
			t.Errorf("verify: %s: %s", ck.What, ck.Note)
		}
	}
	if stderr, _ := os.ReadFile(filepath.Join(a, "runs", r.ID, FileStderr)); len(stderr) != 0 {
		t.Errorf("stderr: %s", stderr)
	}
}

func mustOpen(t *testing.T, scen string, seed int64) *session.Session {
	t.Helper()
	s, err := session.Open(scen, seed)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
