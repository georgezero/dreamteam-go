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

// The LLM agent (agents/llm, V0.5a, V0.5b) against a deterministic mock
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
	last  string // the last message of the last request
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
//	mixed       raise's allocation and a 30-minute shift extension in one
//	            reply (a staffing and an operations batch)
//	details     first asks for details, then decides as raise
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
	m.last = req.Messages[len(req.Messages)-1].Content
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
	case "mixed":
		content = mixedDecision(prompt)
	case "details":
		content = `{"reason":"I need the allocation details","actions":[],"details":["set_reading_allocation","finance.cashHistory","worklist.recent"]}`
		if strings.HasPrefix(req.Messages[len(req.Messages)-1].Content, "Details you asked for:") {
			content = raiseDecision(prompt)
		}
	}
	b, _ := json.Marshal(map[string]any{
		"id": "mock", "object": "chat.completion", "model": req.Model,
		"choices": []any{map[string]any{"index": 0, "message": map[string]any{"role": "assistant", "content": content}, "finish_reason": "stop"}},
		"usage":   map[string]any{"prompt_tokens": len(prompt) / 4, "completion_tokens": len(content) / 4, "total_tokens": len(prompt)/4 + len(content)/4},
	})
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

// promptTargets decodes an action type's targets from the prompt's
// guidance line "Targets (type): <JSON>".
func promptTargets(prompt, typ string, v any) bool {
	marker := "Targets (" + typ + "): "
	i := strings.Index(prompt, marker)
	if i < 0 {
		return false
	}
	return json.NewDecoder(strings.NewReader(prompt[i+len(marker):])).Decode(v) == nil
}

// raiseAction reads the allocation targets in the prompt and raises the
// lowest editable site reading allocation with room and no pending change
// ("" when there is none).
func raiseAction(prompt string) string {
	type version struct {
		Pending bool `json:"pending"`
	}
	var targets []struct {
		Employee          string    `json:"employee"`
		InEffectBP        int       `json:"inEffectBP"`
		MaxBP             int       `json:"maxBP"`
		Editable          bool      `json:"editable"`
		EarliestEffective string    `json:"earliestEffective"`
		Versions          []version `json:"versions"`
	}
	if !promptTargets(prompt, "set_reading_allocation", &targets) {
		return ""
	}
	best := -1
	for k, a := range targets {
		pending := slices.ContainsFunc(a.Versions, func(v version) bool { return v.Pending })
		if a.Editable && a.InEffectBP < a.MaxBP && !pending && (best < 0 || a.InEffectBP < targets[best].InEffectBP) {
			best = k
		}
	}
	if best < 0 {
		return ""
	}
	a := targets[best]
	return fmt.Sprintf(`{"type":"set_reading_allocation","op":"set","employee":%q,"effective":%q,"siteAllocationBP":%d}`,
		a.Employee, a.EarliestEffective, min(a.MaxBP, a.InEffectBP+500))
}

func raiseDecision(prompt string) string {
	if a := raiseAction(prompt); a != "" {
		return `{"reason":"the backlog is growing","actions":[` + a + `]}`
	}
	return `{"reason":"no allocation can be raised","actions":[]}`
}

// mixedDecision is raise's action, if any, and a 30-minute extension
// today of the first employee whose shift ends before the scheduled close
// with 30 minutes' room under the cap, if any.
func mixedDecision(prompt string) string {
	var acts []string
	if a := raiseAction(prompt); a != "" {
		acts = append(acts, a)
	}
	var targets []struct {
		Employee   string `json:"employee"`
		ShiftToday string `json:"shiftToday"`
		CapAt      string `json:"capAt"`
	}
	closeAt := ""
	if i := strings.Index(prompt, "scheduled close today "); i >= 0 && len(prompt) >= i+27 {
		closeAt = prompt[i+22 : i+27]
	}
	if promptTargets(prompt, "extend_shift", &targets) {
		for _, t := range targets {
			_, end, ok := strings.Cut(t.ShiftToday, "–")
			if ok && end < closeAt && addMinutes(end, 30) <= t.CapAt && addMinutes(end, 30) <= closeAt {
				acts = append(acts, fmt.Sprintf(`{"type":"extend_shift","employee":%q,"minutes":30}`, t.Employee))
				break
			}
		}
	}
	return `{"reason":"more capacity","actions":[` + strings.Join(acts, ",") + `]}`
}

