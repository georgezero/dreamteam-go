# V0.13l preservation

Every one of the 3,387 protected starting files is byte-identical (starting commit `ea7bf4bc4718397b2d73568e7a5279104fc795b4`), including every V0.13k module and evidence file and every independent report. The completion-seal contract and finalization subtree are unchanged.

[Exact baseline paths and hashes](BASELINE.md), [machine baseline](BASELINE.json), [group check](PRESERVATION.json).

| Group | Count | Before and after SHA-256 |
| --- | ---: | --- |
| 262 scheduled draws | 262 | `253e5bdea590afd79d0878d573e09c64867ae81f25063d7d02eaa4091c7d9142` |
| 802-call ceiling | 802 | `477f4b2cdd3fe2b9fdd7dfb887f98252ef26cecf13eb53289116a1344382256d` |
| API and scoring | 68 | `01ceaeda3d760da52d5a36f441d2af32af2fe8e905271c5552166b0cb06825a1` |
| V0.13k implementation and evidence | 213 | `579ac4b115353f24a4439d857c9e500a6a3c65592e0a3c3598248cc28a35e257` |
| completion seal contract | 5 | `44360da366cbe09fe61fac4c76315128e875906fa773b43003fa84f78339740e` |
| decision workflow | 28 | `6f6e868dfaf7b18991a03f2513ae7be92d65ea01fa26536858b754c407688663` |
| historical evidence | 2778 | `71b0918e8b664b2482107384c674e731f7decdbdc6e654987e2293c51d883521` |
| historical implementation | 699 | `6fa77078ce13d75c404d775aeb65c240731015a07539cdabaf4161aba559a3f8` |
| human review material | 5 | `1bee2f9d759a99a6e9a776495c6203bb185d86901a3a9bc252ed98f90cf55597` |
| independent and audit reports | 49 | `6a876b180855d56a8eba8ced8ddd848a276a1e29aea8cf7461e7c5c366625456` |
| nine historical contexts | 9 | `b529657c269f64d6498f2f9a9c2618cba7a7c21ddec11061f7d2e90536c151eb` |
| outcome definitions | 75 | `683758e362a60d7ac1692f3edc103352caeea92b52f96a6e256c578a5d01c222` |
| presentation | 3 | `4ee8041a779964ce807c37a6b1cd516e2bf9a61074762143aa568ce2e6da8bfc` |
| schedule and contexts | 3 | `ff0048e067427104cd93b780d0bdcaf669c3a42d37c7d5a9795bb57a1c6b0bfd` |
| selection | 3 | `a69e3b1b5f3d029873821afe7ad5a4a7656b28eaf016311cdc8b669b0933afc1` |
| simulator and session | 240 | `6c8b06568e2d53a514f02ea5ea0ca93ccba187712dc95a33f7968e82696257e3` |
| six sensitivity pairs | 6 | `d49281587ba764c1928bf61dfd6fafe478ded1f6d3f8cf341fe25eed63183481` |
| staticAssets | 7 | `014629929f52f4693f6c33a492b1eb4946f64cc53027cfa8e8578d95c83af83a` |
| transcripts | 93 | `d6ea01d73c363387ed4653f5c7e0cb76044e3560d96d7041ac572c47e4adf792` |

The human presentation material at `ee987a1` is unchanged (93 transcripts, seven static assets; `git diff --exit-code ee987a1 -- docs/v13i/human-review` is clean). Its review remains PENDING. The engine stays `0.13.0-dev.2`.

## Durable evidence ownership

Machine results: [preservation/results.json](preservation/results.json) (`analysis/alternatives/v13l_preservation.py`).

* Inherited completed-draw filesystem/return/error-handling cases: 6/6.
* Model-response handoff faults (outcome write, fsync and close × retention ok, failure-report rename failure, every later write refused): 9/9; every received response is retained in the journal, failure report or structured error; transport invocations equal journal dispatches (no replacement call).
* Exhaustive line-boundary enumeration, completed draw (from a completed, RECORDED result inside run_draw until its ledger commit is durable and the handoff cleared): 258/258 injected handled exceptions retain the exact completed draw (ledger, handoff, failed candidate, failure report or structured error) and it replays; an accepted archive always contains it. One site is run_draw's own session-close line, where the frozen machinery records a genuine close failure as an aborted `engine_close` draw that keeps every result field and is replayed.
* Exhaustive line-boundary enumeration, received model response (from receipt until its outcome record is durable): 62/62; each retains the response and makes no replacement call.
