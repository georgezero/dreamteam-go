//go:build calibration

package sim_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/api"
	"github.com/georgezero/dreamteam-go/internal/rng"
	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
	"github.com/georgezero/dreamteam-go/internal/version"
)

// Slice 5B functional validation (docs/calibration/teleradiology-5b): one
// fixed contract lifecycle on the dynamic benchmark's starting practice,
// development seeds only. It is a check that the contract works end to
// end, not a strategy search: the contract is not tuned and no result
// chooses anything.
//
//	go test -tags calibration ./internal/sim -run 'TeleFunctional$' -v
//	DT_TELE_SEEDS=901,902,903   the seeds (development set; never 1001–1020)
//	DT_TELE_OUT=file            write the results there instead
//
// The plan: at the takeover, a standard-v1 contract on every window, all
// priorities, activated 14 days later; on day 44 it is cancelled at its
// earliest end (day 74, 30 days ahead); the run goes to day 90. Each seed is paired with
// the same seed without the contract.

type teleSeed struct {
	Seed                     int64          `json:"seed"`
	Activation               string         `json:"activation"`
	End                      string         `json:"end"`
	FirstClaim               string         `json:"firstClaim"`
	LastClaim                string         `json:"lastClaim"`
	Claims                   int            `json:"claims"`
	Night                    int            `json:"nightClaims"`
	RemoteFinal              int            `json:"remoteFinal"`
	CrossedEnd               int            `json:"inFlightAtEndCompleted"`
	ClaimsAtOrAfterEnd       int            `json:"claimsAtOrAfterEnd"`
	ByModality               map[string]int `json:"claimsByModality"`
	RetainerCents            int64          `json:"retainerCents"`
	RetainerExpectedCents    int64          `json:"retainerExpectedCents"`
	FeesCents                int64          `json:"readFeesCents"`
	FeesAtCatalogCents       int64          `json:"readFeesAtCatalogCents"`
	PaidCents                int64          `json:"vendorPaidCents"`
	UnpaidAtEndCents         int64          `json:"vendorUnpaidAtEndCents"`
	CashBridgeReconciles     bool           `json:"cashBridgeReconciles"`
	InvariantsChecked        int            `json:"invariantsCheckedDays"`
	AppointmentsUnchanged    bool           `json:"appointmentsUnchanged"`
	LocalReports             int            `json:"localReports"`
	LocalReportsNoContract   int            `json:"localReportsNoContract"`
	RemoteTATMeanMin         int64          `json:"remoteTatMeanMin"`
	OperatingProfitCents     int64          `json:"operatingProfitCents"`
	OperatingProfitNoneCents int64          `json:"operatingProfitNoContractCents"`
}

