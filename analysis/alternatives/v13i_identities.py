"""Pin the V0.13i successor contracts and verify preserved bytes.

    python3 -I -B analysis/alternatives/v13i_identities.py --write
    python3 -I -B analysis/alternatives/v13i_identities.py --check

--check also verifies every file of the starting commit that V0.13i does not
deliberately change (docs/v13i/BASELINE-HASHES.sha256), and that the frozen
human-review material (93 transcripts, 7 static assets) is byte-identical.
"""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[2]
DEST = ROOT / "docs/v13i"


def digest(path):
    return hashlib.sha256((ROOT / path).read_bytes()).hexdigest()


def artifacts():
    rec, ver = "analysis/alternatives/v13i.py", "analysis/alternatives/v13i_verify.py"
    spec = ["analysis/alternatives/lifecycle_v1.json", "analysis/alternatives/lifecycle_spec.py", "docs/v13i/LIFECYCLE.md"]
    paths = {
        "dreamteam.run-lifecycle/v1": spec,
        "dreamteam-v13i-driver/5": [rec, "docs/v13i/DRIVER.md"],
        "dreamteam.alternatives-run/v5": [rec, "docs/v13i/RECORDING.md"],
        "dreamteam.alternatives-partial-run/v5": [ver, "docs/v13i/PARTIAL_RUN_PROCEDURE.md"],
        "dreamteam.alternatives-analysis/v6": [ver],
        "dreamteam.run-failure/v2": [rec, ver, "docs/v13i/RECORDING.md"],
        "dreamteam.health-observation/v1": [rec, "analysis/alternatives/lifecycle_v1.json", "docs/v13i/RECORDING.md"],
        "dreamteam.finalization-marker/v1": [rec, "docs/v13i/RECORDING.md"],
        "dreamteam.alternatives-experiment/v5": ["docs/benchmark/v13i-dev/protocol-draft.json"],
        "dreamteam.alternatives-schedule/v4#v13i-alt-1": ["docs/benchmark/v13i-dev/schedule-draft.json"],
        "dreamteam.v13i-fault-matrix/v1": ["analysis/alternatives/v13i_faults.py", "analysis/alternatives/v13i_scenarios.py",
                                           "docs/v13i/FAULT_MATRIX.md", "docs/v13i/fault-matrix/matrix.json"],
        "dreamteam.v13i-witnesses/v1": ["analysis/alternatives/v13i_witnesses.py", "docs/v13i/witnesses/results.json"],
        "dreamteam.v13i-mutations/v1": ["analysis/alternatives/v13i_mutations.py", "docs/v13i/mutations/results.json"],
        "dreamteam.v13i-signal-tests/v1": ["analysis/alternatives/v13i_sigint.py", "docs/v13i/signals/results.json"],
        "dreamteam.v13i-loopback/v1": ["analysis/alternatives/v13i_loopback.py", "docs/v13i/loopback/results.json"],
        "dreamteam.v13i-lifecycle-examples/v1": ["analysis/alternatives/v13i_examples.py",
                                                 "docs/benchmark/v13i-dev/lifecycle-examples/index.json"],
        "dreamteam.human-review-freeze/v1": ["analysis/alternatives/v13i_human_review.py",
                                             "docs/v13i/human-review/MANIFEST.json", "docs/v13i/human-review/MANIFEST.sha256"],
    }
    index = {"schema": "dreamteam.contract-identities/v2",
             "notice": "Independent technical re-review PENDING; human presentation review PENDING.",
             "identityHashEncoding": "SHA-256 of UTF-8 JSON {identity,files}, sorted keys, compact separators",
             "unchanged": {"dreamteam.alternatives-draw/v4": "V0.13h run_draw, reused unchanged",
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
    print(f"V0.13i contract identities, {n} preserved files and the frozen human-review material: PASS")


if __name__ == "__main__":
    main()
