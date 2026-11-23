#!/usr/bin/env python3
"""V0.13r protocol-lock contract: dreamteam.protocol-lock/v1 (LC-02, DT13O-02).

A lock manifest pins content: the exact protocol, acceptance standard,
schedule, recorder, verifier and analysis modules, engine binary, model
settings, presentation, outcome definition and design values. Content
identity alone confers no authority. A manifest is a lock only when its exact
bytes are committed in an owner-approved lock commit and that immutable Git
identity checks out:

* the commit is given in full (40 hex) by the operator and is an ancestor of
  (or equal to) the repository's HEAD;
* it has exactly one parent, the manifest's ``creation.baseCommit``;
* it changes only the lock files (manifest, approval record, and optionally
  the locked protocol, schedule and human-review record), and adds the
  manifest;
* the manifest file in the commit is byte-identical to the one presented;
* the protocol, schedule, approval record and human-review record named by
  the manifest are in that commit's tree with the pinned SHA-256;
* its author is the manifest's ``approval.owner``;
* the approval record names the lock id, the protocol SHA-256 and the
  decision;
* the engine source tree at that commit has the pinned hash.

A self-declared ``status: LOCKED`` or an edited approval text is therefore
not sufficient. The owner records the lock commit id in a separate decision,
and runs and analyses use that id.

``purpose`` is ``RUN`` (owner approval and an accepted human review
required) or ``TEST ONLY`` (synthetic fixtures; its approval and review
fields carry fixed TEST ONLY markers, never an owner approval). A
dreamteam.lock-candidate/v1 manifest is never a lock.

This module is the lock contract shared by the locked-analysis entry point
(v13r_analyze.py) and the runner preflight (v13r_preflight.py). It imports
neither the recorder nor the verifier. It has its own dreamteam.strict-json/v1
parser.

    python3 -I -B analysis/alternatives/v13r_lock.py check --lock FILE --lock-commit SHA [--repo DIR]
"""
import argparse
import datetime
import hashlib
import json
import math
import os
import re
import subprocess
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(os.path.dirname(HERE))

SCHEMA = "dreamteam.protocol-lock/v1"
CANDIDATE_SCHEMA = "dreamteam.lock-candidate/v1"
STATUS = "LOCKED"
RUN, TEST = "RUN", "TEST ONLY"
DECISION = {RUN: "APPROVE PROTOCOL LOCK", TEST: "TEST ONLY — NOT AN OWNER APPROVAL"}
REVIEW = {RUN: "ACCEPT", TEST: "TEST ONLY — NO HUMAN REVIEW"}
NOTICE = {RUN: "Protocol lock approved by the owner in the lock commit. A run still requires the owner's separate "
               "run authorization.",
          TEST: "TEST ONLY — NOT A PROTOCOL LOCK. Synthetic fixture for the lock contract; never a primary result."}
BINDING = "dreamteam.lock-authorization/v1"
RUN_DESIGN = {"draws": 262, "maxModelCalls": 802, "wallClockSeconds": 14400, "timeoutSeconds": 300,
              "toleranceSeconds": 60}
ENTRY = {"recorder": "analysis/alternatives/v13r_preflight.py", "verifier": "analysis/alternatives/v13p_verify.py",
         "analysis": "analysis/alternatives/v13r_analyze.py"}
LOCKED_PROTOCOL_STATUS = "LOCKED — NOT RUN"  # v13h.LOCKED, the only status v13p.py run accepts
_HEX = re.compile(r"[0-9a-f]{64}")
_COMMIT = re.compile(r"[0-9a-f]{40}")
_DATE = re.compile(r"[0-9]{4}-[0-9]{2}-[0-9]{2}")

