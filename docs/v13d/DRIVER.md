# V0.13d experiment driver: `dreamteam-v13d-driver/1`

**Code:** `analysis/alternatives/v13d.py`; scripted fidelity:
`analysis/alternatives/v13d_fidelity.py`; tests:
`analysis/alternatives/test_v13d.py`. It reuses V0.12b Stage B's harness,
limits, model wrapper and replay transport (`analysis/acknowledgment/stage_b.py`),
unchanged.

**No command contacts a model.** `run` refuses any protocol whose status
is not `LOCKED — NOT RUN`, and any run without the owner's authorization
text; the draft protocol is `DRAFT — NOT LOCKED — NOT AUTHORIZED`.

## 1. Contexts

| Kind | What a draw starts from | Checked |
|---|---|---|
| historical (9: the V0.13c selection from the V0.12b pool) | a fresh engine (`dreamteam agent --stdio -alternatives`) on the recording's replay, the recorded requests fed and every answer checked, up to the recorded digests; the recorded memory text; the recorded request; the recorded proposal | the digests; the memory against the recording's `memorySha256`; the committed source line's SHA-256; the request, byte for byte |
| constructed (12: the six V0.13c sensitivity pairs, both members) | a fresh engine on mini, seed 52, advanced with no action to the member's time; a fresh memory (`llm_agent3.Memory`, empty, as at a run's first decision); the agent's own prompt (`llm_agent3.prepare`); the scripted proposal | the digests; the prompt's SHA-256; the request, byte for byte |

`contexts.jsonl` (`docs/benchmark/v13d-dev/contexts/`) holds both kinds, in
that order. `contexts --check` makes it again and requires it identical.

## 2. One draw

A draw is one schedule entry: context, model, arm and draw number.
1. **The engine.** One engine process per context and arm, opened at the
   arm's first draw and kept for its later draws. The arms never share a
   process. Every request a draw makes there is read-only (`validate`,
   `preview`, `observe`, `actions`, `digests`, `alternatives.turn`,
   `alternatives.decide`); the agent handler checks the session's digests
   unchanged around every extension op. The draw starts only if the
   digests are the context's.
2. **The settings.** The model's frozen settings file
   (`docs/benchmark/v05c-dev/llm-*.json`), loaded by the agent's own
   `load_settings`, unchanged: temperature 0, `max_tokens` 4096,
   `json_object`, timeout 300 s, `maxRetries` 1, details allowed.
3. **The workflow.** `llm_agent_alternatives.ask(..., arm=...)`.
4. **The model** (`ContextModel`, V0.12b's `PointModel` at a context):
   * the first request must equal the context's request, byte for byte;
     it is not sent, and its reply is the initial proposal;
   * every later request is checked against the limits, then sent to the
     transport, and recorded: the request's SHA-256, status, error, the full
     response body and its SHA-256, usage, finish reason, served model,
     send time and latency.
5. **The record** (`dreamteam.alternatives-draw/v1`): the entry, status
   (`completed` or `interrupted`), halt, initial digests, the workflow's
   record, the submitted batches, every engine request and answer, the
   final digests, every model call.

**The same at every draw of a context, in both arms:** the state (by
digests), the initial request (byte for byte), the initial proposal (by
SHA-256), the settings, the memory, the decision cadence (one decision
point) and every instruction before the turn. The turn's head and foot are
the same text. What differs, and stays visible: the comparison, the
inspect round it allows, and the calls that follow. Calls are counted per
draw and never equalized.

## 3. Limits and stopping

* **Per draw:** at most 6 model calls (a guard). The most a draw can
  make after its initial proposal is 3 at a valid proposal (the turn's
  reply, the reply after inspect, one retry) and 5 at a refused one (one
  correction and its retry first).
* **Per run:** the protocol's total, which is the sum of each scheduled
  draw's bound; the wall clock (a request starts only if its whole
  timeout fits); 5 consecutive transport failures; an unhealthy server or
  a material server change before a round.
* **A limit reached during a draw** interrupts it (`Halt`, which no agent
  catches). The draw is recorded as interrupted and the run stops.
* **Any other exception** stops the run. `run.json` is still written, and
  the unfinished draw goes to `live/aborted-draw.json`.
* **`run.json`** (`dreamteam.alternatives-run/v1`): COMPLETE only if every
  scheduled draw completed with no stop; otherwise PARTIAL.
* **No resumption.** A directory that holds a run is refused.

## 4. Replay and analysis

`analyze` (`v13d.analyze`):
* requires the recorded draws to be the schedule's first draws, in order,
  each once, with only the last unfinished;
* replays every draw without a model: the same engine, the agent's own
  code, each recorded response served only to the request with the
  recorded SHA-256 and only if its body still has its SHA-256, every
  recorded response used, and everything but the live metadata identical;
* classifies each completed draw (WORKFLOW.md §5);
* **executes** each distinct submitted set of batches once per context:
  a fresh engine at the context, the batches applied in order with the
  engine's own `apply`, each answer recorded (applied, or the error),
  partial application recorded, then the end of the day reached with no
  further action, and the digests after both;
* tabulates, per context, model and arm: every result, presented,
  inspected, compared, final valid, submitted, executed, partial, model
  calls and retries;
* computes the arm differences and the pair estimates (PROTOCOL_DRAFT.md
  §6), only for a COMPLETE run. A PARTIAL run reports what was observed,
  with no pooled summary.

## 5. Schedule (`dreamteam-v13d-order/1`)

Per model in the protocol's order, rounds `1..n`. Each round has every unit
(context and arm) of the model once, ordered by
`SHA-256("dreamteam-v13d-order/1|<model>|<round>|<context>|<arm>")`. A unit
takes part in round `r` only if `r` is within its draw count (A/A
contexts have fewer). Historical contexts belong to the model that made
them; constructed contexts are run with both models. So the arms are
interleaved within every round, and the order is fixed by the protocol
and the contexts alone.
