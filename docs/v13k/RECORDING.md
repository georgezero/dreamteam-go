# V0.13k recording specification

Driver `dreamteam-v13k-driver/7`; manifest `dreamteam.alternatives-run/v7`; analysis `dreamteam.alternatives-analysis/v8`; lifecycle /3; evidence bindings /1. Engine `0.13.0-dev.2`, draw /4, journal /1, health observation /2, failure event /3, public result /1 and completion seal /1 retain their formats. [Contract identities](CONTRACT-IDENTITIES.json) pin the successor implementations separately from historical evidence.

The archive retains `run.json`, `draws.json`, `calls.jsonl`, optional `server-before.json`, `aborted-draw.json`, `failure-event.json`, and the sole completion marker `SEAL.json`. Every field is covered in [the inventory](EVIDENCE_BINDINGS.md). A valid seal is required but insufficient: the full verifier independently validates the complete evidence set and reconstructs all retained events it can reproduce.

The durable handoff `pending-draw.json` contains `{schema, draw, rowSha256}`. It is removed only after a durable ledger commit and journal transition. Failed persistence candidates are retained, including completed rows. `persistence-failure.json` contains `{stage, error:{type,message}, pending, candidates}` when storage permits recording it. These artifacts are audit evidence for INVALID archives, never part of a sealed COMPLETE or PARTIAL archive. They must not be deleted to conceal a failed transfer.

Cleanup persists `{status, failure, timestamp, started, events}`. Events are numbered `{n,safePoint,state,timestamp}` for exactly three visits, including visits where no signal arrived. Signal records must agree with them. This closes the gap between cleanup result timestamps and claimed interrupt histories without changing the seal commit protocol.

Only external causes may remain UNVERIFIED. Recorded engine answers, workflow decisions, submitted actions, request/response exchanges, counts and digests are checked regardless of an artifact's label. All evidence uses the strict archive JSON parser; duplicate keys and nonfinite values are invalid. Reconstructed domain data use typed canonical equality.

No run resumption, inference authorization or protocol locking is provided. Use the [partial-run procedure](PARTIAL_RUN_PROCEDURE.md) and [lifecycle contract](LIFECYCLE_CONTRACT.md).

The health JSON evidence profile rejects duplicate keys and nonfinite values with the existing `invalid_json` category. Its successor enforcement identity is `dreamteam.health-evidence-classification/v3`; observation fields and trusted read-failure families retain their existing formats and behavior. A journaled returned response without HTTP status must retain a nonempty transport error and no response body, including unknown-draw paths.

When the draw call array is unknown, full analysis reconstructs the retained journal response prefix through the frozen workflow and compares each derived request hash. A null call array never excuses a contradictory reconstructable request. The missing draw result remains unknown.