KEYS = {
    "": {"schema", "status", "purpose", "lockId", "notice", "protocol", "acceptance", "schedule", "recorder", "verifier",
         "analysis", "engine", "models", "presentation", "outcome", "design", "approval", "creation"},
    "protocol": {"file", "sha256", "id"},
    "acceptance": {"file", "sha256"},
    "schedule": {"file", "sha256", "entries"},
    "recorder": {"entryPoint", "modules"},
    "verifier": {"entryPoint", "modules", "data"},
    "analysis": {"entryPoint", "modules"},
    "engine": {"version", "sourceTreeSha256", "binarySha256", "platform"},
    "models": {"models", "modelOrder", "settings", "request", "maxModelCallsPerDraw"},
    "presentation": {"texts", "systemPromptSha256", "workflow", "humanReviewFreeze", "humanReview"},
    "presentation.humanReviewFreeze": {"commit", "manifest", "manifestSha256", "transcripts", "transcriptsConcatSha256",
                                       "staticAssets", "staticAssetsConcatSha256"},
    "presentation.humanReview": {"verdict", "record"},
    "presentation.humanReview.record": {"file", "sha256"},
    "outcome": {"analysisSchema", "definition", "sha256"},
    "design": set(RUN_DESIGN),
    "approval": {"owner", "decision", "date", "record"},
    "approval.owner": {"name", "email"},
    "approval.record": {"file", "sha256"},
    "creation": {"tool", "candidateSha256", "baseCommit", "lockFile", "created"},
}


class LockError(Exception):
    """A lock refusal: a stable code and a message."""

    def __init__(self, code, message):
        super().__init__(f"{code}: {message}")
        self.code, self.message = code, message


def need(ok, code, message):
    if not ok:
        raise LockError(code, message)


def sha(data):
    return hashlib.sha256(data).hexdigest()


def read(path):
    with open(path, "rb") as f:
        return f.read()


def canon(value):
    return json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False, allow_nan=False)


def strict_json(data):
    """dreamteam.strict-json/v1 (lock-contract implementation)."""
    def pairs(items):
        seen = set()
        for k, _ in items:
            if k in seen:
                raise ValueError("duplicate key %r" % k)
            seen.add(k)
        return dict(items)

    def constant(name):
        raise ValueError("non-finite number " + name)

    def number(text):
        value = float(text)
        if not math.isfinite(value):
            raise ValueError("non-finite number " + text)
        return value
    if isinstance(data, str):
        data = data.encode("utf-8")
    return json.loads(data.decode("utf-8", "strict"), object_pairs_hook=pairs, parse_constant=constant,
                      parse_float=number)


def parse(raw, what="lock manifest", code="LK-PARSE"):
    try:
        return strict_json(raw)
    except ValueError as x:
        raise LockError(code, f"the {what} is not strict JSON: {x}") from None


# --- structure ---------------------------------------------------------------------

def _obj(doc, path):
    v = doc
    for part in [p for p in path.split(".") if p]:
        v = v[part]
    return v


def _str(v):
    return type(v) is str and bool(v)


def _int(v):
    return type(v) is int and v >= 0


def _modules(m):
    return isinstance(m, dict) and bool(m) and all(_str(k) and not os.path.isabs(k) and ".." not in k.split("/")
                                                   and type(h) is str and _HEX.fullmatch(h) for k, h in m.items())


def _ref(r):
    return isinstance(r, dict) and set(r) == {"file", "sha256"} and _str(r["file"]) and not os.path.isabs(r["file"]) \
        and type(r["sha256"]) is str and _HEX.fullmatch(r["sha256"]) is not None


