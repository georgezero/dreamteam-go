# V0.13d known limitations

## Process

* **The human presentation review is PENDING.** No human reviewer was
  available. The protocol cannot be locked, and no model may be shown the
  texts, until it is done (PRESENTATION_REVIEW.md).
* **No live inference.** Nothing here measures any model's behaviour.
  Every result in `docs/benchmark/v13d-dev/` comes from the engine, from
  scripted replies, or from recorded V0.12b replies served as scripts.
* **The budget is a proposal.** The 262-draw design (PROTOCOL_DRAFT.md §4)
  needs the owner's approval.

## Information boundary

* **The model still sees its own reply.** The turn follows the
  conversation, which holds the model's proposal with its own free text
  (names, request IDs, reasons). The turn repeats none of it, but the
  model can read its own earlier messages. That is the model's own text.
* **The view drops information.** Validation messages, the comparison's
  descriptions, sources, derivations and the engine's own bases are not
  shown; fixed texts written for the view replace the bases. Whether
  anything a manager needs is now missing is item 9 of the human review.
* **Unsupported content fails closed, for both arms.** A comparison the
  view does not define (an unknown money line, an action field outside the
  allowed adjustments) gives no turn in either arm
  (`not_due:not_representable`). Every protocol context has a view; a
  model's correction at a refused point could still produce one that has
  not, and the arms would then be identical at that draw.
* **Engine labels are matched, not parsed.** A money line is resolved by
  comparing its label with the engine's label of one of the batch's own
  targets (a name and ID, an offer label, a contract ID). If the engine's
  label format changes, the view fails closed rather than showing it.

## Design

* **The arms differ in more than the comparison's content.** Treatment may
  spend one call on inspect, and lists more IDs a reply may select. The
  feet differ in that one line. Calls are counted, never equalized.
* **Control cannot select another alternative or no action,** because it
  lists only the proposal. The arms' outcome sets differ by design.
* **A revision with only the reply's operations actions** is classified as
  `revised`, not as `selected_no_action`.
* **`compared` needs the model to cite keys.** The foot describes an
  optional `cite` field in both arms. Whether models use it is unknown.
* **The control turn is a confirmation turn** that no earlier experiment
  had (V0.13c D1). Control is not llm-agent 5 unchanged.
* **Strict reading.** A reply wrapped in a code fence, or with any extra
  field, is unreadable and uses the retry. Models answering in
  `json_object` mode are not expected to do either, but it is possible.

## Contexts and statistics

* **Few contexts:** 9 historical (5 due at the recorded proposal, 2 only
  after a correction, 2 A/A) and 12 constructed members.
* **The constructed contexts are scripted** and start with an empty memory.
* **The pairs do not isolate one fact** (SENSITIVITY_CASES.md).
* **Repeat variation at temperature 0 is small** on this server, so the
  draws estimate rates that are mostly 0 or 1. Intervals describe that
  repeat variation, not a population.

## Engine and versions

* **The engine version is unchanged (0.13.0-dev.2).** V0.13d adds an
  opt-in operation to `dreamteam agent --stdio` (`-alternatives`), one
  field to the agent handler, and a new package file set in
  `internal/alternatives`. No simulator, session, validation, preview,
  accounting or randomness file changed (`docs/audits/v0.13d-baseline.md`
  and the after-hashes in the implementation report). Bumping the version
  would change the engine string in V0.13b and V0.13c's committed evidence,
  which must stay byte for byte. The lock must decide the version the run
  pins (PROTOCOL_DRAFT.md §10).
* **The extension ops are not listed by `hello`.** This keeps every
  historical replay identical, but a client cannot discover them; they are
  documented in `docs/AGENT_PROTOCOL.md` and served only with
  `-alternatives`.
* **V0.13c's turn (`Steps`, presentation v1, control v1) is kept** and
  still tested, but nothing uses it for the experiment any more.
