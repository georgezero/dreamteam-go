package scenario

import (
	"fmt"
	"slices"

	"github.com/georgezero/dreamteam-go/internal/sim"
	"github.com/georgezero/dreamteam-go/internal/simtime"
)

// finance validates and compiles the financial assumptions and the
// catalog's rates and supply costs (SIMULATION_DESIGN §8).
func (c *checker) finance(f *File, cfg *sim.Config) {
	fin := f.Finance
	out := &cfg.Finance
	for i, x := range f.Catalog {
		xp := fmt.Sprintf("catalog[%d]", i)
		if x.BaseRateCents <= 0 || x.BaseRateCents > 10_000_000 {
			c.fail(xp+".baseRateCents", "%d is outside 1–10000000", x.BaseRateCents)
		}
		if x.SupplyCostCents < 0 || x.SupplyCostCents > 10_000_000 {
			c.fail(xp+".supplyCostCents", "%d is outside 0–10000000", x.SupplyCostCents)
		}
	}
	if f.OpeningCashCents < 0 {
		c.fail("openingCashCents", "negative")
	}

	if len(fin.Payers) == 0 {
		c.fail("finance.payers", "at least one payer class is required")
	}
	ids := map[string]bool{}
	var mix int64
	for i, p := range fin.Payers {
		pp := fmt.Sprintf("finance.payers[%d]", i)
		if p.ID == "" {
			c.fail(pp+".id", "required")
		} else if ids[p.ID] {
			c.fail(pp+".id", "duplicate %q", p.ID)
		}
		ids[p.ID] = true
		if p.Name == "" {
			c.fail(pp+".name", "required")
		}
		if p.MixBP < 0 {
			c.fail(pp+".mixBP", "negative")
		}
		mix += p.MixBP
		if p.MultiplierBP < 1 || p.MultiplierBP > 100_000 {
			c.fail(pp+".multiplierBP", "%d is outside 1–100000", p.MultiplierBP)
		}
		c.dayRange(pp+".delayDays", p.DelayDays, 1, 365)
		pc := sim.PayerConfig{ID: p.ID, Name: p.Name, SelfPay: p.SelfPay, MultiplierBP: p.MultiplierBP,
			DelayDays: p.DelayDays, DeniedPPM: c.ppm(pp+".deniedPPM", p.DeniedPPM),
			RecoveredPPM: c.ppm(pp+".recoveredPPM", p.RecoveredPPM), ShortPPM: c.ppm(pp+".shortPaidPPM", p.ShortPaidPPM),
			ShortMeanBP: p.ShortMeanBP}
		if p.DeniedPPM+p.ShortPaidPPM > 1_000_000 {
			c.fail(pp, "deniedPPM + shortPaidPPM exceed 1000000")
		}
		if p.SelfPay && p.RecoveredPPM != 0 {
			c.fail(pp+".recoveredPPM", "self-pay balances are not reworked; must be 0")
		}
		if p.ShortMeanBP < fin.ShortSpreadBP || p.ShortMeanBP+fin.ShortSpreadBP > 10_000 {
			c.fail(pp+".shortMeanBP", "%d ± shortSpreadBP %d must stay within 0–10000", p.ShortMeanBP, fin.ShortSpreadBP)
		}
		if pc.NCRPPM() <= 0 {
			c.fail(pp, "expected collection rate is zero")
		}
		out.Payers = append(out.Payers, pc)
		out.PayerMixBP = append(out.PayerMixBP, uint64(max(0, p.MixBP)))
	}
	if len(fin.Payers) > 0 && mix != 10_000 {
		c.fail("finance.payers", "mixBP sums to %d, not 10000", mix)
	}
	c.dayRange("finance.chargeLagBusinessDays", fin.ChargeLagBusinessDays, 1, 30)
	// The rework outcome must come after the resubmission (at most three
	// calendar days after the denial: the next business day).
	c.dayRange("finance.reworkDays", fin.ReworkDays, 7, 365)
	out.ChargeLagDays, out.ReworkDays = fin.ChargeLagBusinessDays, fin.ReworkDays
	if fin.ShortSpreadBP < 0 || fin.ShortSpreadBP > 5_000 {
		c.fail("finance.shortSpreadBP", "%d is outside 0–5000", fin.ShortSpreadBP)
	}
	out.ShortSpreadBP = fin.ShortSpreadBP
	// Points in time on a business day: 24:00 would be the next day.
	out.SubmitClock = c.clockOfDay("finance.submitTime", fin.SubmitTime)
	out.PaymentClock = c.clockOfDay("finance.paymentTime", fin.PaymentTime)
	for _, x := range []struct {
		p string
		v int64
	}{{"finance.benefitsBP", fin.BenefitsBP}, {"finance.billingFeeBP", fin.BillingFeeBP}} {
		if x.v < 0 || x.v > 10_000 {
			c.fail(x.p, "%d is outside 0–10000", x.v)
		}
	}
	out.BenefitsBP, out.BillingFeeBP = fin.BenefitsBP, fin.BillingFeeBP
	if fin.PerStudyITCents < 0 {
		c.fail("finance.perStudyITCents", "negative")
	}
	out.PerStudyITCents = fin.PerStudyITCents
	out.PayrollAnchor = c.date("finance.payrollAnchor", fin.PayrollAnchor)
	if fin.PayrollEveryDays < 1 || fin.PayrollEveryDays > 31 {
		c.fail("finance.payrollEveryDays", "%d is outside 1–31", fin.PayrollEveryDays)
	}
	out.PayrollEveryDays = fin.PayrollEveryDays

	var roomMods []string
	for _, r := range cfg.Rooms {
		roomMods = append(roomMods, r.Modality)
	}
	fids := map[string]bool{}
	for i, x := range fin.FixedMonthly {
		fp := fmt.Sprintf("finance.fixedMonthly[%d]", i)
		if x.ID == "" {
			c.fail(fp+".id", "required")
		} else if fids[x.ID] {
			c.fail(fp+".id", "duplicate %q", x.ID)
		}
		fids[x.ID] = true
		if x.Name == "" {
			c.fail(fp+".name", "required")
		}
		c.oneOf(fp+".account", x.Account, sim.FixedAccounts)
		if x.Modality != "" && !slices.Contains(roomMods, x.Modality) {
			c.fail(fp+".modality", "%q is not a room modality", x.Modality)
		}
		if x.Cents < 0 {
			c.fail(fp+".cents", "negative")
		}
		out.Fixed = append(out.Fixed, sim.FixedCost{ID: x.ID, Name: x.Name, Account: x.Account, Modality: x.Modality, Monthly: x.Cents})
	}
}

func (c *checker) dayRange(p string, r [2]int64, lo, hi int64) {
	if r[0] < lo || r[1] > hi || r[0] > r[1] {
		c.fail(p, "[%d, %d] must be an increasing range within %d–%d", r[0], r[1], lo, hi)
	}
}

// clockOfDay parses a point in time within a day (00:00 to before 24:00).
func (c *checker) clockOfDay(p, s string) simtime.Duration {
	d := c.clock(p, s)
	if d >= simtime.Day {
		c.fail(p, "%q must be before 24:00 (a time within the day)", s)
	}
	return d
}