def validate(doc):
    """Structure, types and purpose rules of dreamteam.protocol-lock/v1 (LK-SCHEMA, LK-STATUS, LK-PURPOSE)."""
    need(isinstance(doc, dict), "LK-SCHEMA", "the lock manifest is not an object")
    if doc.get("schema") == CANDIDATE_SCHEMA:
        raise LockError("LK-SCHEMA", "a lock candidate is not a lock")
    need(doc.get("schema") == SCHEMA, "LK-SCHEMA", f"schema is not {SCHEMA}")
    need(doc.get("status") == STATUS, "LK-STATUS", "status is not LOCKED")
    for path, keys in KEYS.items():
        try:
            v = _obj(doc, path)
        except (KeyError, TypeError):
            raise LockError("LK-SCHEMA", f"missing field {path}") from None
        need(isinstance(v, dict), "LK-SCHEMA", f"{path or 'manifest'} is not an object")
        need(set(v) == keys, "LK-SCHEMA", f"{path or 'manifest'}: fields "
             f"missing {sorted(keys - set(v))}, unexpected {sorted(set(v) - keys)}")
    purpose = doc["purpose"]
    need(purpose in (RUN, TEST), "LK-PURPOSE", "purpose is neither RUN nor TEST ONLY")
    need(doc["notice"] == NOTICE[purpose], "LK-SCHEMA", "notice differs")
    need(_str(doc["lockId"]), "LK-SCHEMA", "lockId")
    for k in ("protocol", "acceptance", "schedule"):
        need(_str(doc[k]["file"]) and type(doc[k]["sha256"]) is str and _HEX.fullmatch(doc[k]["sha256"]),
             "LK-SCHEMA", f"{k} file or sha256")
    need(_str(doc["protocol"]["id"]) and _int(doc["schedule"]["entries"]), "LK-SCHEMA", "protocol id or schedule entries")
    for role in ("recorder", "verifier", "analysis"):
        need(doc[role]["entryPoint"] == ENTRY[role], "LK-SCHEMA", f"{role} entry point is not {ENTRY[role]}")
        need(_modules(doc[role]["modules"]) and ENTRY[role] in doc[role]["modules"], "LK-SCHEMA", f"{role} modules")
    need(_modules(doc["verifier"]["data"]), "LK-SCHEMA", "verifier data")
    e = doc["engine"]
    need(_str(e["version"]) and _str(e["platform"]) and all(type(e[k]) is str and _HEX.fullmatch(e[k])
                                                           for k in ("sourceTreeSha256", "binarySha256")),
         "LK-SCHEMA", "engine identity")
    f = doc["presentation"]["humanReviewFreeze"]
    need(type(f["commit"]) is str and _COMMIT.fullmatch(f["commit"]) and _str(f["manifest"])
         and all(type(f[k]) is str and _HEX.fullmatch(f[k]) for k in ("manifestSha256", "transcriptsConcatSha256",
                                                                     "staticAssetsConcatSha256"))
         and _int(f["transcripts"]) and _int(f["staticAssets"]), "LK-SCHEMA", "human-review freeze identity")
    r = doc["presentation"]["humanReview"]
    need(_ref(r["record"]), "LK-SCHEMA", "human-review record")
    need(r["verdict"] == REVIEW[purpose], "LK-REVIEW",
         f"human-review verdict for a {purpose} lock must be {REVIEW[purpose]!r}")
    o = doc["outcome"]
    need(_str(o["analysisSchema"]) and isinstance(o["definition"], dict) and type(o["sha256"]) is str
         and o["sha256"] == sha(canon(o["definition"]).encode("utf-8")), "LK-OUTCOME",
         "outcome definition and its SHA-256 differ")
    d = doc["design"]
    need(all(_int(d[k]) for k in d), "LK-SCHEMA", "design values are not non-negative integers")
    if purpose == RUN:
        need(d == RUN_DESIGN, "LK-DESIGN", f"a RUN lock must pin the approved design {RUN_DESIGN}")
    a = doc["approval"]
    need(_str(a["owner"]["name"]) and _str(a["owner"]["email"]) and _ref(a["record"])
         and type(a["date"]) is str and _DATE.fullmatch(a["date"]), "LK-APPROVAL", "approval record fields")
    need(a["decision"] == DECISION[purpose], "LK-APPROVAL",
         f"the approval decision for a {purpose} lock must be {DECISION[purpose]!r}")
    c = doc["creation"]
    need(type(c["baseCommit"]) is str and _COMMIT.fullmatch(c["baseCommit"]) and _str(c["lockFile"])
         and _str(c["tool"]) and type(c["candidateSha256"]) is str and _HEX.fullmatch(c["candidateSha256"])
         and type(c["created"]) is str and _DATE.fullmatch(c["created"]), "LK-SCHEMA", "creation identity")
    return doc


def load(raw):
    return validate(parse(raw))


# --- the lock commit ---------------------------------------------------------------

