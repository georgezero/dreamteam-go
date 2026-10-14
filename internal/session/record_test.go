package session

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/scenario"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// Step 14: session records, replay and digests.

func openFile(t *testing.T, ref string, seed int64, edit func(*scenario.File)) *Session {
	t.Helper()
	f, err := scenario.ResolveFile(ref)
	if err != nil {
		t.Fatal(err)
	}
	if edit != nil {
		edit(f)
	}
	s, err := FromFile(f, seed)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// at moves s to day d (0 = the start date) at clock hh:mm.
func at(t *testing.T, s *Session, d int64, clock string) {
	t.Helper()
	c, err := simtime.ParseClock(clock)
	if err != nil {
		t.Fatal(err)
	}
	cal := s.Calendar()
	target := cal.StartOfDay(cal.DayIndex(s.w.StartTime()) + d).Add(c)
	s.w.Engine().RunUntil(target, nil)
}

// play gives a session a varied history: accepted, rejected and no-op
// actions, several in one batch, a future-day extension, across days.
func play(t *testing.T, s *Session) {
	t.Helper()
	at(t, s, 0, "13:30") // after a 06:00 or a 13:00 takeover
	date := func(d int64) string {
		cal := s.Calendar()
		return cal.Date(cal.StartOfDay(cal.DayIndex(s.w.StartTime()) + d)).String()
	}
	s.Apply(api.Action{Type: api.ActExtendShift, Employee: "E06", Minutes: 120, Date: date(2)},
		api.Action{Type: api.ActExtendShift, Employee: "E99", Minutes: 10},
		api.Action{Type: api.ActSetWalkInPolicy, Policy: api.WalkInAcceptIfGap})
	at(t, s, 0, "15:37")
	s.Apply(api.Action{Type: api.ActSetWorklistPolicy, Policy: api.WorklistFIFO})
	at(t, s, 1, "09:00")
	s.Apply(api.Action{Type: api.ActSetWalkInPolicy, Policy: api.WalkInAcceptUntil, Cutoff: "17:00"},
		api.Action{Type: api.ActExtendShift, Employee: "E05", Minutes: 300})
	at(t, s, 1, "18:45")
}

func saveTemp(t *testing.T, s *Session) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "run.session.json")
	if err := s.Save(p); err != nil {
		t.Fatal(err)
	}
	return p
}

// Save → load reproduces the run exactly, and both continue identically.
func TestSaveLoadRoundTrip(t *testing.T) {
	s := openFile(t, "mini", 4, nil)
	play(t, s)
	if !s.Unsaved() {
		t.Fatal("a played session is not marked unsaved")
	}
	p := saveTemp(t, s)
	if s.Unsaved() || s.Path() != p {
		t.Fatal("saving did not clear the unsaved mark")
	}
	l, rep, err := Load(p, ReplayOptions{})
	if err != nil {
		t.Fatalf("%v\n%s", err, rep)
	}
	r, _ := ReadRecord(p)
	var rejected, unchanged, multi int
	for _, b := range r.Batches {
		if len(b.Actions) > 1 {
			multi++
		}
		for _, x := range b.Results {
			switch {
			case !x.OK:
				rejected++
			case x.Code == api.ActionUnchanged:
				unchanged++
			}
		}
	}
	if !rep.OK || rep.Batches != 3 || rep.Actions != 6 || rejected != 2 || unchanged != 1 || multi != 2 ||
		l.Now() != s.Now() || l.Digest() != s.Digest() || l.Unsaved() {
		t.Fatalf("report %+v; rejected %d unchanged %d multi %d", rep, rejected, unchanged, multi)
	}
	// Resume: the loaded session and the original continue identically,
	// past the future-day extension (day 2).
	for _, x := range []*Session{s, l} {
		at(t, x, 3, "12:00")
		x.Apply(api.Action{Type: api.ActSetWorklistPolicy, Policy: api.WorklistOldestFirst})
		at(t, x, 4, "08:00")
	}
	if s.Digests() != l.Digests() {
		t.Fatal("the loaded session diverged after resuming")
	}
}

