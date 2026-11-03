# V0.12b Stage B: the driver (`dreamteam-v12b-driver/1`)

**Source:** `analysis/acknowledgment/stage_b.py`. **Fidelity evidence:**
`analysis/acknowledgment/stage_b_fidelity.py`, with its outputs in
`fidelity/`. **Tests:** `analysis/acknowledgment/test_stage_b.py`.

The driver runs the protocol's 420 draws (`PROTOCOL.md`). It records each
draw, replays every recorded draw without a model, and analyses the
outcomes. It does not contact a model unless the `run` command is given
an endpoint and an owner authorization. The lock does not authorize a run.

## 1. How a draw runs

A draw is one scheduled entry: point, model, arm and draw number. Each
entry also carries what the draw must start from: the SHA-256 of the
point's state digests, of the initial request and of the initial proposal.

1. **The engine.**
   * The draw gets the engine session of its point and arm.
   * Each point and arm has its own `dt agent --stdio` process, opened at
     the arm's first draw at the point, on the recording's replay. The
     recorded requests are fed and every answer is checked
     (`Session.to_point`).
   * The arms never share a process.
   * The draw first asks for the digests and refuses to start unless their
     SHA-256 is the schedule's.
   * At its end it records the digests again.
2. **The harness.** The agent's interface, `request(op, args)`. It:
   * returns the result;
   * raises the agent's own `ProtocolError` on a refusal, as
     `llm_agent3.Harness.request` does;
   * uses the agent's request ids `r1, r2, …`;
   * reads the engine's lines with the arm's agent's own reader:
     `json.loads` for A, and `llm_agent6.json_loads` for B (duplicate keys
     marked, NaN and Infinity refused);
   * logs every request and answer.
3. **The settings.** The model's frozen settings file
   (`docs/benchmark/v05c-dev/llm-*.json`, pinned by SHA-256), loaded by
   the agent's own `load_settings`:
   * temperature 0, `max_tokens` 4096, json_object;
   * timeout 300 s, `maxRetries` 1, `allowDetails`;
   * decision budget 900 s;
   * the built-in system prompt.

   Nothing is overridden.
4. **The memory.** The memory text at the point is the recording's own,
   from its replay's `memory.jsonl`. Its SHA-256 is checked against the
   recording's `memorySha256`, and it is given to the agent's `prepare()`
   unchanged.
5. **The workflow.** The arm's agent's own `ask()`:
   * A: `llm_agent5.ask(..., workflow="validation-first-preview")`;
   * B: `llm_agent6.ask(...)`.

   The driver does not reimplement any step. Validation, feedback, preview,
   eligibility, assessment, the request and rejection texts, matching,
   retries, the final check and withholding are all the agent's own code.
6. **The model** (`PointModel`, the agent's model interface):
   * **Payloads:** `payload()` is the agent's own `ModelClient.payload()`.
   * **The first request:** the agent's first request must equal the
     recorded request byte for byte, as `ModelClient.call` would send it
     (`json.dumps(payload)`), against the committed source context, which
     keeps the client's key order. Since the prompt is rebuilt from the
     engine's observation, the catalog and the memory, this checks all
     three. The first request is not sent; its reply is the recorded
     proposal.
   * **Every later request:**
     1. the limits are checked (§3);
     2. the request goes to the transport (live: the agent's own
        `ModelClient.call`);
     3. it is recorded: request SHA-256, status, error, the full response
        body and its SHA-256, usage, finish reason, served model, send time
        and latency.
   * **`spent()`:** `llm_agent6.LiveModel.spent`, unchanged.

A draw's record (`dreamteam.acknowledgment-draw/v1`) contains:
* the schedule entry, the status (`completed` or `interrupted`) and the
  halt;
* the initial and final digests;
* the agent's own decision record, exactly as `ask()` returned it (v5 or
  v6);
* the submitted batches;
* every engine request and answer;
* every model call.

## 2. Same start, different calls

At every draw of a point, both arms get the same four things:
* the same state, checked by digests;
* the same request, checked byte for byte;
* the same proposal, checked by SHA-256;
* the same settings and memory.

What follows differs, and is kept visible:
* **Model calls** are counted per draw (`modelCalls`, `callsPerDraw`), per
  arm and per point. They are never equalized.
* **The extra request:** at a valid qualifying proposal, A may show a
  preview and B shows the acknowledgment request. At a valid
  non-qualifying proposal, A may show a preview while B calls no model.
  The tables print the calls next to the outcomes, so a difference in
  calls is not read as an acknowledgment effect.

**The bound on calls:** with `maxRetries` 1, neither workflow can make
more than **2 model calls** after the recorded proposal.
* Every path to a model call closes the details option first.
* The first unaccepted reply uses the only retry.
* `test_stage_b.CallBound` runs every sequence of three reply kinds at
  five kinds of start and finds 2 as the maximum.

The protocol's limit of 4 per draw is a guard that should never bind. If
it does, the run stops.

## 3. Limits and stopping

Before every model request (`Limits.before_call`), in this order:
* **The draw's calls.** 4 already made in the draw: `draw_call_limit`.
* **The run's calls.** 1,680 already made: `call_budget`.
* **The wall clock.** If the elapsed time plus the request's 300 s
  timeout would pass 14,400 s: `wall_clock`.
  * A request starts only if its whole timeout fits.
  * No request is cut short.
  * The run cannot overrun four hours by a request.

