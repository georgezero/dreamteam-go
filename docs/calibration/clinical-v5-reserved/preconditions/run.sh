#!/bin/bash
# The reserved run: LOCK.md §3's command, with AMENDMENT-1's prebuilt linux/amd64 binary.
set -o pipefail
D=$HOME/dt-v5-run
"$D/preflight.sh" > "$D/preflight.log" 2>&1 || { echo "preflight failed: nothing run" >> "$D/preflight.log"; exit 1; }
cd "$D/repo"
sha256sum "$HOME/dt-v5-run/bin/eval.test"   # must equal AMENDMENT-1 §2
mkdir -p docs/calibration/clinical-v5-reserved
cd internal/eval
date -u +%Y-%m-%dT%H:%M:%SZ > "$D/start"
env -u DT_MANIFEST_SEEDS -u DT_MANIFEST_SCENARIOS -u DT_MANIFEST_IN \
  DT_MANIFEST_FAMILY=20261201 \
  DT_MANIFEST_OUT=../../docs/calibration/clinical-v5-reserved/reserved-run.jsonl \
  DT_CRITERIA_OUT=../../docs/calibration/clinical-v5-reserved/reserved-criteria.json \
  GOMEMLIMIT=5GiB systemd-run --user --scope -q -p MemoryMax=6G nice -n 10 \
  "$HOME/dt-v5-run/bin/eval.test" -test.run 'TestReservedManifestRun$' -test.v -test.timeout 2000m \
  2>&1 | tee ../../docs/calibration/clinical-v5-reserved/reserved-run.log
echo $? > "$D/exit"
date -u +%Y-%m-%dT%H:%M:%SZ > "$D/end"
