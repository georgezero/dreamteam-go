# V0.13k known limitations

Independent review and owner acceptance are PENDING. Human presentation review is separately PENDING at the unchanged `ee987a1` freeze. Protocol lock, live inference, tag and release are NO GO.

A failed persistence operation, unrecoverable handoff, malformed or incomplete archive, or forced termination before the completion seal is INVALID. Invalid archives remain for audit; they are not resumed or repaired into valid runs. Completed results and failed candidates survive handled persistence failures when storage permits the retained write. If storage refuses every subsequent write, the recorder cannot promise a new durable copy; it preserves existing candidates and exposes retained error/result data through the recording exception. This limitation never permits a false COMPLETE or contradictory PARTIAL.

The existing seal rename remains the commit point. Its process-crash behavior is preserved. Power-loss recovery and preservation of arbitrary in-memory data across forced termination before the first durable handoff are not claimed. Fault tests require pending-result preservation across handled failures; hard-kill tests require already durable ledger/call evidence and the inherited seal behavior. This is the owner's bounded acceptance standard, not a general crash-recovery project.

External failure causes, external timing and server metadata cannot all be reproduced offline. Their retained internal relationships, stage identities, timestamp intervals, bodies, hashes and counters are nevertheless validated. UNVERIFIED is never a shortcut around engine, workflow or count checks.

The seal is a self-hash rather than a signature. A writer able to remove every trace consistently can erase evidence without leaving a reconstructable contradiction. An external authenticity anchor would require an owner policy decision and is outside this milestone.

The systematic matrix covers each declared relationship and ten mutation kinds with explicit targets. It is finite evidence, not proof that every possible contradiction has been excluded. The separate independent review must generate additional contradictions and re-seal them before running full analysis.
