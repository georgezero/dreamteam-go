# V0.13l lifecycle fault verification

Independent review PENDING. Protocol lock and live inference NO GO.

The shared contract generated 5288 fault cases at 1619 points. Disagreements: 0; false COMPLETE: 0; unreached declared points: 0; transitions not observed: 0; admissible stop states not observed: 0.

DT13K-07: every generated case context keeps `pendingRetained`, `receivedUnrecorded` and the other preservation keys (`CONTEXT_KEYS`); the oracle fails a case whose context drops one, and additionally requires a model response received at the injection point to survive as its durable journal outcome, in the failure report or in the structured error.

Real-process signal/termination checks: {"agree": 353, "byKind": {"sigint": {"agree": 41, "cases": 41}, "sigint-line": {"agree": 4, "cases": 4}, "sigint-process-group": {"agree": 4, "cases": 4}, "sigkill-external": {"agree": 7, "cases": 7}, "sigkill-self": {"agree": 294, "cases": 294}, "sigterm-self": {"agree": 3, "cases": 3}}, "cases": 353, "falseComplete": 0}. Real localhost socket checks: {"agree": 12, "cases": 12}. All ran on horo. Seal format and process-crash commit semantics are unchanged; arbitrary power-loss recovery is outside this milestone.

Machine results: [fault matrix](fault-matrix/matrix.json), [signals](signals/results.json), [loopback](loopback/results.json), [preservation](preservation/results.json).
