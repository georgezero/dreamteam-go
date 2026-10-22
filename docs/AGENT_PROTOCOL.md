# DreamTeam agent protocol (`dreamteam.agent/v1`)

V0.3a: a deterministic JSON protocol that lets an external program run a
DreamTeam practice. It is a thin layer over the session API, the same
API the terminal game and the headless runs use. It adds no simulation
behaviour and shows an agent only what the game shows a player.

Status: **development (0.3.0-dev.2: V0.3a, and V0.3b's `t90`, `sections`
and observation projections)**, not yet released.

## 1. Transport

```
dreamteam agent --stdio [-scenario ID|FILE] [-seed N] [-load FILE]
```

* **Requests and responses.** One JSON request per line on standard
  input; one JSON response per line on standard output, in the same
  order, exactly one per non-blank input line. Blank lines are skipped.
  The process ends when the input ends.
* **Output streams.** Standard output carries nothing but responses.
  Diagnostics (a "ready" line, startup errors) go to standard error.
* **The session.** It starts from a bundled scenario or a scenario file
  at a seed (defaults: `established`, seed 1, as for `play` and `run`),
  or from a saved session, which is replayed and verified first. `hello`
  reports which.

The request handling (`internal/agent.Handler.Handle`) takes one request
as bytes and returns one response as bytes; the stdio loop only carries
them. Another transport (HTTP, WebSocket) can carry the same bytes
without changing what they mean.

## 2. Messages

Request:

```json
{"id": 7, "op": "advance", "args": {"minutes": 60}}
```

* `id`: a string or a number, echoed in the response (optional; `null`
  when absent).
* `op`: the operation (§3).
* `args`: the operation's arguments, an object; optional when it takes
  none.

Unknown fields are errors, in the request and in `args`.

Response:

```json
{"protocol":"dreamteam.agent/v1","id":7,"ok":true,"result":{...}}
{"protocol":"dreamteam.agent/v1","id":7,"ok":false,"error":{"code":"invalid_args","message":"…","field":"minutes"}}
```

* **Fields.** Every response has `protocol`, `id` and `ok`, then either
  `result` or `error`.
* **Determinism.** Serialization is fixed: field order as specified,
  object keys of maps sorted. The same requests on the same session
  give byte-identical responses.
* **Clock.** Results that move or read the session carry its clock:
  practice-local `time`, `day` since the start, `logTotal` (observable
  log entries so far), and the five `digests` (event log, ledger,
  actions, state, run). The digests are the ones saved sessions and
  `dreamteam verify` check. They are one-way hashes and expose no
  state.

## 3. Operations

| Op | Args | Result | Changes the session |
|---|---|---|---|
| `hello` | — | protocol, engine version, scenario (`ref`, `id`, `name`, `fingerprint`), seed, the ops, the clock | no |
| `observe` | `recentLog` (0–2000, default 20); optional `sections` (names, §3.1) and `compact` (true) | `observation`: the game's observation (`dreamteam.observation/v1`): practice, rooms, staff, desk, demand, flow, worklist, finance, teleradiology, today's metrics, alerts, policies, actions, extensions, the recent log. With `sections` or `compact`: a projection of it (§3.1) | no |
| `sections` | — | the observation's sections (name, type, fields, whether always included, what compact leaves out), the header, and the compact omissions | no |
| `actions` | — | the action catalog: batch rules, and for each action type its JSON Schema, constraints and current targets (§4) | no |
| `validate` | `actions`: a batch | `valid`, and each action's result (accepted, unchanged or refused with the simulator's code and message) | no |
| `preview` | `actions`: a batch of V0.2 staffing actions | `preview`: the session's staffing preview: each result, hires, terminations, roster and allocation changes, teleradiology, slots opened and closed, displaced bookings, guaranteed and forecast costs | no |
| `apply` | `actions`: a batch | `applied`, each action's result and the clock | only when every action is valid |
| `advance` | exactly one of `minutes` (1–527,040, optional `stopOnAlert`), `to` (a later practice-local `YYYY-MM-DDTHH:MM:SS`, within 366 days), `next` (`"visible"` or `"end_of_day"`) | `reason` (`reached`, `alert`, `visible`, `horizon`) and the clock | yes: time only |
| `save` | `path` | `path`, the number of decision batches recorded, the clock | writes the session file |
| `digests` | — | the clock | no |
| `t90` | — | the T90 evaluation of the session as it stands (§3.2): `scenario` (`ref`, `id`, `fingerprint`), `seed`, `evaluation` | no |

