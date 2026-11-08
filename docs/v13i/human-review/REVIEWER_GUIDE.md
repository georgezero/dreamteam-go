# V0.13i limited human presentation review — reviewer guide

HUMAN REVIEW PENDING. No reviewer has been assigned. Nothing in this
directory records a human finding or approval.

## What you are asked to judge

Only the **presentation neutrality and clarity** of the complete
model-visible interactions listed in [MANIFEST.json](MANIFEST.json): 93
transcripts and the 7 active static assets they are built from.

You are **not** asked to approve, and your review cannot be taken as approval
of:

* the run-recording system, run lifecycle, partial-run handling or verifier
  (that technical work is reviewed separately, by an independent technical
  re-review);
* the experimental protocol, schedule, estimand or analysis plan;
* any live inference.

Final protocol acceptance requires both this human review and the separate
technical review to pass. Protocol lock and live inference remain NO GO.

## Exact material

All files are in the repository at commit
`a4d7204c14f36977fee044b3d1a17518d58cc02c` (or any later commit at which
`python3 -I -B analysis/alternatives/v13i_human_review.py --check` passes,
which proves the listed bytes are unchanged).

* Transcripts: `docs/v13h/review-package/transcripts/001.json` … `093.json`.
  Each contains every delivered request message (system instruction, public
  context, proposal, engine feedback, comparison) and the scripted reply.
  Every reply is labelled SCRIPTED-MODEL OUTPUT; no model was sampled.
* Static assets: `docs/v13h/review-package/sources/ACTIVE/` (system
  instruction, initial and correction wrappers, response schemas,
  presentation template, inspection view table, selection text).
* Frozen identity: [MANIFEST.json](MANIFEST.json) lists the path and SHA-256
  of each of these 100 files; [MANIFEST.sha256](MANIFEST.sha256) can be checked
  from the repository root with `sha256sum -c docs/v13i/human-review/MANIFEST.sha256`.
* Navigation only (not model-visible): `docs/v13h/review-package/RECORD_ONLY_INDEX.json`
  names each transcript's case, context, arm and outcome label.
* `docs/v13h/review-package/FAILURE_FEEDBACK_DISCLOSURE.md` explains the one
  disclosed invalid-number retry message.

Ignore `RECORD_ONLY_LIFECYCLE/` and `RECORD_ONLY_PARTIALS/` in that package:
they are run-recording examples, outside this review.

If any listed byte changes later, every affected transcript or asset must be
reviewed again; the check command above fails when that happens.

## Suggested reading order

| Topic | Where to look |
|---|---|
| Control and treatment arms, same context | 070 (S1-a control) and 071 (S1-a treatment); 052/053, 060/061, 066/067 for historical contexts. The control arm sees validation and the shared head and foot only; the treatment arm also sees the verified comparison and one inspection round. |
| Inspect details | 071 (inspection offered), 028 (second inspect counted as a failure), 031 (inspect request in control) |
| No-action obligations | 012 (no action selected), 023 (mixed operations and staffing, no action), 011 ("Obligations that continue" and continuing employment commitments relative to no action) |
| Capacity versus displacement | 071, 073, 091, 002, 004: "Slots closed: 2,481 slots … Capacity, not patients, orders or revenue" next to "Bookings displaced: 0 bookings". These are distinct quantities. |
| Reversibility and withdrawal wording | 003 (roster capacity and withdrawal), 002 (partial termination commitments), 047/053 (termination with displaced bookings) |
| Teleradiology fee context | 004 and 061 ("Historical remote fee proxy … from N relevant studies; requested trailing lookback 28 days … Incomplete lookback: true"); 049 (teleradiology guarantee); 043/044 (retainer refused as not funded); retainer-to-earliest-end lines in 055, 057, 059 |
| Disclosed invalid-number message | 001: "Your reply could not be used: action number must be an in-range integer. Reply again with one JSON object …". It is a general failure-path example, not shown on the fixed schedule. |
| Refusal, correction and retry paths | 007, 008, 026, 027, 029, 030, 035–042 |
| Hostile free text in the scripted reply | 021 (an invented attack string supplied as test input) |

Some transcripts are byte-identical duplicates (the predecessor package kept
multiplicities); MANIFEST.json lists each file separately.

## The 12-item questionnaire

Answer each item in [REVIEW_RECORD.md](REVIEW_RECORD.md) with written
findings. For each, judge whether the wording is neutral (does not steer
toward or away from any alternative, including no action), understandable,
and complete enough for the decision shown.

1. Neutrality
2. Readability
3. Financial terminology
4. Target identification (which employee, contract or room an action names)
5. Continuing obligations and no action
6. Bookings displaced
7. Slots closed and replacement capacity
8. Missing information (what is stated as not known before deciding)
9. Withdrawal and reversibility
10. Potentially misleading comparisons
11. Information boundary (nothing evaluation-only appears in a model-visible message)
12. Declared treatment bundle and complete interactions

## Recording the review

Fill in [REVIEW_RECORD.md](REVIEW_RECORD.md) — reviewer identity, an
independence declaration (no part in designing the simulator, presentation or
experiment), the date, the exact commit and MANIFEST.json SHA-256 reviewed,
findings per item and a verdict on presentation neutrality and clarity only.
Return it separately; it is committed verbatim. Until then the status remains
HUMAN REVIEW PENDING.
