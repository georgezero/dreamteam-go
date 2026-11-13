#!/usr/bin/env python3
"""V0.13o independent archive verifier and analyzer: the V0.13m-lite verifier,
unchanged, behind a protocol acceptance-standard binding (DT13M-01).

R-PIN-001. Before classifying or analysing an archive, the verifier requires
the archive's protocol (bound to the archive by SHA-256) to carry an
``acceptance`` field that

* names dreamteam.acceptance-standard/v1 and analysis/alternatives/
  acceptance_v1.json;
* gives the SHA-256 of the exact acceptance-file bytes this verifier reads,
  and those bytes parse strictly (dreamteam.strict-json/v1) to the same
  relationships the V0.13m-lite verifier applies;
* states the clock-reconciliation policy: R-HALT-003, the integer
  toleranceSeconds the applied standard contains, and the scope, excess and
  operational-assumption statements below, verbatim.

Any difference makes the archive INVALID (classification) and refuses
analysis. A changed acceptance file therefore cannot change the tolerance
applied to an archive recorded under a protocol that pinned the original:
its hash no longer matches. Every other check is the V0.13m-lite verifier's.
This module does not import any recorder.

    python3 -I -B analysis/alternatives/v13o_verify.py classify --protocol FILE --contexts DIR --dir RUN
    python3 -I -B analysis/alternatives/v13o_verify.py analyze --binary ./dt --protocol FILE --contexts DIR \\
        --replays DIR --dir RUN --out DIR
"""
import argparse
import hashlib
import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import v13m_verify as V  # noqa: E402  (the V0.13m-lite verifier, unchanged; not a recorder)

H = V.H
PIN_RULE = "R-PIN-001"
ACCEPTANCE_ID = "dreamteam.acceptance-standard/v1"
ACCEPTANCE_FILE = "analysis/alternatives/acceptance_v1.json"
CLOCK_RULE = "R-HALT-003"
# The verifier's own copy of the policy statements: a protocol that weakens
# or omits them is refused.
SCOPE = ("reconciles each halt's evidence elapsedSeconds with the recorded wall-clock (SYSTEM_UTC) journal and "
         "call-journal timestamps only; it does not change the four-hour request-start guard "
         "(stop.wallClockSeconds, request.timeoutSeconds), which is applied without tolerance")
ON_EXCESS = ("INVALID: a discrepancy above toleranceSeconds rejects the archive, with no retry, resumption or "
             "replacement draw")
BASIS = ("operational assumption that the monotonic and UTC clocks of the recording host agree within "
         "toleranceSeconds over the run; it is not a measured clock-accuracy guarantee")
KEYS = {"id", "file", "sha256", "clockReconciliation"}
POLICY_KEYS = {"rule", "toleranceSeconds", "scope", "onExcess", "basis"}


def pinned_acceptance(proto, path=None):
    """R-PIN-001: the acceptance standard the protocol pins, which must be the
    one this verifier applies. Returns its facts or raises V.Invalid."""
    def need(ok, message):
        if not ok:
            raise V.Invalid(PIN_RULE, message)

    b = proto.get("acceptance") if isinstance(proto, dict) else None
    need(isinstance(b, dict) and set(b) == KEYS, "the protocol does not bind an acceptance standard")
    need(V._same(b["id"], ACCEPTANCE_ID) and V._same(b["file"], ACCEPTANCE_FILE),
         "the protocol binds a different acceptance standard")
    raw = open(path or os.path.join(HERE, "acceptance_v1.json"), "rb").read()
    need(V._same(b["sha256"], hashlib.sha256(raw).hexdigest()),
         "the acceptance file is not the one the protocol pins (SHA-256 differs)")
    try:
        doc = V._strict_json(raw)
    except ValueError as x:
        raise V.Invalid(PIN_RULE, f"the acceptance file is not strict JSON: {x}") from None
    need(isinstance(doc, dict) and V._same(doc.get("schema"), ACCEPTANCE_ID), "acceptance standard identity")
    applied = {r["id"]: r for r in doc["relationships"]}
    need(V._same(applied, V.ACCEPTANCE), "the applied acceptance standard is not the pinned file's")
    c = b["clockReconciliation"]
    need(isinstance(c, dict) and set(c) == POLICY_KEYS, "the protocol's clock-reconciliation policy is incomplete")
    tol = V.ACCEPTANCE[CLOCK_RULE]["toleranceSeconds"]
    need(type(tol) is int and V._same(c["toleranceSeconds"], tol) and V._same(c["rule"], CLOCK_RULE),
         "the protocol's clock tolerance is not the applied standard's")
    need(V._same(c["scope"], SCOPE) and V._same(c["onExcess"], ON_EXCESS) and V._same(c["basis"], BASIS),
         "the protocol's clock-reconciliation statement differs (scope, excess rule or operational assumption)")
    return {"acceptance": b["id"], "sha256": b["sha256"], "toleranceSeconds": tol}


def classify(directory, proto, proto_sha, sched, contexts=None, allow_synthetic=False):
    """V.classify, INVALID as well when the protocol does not pin the applied standard."""
    verdict = V.classify(directory, proto, proto_sha, sched, contexts, allow_synthetic)
    try:
        verdict.setdefault("derived", {})
        if isinstance(verdict["derived"], dict):
            verdict["derived"]["acceptancePin"] = pinned_acceptance(proto)
    except V.Invalid as x:
        verdict["violations"] = [{"invariant": x.invariant, "message": x.message}] + verdict["violations"]
        verdict["state"] = "INVALID"
    return verdict


def analyze(binary, run_dirs, ctxs, proto, sched, directory, proto_sha, allow_synthetic=False):
    """V.analyze, refused unless the protocol pins the applied standard."""
    pin = pinned_acceptance(proto)
    doc = V.analyze(binary, run_dirs, ctxs, proto, sched, directory, proto_sha, allow_synthetic)
    doc["acceptancePin"] = pin
    return doc


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    sub = ap.add_subparsers(dest="cmd", required=True)
    for name in ("classify", "analyze"):
        p = sub.add_parser(name)
        p.add_argument("--protocol", required=True)
        p.add_argument("--contexts", required=True)
        p.add_argument("--dir", required=True)
        if name == "analyze":
            p.add_argument("--binary", required=True)
            p.add_argument("--replays")
            p.add_argument("--out", required=True)
    a = ap.parse_args(argv)
    proto, proto_sha, sched, rows = V._inputs(a)
    if a.cmd == "classify":
        verdict = classify(a.dir, proto, proto_sha, sched, {c["id"]: c for c in rows})
        print(json.dumps(verdict, indent=1, sort_keys=True))
        raise SystemExit(0 if verdict["state"] != "INVALID" else 1)
    ctxs = H.Contexts(proto, rows, a.replays, {V.SB.SETTINGS_ENV: V.SB.REPLAY_URL})
    doc = analyze(a.binary, ctxs.run_dirs, ctxs, proto, sched, a.dir, proto_sha)
    H.write_json(os.path.join(a.out, "analysis.json"), doc)
    print(f"analysis: {len(doc['draws'])} completed draws ({doc['runStatus']}), {len(doc['executions'])} executions")


if __name__ == "__main__":
    main()
