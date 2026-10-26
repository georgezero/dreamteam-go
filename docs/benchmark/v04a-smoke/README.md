# V0.4a development run: v04a-smoke

The harness smoke test of V0.4a ([BENCHMARK.md](../../BENCHMARK.md)): the
three initial policies (`noop`, `greedy`, `rules`) on the bundled
sustainable scenario, five fresh development seeds, 30 days, decisions
daily at 07:00, T90 evaluated. That is 15 runs.

**It shows that the harness works. It is not a policy-performance claim.**
Five seeds are too few for any claim, and the report makes none.

## Files

* `config.json` is the configuration. It was committed alone (b9cc0b6)
  before any benchmark run on these seeds; its `seedPlan` says how the
  seeds were chosen.
* `manifest.json` records the engine, protocol, policies and metric
  definitions, and the SHA-256 of every deterministic file of the full
  results.
* `report.txt` and `report.json` are the report.

The full results directory is 28 MB, mostly protocol transcripts. It is
not committed; it is kept outside the repository. Any copy can be checked
against `manifest.json`. To reproduce it:

    dreamteam benchmark run docs/benchmark/v04a-smoke/config.json -out DIR
    sha256sum DIR/manifest.json docs/benchmark/v04a-smoke/manifest.json
    dreamteam benchmark verify DIR -t90

A rerun on engine 0.4.0-dev.1 must give the same `manifest.json` byte for
byte, which means every file it hashes is identical.

## Evidence (2026-10-26)

**Run.** On a Raspberry Pi 4 (linux/arm64, go1.27.1), all 15 runs
completed:
* every saved session replayed with every digest verified;
* every session's batches matched the applied decisions;
* every scenario and seed began from identical initial conditions;
* each run took 51–65 s, about 40 s of it T90.

**Reproducibility.** On horo (linux/amd64, go1.27.0) the same
configuration gave a byte-identical `manifest.json` (SHA-256 `81ecfaa1…`).
Every deterministic file is therefore identical across machines and
architectures. `benchmark verify` passed there, 156 of 156 checks.

**Verification.** On the Pi, `benchmark verify -t90` passed all 171 checks:
* the hashes, with no unlisted file;
* replay, result, metrics and decisions for each of the 15 runs;
* the rebuilt report;
* T90 recomputed for each of the 15 runs.

## What the run shows (development observations, not claims)

* **Noop never acts.** Greedy and rules each applied 4 reading-allocation
  changes per seed: 20 batches, none refused, no protocol error and no
  denied request.
* **Greedy and rules made the same changes in a different order.** Rules
  raises the radiologist with the lower allocation first; greedy raises
  the first listed. On these seeds the oldest unread study stayed over 24
  hours, so rules' lowering branch never fired.
* **The order barely matters at 30 days.** A change takes effect 14 days
  after it is set, so only each policy's first two changes take effect
  within the horizon. The two policies' outcomes are identical on four
  seeds and differ in reporting only on seed 40401, where one change took
  effect a week later.
* **Effect against noop.** Against noop, both policies finalized more
  reports and had lower TAT and fewer breaches on 5 of 5 seeds. They had
  more net revenue and more expense on 5 of 5. Operating profit was
  higher on 3 seeds and lower on 2. Imaging (orders, lost demand, exams,
  waits, overtime) was unchanged: a reading allocation does not change
  imaging capacity.
* **T90 (v0.2-eval-4, projected from day 30).**
  * Noop is NO GO on every seed: it is no action against no action.
  * Greedy and rules are NO GO on three seeds and CONDITIONAL GO on two
    (40403, 40405).
* **Rules is not yet a distinct baseline.** On this scenario and horizon
  it behaves almost like greedy. The fix is a V0.4b scope decision. Rules
  is not to be tuned to these results.
