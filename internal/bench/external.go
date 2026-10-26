package bench

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// PolicyProtocol is the external policy protocol (V0.4b): JSON lines over
// a policy process's standard input and output (docs/BENCHMARK.md §4).
//
// The harness writes a turn to the policy: {"protocol": PolicyProtocol,
// "turn": "decide", "point": N} at each decision point, and {"protocol":
// PolicyProtocol, "turn": "end"} after the last one, before closing its
// input. During a turn the policy writes agent/v1 requests, one per line,
// each answered by one response line, in order, through the gate; it ends
// the turn with {"decision": {"batches": [[action, …], …]}} (no batches:
// no change). After "end" it exits with status 0.
const PolicyProtocol = "dreamteam.benchmark-policy/v1"

// Limits on what an external policy writes.
const (
	maxPolicyLine    = 1 << 20 // bytes in one line
	maxBatches       = 64      // batches in one decision
	maxStderr        = 1 << 20 // bytes of standard error kept
	FileStderr       = "stderr.txt"
	policyWorkPrefix = "dreamteam-policy-"
)

// The statuses of an external policy's failures (a built-in policy's
// error or panic is StatusPolicy, as is a decision that cannot be read).
const (
	StatusTimeout = "policy_timeout" // no decision, or no exit after the last one, within the timeout
	StatusCrash   = "policy_crash"   // the process could not start, ended early, or exited with an error
)

// policyFailure is a policy's failure and its status.
type policyFailure struct {
	class string
	msg   string
}

func (f *policyFailure) Error() string { return f.msg }

// prepare resolves the configuration's policies before anything runs: an
// external policy's program must be found, and each file its command
// names must be readable (its SHA-256 goes in the manifest).
func (c *Config) prepare() ([]PolicySpec, error) {
	var out []PolicySpec
	for _, id := range c.Policies {
		spec, _ := c.policy(id)
		if x := spec.External; x != nil {
			x.Files = []FileHash{}
			for i, a := range x.Command {
				switch {
				case strings.HasPrefix(a, "./") || strings.HasPrefix(a, "../"):
					abs, err := filepath.Abs(a)
					if err != nil {
						return nil, err
					}
					raw, err := os.ReadFile(abs)
					if err != nil {
						return nil, fmt.Errorf("external policy %s: %w", id, err)
					}
					sum := sha256.Sum256(raw)
					x.Files = append(x.Files, FileHash{a, hex.EncodeToString(sum[:])})
					a = abs
				case i == 0:
					p, err := exec.LookPath(a)
					if err != nil {
						return nil, fmt.Errorf("external policy %s: %w", id, err)
					}
					a = p
				}
				x.argv = append(x.argv, a)
			}
		}
		out = append(out, spec)
	}
	return out, nil
}

// line is one line the policy wrote, or why there are no more.
type line struct {
	b   []byte
	err error
}

// external is one run's policy process. Its working directory is a fresh
// empty directory outside the results, removed after the run; it inherits
// the harness's environment, and nothing about the run (not the seed, the
// run's ID or a results path) is in its arguments, environment or
// directory name.
type external struct {
	spec    PolicySpec
	gate    *gate
	tw      io.Writer
	timeout time.Duration

	cmd     *exec.Cmd
	work    string
	stdin   *os.File
	lines   chan line
	exited  chan struct{}
	stderr  capped
	started bool
	done    bool
	n       int // requests
}

func newExternal(spec PolicySpec, g *gate, tw io.Writer) *external {
	return &external{spec: spec, gate: g, tw: tw, timeout: time.Duration(spec.External.TimeoutSeconds) * time.Second}
}

func (x *external) requests() int { return x.n }

