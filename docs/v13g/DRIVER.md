# Driver v3 — dreamteam-v13g-driver/3

Implementation: analysis/alternatives/v13g.py; scripted fidelity:
v13g_fidelity.py. Protocol v13g-alt-1 remains DRAFT, NOT LOCKED, NOT AUTHORIZED.
Independent re-review: PENDING. The run command has no live implementation.

Historical sessions are rebuilt from the nine frozen recording contexts and
verified digests. The twelve sensitivity members use their unchanged public
states and scripted proposals. Both arms get the same successor shared system
instruction and initial-phase wrapper. Historical source request hashes and
new delivered request hashes are separate: the original public context and
proposal are preserved, but the request's system instruction and phase wrapper
are successors. Initial historical proposals were not sampled under the new
instruction, and the experiment studies reconsideration conditional on them.

The deterministic schedule entries use the existing order rule unchanged.
The new protocol and schedule are regenerated from committed inputs and pinned
by content hash. Requests are checked against their exact expected serialized
bytes; subsequent requests and responses retain their full bodies and hashes,
status, metadata, and every engine exchange. A parsed request object is stored
alongside its exact serialized requestBody to preserve original key order.

Each draw has a fresh memory or its checked historical memory. Sessions are
pooled separately by context and arm and checked at the recorded state before
every draw. All workflow operations are read-only. Analysis replays recorded
responses through the actual successor workflow without sampling a model,
checks every deterministic field, and stores execution per exact submitted representation
and context. Operations and staffing apply sequentially, with individual
results, partial application and digests after application and end of day.

Analysis reports canonical submitted-action distributions within the exposed
completed denominator. Eligibility, correction failure, deliberate inaction,
process failures and original routes remain separate. Incomplete runs receive
no complete-run arm comparison or pair estimate. The complete-run pair
secondary measure is departure from the validated proposal's canonical identity,
not departure from an arm-dependent response label.

Commands: contexts --binary --out/--check; protocol --contexts --out;
schedule --protocol --contexts --out; check --binary --protocol --contexts
--replays; analyze --binary --protocol --contexts --replays --dir --out.
All are offline. The run parser always refuses execution; changing status or
supplying authorization does not implement or authorize a transport.

Initialization is inside the guarded draw lifecycle. Every caught opening,
setup, workflow, final-digest or close failure retains an aborted skeleton with
scheduled identity, failure stage, initialization status, completed-call count
and stop reason. External failures remain UNVERIFIED where reconstruction is
unavailable. No engine response is invented.