func addMinutes(hhmm string, m int) string {
	var h, mm int
	fmt.Sscanf(hhmm, "%d:%d", &h, &mm)
	t := h*60 + mm + m
	return fmt.Sprintf("%02d:%02d", t/60, t%60)
}

// llmPolicy is the agent as an external policy asking the mock for model,
// with any settings in extra added.
func llmPolicy(t *testing.T, id, model string, extra map[string]any) ExternalPolicy {
	settings := map[string]any{"schema": "dreamteam.llm-agent-settings/v1", "model": model,
		"baseUrlEnv": "DT_TEST_LLM_URL", "apiKeyEnv": "DT_TEST_LLM_KEY",
		"requestTimeoutSeconds": 1, "maxRetries": 1, "decisionBudgetSeconds": 30, "maxOutputTokens": 256,
		"pricing": map[string]any{"currency": "USD", "inputPerMillion": 2, "outputPerMillion": 8}}
	for k, v := range extra {
		settings[k] = v
	}
	b, _ := json.Marshal(settings)
	p := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := []string{"python3", "-I", "-B", "./agents/llm/llm_agent.py", "--settings", p}
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
	Point            int                 `json:"point"`
	Outcome          string              `json:"outcome"`
	Attempts         int                 `json:"attempts"`
	Retries          int                 `json:"retries"`
	Errors           []string            `json:"errors"`
	Batches          [][]json.RawMessage `json:"batches"`
	DetailsRequested []string            `json:"detailsRequested"`
	MemorySha256     string              `json:"memorySha256"`
	MemoryChars      int                 `json:"memoryChars"`
}

// agentMemory is one line of the agent's memory.jsonl.
type agentMemory struct {
	Point  int    `json:"point"`
	Sha256 string `json:"sha256"`
	Chars  int    `json:"chars"`
	Text   string `json:"text"`
}

