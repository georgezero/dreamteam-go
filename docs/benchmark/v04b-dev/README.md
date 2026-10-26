# V0.4b development comparison: v04b-dev

The V0.4b development run ([BENCHMARK.md](../../BENCHMARK.md)). Four
policies on the bundled sustainable scenario, five fresh development
seeds, 90 days, decisions daily at 07:00, T90 evaluated. That is 20 runs.
The policies:
* `noop`, the baseline;
* `greedy` and `rules`, built in;
* `python-ref`, the reference external policy, run as a separate Python
  process per run.

**It demonstrates external execution and a longer horizon. It is not a
confirmatory benchmark or a policy-performance claim.** Five seeds
support no claim, and the report makes none. No policy was changed after
seeing these results.

## Files

* `config.json` is the configuration. It was committed with the
  reference policy (0853bcc) before any run on these seeds. Its
  `seedPlan` says how the seeds were chosen.
* `manifest.json` records the engine (0.4.0-dev.2), protocol, policies
  and metric definitions, and the SHA-256 of every deterministic file of
  the full results. That includes `python-ref`'s program file:
  `reference_policy.py`, `72292464…`.
* `report.txt` and `report.json` are the report.

The full results directory is 70 MB, mostly protocol transcripts, and is
kept outside the repository. To reproduce it, from the repository root
(the policy's command is relative to it):

    dreamteam benchmark run docs/benchmark/v04b-dev/config.json -out DIR
    sha256sum DIR/manifest.json docs/benchmark/v04b-dev/manifest.json
    dreamteam benchmark verify DIR -t90
    dreamteam benchmark reproduce DIR

## Evidence (2026-10-26)

**Run.** On a Raspberry Pi 4 (linux/arm64, go1.27.1, Python 3.11), all 20
runs completed in 2,094 s, with a peak of 200 MB:
* every saved session replayed with every digest verified;
* every session's batches matched the applied decisions;
* every seed began from identical initial conditions under all four
  policies;
* there were no policy failures, timeouts, refusals, protocol errors,
  denied requests or request errors;
* `python-ref`'s standard error was empty in all five runs, and its
  working directories were removed.

**Reproducibility.** On horo (linux/amd64, go1.27.0, Python 3.13) the same
configuration gave a byte-identical `manifest.json` (SHA-256 `dd957074…`).
Every deterministic file, the Python policy's transcripts included, is
therefore identical across machines, architectures and Python versions.
On horo:
* `benchmark verify -t90` passed 226 of 226;
* `benchmark reproduce` reran the configuration and matched all 147
  checks, `manifest.json` byte for byte.

**Verification.** On the Pi, `benchmark verify -t90` passed all 226
checks:
* the hashes, with no unlisted file;
* replay, result, metrics and decisions for each of the 20 runs;
* the rebuilt report;
* T90 recomputed for each of the 20 runs.

horo's verify gave the same 226 of 226.

## What the run shows (development observations, not claims)

**The external path works.** `python-ref` ran as a real subprocess on
every run:
* it read `hello` with the seed withheld;
* it observed at each weekly review, 14 requests per run;
* it returned a decision at all 90 decision points;
* it exited cleanly.

**`python-ref` never acted, and that is its declared rule, not a fault.**
It reviews at decision points 0, 7, 14, …, which here are all Tuesday
mornings. The reading backlog passes its 24-hour threshold only on
Sunday and Monday mornings, after the weekend. At its reviews the oldest
unfinished study was 11–17 hours old. Its runs are therefore exactly
noop's: the report pairs them "same decisions, identical outcome", the
harness's determinism check. The rule is not adjusted after the fact.

**Greedy and rules: the 90-day horizon does not separate them.** Both
made the same 12 reading-allocation raises per seed. They raised the
same two radiologists to the same final allocations (90 % and 80 % of
site time), in the opposite order: rules raises the lower allocation
first, greedy the first listed.

Rules' lowering branch never fired: the oldest unfinished study never
stayed at or under 8 hours. Their outcomes differ only in reporting:
* TAT p90 by 1–20 minutes;
* breaches by 0–13.

Profit is identical to the dollar on four seeds and differs by $1 on the
fifth. At 30 days their outcomes differed on one seed of five; at 90 days
they differ slightly on all five, but stay close.

**Effect of the raises against noop (greedy and rules alike, 5 of 5
seeds):**

| Measure | Against noop |
|---|---|
| Reports finalized | more |
| TAT p50 | about 130 min down to about 43 min |
| TAT p90 | 204–472 min lower |
| TAT breaches | 113–367 fewer |
| Alerts | fewer |
| Net revenue | +$511 to +$5,349 |
| Operating expense | about +$33,000 |
| Operating profit | −$27,357 to −$32,719 |
| Minimum cash | lower |

Imaging is unchanged: orders, lost demand, exams, waits and overtime.

**T90 (v0.2-eval-4, evaluated at day 90, not projected): NO GO for all
20 runs, for different reasons.**
* `noop` and `python-ref` fail service reliability: the breach rate
  stays over the gate.
* `greedy` and `rules` fail solvency: cash falls below the floor. On
  40503 and 40505, committed liabilities also exceed cash.

At 30 days (V0.4a), greedy and rules had been CONDITIONAL GO on two of
five seeds under projection. Over 90 days the added cost overtakes the
added revenue.

**The baselines do not yet differ in kind.** All three acting policies
use one lever, the site reading allocation, with nearly the same
trigger. A second, different baseline is an owner decision for a later
milestone.