def git(repo, *args, check=True):
    r = subprocess.run(["git", "-C", repo, *args], capture_output=True)
    if check and r.returncode != 0:
        raise LockError("LK-COMMIT", f"git {args[0]} failed: {r.stderr.decode('utf-8', 'replace').strip()[:200]}")
    return r


def engine_tree(repo, commit):
    return sha(git(repo, "ls-tree", "-r", commit, "internal", "cmd").stdout)


def verify_commit(doc, raw, commit, repo=ROOT):
    """The lock's immutable Git identity (LK-COMMIT, LK-APPROVAL, LK-REVIEW)."""
    need(type(commit) is str and _COMMIT.fullmatch(commit) is not None, "LK-COMMIT",
         "the lock commit must be a full 40-hex commit id")
    need(git(repo, "cat-file", "-t", commit, check=False).stdout.strip() == b"commit", "LK-COMMIT",
         "the lock commit is not in the repository")
    need(git(repo, "merge-base", "--is-ancestor", commit, "HEAD", check=False).returncode == 0, "LK-COMMIT",
         "the lock commit is not in the history of HEAD")
    parents = git(repo, "rev-list", "--parents", "-n", "1", commit).stdout.split()[1:]
    c = doc["creation"]
    need([p.decode() for p in parents] == [c["baseCommit"]], "LK-COMMIT",
         "the lock commit's single parent is not creation.baseCommit")

    def blob(path):
        r = git(repo, "show", f"{commit}:{path}", check=False)
        need(r.returncode == 0, "LK-COMMIT", f"{path} is not in the lock commit")
        return r.stdout
    need(blob(c["lockFile"]) == raw, "LK-COMMIT", "the lock manifest differs from the one in the lock commit")
    changed = git(repo, "diff-tree", "--no-commit-id", "--name-status", "-r", c["baseCommit"], commit).stdout
    changes = dict(reversed(line.split("\t", 1)) for line in changed.decode("utf-8").splitlines())
    allowed = {c["lockFile"], doc["approval"]["record"]["file"], doc["presentation"]["humanReview"]["record"]["file"],
               doc["protocol"]["file"], doc["schedule"]["file"]}
    need(set(changes) <= allowed, "LK-COMMIT",
         f"the lock commit changes files other than the lock files: {sorted(set(changes) - allowed)}")
    need(changes.get(c["lockFile"]) == "A", "LK-COMMIT", "the lock commit does not add the lock manifest")
    for k in ("protocol", "schedule"):
        need(sha(blob(doc[k]["file"])) == doc[k]["sha256"], "LK-COMMIT", f"the lock commit's {k} is not the pinned one")
    author = git(repo, "log", "-1", "--format=%an%x00%ae", commit).stdout.decode("utf-8").strip().split("\x00")
    o = doc["approval"]["owner"]
    need(author == [o["name"], o["email"]], "LK-APPROVAL", "the lock commit's author is not approval.owner")
    record = blob(doc["approval"]["record"]["file"])
    need(sha(record) == doc["approval"]["record"]["sha256"], "LK-APPROVAL",
         "the approval record differs from the pinned one")
    text = record.decode("utf-8", "replace")
    for line in (f"Lock: {doc['lockId']}", f"Protocol SHA-256: {doc['protocol']['sha256']}",
                 f"Decision: {doc['approval']['decision']}", f"Owner: {o['name']} <{o['email']}>"):
        need(line in text.splitlines(), "LK-APPROVAL", f"the approval record does not state {line.split(':')[0]}")
    review = blob(doc["presentation"]["humanReview"]["record"]["file"])
    need(sha(review) == doc["presentation"]["humanReview"]["record"]["sha256"], "LK-REVIEW",
         "the human-review record differs from the pinned one")
    need(f"Verdict: {doc['presentation']['humanReview']['verdict']}" in review.decode("utf-8", "replace").splitlines(),
         "LK-REVIEW", "the human-review record does not state the pinned verdict")
    need(engine_tree(repo, commit) == doc["engine"]["sourceTreeSha256"], "LK-ENGINE",
         "the engine source tree at the lock commit is not the pinned one")
    return {"lockId": doc["lockId"], "purpose": doc["purpose"], "lockSha256": sha(raw), "lockCommit": commit}


