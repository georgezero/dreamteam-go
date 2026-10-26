# DreamTeam agent evaluation harness (V0.4)

The harness runs agent policies, built in or as separate processes, on
identical scenarios and seeds through the released agent protocol (`dreamteam.agent/v1`,
[AGENT_PROTOCOL.md](AGENT_PROTOCOL.md)). It records what each policy
decided and what came of it, verifies every saved session by replay, and
compares the policies against a baseline.

It is product infrastructure, not a scoring system. It adds no scoring
rule, no composite score and no ranking. Its outcomes are the
simulator's own measurements. It uses no clinical v5 or v6 policy.

Status: V0.4b, development (0.4.0-dev.2). Not released. V0.4a built the
harness; V0.4b adds external policy processes (§4), withholds the seed
from every policy (§3, §11) and adds `reproduce` (§8).

## 1. Commands

    dreamteam benchmark run CONFIG [-out DIR]
    dreamteam benchmark report RESULTS [-format text|json]
    dreamteam benchmark verify RESULTS [-t90]
    dreamteam benchmark reproduce RESULTS [-out DIR]

`run` works in four steps:
1. It validates the configuration; nothing runs if it is invalid.
2. It plays every scenario × seed × policy, in that order.
3. It writes the results directory (§8).
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

`verify` checks a results directory (§9). It prints one line per check
and exits 0 when every check passes and 3 when one fails.

`reproduce` reruns a results directory's recorded configuration and
compares the rerun's files with the recorded manifest (§8). It exits 0
when everything matches, 3 when something differs, and 1 when the rerun
cannot be made (for example, results recorded by another engine).

## 2. Configuration

A configuration is a JSON object (`dreamteam.benchmark/v1`). Unknown
fields are refused. The V0.4a development configuration is
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
| `policies` | Built-in policy IDs (§3) and the IDs of `external` policies (§4), none repeated. |
| `external` | Optional: external policy processes (§4). |
| `baseline` | One of the policies; the report pairs every other policy with it. |
| `output` | A relative directory. |
| `evaluation.t90` | Ask for the T90 evaluation of each run (§7). |

An external policy is declared with its command and a timeout. The V0.4b
development configuration
([benchmark/v04b-dev/config.json](benchmark/v04b-dev/config.json))
declares the reference Python policy:

```json
 "policies": ["noop", "greedy", "rules", "python-ref"],
 "external": [
  {
   "id": "python-ref",
   "version": "1",
   "description": "…",
   "command": ["python3", "-I", "-B", "./clients/python/reference_policy.py"],
   "timeoutSeconds": 120
  }
 ],
```

| Field | Rule |
|---|---|
| `id` | As a configuration's `id`; not a built-in policy's; listed in `policies`. |
| `version` | Required. Change it whenever the policy's behaviour changes. |
| `description` | Required. |
| `command` | The program and its arguments. An element beginning with `./` or `../` is a file relative to the directory the benchmark is **run from** (the repository root for the configurations here); the harness makes it absolute and records its SHA-256 in the manifest. Every other element is passed unchanged; a program named without a path is looked up in `PATH`. Before any run, the program must be found and every named file must be readable. |
| `timeoutSeconds` | 1–3600: the limit for each decision point, and for the exit after the last one. |

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

Every policy reaches the session through a **gate**. The gate decodes each
request exactly as the server does and passes a permitted request's
bytes on unchanged. It answers any other request itself with the error
code `not_permitted`; that request never reaches the session. The
harness alone advances time, saves, and asks for `t90`. `t90` simulates
the rest of the horizon with the run's seed, which no policy may see
while it decides.

A new policy instance plays each run, and each run has its own session.
Nothing a policy keeps between decision points outlives its run, and no
policy sees another run's state or outcome.

A policy sees exactly what an external agent sees through the protocol,
with two differences, the same for every policy:
* the gate removes `apply`, `advance`, `save` and `t90`;
* under the harness, `hello` withholds the seed (§11).