**`advance`** uses the session's own semantics:

* `minutes`: run that long (`stopOnAlert`: stop right after a
  transaction that logged an alert, as the game's auto-pause does).
* `to`: run to that time.
* `next: "visible"`: run to the next observable log entry, searching up
  to 7 days (`horizon` if none).
* `next: "end_of_day"`: finish today's operations.

### 3.1 Observation projections

Without `sections` or `compact`, `observe` returns the full observation,
exactly as before. With either, it returns a projection of that same
observation. It is cut from the full observation's own JSON, so it can
show nothing the full observation does not, and every value it shows is
the full observation's (the same bytes, in the same order):

```json
{"id":1,"op":"observe","args":{"sections":["worklist","alerts","staff"],"compact":true}}
→ {"observation":{"schema":…,"time":…,"day":…,"weekday":…,"date":…,"clock":…,"staff":[…],"worklist":{…},"alerts":{…}},
   "sections":["schema","time","day","weekday","date","clock","staff","worklist","alerts"],"compact":true,
   "omitted":{"alerts.resolved":20,"worklist.recent":50}}
```

* **`sections`:** the top-level keys of the observation, as `sections`
  lists them: `practice`, `rooms`, `staff`, `desk`, `demand`, `flow`,
  `worklist`, `finance`, `teleradiology`, `today`, `alerts`, `policies`,
  `actions`, `extensions`, `logTotal`, `recentLog`, and the header. The
  header (`schema`, `time`, `day`, `weekday`, `date`, `clock`) is always
  included. Each section's fields are those of its type in
  `internal/api` (`sections` lists them). An unknown name is
  `unknown_section`, a repeated one `invalid_args`; neither changes
  anything.
* **`compact`:** leaves out four record listings, which are history, not
  current state, and reports how many entries each had in `omitted`:
  * `finance.claims.openClaims`: the open claims one by one (the counts
    and aging stay);
  * `finance.journal`: the latest journal entries (totals and balances
    stay);
  * `worklist.recent`: the latest finalized reports (the unread and
    reading items, with urgent and overdue flags, stay);
  * `alerts.resolved`: the resolved alerts (the active ones stay).
  
  Everything else stays, including every count, rate, aggregate, active
  alert and unread study. On the sustainable scenario at day 22 the full
  observation is ≈116 KB and the compact one ≈29 KB.

### 3.2 The T90 evaluation

`t90` evaluates the session's plan by the evaluation policy in force
(`v0.2-eval-4`), exactly as `dreamteam t90` does for the saved session:

* **What is compared:** the decisions before day 90, against no action
  on the same seed.
* **What it returns:** the metric vector, every service gate (pass, fail
  or insufficient), the plan gates and the verdict, under `evaluation`
  (the same object as `dreamteam t90 -format json`).
* **Projection:** `evaluation.projected` is true while the session has
  not reached day 90. Days not yet played are then your plan as it
  stands, with no further decision. After day 90 both plans run with no
  decision (the evaluation tail).
* **Explicit only.** It is never part of `observe`; an agent gets
  evaluation results only by asking. It runs two fresh ~200-day
  simulations (about 40 s on a Raspberry Pi), so it is not for every
  step.
* **Read-only.** It changes nothing, not a digest, and draws nothing
  from the session: the session goes on exactly as one never evaluated.
* **Where it is defined:** for the `sustainable` scenario only. Sessions
  saved by V0.2.0 are evaluated with their earlier decisions. A
  refusal is a stable code: `t90_unsupported_scenario`,
  `t90_earlier_rules`, `t90_not_started` or `t90_sealed_seed`.

**`apply` is atomic.** The batch is validated first and applied only if
every action is valid. A refused batch changes nothing: not the state,
not the action record, not a digest. Each action's result says why it
was refused. A batch is 1–64 actions, applied at the current time:

* **V0.2 staffing actions** (`hire`, `terminate`, `set_roster`,
  `set_reading_allocation`, `contract_teleradiology`) form one atomic
  unit and cannot share a batch with V0.1 actions (`mixed_batch`).
* **V0.1 actions** (`extend_shift`, `set_walkin_policy`,
  `set_worklist_policy`) are checked one by one, so two extensions for
  the same employee and day must go in separate batches
  (`duplicate_target`).

A saved session records the applied batches. Reopening it
(`-load`, `dreamteam replay`, `dreamteam verify`) replays them and checks
every digest.

## 4. The action catalog

`actions` returns, for every action type:

* `schema`: a JSON Schema of the action object (the `api.Action` fields
  that type takes, `additionalProperties: false`);
* `constraints`: the rules in words;
* `targets`: what the action can name now, read from the session's own
  views:

| Type | Targets |
|---|---|
| `extend_shift` | active hourly employees, today's shift, approved minutes, cap |
| `set_walkin_policy`, `set_worklist_policy` | the policies in force |
| `hire` | the fixed offer table (role, skills, standard and premium rates and leads, recruiting fee), the start-date window, the rooms, current headcount and caps |
| `terminate` | employees who can be given notice, the earliest last day, severance |
| `set_roster` | editable roster plans: versions with their rosters, earliest effective Monday, contract minutes |
| `set_reading_allocation` | editable reading allocations: in effect, bounds, earliest effective date, versions |
| `contract_teleradiology` | the plan catalog, the earliest activation, the contracts that can be cancelled with their earliest end dates |

The catalog says what to send; the simulator decides what is accepted
(`validate`).

## 5. Error codes

Protocol errors (`ok: false`). A refused action is not one: it is a
result with the simulator's code (for example `notice_too_short`,
`allocation_out_of_band`, `invalid_roster`, `mixed_batch`).

