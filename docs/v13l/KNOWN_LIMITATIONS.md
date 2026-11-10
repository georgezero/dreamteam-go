# V0.13l known limitations

Independent review PENDING. Human presentation review PENDING at the unchanged `ee987a1` freeze. Protocol lock, live inference, tag and release NO GO.

**DT13K-06 — conservative INVALID (availability limitation).** A genuine handled failure after the final digest answer but inside run_draw (before its engine log assignment) yields an aborted `final_digests` draw whose retained `digestsAfter` contradicts that stage's shape. It is classified INVALID. No false COMPLETE occurs, no contradictory PARTIAL passes, the evidence (including the retained final digest) is preserved, no retry or resume occurs and the failure is reported. This is an availability limitation, not a correctness success; verification is not weakened to accept it.

**Recorder-internal faults are conservative.** An exception injected inside the recorder between a response's receipt and its durable record, or after its record but before it is returned, fails closed or yields an INVALID archive with the response retained (journal, failure report or structured error). A failure to close the journal descriptor after a successful fsync is treated as a recording failure. These are INVALID by design.

**Engine errors caught by the workflow.** The frozen workflow converts a `ValueError` raised by an engine request during validation into a retry. A retained failed engine request is accepted only as the last engine event of an aborted draw, bound to the draw failure; an archive in which the workflow continued past one is INVALID (availability, as in V0.13k).

**Storage that refuses every write.** If storage refuses every later write, the recorder cannot create a new durable copy; it preserves existing candidates and returns the pending row, partial draw and received responses in the structured recording error.

**Single-fault boundaries.** The exhaustive boundary enumeration injects one handled exception per run at each line between draw completion and ledger commit, and between response receipt and its durable record. A second fault inside error handling (for example while writing the failure report) is covered by the filesystem cases, not by line enumeration.

The seal rename remains the commit point with process-crash semantics; power-loss recovery and forced termination before the first durable handoff are not claimed. External failure causes, timing and server metadata cannot be reproduced offline; their retained internal relationships are validated. The seal is a self-hash rather than a signature: a writer able to rewrite every trace consistently can erase evidence without leaving a contradiction. The matrices are finite engineering evidence; the separate independent review must generate unseen contradictions.
