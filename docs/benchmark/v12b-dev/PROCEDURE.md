# V0.12b Stage B: run instructions and the partial-run procedure

**Status:** nothing here may be done before the owner explicitly
authorizes live inference for Stage B. The protocol lock
(`protocol.json`, `LOCKED — NOT RUN`) does not authorize a run.

## 1. Before the run (no model calls)

On the Pi, from a clean checkout of the locked commit:

```sh
git status --short        # nothing but the two never-committed local files
go build -o ~/dt-v12b-run/dt ./cmd/dreamteam
R=~/dt-v12b-run/rp; mkdir -p $R
for f in docs/benchmark/v08c-dev/eval-replay-*.json docs/benchmark/v09c-dev/eval-replay-*.json; do
  g=$(basename $(dirname $f)); g=${g%-dev}; e=$(basename "$f" .json); e=${e#eval-replay-}
  ~/dt-v12b-run/dt benchmark suite run "$f" -out $R/llm-$g-replay-$e > /dev/null
  ~/dt-v12b-run/dt benchmark verify $R/llm-$g-replay-$e | tail -1
done
python3 -I -B analysis/acknowledgment/stage_b.py check --binary ~/dt-v12b-run/dt --replays $R \
  --protocol docs/benchmark/v12b-dev/protocol.json
```

**What `check` must report:**
* every point-arm start at the recorded state, with the agent's first
  request equal to the recorded request;
* `lock: holds`.

**What it is:** the dry start of all 24 point-arm combinations. It sends no
model request.

## 2. The run

Only after the owner's written authorization. Quote the authorization in
`--authorization`; it is recorded verbatim in `run.json`.

```sh
export DREAMTEAM_LLM_BASE_URL=...        # the oMLX server; never recorded or committed
systemd-run --user --scope -q -p MemoryMax=3G nice -n 10 \
  python3 -I -B analysis/acknowledgment/stage_b.py run --binary ~/dt-v12b-run/dt --replays ~/dt-v12b-run/rp \
  --protocol docs/benchmark/v12b-dev/protocol.json --out ~/dt-v12b-run/out \
  --authorization "<the owner's authorization, verbatim>"
```

* **One run.** One process, one request at a time. No other heavy job runs
  on the Pi or the server meanwhile.
* **Ending:**
  * The run ends by itself: COMPLETE after draw 420, or PARTIAL at the
    first stop condition.
  * Ctrl-C ends it as PARTIAL (`operator`).
  * Either way, `run.json` is written.
* **No retry or restart.** The output directory is used once. `run`
  refuses a directory that already holds a run.

## 3. After the run

```sh
python3 -I -B analysis/acknowledgment/stage_b.py analyze --binary ~/dt-v12b-run/dt --replays ~/dt-v12b-run/rp \
  --dir ~/dt-v12b-run/out --protocol docs/benchmark/v12b-dev/protocol.json --out ~/dt-v12b-run/out
```

**What `analyze` does:**
* replays every recorded draw without the model;
* refuses any difference;
* writes `decisions.json`, `decision-tables.md` and `inference.json`.

**What is committed:**
* the run directory (`live/`: `run.json`, `draws.jsonl`, the server
  states) and the analysis, under `docs/benchmark/v12b-dev/`;
* CI then replays it on every platform.

## 4. If the run is PARTIAL

**What can stop a run:**
* the four-hour limit;
* the call budget;
* the per-draw limit;
* transport failures;
* server health or a server change;
* a driver failure;
* the operator.

**When it stops, without exception:**
1. **Report it as PARTIAL.**
   * `run.json`'s `status` is `PARTIAL`;
   * the analysis's status is `PARTIAL`;
   * the tables open with the status, the coverage (`n of 420`) and the
     stop's code and message.
2. **Do not resume.** No command continues a run. Do not start another
   run into the same directory or into a new one.
3. **Do not substitute.** Missing draws stay missing. An interrupted draw
   stays interrupted: it is replayed, but not classified. No draw is
   rerun, reordered or replaced.
4. **Do not conclude about the complete experiment.** The analysis
   computes:
   * no B − A comparison;
   * no pooled summary;
   * no intervals against the scheduled denominators.

   Report only what was observed: the counts per point and arm, with the
   observed and scheduled draws side by side, and the stop.
5. **Commit and report.**
   * Commit the partial run and its analysis as they are.
   * Report the stop and the coverage to the owner.

**Any continuation requires both:**
* the owner's separate authorization;
* a documented protocol decision, recorded as an amendment to
  `PROTOCOL.md` with its reason: for example, a new run of the remaining
  schedule under a new protocol identity, or no continuation.

No continuation is defined in advance.

## 5. If something looks wrong during the run

**What to do:** stop it (Ctrl-C) and treat the run as PARTIAL (§4).

**Examples:** unexpected server behaviour, a model that is not the
protocol's, or memory pressure.

**What never to do:**
* change the server, models, settings or code mid-run;
* call admin endpoints;
* load, unload or clear caches.
