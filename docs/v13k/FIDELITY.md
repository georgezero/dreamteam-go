# V0.13k scripted fidelity

Independent review PENDING. Protocol lock and live inference NO GO.

All 66 distinct scheduled context/model/arm combinations were recorded through the successor recorder and independently replayed by full analysis. This includes all 21 contexts and all nine historical contexts, restored from recorded replay states and memory. All 66 completed draws verified; 62 post-proposal scripted calls and 834 engine events were retained and checked. Historical initial proposals remain fixed inputs under the unchanged call-count definition.

This is an explicitly synthetic record-only fixture with one draw per scheduled unit. It is not the 262-draw experiment, does not estimate the primary outcome, and authorizes no inference. The scientific schedule, contexts, sensitivity pairs, call ceiling and canonical submitted-action estimand are unchanged. The frozen workflow's broader branch cases are separately reproduced by the historical fidelity gates.

[Machine results](../benchmark/v13k-dev/fidelity/results.json), [complete fixture archive](../benchmark/v13k-dev/fidelity/archive/live/run.json), [full analysis](../benchmark/v13k-dev/fidelity/archive/analysis.json). The fixture retains its own protocol identity and declared synthetic provenance; ordinary command-line analysis refuses it unless the engineering fixture admission is explicit.

Reproduction uses `analysis/alternatives/v13k_fidelity.py` with absolute binary, contexts, replay, fresh scratch and output paths; see [the test map](gates/TEST_MAP.md). Both cross-platform workflows regenerate and compare the deterministic results.
