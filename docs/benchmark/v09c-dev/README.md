# V0.9c: prospective preview evaluation

**Status: declared, before any live run.** The plan is
[PLAN.md](PLAN.md) (`v09c-plan/1`), and the treatment isolation record is
[TREATMENT.md](TREATMENT.md). This page gets the results after the
evaluation; the plan does not change.

**Question:** does showing the agent the simulator's existing preview
before it submits a staffing proposal change the management decisions it
proposes and executes?

## 1. Baseline

* **Start:** clean main at 8ffae8d (V0.9b, accepted).
* **Tag:** `v0.8.0` (object db5c147) still points at b803302. No tag was
  moved.
* **Present and unchanged at 8ffae8d:**

| Item | File | SHA-256 |
|---|---|---|
| llm-agent 1 and 2 | agents/llm/llm_agent.py | `5214e7bfb456075f38aca92a5c45a6db3141d4042fb1cd5822c8c1c65c286d87` |
| llm-agent 3 (the baseline) | agents/llm/llm_agent3.py | `ef2c17a06749d7c287cda918f77ff0e518d79475048e87fd3404848c2fb0015c` |
| model settings (both conditions) | docs/benchmark/v05c-dev/llm-qwen36-35b-a3b.json | `e19a47dce948a8acbd3262ef5053b89f1e23c32603860e03628d38b184acbf5d` |
| | docs/benchmark/v05c-dev/llm-gemma4-26b-a4b.json | `1a1edae5a492b67a72086bec51faecc8ed0930800bdb6e548412b65fa444aa6a` |
| financial evidence v1 | analysis/evidence/financial_evidence.py | `8158addf8a8236b5b0426d7e520b13325291aa8b7804d81c8f8959de51cbe4b8` |
| | docs/benchmark/v09b-dev/financial-evidence.json | `bf61212f832eef1a76bca3f9fd827af34251aad6d3d55c5719a85f2a52341b1a` |
| parameter evidence v1 (frozen) | analysis/evidence/frozen-parameter-v1.json | `27e13a79f7787512a9a4a36fcf11fdb437823a608bad02b210ee51aece948619` |
| | docs/benchmark/v09a-dev/parameter-evidence.json | `9f556c69ad187c22b9dbb1c4a3149b1e450c368b97815b1e3e75368ae3c01a71` |
| decision evidence v1 (frozen) | docs/benchmark/v08a-dev/evidence.json | `5743bc223965e75d77440021ecf602b74c1e4bb35c6b6a08afe8be08b8955d4d` |
| | analysis/evidence/frozen-v1.json | `64a8e015f9b6901447ec6ddbd979e8882047cdfef62911cf6db983db7ff76a78` |
| repeat evidence v2 | docs/benchmark/v08c-dev/history/repeat-v2.json | `a80b704cc62d960c46075b723801ef1c8cd47b682185f3ed27daa72abfde2624` |
| refusal-response v1 | docs/benchmark/v08b-dev/refusal-response.json | `55d22b53d65cc49f6e72a2379238d5ebfa80f7e0027810020fd3a068ff7d3cc9` |
| refusal-response v2 | docs/benchmark/v08c-dev/history/refusal-response-v2.json | `cdb0e1cc0d59495f09733d2eb0a1e8b5e98dc1edfc574cd2010f2a242e6f0aa0` |
| review status (PENDING) | docs/benchmark/v09a-dev/review/review-status.json | `36e6efabcb42fae765d52cbe72d7d31ebafc57a232e4f3f7aed4ffd8011bfed4` |

Aggregates (SHA-256 of the lines `<path> <sha256>` in path order):

| Set | Files | Aggregate at 8ffae8d |
|---|---|---|
| simulator: Go sources without tests and the version package, scenarios, go.mod | 129 | `2316df3d8bd1a3016806e7af6c398bc611b3af6b2cc9e21981ceab8c2bb92f58` |
| historical recordings, V0.5c–V0.8c (`*-decisions.jsonl`) | 96 | `041a30b0788edc3017b18b2fa07d36087e73f5786af33d33a98559794234b5c3` |
| benchmark suites (docs/benchmark/suite) | 4 | `63c007ccedc023a501690aa14ca76d400a18be68024cfa658ec5804bd7265c65` |
| golden references (testdata/golden, without MIGRATIONS.json) | 12 | `d84f80718f3ab0463cc3768cf89b5ba09885b654e674a92a87c82c6998a2b25f` |
| testdata/golden/MIGRATIONS.json | 1 | `d0e29c0f96d68607dacadc6bd7129553f793a236ee870ad2c6a85a4674713599` |
| session fixtures (testdata/sessions/*.json) | 104 | `ecc3931fb4697d0ddf356f44034179ae61c541f0c8b5ee311ec97476b3444845` |

**What the declaration changes among them,** as every version does:
* the version, 0.9.0-dev.3;
* the registration of 0.9.0-dev.2 for replay. That is two lines in
  `internal/session/record.go` and one in `record_v1.go`, which change the
  simulator aggregate; no behaviour changes.
* MIGRATIONS.json's engine and explanation line;
* two new session fixture files (`mini-seed52-v092`).

Nothing else in the table changes. llm-agent 3 is not edited: llm-agent 4
loads it.
