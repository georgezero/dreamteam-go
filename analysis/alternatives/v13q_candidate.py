#!/usr/bin/env python3
"""V0.13q protocol lock CANDIDATE manifest: build and check.

The candidate pins the 12 items of docs/v13p/LOCK-REQUIREMENTS.md, each
re-derived from the repository: file identities by SHA-256 of the exact
bytes, values from the strictly parsed protocol. It is not a lock. Its status
is CANDIDATE — NOT LOCKED — NOT AUTHORIZED, and it lists the blockers that
must clear before an owner lock gate. Nothing here records, runs, or
contacts a model.

    python3 -I -B analysis/alternatives/v13q_candidate.py --write docs/v13q/LOCK-CANDIDATE.json
    python3 -I -B analysis/alternatives/v13q_candidate.py --check docs/v13q/LOCK-CANDIDATE.json
"""
import argparse
import hashlib
import json
import os
import re
import subprocess
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(os.path.dirname(HERE))
sys.path.insert(0, HERE)
import v13p  # noqa: E402  (strict parsing and the acceptance binding; read-only use)

SCHEMA = "dreamteam.lock-candidate/v1"
STATUS = "CANDIDATE — NOT LOCKED — NOT AUTHORIZED"
PROTOCOL = "docs/benchmark/v13o-dev/protocol-draft.json"
SCHEDULE = "docs/benchmark/v13o-dev/schedule-draft.json"
ACCEPTANCE = "analysis/alternatives/acceptance_v1.json"
REQUIREMENTS = "docs/v13p/LOCK-REQUIREMENTS.md"
REVIEW = "docs/v13i/human-review"
TRANSCRIPTS = "docs/v13h/review-package/transcripts"
ASSETS = "docs/v13h/review-package/sources/ACTIVE"
FREEZE_COMMIT = "ee987a100896f1ea942e0e873ae833ef3c70edff"
BLOCKERS = [
    {"id": "LC-01", "item": 6, "blocker": "human presentation review PENDING: no independent reviewer has been named; "
     "the frozen record docs/v13i/human-review/REVIEW_RECORD.md is blank"},
    {"id": "LC-02", "item": 1, "blocker": "DT13O-02 enforcement not implemented: analysis checks the archive against "
     "the protocol file it is given (L15), not against the locked protocol SHA-256 in an approved lock manifest; "
     "see docs/v13q/README.md, protocol-hash enforcement plan"},
    {"id": "LC-03", "item": 3, "blocker": "the engine binary for the run is to be built from the locked commit and "
     "its SHA-256 recorded at lock time (this candidate pins the engine source tree only)"},
    {"id": "LC-04", "item": None, "blocker": "owner lock-gate approval (never automatic) and a separate owner "
     "run authorization"},
]


def sha(path):
    with open(os.path.join(ROOT, path), "rb") as f:
        return hashlib.sha256(f.read()).hexdigest()


def strict(path):
    with open(os.path.join(ROOT, path), "rb") as f:
        return v13p.strict_json(f.read())


def concat_sha(paths):
    h = hashlib.sha256()
    for p in paths:
        with open(os.path.join(ROOT, p), "rb") as f:
            h.update(f.read())
    return h.hexdigest()


def loaded_modules(entry):
    """The analysis/alternatives modules a fresh interpreter loads for `entry`, with their SHA-256."""
    code = ("import json,os,sys;sys.path.insert(0,%r);import %s;"
            "print(json.dumps(sorted(os.path.relpath(m.__file__,%r) for m in list(sys.modules.values()) "
            "if getattr(m,'__file__',None) and os.path.dirname(os.path.abspath(m.__file__))==%r)))"
            % (HERE, entry, ROOT, HERE))
    out = subprocess.run([sys.executable, "-I", "-B", "-c", code], capture_output=True, text=True, check=True,
                         cwd=ROOT, env={"PATH": os.environ.get("PATH", ""), "TMPDIR": os.environ.get("TMPDIR", ROOT)})
    return {p.replace(os.sep, "/"): sha(p) for p in json.loads(out.stdout)}