In their place `hello` carries a `benchmark` object:

```json
"benchmark": {
 "seedWithheld": true,
 "permittedOps": ["hello", "observe", "sections", "actions", "validate", "preview", "digests"],
 "decision": "dreamteam.benchmark-policy/v1"
}
```

Outside the harness, `hello` is unchanged: it reports the seed and has no
`benchmark` object.

Built-in policies run in the harness's process, behind the gate. External
policies (§4) run as separate processes, and their requests pass through
the same gate.

### The built-in policies

| ID | Version | What it does |
|---|---|---|
| `noop` | 1 | Never submits an action. |
| `greedy` | 1 | The V0.3 protocol demonstration (`agent.Greedy`), unchanged. When the oldest unread study is over a day old, it raises the first radiologist with room and no pending change by 5 points of site reading allocation. It is not a strategy. |
| `rules` | 1 | The rule-based baseline below. |

V0.4b adds no built-in policy. Its reference external policy is
`python-ref` (§4).

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

## 4. External policies

An external policy is a program the harness runs as a separate process,
one fresh process per run. It speaks the **policy protocol**
(`dreamteam.benchmark-policy/v1`): JSON lines over its standard input
and output. No HTTP or socket is involved.

### The policy protocol

The harness writes **turns** to the policy's standard input:

    {"protocol":"dreamteam.benchmark-policy/v1","turn":"decide","point":0}
    {"protocol":"dreamteam.benchmark-policy/v1","turn":"end"}

`point` counts the decision points from 0. A turn carries nothing else:
no seed, no time and no run identity. The policy observes what it needs.

During a `decide` turn, the policy writes agent/v1 **requests** to its
standard output, one per line. Each is answered by exactly one
**response** line on its standard input, in order, before the next is
read. The requests go through the gate. Permitted operations reach the
session; the others are answered `not_permitted`. A line that is not a
request is answered as the protocol answers it (`invalid_json`,
`invalid_request`). The policy chooses its own request ids. The
harness's ids begin with `h`, so a policy should not use that prefix.

The policy ends the turn with its **decision**, one line:

    {"decision":{"batches":[[{"type":"set_reading_allocation", …}], …]}}

`batches` is required: `[]` means no change, and at most 64 batches are
allowed. A line is a decision when it is a JSON object with a `decision`
member. It must then have exactly that shape, otherwise the run fails
(`policy_error`). The harness applies the batches in order, as it
applies a built-in policy's.

After the last decision point, the harness writes the `end` turn and
closes the policy's input. The policy must then exit with status 0.

### What the harness does

* **Process.** It starts one process per run, with the configured
  command, in a fresh empty working directory. That directory is created
  in the system's temporary directory (`TMPDIR`), outside the results,
  and removed after the run. The process inherits the harness's
  environment.
* **Nothing about the run reaches the process outside the protocol.**
  The seed, the run's ID and the results' paths are not in its
  arguments, environment or working directory name. The command is the
  configuration's, the same for every run.
* **Timeouts.** `timeoutSeconds` bounds each decision point, from the
  turn to the decision, including the time the harness spends answering
  requests. It also bounds the exit after `end`.
* **Standard error.** The process's standard error is captured
  separately, up to 1 MiB, in the run's `stderr.txt`. It is kept but not
  hashed: it is the process's own output, outside the protocol.
* **Failures** are classified, the process is stopped (killed if it
  still runs) and reaped, and the other runs go on:

| Status | What happened |
|---|---|
| `policy_crash` | The process could not start, ended before the end of the run, or exited with a status other than 0 after `end`. |
| `policy_timeout` | No decision within the timeout, or no exit within the timeout after `end`. |
| `policy_error` | A decision line that is not a decision as specified, or a line over 1 MiB. |

Requests the protocol refuses (a malformed line, bad arguments) are not
failures. They are counted as request errors, and the turn goes on.

### The reference policy: `python-ref`

