# V0.6a baseline characterization protocol

Protocol version `v06a-protocol/1`, written 2026-10-28 and committed with
the suite before any run on the seeds below. It checks that the five
development scenarios ([BENCHMARK_SUITE.md](../../BENCHMARK_SUITE.md)) are
usable and differ from one another. **It is not a policy comparison:**
no score, ranking or significance test, and no claim about any policy.

## 1. What runs

| | |
|---|---|
| Configuration | [config.json](config.json) (`v06a-characterization`) |
| Suite | `dreamteam-dev` version 1 ([suite file](../suite/dreamteam-dev-1.json)) |
| Scenarios | `bench-financial-pressure`, `sustainable`, `bench-demand-growth`, `bench-staffing-shortage`, `bench-stable` |
| Seeds | 40901, 40902, 40903 |
| Horizon and cadence | 90 days from the takeover; decisions daily at 07:00 |
| Policies | `noop` (the baseline), `greedy` and `rules`, built in, version 1, unchanged |
| T90 | Requested. It is evaluated for `sustainable` only; the harness reports the others as `not_applicable`. |
| Runs | 45 (5 scenarios × 3 seeds × 3 policies) |

**Why 90 days:** the reading allocation and teleradiology need 14 days'
notice, and a hired technologist 21–28 days plus training, so a change
decided in the first weeks must have time to act.

**Why three seeds:** enough to see whether a scenario's condition holds
beyond one seed; not a sweep.

**The seeds** are the next unused block after V0.5c's 40801–40805. They
are not reference seeds (1001–1020), not V0.2 development seeds (901–910),
not from a reserved stream, and not benchmark holdout seeds (none exist).
`TestCommittedConfigs` loads the configuration through the sealed holdout
guard.

**The machine:** a Raspberry Pi 4 (linux/arm64), one benchmark at a time,
memory-capped, with a 4-hour limit. If the limit is reached, the
completed runs are reported and the rest listed as incomplete; nothing is
shortened.

## 2. What is reported

From the harness's report and run files, by `analyze.py` (committed with
this protocol), into `tables.md`:

1. **Runs:** status, replay verification, initial-state identity, T90
   status.
2. **No-action profile** of every scenario, by the contract's dimensions
   (§3 of the specification): minimum, median and maximum over the three
   seeds.
3. **Intended conditions,** for no action on each seed, against the same
   criteria as the design check (§3 below):
   * financial pressure: operating profit below zero;
   * reporting backlog: TAT p90 above 600 minutes and breaches above 5%
     of final reports;
   * demand growth: more referral orders in days 61–90 than in days 1–30,
     and more in all than the reporting-backlog scenario on that seed;
   * staffing shortage: more lost orders than the reporting-backlog
     scenario on that seed;
   * stable: operating profit above zero, and fewer lost orders and a
     lower TAT p90 than the reporting-backlog scenario on that seed.
4. **Distinctness:**
   * every scenario's fingerprint differs from every other's;
   * on each seed, the starting-state digests differ between scenarios,
     except demand growth and reporting backlog. Those two are identical at
     the takeover by construction: the warm-up (April to October) is the
     same, and demand growth differs only from November. A 5-day test on
     seed 1, run while writing `analyze.py`, showed this; the check was
     corrected before this protocol was committed;
   * for each pair of scenarios, the primary metrics (§3 of the
     specification) on which their no-action ranges over the three seeds
     do not overlap.
5. **Baselines** `greedy` and `rules`, paired with no action on each
   scenario and seed, by dimension, with action counts by type.
6. **T90** for the reporting-backlog scenario: each policy's verdict.

Nothing is combined into a score. A condition that does not hold on a
seed is reported as such; the scenario is not changed in response.

## 3. Scenario design before this protocol (disclosed)

The scenarios were built from documented calibration overlays
(specification §2). Their intended conditions were checked once with
**no-action runs on seed 1 only** (the public default and golden seed),
90 days. No other policy was run during design, and no characterization
seed was used.

* **Criteria** were written first (2026-10-28 09:13 EDT); they are the
  ones in §2.3.
* **Iteration 1.** Four of five scenarios met their criterion. Demand
  growth did not: it raised the November, December and January factors
  by 5%, 10% and 15% over the base profile's own, whose December factor
  is 0.93. Orders per 30 days went 1,903, 1,960, 1,874, so demand did not
  rise.
* **Revision.** Growth was redefined once, relative to October's level
  (×1.05, ×1.10, ×1.15). On seed 1 orders per 30 days then went 1,919,
  2,062, 1,992, above the first window by the third, and lost orders
  rose 293, 317, 330. No other scenario changed.

A 5-day run of all three policies on seed 1, made only to test
`analyze.py`, is the one other design-time run. It showed that demand
growth and reporting backlog share their starting state (§2.4).

Seed 1, no action, 90 days, after the revision:

| Scenario | Operating profit | Orders received | Lost | TAT p90 | Breaches |
|---|---|---|---|---|---|
| Financial pressure | −$44,547 | 4,779 | 492 | 815 min | 265 |
| Reporting backlog | $5,549 | 5,399 | 734 | 1,032 min | 477 |
| Demand growth | $17,924 | 5,973 | 940 | 1,133 min | 557 |
| Staffing shortage | −$22,852 | 5,399 | 986 | 728 min | 237 |
| Stable | $20,973 | 5,399 | 596 | 929 min | 342 |

## 4. Evidence

**Committed** after the run, in this directory:
* `report.txt`, `report.json`, `manifest.json`;
* `tables.md`;
* `README.md`, the characterization results.

**Kept outside the repository:** the run directories (sessions,
transcripts, results).

**Checks:**
* `benchmark verify -t90` on the results;
* `benchmark reproduce`;
* a second run on linux/amd64 with a byte-identical manifest.

**CI:** [ci.json](ci.json), the five scenarios with the three policies on
seed 40901 for 30 days, compared byte for byte across linux/amd64,
linux/arm64 and darwin/arm64.

## Amendments

None.