def engine_tree():
    out = subprocess.run(["git", "ls-tree", "-r", "HEAD", "internal", "cmd"], capture_output=True, check=True, cwd=ROOT)
    return hashlib.sha256(out.stdout).hexdigest()


def engine_version():
    with open(os.path.join(ROOT, "internal/version/version.go"), encoding="utf-8") as f:
        src = f.read()
    m = re.search(r'Number\s*=\s*"([0-9][^"]*)"', src)
    return m.group(1) if m else None


def build():
    raw = open(os.path.join(ROOT, PROTOCOL), "rb").read()
    p = v13p.load_protocol(raw)
    v13p.check_binding(p)                         # recorder side: the protocol binds the acceptance standard
    import v13p_verify                            # verifier side, independently (R-POLICY-001, R-PIN-001)
    pin = v13p_verify.pinned_acceptance(v13p_verify.load_protocol(raw))
    sched = strict(SCHEDULE)
    psha = hashlib.sha256(raw).hexdigest()
    if sched["protocolSha256"] != psha or len(sched["schedule"]) != p["draws"]:
        raise SystemExit("the schedule is not the protocol's")
    transcripts = sorted(os.path.join(TRANSCRIPTS, n) for n in os.listdir(os.path.join(ROOT, TRANSCRIPTS))
                         if n.endswith(".json"))
    assets = sorted(os.path.join(ASSETS, n) for n in os.listdir(os.path.join(ROOT, ASSETS)))
    texts = p["texts"]
    for t in ("presentation", "selectionRule", "viewTable"):
        if sha(texts[t]["file"]) != texts[t]["sha256"]:
            raise SystemExit(f"texts.{t} file does not match the protocol")
    for m, f in p["settings"]["files"].items():
        if sha(f["file"]) != f["sha256"]:
            raise SystemExit(f"settings file for {m} does not match the protocol")
    if sha(p["lifecycle"]["file"]) != p["lifecycle"]["sha256"]:
        raise SystemExit("lifecycle file does not match the protocol")
    items = {
        "01-protocol": {"file": PROTOCOL, "sha256": psha, "id": p["id"], "schema": p["schema"],
                        "statusInFile": p["status"]},
        "02-acceptance": {"file": ACCEPTANCE, "sha256": sha(ACCEPTANCE), "id": pin["acceptance"],
                          "equalsProtocolPin": p["acceptance"]["sha256"] == sha(ACCEPTANCE)},
        "03-recorder": {"entryPoint": "analysis/alternatives/v13p.py", "runSchedule": "v13p.run_schedule(protocol bytes)",
                        "modules": loaded_modules("v13p"),
                        "contractIdentities": {"file": "docs/v13m/CONTRACT-IDENTITIES.json",
                                               "sha256": sha("docs/v13m/CONTRACT-IDENTITIES.json"),
                                               "check": "v13m_identities.py --check"},
                        "driver": p["driver"],
                        "lifecycle": p["lifecycle"],
                        "engine": {"version": engine_version(), "sourceTreeSha256": engine_tree(),
                                   "sourceTree": "sha256 of `git ls-tree -r HEAD internal cmd`",
                                   "binary": "built from the locked commit; SHA-256 recorded at lock (LC-03)"}},
        "04-verifier": {"entryPoint": "analysis/alternatives/v13p_verify.py", "rules": ["R-PROTO-001", "R-POLICY-001",
                                                                                       "R-PIN-001"],
                        "modules": loaded_modules("v13p_verify"),
                        "data": {"analysis/alternatives/bindings_v2.json": sha("analysis/alternatives/bindings_v2.json"),
                                 ACCEPTANCE: sha(ACCEPTANCE)}},
        "05-schedule": {"file": SCHEDULE, "sha256": sha(SCHEDULE), "entries": len(sched["schedule"]),
                        "orderRule": sched["orderRule"], "protocolSha256": sched["protocolSha256"]},
        "06-presentation": {"texts": texts, "systemPromptSha256": p["systemPromptSha256"],
                            "workflow": p["workflow"],
                            "humanReviewFreeze": {"commit": FREEZE_COMMIT,
                                                  "manifest": {"file": REVIEW + "/MANIFEST.json",
                                                               "sha256": sha(REVIEW + "/MANIFEST.json")},
                                                  "manifestSums": {"file": REVIEW + "/MANIFEST.sha256",
                                                                   "sha256": sha(REVIEW + "/MANIFEST.sha256")},
                                                  "transcripts": len(transcripts),
                                                  "transcriptsConcatSha256": concat_sha(transcripts),
                                                  "staticAssets": len(assets),
                                                  "staticAssetsConcatSha256": concat_sha(assets)},
                            "humanReview": "PENDING (LC-01)"},
        "07-primaryOutcome": {"analysis": p["analysis"]},
        "08-modelSettings": {"models": p["models"], "modelOrder": p["modelOrder"], "settings": p["settings"],
                             "request": p["request"], "maxModelCallsPerDraw": p["maxModelCallsPerDraw"]},
        "09-allocation": {"draws": p["draws"], "drawsPerUnit": p["drawsPerUnit"],
                          "contextsFileSha256": p["contextsFileSha256"], "contexts": len(p["contexts"]),
                          "duplicateInputs": p["duplicateInputs"]},
        "10-callCeiling": {"maxModelCalls": p["maxModelCalls"], "maxModelCallsPerDraw": p["maxModelCallsPerDraw"]},
        "11-requestStartGuard": {"stop.wallClockSeconds": p["stop"]["wallClockSeconds"],
                                 "request.timeoutSeconds": p["request"]["timeoutSeconds"],
                                 "timeGuarantee": p["stop"]["timeGuarantee"], "tolerance": "none"},
        "12-clockTolerance": dict(p["acceptance"]["clockReconciliation"], appliedByVerifier=pin["toleranceSeconds"]),
    }
    expect = {"draws": 262, "maxModelCalls": 802, "wallClockSeconds": 14400, "toleranceSeconds": 60}
    got = {"draws": p["draws"], "maxModelCalls": p["maxModelCalls"], "wallClockSeconds": p["stop"]["wallClockSeconds"],
           "toleranceSeconds": pin["toleranceSeconds"]}
    if got != expect or len(transcripts) != 93 or len(assets) != 7:
        raise SystemExit(f"design values differ from the approved design: {got}")
    return {"schema": SCHEMA, "status": STATUS,
            "notice": "Not a lock manifest. A lock requires the blockers below to clear and the owner's lock-gate "
                      "approval; it never happens automatically. Protocol lock, live inference and V0.13.0 release: "
                      "NO GO.",
            "requirements": {"file": REQUIREMENTS, "sha256": sha(REQUIREMENTS)},
            "items": items, "blockers": BLOCKERS}


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    g = ap.add_mutually_exclusive_group(required=True)
    g.add_argument("--write")
    g.add_argument("--check")
    a = ap.parse_args()
    doc = build()
    text = json.dumps(doc, indent=1, sort_keys=True, ensure_ascii=False) + "\n"
    if a.write:
        with open(a.write, "w", encoding="utf-8") as f:
            f.write(text)
        print(f"lock candidate written: {len(doc['items'])} items, {len(doc['blockers'])} blockers; NOT LOCKED")
        return
    with open(a.check, "rb") as f:
        committed = f.read()
    v13p.strict_json(committed)
    if committed != text.encode("utf-8"):
        raise SystemExit("the lock candidate differs from the repository it pins")
    print(f"lock candidate {hashlib.sha256(committed).hexdigest()[:12]}: {len(doc['items'])} items re-derived and "
          f"equal; {len(doc['blockers'])} blockers; NOT LOCKED")


if __name__ == "__main__":
    main()
