#!/usr/bin/env python3
"""V0.13r locked analysis: the only analysis entry point for a locked run
(LC-02, DT13O-02).

``--lock`` and ``--lock-commit`` are required. There is no fallback to
unlocked analysis. Before the existing verifier (v13p_verify.py, unchanged)
classifies or analyses anything, this entry point refuses unless every one of
these holds (codes in docs/v13r/README.md):

1. the lock parses strictly (LK-PARSE) and is a dreamteam.protocol-lock/v1
   manifest with every field (LK-SCHEMA), status LOCKED (LK-STATUS), and a
   purpose matching ``--test-only`` (LK-PURPOSE);
2. its immutable Git identity checks out: the lock commit, the exact
   manifest bytes, the approval record, the human-review record and the
   engine source tree (LK-COMMIT, LK-APPROVAL, LK-REVIEW, LK-ENGINE);
3. every repository module loaded in this process is exactly the lock's
   verifier and analysis modules, byte for byte (LK-MODULE);
4. the protocol is the locked protocol's exact bytes, with the locked status
   (LK-PROTOCOL);
5. the acceptance file is the locked one and passes R-POLICY-001 and
   R-PIN-001 (LK-ACCEPTANCE);
6. the schedule file is the locked one and is the protocol's schedule
   (LK-SCHEDULE);
7. the engine binary is the locked one (LK-ENGINE);
8. the model settings, presentation, outcome definition and design values
   are the locked ones (LK-MODEL, LK-PRESENTATION, LK-OUTCOME, LK-DESIGN);
9. the archive is bound to it: run.json and the seal name the locked
   protocol SHA-256, and the run's authorization starts with this lock's
   binding (LK-ARCHIVE).

An unreadable input is refused (LK-IO).

Only then does it call v13p_verify.classify or v13p_verify.analyze with the
same arguments the unlocked entry point would use. The primary estimand is
not touched. A TEST ONLY lock is accepted only with ``--test-only``, and the
output is marked TEST ONLY.

    python3 -I -B analysis/alternatives/v13r_analyze.py classify --lock FILE --lock-commit SHA \\
        --protocol FILE --schedule FILE --contexts DIR --binary ./dt --dir RUN
    python3 -I -B analysis/alternatives/v13r_analyze.py analyze  --lock FILE --lock-commit SHA \\
        --protocol FILE --schedule FILE --contexts DIR --binary ./dt --dir RUN --replays DIR --out DIR
"""
import argparse
import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import v13r_lock as L  # noqa: E402  (the lock contract; imports no recorder or verifier)
import v13p_verify as PV  # noqa: E402  (the V0.13p verifier, unchanged; not a recorder)

V, H = PV.V, PV.H
ACCEPTANCE = os.path.join(HERE, "acceptance_v1.json")


def _archive(directory, psha, proto_id, bind):
    """LK-ARCHIVE: the archive names the locked protocol and carries this lock's binding."""
    live = os.path.join(directory, "live")
    try:
        doc = L.parse(L.read(os.path.join(live, "run.json")), "archive manifest", "LK-ARCHIVE")
    except OSError:
        raise L.LockError("LK-ARCHIVE", "the archive has no run.json") from None
    L.need(isinstance(doc, dict) and doc.get("protocolSha256") == psha and doc.get("protocol") == proto_id,
           "LK-ARCHIVE", "the archive is not bound to the locked protocol")
    auth = doc.get("authorization")
    L.need(isinstance(auth, str) and auth.startswith(bind) and auth[len(bind):].strip(), "LK-ARCHIVE",
           "the archive was not recorded under this lock (no lock binding in its authorization)")
    seal = os.path.join(live, V.SEAL)
    if os.path.exists(seal):
        s = L.parse(L.read(seal), "completion seal", "LK-ARCHIVE")
        L.need(isinstance(s, dict) and s.get("protocol") == {"id": proto_id, "sha256": psha}, "LK-ARCHIVE",
               "the completion seal is not bound to the locked protocol")


