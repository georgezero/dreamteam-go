package sim_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// The remote reading engine (Slice 5A, V0.2_DESIGN §8.2–8.3; no action
// creates a contract yet, so the tests use TeleContractForTest, which goes
// through the engine's own contract checks).
//
// micro-tele: Mon Oct 12 start; on Tue Oct 13 the X-ray room's one
// technologist images 24 booked patients from 08:09, the last on the
// worklist at 17:45. No radiologist reads on site, so all 24 wait for
// remote coverage. S00002, S00007 and S00013 are soon, overdue by 18:00
// (target 4 h), so the default worklist order promotes them; S00004,
// S00010 and S00015 are urgent; the rest routine. Every read takes exactly
// the minutes the test sets. At 18:00 the order is S2 S4 S7 S10 S13 S15,
// then S1 S3 S5 S6 S8 S9 S11 S12 S14 S16 … S24 (teleOrder).
// micro-tele-local adds Dr. Rae (D1) reading on site Tue 08:00–20:00 at
// 100%.

var teleOrder = []int{2, 4, 7, 10, 13, 15, 1, 3, 5, 6, 8, 9, 11, 12, 14, 16, 17, 18, 19, 20, 21, 22, 23, 24}

func teleWorld(t *testing.T, fixture string, read int64) (*sim.World, simtime.Time) {
	t.Helper()
	cfg := microCfg(t, fixture)
	for k := range cfg.Catalog {
		if cfg.Catalog[k].Code != "XR_A" {
			continue
		}
		v := int64(simtime.Duration(read) * simtime.Minute)
		q, err := rng.NewQuantile([]rng.QuantilePoint{{P: 0, V: v}, {P: rng.OneMillion, V: v}})
		if err != nil {
			t.Fatal(err)
		}
		cfg.Catalog[k].ReadTime = q
	}
	w := sim.New(cfg, rng.SeedFromInt(1))
	w.Engine().CheckInvariants = true
	return w, w.Calendar().StartOfDate(cfg.Start.AddDays(1))
}