// start launches the process with pipes of its own (not exec's copying
// goroutines), so that its output is read to the end whenever it exits.
func (x *external) start() error {
	work, err := os.MkdirTemp("", policyWorkPrefix)
	if err != nil {
		return err
	}
	x.work = work
	inR, inW, err := os.Pipe()
	if err != nil {
		return err
	}
	outR, outW, err := os.Pipe()
	if err != nil {
		inR.Close()
		inW.Close()
		return err
	}
	argv := x.spec.External.argv
	x.cmd = exec.Command(argv[0], argv[1:]...)
	x.cmd.Dir, x.cmd.Env = work, os.Environ()
	x.cmd.Stdin, x.cmd.Stdout, x.cmd.Stderr = inR, outW, &x.stderr
	x.cmd.WaitDelay = 5 * time.Second
	err = x.cmd.Start()
	inR.Close()
	outW.Close()
	if err != nil {
		inW.Close()
		outR.Close()
		return err
	}
	x.started, x.stdin = true, inW
	x.lines, x.exited = make(chan line, 1), make(chan struct{})
	go func() {
		defer close(x.lines)
		defer outR.Close()
		r := bufio.NewReaderSize(outR, 64<<10)
		for {
			b, err := readLine(r)
			if err != nil {
				if err != io.EOF {
					x.lines <- line{err: err}
				}
				return
			}
			x.lines <- line{b: b}
		}
	}()
	go func() {
		x.cmd.Wait()
		close(x.exited)
	}()
	return nil
}

// readLine reads one line, without its end, of at most maxPolicyLine bytes.
func readLine(r *bufio.Reader) ([]byte, error) {
	var b []byte
	for {
		chunk, err := r.ReadSlice('\n')
		b = append(b, chunk...)
		if len(b) > maxPolicyLine {
			return nil, fmt.Errorf("a line longer than %d bytes", maxPolicyLine)
		}
		switch {
		case err == bufio.ErrBufferFull:
			continue
		case err == io.EOF && len(b) > 0:
			return b, nil // a last line without its end
		case err != nil:
			return nil, err
		}
		return b, nil
	}
}

// turn writes a turn line.
func turn(kind string, point int) []byte {
	m := map[string]any{"protocol": PolicyProtocol, "turn": kind}
	if kind == "decide" {
		m["point"] = point
	}
	b, _ := json.Marshal(m)
	return b
}

// decide plays one decision point: the turn, then the policy's requests
// (each through the gate, into the transcript) until its decision, all
// within the timeout.
func (x *external) decide(point int) ([][]json.RawMessage, error) {
	if !x.started {
		if err := x.start(); err != nil {
			x.done = true
			return nil, &policyFailure{StatusCrash, fmt.Sprintf("the policy process could not start: %v", err)}
		}
	}
	deadline := time.NewTimer(x.timeout)
	defer deadline.Stop()
	if err := x.send(turn("decide", point), deadline.C); err != nil {
		return nil, x.failed(point, err)
	}
	for {
		select {
		case l, ok := <-x.lines:
			if !ok {
				return nil, x.failed(point, errEnded)
			}
			if l.err != nil {
				return nil, x.failed(point, &policyFailure{StatusPolicy, fmt.Sprintf("decision point %d: the policy wrote %v", point, l.err)})
			}
			b := bytes.TrimSpace(l.b)
			if len(b) == 0 {
				continue
			}
			if bs, ok, err := decision(b); ok {
				if err != nil {
					return nil, x.failed(point, &policyFailure{StatusPolicy, fmt.Sprintf("decision point %d: the decision cannot be read: %v", point, err)})
				}
				return bs, nil
			}
			x.n++
			resp := x.gate.Handle(b)
			fmt.Fprintf(x.tw, "%s\n%s\n", b, resp)
			if err := x.send(resp, deadline.C); err != nil {
				return nil, x.failed(point, err)
			}
		case <-deadline.C:
			return nil, x.failed(point, errTimeout)
		}
	}
}

var (
	errEnded   = errors.New("ended")
	errTimeout = errors.New("timeout")
)

// send writes one line to the policy, within the deadline.
func (x *external) send(b []byte, deadline <-chan time.Time) error {
	done := make(chan error, 1)
	go func() {
		_, err := x.stdin.Write(append(b, '\n'))
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			return errEnded
		}
		return nil
	case <-deadline:
		return errTimeout
	}
}