// agentUsage is the agent's summary.json, in part.
type agentUsage struct {
	Requests int `json:"requests"`
	Retries  int `json:"retries"`
	Details  int `json:"detailRequests"`
	Tokens   struct {
		Reported struct{ Requests, Prompt, Completion, Total int } `json:"reported"`
		Unknown  struct{ Requests int }                            `json:"unknown"`
	} `json:"tokens"`
	Decisions struct{ Valid, WithActions, NoAction, Refused, Fallback int } `json:"decisions"`
	Cost      *struct {
		Estimated float64 `json:"estimated"`
		Currency  string  `json:"currency"`
	} `json:"cost"`
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
	modes := []string{"noop", "raise", "fenced", "garbage", "unknown", "badparams", "extrafield", "retry", "slow", "http500",
		"mixed", "details"}
	var ext []ExternalPolicy
	for _, mode := range modes {
		ext = append(ext, llmPolicy(t, "llm-"+mode, mode, nil))
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
		for _, f := range []string{"agent.json", "model.jsonl", "decisions.jsonl", "memory.jsonl", "summary.json", "usage.txt"} {
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
		// The memory: one per decision point, empty at the start of every
		// run, recorded with its hash, within its bound.
		mem := readJSONL[agentMemory](t, filepath.Join(ev, "memory.jsonl"))
		if len(mem) != points || !strings.Contains(mem[0].Text, "No earlier decisions in this run.") {
			t.Errorf("%s: memory %d entries, first %q", r.ID, len(mem), mem[0].Text)
		}
		for k, x := range mem {
			if x.Point != k || x.Sha256 != decs[k].MemorySha256 || x.Chars != decs[k].MemoryChars || x.Chars > 6000 ||
				!strings.Contains(x.Text, "Pending changes now") || (k > 0 && !strings.Contains(x.Text, "- point 0 (")) {
				t.Errorf("%s: memory at point %d: %+v", r.ID, k, x)
			}
		}
		// Usage: every request counted; tokens as the mock reported them;
		// the cost estimated from them at the configured prices.
		var u agentUsage
		if raw, err := os.ReadFile(filepath.Join(ev, "summary.json")); err != nil || json.Unmarshal(raw, &u) != nil {
			t.Fatalf("%s: summary.json: %v", r.ID, err)
		}
		reported, prompt, completion := 0, 0, 0
		for _, c := range calls {
			if us, ok := c["usage"].(map[string]any); ok {
				reported++
				prompt += int(us["prompt_tokens"].(float64))
				completion += int(us["completion_tokens"].(float64))
			}
		}
		if u.Requests != len(calls) || u.Tokens.Reported.Requests != reported || u.Tokens.Reported.Prompt != prompt ||
			u.Tokens.Reported.Completion != completion || u.Tokens.Unknown.Requests != len(calls)-reported ||
			u.Decisions.Valid+u.Decisions.Fallback != points || u.Cost == nil || u.Cost.Currency != "USD" ||
			u.Cost.Estimated != float64(prompt*2+completion*8)/1e6 {
			t.Errorf("%s: usage %+v (calls %d, reported %d, tokens %d/%d)", r.ID, u, len(calls), reported, prompt, completion)
		}
		switch mode {
		case "noop":
			if outcomes["no_action"] != points || len(calls) != points || r.Counts.Batches != 0 {
				t.Errorf("noop: %v, %d calls, %+v", outcomes, len(calls), r.Counts)
			}
		case "raise", "fenced", "retry", "details":
			// Point 0 raises one allocation, point 1 the other; then both
			// are pending: no action.
			if outcomes["actions"] != 2 || outcomes["no_action"] != 1 || r.Counts.Applied != 2 || r.Counts.RequestErrors != 0 {
				t.Errorf("%s: %v %+v", mode, outcomes, r.Counts)
			}
			want := points
			if mode == "retry" || mode == "details" {
				want = 2 * points
			}
			if len(calls) != want {
				t.Errorf("%s: %d model calls, want %d", mode, len(calls), want)
			}
			// The memory shows what became of point 0's decision, and the
			// pending allocation changes the catalog shows.
			if !strings.Contains(mem[1].Text, "applied: 1 accepted, 0 refused") ||
				!strings.Contains(mem[2].Text, "set_reading_allocation E") || !strings.Contains(mem[2].Text, `"pending":true`) {
				t.Errorf("%s: memory %q", mode, mem[2].Text)
			}
			if mode == "details" {
				// The full catalog entry and a path the compact observation
				// leaves out (once there is one), from the full
				// observation; not a retry.
				for k, d := range decs {
					last := calls[2*k+1]["request"].(map[string]any)["messages"].([]any)
					text := last[len(last)-1].(map[string]any)["content"].(string)
					if len(d.DetailsRequested) != 3 || d.Retries != 0 || calls[2*k+1]["turn"] != "details" ||
						!strings.Contains(text, `set_reading_allocation (catalog entry): {"type":"set_reading_allocation"`) ||
						(k > 0 && !strings.Contains(text, "worklist.recent (full observation): [")) {
						t.Errorf("details: point %d: %+v %q", k, d, text[:min(len(text), 300)])
					}
				}
				if u.Details != points || u.Retries != 0 {
					t.Errorf("details: usage %+v", u)
				}
			}
		case "mixed":
			// A staffing and an operations batch at one decision point,
			// both applied.
			two := 0
			for _, d := range decs {
				if len(d.Batches) == 2 {
					two++
				}
			}
			if two == 0 || r.Counts.Refused != 0 || outcomes["fallback"] != 0 {
				t.Errorf("mixed: %d decisions with two batches, %v %+v", two, outcomes, r.Counts)
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
			if !strings.Contains(mem[1].Text, "fallback") || !strings.Contains(mem[1].Text, "no usable decision after 2 errors") {
				t.Errorf("%s: memory %q", mode, mem[1].Text)
			}
			if mode == "badparams" && !strings.Contains(mem[1].Text, "Validation refusals met in this run") {
				t.Errorf("badparams: memory %q", mem[1].Text)
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
// live run's simulator outcome exactly, and the agent's memory and prompt
// rebuilt from the run are the recorded ones; a replay that does not belong
// to the run, a recording whose memory differs and one of another memory
// format fail.
func TestLLMAgentReplay(t *testing.T) {
	m := llmSetup(t)
	c := externalConfig(t, llmPolicy(t, "llm", "mixed", nil))
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
	calls := m.count("mixed")

	// Both recordings are given; each run picks its own by its digests.
	replay := func(args ...string) ExternalPolicy {
		return ExternalPolicy{ID: "llm", Version: "1", Description: "the LLM agent, replayed",
			Command: append([]string{"python3", "-I", "-B", "./agents/llm/llm_agent.py"}, args...), TimeoutSeconds: 60}
	}
	c2 := externalConfig(t, replay(recordings...))
	c2.Seeds, c2.Evaluation.T90 = c.Seeds, false
	a, b := runMatrix(t, c2), runMatrix(t, c2)
	if m.count("mixed") != calls {
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
		if r.Counts.Applied != l.Counts.Applied || r.Counts.Batches != l.Counts.Batches || (r.Policy.ID == "llm" && r.Counts.Applied < 3) {
			t.Errorf("%s: %+v, live %+v", r.ID, r.Counts, l.Counts)
		}
		if r.Policy.ID == "llm" {
			// The memory rebuilt in replay is the live run's, point by point.
			lm := readJSONL[agentMemory](t, filepath.Join(live, "runs", r.ID, FileEvidence, "memory.jsonl"))
			rm := readJSONL[agentMemory](t, filepath.Join(a, "runs", r.ID, FileEvidence, "memory.jsonl"))
			if len(lm) != len(rm) || len(rm) == 0 {
				t.Fatalf("%s: memory %d live, %d replayed", r.ID, len(lm), len(rm))
			}
			for k := range lm {
				if lm[k] != rm[k] {
					t.Errorf("%s: memory at point %d differs", r.ID, k)
				}
			}
		}
	}

	// A recording of another run, and tampered ones, are refused.
	one := recordings[1]
	tamper := func(point int, f func(d map[string]any)) string {
		p := filepath.Join(t.TempDir(), "decisions.jsonl")
		raw, _ := os.ReadFile(one)
		lines := bytes.Split(bytes.TrimSpace(raw), []byte("\n"))
		var d map[string]any
		json.Unmarshal(lines[point], &d)
		f(d)
		lines[point], _ = json.Marshal(d)
		os.WriteFile(p, append(bytes.Join(lines, []byte("\n")), '\n'), 0o644)
		return p
	}
	digests := tamper(1, func(d map[string]any) { d["digests"].(map[string]any)["state"] = strings.Repeat("0", 64) })
	memory := tamper(2, func(d map[string]any) { d["memorySha256"] = strings.Repeat("0", 64) })
	format := tamper(0, func(d map[string]any) { d["memoryFormat"] = "llm-agent-memory/v0" })
	for _, tc := range []struct {
		args []string
		seed int64
		want string
	}{
		{[]string{"--replay", recordings[3]}, privateSeed, "no recording begins"},
		{[]string{"--replay", digests}, privateSeed, "decision point 1: the digests differ"},
		{[]string{"--replay", memory}, privateSeed, "decision point 2: the memory rebuilt from this run differs from the recording's"},
		{[]string{"--replay", format}, privateSeed, "version incompatible: decision point 0 was recorded with memory format llm-agent-memory/v0"},
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

// The memory stays within its bound over 90 daily decision points, with
// actions at many of them; the run completes and verifies.
func TestLLMAgentMemoryBounded(t *testing.T) {
	if testing.Short() {
		t.Skip("90 simulated days")
	}
	llmSetup(t)
	c := externalConfig(t, llmPolicy(t, "llm", "mixed",
		map[string]any{"memoryDecisions": 3, "memoryMetricPoints": 4, "memoryMaxChars": 1500}))
	c.Days, c.Evaluation.T90 = 90, false
	dir := runMatrix(t, c)
	_, _, recs, err := LoadResults(dir)
	if err != nil {
		t.Fatal(err)
	}
	r := recs[1]
	if r.Status != StatusCompleted || r.Counts.DecisionPoints != 90 || r.Counts.Applied == 0 {
		t.Fatalf("%s: %s %+v", r.ID, r.Status, r.Counts)
	}
	mem := readJSONL[agentMemory](t, filepath.Join(dir, "runs", r.ID, FileEvidence, "memory.jsonl"))
	largest := 0
	for _, x := range mem {
		largest = max(largest, x.Chars)
		if x.Chars > 1500 || len(x.Text) == 0 || strings.Count(x.Text, "\n- point ") > 3 {
			t.Errorf("point %d: %d characters, %d decisions", x.Point, x.Chars, strings.Count(x.Text, "\n- point "))
		}
	}
	// The last memory covers the 89 decisions before it: the earliest as
	// totals, the most recent in full.
	var earlier int
	fmt.Sscanf(mem[len(mem)-1].Text[strings.Index(mem[len(mem)-1].Text, "Earlier, points 0–")+len("Earlier, points 0–"):], "%d", &earlier)
	if len(mem) != 90 || earlier < 80 || strings.Count(mem[89].Text, "\n- point ") != 88-earlier {
		t.Errorf("%d memories; last %q", len(mem), mem[len(mem)-1].Text)
	}
	t.Logf("largest memory %d characters", largest)
	for _, ck := range mustVerify(t, dir) {
		if !ck.OK {
			t.Errorf("verify: %s: %s", ck.What, ck.Note)
		}
	}
}
