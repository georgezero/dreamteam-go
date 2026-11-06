# V0.13d scripted fidelity evidence

**Evidence:** `docs/benchmark/v13d-dev/fidelity/fidelity.json` and
`fidelity.md`, made by `analysis/alternatives/v13d_fidelity.py fidelity`
with the V0.8c and V0.9c replays, and required identical in CI.
**Scripted models only.** No model was contacted. Where a historical case
needs a model's correction, the reply is V0.12b Stage B's recorded live
reply at that point (arm A, draw 1), served as a script.

**Result: 59 cases, 0 failed checks.**

## 1. What every case checks

Every case runs one draw through the driver's own `run_draw`, then:
* **state digest equality:** the engine is at the context's digests when
  the draw starts;
* **initial request equality:** the first request equals the context's,
  byte for byte (so it is equal across arms);
* **initial proposal equality:** its reply is the context's proposal;
* **message fidelity:** every message sent after the initial proposal is
  an engine text, verbatim (the turn's message, an inspect or retry
  answer), or validation-first's own feedback;
* **control and treatment message fidelity:** the turn the workflow sent is
  the one a fresh engine makes for the same reply, its hash is the
  record's, and the other arm's turn has the same head and the same foot
  apart from the IDs line; control shows no alternatives;
* **inspect fidelity:** the inspect answer is the turn's details, in the
  order asked;
* **the result, the model calls, the retries and the batches** are the
  case's;
* **recording completeness:** every call recorded, and every exchange the
  engine answered logged;
* **model-free replay:** the draw replays without a model, identical;
* **deterministic continuation:** the submitted batches, applied in a
  fresh engine at the context and run to the end of the day, give the
  same digests twice, and are executed in full.

## 2. Cases

| Required case | Where |
|---|---|
| Original retained | S1-a, both arms; historical termination with displacement, both arms; the refused Gemma point after its correction (control) |
| Alternative selected | S1-a after inspect; the historical hire with an operations action; the historical teleradiology guarantee |
| No action selected | S1-a; F-mixed (operations kept); the refused Gemma point after its correction, after inspect; the historical staffing adjustment |
| Revision | S1-a, both arms; F-mixed (the whole reply replaced) |
| Abandonment | S1-a, both arms; F-mixed; the historical hire |
| Invalid selection | an unlisted ID in treatment; a non-proposal ID in control |
| Missing comparison | F-rule (the rule does not apply), both arms: no turn, validation-first's batch submitted; the historical A/A point |
| Tampered comparison | the turn message edited in a recorded draw; a recorded response edited: neither replays |
| Stale comparison | a recorded draw replayed at another member's state: refused |
| Malicious free text | F-hostile: a hostile `requestId`; no message the engine made contains it |
| Mixed operations and staffing | F-mixed: select keeps the operations batch first, unchanged; no action keeps it; revise replaces it; abandon drops it. The historical hire: a corrected reply with `extend_shift` and `hire`, select adjusted, two batches |
| Refused initial proposal | F-refused (one day's notice) corrected by script; two historical refused points corrected by their recorded V0.12b replies |
| Missing engine fields | the turn answer without its message: the draw aborts, nothing submitted, no model call |
| Retry exhaustion | two unreadable replies: `no_decision`, nothing submitted |
| Model timeout | a timeout then a decision (one retry used); two timeouts: `no_decision` |
| Partial experiment | the mock run with a call budget of 3: PARTIAL, the interrupted draw replayed, no pooled summary |

Also: a second inspect request counts as a failure; an inspect in control
gets the fixed answer; every one of the 21 protocol contexts starts in both
arms with the same state and turn test, the turn due exactly where V0.13c
declared; and a complete mock run (S1 and S6 members, both arms, both
models: 18 draws) is analysed.

## 3. Go-level fidelity

`internal/alternatives/turn_test.go`: arm symmetry, the due test against
V0.13c's, hostile names and request IDs, twelve fail-closed edits, every
outcome in both arms, the operations batch, citations, strict reading, and
the protocol operations (with `hello` unchanged). DecodeAgentAt's binding to
the decision point (a stale or edited comparison is refused) is V0.13b's,
unchanged and still tested.