def binding(identity, protocol_sha):
    """The lock binding a preflighted run records at the start of its authorization."""
    return (f"{BINDING} lock={identity['lockSha256']} commit={identity['lockCommit']} protocol={protocol_sha} "
            f"purpose={identity['purpose']};")


# --- loaded modules ----------------------------------------------------------------

def loaded_modules(entry_dir):
    """Every repository file loaded as a module in this process, by repository path, with SHA-256.

    Modules in ``entry_dir`` (the entry point's directory) are keyed as
    analysis/alternatives/<name>; other modules under the repository by their
    repository path. Modules from outside the repository (the interpreter,
    its site configuration) are the environment and are not pinned; a pinned
    module loaded from anywhere else is therefore missing, and refused by
    check_modules."""
    entry_dir, root = os.path.realpath(entry_dir), os.path.realpath(ROOT)
    out = {}
    for m in list(sys.modules.values()):
        f = getattr(m, "__file__", None)
        if not f:
            continue
        f = os.path.realpath(f)
        if os.path.dirname(f) == entry_dir:
            key = "analysis/alternatives/" + os.path.basename(f)
        elif f.startswith(root + os.sep):
            key = os.path.relpath(f, root).replace(os.sep, "/")
        else:
            continue
        out[key] = sha(read(f))
    return out


def check_modules(expected, entry_dir):
    got = loaded_modules(entry_dir)
    need(set(got) == set(expected), "LK-MODULE", f"loaded modules differ from the lock: missing "
         f"{sorted(set(expected) - set(got))}, unpinned {sorted(set(got) - set(expected))}")
    bad = sorted(k for k in got if got[k] != expected[k])
    need(not bad, "LK-MODULE", f"module bytes differ from the lock: {bad}")
    return got


def module_identities(entry):
    """The modules a fresh interpreter loads for `entry` (creation only)."""
    code = ("import json,sys\nsys.path.insert(0,%r)\nimport %s\nhad='v13r_lock' in sys.modules\n"
            "import v13r_lock as L\nm=L.loaded_modules(%r)\n"
            "if not had: m.pop('analysis/alternatives/v13r_lock.py',None)\nprint(json.dumps(m))" % (HERE, entry, HERE))
    out = subprocess.run([sys.executable, "-I", "-B", "-c", code], capture_output=True, text=True, check=True,
                         cwd=ROOT, env={"PATH": os.environ.get("PATH", ""), "TMPDIR": os.environ.get("TMPDIR", ROOT)})
    return json.loads(out.stdout)


# --- content checks shared by analysis and preflight --------------------------------