// The established practice with a midday (13:00) takeover, saved Thursday
// night across payroll (Friday) and remittances, loaded and continued into
// the next week, matches an uninterrupted run.
func TestReplayMiddayTakeoverAcrossPayrollAndWeek(t *testing.T) {
	edit := func(f *scenario.File) { f.StartTime = "13:00" }
	whole := openFile(t, "established", 2, edit)
	part := openFile(t, "established", 2, edit)
	if c := simtime.ClockOf(whole.w.StartTime()); c != 13*simtime.Hour {
		t.Fatalf("takeover at %s", simtime.FormatClock(c))
	}
	for _, s := range []*Session{whole, part} {
		play(t, s)
		at(t, s, 2, "23:00") // Thursday, before Friday's payroll
	}
	p := saveTemp(t, part)
	loaded, rep, err := Load(p, ReplayOptions{})
	if err != nil {
		t.Fatalf("%v\n%s", err, rep)
	}
	for _, s := range []*Session{whole, loaded} {
		at(t, s, 7, "11:00") // the following Tuesday
	}
	o := loaded.Observe(5000)
	paid, remit := false, false
	for _, e := range o.RecentLog {
		paid = paid || strings.HasPrefix(e.Text, "Payroll paid")
		remit = remit || strings.HasPrefix(e.Text, "Payment received")
	}
	if !paid || !remit || whole.Digests() != loaded.Digests() {
		t.Fatalf("payroll %v, remittances %v; digests equal %v", paid, remit, whole.Digests() == loaded.Digests())
	}
	// And the continued run saves and replays too.
	p2 := saveTemp(t, loaded)
	if _, rep, err := Load(p2, ReplayOptions{}); err != nil {
		t.Fatalf("%v\n%s", err, rep)
	}
}

// Replay cannot depend on how the clock is advanced.
func TestReplayChunkingInvariance(t *testing.T) {
	s := openFile(t, "mini", 5, nil)
	play(t, s)
	r, err := s.Record()
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range []simtime.Duration{17 * simtime.Minute, 3 * simtime.Hour, 0} {
		if _, rep, err := Replay(r, ReplayOptions{Step: step}); err != nil {
			t.Fatalf("step %s: %v\n%s", step, err, rep)
		}
	}
	if rep, err := s.Verify(); err != nil || !rep.OK {
		t.Fatalf("verify: %v %v", err, rep)
	}
}

// rewrite edits a saved record and re-signs it (as a careful editor
// would), so only the deeper checks can catch it.
func rewrite(t *testing.T, p string, edit func(*Record)) {
	t.Helper()
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	r, err := DecodeRecord(raw)
	if err != nil {
		t.Fatal(err)
	}
	edit(r)
	r.Checksum = r.checksum()
	out, _ := r.Encode()
	if err := os.WriteFile(p, out, 0o644); err != nil {
		t.Fatal(err)
	}
}

