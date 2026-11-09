"""Pin the V0.13j successor contracts and verify preserved bytes.

    python3 -I -B analysis/alternatives/v13j_identities.py --write
    python3 -I -B analysis/alternatives/v13j_identities.py --check

--check also verifies every file of the starting commit that V0.13j does not
deliberately change (docs/v13j/BASELINE-HASHES.sha256), and that the frozen
human-review material (93 transcripts, 7 static assets; docs/v13i/human-review,
frozen at ee987a1) is byte-identical.
"""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[2]
DEST = ROOT / "docs/v13j"


def digest(path):
    return hashlib.sha256((ROOT / path).read_bytes()).hexdigest()


def artifacts():
    rec, ver = "analysis/alternatives/v13j.py", "analysis/alternatives/v13j_verify.py"
    spec = ["analysis/alternatives/lifecycle_v2.json", "analysis/alternatives/lifecycle_v2_spec.py",
            "docs/v13j/SAFETY_CONTRACT.md"]
    paths = {
        "dreamteam.run-lifecycle/v2": spec,
        "dreamteam.run-finalization/v1": ["analysis/alternatives/lifecycle_v2.json", rec, "docs/v13j/SAFETY_CONTRACT.md"],
        "dreamteam.completion-seal/v1": ["analysis/alternatives/lifecycle_v2.json", rec, ver, "docs/v13j/RECORDING.md"],
        "dreamteam.call-journal/v1": ["analysis/alternatives/lifecycle_v2.json", rec, ver, "docs/v13j/RECORDING.md"],
        "dreamteam.run-result/v1": [rec, "docs/v13j/RECORDING.md"],
        "dreamteam-v13j-driver/6": [rec, "docs/v13j/DRIVER.md"],
        "dreamteam.alternatives-run/v6": [rec, "docs/v13j/RECORDING.md"],
        "dreamteam.alternatives-partial-run/v6": [ver, "docs/v13j/PARTIAL_RUN_PROCEDURE.md"],
        "dreamteam.alternatives-analysis/v7": [ver],
        "dreamteam.run-failure/v3": [rec, ver, "docs/v13j/RECORDING.md"],
        "dreamteam.health-observation/v2": [rec, "analysis/alternatives/lifecycle_v2.json", "docs/v13j/RECORDING.md"],
        "dreamteam.alternatives-experiment/v6": ["docs/benchmark/v13j-dev/protocol-draft.json"],
        "dreamteam.alternatives-schedule/v4#v13j-alt-1": ["docs/benchmark/v13j-dev/schedule-draft.json"],
        "dreamteam.v13j-fault-matrix/v1": ["analysis/alternatives/v13j_faults.py", "analysis/alternatives/v13j_scenarios.py",
                                           "docs/v13j/FAULT_MATRIX.md", "docs/v13j/fault-matrix/matrix.json"],
        "dreamteam.v13j-signal-tests/v1": ["analysis/alternatives/v13j_signals.py", "docs/v13j/signals/results.json"],
        "dreamteam.v13j-loopback/v1": ["analysis/alternatives/v13j_loopback.py", "docs/v13j/loopback/results.json"],
        "dreamteam.v13j-witnesses/v1": ["analysis/alternatives/v13j_witnesses.py", "docs/v13j/witnesses/results.json"],
        "dreamteam.v13j-mutations/v1": ["analysis/alternatives/v13j_mutations.py", "docs/v13j/mutations/results.json"],
        "dreamteam.v13j-lifecycle-examples/v1": ["analysis/alternatives/v13j_examples.py",
                                                 "docs/benchmark/v13j-dev/lifecycle-examples/index.json"],
    }
    index = {"schema": "dreamteam.contract-identities/v2",
             "notice": "Independent technical re-review PENDING; human presentation review PENDING.",
             "identityHashEncoding": "SHA-256 of UTF-8 JSON {identity,files}, sorted keys, compact separators",
             "unchanged": {"dreamteam.alternatives-draw/v4": "V0.13h run_draw, reused unchanged",
                           "dreamteam.human-review-freeze/v1": "docs/v13i/human-review at ee987a1, unchanged",
                           "engine": "0.13.0-dev.2"},
             "identities": {i: {p: digest(p) for p in ps} for i, ps in paths.items()}}
    index["identitySha256"] = {i: hashlib.sha256(json.dumps({"identity": i, "files": files}, sort_keys=True,
                                                            separators=(",", ":")).encode()).hexdigest()
                               for i, files in index["identities"].items()}
    pins = {"schema": "dreamteam.lifecycle-contract-pins/v1",
            "files": {p: digest(p) for p in sorted({p for ps in paths.values() for p in ps})}}
    return {"CONTRACT-IDENTITIES.json": index, "CONTRACT-PINS.json": pins}


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    g = ap.add_mutually_exclusive_group(required=True)
    g.add_argument("--write", action="store_true")
    g.add_argument("--check", action="store_true")
    a = ap.parse_args()
    for name, obj in artifacts().items():
        data = (json.dumps(obj, indent=1, sort_keys=True) + "\n").encode()
        if a.write:
            (DEST / name).write_bytes(data)
        elif (DEST / name).read_bytes() != data:
            raise SystemExit("contract mismatch: " + name)
    n = 0
    for line in (DEST / "BASELINE-HASHES.sha256").read_text().splitlines():
        want, path = line.split("  ", 1)
        if digest(path) != want:
            raise SystemExit("preserved bytes changed: " + path)
        n += 1
    subprocess.run([sys.executable, "-I", "-B", str(ROOT / "analysis/alternatives/v13i_human_review.py"), "--check"],
                   check=True, stdout=subprocess.DEVNULL)
    print(f"V0.13j contract identities, {n} preserved files and the frozen human-review material: PASS")


if __name__ == "__main__":
    main()