[clients/python/reference_policy.py](../clients/python/reference_policy.py)
is a standard-library Python program that uses nothing but the
protocol. It is a weekly review, deliberately simple, fixed before any
benchmark run and not tuned to outcomes:
* At its first turn it reads `hello` and checks the benchmark contract.
* It reviews at decision points 0, 7, 14, … and does nothing between
  them.
* At a review, if the oldest unfinished study is more than 24 hours old,
  it reads the action catalog. It proposes raising the site reading
  allocation of the editable radiologist with the lowest allocation in
  effect, room to grow and no pending change, by 5 points (to at most
  the maximum), from the earliest lawful date.
* It validates the proposal first and passes over a refused one for the
  next candidate. It submits at most one change, and only a valid one.
* A protocol error ends that turn with no change, noted on standard
  error. A broken exchange ends the process with status 1.

Run it with `python3 -I -B`: `-I` keeps the user's Python environment
out, and `-B` writes no bytecode into the repository.

## 5. A run

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
   metrics from it (§6). Finally it checks that the session's batches are
   the applied decisions, in order: the same time, actions and results.

**Statuses.** A run that does not complete is classified by what stopped
it, and its files are written as far as it got:

| Status | What stopped the run |
|---|---|
| `completed` | Nothing. |
| `policy_error` | The policy returned an error or panicked, or (external) its decision could not be read. The session is saved at that point. |
| `policy_crash`, `policy_timeout` | An external policy's process failed (§4). The session is saved at that point. |
| `protocol_error` | The harness's own request failed (`advance`, `save`, `digests`). |
| `evaluation_error` | T90 failed other than as not applicable. |
| `replay_error` | The saved session did not replay, or its batches differ from the decisions. |
| `harness_error` | The session could not be opened, or a file could not be written. |

**Refusals and protocol errors are not failures.** They are counted, and
the run goes on:
* a refused batch is one the session found invalid; nothing is applied;
* a protocol error is a batch the protocol could not read (not JSON, an
  unknown field, more than 64 actions);
* a denied request is a policy request the gate answered;
* a request error is any other policy request answered with an error.

## 6. Metrics

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

## 7. T90

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