func copyFile(t *testing.T, from string) string {
	t.Helper()
	raw, err := os.ReadFile(from)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "copy.json")
	if err := os.WriteFile(p, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// Damaged, edited and incompatible files are refused with the reason;
// nothing partial is accepted.
func TestCorruptAndIncompatibleFiles(t *testing.T) {
	s := openFile(t, "mini", 6, nil)
	play(t, s)
	good := saveTemp(t, s)
	raw, _ := os.ReadFile(good)

	check := func(name string, p string, want error, text string) {
		t.Helper()
		_, rep, err := Load(p, ReplayOptions{})
		if !errors.Is(err, want) || !strings.Contains(err.Error()+repString(rep), text) {
			t.Errorf("%s: got %v (%s), want %v containing %q", name, err, repString(rep), want, text)
		}
	}
	// Truncated.
	p := copyFile(t, good)
	os.WriteFile(p, raw[:len(raw)/2], 0o644)
	check("truncated", p, ErrCorrupt, "cannot parse")
	// Empty.
	p = copyFile(t, good)
	os.WriteFile(p, nil, 0o644)
	check("empty", p, ErrCorrupt, "cannot parse")
	// A changed byte: the checksum no longer matches.
	p = copyFile(t, good)
	os.WriteFile(p, bytes.Replace(raw, []byte(`"minutes": 120`), []byte(`"minutes": 121`), 1), 0o644)
	check("edited", p, ErrCorrupt, "checksum")
	// Garbage after the record.
	p = copyFile(t, good)
	os.WriteFile(p, append(append([]byte{}, raw...), []byte("{}")...), 0o644)
	check("trailing", p, ErrCorrupt, "after the record")
	// The scenario edited and the record re-signed: the fingerprint catches it.
	p = copyFile(t, good)
	rewrite(t, p, func(r *Record) {
		f, err := scenario.Decode(r.Scenario.Resolved)
		if err != nil {
			t.Fatal(err)
		}
		f.OpeningCashCents++
		r.Scenario.Resolved, _ = scenario.Canonical(f)
	})
	check("scenario", p, ErrCorrupt, "fingerprint")
	// Another schema.
	p = copyFile(t, good)
	rewrite(t, p, func(r *Record) { r.Schema = "dreamteam.session/v2" })
	check("schema", p, ErrIncompatible, "schema")
	// Another engine: refused by default, replayable on request.
	p = copyFile(t, good)
	rewrite(t, p, func(r *Record) { r.Engine = "0.1.0-dev.1" })
	check("engine", p, ErrIncompatible, "recorded by 0.1.0-dev.1")
	if _, rep, err := Load(p, ReplayOptions{AllowEngineMismatch: true}); err != nil || !rep.OK || rep.Engine != "0.1.0-dev.1" {
		t.Errorf("engine mismatch allowed: %v %v", err, rep)
	}
	// A recorded result that the rules do not reproduce.
	p = copyFile(t, good)
	rewrite(t, p, func(r *Record) { r.Batches[0].Results[1].OK = true })
	check("result", p, ErrMismatch, "batch 1 at 2026-10-13T13:30:00, action 2")
	// A digest that does not match.
	p = copyFile(t, good)
	rewrite(t, p, func(r *Record) { r.Digests.State = strings.Repeat("0", 64) })
	check("digest", p, ErrMismatch, "state digest")
	// Batches out of order.
	p = copyFile(t, good)
	rewrite(t, p, func(r *Record) { r.Batches[0], r.Batches[1] = r.Batches[1], r.Batches[0] })
	check("order", p, ErrMismatch, "out of order")
}

func repString(r *Report) string {
	if r == nil {
		return ""
	}
	return r.String()
}

// A failed save leaves the previous file exactly as it was and no
// temporary file behind.
func TestAtomicSaveFailure(t *testing.T) {
	s := openFile(t, "mini", 7, nil)
	p := saveTemp(t, s)
	before, _ := os.ReadFile(p)
	at(t, s, 0, "12:00")
	failBeforeRename = func() error { return errors.New("disk unplugged") }
	err := s.Save(p)
	failBeforeRename = nil
	after, _ := os.ReadFile(p)
	entries, _ := os.ReadDir(filepath.Dir(p))
	if err == nil || !bytes.Equal(before, after) || len(entries) != 1 || !s.Unsaved() {
		t.Fatalf("err %v, file changed %v, %d files in the directory", err, !bytes.Equal(before, after), len(entries))
	}
	if err := s.Save(filepath.Join(t.TempDir(), "no", "such", "dir", "x.json")); err == nil {
		t.Fatal("saved into a missing directory")
	}
	if _, _, err := Load(p, ReplayOptions{}); err != nil {
		t.Fatalf("the earlier save no longer loads: %v", err)
	}
}

// Saving, recording and digesting change nothing, and a session file
// carries no hidden state: no latent draws, no future events — only the
// scenario, seed, actions, results and digests.
func TestSavingHasNoSideEffects(t *testing.T) {
	quiet := openFile(t, "mini", 8, nil)
	busy := openFile(t, "mini", 8, nil)
	dir := t.TempDir()
	for d := int64(0); d < 3; d++ {
		at(t, quiet, d, "15:00")
		at(t, busy, d, "15:00")
		if err := busy.Save(filepath.Join(dir, "x.json")); err != nil {
			t.Fatal(err)
		}
		_ = busy.Digests()
		_, _ = busy.Verify()
	}
	at(t, quiet, 4, "10:00")
	at(t, busy, 4, "10:00")
	if quiet.Digests() != busy.Digests() || quiet.Observe(100).LogTotal != busy.Observe(100).LogTotal {
		t.Fatal("saving or verifying changed the run")
	}
	raw, _ := os.ReadFile(filepath.Join(dir, "x.json"))
	for _, hidden := range []string{"draw", "outcome", "patience", "latent", "pending", "queue"} {
		if bytes.Contains(bytes.ToLower(raw), []byte(`"`+hidden)) {
			t.Errorf("the session file has a %q field", hidden)
		}
	}
}

// A session built from a bare configuration cannot be saved.
func TestUnsavableSession(t *testing.T) {
	cfg, err := scenario.Resolve("mini")
	if err != nil {
		t.Fatal(err)
	}
	if err := New(cfg, 1).Save(filepath.Join(t.TempDir(), "x.json")); !errors.Is(err, ErrUnsavable) {
		t.Fatalf("got %v", err)
	}
}

// BenchmarkSave30Days saves the established practice 30 days after the
// takeover (record, digests, atomic write).
func BenchmarkSave30Days(b *testing.B) {
	f, err := scenario.ResolveFile("established")
	if err != nil {
		b.Fatal(err)
	}
	s, err := FromFile(f, 1)
	if err != nil {
		b.Fatal(err)
	}
	s.AdvanceTo(s.Now().Add(30 * simtime.Day))
	p := filepath.Join(b.TempDir(), "x.json")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := s.Save(p); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkReplay30Days loads (replays and verifies) that save.
func BenchmarkReplay30Days(b *testing.B) {
	f, _ := scenario.ResolveFile("established")
	s, _ := FromFile(f, 1)
	s.AdvanceTo(s.Now().Add(30 * simtime.Day))
	p := filepath.Join(b.TempDir(), "x.json")
	if err := s.Save(p); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := Load(p, ReplayOptions{}); err != nil {
			b.Fatal(err)
		}
	}
}