| Code | When |
|---|---|
| `invalid_json` | the line is not exactly one JSON value |
| `invalid_request` | not an object, an unknown field, no `op`, an `id` that is not a string or a number |
| `unknown_op` | an operation not in §3 |
| `invalid_args` | arguments missing, unknown, or out of range (`field` names the argument, `index` the action) |
| `duplicate_target` | two `extend_shift` for one employee and day in one batch |
| `not_previewable` | `preview` of an action that has no preview (the V0.1 actions; `validate` checks them) |
| `save_failed` | the file could not be written |
| `unsavable` | the session has no scenario file to record |
| `apply_diverged` | a validated batch was refused on apply (never expected; it would be a defect) |
| `unknown_section` | `observe` named a section that does not exist |
| `t90_unsupported_scenario` | `t90` on a scenario the evaluation policy is not defined for (it is defined for `sustainable`) |
| `t90_earlier_rules` | `t90` on a session continuing one recorded under earlier action rules |
| `t90_not_started` | `t90` before play has started |
| `t90_sealed_seed` | `t90` on a seed of the sealed evaluation holdout |

A failed request changes nothing, and the session continues.

## 6. What an agent can see

Exactly what a player can see:

* the observation;
* the action catalog's targets, all read from the same views (the
  observation, roster plans, reading allocations, teleradiology and the
  published offer table);
* validation and preview results, the same as the game's staff screen;
* digests.

There are no future events, random streams, latent patient attributes
or evaluator internals. The simulator's leakage tests (same past,
different future: identical observations and previews) cover these
views. The agent's tests check that its observation is the session's,
byte for byte, and that catalog targets name only what the observation
shows.

## 7. A scripted runner

```
dreamteam agent -policy noop|greedy [-days N] [-scenario ID|FILE] [-seed N] [-transcript FILE] [-save FILE]
```

It runs a policy through the same protocol in one process: `hello`, the
policy's decisions at the start and each day at 07:00 (each batch
applied only if valid), and `advance` to the next 07:00. It prints a
summary (JSON) to stdout. It is an interface check, not a strategy:

* `noop` never decides;
* `greedy`, when the oldest unread study is more than a day old, raises
  the first editable radiologist's site reading allocation by 5 points
  from the earliest date. It reads only the observation and the catalog.

## 8. Example

A short exchange (responses abridged with `…`):