def check_content(doc, protocol_raw, proto, schedule_raw, sched, binary, acceptance_path):
    """Protocol, acceptance, schedule, engine, models, presentation, outcome and design (LK-*)."""
    psha = sha(protocol_raw)
    need(psha == doc["protocol"]["sha256"], "LK-PROTOCOL", "the protocol bytes are not the locked protocol")
    need(proto.get("id") == doc["protocol"]["id"], "LK-PROTOCOL", "the protocol id is not the locked id")
    need(proto.get("status") == LOCKED_PROTOCOL_STATUS, "LK-PROTOCOL", "the protocol's status is not locked")
    acc = read(acceptance_path)
    need(sha(acc) == doc["acceptance"]["sha256"] == (proto.get("acceptance") or {}).get("sha256"), "LK-ACCEPTANCE",
         "the acceptance file is not the locked one")
    need(sha(schedule_raw) == doc["schedule"]["sha256"], "LK-SCHEDULE", "the schedule file is not the locked one")
    s = parse(schedule_raw, "schedule", "LK-SCHEDULE")
    need(isinstance(s, dict) and s.get("protocolSha256") == psha and s.get("protocol") == proto["id"]
         and s.get("schedule") == sched and s.get("draws") == len(sched) == doc["schedule"]["entries"],
         "LK-SCHEDULE", "the schedule is not the locked protocol's")
    need(sha(read(binary)) == doc["engine"]["binarySha256"], "LK-ENGINE", "the engine binary is not the locked one")
    m = doc["models"]
    need(m == {"models": proto.get("models"), "modelOrder": proto.get("modelOrder"), "settings": proto.get("settings"),
               "request": proto.get("request"), "maxModelCallsPerDraw": proto.get("maxModelCallsPerDraw")},
         "LK-MODEL", "the locked model configuration is not the protocol's")
    for name, f in sorted(m["settings"]["files"].items()):
        need(sha(read(os.path.join(ROOT, f["file"]))) == f["sha256"], "LK-MODEL", f"model settings for {name} differ")
    pr = doc["presentation"]
    need(pr["texts"] == proto.get("texts") and pr["systemPromptSha256"] == proto.get("systemPromptSha256")
         and pr["workflow"] == proto.get("workflow"), "LK-PRESENTATION", "the locked presentation is not the protocol's")
    for k in ("presentation", "selectionRule", "viewTable"):
        need(sha(read(os.path.join(ROOT, pr["texts"][k]["file"]))) == pr["texts"][k]["sha256"], "LK-PRESENTATION",
             f"presentation text {k} differs")
    fz = pr["humanReviewFreeze"]
    need(sha(read(os.path.join(ROOT, fz["manifest"]))) == fz["manifestSha256"], "LK-PRESENTATION",
         "the frozen human-review manifest differs")
    o = doc["outcome"]
    need(o["definition"] == proto.get("analysis"), "LK-OUTCOME", "the locked outcome definition is not the protocol's")
    d = {"draws": proto.get("draws"), "maxModelCalls": proto.get("maxModelCalls"),
         "wallClockSeconds": (proto.get("stop") or {}).get("wallClockSeconds"),
         "timeoutSeconds": (proto.get("request") or {}).get("timeoutSeconds"),
         "toleranceSeconds": ((proto.get("acceptance") or {}).get("clockReconciliation") or {}).get("toleranceSeconds")}
    need(d == doc["design"], "LK-DESIGN", "the locked design values are not the protocol's")
    return psha


# --- creation (candidate or TEST ONLY; a RUN lock needs owner-written records) -------

def locked_protocol(draft):
    """The draft protocol object with the locked status."""
    return dict(draft, status=LOCKED_PROTOCOL_STATUS)


