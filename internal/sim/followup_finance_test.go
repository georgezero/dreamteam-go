package sim_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/georgezero/dreamteam-go/internal/eval"
)

// Slice 5C.4l (audit 5C.4k K6): the candidate's 45-day follow-up moves the
// end of observation from H + 31 d to H + 45 d, and with it the financial
// observation: the minimum cash and the past-due payables are taken
// through the end, the funding gap at it. This records the no-action
// practice on the development seeds to both ends and reports, seed by
// seed, every financial measure and the per-seed financial outcomes under
// the candidate's finance rules (unchanged), so that any difference is
// reported rather than silently adopted. Set DT_FIN_FOLLOWUP=<file>.
func TestFollowUpFinancialObservation(t *testing.T) {
	out := os.Getenv("DT_FIN_FOLLOWUP")
	if out == "" {
		t.Skip("set DT_FIN_FOLLOWUP=<file> for the follow-up's financial comparison")
	}
	pol4, _, err := eval.LoadPolicy("../../" + eval.PolicyPath)
	if err != nil {
		t.Fatal(err)
	}
	cand, _, err := eval.LoadPolicy("../../" + candidatePolicy)
	if err != nil {
		t.Fatal(err)
	}
	seeds, err := eval.DevelopmentSeeds(pol4)
	if err != nil {
		t.Fatal(err)
	}
	cfg := profileConfig(t)
	type fin struct {
		ObservedDays  int64 `json:"observedDaysPastHorizon"`
		MinCash       int64 `json:"minCashCents"`
		MinCashAt     int64 `json:"minCashAtSec"`
		PastDue       int64 `json:"maxPastDueCents"`
		FundingGap    int64 `json:"fundingGapCents"`
		Solvent       bool  `json:"solvent"`
		Funded        bool  `json:"funded"`
		CashAtEnd     int64 `json:"cashAtEndCents"`
		CommittedEnd  int64 `json:"committedAtEndCents"`
		PayableNowEnd int64 `json:"payableNowAtEndCents"`
	}
	measure := func(rec eval.Record) fin {
		m := eval.Measure(cand, &rec)
		l := m.Liquidity["end"]
		return fin{(rec.ObservedUntilSec - rec.HorizonEndSec) / 86400, m.MinCash, m.MinCashAt, m.PastDue, m.FundingGap,
			m.MinCash >= cand.Finance.MinCashFloorCents && m.PastDue <= cand.Finance.MaxPastDueCents, m.FundingGap <= 0,
			l.CashCents, l.Committed, l.PayableNow}
	}
	type row struct {
		Seed    int64 `json:"seed"`
		At31    fin   `json:"observed31"`
		At45    fin   `json:"observed45"`
		Changed bool  `json:"solvencyOrFundingChanged"`
	}
	var rows []row
	changed := 0
	for _, seed := range seeds {
		a, b := measure(recordBaseline(pol4, cfg, seed)), measure(recordBaseline(cand, cfg, seed))
		x := row{seed, a, b, a.Solvent != b.Solvent || a.Funded != b.Funded}
		if x.Changed {
			changed++
		}
		rows = append(rows, x)
		t.Logf("seed %d: min cash %d → %d, past due %d → %d, funding gap %d → %d, solvent %v → %v, funded %v → %v", seed,
			a.MinCash, b.MinCash, a.PastDue, b.PastDue, a.FundingGap, b.FundingGap, a.Solvent, b.Solvent, a.Funded, b.Funded)
	}
	doc := map[string]any{
		"schema": "dreamteam.followup-financial-observation/v1",
		"source": "internal/sim/followup_finance_test.go TestFollowUpFinancialObservation: the evaluation scenario's practice with no action, development seeds, recorded to the v4 policy's end of observation (H + 31 d) and to the candidate's (H + 45 d), measured under the candidate's finance rules",
		"floor":  map[string]int64{"minCashFloorCents": cand.Finance.MinCashFloorCents, "maxPastDueCents": cand.Finance.MaxPastDueCents},
		"seeds":  rows, "seedsWithChangedSolvencyOrFunding": changed,
	}
	raw, _ := json.MarshalIndent(doc, "", " ")
	if err := os.WriteFile(out, append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}
