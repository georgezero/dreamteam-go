#!/usr/bin/env python3
"""V0.13p independent archive verifier and analyzer: the V0.13o verifier,
unchanged, behind strict policy and protocol parsing (DT13O-01, DT13O-04).

Both checks use dreamteam.strict-json/v1 (the V0.13m-lite verifier's parser:
strict UTF-8, no duplicate object key at any nesting level, no NaN/Infinity,
no number that overflows to infinity, exact integers, no trailing content).

R-PROTO-001. The protocol file must parse strictly to an object. V0.13o read
it with json.load, so a duplicated key was silently resolved last-key-wins.
A protocol that does not parse strictly makes the archive INVALID and
refuses analysis.

R-POLICY-001. Before R-PIN-001, the acceptance file must parse strictly to a
well-formed policy: an object whose schema is dreamteam.acceptance-standard/v1,
with a non-empty ``relationships`` list of objects, each with a non-empty
string ``id``; identifiers are unique (V0.13o collapsed a duplicate
R-HALT-003, applying the last); R-HALT-003 carries a positive integer
``toleranceSeconds``. Otherwise the archive is INVALID and analysis is refused.

Then R-PIN-001 and every other check are V0.13o's and V0.13m-lite's,
unchanged. This module does not import any recorder.

    python3 -I -B analysis/alternatives/v13p_verify.py classify --protocol FILE --contexts DIR --dir RUN
    python3 -I -B analysis/alternatives/v13p_verify.py analyze --binary ./dt --protocol FILE --contexts DIR \\
        --replays DIR --dir RUN --out DIR
"""
import argparse
import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import v13o_verify as OV  # noqa: E402  (the V0.13o verifier, unchanged; not a recorder)

V, H = OV.V, OV.H
POLICY_RULE = "R-POLICY-001"
PROTOCOL_RULE = "R-PROTO-001"


def _bytes(path):
    with open(path, "rb") as f:
        return f.read()


def load_protocol(raw):
    """R-PROTO-001: the protocol from its exact bytes, or V.Invalid."""
    try:
        proto = V._strict_json(raw)
    except ValueError as x:
        raise V.Invalid(PROTOCOL_RULE, f"the protocol file is not strict JSON: {x}") from None
    except UnicodeDecodeError as x:
        raise V.Invalid(PROTOCOL_RULE, f"the protocol file is not UTF-8: {x}") from None
    if not isinstance(proto, dict):
        raise V.Invalid(PROTOCOL_RULE, "the protocol file is not an object")
    return proto


def policy(path=None):
    """R-POLICY-001: the acceptance file is a strictly parsed, well-formed policy."""
    raw = _bytes(path or os.path.join(HERE, "acceptance_v1.json"))
    try:
        doc = V._strict_json(raw)
    except (ValueError, UnicodeDecodeError) as x:
        raise V.Invalid(POLICY_RULE, f"the acceptance file is not strict JSON: {x}") from None
    if not isinstance(doc, dict) or not V._same(doc.get("schema"), OV.ACCEPTANCE_ID):
        raise V.Invalid(POLICY_RULE, "the acceptance file is not dreamteam.acceptance-standard/v1")
    rel = doc.get("relationships")
    if not isinstance(rel, list) or not rel:
        raise V.Invalid(POLICY_RULE, "the acceptance file has no relationships list")
    ids = []
    for r in rel:
        if not isinstance(r, dict) or not isinstance(r.get("id"), str) or not r["id"]:
            raise V.Invalid(POLICY_RULE, "a relationship is not an object with a non-empty string id")
        if r["id"] in ids:
            raise V.Invalid(POLICY_RULE, f"duplicate relationship id {r['id']}")
        ids.append(r["id"])
    clock = [r for r in rel if r["id"] == OV.CLOCK_RULE]
    tol = clock[0].get("toleranceSeconds") if clock else None
    if not (V._nat(tol) and tol > 0):
        raise V.Invalid(POLICY_RULE, "R-HALT-003 toleranceSeconds is not a positive integer")
    return ids


def pinned_acceptance(proto, path=None):
    """R-POLICY-001, then R-PIN-001 (V0.13o)."""
    policy(path)
    return OV.pinned_acceptance(proto, path)


def classify(directory, proto, proto_sha, sched, contexts=None, allow_synthetic=False):
    """V.classify, INVALID as well when the policy or the pin fails."""
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
    """V.analyze, refused unless the policy and the pin hold."""
    pin = pinned_acceptance(proto)
    doc = V.analyze(binary, run_dirs, ctxs, proto, sched, directory, proto_sha, allow_synthetic)
    doc["acceptancePin"] = pin
    return doc


def _inputs(a):
    """V._inputs, with the protocol parsed strictly (R-PROTO-001)."""
    raw = _bytes(a.protocol)
    proto = load_protocol(raw)
    rows, csha = H.load_contexts(a.contexts)
    if csha != proto.get("contextsFileSha256"):
        raise SystemExit("the contexts file is not the protocol's")
    return proto, V.sha256(raw), H.schedule(proto, rows), rows


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
    try:
        proto, proto_sha, sched, rows = _inputs(a)
    except V.Invalid as x:
        if a.cmd == "classify":
            print(json.dumps({"state": "INVALID", "violations": [{"invariant": x.invariant, "message": x.message}]},
                             indent=1, sort_keys=True))
            raise SystemExit(1)
        raise SystemExit(f"INVALID protocol: {x}")
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