func TestTeleFunctional(t *testing.T) {
	var bm dynBenchmark
	readJSON(t, filepath.Join(dynDir, "benchmark.json"), &bm)
	var m manifest
	readJSON(t, filepath.Join(dynDir, bm.Scenario), &m)
	cfg, _, fp, _ := manifestConfig(t, m)
	seeds := parseSeeds(envOr("DT_TELE_SEEDS", "901,902,903"))
	for _, s := range seeds {
		if s >= 1001 && s <= 1020 {
			t.Fatalf("seed %d is a held-out reference seed", s)
		}
	}
	type result struct {
		Engine              string     `json:"engine"`
		ScenarioManifest    string     `json:"scenarioManifest"`
		ScenarioFingerprint string     `json:"scenarioFingerprint"`
		Plan                string     `json:"plan"`
		Days                int        `json:"days"`
		PerSeed             []teleSeed `json:"perSeed"`
	}
	res := result{Engine: version.Number, ScenarioManifest: bm.Scenario, ScenarioFingerprint: fp, Days: 90,
		Plan: "standard-v1, weeknights+saturday+sunday, all priorities, effective takeover+14; cancelled on day 44 with end day 74 (its earliest)"}
	for _, seed := range seeds {
		res.PerSeed = append(res.PerSeed, teleFunctional(t, cfg, seed))
	}
	raw, _ := json.MarshalIndent(res, "", " ")
	out := envOr("DT_TELE_OUT", "../../docs/calibration/teleradiology-5b/results/functional-dev.json")
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

func teleFunctional(t *testing.T, cfg *sim.Config, seed int64) teleSeed {
	plain, w := sim.New(cfg, rng.SeedFromInt(seed)), sim.New(cfg, rng.SeedFromInt(seed))
	cal := w.Calendar()
	start := w.StartTime()
	day0 := cal.DayIndex(start)
	date := func(d int64) string { return cal.Date(cal.StartOfDay(day0 + d)).String() }
	plain.Engine().RunUntil(start, nil)
	w.Engine().RunUntil(start, nil)
	r := teleSeed{Seed: seed, Activation: date(14), End: date(74), ByModality: map[string]int{}}
	apply := func(a api.Action) {
		if res := w.Apply([]api.Action{a}); !res[0].OK {
			t.Fatalf("seed %d: %+v", seed, res[0])
		}
	}
	apply(api.Action{Type: api.ActContractTeleradiology, Op: api.TeleOpStart, RequestID: "functional", Plan: "standard-v1",
		Windows: []string{"weeknights", "saturday", "sunday"}, Scope: "all", Effective: date(14)})
	endAt := cal.StartOfDay(day0 + 90)
	for _, x := range []*sim.World{plain, w} {
		x.Engine().CheckInvariants = false
	}
	for d := int64(1); d <= 90; d++ {
		if d == 45 { // day 44, 09:00: the cancellation at its earliest end
			at := cal.StartOfDay(day0 + 44).Add(9 * simtime.Hour)
			plain.Engine().RunUntil(at, nil)
			w.Engine().RunUntil(at, nil)
			apply(api.Action{Type: api.ActContractTeleradiology, Op: api.TeleOpCancel, Contract: "T00001", EndDate: date(74)})
		}
		at := cal.StartOfDay(day0 + d)
		plain.Engine().RunUntil(at, nil)
		w.Engine().RunUntil(at, nil)
		for _, x := range []*sim.World{plain, w} {
			if err := x.Check(); err != nil {
				t.Fatalf("seed %d day %d: %v", seed, d, err)
			}
		}
		r.InvariantsChecked++
	}
	plain.Engine().RunUntil(endAt, nil)
	w.Engine().RunUntil(endAt, nil)
	end := cal.StartOfDay(day0 + 74)
	fees := w.TeleFees()
	claims := w.TeleClaims()
	price := map[string]int64{"XR": 1_200, "US": 3_000, "CT": 4_500, "MR": 6_000}
	for k, c := range claims {
		s := studyByID(t, w, c.Study)
		f := fees[k]
		r.Claims++
		if k == 0 {
			r.FirstClaim = cal.ISO(c.Claimed)
		}
		r.LastClaim = cal.ISO(c.Claimed)
		if c.Night {
			r.Night++
		}
		if c.Claimed >= end {
			r.ClaimsAtOrAfterEnd++
		}
		mod := s.Exam[:2] // catalog codes start with the modality
		r.ByModality[mod]++
		if s.State == "final" {
			r.RemoteFinal++
			want := price[mod]
			if c.Night {
				want += want / 5
			}
			r.FeesAtCatalogCents += want
			if f.Fee+f.Surcharge != want {
				t.Fatalf("seed %d: %s at %d, catalog %d", seed, c.Study, f.Fee+f.Surcharge, want)
			}
			if s.Final > end {
				r.CrossedEnd++
			}
		}
	}
	r.RetainerCents, r.FeesCents, r.PaidCents = w.TeleTotals()
	for d := int64(14); d < 74; d++ {
		x := cal.Date(cal.StartOfDay(day0 + d))
		for _, m := range []int64{150_000, 80_000, 80_000} {
			r.RetainerExpectedCents += accrual(m, x.Day, x.DaysInMonth())
		}
	}
	o, op := w.Observe(0), plain.Observe(0)
	v := o.Teleradiology
	r.UnpaidAtEndCents, r.RemoteTATMeanMin, r.LocalReports = v.UnpaidCents, v.RemoteTATMeanMin, v.LocalReports
	r.CashBridgeReconciles = w.Evaluation().Money.Reconciled
	r.AppointmentsUnchanged = slices.Equal(plain.Appointments(), w.Appointments())
	r.OperatingProfitCents = o.Finance.SinceStart.OperatingProfitCents
	r.OperatingProfitNoneCents = op.Finance.SinceStart.OperatingProfitCents
	start0 := cal.StartOfDay(day0)
	for _, s := range plain.Studies() {
		if s.State == "final" && s.Final >= start0 {
			r.LocalReportsNoContract++
		}
	}
	switch {
	case r.Claims == 0 || r.RemoteFinal != r.Claims:
		t.Fatalf("seed %d: %d claims, %d final", seed, r.Claims, r.RemoteFinal)
	case r.ClaimsAtOrAfterEnd != 0:
		t.Fatalf("seed %d: claims after the end", seed)
	case r.RetainerCents != r.RetainerExpectedCents || r.FeesCents != r.FeesAtCatalogCents:
		t.Fatalf("seed %d: retainer %d (expected %d), fees %d (catalog %d)", seed, r.RetainerCents, r.RetainerExpectedCents, r.FeesCents, r.FeesAtCatalogCents)
	case r.PaidCents != r.RetainerCents+r.FeesCents || r.UnpaidAtEndCents != 0:
		t.Fatalf("seed %d: paid %d of %d, unpaid %d", seed, r.PaidCents, r.RetainerCents+r.FeesCents, r.UnpaidAtEndCents)
	case !r.CashBridgeReconciles || !r.AppointmentsUnchanged:
		t.Fatalf("seed %d: cash bridge %v, appointments unchanged %v", seed, r.CashBridgeReconciles, r.AppointmentsUnchanged)
	}
	t.Logf("seed %d: %d claims (%d night) %v, %d crossed the end; retainer $%.2f, fees $%.2f, paid $%.2f; local reports %d (no contract %d); remote TAT %dm; profit $%.0f vs $%.0f",
		seed, r.Claims, r.Night, r.ByModality, r.CrossedEnd, float64(r.RetainerCents)/100, float64(r.FeesCents)/100, float64(r.PaidCents)/100,
		r.LocalReports, r.LocalReportsNoContract, r.RemoteTATMeanMin, float64(r.OperatingProfitCents)/100, float64(r.OperatingProfitNoneCents)/100)
	return r
}