def create(*, purpose, lock_id, protocol_file, protocol_raw, schedule_file, schedule_raw, binary, platform,
           candidate_raw, base_commit, lock_file, owner, date, approval_file, approval_raw, review_file, review_raw,
           acceptance_path, repo=ROOT):
    """A dreamteam.protocol-lock/v1 manifest for the given content. The approval
    and human-review records are supplied by the caller (owner-written for
    RUN; fixed TEST ONLY markers for TEST ONLY) and never invented here."""
    proto = parse(protocol_raw, "protocol", "LK-PROTOCOL")
    fz_manifest = "docs/v13i/human-review/MANIFEST.json"
    cand = parse(candidate_raw, "lock candidate", "LK-SCHEMA")
    need(cand.get("schema") == CANDIDATE_SCHEMA, "LK-SCHEMA", "the source is not a lock candidate")
    freeze = cand["items"]["06-presentation"]["humanReviewFreeze"]
    doc = {
        "schema": SCHEMA, "status": STATUS, "purpose": purpose, "lockId": lock_id, "notice": NOTICE[purpose],
        "protocol": {"file": protocol_file, "sha256": sha(protocol_raw), "id": proto["id"]},
        "acceptance": {"file": "analysis/alternatives/acceptance_v1.json", "sha256": sha(read(acceptance_path))},
        "schedule": {"file": schedule_file, "sha256": sha(schedule_raw),
                     "entries": len(parse(schedule_raw, "schedule", "LK-SCHEDULE")["schedule"])},
        "recorder": {"entryPoint": ENTRY["recorder"], "modules": module_identities("v13r_preflight")},
        "verifier": {"entryPoint": ENTRY["verifier"], "modules": module_identities("v13p_verify"),
                     "data": {k: sha(read(os.path.join(ROOT, k))) for k in
                              ("analysis/alternatives/acceptance_v1.json", "analysis/alternatives/bindings_v2.json")}},
        "analysis": {"entryPoint": ENTRY["analysis"], "modules": {}},
        "engine": {"version": _engine_version(), "sourceTreeSha256": engine_tree(repo, base_commit),
                   "binarySha256": sha(read(binary)), "platform": platform},
        "models": {"models": proto["models"], "modelOrder": proto["modelOrder"], "settings": proto["settings"],
                   "request": proto["request"], "maxModelCallsPerDraw": proto["maxModelCallsPerDraw"]},
        "presentation": {"texts": proto["texts"], "systemPromptSha256": proto["systemPromptSha256"],
                         "workflow": proto["workflow"],
                         "humanReviewFreeze": {"commit": freeze["commit"], "manifest": fz_manifest,
                                               "manifestSha256": freeze["manifest"]["sha256"],
                                               "transcripts": freeze["transcripts"],
                                               "transcriptsConcatSha256": freeze["transcriptsConcatSha256"],
                                               "staticAssets": freeze["staticAssets"],
                                               "staticAssetsConcatSha256": freeze["staticAssetsConcatSha256"]},
                         "humanReview": {"verdict": REVIEW[purpose], "record": {"file": review_file,
                                                                                "sha256": sha(review_raw)}}},
        "outcome": {"analysisSchema": "dreamteam.alternatives-analysis/v10", "definition": proto["analysis"],
                    "sha256": sha(canon(proto["analysis"]).encode("utf-8"))},
        "design": {"draws": proto["draws"], "maxModelCalls": proto["maxModelCalls"],
                   "wallClockSeconds": proto["stop"]["wallClockSeconds"],
                   "timeoutSeconds": proto["request"]["timeoutSeconds"],
                   "toleranceSeconds": proto["acceptance"]["clockReconciliation"]["toleranceSeconds"]},
        "approval": {"owner": owner, "decision": DECISION[purpose], "date": date,
                     "record": {"file": approval_file, "sha256": sha(approval_raw)}},
        "creation": {"tool": "analysis/alternatives/v13r_lock.py", "candidateSha256": sha(candidate_raw),
                     "baseCommit": base_commit, "lockFile": lock_file, "created": date},
    }
    analysis = module_identities("v13r_analyze")
    doc["analysis"]["modules"] = {k: v for k, v in analysis.items() if k not in doc["verifier"]["modules"]}
    return validate(doc)


def approval_text(doc_lock_id, protocol_sha, purpose, owner, extra=""):
    """The lines an approval record must state. For RUN, the owner writes the
    record; this only lists the required lines."""
    return (f"Lock: {doc_lock_id}\nProtocol SHA-256: {protocol_sha}\nDecision: {DECISION[purpose]}\n"
            f"Owner: {owner['name']} <{owner['email']}>\n{extra}")


def _engine_version():
    src = read(os.path.join(ROOT, "internal/version/version.go")).decode("utf-8")
    m = re.search(r'Number\s*=\s*"([0-9][^"]*)"', src)
    return m.group(1) if m else "unknown"


def encode(doc):
    return (json.dumps(doc, indent=1, sort_keys=True, ensure_ascii=False) + "\n").encode("utf-8")


def today():
    return datetime.date.today().isoformat()


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    sub = ap.add_subparsers(dest="cmd", required=True)
    p = sub.add_parser("check", help="strict structure and Git identity of a lock (no content checks)")
    p.add_argument("--lock", required=True)
    p.add_argument("--lock-commit", required=True)
    p.add_argument("--repo", default=ROOT)
    a = ap.parse_args(argv)
    try:
        raw = read(a.lock)
        ident = verify_commit(load(raw), raw, a.lock_commit, a.repo)
    except LockError as x:
        raise SystemExit(f"LOCK REFUSED {x}")
    print(json.dumps(ident, sort_keys=True))


if __name__ == "__main__":
    main()