// failed stops the process and classifies the failure.
func (x *external) failed(point int, err error) error {
	if err == errEnded { // let it finish exiting, for its exit status
		select {
		case <-x.exited:
		case <-time.After(5 * time.Second):
		}
	}
	x.stop()
	switch err {
	case errTimeout:
		return &policyFailure{StatusTimeout, fmt.Sprintf("no decision within %d s at decision point %d", x.spec.External.TimeoutSeconds, point)}
	case errEnded:
		return &policyFailure{StatusCrash, fmt.Sprintf("the policy process ended at decision point %d (%s)", point, x.cmd.ProcessState)}
	}
	return err
}

// decision reads a decision line: ok when the line is an object with a
// "decision" member; then err when it is not a decision as specified.
func decision(b []byte) (bs [][]json.RawMessage, ok bool, err error) {
	var probe map[string]json.RawMessage
	if json.Unmarshal(b, &probe) != nil {
		return nil, false, nil
	}
	if _, ok := probe["decision"]; !ok {
		return nil, false, nil
	}
	var d struct {
		Decision struct {
			Batches *[][]json.RawMessage `json:"batches"`
		} `json:"decision"`
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&d); err != nil {
		return nil, true, err
	}
	switch {
	case d.Decision.Batches == nil:
		return nil, true, errors.New("decision.batches is required (an empty list: no change)")
	case len(*d.Decision.Batches) > maxBatches:
		return nil, true, fmt.Errorf("%d batches; at most %d", len(*d.Decision.Batches), maxBatches)
	}
	return *d.Decision.Batches, true, nil
}

// stop kills the process (if it still runs) and waits for it.
func (x *external) stop() {
	if !x.started || x.done {
		return
	}
	x.done = true
	x.stdin.Close()
	x.cmd.Process.Kill()
	<-x.exited
	go drain(x.lines)
}

func drain(c chan line) {
	for range c {
	}
}

// finish ends the process after the run: after a completed run, the end
// turn, then the exit (status 0) within the timeout; otherwise it is
// stopped. Its standard error is kept in dir, and its working directory
// removed.
func (x *external) finish(completed bool, dir string) error {
	var fail error
	if x.started && !x.done && completed {
		x.done = true
		deadline := time.NewTimer(x.timeout)
		defer deadline.Stop()
		err := x.send(turn("end", 0), deadline.C)
		x.stdin.Close()
		go drain(x.lines)
		if err == nil {
			select {
			case <-x.exited:
				if !x.cmd.ProcessState.Success() {
					fail = &policyFailure{StatusCrash, fmt.Sprintf("the policy process exited after the last decision with %s", x.cmd.ProcessState)}
				}
			case <-deadline.C:
				err = errTimeout
			}
		}
		if err != nil {
			x.cmd.Process.Kill()
			<-x.exited
			fail = &policyFailure{StatusTimeout, fmt.Sprintf("the policy process did not exit within %d s after the last decision", x.spec.External.TimeoutSeconds)}
			if err == errEnded {
				fail = &policyFailure{StatusCrash, fmt.Sprintf("the policy process ended before the end of the run (%s)", x.cmd.ProcessState)}
			}
		}
	}
	x.stop()
	werr := os.WriteFile(filepath.Join(dir, FileStderr), x.stderr.bytes(), 0o644)
	if x.work != "" {
		werr = errors.Join(werr, os.RemoveAll(x.work))
	}
	if werr != nil {
		return werr
	}
	return fail
}

// capped keeps the first maxStderr bytes written to it.
type capped struct {
	mu   sync.Mutex
	b    []byte
	lost bool
}

func (c *capped) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if room := maxStderr - len(c.b); room < len(p) {
		c.b, c.lost = append(c.b, p[:max(room, 0)]...), true
	} else {
		c.b = append(c.b, p...)
	}
	return len(p), nil
}

func (c *capped) bytes() []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lost {
		return append(c.b, fmt.Sprintf("\n[standard error truncated at %d bytes]\n", maxStderr)...)
	}
	return c.b
}
