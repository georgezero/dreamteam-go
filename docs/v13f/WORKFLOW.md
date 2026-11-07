# V0.13f workflow contract — llm-agent-alternatives-workflow/v2

Implementation: agents/llm/llm_agent_alternatives_v2.py.
The shared instruction is llm-agent-alternatives-system/v2. Its schemas and
precedence apply identically to both arms. All example responses are scripted.
Independent re-review: PENDING. Human presentation review: PENDING.

| Phase | Delivered template and accepted responses |
|---|---|
| 1 Initial proposal | initial-v2.txt wraps the existing public-context renderer; reason/actions, or one allowed public-details request |
| 2 Validation correction | correction-v2.txt includes refused engine results verbatim; reason/actions only, including deliberate empty correction |
| 3 Alternative inspection | presentation-v3.tmpl head/comparison/foot; inspect once, or any final decision kind |
| 4 Final decision | presentation-v3.tmpl foot and inspect/retry answers; select, revise, abandon, or the system-authorized reason/actions whole-reply revision |

The system schema governs conflicts with lower-priority templates. A
system-compliant reason/actions final reply is accepted as revision, even
though the lower-priority foot illustrates decision/select/revise/abandon.
Empty revisions are valid deliberate no action. Duplicate keys and unknown
or case-aliased fields fail strict parsing. Exact action fields remain the
engine catalog's responsibility; simulator validation is unchanged.

The validation-first control flow reuses the established batching and one
failure allowance, in a successor implementation with new phase templates
and record identities. A refused replayed proposal consumes that allowance;
one correction can follow, without public details. A failed correction is
not retried. Initial proposals are supplied from frozen recordings or the
frozen sensitivity scripts, without an inference call.

If the validated reply has actions, alternatives.v2.turn rebuilds selection
v2 and the bound comparison, then makes view v2 and presentation v3. It adds
no alternative and changes no selection policy. A turn not due leaves the
validated batches unchanged. A valid pre-turn empty decision is non-exposed.
Failure to obtain a validated proposal is recorded separately.

When due, the workflow continues the actual conversation after the last
proposal. It sends only the engine's message and designated answer strings,
verbatim. One inspect round returns all requested listed details together,
in requested order, with the actual closing instruction. One failure
allowance covers unreadable replies, transport failures and a second inspect
request. At most three calls occur after a valid proposal, or four including
one correction after a refused proposal. Original replies are never relabelled
as another route to calculate the primary identity.

Select preserves original operations and selects candidate staffing; revise
replaces the entire reply; abandon submits nothing. The protocol validates
both batches again. A refused final validation submits neither. The state
must be unchanged at return. alternatives.v2.canonical records the actual
submitted batch, independently of the secondary labels. The analysis's
execution ledger records successful and partial application separately.

Each record contains the four action concepts, eligibility, exposure,
correction and failure fields, the validation-first attempt ledger, every
final exchange, and every complete model-visible delivery and scripted
response. A retained exact requestBody preserves byte-order-dependent
request hashes even when the surrounding evidence JSON sorts object keys.
Initial replayed calls appear in delivered interactions but are excluded
from the model-call budget; every subsequent attempt is counted.

Only the new opt-in extension names are added. The command registers them
alongside the historical operations. Historical operations, hello, workflows,
contracts and evidence remain unchanged. The shared lower-level engine and
session mechanics are read-only. New source, prompt, schema and parser pins
are listed in CONTRACT-PINS.json; frozen-file checks use BASELINE-HASHES.sha256.