func mustTele(t *testing.T, w *sim.World, windows, scope, from, until string) string {
	t.Helper()
	id, err := w.TeleContractForTest(strings.Split(windows, ","), scope, from, until)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// clk is day plus a clock of any hours ("24:30" is 00:30 the next day).
func clk(day simtime.Time, hhmm string) simtime.Time {
	var h, m, sec int
	if _, err := fmt.Sscanf(hhmm+":00", "%d:%d:%d", &h, &m, &sec); err != nil {
		panic(err)
	}
	return day.Add(simtime.Duration(h)*simtime.Hour + simtime.Duration(m)*simtime.Minute + simtime.Duration(sec))
}

func sid(n int) string { return fmt.Sprintf("S%05d", n) }

func mustRun(t *testing.T, w *sim.World, to simtime.Time) {
	t.Helper()
	if err := runTo(w, to); err != nil {
		t.Fatal(err)
	}
}

// The two lanes: each claim holds its lane through the ten-minute handoff
// and the read, a third eligible study waits for a lane to finish (not for
// a handoff to end), lanes claim in lane order in worklist order, and
// every study is reported once, with one revenue entry and one payer
// claim. 30-minute reads: a pair every 40 minutes from 18:00, through
// midnight (claims at 00:00 belong to Tuesday's night window), the last at
// 01:20 Wednesday.
func TestRemoteLanesHoldHandoffAndRead(t *testing.T) {
	w, tue := teleWorld(t, "micro-tele.json", 30)
	mustTele(t, w, "weeknights", "all", "2026-10-13", "")
	mustRun(t, w, clk(tue, "17:00"))
	w.PushLiveNoise(clk(tue, "18:05")) // a dispatch mid-handoff: the lanes stay taken
	mustRun(t, w, clk(tue, "18:05"))
	if l := w.TeleLanes(); l[0] != [2]string{sid(2), "handoff"} || l[1] != [2]string{sid(4), "handoff"} {
		t.Fatalf("18:05: %v", l)
	}
	o := w.Observe(0).Worklist
	if o.Unread != 22 || o.Reading != 2 || o.Items[0].Radiologist != "Teleradiology lane 1" {
		t.Fatalf("18:05 worklist: %d unread, %d reading, %+v", o.Unread, o.Reading, o.Items[0])
	}
	mustRun(t, w, clk(tue, "18:10"))
	if l := w.TeleLanes(); l[0] != [2]string{sid(2), "reading"} || l[1] != [2]string{sid(4), "reading"} || w.Observe(0).Worklist.Unread != 22 {
		t.Fatalf("18:10: handoff over, the lanes still busy: %v", l)
	}
	mustRun(t, w, clk(tue, "39:00")) // Thu 15:00
	claims := w.TeleClaims()
	if len(claims) != 24 {
		t.Fatalf("%d claims", len(claims))
	}
	for k, c := range claims {
		claimed := clk(tue, "18:00").Add(simtime.Duration(k/2) * 40 * simtime.Minute)
		clock := simtime.ClockOf(claimed)
		s := studyByID(t, w, c.Study)
		if c.Study != sid(teleOrder[k]) || c.Lane != k%2 || c.Contract != "T00001" || c.Claimed != claimed ||
			c.ReadAt != claimed.Add(10*simtime.Minute) || c.Night != (clock >= 22*simtime.Hour || clock < 8*simtime.Hour) ||
			s.State != "final" || s.Final != claimed.Add(40*simtime.Minute) || s.Claimed != claimed || !s.Remote {
			t.Fatalf("claim %d: %+v, study %+v; want %s on lane %d at %s", k, c, s, sid(teleOrder[k]), k%2, w.Calendar().ISO(claimed))
		}
	}
	revenue := map[int64]int{}
	for _, e := range w.Journal() {
		if e.Kind == "revenue" {
			revenue[e.A]++
		}
	}
	payer := map[string]int{}
	for _, c := range w.Claims() {
		payer[c.Study]++
	}
	for n := 1; n <= 24; n++ {
		if revenue[int64(n-1)] != 1 || payer[sid(n)] != 1 {
			t.Fatalf("%s: %d revenue entries, %d payer claims", sid(n), revenue[int64(n-1)], payer[sid(n)])
		}
	}
	if f := w.Observe(0).Worklist.FinalTotal; f != 24 {
		t.Fatalf("%d reports final", f)
	}
}

// An urgent-only contract reads by the study's original priority: the
// overdue soon studies the worklist promotes to the top are not urgent.
func TestRemoteUrgentOnlyUsesOriginalPriority(t *testing.T) {
	w, tue := teleWorld(t, "micro-tele.json", 30)
	mustTele(t, w, "weeknights", "urgent", "2026-10-13", "")
	mustRun(t, w, clk(tue, "17:59"))
	if top := w.Observe(0).Worklist.Items[0]; top.Study != sid(2) || top.Priority != "soon" || !top.Overdue {
		t.Fatalf("worklist top at 17:59: %+v", top)
	}
	mustRun(t, w, clk(tue, "39:00"))
	var got []string
	for _, c := range w.TeleClaims() {
		got = append(got, c.Study+"@"+w.Calendar().Clock(c.Claimed))
	}
	if want := []string{"S00004@18:00", "S00010@18:00", "S00015@18:40"}; !slices.Equal(got, want) {
		t.Fatalf("claims %v, want %v", got, want)
	}
	for _, n := range []int{2, 7, 13} {
		if s := studyByID(t, w, sid(n)); s.State != "unread" {
			t.Fatalf("%s (soon, overdue) %s", s.ID, s.State)
		}
	}
}

// Coverage is the union of the selected half-open windows, intersected
// with [activation 00:00, end 00:00): exact starts in, exact ends out;
// Friday's night runs to Saturday 08:00; Saturday 08:00–14:00 and Monday
// 00:00–08:00 are never covered; Saturday's and Sunday's windows touch at
// Sunday 00:00 without a gap.
func TestRemoteCoverageWindows(t *testing.T) {
	cases := []struct {
		windows, from, until string
		in, out              []string // "Tue 13 07:59:59"-style instants, Oct 2026
	}{
		{"weeknights,saturday,sunday", "2026-10-13", "2026-10-26",
			[]string{"13 00:00:00", "13 07:59:59", "13 18:00:00", "16 18:00:00", "17 07:59:59", "17 14:00:00", "17 23:59:59",
				"18 00:00:00", "18 23:59:59", "19 18:00:00", "25 23:59:59"},
			[]string{"12 23:59:59", "13 08:00:00", "13 17:59:59", "17 08:00:00", "17 13:59:59", "19 00:00:00", "19 07:59:59",
				"26 00:00:00", "26 18:00:00"}},
		{"weeknights", "2026-10-13", "",
			[]string{"16 18:00:00", "17 07:59:59", "19 18:00:00", "20 00:00:00"},
			[]string{"17 08:00:00", "17 14:00:00", "18 12:00:00", "19 00:00:00", "19 07:59:59", "19 17:59:59"}},
		{"saturday", "2026-10-13", "",
			[]string{"17 14:00:00", "17 23:59:59"},
			[]string{"17 13:59:59", "18 00:00:00", "16 18:00:00"}},
		{"sunday", "2026-10-13", "",
			[]string{"18 00:00:00", "18 23:59:59"},
			[]string{"17 23:59:59", "19 00:00:00", "19 07:59:59"}},
		{"weeknights", "2026-10-14", "2026-10-16", // Wed 00:00 to Fri 00:00
			[]string{"14 00:00:00", "14 07:59:59", "15 23:59:59"},
			[]string{"13 23:59:59", "16 00:00:00", "16 18:00:00"}},
	}
	for _, c := range cases {
		w, _ := teleWorld(t, "micro-tele.json", 30)
		mustTele(t, w, c.windows, "all", c.from, c.until)
		instant := func(s string) simtime.Time {
			tm, err := w.Calendar().ParseISO("2026-10-" + strings.Replace(s, " ", "T", 1))
			if err != nil {
				t.Fatal(err)
			}
			return tm
		}
		for _, s := range c.in {
			if w.TeleCovering(instant(s)) != "T00001" {
				t.Errorf("%s from %s: Oct %s not covered", c.windows, c.from, s)
			}
		}
		for _, s := range c.out {
			if w.TeleCovering(instant(s)) != "" {
				t.Errorf("%s from %s: Oct %s covered", c.windows, c.from, s)
			}
		}
	}
}

// Coverage changes are wakes at exactly the edges of the union: no edge
// where windows touch (Sunday 00:00), none inside a window.
func TestRemoteCoverageEdges(t *testing.T) {
	w, _ := teleWorld(t, "micro-tele.json", 30)
	mustTele(t, w, "weeknights,saturday,sunday", "all", "2026-10-13", "")
	var edges []string
	w.SetLogSink(func(e api.LogEntry) {
		if strings.HasPrefix(e.Text, "Teleradiology coverage") {
			edges = append(edges, e.Time[8:16]+" "+strings.Fields(e.Text)[2])
		}
	})
	mustRun(t, w, w.Calendar().StartOfDate(simtime.Date{Year: 2026, Month: 10, Day: 20}))
	want := []string{"13T00:00 opens:", "13T08:00 closes", "13T18:00 opens:", "14T08:00 closes", "14T18:00 opens:",
		"15T08:00 closes", "15T18:00 opens:", "16T08:00 closes", "16T18:00 opens:", "17T08:00 closes", "17T14:00 opens:",
		"19T00:00 closes", "19T18:00 opens:"}
	if !slices.Equal(edges, want) {
		t.Fatalf("edges\n%v\nwant\n%v", edges, want)
	}
}

// A contract's end stops claims at its exclusive end and nothing else:
// with lanes free and studies waiting at Wed 00:00, nothing is claimed;
// reads claimed before finish at or after it.
func TestRemoteNoClaimAtExclusiveEnd(t *testing.T) {
	w, tue := teleWorld(t, "micro-tele.json", 30)
	mustTele(t, w, "weeknights", "all", "2026-10-13", "2026-10-14")
	mustRun(t, w, clk(tue, "39:00"))
	claims := w.TeleClaims()
	if len(claims) != 18 || claims[17].Claimed != clk(tue, "23:20") || studyByID(t, w, claims[17].Study).Final != clk(tue, "24:00") {
		t.Fatalf("%d claims, last %+v", len(claims), claims[len(claims)-1])
	}
	for _, n := range teleOrder[18:] {
		if s := studyByID(t, w, sid(n)); s.State != "unread" {
			t.Fatalf("%s %s after the contract ended", s.ID, s.State)
		}
	}
}

// Expiry during a handoff: claimed at 23:51 under a contract ending at
// midnight, the handoff runs to 00:01 and the read to 00:30, on the
// contract it was claimed under. 29-minute reads: a pair every 39 minutes.
func TestRemoteExpiryDuringHandoff(t *testing.T) {
	w, tue := teleWorld(t, "micro-tele.json", 29)
	mustTele(t, w, "weeknights", "all", "2026-10-13", "2026-10-14")
	mustRun(t, w, clk(tue, "24:00:30"))
	if l := w.TeleLanes(); l[0][1] != "handoff" || l[1][1] != "handoff" {
		t.Fatalf("00:00:30: %v", l)
	}
	mustRun(t, w, clk(tue, "39:00"))
	claims := w.TeleClaims()
	if len(claims) != 20 {
		t.Fatalf("%d claims", len(claims))
	}
	for _, c := range claims[18:] {
		if s := studyByID(t, w, c.Study); c.Claimed != clk(tue, "23:51") || c.ReadAt != clk(tue, "24:01") || c.Contract != "T00001" ||
			!c.Night || s.Final != clk(tue, "24:30") {
			t.Fatalf("%+v %+v", c, s)
		}
	}
}

// Expiry and midnight during an active read: claimed at 23:30 (handoff to
// 23:40), reading through the contract's end and the day start at Wed
// 00:00, final at 00:25. The report, its revenue and its payer claim
// belong to Wednesday. 45-minute reads: a pair every 55 minutes.
func TestRemoteReadAcrossExpiryAndMidnight(t *testing.T) {
	w, tue := teleWorld(t, "micro-tele.json", 45)
	mustTele(t, w, "weeknights", "all", "2026-10-13", "2026-10-14")
	mustRun(t, w, clk(tue, "24:00"))
	if l := w.TeleLanes(); l[0] != [2]string{sid(teleOrder[12]), "reading"} || l[1] != [2]string{sid(teleOrder[13]), "reading"} {
		t.Fatalf("Wed 00:00: %v", l)
	}
	if o := w.Observe(0); o.Date != "2026-10-14" || o.Worklist.Reading != 2 {
		t.Fatalf("Wed 00:00: %s, %d reading", o.Date, o.Worklist.Reading)
	}
	mustRun(t, w, clk(tue, "24:30"))
	if o := w.Observe(0).Worklist; o.FinalToday != 2 || o.FinalTotal != 14 {
		t.Fatalf("Wed 00:30: %d final today, %d in all", o.FinalToday, o.FinalTotal)
	}
	if l := w.TeleLanes(); l[0] != [2]string{} || l[1] != [2]string{} {
		t.Fatalf("Wed 00:30: %v", l)
	}
	for _, n := range teleOrder[12:14] {
		var rev []simtime.Time
		for _, e := range w.Journal() {
			if e.Kind == "revenue" && e.A == int64(n-1) {
				rev = append(rev, e.At)
			}
		}
		if s := studyByID(t, w, sid(n)); s.Claimed != clk(tue, "23:30") || s.Final != clk(tue, "24:25") || !slices.Equal(rev, []simtime.Time{s.Final}) {
			t.Fatalf("%s: %+v, revenue at %v", s.ID, s, rev)
		}
	}
	if n := len(w.TeleClaims()); n != 14 {
		t.Fatalf("%d claims", n)
	}
}

// A window's end: a read claimed at 07:20 finishes at 08:40, after the
// weeknight window closes at 08:00; the lanes, free at 08:40 with two
// studies waiting, claim nothing until the next window at 18:00. 70-minute
// reads: a pair every 80 minutes.
func TestRemoteReadPastWindowEnd(t *testing.T) {
	w, tue := teleWorld(t, "micro-tele.json", 70)
	mustTele(t, w, "weeknights", "all", "2026-10-13", "")
	mustRun(t, w, clk(tue, "48:00"))
	claims := w.TeleClaims()
	if len(claims) != 24 {
		t.Fatalf("%d claims", len(claims))
	}
	for k, c := range claims {
		want := clk(tue, "18:00").Add(simtime.Duration(k/2) * 80 * simtime.Minute)
		if k >= 22 {
			want = clk(tue, "42:00") // Wed 18:00
		}
		if c.Claimed != want {
			t.Fatalf("claim %d at %s, want %s", k, w.Calendar().ISO(c.Claimed), w.Calendar().ISO(want))
		}
	}
	if s := studyByID(t, w, claims[21].Study); s.Final != clk(tue, "32:40") {
		t.Fatalf("%+v", s)
	}
}

// A successor contract starts where its predecessor ends and shares the
// same two lanes: at Wed 00:00 both are still reading the predecessor's
// 23:30 claims, so the successor's first claims wait until 00:25. At no
// instant are more than two remote reads in progress.
func TestRemoteSuccessorSharesLanes(t *testing.T) {
	w, tue := teleWorld(t, "micro-tele.json", 45)
	mustTele(t, w, "weeknights", "all", "2026-10-13", "")
	for _, c := range []struct{ from, until string }{{"2026-10-14", ""}, {"2026-10-20", "2026-10-21"}} {
		if _, err := w.TeleContractForTest([]string{"weeknights"}, "all", c.from, c.until); err == nil || !strings.Contains(err.Error(), "overlaps T00001") {
			t.Fatalf("a contract overlapping an open-ended one: %v", err)
		}
	}
	if err := w.EndTeleContractForTest("T00001", "2026-10-14"); err != nil {
		t.Fatal(err)
	}
	if _, err := w.TeleContractForTest([]string{"weeknights"}, "all", "2026-10-13", ""); err == nil {
		t.Fatal("a contract starting before its predecessor's end")
	}
	mustTele(t, w, "weeknights", "all", "2026-10-14", "")
	if err := w.EndTeleContractForTest("T00001", "2026-10-15"); err == nil || !strings.Contains(err.Error(), "past T00002's start") {
		t.Fatalf("ending past the successor's start: %v", err)
	}
	mustRun(t, w, clk(tue, "39:00"))
	claims := w.TeleClaims()
	for k, c := range claims {
		want := "T00001"
		if c.Claimed >= clk(tue, "24:00") {
			want = "T00002"
		}
		if c.Contract != want {
			t.Fatalf("claim %d %+v", k, c)
		}
	}
	if c := claims[14]; c.Claimed != clk(tue, "24:25") || claims[15].Claimed != clk(tue, "24:25") {
		t.Fatalf("successor's first claims %+v %+v", c, claims[15])
	}
	for _, c := range claims {
		busy := 0
		for _, d := range claims {
			if d.Claimed <= c.Claimed && c.Claimed < studyByID(t, w, d.Study).Final {
				busy++
			}
		}
		if busy > 2 {
			t.Fatalf("%d remote reads at %s", busy, w.Calendar().ISO(c.Claimed))
		}
	}
}

// Local and remote readers never share a study: Dr. Rae (100%, 27-minute
// reads from 08:33, back to back, finishing one exactly at 18:00) claims
// first at 18:00,
// then the lanes take the next two in worklist order; every study has
// exactly one reader and one report.
func TestRemoteAndLocalOwnershipExclusive(t *testing.T) {
	w, tue := teleWorld(t, "micro-tele-local.json", 27)
	mustTele(t, w, "weeknights", "all", "2026-10-13", "")
	mustRun(t, w, clk(tue, "17:59"))
	var next []string
	for _, it := range w.Observe(0).Worklist.Items {
		if it.State == api.StudyUnread {
			next = append(next, it.Study)
		}
	}
	mustRun(t, w, clk(tue, "18:00"))
	for k, want := range []string{"Dr. Rae", "Teleradiology lane 1", "Teleradiology lane 2"} {
		if s := studyByID(t, w, next[k]); s.State != "reading" || s.By != want || s.Claimed != clk(tue, "18:00") {
			t.Fatalf("18:00, %d in order: %+v, want %s", k, s, want)
		}
	}
	mustRun(t, w, clk(tue, "39:00"))
	remote := map[string]bool{}
	for _, c := range w.TeleClaims() {
		remote[c.Study] = true
	}
	local := 0
	for _, s := range w.Studies() {
		switch {
		case s.State != "final":
			t.Fatalf("%s %s", s.ID, s.State)
		case s.Remote != remote[s.ID] || s.Remote != strings.HasPrefix(s.By, "Teleradiology"):
			t.Fatalf("%s: read by %s, remote claim %v", s.ID, s.By, remote[s.ID])
		case s.By == "Dr. Rae":
			local++
		}
	}
	if local+len(remote) != 24 || local == 0 || len(remote) == 0 {
		t.Fatalf("%d local, %d remote", local, len(remote))
	}
}

// Replay is authoritative: a session saved during a remote handoff or read
// and reloaded is the same run replayed to that instant. Rebuilt from the
// same inputs and run in other chunkings, or stopped after every few
// transactions, the run reaches the same state at the save instant (the
// lanes mid-handoff and mid-read) and the same log, ledger and state
// afterwards.
func TestRemoteReplayAndChunking(t *testing.T) {
	build := func() (*sim.World, simtime.Time) {
		w, tue := teleWorld(t, "micro-tele-local.json", 60)
		mustTele(t, w, "weeknights", "all", "2026-10-13", "2026-10-14")
		mustTele(t, w, "weeknights,saturday", "urgent", "2026-10-14", "")
		return w, tue
	}
	ref, tue := build()
	saves := []struct {
		at    string
		phase string
	}{{"23:40", "reading"}, {"23:55", "handoff"}, {"24:00:30", "reading"}}
	var want []string
	tele := ""
	for _, s := range saves {
		mustRun(t, ref, clk(tue, s.at))
		if l := ref.TeleLanes(); l[0][1] != s.phase {
			t.Fatalf("%s: lanes %v", s.at, l)
		}
		want = append(want, ref.StateDigest())
		// The remote state is in the state digest: each save point
		// differs in it (new claims; the handoff's end).
		if p := ref.StateDigestParts()["tele"]; p == "" || p == tele {
			t.Fatalf("%s: remote state digest part %q, before %q", s.at, p, tele)
		}
		tele = ref.StateDigestParts()["tele"]
	}
	end := clk(tue, "60:00")
	mustRun(t, ref, end)
	final := [3]string{ref.StateDigest(), ref.LogHash(), ref.LedgerHash()}
	for _, step := range []simtime.Duration{simtime.Minute, 13 * simtime.Minute, 7 * simtime.Hour, 0} {
		w, _ := build()
		k := 0
		for now := w.Engine().Now(); now < end; {
			next := end
			if step > 0 {
				next = min(end, now.Add(step))
			}
			if k < len(saves) && next >= clk(tue, saves[k].at) {
				next = clk(tue, saves[k].at)
			}
			n := 0
			for w.Engine().RunUntil(next, func() bool { n++; return step == 0 && n%3 == 0 }) == sim.Stopped {
			}
			now = next
			if k < len(saves) && now == clk(tue, saves[k].at) {
				if got := w.StateDigest(); got != want[k] {
					t.Fatalf("step %s: state at %s differs", step, saves[k].at)
				}
				k++
			}
		}
		if got := [3]string{w.StateDigest(), w.LogHash(), w.LedgerHash()}; got != final {
			t.Fatalf("step %s: final digests differ", step)
		}
	}
}

// Remote reading never touches technologist booking: with a contract
// covering every window, the established practice books, images and
// supervises exactly as without one, while the lanes read.
func TestRemoteBookingInvariance(t *testing.T) {
	plain, tele := atTakeover(t, `{}`, 31), atTakeover(t, `{}`, 31)
	mustTele(t, tele, "weeknights,saturday,sunday", "all", "2026-10-15", "")
	end := plain.Calendar().StartOfDay(dayOf(plain, "2026-11-05"))
	runChecked(t, plain, end)
	runChecked(t, tele, end)
	if len(plain.TeleClaims()) != 0 || slices.ContainsFunc(plain.Studies(), func(s sim.StudyInfo) bool { return s.Remote }) {
		t.Fatal("remote reading without a contract")
	}
	if n := len(tele.TeleClaims()); n < 50 {
		t.Fatalf("%d remote claims: the case does not exercise remote reading", n)
	}
	if !slices.Equal(plain.Appointments(), tele.Appointments()) {
		t.Fatal("appointments differ")
	}
	if !slices.EqualFunc(plain.Visits(), tele.Visits(), func(a, b sim.VisitInfo) bool { return fmt.Sprint(a) == fmt.Sprint(b) }) {
		t.Fatal("visits differ")
	}
	for d := dayOf(plain, "2026-10-13"); d < plain.DaysBuilt(); d++ {
		if plain.BookingView(d) != tele.BookingView(d) || !slices.Equal(plain.SupervisionOn(d), tele.SupervisionOn(d)) ||
			planText(plain.PlanOn(d)) != planText(tele.PlanOn(d)) {
			t.Fatalf("day %d: booking, supervision or plan differs", d)
		}
	}
}

// The employed radiologists' reading-time account (reading_blocks_test.go)
// still conserves borrowed time with remote lanes reading beside them.
func TestRemoteLeavesLocalReadingDebtConserved(t *testing.T) {
	w := atTakeover(t, `{}`, 41)
	mustApply(t, w, setAlloc("E07", "2026-10-27", 10_000), setAlloc("E08", "2026-10-27", 9_500))
	mustTele(t, w, "weeknights,saturday,sunday", "all", "2026-10-20", "")
	end := w.Calendar().StartOfDay(dayOf(w, "2026-11-24"))
	mustRun(t, w, end)
	acc := readingAccount(t, w, end)
	if acc.owingClaims > 0 || acc.outsideClaims > 0 || acc.inBlock+acc.repaid > acc.block || acc.borrowed == 0 {
		t.Fatalf("%+v", acc)
	}
	if n := len(w.TeleClaims()); n < 100 {
		t.Fatalf("%d remote claims", n)
	}
}

// Window-edge wakes change eligibility only: dispatches at every edge
// instant, with nothing to claim, leave the on-site run unchanged.
func TestRemoteEdgeDispatchIsNeutral(t *testing.T) {
	plain, noisy := atTakeover(t, `{}`, 32), atTakeover(t, `{}`, 32)
	cal := plain.Calendar()
	first := dayOf(plain, "2026-10-14")
	for d := first; d < first+14; d++ {
		for _, w := range []*sim.World{plain, noisy} {
			mustRun(t, w, cal.StartOfDay(d))
		}
		for _, h := range []simtime.Duration{8, 14, 18} {
			noisy.PushLiveNoise(cal.At(d, h*simtime.Hour))
		}
	}
	for _, w := range []*sim.World{plain, noisy} {
		mustRun(t, w, cal.StartOfDay(first+14))
	}
	if digestsOf(plain) != digestsOf(noisy) {
		t.Fatal("edge dispatches changed the run")
	}
}

func planText(p sim.DayPlan) string {
	h := "closed"
	if p.Hours != nil {
		h = fmt.Sprint(*p.Hours)
	}
	return fmt.Sprint(p.Date, p.Week, h, p.Shifts, p.ByEmp, p.Reading)
}
