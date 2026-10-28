# V0.6b development matrix protocol

Protocol version `v06b-protocol/1`, written 2026-10-28 and committed with
the suite runner, the conservative baseline and the demand diagnostic
before any run on the seeds below. It demonstrates:
* the suite runner and the contract report
  ([BENCHMARK_SUITE.md](../../BENCHMARK_SUITE.md) §9);
* the conservative baseline's behaviour (§7);
* the demand diagnostic (§8);
* verification and reproduction.

**It is not a policy comparison:** no score, ranking or significance
test, and no claim about any policy.

## 1. What runs

| | |
|---|---|
| Configuration | [suite-run.json](suite-run.json) (`v06b-development`) |
| Suite | `dreamteam-dev` version 1, all five entries |
| Seeds | 41001, 41002, 41003 |
| Horizon and cadence | The suite's: 90 days, decisions daily at 07:00 |
| Policies | `noop` (the baseline), `greedy`, `rules` (all version 1, unchanged) and `conservative` version 1 (new) |
| T90 | Requested; evaluated for the reporting-backlog entry only, `not_applicable` elsewhere |
| Runs | 60 (5 entries × 3 seeds × 4 policies) |

**The seeds** are the next unused block after V0.6a's 40901–40903:
* not reference seeds (1001–1020);
* not V0.2 development seeds (901–910);
* not from a reserved stream;
* not benchmark holdout seeds (none exist).

`TestCommittedSuiteRuns` loads the configuration through the sealed
holdout guard.

**Not run:**
* the V0.6a matrix (no integrity issue requires it);
* any language model.

**The machine:** a Raspberry Pi 4 (linux/arm64), one job at a time,
memory-capped, with a 4-hour limit. At the limit, the completed runs are
reported and the suite is reported as partial; nothing is shortened.

## 2. What is reported

1. **Suite execution:** completeness (`complete` or `partial`), failed
   runs if any, and the time taken.
2. **The contract report** (`contract.txt`, `contract.json`) as the
   runner writes it, committed with the results.
3. **The conservative baseline**, from the contract report, for each entry
   and seed:
   * the actions it took (count and dates, from `decisions.json`);
   * its differences from no action on every dimension;
   * whether it chose no action throughout.

   Nothing about it is tuned after the run.
4. **The demand diagnostic** (`dreamteam.demand-diagnostic/v1`,
   [demand_diagnostic.py](demand_diagnostic.py)):
   * R1, generated demand identical across policies;
   * per operating day: referral orders, walk-ins, exams completed and
     lost orders, by 30-day window;
   * the declared conditions G1 and G2 for demand growth, with no action,
     on every seed.

   A condition that does not hold is reported as not holding.
5. **Integrity:**
   * `benchmark verify -t90` (with the suite checks);
   * `benchmark reproduce`;
   * a fresh run on linux/amd64 with a byte-identical manifest;
   * starting states identical across policies for each entry and seed;
   * scenario fingerprints as the suite records them.

## 3. Design-time runs (disclosed)

Before this protocol was committed, two kinds of run were made, none on
its seeds:
* **Unit tests of the conservative baseline** (test seed 7, the
  sustainable and financial-pressure scenarios and a test-only overloaded
  practice). In the sustainable practice the oldest study passed 24
  hours only on Sunday and Monday mornings, so the policy never raised.
  In the overloaded practice as first written, the backlog made it lose
  money, so the profit guardrail forbade raising. The test practice was
  then made profitable (payer rates doubled) so that the raise path is
  exercised; the policy's rules and thresholds were not changed.
* **A test of `demand_diagnostic.py`** on design seed 1 (the public seed
  V0.6a's design used): reporting backlog and demand growth, no action
  and conservative, 90 days, no T90. It found that the saved session
  keeps the scenario under `scenario.resolved`; the script was corrected.
  On that seed R1 held, G1 held (73.8 → 79.7 orders per operating day)
  and G2 held (ratios 1.025, 1.125, 1.177). The conservative policy made
  no change there.

## 4. Evidence

**Committed** after the run, in this directory:
* `contract.txt`, `contract.json`, `report.txt`, `report.json`,
  `manifest.json`;
* `demand.md`, the diagnostic's output;
* `README.md`, the results.

**Kept outside the repository:** the run directories.

**CI:** [ci-suite-run.json](ci-suite-run.json), the five entries with the
four policies on seed 41001 for 21 days, run, verified and reproduced on
linux/amd64, linux/arm64 and darwin/arm64, with the manifests compared
byte for byte.

## Amendments

None.
