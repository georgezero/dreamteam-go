# Holdout exposure record (Slice 5C.4a)

This note records how ordinary tests used the sealed holdout
(`docs/evaluation/holdout-v0.2.json`, `v0.2-holdout-1`, 40 seeds) before
Slice 5C.4a. The independent audit at 7707dbd found it
([`../audits/slice-5c3-eval4-framework-audit.md`](../audits/slice-5c3-eval4-framework-audit.md),
finding H3).

* **No seed values here.** This record names no holdout seed, and
  none was derived, printed or inspected to write it.
* **The holdout stands.** It is neither invalidated nor replaced (owner
  decision, 2026-10-18).
* **The claim it supports is limited.** The holdout seeds have been
  *used* by ordinary tests. No holdout outcome has been *disclosed*, and
  no *confirmatory evaluation* has been run. It is not true that the
  holdout was never touched.

## Three kinds of contact

| Kind | Meaning | Happened? |
|---|---|---|
| **Seed use** | A holdout seed value computed in a process, or a world built from it | **Yes** (below) |
| **Outcome disclosure** | Any clinical, operational or financial result of a holdout seed recorded, printed, logged or compared | **No** evidence of any (below) |
| **Full confirmatory evaluation** | The canonical runner's strategy-by-seed matrix (`TestHoldoutRun` with an unseal record) | **No**: no unseal record has ever existed, and `TestHoldoutRun` skips without one |

## Seed use

### 1. A world built on a holdout seed

* **Where.** `TestHoldoutRecordersGuarded` (`internal/sim/holdout_protocol_test.go`).
* **When.** From d66208c (Slice 5C.0b, 2026-10-17) through 7707dbd,
  unchanged.
* **What.**
  * The test loaded the repository's holdout manifest and regenerated
    its seeds in memory.
  * It built a world on the **first holdout seed**, under the **mini**
    scenario (the small test practice, not the evaluation scenario
    whose fingerprint the policy pins).
  * It advanced that world **only to the takeover instant**: the
    pre-takeover history, with no strategy action and no evaluation
    window.
  * It then asserted that the development recorder refuses that seed,
    and that a forged protocol cannot record it.
* **What it did not do.**
  * No recorder was created, because both refused as intended.
  * No sample, capture, record or report was produced.
  * No strategy or action was applied, and nothing ran past the
    takeover.
  * The seed was never printed: the refusal messages give only a count
    of seeds, by design (`GuardSeeds`).
* **How often.** Every ordinary run of the `internal/sim` tests in that
  period ran it, so it ran many times: each full-suite verification
  from Slice 5C.0b to 5C.3, and test runs during development. Runs are
  not logged individually, so the exact count is unknown.
* **What the test exposed.** Its only output was pass or fail, and that
  depends on whether the guards refuse, not on the world's state.

### 2. Seeds regenerated in memory without a world

From e2ce1fd (Slice 5C.0) on, three places regenerated the 40 holdout
seeds in memory from the committed manifest. None built a world.

* **`TestHoldoutSealed`** (`internal/eval`) used them to check the
  manifest: its digest, its count, no overlap with the development and
  reference sets, and no unseal record.
* **The same test** also passed one holdout seed to `GuardSeeds`, to
  confirm the refusal and that the refusal does not name the seed.
* **`TestGuardRepo`** passed one holdout seed to `GuardRepo`.
* **Every guarded harness** calls `GuardRepo` before it runs. That
  includes the development recorder, the calibration and dynamic
  benchmarks, and the development-set loader. It regenerates the seeds
  only to refuse them; it returns only an error.

In Slice 5C.4a, a mistake in the editing script left the old
`TestHoldoutSealed` and `TestGuardRepo` in place, and they ran once more
before their replacement. This was an in-memory derivation only: no
world was built and nothing was printed.

### 3. Copies of the manifest

The manifest names the label, range and method, so anyone holding the
repository can regenerate the seeds. That includes the read-only
snapshots given to the independent auditors.

* **The auditors.** They were instructed not to regenerate the seeds,
  and their reports name no seed.
* **What the commitment does.** It makes the set reproducible and
  tamper-evident. It does not keep the set from anyone with the
  repository (audit H3; 5C.4c adds external custody).

## No outcome disclosure, and no tuning from exposure

* **No artifact holds a holdout outcome.** No committed or uncommitted
  artifact known to the maintainer holds any holdout-seed outcome:
  * `docs/evaluation/holdout-runs/` does not exist;
  * the development runs (`docs/calibration/strategy-development/results/*/run.json`)
    name seeds 901–910 only;
  * the guarded harnesses refuse holdout seeds before running.
* **Strategy design used only development and reference evidence.**
  Every committed strategy (sets p1 to p6) was designed and compared on
  development seeds 901–910, with earlier reference-seed benchmarks as
  background (V0.2 design §5.9). The only holdout contact any test had
  is described above, and it produced no state that a strategy could be
  tuned against.
* **Two limits on that claim.**
  * It rests on the code history and the maintainer's working record,
    not on an independent log of every process ever run. This
    repository cannot prove that no one regenerated the seeds outside
    it.
  * No step in the documented workflow did so for any purpose but the
    guards above.

## Containment (Slice 5C.4a, 0.2.0-dev.15)

* **`TestHoldoutRecordersGuarded` uses a synthetic holdout.** It is the
  protocol test fixture's: its own label `dreamteam/test/holdout-runner`
  and range, sealed by removing its unseal record. The repository's
  manifest is no longer read.
* **`NewEvalWorld` guards before the world exists.** It is the test-only
  constructor for evaluation harnesses, and it refuses a holdout seed
  before any world is built. The development strategy runner and the
  evaluation-recorder tests now use it.
* **`TestGuardSeeds` and `TestGuardRepo` use a synthetic manifest.**
  `TestHoldoutSealed` still regenerates the real seeds in memory, to
  check the manifest's integrity; that is the only check that can
  confirm the commitment. It passes no holdout seed to anything.
* **`eval.Authorize` refuses a sealed holdout before regenerating any
  seed.** `TestAuthorizeSealedFirst` shows this. Before, it regenerated
  the seeds and then found no unseal record.
* **`TestHoldoutContainment` lists every reviewed site.** It reads all
  Go source, build-tagged files included, and lists every place that
  names the manifest path, regenerates a manifest's seeds or authorizes
  with the repository's paths. Any new site fails until it is reviewed.
  `HoldoutSeeds` outside the loader must take a `dreamteam/test/` label.

## What remains

* **Seed regeneration in the guards.** The guards still regenerate the
  real seeds in memory, so that they can refuse them.
* **The holdout is not secret.** Its custody, authenticated approval and
  a durable run-once record are Slice 5C.4c's.
* **Ordinary play is not guarded.** A session can still be started on
  any seed, as a game must allow. The guards cover the evaluation
  harnesses, not ordinary play.
* **The containment test checks names, not data flow.** It is a
  tripwire for reviewers, not a proof.
