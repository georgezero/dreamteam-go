#!/bin/bash
# Reserved-run preconditions (LOCK.md §3, AMENDMENT-1.md). Any failure: exit 1, nothing run.
set -u
fail() { echo "PRECONDITION FAILED: $*"; exit 1; }
ok() { echo "ok: $*"; }
R=$HOME/dt-v5-run/repo; B=$HOME/dt-v5-run/bin/eval.test
cd "$R" || fail "no repository"
echo "time: $(date -u +%Y-%m-%dT%H:%M:%SZ)  host: $(hostname)  $(uname -srm)"
[ "$(git rev-parse HEAD)" = "$(git rev-parse 38881d4)" ] || fail "HEAD is not 38881d4"
git merge-base --is-ancestor ff5b566 HEAD || fail "ff5b566 is not an ancestor"
[ "$(git rev-parse ff5b566^)" = "$(git rev-parse 7bf1ed6)" ] || fail "lock parent is not 7bf1ed6"
[ "$(git diff --name-only ff5b566 HEAD)" = "docs/calibration/clinical-v5-lock/AMENDMENT-1.md" ] || fail "changes since the lock beyond the amendment"
[ -z "$(git status --porcelain)" ] || fail "repository not clean"
ok "repository at $(git rev-parse --short HEAD), lock ff5b566 (parent 7bf1ed6), clean, only the amendment since the lock"
n=$(sha256sum -c docs/calibration/clinical-v5-lock/lock.sha256 | grep -c ': OK$'); [ "$n" = 49 ] || fail "lock.sha256: $n of 49 OK"
ok "lock.sha256: 49 of 49 files OK"
src=$(cd internal/eval && python3 -I -c '
import hashlib,os
h=hashlib.sha256()
for n in sorted(f for f in os.listdir(".") if f.endswith(".go")):
    raw=open(n,"rb").read(); h.update(f"{n} {len(raw)}\n".encode()); h.update(raw)
print(h.hexdigest())')
[ "$src" = 5cd6a438a3c2e741f691815ba96e805f022f39f34a51f131d8a2164f9a07588b ] || fail "sourceSHA256 $src"
ok "sourceSHA256 $src"
b=$(sha256sum "$B" | cut -d' ' -f1); [ "$b" = bf9e1c46a90022331efa571ff66f83e2d38ae4e73aa3f4c93f63a9677724e088 ] || fail "binary $b"
ok "binary (linux/amd64, AMENDMENT-1) $b"
while read -r want path; do got=$(sha256sum "$path" | cut -d' ' -f1); [ "$got" = "$want" ] || fail "$path $got"; ok "$path $got"; done <<'L'
83d481b7040b69d78a58de68afc938a588b5b37ba396438509f38c56ca163948 docs/calibration/clinical-5c4b/candidate-policy-v5.json
250bc7fc835d99140212505e0e79f74f2da1484dbe856fea6a5b93eec9153ef4 docs/calibration/clinical-5c4l/scenario-manifest.json
12d1eda4f8543e64cdbf1d5118339b65bc6a620e8ee3f6cb01dfd974d0c91a0d docs/calibration/clinical-5c4n/reserved-validation-protocol.md
4e277c7263dcff7ddabe63741beb44abfc9a46d3bc23f6b6418715b58ffb6fb3 docs/evaluation/policy-v0.2-eval-4.json
ac8621889813a2d582b4b19674f04871c5711d27d0ac1620839ba01df4596100 internal/eval/testdata/sim-profile-5c4b.json
L
grep -q "authorized, on 2026-10-21, \*\*one\*\* reserved execution" docs/calibration/clinical-v5-lock/AMENDMENT-1.md || fail "authorization not recorded"
ok "owner authorization recorded (AMENDMENT-1.md §1)"
[ ! -e docs/calibration/clinical-v5-reserved ] || fail "output directory exists"
ok "output directory docs/calibration/clinical-v5-reserved absent (unused)"
[ "$(ls -A $HOME/dt-v5-run | grep -vx -e preflight.log -e preflight-aborted-1.log)" = "$(printf 'bin\ndt.bundle\npreflight.sh\nrepo\nrun.sh')" ] || fail "dt-v5-run not fresh: $(ls -A $HOME/dt-v5-run | tr '\n' ' ')"
[ ! -e "$HOME/dt-v5-run/start" ] || fail "a reserved execution has started before (start marker)"
ok "no previous reserved execution on this host (fresh run directory, no start marker, no output)"
env | grep -q '^DT_' && fail "DT_ variables set: $(env | grep '^DT_' | tr '\n' ' ')"
ok "no DT_* variable set (no seed, scenario or input filter)"
avail=$(df -BG --output=avail "$R" | tail -1 | tr -dc 0-9); [ "$avail" -ge 20 ] || fail "disk ${avail}G"
mem=$(awk '/MemAvailable/ {print int($2/1048576)}' /proc/meminfo); [ "$mem" -ge 8 ] || fail "memory ${mem}G available"
[ "$(loginctl show-user $USER -p Linger --value)" = yes ] || fail "no linger"
ok "disk ${avail}G free; memory ${mem}G available; linger yes; uptime: $(uptime -p); load $(cut -d' ' -f1-3 /proc/loadavg)"
echo "ALL PRECONDITIONS MET"