Before every draw (`Limits.before_draw`), the same wall-clock test is
applied.

The other stop conditions, as in V0.11b:
* 5 consecutive transport failures;
* `/health` not healthy before a round;
* a material server change before a round.

**When a limit is hit:**
* **During a draw:** a `Halt` is raised. It is neither a `ValueError` nor
  a `ProtocolError`, so no agent catches it. The draw is recorded as
  `interrupted`, with the calls it made and no decision record. The run
  stops.
* **On any other exception:** the run stops, records `drawFailed`, and
  still writes `run.json`.
* **Ctrl-C:** the run stops, as `operator`.

**`run.json`** (`dreamteam.acknowledgment-run/v1`):
* `status`: **COMPLETE** only if all 420 draws completed with no stop;
  otherwise **PARTIAL**;
* the completed and not-run counts, the interrupted draw, the stop code
  and message;
* the owner's authorization text, verbatim.

**No resumption:** `run` refuses a directory that already holds a run.
The driver has no command to resume, extend or fill in a run.

## 4. Replay

`analyze` replays every recorded draw, completed or interrupted, without a
model:
* the same pooled sessions, in the recorded order;
* the agent's own `ask()`;
* each recorded response served only to the request with the recorded
  SHA-256, and only if its body still has its SHA-256;
* an interrupted draw halted at the same call.

Everything deterministic must be identical (`deterministic()`):
* the schedule entry and status;
* the digests;
* the agent's decision record (every validation, preview, assessment,
  payload and token, acknowledgment match and final check);
* the batches;
* every engine request and answer;
* each call's request hash, status, error and response hash.

Every recorded response must be used. Send times and latencies are live
metadata. They are excluded from the comparison and reported only in
`inference.json`.

Before replaying, the recorded draws must be the schedule's first draws:
* in order and each once;
* each with the schedule's point, model, arm, draw and expected start;
* with only the last allowed to be unfinished.

A missing, duplicated, unlisted or reordered draw is refused, and so is a
wrong arm or a wrong expected start.

## 5. Classification

Every completed draw is classified. The definitions are in `PROTOCOL.md`
§5.

**The proposal:**
* kept, revised or abandoned (llm-agent 4's rule, against the initial
  proposal);
* or nothing submitted:
  * **refused:** the last attempt was refused;
  * **withheld:** B's fail-closed withholding;
  * **invalid:** an unreadable reply or a failed request.

**The final proposal:**
* whether it was valid;
* whether it was submitted (or `"actions": []`);
* whether it was executed (applied at the recorded state in the
  continuation).

**Arm B's class:**
* **A proceeded, B revised, C abandoned, D withheld:** these apply from
  the first proposal whose consequences qualified (a request shown) or
  could not be read (withheld, fail-closed).
* **E:** acknowledgment was never required.
* **Two qualifiers:**
  * whether a reply that changed the actions had first stated the
    request's consequences exactly ("accurate before a change", checked
    with `llm_agent6.match`);
  * every acknowledgment failure code.

**Arm A:** the selective preview's status, and what followed a shown one:
proceeded, revised, abandoned or failed.

**Also recorded:**
* retries, model calls and the reply kinds the agent sent;
* tokens, as the server reports them.

## 6. Continuation and financial evidence

**What runs:** for each point, each distinct final submitted decision and
the initial proposal are applied at the recorded state. This is
shared-prefix/v2's `decision_evidence`, unchanged: validate, apply, then
the no-action suffix. Financial evidence v1 follows.

**What is reported is only what the engine reports:**
* commitment and credit support;
* severance (the previews' `terminations[].severanceCents`);
* retained obligations (the previews' `guaranteed` lines, label, cents
  and timing);
* displaced bookings;
* direct capacity changes;
* reversibility;
* the apply outcome;
* the digests after the decision and after the suffix;
* the ledger's SHA-256.

Nothing is inferred, and no later outcome is attributed to a decision.

## 7. What a PARTIAL run reports

**What it includes:**
* every completed draw, replayed and classified;
* the counts per point and arm, with the observed and scheduled numbers
  side by side.

**What it leaves out:** B − A comparisons and pooled summaries are not
computed (`BminusA: null`, `pooled: null`).

**How the tables mark it:** they open with the status and coverage, and
state that no conclusion about the complete experiment is drawn.

**Continuing:** this needs the owner's separate authorization and a
documented protocol decision (`PROCEDURE.md`).

## 8. Known limitations

* **Session reuse:**
  * Sessions are reused per point and arm, because a fresh engine per
    draw costs 5–36 s on the Pi and would not fit the four hours.
  * The state is checked at every draw's start and recorded at its end.
  * Reuse rests on the agents' requests at a point being read-only, which
    the digests confirm.
* **The memory text** comes from the recording's replay. It is not
  rebuilt from the run's earlier points; its SHA-256 and the rebuilt
  prompt's bytes are checked instead.
* **The decision budget:** with at most 2 calls of at most 300 s, it
  cannot be reached. If it ever were, the record would show it, and
  replay, which has no wall clock, would refuse that draw.
* **Fidelity is tested, not proved:**
  * against the agents' own runs on the real engine (`fidelity/`), with
    scripted replies;
  * on the agents' scripted engines, for every V0.12a scripted case and
    V0.11b's arm C cases.

  A live reply of a kind not covered there still runs through the agents'
  own code.