```
→ {"id":1,"op":"hello"}
← {"protocol":"dreamteam.agent/v1","id":1,"ok":true,"result":{"protocol":"dreamteam.agent/v1","engine":"0.3.0-dev.1","scenario":{"ref":"sustainable","id":"established","name":"DreamTeam Imaging Center","fingerprint":"e8718dbd…"},"seed":908,"ops":["hello","observe",…],"clock":{"time":"2026-10-13T06:00:00","day":1,"logTotal":70179,"digests":{…}}}}
→ {"id":5,"op":"apply","args":{"actions":[{"type":"set_reading_allocation","op":"set","employee":"E07","effective":"2026-10-27","siteAllocationBP":100}]}}
← {"protocol":"dreamteam.agent/v1","id":5,"ok":true,"result":{"applied":false,"results":[{…,"ok":false,"code":"allocation_out_of_band","message":"siteAllocationBP must be 3000–10000 (30%–100% of paid time), not 100."}],"clock":{…}}}
→ {"id":8,"op":"apply","args":{"actions":[{"type":"set_reading_allocation","op":"set","employee":"E07","effective":"2026-10-27","siteAllocationBP":6500}]}}
← {"protocol":"dreamteam.agent/v1","id":8,"ok":true,"result":{"applied":true,"results":[{"id":"X00001","batch":1,…,"ok":true,"code":"accepted",…}],"clock":{…}}}
→ {"id":9,"op":"advance","args":{"to":"2026-10-14T07:00:00"}}
← {"protocol":"dreamteam.agent/v1","id":9,"ok":true,"result":{"reason":"reached","clock":{"time":"2026-10-14T07:00:00","day":2,…}}}
→ {"id":11,"op":"advance","args":{"minutes":0}}
← {"protocol":"dreamteam.agent/v1","id":11,"ok":false,"error":{"code":"invalid_args","message":"minutes must be 1–527040, not 0","field":"minutes"}}
→ not json
← {"protocol":"dreamteam.agent/v1","id":null,"ok":false,"error":{"code":"invalid_json","message":"the request is not one JSON value"}}
```

The complete 30-day no-op transcript (the sustainable scenario, seed
909: 31 requests and the save, every request and response) is
`docs/agent/noop-30d.transcript.jsonl`. The greedy policy's 30-day
transcript (73 requests, 3.7 MB because it observes each day) is
regenerated byte for byte by:

```
dreamteam agent -policy greedy -days 30 -scenario sustainable -seed 909 \
  -transcript greedy.transcript.jsonl -save greedy.session.json
```

Its SHA-256 is `5bf803a2ca8005f929b7ddeaa18abaa5de8a7582ff0a70f683baf13e124e451b`,
and the session it saves verifies with run digest `da6a61c3…0d`.

## 9. The reference Python client

`clients/python/` is a standard-library client: `dreamteam_agent.py`
(the `AgentClient` class: launches `dreamteam agent --stdio`, sends
requests with stable ids, checks every response's protocol and id, raises
`ProtocolError` with the stable code) and `example_session.py`, a complete
session:

```
go build -o dreamteam ./cmd/dreamteam
python3 clients/python/example_session.py --binary ./dreamteam --days 5 --t90 --transcript t.jsonl
```

It observes, validates, previews, applies, advances, saves and requests
T90, then checks the save with `dreamteam verify` and `dreamteam t90`.
Its transcript is byte-identical run to run and across linux/arm64 and
linux/amd64 (CI compares it on linux/amd64, linux/arm64 and darwin/arm64).

## 10. Compatibility and determinism

* **Unchanged requests, unchanged responses.** A V0.3a request gets the
  same response, byte for byte, except `hello`, whose `engine` and `ops`
  name the build (`TestLegacyTranscript` replays the committed V0.3a
  transcript).
* **Platforms.** The evaluation's reported statistics (design effects,
  interval bounds, dispersion) round every product before a sum, so they
  agree across processors. Before 0.3.0-dev.2 they could differ in the
  last digit or two between arm64 and amd64, with every status and
  verdict the same. A `t90` response and the reference client's
  transcript are byte-identical on linux/arm64 and linux/amd64, and CI
  compares the transcript on darwin/arm64 too. One known exception
  remains outside the protocol's T90: a multi-seed evaluation's exact
  Clopper–Pearson bound on the joint service gate goes through the Go
  math library and can differ in its last digit between processor
  families.

## 11. Not in V0.3

No HTTP or WebSocket transport, no model or LLM integration, no benchmark
scoring or leaderboard, no new actions.
