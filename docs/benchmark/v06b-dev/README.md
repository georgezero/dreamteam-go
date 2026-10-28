# V0.6b development matrix: v06b-dev

The development matrix of [PROTOCOL.md](PROTOCOL.md) (committed in 8556766,
before any run on its seeds):
* **Run:** with `dreamteam benchmark suite run` (§9 of
  [BENCHMARK_SUITE.md](../../BENCHMARK_SUITE.md)).
* **Matrix:** every entry of `dreamteam-dev` version 1, seeds 41001–41003,
  90 days.
* **Policies:** `noop`, `greedy` and `rules` (unchanged) and the new
  `conservative` baseline.

**It demonstrates the workflow, not a ranking:** no score, and no claim
about any policy.

## Files

| File | What it is |
|---|---|
| `PROTOCOL.md` | The protocol, with the disclosed design-time runs |
| `suite-run.json` | The suite-run configuration, as run |
| `ci-suite-run.json` | The CI smoke test (seed 41001, 21 days) |
| `contract.txt`, `contract.json` | The contract report the runner wrote: a complete example of the standard report |
| `report.txt`, `report.json`, `manifest.json` | The harness's report and the manifest (with the suite files hashed) |
| `demand_diagnostic.py`, `demand.md` | The demand diagnostic and its output |

Kept outside the repository: the 60 run directories (247 MB).

## 1. Suite execution

* **Command:** `dreamteam benchmark suite run
  docs/benchmark/v06b-dev/suite-run.json` ran 60 runs (5 entries × 3 seeds
  × 4 policies) in 76 minutes on a Raspberry Pi 4 (linux/arm64, engine
  0.6.0-dev.2).
* **Status:** complete, 60 of 60 runs completed; no failed run.
* **Report:** the contract report opens `complete: 60 of 60 runs`.

## 2. Integrity

| Check | Result |
|---|---|
| `benchmark verify -t90` (Pi), with the suite checks | 578 of 578 checks |
| Suite file against the bundled scenarios (every fingerprint as recorded) | ok |
| `config.json` is the expansion of `suite-run.json` | ok |
| Every run's scenario fingerprint is its entry's | ok |
| Contract report rebuilt from the run records | ok |
| Starting state identical across the four policies, for every entry and seed | yes (15 of 15) |
| `benchmark reproduce` on linux/amd64 (horo) | 383 of 383 checks, manifest byte-identical (765 s) |
| A fresh suite run on linux/amd64 | manifest and `contract.txt` byte-identical to the Pi's; its own `verify -t90` 578 of 578 |
| CI smoke test (seed 41001, 21 days) on linux/amd64, linux/arm64, darwin/arm64 | manifest and `contract.txt` byte-identical (run 40732527526) |
| T90 | evaluated for the reporting-backlog entry only (12 runs, all NO GO); `not_applicable` for the other 48 |

Demand growth and reporting backlog share their starting state on every
seed, by construction (V0.6a).

## 3. No action across the suite

Median over the three seeds, 90 days (every value per run is in
`contract.txt` §3–8):

| Entry | Operating profit | Lost orders | TAT p90 | Breaches | Minimum cash |
|---|---|---|---|---|---|
| Financial pressure | −$30,935 | 516 (10.4%) | 854 min | 291 | $49,245 |
| Reporting backlog | $3,250 | 771 (13.9%) | 1,036 min | 441 | $56,154 |
| Demand growth | $13,146 | 1,008 (16.4%) | 1,215 min | 551 | $56,154 |
| Staffing shortage | −$28,157 | 1,070 (19.2%) | 742 min | 238 | $57,214 |
| Stable | $23,893 | 604 (10.9%) | 929 min | 356 | $60,778 |

These are the same practices as in V0.6a, on new seeds, and the pattern
is the same. No run went below zero cash.

## 4. The conservative baseline

**It took no action in any of its 15 runs.** Every run equals no action
on every dimension (`contract.txt` §9), and its paired differences are
all zero.

Its observations, read back from its own requests in the run
transcripts, say why:

| Entry | Most days in a row with the oldest study over 24 h at 07:00 | Decision points with trailing-30-day profit below zero (of 90) |
|---|---|---|
| Financial pressure | 2, 2, 2 | 90, 90, 90 |
| Reporting backlog | 2, 2, 2 | 88, 54, 61 |
| Demand growth | 2, 2, 2 | 78, 36, 52 |
| Staffing shortage | 1, 1, 2 | 90, 90, 90 |
| Stable | 2, 2, 2 | 63, 12, 28 |

(Seeds 41001, 41002, 41003.)

* **The trigger never fired.** Its trigger is a backlog over 24 hours on
  three decision points in a row, and that never happened. The oldest
  study passes a day only on Sunday and Monday mornings, after the day
  with no reading.
* **The guardrail would often have blocked it anyway.** Where the
  practice lost money over the trailing 30 days, the profit guardrail
  forbids a raise.

The suite's reporting problem is real: TAT p90 of 742–1,215 minutes, and
hundreds of breaches of the soon (4 h) and urgent (1 h) targets. But the
oldest-study age at the morning decision point does not show it. That is
a property of the policy's service signal, declared before the run, and
it is reported, not tuned.

**The other baselines:**
* `greedy` and `rules` again made 12 allocation raises in every run, with
  operating profit $25,961–$33,003 lower than no action.
* The effect on TAT p90 again depended on the scenario: −249 to −433
  minutes under demand growth, −79 to −107 under staffing shortage (one
  `rules` run, −575).
* `rules` and `greedy` differed in TAT p90 by at most 22 minutes, apart
  from that one run.

**What it adds to the suite.** As declared, `conservative` v1 is a
baseline that recognizes no sustained problem in any of these
practices. It is therefore identical to no action here. It does not yet
give the strategic diversity the owner asked for.

A policy with a different service signal (for example, overdue soon and
urgent studies, or breaches per day) would be a new, separately versioned
policy, declared before any evaluation. That is a decision for the next
slice, not a change to version 1.

## 5. Demand diagnostic (`dreamteam.demand-diagnostic/v1`)

From `demand.md`. **R1:** generated demand (referral orders and walk-ins,
day by day) was identical across the four policies for every entry and
seed (15 of 15). Generated demand is therefore a measure independent of
decisions.

**Declared conditions** (demand growth, no action):

| Seed | G1: referral orders per operating day, days 1–30 → 61–90 | G2: ratio to reporting backlog by window | G1 | G2 |
|---|---|---|---|---|
| 41001 | 75.8 → 79.6 | 1.018, 1.117, 1.182 | holds | holds |
| 41002 | 75.5 → 83.4 | 1.022, 1.111, 1.191 | holds | holds |
| 41003 | 78.4 → 79.9 | 1.027, 1.119, 1.167 | holds | holds |

**Demand against outcome.** Under demand growth, per operating day:
* referral orders rose by 2–11%;
* completed exams stayed flat (59.7–62.1);
* lost orders rose (10.4–10.7 → 14.6–16.2).

The added demand is lost, not imaged, so imaging capacity binds. Reporting
backlog, staffing shortage and stable received the same referral orders in
every window on each seed: the same demand meets different staffing.

V0.6a's original check (raw 30-day totals) stays recorded as failed on
two seeds of three ([v06a-dev/README.md](../v06a-dev/README.md)). This is a
different, prospectively declared diagnostic on new seeds.

## 6. What this shows and does not

**Shows:**
* One command runs the whole suite and writes a standard report on the
  contract's six dimensions.
* Verify and reproduce cover the suite's own files.
* The runs are byte-identical to the ordinary harness's.
* Demand growth raises generated demand per operating day, measured
  independently of decisions and outcomes.

**Does not show:**
* **A distinct baseline.** `conservative` v1 never acted, so the
  baselines still offer two behaviours: no action, and 12 allocation
  raises.
* **Anything about LLM agents** (none ran).
* **Statistical differences:** three development seeds.
