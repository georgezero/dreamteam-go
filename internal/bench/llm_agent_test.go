package bench

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// The LLM agent (agents/llm, V0.5a) against a deterministic mock
// OpenAI-compatible server: the agent runs as an external policy, and the
// mock answers by the model name it is asked for.

// llmSecret stands for an API key: it must reach the server and nothing
// else.
const llmSecret = "sk-dreamteam-test-0f3c9a1e7b"

// mockLLM is the mock server: per model, the requests it received.
type mockLLM struct {
	mu    sync.Mutex
	calls map[string]int
	auth  []string
}

func newMockLLM(t *testing.T) (*mockLLM, *httptest.Server) {
	m := &mockLLM{calls: map[string]int{}}
	srv := httptest.NewServer(http.HandlerFunc(m.serve))
	t.Cleanup(srv.Close)
	return m, srv
}

func (m *mockLLM) count(model string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls[model]
}

// serve answers a chat completion. The model names the behaviour:
//
//	noop        no action
//	raise       raises the lowest editable site reading allocation by 5
//	            points from its earliest date (from the prompt's catalog)
//	fenced      raise's decision inside a Markdown code fence
//	garbage     prose, not JSON
//	unknown     an action type that does not exist
//	badparams   an allocation out of its band from a fixed date (refused by
//	            validate)
//	extrafield  an action with a field its schema does not have
//	retry       garbage on the first attempt, raise's decision on the retry
//	slow        answers after 3 s (the agent's request timeout is 1 s)
//	http500     HTTP 500
func (m *mockLLM) serve(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	if r.URL.Path != "/v1/chat/completions" || json.NewDecoder(r.Body).Decode(&req) != nil || len(req.Messages) < 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	m.mu.Lock()
	m.calls[req.Model]++
	m.auth = append(m.auth, r.Header.Get("Authorization"))
	m.mu.Unlock()
	prompt := req.Messages[1].Content
	content := `{"reason":"nothing needs to change","actions":[]}`
	switch req.Model {
	case "raise":
		content = raiseDecision(prompt)
	case "fenced":
		content = "```json\n" + raiseDecision(prompt) + "\n```"
	case "garbage":
		content = "I would hire another radiologist."
	case "unknown":
		content = `{"reason":"more capacity","actions":[{"type":"buy_scanner","rooms":1}]}`
	case "badparams":
		content = `{"reason":"cut reading","actions":[{"type":"set_reading_allocation","op":"set","employee":"E07","effective":"2026-10-27","siteAllocationBP":100}]}`
	case "extrafield":
		content = `{"reason":"walk-ins","actions":[{"type":"set_walkin_policy","policy":"accept","urgency":"high"}]}`
	case "retry":
		content = "Let me think."
		if len(req.Messages) > 2 {
			content = raiseDecision(prompt)
		}
	case "slow":
		select {
		case <-time.After(3 * time.Second):
		case <-r.Context().Done():
			return
		}
	case "http500":
		http.Error(w, `{"error":"overloaded"}`, http.StatusInternalServerError)
		return
	}
	b, _ := json.Marshal(map[string]any{
		"id": "mock", "object": "chat.completion", "model": req.Model,
		"choices": []any{map[string]any{"index": 0, "message": map[string]any{"role": "assistant", "content": content}, "finish_reason": "stop"}},
		"usage":   map[string]any{"prompt_tokens": len(prompt) / 4, "completion_tokens": len(content) / 4, "total_tokens": len(prompt)/4 + len(content)/4},
	})
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

// raiseDecision reads the catalog in the prompt and raises the lowest
// editable site reading allocation with room and no pending change.
func raiseDecision(prompt string) string {
	none := `{"reason":"no allocation can be raised","actions":[]}`
	i := strings.Index(prompt, "ACTION CATALOG")
	if i < 0 {
		return none
	}
	j := strings.Index(prompt[i:], "{")
	var cat struct {
		Types []struct {
			Type    string          `json:"type"`
			Targets json.RawMessage `json:"targets"`
		} `json:"types"`
	}
	if j < 0 || json.NewDecoder(strings.NewReader(prompt[i+j:])).Decode(&cat) != nil {
		return none
	}
	type version struct {
		Pending bool `json:"pending"`
	}
	for _, t := range cat.Types {
		var targets []struct {
			Employee          string    `json:"employee"`
			InEffectBP        int       `json:"inEffectBP"`
			MaxBP             int       `json:"maxBP"`
			Editable          bool      `json:"editable"`
			EarliestEffective string    `json:"earliestEffective"`
			Versions          []version `json:"versions"`
		}
		if t.Type != "set_reading_allocation" || json.Unmarshal(t.Targets, &targets) != nil {
			continue
		}
		best := -1
		for k, a := range targets {
			pending := slices.ContainsFunc(a.Versions, func(v version) bool { return v.Pending })
			if a.Editable && a.InEffectBP < a.MaxBP && !pending && (best < 0 || a.InEffectBP < targets[best].InEffectBP) {
				best = k
			}
		}
		if best >= 0 {
			a := targets[best]
			return fmt.Sprintf(`{"reason":"the backlog is growing","actions":[{"type":"set_reading_allocation","op":"set","employee":%q,"effective":%q,"siteAllocationBP":%d}]}`,
				a.Employee, a.EarliestEffective, min(a.MaxBP, a.InEffectBP+500))
		}
	}
	return none
}

// llmPolicy is the agent as an external policy asking the mock for model.
func llmPolicy(t *testing.T, id, model string, extra ...string) ExternalPolicy {
	settings := map[string]any{"schema": "dreamteam.llm-agent-settings/v1", "model": model,
		"baseUrlEnv": "DT_TEST_LLM_URL", "apiKeyEnv": "DT_TEST_LLM_KEY",
		"requestTimeoutSeconds": 1, "maxRetries": 1, "decisionBudgetSeconds": 30, "maxOutputTokens": 256}
	b, _ := json.Marshal(settings)
	p := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := append([]string{"python3", "-I", "-B", "./agents/llm/llm_agent.py", "--settings", p}, extra...)
	return ExternalPolicy{ID: id, Version: "1", Description: "the LLM agent, mock " + model, Command: cmd, TimeoutSeconds: 60}
}

// llmSetup: python3, the repository root as the working directory, and the
// mock's address and the stand-in key in the environment.
func llmSetup(t *testing.T) *mockLLM {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not found")
	}
	wd, _ := os.Getwd()
	t.Chdir(filepath.Join(wd, "..", ".."))
	m, srv := newMockLLM(t)
	t.Setenv("DT_TEST_LLM_URL", srv.URL+"/v1")
	t.Setenv("DT_TEST_LLM_KEY", llmSecret)
	return m
}

