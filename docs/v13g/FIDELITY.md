# V0.13g scripted fidelity

Implementation: analysis/alternatives/v13g_fidelity.py. No inference transport
is used. Historical responses, where needed, are preserved responses served
as scripts. New scripted responses are labelled as scripts, never observations
of model behavior. Independent re-review: PENDING.

The suite exercises the actual successor workflow, driver, protocol operations
and analysis. Every completed case is replayed without a model; submitted
batches are executed twice in fresh sessions with deterministic continuations.
The nine historical contexts require the V0.8c/V0.9c replay prerequisites.
All twelve sensitivity members are exercised in both arms. Additional fidelity
contexts are regression witnesses, not additions to the fixed experiment.

The complete-context cases retain actual delivered system, initial context,
proposal, correction/retry, final-turn and combined-inspection messages. Extra
cases cover partial termination, roster notice/capacity, mixed unsupported
capacity, system-schema precedence, empty revision, failed correction,
deliberate correction to no action, transport failure, retry exhaustion,
refusal, changed operations, selection and abandonment. Record-only outcomes
remain separate from the model-visible package.

Reproducible interactions retain the complete delivered message and response
bytes, requests, request bodies and hashes. Nondeterministic send timestamps
and measured latency are excluded from these scripted artifact identities;
they remain part of ordinary run recording. No model-visible text is dropped
or rewritten. The package generator derives transcripts from those records.

The empty-action audit witness aggregates five control abandonments and five
treatment no_action selections. Primary canonical-action TV must be exactly
zero. Secondary response-route differences remain observable. The pure and
engine regression suites additionally cover malformed partial-run metadata,
reconstructible aborts and explicitly UNVERIFIED aborts.

Reference outputs belong only under docs/benchmark/v13g-dev/fidelity/.
The historical V0.13d fidelity and evidence remain byte-identical and are
regenerated separately. Engineering checks do not constitute human review
or an independent technical re-review.

The successor adds numeric spellings in final and correction turns, 64/65-action
boundaries, fractional/overflow refusals and invalid operations mixed with valid staffing. Separate actual
scripted initialization failures and between-draw stops are packaged as partial
examples. Final case/interaction/package counts and engineering dispositions are
recorded in gates/README.md. The empty-action witness retains five exposed draws
per arm and primary TV = 0; its identity remains
4f53cda18c2baa0c0354bb5f9a3ecbe5ed12ab4d8e11ba873c2f11161202b945.
Engineering results do not supply reviewer findings or acceptance.