def check(a):
    """Every lock check, in order; returns what the existing verifier needs. Raises L.LockError."""
    raw = L.read(a.lock)
    doc = L.load(raw)
    L.need((doc["purpose"] == L.TEST) == a.test_only, "LK-PURPOSE",
           "a TEST ONLY lock requires --test-only, and --test-only accepts only a TEST ONLY lock")
    ident = L.verify_commit(doc, raw, a.lock_commit, a.repo)
    expected = dict(doc["verifier"]["modules"], **doc["analysis"]["modules"])
    L.check_modules(expected, HERE)
    L.need(L.sha(L.read(os.path.join(HERE, "bindings_v2.json"))) ==
           doc["verifier"]["data"].get("analysis/alternatives/bindings_v2.json"), "LK-MODULE",
           "the lifecycle bindings file differs from the lock")
    praw = L.read(a.protocol)
    try:
        proto = PV.load_protocol(praw)
    except V.Invalid as x:
        raise L.LockError("LK-PROTOCOL", str(x)) from None
    rows, csha = H.load_contexts(a.contexts)
    L.need(csha == proto.get("contextsFileSha256"), "LK-PROTOCOL", "the contexts file is not the protocol's")
    sched = H.schedule(proto, rows)
    psha = L.sha(praw)
    L.need(psha == doc["protocol"]["sha256"], "LK-PROTOCOL", "the protocol bytes are not the locked protocol")
    L.check_content(doc, praw, proto, L.read(a.schedule), sched, a.binary, ACCEPTANCE)
    try:
        PV.pinned_acceptance(proto)
    except V.Invalid as x:
        raise L.LockError("LK-ACCEPTANCE", str(x)) from None
    _archive(a.dir, psha, proto.get("id"), L.binding(ident, psha))
    return proto, psha, sched, rows, ident, expected


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    sub = ap.add_subparsers(dest="cmd", required=True)
    for name in ("classify", "analyze"):
        p = sub.add_parser(name)
        p.add_argument("--lock", required=True)
        p.add_argument("--lock-commit", required=True)
        p.add_argument("--repo", default=L.ROOT)
        p.add_argument("--protocol", required=True)
        p.add_argument("--schedule", required=True)
        p.add_argument("--contexts", required=True)
        p.add_argument("--binary", required=True)
        p.add_argument("--dir", required=True)
        p.add_argument("--test-only", action="store_true")
        if name == "analyze":
            p.add_argument("--replays")
            p.add_argument("--out", required=True)
    a = ap.parse_args(argv)
    try:
        try:
            proto, psha, sched, rows, ident, expected = check(a)
        except OSError as x:
            raise L.LockError("LK-IO", f"cannot read an input: {x.strerror}: {os.path.basename(str(x.filename))}") \
                from None
    except L.LockError as x:
        if a.cmd == "classify":
            print(json.dumps({"state": "INVALID", "lockRefused": True,
                              "violations": [{"invariant": x.code, "message": x.message}]}, indent=1, sort_keys=True))
            raise SystemExit(1)
        raise SystemExit(f"LOCK REFUSED {x}")
    lock = dict(ident, protocolSha256=psha,
                marker="TEST ONLY — NOT A PRIMARY RESULT" if ident["purpose"] == L.TEST else "LOCKED RUN")
    if a.cmd == "classify":
        verdict = PV.classify(a.dir, proto, psha, sched, {c["id"]: c for c in rows})
        L.check_modules(expected, HERE)  # nothing unpinned was loaded during classification
        verdict["lock"] = lock
        print(json.dumps(verdict, indent=1, sort_keys=True))
        raise SystemExit(0 if verdict["state"] != "INVALID" else 1)
    ctxs = H.Contexts(proto, rows, a.replays, {V.SB.SETTINGS_ENV: V.SB.REPLAY_URL})
    doc = PV.analyze(a.binary, ctxs.run_dirs, ctxs, proto, sched, a.dir, psha)
    try:
        L.check_modules(expected, HERE)  # nothing unpinned was loaded during analysis
    except L.LockError as x:
        raise SystemExit(f"LOCK REFUSED {x}")
    doc["lock"] = lock
    H.write_json(os.path.join(a.out, "analysis.json"), doc)
    print(f"locked analysis ({lock['marker']}): {len(doc['draws'])} completed draws ({doc['runStatus']}), "
          f"{len(doc['executions'])} executions")


if __name__ == "__main__":
    main()
