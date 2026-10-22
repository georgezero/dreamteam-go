# V0.2 release acceptance (2026-10-21)

A bounded product acceptance pass of `main` at f03453c (0.2.0-dev.26)
against V0.2's own release contract: `docs/ROADMAP.md` §3 (deliverables
and acceptance) and `docs/V0.2_DESIGN.md` §15 (required evidence; "a
release requires all five actions in sim, API, session serialization,
headless use and TUI"). Not a clinical evaluation: the v5 confirmation
is closed (`../calibration/clinical-v5-closure/`) and the shipped binary
does not import the evaluation package at all.

**Update (2026-10-21, owner decision): F1 and F2 built before the
release** (0.2.0-dev.27, eb003a1): the T90 evaluation is visible in the
TUI and through `dreamteam t90`, equal to the canonical runner's report
byte for byte, and `sustainable` is a bundled scenario resolving to the
evaluation policy's exact scenario. Their acceptance is in
`docs/PROGRESS.md` (V0.2 completion) and the release notes. The P2
findings are accepted as nonblocking and listed in `README.md`.

**Recommendation at the time of this pass: CONDITIONAL GO.** No P0 and no defect in an
implemented feature was found. Two V0.2 contract deliverables are not
built (§2, F1 and F2); releasing needs the owner either to move them out
of V0.2 or to have them built first. Nothing was changed or fixed in
this pass.

## 1. Acceptance matrix

| Feature | Promised | Verified | Result | Evidence |
|---|---|---|---|---|
| `hire` | validate, preview, commit, replay in sim/API/session/headless/TUI | headless (2 hires, `request_conflict` refusal); TUI hire form preview and commit (H00003) | PASS | e2e run; TUI session |
| `terminate` | the same, notice and severance obligations | headless (notice 18 days, contract pay); TUI rows with severance; `unknown_employee` refusal | PASS | e2e run; TUI |
| `set_roster` | the same, whole-schedule reconciliation, displacement | headless (Riley 40 h from Nov 2); TUI roster editor; roster golden identical | PASS | e2e; golden 60d-roster |
| `set_reading_allocation` | the same, 30–100% band | headless (62.5%/52.5%); `allocation_out_of_band` makes the whole batch refuse (`batch_rejected`); TUI rows; allocation golden identical | PASS | e2e; golden 60d-allocation |
| `contract_teleradiology` | start, cancel, successor; economics | headless start T00001, cancel; `unknown_contract`, `request_conflict` refusals; TUI start/cancel rows, vendor money and lanes | PASS | e2e; TUI |
| Atomic batches | an invalid item refuses the batch with no side effect | `batch_rejected` with the first error; replay identical | PASS | e2e |
| Finance and operating costs | postings for wages, guarantees, recruiting, severance, vendor | 90-day run: revenue, expense, payroll, vendor unpaid/paid, guarantee pay in the log; invariants (balanced ledger) hold over 150 runs | PASS | e2e; acceptance invariants |
| Effects of decisions | decisions change operations and money | e2e plan vs no action, seed 3, 90 days: margin −34.1% vs −16.0%, lost demand 650 vs 383, report TAT p50 22 vs 30 min, p90 80 vs 116 min | PASS | e2e vs no-action results |
| Integration with simulator state | V0.2 actions on V0.1 operations | 150 × 90-day invariant runs with V0.1, staffing and roster plans; 720-day turnover (155 hires) and churn (340 roster versions) | PASS | `-tags acceptance` on horo, 2,125 s |
| TUI workflows | staff registry, previews, commit, reading coverage/cost, teleradiology | staff screen (`m`) with all five actions, guaranteed vs forecast costs, reading and teleradiology sections; end of day; save | PASS (with F3–F6) | tmux session at 160×48 |
| Headless CLI | `run -actions`, `-save`, `-log`, `-verify-replay`, `replay`, `verify` | all exercised; exit 0 | PASS | e2e |
| Save/load and replay | save, reload by replay, five digests | e2e session `verify` ok (run `c968a865…`); TUI-saved session (partial day, TUI hire) verifies headlessly (`2bb8b92b…`) | PASS | sessions |
| Determinism | same inputs, same outputs | rerun byte-identical result; `-verify-replay` true; amd64 and arm64 development lines byte-identical (v5 amendment 1) | PASS | e2e; `../calibration/clinical-v5-lock/AMENDMENT-1.md` |
| TUI/headless agreement | the same record | TUI-saved session replays headlessly with every digest | PASS | `verify` |
| V0.1 behaviour | goldens preserved | 4 V0.1 goldens and 3 V0.2 goldens: "outcomes identical" (documented `-allow-engine-mismatch` + migration manifest), each with `-verify-replay` | PASS | golden checks |
| Old saves | v1 and every registered earlier build replay | all 25 committed session fixtures `verify` ok | PASS | `testdata/sessions` |
| Evaluation v4 | unchanged | `policy-v0.2-eval-4.json` `4e277c72…` unchanged since 0.2.0-dev.14 | PASS | git |
| No dependency on v5 | — | the CLI's dependencies exclude `internal/eval` | PASS | `go list -deps` |
| Build | Linux and macOS | linux/arm64, linux/amd64, darwin/arm64, darwin/amd64 build | PASS | cross-builds |
| Tests | full suite, race, vet, gofmt | full uncached suite on this code (lock tree, no Go change since): every package passes, 3,478 s; `-race` on every shipped package (amd64, horo): pass, no race, 971 s; vet and gofmt clean | PASS | MEASUREMENTS 2026-10-21 |
| Performance | 90-day headless run (Pi waiver retained) | 3.9 s on this Pi (V0.1: ≈3.3 s under the waiver) | PASS (P2 note) | timing |
| **T90 vector and gates** | "T90 primary management vector … raw metrics visible"; TUI/API expose the T90 vector/gates | the framework exists in `internal/eval` (v4 in force) but only tests reach it; the TUI summary says "Measurements only; scoring comes later"; no CLI | **NOT BUILT** (F1) | `internal/tui/summary.go:295` |
| **Sustainable scenario** | "separate candidate sustainable scenario" | exists only as test overlays (`calib_dynamic_test.go`); `scenario list` shows `established`, `mini` | **NOT BUILT** (F2) | `scenarios/` |
| Batch preview/commit in the TUI | batch preview/commit | the TUI previews and commits one action at a time; batches via CLI/API | PARTIAL (F3) | `internal/tui/staff.go` |
| Allocation and teleradiology long runs | invariants over ≥50 seeds × 90 days with the actions | covered by the suite and goldens; the tagged long runs cover V0.1, staffing and roster plans only | NOT TESTED at that scale (F9) | `internal/sim/acceptance_staffing_test.go` |

## 2. Findings

| # | Class | Finding | Release impact |
|---|---|---|---|
| F1 | **P1 (contract gap)** | The T90 management vector and service gates are not exposed to the player (TUI, CLI or result). The evaluation code exists but is test-only. | A promised V0.2 deliverable. Owner: move to a later milestone, or build a read-only T90 view/CLI output first |
| F2 | **P1 (contract gap)** | No bundled `sustainable` scenario; the owner's chosen V0.2 start (shared XR/CT staffing at ×1.30 demand) exists only in test overlays. | A promised V0.2 deliverable and the premise of reorganization play. Owner: bundle the existing overlay as a scenario (small), or defer |
| F3 | P2 | TUI commits one action at a time; no multi-action batch preview/commit. | Usable; batches available headless |
| F4 | P2 | A refused action shows its message but not its stable code in the TUI (`staff.go:479-485`). | The message carries the reason |
| F5 | P2 | The staff table shows contract minutes, not rostered and worked hours (in the API, unused by the TUI). | Information gap |
| F6 | P2 | Displacement consequences appear only in previews; the after-the-fact displacement metrics are not shown. | Information gap |
| F7 | P2 | TUI teleradiology always uses the first catalog plan. | Plan choice headless only |
| F8 | P2 | README is stale: says `0.2.0-dev.2`, describes hire/terminate only, "two V0.2 goldens"; no V0.2 release notes yet. | Release documentation must be updated at release |
| F9 | P2 | The tagged long-run acceptance tests have no reading-allocation or teleradiology plan. | Coverage gap; those actions pass the suite and goldens |
| F10 | P3 | `run -golden` against an older-engine golden fails with "not comparable" without naming `-allow-engine-mismatch`. | Usability of the CLI message |
| F11 | P2 | 90-day headless run 3.9 s on the Pi (≈3.3 s at V0.1). | Within the retained waiver |

No P0: no crash, data loss, state corruption, save/replay divergence,
broken navigation or accounting error was found. No P1 defect in an
implemented feature: F1 and F2 are unbuilt deliverables, not broken
ones, so the stopping rule's "fix only P0/P1 defects" does not authorize
building them here.

## 3. Reproduction

```
go build -o dreamteam ./cmd/dreamteam
dreamteam run -scenario established -seed 3 -days 90 \
  -actions docs/v02-acceptance/e2e.actions.json -save e2e.session.json -format json
dreamteam verify e2e.session.json          # ok c968a865…
dreamteam -load e2e.session.json           # TUI: m (staff), e (end of day), ctrl+s
go test -tags acceptance ./internal/sim -run Acceptance -timeout 600m
```

`e2e.actions.json` combines the roster golden's first batch, the
allocation golden's first batch, a teleradiology start and cancellation,
a termination, and four refused batches (unknown employee, an invalid
last item, a conflicting request ID, an unknown contract).

## 4. V0.3 handoff: the first agent-interface slice

The roadmap's V0.3 item is `dreamteam agent --stdio` (observation/action
JSONL and a scripted-policy runner over the same API). The session layer
already offers `Observe`, `Validate`/`ValidateBatch`, `PreviewStaffing`,
`Apply`, `Advance`/`StepVisible`, digests and save. The smallest useful
slice adds no simulation behaviour:

* **`dreamteam agent --stdio`**: one JSON request per line, one JSON
  response per line, over a session started from a scenario and seed
  (or `-load`):
  * `observe` (the `api.Observation`, no hidden state);
  * `actions` (the permitted action types with their JSON schemas and
    current valid targets, e.g. employees and contracts);
  * `validate` and `preview` (no state change, digest unchanged);
  * `apply` (an atomic batch; the results with stable codes);
  * `advance` (by duration, to a time, or to the next visible event);
  * `save` and `digests`.
* **A scripted-policy runner** (`dreamteam agent -policy noop|greedy`)
  using the same protocol in process, for tests.
* **Acceptance:** schema-valid JSON for 30 days; an agent session saved
  and replayed headlessly with every digest; previews and validation
  leave the digest unchanged; the observation equals the TUI's for the
  same state; refusals carry stable codes.

Not in the slice: the eleven V0.3 operations actions, disruptions, HTTP
transport, LLM harness (V0.6).