## 8. Results directory and reproducibility

    config.json          the configuration, verbatim
    manifest.json        engine, protocol, policies (ID, version; an external
                         policy's command, timeout and the SHA-256 of its
                         files), the gate's operations, the metric
                         definitions, and the SHA-256 of every file below
    report.json          the report (§10)
    report.txt
    runs/<scenario>-s<seed>-<policy>/
      run.json           the run record: identity, status and failure, initial
                         and final clock and digests, counts, replay, the
                         metrics, T90
      decisions.json     each decision point: time, batches, each batch's answer
      transcript.jsonl   every request and response line of the run (ids h… are
                         the harness's; the others, the policy's: p… for a
                         built-in policy, an external policy's own ids)
      session.json       the saved session (dreamteam verify / replay / t90)
      result.json        the replayed session's canonical result
      t90.json           the protocol's t90 response (when evaluated)
      stderr.txt         an external policy's standard error (not hashed)
    timing.json          wall-clock seconds per run (not hashed)

**Guarantees:**
* Every file except `timing.json` and `stderr.txt` is deterministic,
  given deterministic policies. Identical configurations and policy
  versions on the same engine give byte-identical files, `manifest.json`
  included. A rerun into another directory can be compared file by file,
  or by its manifest alone.
* An external policy is deterministic only if its program is: same
  observations in, same requests and decisions out. The harness cannot
  enforce that. It records the program's files' SHA-256, not its
  interpreter's version. A timeout depends on the machine's speed, so a
  run that times out is not reproducible.
* No hashed file holds an absolute path or a wall-clock time.
  `timing.json` and `stderr.txt` are listed in the manifest as
  nondeterministic and are not hashed.
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

**Reproducing results.** `dreamteam benchmark reproduce RESULTS [-out
DIR]` reruns RESULTS' `config.json` into `DIR`, which defaults to
`RESULTS.reproduce` and must not exist or must be empty. It then compares
the following with RESULTS' manifest:
* engine and protocol;
* policies, including an external policy's file hashes;
* metric definitions;
* each run's status;
* every hashed file;
* `manifest.json` byte for byte.

It requires the engine that recorded the results. Like `run`, it
resolves an external policy's files from the directory it is run from.
Timing and standard error are not compared. The V0.4a smoke run, for
example:

    dreamteam benchmark run docs/benchmark/v04a-smoke/config.json -out A
    dreamteam benchmark reproduce A

## 9. Verification

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

## 10. Report

The report, text and JSON, has:
1. the runs table: status, applied, refused and protocol errors, replay,
   key metrics and T90 (external policies are marked in the header);
2. initial conditions per scenario and seed;
3. a per-policy summary: minimum, median and maximum of every integer
   metric over completed runs;
4. paired comparisons with the baseline per scenario and seed:
   decisions same or different, outcome identical or different, the
   differences, and per metric the seeds where the policy's value is
   higher, lower or equal;
5. failures (including process crashes and timeouts), refusals,
   protocol errors, denied requests and request errors;
6. replay verification.

Development samples are small. The report makes no significance claim,
and none should be read into it.

## 11. Seed privacy and the trust boundary

Under the harness, no policy is told the run's seed:
* `hello` omits it and says `"seedWithheld": true`;
* `observe`, `sections`, `actions`, `validate`, `preview` and `digests`
  never carried it, and their errors do not mention it;
* `t90` and `save`, whose answers name the seed, are the harness's alone.

**Tested.** `TestSeedIsWithheld` checks, for a built-in and an external
policy on two scenarios, every permitted operation, including failing
calls. The seed's digits appear in no response a policy receives.
`TestExternalPolicy` checks that they appear nowhere in an external
process's arguments, environment or working directory.

The harness keeps the seed in its own evidence: the configuration,
`run.json`, `session.json`, `t90.json` and the report.

**The boundary.** This prevents disclosure through the benchmark's
protocol only. It is not isolation. The digests an agent may read are
derived from the session's state; an agent holding the binary could, in
principle, search for a seed that reproduces them. External policies
also run with the harness user's rights. They are not sandboxed: they
can read local files, including the configuration and the results
written so far, inspect the binary, and run their own simulations. No
OS-level isolation is claimed.

**External policies are trusted development participants in V0.4.**
Benchmarking an untrusted agent would need OS-level isolation and a
withheld or derived seed scheme; that is a later requirement.

## 12. Known limitations (V0.4)

* **Trusted participants only.** See §11. An external policy can read
  the harness's files and could infer the seed by brute force from the
  binary and the digests.
* **No feedback within a decision point.** A policy's batches are
  applied together after its decision. It cannot observe the effect of
  one batch before submitting the next at the same decision point.
* **Lead times.** A reading allocation takes effect at least 14 days
  after it is set. In a 30-day run, an allocation change affects at most
  the last 16 days; hence the V0.4b development run's 90 days.
* **Timeouts are wall-clock.** They include the harness's own time to
  answer requests, so a slow machine needs a generous timeout. A run
  that times out is not reproducible.
* **One process tree level.** On failure the harness kills the policy
  process it started. A process that starts children of its own must
  stop them itself.
* **T90 is slow.** It runs two ~200-day simulations, about 40 s per run
  on a Raspberry Pi 4.
* **Scenarios.** Configurations name bundled scenarios only.
* **Isolation of built-in policies is by construction.** The gate and
  one session per run isolate them in the harness's process. The gate
  holds the session's handler in an unexported field.
* **The policies are demonstrations, not strategies.** `greedy` reacts
  only to the oldest unread study's age. `rules` and `python-ref` are
  simple rules fixed in advance. All three use one lever, the site
  reading allocation.
* **No LLM integration.** An LLM agent would be one more external
  process, but none is provided.