// agentDecision is one line of the agent's decisions.jsonl.
type agentDecision struct {
	Point    int                 `json:"point"`
	Outcome  string              `json:"outcome"`
	Attempts int                 `json:"attempts"`
	Errors   []string            `json:"errors"`
	Batches  [][]json.RawMessage `json:"batches"`
}

func readJSONL[T any](t *testing.T, path string) []T {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var out []T
	for _, l := range bytes.Split(bytes.TrimSpace(raw), []byte("\n")) {
		var v T
		if err := json.Unmarshal(l, &v); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		out = append(out, v)
	}
	return out
}

// policyOps is the set of operations a run's policy asked for (its
// request ids are its own, the harness's begin with "h").
func policyOps(t *testing.T, transcript string) map[string]int {
	t.Helper()
	raw, err := os.ReadFile(transcript)
	if err != nil {
		t.Fatal(err)
	}
	ops := map[string]int{}
	for _, l := range bytes.Split(bytes.TrimSpace(raw), []byte("\n")) {
		var req struct {
			ID any    `json:"id"`
			Op string `json:"op"`
		}
		if json.Unmarshal(l, &req) == nil && req.Op != "" {
			if id, _ := req.ID.(string); !strings.HasPrefix(id, "h") {
				ops[req.Op]++
			}
		}
	}
	return ops
}

