# DreamTeam agent evaluation harness (V0.4a)

The harness runs agent policies on identical scenarios and seeds through
the released agent protocol (`dreamteam.agent/v1`,
[AGENT_PROTOCOL.md](AGENT_PROTOCOL.md)). It records what each policy
decided and what came of it, verifies every saved session by replay, and
compares the policies against a baseline.

It is product infrastructure, not a scoring system. It adds no scoring
rule, no composite score and no ranking. Its outcomes are the
simulator's own measurements. It uses no clinical v5 or v6 policy.

Status: V0.4a, development (0.4.0-dev.1). Not released.

## 1. Commands

    dreamteam benchmark run CONFIG [-out DIR]
    dreamteam benchmark report RESULTS [-format text|json]
    dreamteam benchmark verify RESULTS [-t90]

`run` works in four steps:
1. It validates the configuration; nothing runs if it is invalid.
2. It plays every scenario × seed × policy, in that order.
3. It writes the results directory (§7).
4. It prints the text report.

Progress goes to stderr. The results go to `-out`; by default they go to
the configuration's `output`, relative to the configuration file. Results
are written once: `run` refuses an output directory that is not empty.

Exit codes:

| Code | Meaning |
|---|---|
| 0 | Every run completed. |
| 1 | A run failed (it is classified in the results), or the results could not be written. |
| 2 | The configuration is invalid. |

`report` rebuilds the report from a results directory's run records.

`verify` checks a results directory (§8). It prints one line per check
and exits 0 when every check passes and 3 when one fails.

## 2. Configuration

A configuration is a JSON object (`dreamteam.benchmark/v1`). Unknown
fields are refused. The development configuration is
[benchmark/v04a-smoke/config.json](benchmark/v04a-smoke/config.json):

```json
{
 "schema": "dreamteam.benchmark/v1",
 "id": "v04a-smoke",
 "description": "…",
 "scenarios": ["sustainable"],
 "seeds": [40401, 40402, 40403, 40404, 40405],
 "seedPlan": "Chosen 2026-10-26, before any benchmark run: …",
 "days": 30,
 "cadence": {"everyDays": 1, "at": "07:00"},
 "policies": ["noop", "greedy", "rules"],
 "baseline": "noop",
 "output": "results",
 "evaluation": {"t90": true}
}
```

| Field | Rule |
|---|---|
| `id` | 1–64 lowercase letters, digits, dots or hyphens. |
| `scenarios` | Bundled scenario IDs (`dreamteam scenario list`), none repeated. |
| `seeds` | An explicit list of positive integers, none repeated. The harness never generates or substitutes a seed. The V0.2 reference evaluation seeds 1001–1020 are refused. So is any seed of the sealed evaluation holdout: the error says how many, never which. |
| `seedPlan` | How the seeds were chosen. Required, because seeds must be chosen before any outcome is seen. |
| `days` | 1–366: the number of days of decision points. |
| `cadence` | `everyDays` (1–`days`) and `at` (practice-local `HH:MM`): the policy decides every `everyDays` days at `at`. |
| `policies` | Built-in policy IDs (§3), none repeated. |
| `baseline` | One of the policies; the report pairs every other policy with it. |
| `output` | A relative directory. |
| `evaluation.t90` | Ask for the T90 evaluation of each run (§6). |

## 3. Policies and the policy interface

### The interface

A policy is a decision maker driven through the protocol. At each
decision point the harness hands it a protocol client. It may call
`hello`, `observe`, `sections`, `actions`, `validate`, `preview` and
`digests`: the protocol's read and check operations. It returns the
batches to submit, in order.

The harness applies each batch with `apply`. A batch with any invalid
action is refused whole and changes nothing. The policy never calls
`apply` itself.

The client reaches the session through a **gate**. The gate decodes each
request exactly as the server does and passes a permitted request's
bytes on unchanged. It answers any other request itself with the error
code `not_permitted`; that request never reaches the session. The
harness alone advances time, saves, and asks for `t90`. `t90` simulates
the rest of the horizon with the run's seed, which no policy may see
while it decides.

A new policy instance plays each run, and each run has its own session.
Nothing a policy keeps between decision points outlives its run, and no
policy sees another run's state or outcome.

A policy sees exactly what an external agent sees through the protocol.
The gate removes `apply`, `advance`, `save` and `t90`, and adds nothing.

An external policy process can later stand in for a built-in one by
speaking the same exchange. At each decision point the harness would
relay the process's requests through the gate and receive its batches.
V0.4a has the built-in policies only.

### The built-in policies

| ID | Version | What it does |
|---|---|---|
| `noop` | 1 | Never submits an action. |
| `greedy` | 1 | The V0.3 protocol demonstration (`agent.Greedy`), unchanged. When the oldest unread study is over a day old, it raises the first radiologist with room and no pending change by 5 points of site reading allocation. It is not a strategy. |
| `rules` | 1 | The rule-based baseline below. |

**`rules`, version 1.** It keeps the site reading allocation in step with
the reading backlog. It reads only the observation's worklist (compact)
and the action catalog. At each decision point:

* **Pressure.** If the oldest unfinished study is more than 24 hours old,
  it raises an editable radiologist with room to grow and no pending
  change by 5 points (to at most the maximum), from the earliest lawful
  date. It picks the one with the lowest allocation in effect.
* **Clear.** If the oldest unfinished study has been at most 8 hours old
  at three decision points in a row, it lowers an editable radiologist
  with no pending change by 5 points, never below the scenario
  allocation. It picks the one furthest above the scenario allocation.
  The count of clear decision points then starts again.
* **Otherwise** it does nothing.

It submits a change only if `validate` finds it valid and the staffing
`preview` accepts it without displacing a booked patient. It makes at
most one change per decision point. The thresholds (24 h, 8 h, 3 points,
5 points) are round numbers fixed before any benchmark run. They are not
tuned to outcomes and must not be.

A policy's version changes whenever its behaviour does. Results record
each policy's ID and version.

## 4. A run

For one scenario, seed and policy:

1. **Open.** The harness opens a fresh session (`session.Open`, the
   scenario's warm-up included) and serves it through the protocol. It
   sends `hello` and records the clock and digests: the run's initial
   conditions.
2. **Decide and advance.** The clock moves to the first cadence time at
   or after the takeover. That is 07:00 on the takeover day for the
   bundled scenarios, whose takeover is at 06:00. Then, for each of the
   `days` days:
   * on decision days, the policy decides and its batches are applied;
   * the clock moves (`advance`) to the next day's cadence time.

   A run of `days` days ends at the cadence time `days` days after the
   first decision point.
3. **Save.** The harness asks for the final `digests` and `save`s the
   session.
4. **T90, if asked.** The harness asks the protocol for `t90`.
5. **Measure.** The harness loads the saved session, which replays it
   from its scenario, seed and batches and verifies every digest
   (`session.Load`). It then writes the canonical result of the replayed
   session, as `dreamteam replay FILE -format json` does, and reads the
   metrics from it (§5). Finally it checks that the session's batches are
   the applied decisions, in order: the same time, actions and results.

**Statuses.** A run that does not complete is classified by what stopped
it, and its files are written as far as it got:

| Status | What stopped the run |
|---|---|
| `completed` | Nothing. |
| `policy_error` | The policy returned an error or panicked. The session is saved at that point. |
| `protocol_error` | The harness's own request failed (`advance`, `save`, `digests`). |
| `evaluation_error` | T90 failed other than as not applicable. |
| `replay_error` | The saved session did not replay, or its batches differ from the decisions. |
| `harness_error` | The session could not be opened, or a file could not be written. |

**Refusals and protocol errors are not failures.** They are counted, and
the run goes on:
* a refused batch is one the session found invalid; nothing is applied;
* a protocol error is a batch the protocol could not read (not JSON, an
  unknown field, more than 64 actions);
* a denied request is a policy request the gate answered.

## 5. Metrics

Every metric is a value of the replayed session's canonical result
(`dreamteam.result/v1`, `session.Result`; [result.go](../internal/session/result.go)),
copied without conversion. The definitions are the result's:

* **Money** is the evaluation window's: from the exact takeover to the
  end of the run, baseline adjusted (`api.WindowMoney`).
* **Operations** are the window's operational metrics
  (`api.PeriodMetrics`).

The run record lists each value under its name. The manifest lists the
name, path, kind and group of every metric.

| Name | Path in result.json | Group |
|---|---|---|
| netRevenueCents | window.money.netRevenueCents | finance |
| operatingExpenseCents | window.money.operatingExpenseCents | finance |
| operatingProfitCents | window.money.operatingProfitCents | finance |
| margin | window.money.margin (num/den) | finance |
| operatingCashFlowCents | window.money.operatingCashFlowCents | finance |
| minCashCents | window.money.minCashCents | finance |
| shortfallCents | window.money.shortfallCents | finance |
| ordersReceived | window.ops.demand.received | demand |
| ordersLost | window.ops.demand.lost | demand |
| lostRate | window.ops.demand.lostRate (num/den) | demand |
| examsCompleted | window.ops.exams | demand |
| reportsFinal | window.ops.reporting.final | reporting |
| tatP50Sec, tatP90Sec | window.ops.reporting.tat (censored, over studies acquired in the window) | reporting |
| tatOverTarget | window.ops.reporting.overTarget | reporting |
| tatBreaches | window.ops.reporting.breaches | reporting |
| backlogEnd | window.ops.reporting.backlogEnd | reporting |
| waitToExamP50Sec, waitToExamP90Sec | window.ops.waits.toExam | service |
| waitsOver30 | window.ops.waits.over30 | service |
| displacedAppointments | window.ops.displacement.events (absent: none displaced) | service |
| overtimeSec, heldOver, skippedBreaks, noCoverage | window.ops.workforce | staffing |
| alertsRaisedWarn, alertsRaisedAlert, alertsActiveAtEnd | alerts | safety |
| daysBelowZero | balances.daysBelowZero | safety |
| actionsAttempted, actionsAccepted, actionsRejected | actions | actions |

Ratios stay as numerator and denominator pairs. The report computes
nothing from the metrics except:
* paired differences of integers (policy minus baseline);
* their signs (counts of higher, lower and equal);
* each policy's minimum, median (the lower middle value) and maximum.

The text report shows cents as whole dollars and seconds as whole
minutes; the JSON keeps the integers.

## 6. T90

With `evaluation.t90`, the harness asks the protocol for `t90` after the
run. The answer is the evaluation `dreamteam t90` gives for the saved
session: the embedded policy in force, **v0.2-eval-4**, with its
metric vector, gates and verdict unchanged. The plan is judged against
no action on the same seed.

`t90.json` keeps the response. The run record copies:
* the policy ID and digest;
* `asOfDay` and `projected` (a 30-day run is projected to T90 with no
  further decision, and labelled as such);
* the verdict and classification.

Where it does not apply:
* For a scenario the policy does not cover, T90 is `not_applicable`;
  there is no substitute verdict.
* A run that did not complete is `skipped`.

The noop policy's T90 compares no action with no action.

## 7. Results directory and reproducibility

    config.json          the configuration, verbatim
    manifest.json        engine, protocol, policies (ID, version), the gate's
                         operations, the metric definitions, and the SHA-256 of
                         every file below
    report.json          the report (§9)
    report.txt
    runs/<scenario>-s<seed>-<policy>/
      run.json           the run record: identity, status and failure, initial
                         and final clock and digests, counts, replay, the
                         metrics, T90
      decisions.json     each decision point: time, batches, each batch's answer
      transcript.jsonl   every request and response line of the run (ids h… are
                         the harness's, p… the policy's)
      session.json       the saved session (dreamteam verify / replay / t90)
      result.json        the replayed session's canonical result
      t90.json           the protocol's t90 response (when evaluated)
    timing.json          wall-clock seconds per run

**Guarantees:**
* Every file except `timing.json` is deterministic. Identical
  configurations and policy versions on the same engine give
  byte-identical files, `manifest.json` included. A rerun into another
  directory can be compared file by file, or by its manifest alone.
* No file holds an absolute path or a wall-clock time, except
  `timing.json`, which the manifest lists as nondeterministic and does
  not hash.
* Every number is an integer or the simulator's own value, so the files
  do not depend on the machine.

**Two kinds of comparison.** The report keeps deterministic equality
apart from outcome differences:
* **Same decisions** (equal action digests) must give an **identical
  outcome** (equal log, ledger and state digests). Otherwise the report
  flags the pair: determinism is broken.
* **Different decisions** may give different outcomes. Policies are not
  expected to share final digests.

**Initial conditions.** The report checks that every run on a scenario
and seed began at the same time with the same digests.

## 8. Verification

`dreamteam benchmark verify RESULTS` checks:

1. **Hashes.** Every file's SHA-256 against the manifest, and that no
   unlisted file is present.
2. **Replay.** For each run with a saved session, the session replays
   with every digest verified, and its final digests are the run
   record's.
3. **Result.** `result.json` is the replayed session's canonical result,
   byte for byte.
4. **Metrics.** The run record's metrics are that result's values.
5. **Decisions.** The session's batches are the applied decisions.
6. **Report.** `report.json` and `report.txt` are the report of the run
   records.
7. **T90, with `-t90`.** Each evaluated run's T90 is recomputed from its
   saved session and compared with `t90.json`. That is two simulations
   per run.

These checks show that the files agree with each other and with the
simulator. They are an integrity check, not an independent
certification. A run is checked only as far as its files go.

## 9. Report

The report, text and JSON, has:
1. the runs table: status, applied, refused and protocol errors, replay,
   key metrics and T90;
2. initial conditions per scenario and seed;
3. a per-policy summary: minimum, median and maximum of every integer
   metric over completed runs;
4. paired comparisons with the baseline per scenario and seed:
   decisions same or different, outcome identical or different, the
   differences, and per metric the seeds where the policy's value is
   higher, lower or equal;
5. failures, refusals, protocol errors and denied requests;
6. replay verification.

Development samples are small. The report makes no significance claim,
and none should be read into it.

## 10. Known limitations (V0.4a)

* **Built-in policies only.** The external-process adapter (§3) is
  designed but not built. LLM integration is deferred.
* **The seed is public.** `hello` reports the seed. An agent holding the
  binary could simulate the run ahead with it. The built-in policies do
  not. A benchmark of external agents needs the seed withheld or the
  agent isolated: see V0.4b.
* **No feedback within a decision point.** A policy's batches are
  applied together after it returns. It cannot observe the effect of one
  batch before submitting the next at the same decision point.
* **Timing in decision points.** A reading allocation takes effect at
  least 14 days after it is set. In a 30-day run, an allocation change
  affects at most the last 16 days.
* **T90 is slow.** It runs two ~200-day simulations, about 40 s per run
  on a Raspberry Pi 4.
* **Scenarios.** Configurations name bundled scenarios only.
* **Isolation is by construction.** The gate and one session per run
  isolate the built-in policies, which run in the harness's process. The
  gate holds the session's handler in an unexported field.
* **Greedy is a demonstration, not a strategy.** It reacts only to the
  oldest unread study's age.
