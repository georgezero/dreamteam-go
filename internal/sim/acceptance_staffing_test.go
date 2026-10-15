//go:build acceptance

package sim_test

import (
	"encoding/json"
	"os"
	"strconv"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// V0.2 Slice 2 at scale (acceptance build tag, slow):
//
//	go test -tags acceptance ./internal/sim -run AcceptanceStaffing -v -timeout 600m
//
// DT_SEEDS shrinks the invariant run; DT_DAYS the turnover run.

// Every invariant holds after every timestamp transaction over 90 days
// for many seeds with the V0.2 reference plan (hire and terminate with
// and without a gap, a hire terminated before its start, displacement,
// severance paydays): the golden fixture's batches, at their times.
func TestAcceptanceStaffingInvariants(t *testing.T) {
	raw, err := os.ReadFile("../../testdata/golden/established-staffing.actions.json")
	if err != nil {
		t.Fatal(err)
	}
	var batches []struct {
		Time    string       `json:"time"`
		Actions []api.Action `json:"actions"`
	}
	if err := json.Unmarshal(raw, &batches); err != nil {
		t.Fatal(err)
	}
	cfg := established(t)
	for seed := int64(1); seed <= int64(envInt("DT_SEEDS", 50)); seed++ {
		t.Run(strconv.FormatInt(seed, 10), func(t *testing.T) {
			t.Parallel()
			w := sim.New(cfg, rng.SeedFromInt(seed))
			w.Engine().RunUntil(w.StartTime(), nil)
			w.Engine().CheckInvariants = true
			for _, b := range batches {
				at, err := w.Calendar().ParseISO(b.Time)
				if err != nil {
					t.Fatal(err)
				}
				if err := runTo(w, at); err != nil {
					t.Fatalf("seed %d before %s: %v", seed, b.Time, err)
				}
				w.Apply(b.Actions)
			}
			if err := runTo(w, w.StartTime().Add(90*simtime.Day)); err != nil {
				t.Fatalf("seed %d: %v", seed, err)
			}
		})
	}
}

// Long repeated turnover: a technologist and a front desk clerk hired and
// given notice every few weeks, past 64 lifetime employees (registry
// indices, slot staffing sets and metrics all grow), with the invariants
// checked daily and the run reproducible.
func TestAcceptanceStaffingTurnover(t *testing.T) {
	days := int64(envInt("DT_DAYS", 720))
	run := func() (*sim.World, int) {
		w := sim.New(established(t), rng.SeedFromInt(3))
		w.Engine().RunUntil(w.StartTime(), nil)
		cal := w.Calendar()
		xr := weekOf(shiftAt("R1", "07:00", "15:30", "11:00", "11:30"), weekdays...)
		desk := weekOf(shiftAt("desk", "07:45", "16:15", "12:00", "12:30"), weekdays...)
		date := func(d int64) string { return cal.Date(cal.StartOfDay(d)).String() }
		hired := 0
		for k := int64(0); k < days; k++ {
			day := w.DayIndex()
			var batch []api.Action
			if k%9 == 0 {
				batch = append(batch, hireOf("xr-"+strconv.FormatInt(k, 10), "technologist", []string{"XR"}, 4000, 2400, xr, date(day+21)))
			}
			if k%7 == 0 {
				batch = append(batch, hireOf("desk-"+strconv.FormatInt(k, 10), "front_desk", []string{"registration"}, 2400, 2400, desk, date(day+7)))
			}
			for _, r := range w.Apply(batch) {
				if r.OK && r.EmployeeID != "" {
					hired++
				}
			}
			// Notice for every hire that has started: last day 14 days on.
			var notice []api.Action
			for _, e := range w.Observe(0).Staff {
				if e.Hired != nil && e.Leaving == nil && e.Employment != api.EmploymentRecruiting {
					notice = append(notice, terminateOf(e.ID, date(day+14)))
				}
			}
			if len(notice) > 0 {
				for _, r := range w.Apply(notice) {
					if !r.OK {
						t.Fatalf("day %d: %+v", k, r)
					}
				}
			}
			if err := runTo(w, cal.StartOfDay(day+1).Add(6*simtime.Hour)); err != nil {
				t.Fatalf("day %d: %v", k, err)
			}
			if err := w.Check(); err != nil {
				t.Fatalf("day %d: %v", k, err)
			}
		}
		return w, hired
	}
	w, hired := run()
	if days >= 720 && (hired <= 64 || w.RegistryLen() <= 72) {
		t.Fatalf("%d hires, registry %d: not past 64 lifetime employees", hired, w.RegistryLen())
	}
	again, _ := run()
	if again.StateDigest() != w.StateDigest() || again.LedgerHash() != w.LedgerHash() {
		t.Fatal("turnover run not reproducible")
	}
	t.Logf("%d hires over %d days; registry %d; slot staff words %d", hired, days, w.RegistryLen(), w.SlotStaffWords())
}
