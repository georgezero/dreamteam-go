# V0.13m-lite known limitations

Protocol lock, live inference, tag and release NO GO. Human presentation
review PENDING. All items below are availability or disclosure limitations
under the owner's [revised acceptance standard](ACCEPTANCE_CONTRACT.md); none
permits a false COMPLETE, an accepted contradictory PARTIAL, invalid
evidence in analysis, or an automatic retry/replacement draw.

**DT13L-03/04 — policy disposition, not resolved.** In an unrecoverable
double failure (a second exception while recovering a draw's result, or
while building the failure report) evidence may be lost: a completed draw
held only in the failed draw's frame is not recovered. Such a run is stopped,
never sealed (INVALID), excluded from analysis, not retried or resumed, and
the possible loss is disclosed in `persistence-failure.json` and the
structured recording error. If storage refuses every write, the disclosure
exists only in the structured error.

**Fail-closed verification costs availability.** A `draw_unrecorded` draw
whose retained calls are enough to complete the workflow is rejected even if
the actual failure happened before the result was produced (R-HANDOFF-003);
a server that returns a response with duplicate keys, NaN/Infinity or an
overflowing number makes the run INVALID (R-JSON-004); a clock step of more
than 60 s between the monotonic and UTC clocks makes a genuine halt INVALID
(R-HALT-003).

**Unparsed responses.** A response body that neither the workflow parser nor
the strict parser reads (malformed, trailing content after the JSON value)
is an unparsed reply, handled by the frozen workflow as such; it carries no
metadata and never contributes a decision.

**Inherited (V0.13l, unchanged):** DT13K-06 conservative INVALID; recorder
faults between receipt and durable record are INVALID with the response
retained; no recovery or resumption of INVALID archives; no power-loss or
SIGKILL recovery; the seal is a self-hash, not an authenticity anchor; the
wall-clock guard is a request-start guard, not a hard deadline. Synthetic
fixture timestamps are admissible only to explicit fixture checks, where the
R-HALT-003 timestamp agreement is not applied. See
[V0.13l known limitations](../v13l/KNOWN_LIMITATIONS.md).