func TestLLMAgentMock(t *testing.T) {
	m := llmSetup(t)
	modes := []string{"noop", "raise", "fenced", "garbage", "unknown", "badparams", "extrafield", "retry", "slow", "http500"}
	var ext []ExternalPolicy
	for _, mode := range modes {
		ext = append(ext, llmPolicy(t, "llm-"+mode, mode))
	}
	c := externalConfig(t, ext...)
	c.Evaluation.T90 = false
	dir := runMatrix(t, c)
	man, _, recs, err := LoadResults(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, ck := range mustVerify(t, dir) {
		if !ck.OK {
			t.Errorf("verify: %s: %s", ck.What, ck.Note)
		}
	}
	const points = 3
	for i, mode := range modes {
		r := recs[i+1]
		rd := filepath.Join(dir, "runs", r.ID)
		if r.Status != StatusCompleted || r.Counts.DecisionPoints != points || r.Counts.Denied != 0 || r.Counts.Refused != 0 {
			t.Errorf("%s: %s %+v %+v", r.ID, r.Status, r.Counts, r.Failure)
			continue
		}
		// Only the permitted operations, and never the harness's.
		for op := range policyOps(t, filepath.Join(rd, FileTranscript)) {
			if !slices.Contains([]string{"hello", "digests", "observe", "actions", "validate"}, op) {
				t.Errorf("%s: the agent asked for %s", r.ID, op)
			}
		}
		// The evidence, kept and listed, not hashed.
		ev := filepath.Join(rd, FileEvidence)
		for _, f := range []string{"agent.json", "model.jsonl", "decisions.jsonl", "summary.json"} {
			rel := filepath.ToSlash(filepath.Join("runs", r.ID, FileEvidence, f))
			if !fileExists(filepath.Join(ev, f)) || !slices.Contains(man.Nondeterministic, rel) {
				t.Errorf("%s: evidence %s missing or not listed", r.ID, f)
			}
		}
		decs := readJSONL[agentDecision](t, filepath.Join(ev, "decisions.jsonl"))
		calls := readJSONL[map[string]any](t, filepath.Join(ev, "model.jsonl"))
		if len(decs) != points || m.count(mode) != len(calls) {
			t.Errorf("%s: %d decisions, %d calls recorded, %d received", r.ID, len(decs), len(calls), m.count(mode))
		}
		outcomes := map[string]int{}
		for _, d := range decs {
			outcomes[d.Outcome]++
		}
		switch mode {
		case "noop":
			if outcomes["no_action"] != points || len(calls) != points || r.Counts.Batches != 0 {
				t.Errorf("noop: %v, %d calls, %+v", outcomes, len(calls), r.Counts)
			}
		case "raise", "fenced", "retry":
			// Point 0 raises one allocation, point 1 the other; then both
			// are pending: no action.
			if outcomes["actions"] != 2 || outcomes["no_action"] != 1 || r.Counts.Applied != 2 || r.Counts.RequestErrors != 0 {
				t.Errorf("%s: %v %+v", mode, outcomes, r.Counts)
			}
			want := points
			if mode == "retry" {
				want = 2 * points
			}
			if len(calls) != want {
				t.Errorf("%s: %d model calls, want %d", mode, len(calls), want)
			}
		default:
			// Every failure: two attempts (one retry), then no action;
			// nothing applied, nothing fabricated.
			if outcomes["fallback"] != points || len(calls) != 2*points || r.Counts.Batches != 0 {
				t.Errorf("%s: %v, %d calls, %+v", mode, outcomes, len(calls), r.Counts)
			}
			code := map[string]string{"garbage": "invalid_json", "unknown": "unknown_action", "badparams": "refused: action 0: ",
				"extrafield": "invalid_action: validate answered invalid_args", "slow": "timeout", "http500": "http_status: HTTP 500"}[mode]
			for _, d := range decs {
				if d.Attempts != 2 || len(d.Errors) != 2 || !strings.Contains(d.Errors[0], code) {
					t.Errorf("%s: point %d: %d attempts, %v", mode, d.Point, d.Attempts, d.Errors)
				}
			}
			if mode == "extrafield" && r.Counts.RequestErrors != 2*points {
				t.Errorf("extrafield: %d request errors", r.Counts.RequestErrors)
			}
		}
		if stderr, _ := os.ReadFile(filepath.Join(rd, FileStderr)); len(stderr) != 0 {
			t.Errorf("%s: stderr: %s", r.ID, stderr)
		}
	}
	// The key reached the server on every request, and nothing else.
	for _, a := range m.auth {
		if a != "Bearer "+llmSecret {
			t.Fatalf("authorization %q", a)
		}
	}
	filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			if b, _ := os.ReadFile(p); bytes.Contains(b, []byte(llmSecret)) {
				t.Errorf("%s holds the API key", p)
			}
		}
		return nil
	})
}

