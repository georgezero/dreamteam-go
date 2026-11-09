# V0.13k preservation

Every one of the 3,172 protected starting files is byte-identical. Historical code, recordings and independent reports remain unchanged. The existing completion-seal contract and finalization subtree are unchanged.

[Exact baseline paths and hashes](BASELINE.md), [machine baseline](BASELINE.json), [before/after results](PRESERVATION.json).

| Group | Count | Before and after SHA-256 |
| --- | ---: | --- |
| 262 scheduled draws | 262 | `253e5bdea590afd79d0878d573e09c64867ae81f25063d7d02eaa4091c7d9142` |
| 802-call ceiling | 802 | `477f4b2cdd3fe2b9fdd7dfb887f98252ef26cecf13eb53289116a1344382256d` |
| API and scoring | 68 | `01ceaeda3d760da52d5a36f441d2af32af2fe8e905271c5552166b0cb06825a1` |
| completion seal contract | 3 | `b62cd5711013e0301a9f6ab004c886377408006fe968eb94a7d844ce32fcd75c` |
| decision workflow | 28 | `6f6e868dfaf7b18991a03f2513ae7be92d65ea01fa26536858b754c407688663` |
| historical evidence | 2563 | `93112ed1fafb4e5e8c5dde1c0530eafe8d40086217f3a8778903e640528ed772` |
| historical implementation | 680 | `7daea6813566434e96745ee6827ac135b63b3168975dfe104e739cb5660039bd` |
| human review material | 5 | `1bee2f9d759a99a6e9a776495c6203bb185d86901a3a9bc252ed98f90cf55597` |
| nine historical contexts | 9 | `b529657c269f64d6498f2f9a9c2618cba7a7c21ddec11061f7d2e90536c151eb` |
| outcome definitions | 75 | `683758e362a60d7ac1692f3edc103352caeea92b52f96a6e256c578a5d01c222` |
| presentation | 3 | `4ee8041a779964ce807c37a6b1cd516e2bf9a61074762143aa568ce2e6da8bfc` |
| schedule and contexts | 3 | `64c08d573e7bf349929729b19ac192b64e088f48dabfcf0267ab04bdfdabce1d` |
| selection | 3 | `a69e3b1b5f3d029873821afe7ad5a4a7656b28eaf016311cdc8b669b0933afc1` |
| simulator and session | 240 | `6c8b06568e2d53a514f02ea5ea0ca93ccba187712dc95a33f7968e82696257e3` |
| six sensitivity pairs | 6 | `d49281587ba764c1928bf61dfd6fafe478ded1f6d3f8cf341fe25eed63183481` |
| staticAssets | 7 | `014629929f52f4693f6c33a492b1eb4946f64cc53027cfa8e8578d95c83af83a` |
| transcripts | 93 | `d6ea01d73c363387ed4653f5c7e0cb76044e3560d96d7041ac572c47e4adf792` |

The human presentation material at `ee987a1` is unchanged: 93 transcripts, seven static assets. Its review remains PENDING. The decision workflow, primary canonical submitted-action estimand, selection, presentation, simulator, session, API and scoring retain their bytes. The engine stays `0.13.0-dev.2`.

Canonical design hashes describe the same nine contexts, six pairs, 262 scheduled draws and 802-call ceiling. Scientific design identities are preserved while recording and verification receive successor identities.