// A recorded decision sequence, replayed without the model, gives the
// live run's simulator outcome exactly; a replay that does not belong to
// the run fails.
func TestLLMAgentReplay(t *testing.T) {
	m := llmSetup(t)
	c := externalConfig(t, llmPolicy(t, "llm", "raise"))
	c.Seeds, c.Evaluation.T90 = []int64{privateSeed, 5}, false
	live := runMatrix(t, c)
	_, _, liveRecs, err := LoadResults(live)
	if err != nil {
		t.Fatal(err)
	}
	var recordings []string
	for _, r := range liveRecs {
		if r.Policy.ID == "llm" {
			recordings = append(recordings, "--replay", filepath.Join(live, "runs", r.ID, FileEvidence, "decisions.jsonl"))
		}
	}
	calls := m.count("raise")

	// Both recordings are given; each run picks its own by its digests.
	replay := func(args ...string) ExternalPolicy {
		return ExternalPolicy{ID: "llm", Version: "1", Description: "the LLM agent, replayed",
			Command: append([]string{"python3", "-I", "-B", "./agents/llm/llm_agent.py"}, args...), TimeoutSeconds: 60}
	}
	c2 := externalConfig(t, replay(recordings...))
	c2.Seeds, c2.Evaluation.T90 = c.Seeds, false
	a, b := runMatrix(t, c2), runMatrix(t, c2)
	if m.count("raise") != calls {
		t.Fatal("the replay called the model")
	}
	x, _ := os.ReadFile(filepath.Join(a, FileManifest))
	y, _ := os.ReadFile(filepath.Join(b, FileManifest))
	if !bytes.Equal(x, y) {
		t.Fatal("two replays differ")
	}
	_, _, recs, err := LoadResults(a)
	if err != nil {
		t.Fatal(err)
	}
	for i, r := range recs {
		l := liveRecs[i]
		if r.ID != l.ID || r.Status != StatusCompleted || r.Final == nil || l.Final == nil || r.Final.Digests != l.Final.Digests {
			t.Fatalf("%s: %s, final digests differ from the live run's", r.ID, r.Status)
		}
		for _, f := range []string{FileResult, FileSession, FileDecisions} {
			p, q := filepath.Join(live, "runs", r.ID, f), filepath.Join(a, "runs", r.ID, f)
			x, _ := os.ReadFile(p)
			y, _ := os.ReadFile(q)
			if !bytes.Equal(x, y) {
				t.Errorf("%s: %s differs from the live run's", r.ID, f)
			}
		}
		if r.Policy.ID == "llm" && r.Counts.Applied != 2 {
			t.Errorf("%s: %+v", r.ID, r.Counts)
		}
	}

	// A recording of another run, and a tampered one, are refused.
	one := recordings[1]
	tampered := filepath.Join(t.TempDir(), "decisions.jsonl")
	raw, _ := os.ReadFile(one)
	lines := bytes.Split(bytes.TrimSpace(raw), []byte("\n"))
	var d map[string]any
	json.Unmarshal(lines[1], &d)
	d["digests"].(map[string]any)["state"] = strings.Repeat("0", 64)
	lines[1], _ = json.Marshal(d)
	os.WriteFile(tampered, append(bytes.Join(lines, []byte("\n")), '\n'), 0o644)
	for _, tc := range []struct {
		args []string
		seed int64
		want string
	}{
		{[]string{"--replay", recordings[3]}, privateSeed, "no recording begins"},
		{[]string{"--replay", tampered}, privateSeed, "decision point 1: the digests differ"},
	} {
		c3 := externalConfig(t, replay(tc.args...))
		c3.Seeds, c3.Evaluation.T90 = []int64{tc.seed}, false
		dir := runMatrix(t, c3)
		_, _, recs, _ := LoadResults(dir)
		r := recs[1]
		stderr, _ := os.ReadFile(filepath.Join(dir, "runs", r.ID, FileStderr))
		if r.Status != StatusCrash || !strings.Contains(string(stderr), tc.want) {
			t.Errorf("%v: %s, stderr %q", tc.args, r.Status, stderr)
		}
	}
}
