#!/usr/bin/env python3
"""V0.13p recorder entry point: the V0.13o protocol successor (unchanged
protocol v13o-alt-1), with strict policy and protocol parsing
(DT13O-01, DT13O-04).

V0.13o parsed the acceptance file and the protocol file with Python's
permissive json.load, which keeps the last of duplicate keys and accepts
NaN/Infinity. This entry point parses both with its own implementation of
dreamteam.strict-json/v1, the same accepted syntax as the verifier:

* strict UTF-8; no duplicate object key at any nesting level; no NaN,
  Infinity or -Infinity literal; no number that overflows to infinity;
  integers stay exact; no trailing content;
* the acceptance file must be a well-formed policy (R-POLICY-001): an object
  whose schema is dreamteam.acceptance-standard/v1, with a non-empty
  ``relationships`` list of objects, each with a non-empty string ``id``;
  identifiers are unique; R-HALT-003 carries a positive integer
  ``toleranceSeconds``;
* the protocol file must be a strict JSON object (R-PROTO-001).

The recorder does not import the verifier: the parser below is an independent
implementation of the same syntax, and test_v13p.py checks the two agree.

The protocol, its binding, the schedule, the driver, the recorder, the seal
and every model-visible input are V0.13o's and V0.13m-lite's, unchanged. A
future live runner must enter through ``run_schedule`` here (or a successor):
it takes the protocol's exact bytes, never a parsed object.

    python3 -I -B analysis/alternatives/v13p.py protocol --contexts DIR --out FILE
    python3 -I -B analysis/alternatives/v13p.py schedule --protocol FILE --contexts DIR --out FILE
    python3 -I -B analysis/alternatives/v13p.py check --binary ./dt --protocol FILE --contexts DIR
"""
import argparse
import hashlib
import json
import math
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import v13o as O  # noqa: E402  (the V0.13o protocol successor, unchanged)

K = O.K
H, sha256, write_json = K.H, K.sha256, K.write_json
POLICY_RULE = "R-POLICY-001"
PROTOCOL_RULE = "R-PROTO-001"


class ParseError(K.ExperimentError):
    pass


def _bytes(path):
    with open(path, "rb") as f:
        return f.read()


def strict_json(data):
    """dreamteam.strict-json/v1 (recorder implementation)."""
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


def _policy(raw):
    """The acceptance file's relationships by id, or ParseError (R-POLICY-001)."""
    try:
        doc = strict_json(raw)
    except ValueError as x:
        raise ParseError(f"{POLICY_RULE}: the acceptance file is not strict JSON: {x}") from None

    def need(ok, message):
        if not ok:
            raise ParseError(f"{POLICY_RULE}: {message}")
    need(isinstance(doc, dict), "the acceptance file is not an object")
    need(doc.get("schema") == O.ACCEPTANCE_ID and type(doc.get("schema")) is str, "acceptance standard identity")
    rel = doc.get("relationships")
    need(isinstance(rel, list) and rel, "the acceptance file has no relationships list")
    need(all(isinstance(r, dict) and type(r.get("id")) is str and r["id"] for r in rel),
         "every relationship must be an object with a non-empty string id")
    ids = [r["id"] for r in rel]
    need(len(ids) == len(set(ids)), "duplicate relationship id: " + ", ".join(sorted({i for i in ids if ids.count(i) > 1})))
    rules = {r["id"]: r for r in rel}
    need(O.CLOCK_RULE in rules, "no R-HALT-003 relationship")
    tol = rules[O.CLOCK_RULE].get("toleranceSeconds")
    need(type(tol) is int and tol > 0, "R-HALT-003 toleranceSeconds must be a positive integer")
    return rules


def acceptance_binding(path=None):
    """O.acceptance_binding, from a strictly parsed, well-formed acceptance file."""
    raw = _bytes(path or os.path.join(HERE, "acceptance_v1.json"))
    rules = _policy(raw)
    return {"id": O.ACCEPTANCE_ID, "file": O.ACCEPTANCE_FILE, "sha256": hashlib.sha256(raw).hexdigest(),
            "clockReconciliation": dict(O.CLOCK_POLICY, toleranceSeconds=rules[O.CLOCK_RULE]["toleranceSeconds"])}


def load_protocol(raw):
    """A protocol from its exact bytes (R-PROTO-001): strict JSON object, or ParseError."""
    try:
        proto = strict_json(raw)
    except ValueError as x:
        raise ParseError(f"{PROTOCOL_RULE}: the protocol file is not strict JSON: {x}") from None
    if not isinstance(proto, dict):
        raise ParseError(f"{PROTOCOL_RULE}: the protocol file is not an object")
    return proto


def check_binding(proto, path=None):
    """Refuse a protocol that does not bind the strictly parsed acceptance standard."""
    if json.dumps(proto.get("acceptance"), sort_keys=True) != json.dumps(acceptance_binding(path), sort_keys=True):
        raise K.ExperimentError("the protocol does not bind the acceptance standard (DT13M-01): "
                                "acceptance identity, file hash or clock-reconciliation policy differs")


def draft_protocol(rows, contexts_sha):
    acceptance_binding()  # R-POLICY-001 before drafting
    return O.draft_protocol(rows, contexts_sha)


def run_schedule(protocol_raw, sched, ctxs, opener, transports, out, authorization, **kw):
    """The only recording entry point for a future live runner: the protocol's
    exact bytes are parsed strictly, hashed, and must bind the acceptance
    standard; then K.run_schedule records unchanged."""
    if not isinstance(protocol_raw, bytes):
        raise K.ExperimentError("run_schedule takes the protocol file's exact bytes")
    proto = load_protocol(protocol_raw)
    check_binding(proto)
    return K.run_schedule(proto, sha256(protocol_raw), sched, ctxs, opener, transports, out, authorization, **kw)


# --- commands (V0.13o's, with strict parsing) ----------------------------------------

def _read(path):
    raw = _bytes(path)
    return load_protocol(raw), raw


def cmd_protocol(a):
    rows, csha = H.load_contexts(a.contexts)
    proto, _ = draft_protocol(rows, csha)
    write_json(a.out, proto)
    print(f"protocol {proto['id']}: {proto['draws']} draws, at most {proto['maxModelCalls']} model calls; "
          f"acceptance {proto['acceptance']['sha256'][:12]} tolerance "
          f"{proto['acceptance']['clockReconciliation']['toleranceSeconds']} s")


def cmd_schedule(a):
    proto, raw = _read(a.protocol)
    rows, csha = H.load_contexts(a.contexts)
    if csha != proto.get("contextsFileSha256"):
        raise SystemExit("the contexts file is not the protocol's")
    check_binding(proto)
    again, sched = draft_protocol(rows, csha)
    if {k: v for k, v in again.items() if k != "status"} != {k: v for k, v in proto.items() if k not in ("status", "lock")}:
        raise SystemExit("the protocol is not the one the committed inputs make")
    doc = {"schema": H.SCHEDULE_SCHEMA, "protocol": proto["id"], "protocolSha256": sha256(raw),
           "orderRule": H.ORDER_RULE, "draws": len(sched), "schedule": sched}
    write_json(a.out, doc)
    print(f"schedule: {len(sched)} draws")


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    sub = ap.add_subparsers(dest="cmd", required=True)
    p = sub.add_parser("protocol")
    p.add_argument("--contexts", required=True)
    p.add_argument("--out", required=True)
    p = sub.add_parser("schedule")
    p.add_argument("--protocol", required=True)
    p.add_argument("--contexts", required=True)
    p.add_argument("--out", required=True)
    p = sub.add_parser("check")
    p.add_argument("--binary", required=True)
    p.add_argument("--protocol", required=True)
    p.add_argument("--contexts", required=True)
    p.add_argument("--replays")
    p = sub.add_parser("run")
    p.add_argument("--protocol", required=True)
    p.add_argument("--authorization", default="")
    a = ap.parse_args(argv)
    if a.cmd == "protocol":
        cmd_protocol(a)
    elif a.cmd == "schedule":
        cmd_schedule(a)
    elif a.cmd == "check":
        check_binding(_read(a.protocol)[0])
        H.cmd_check(a)  # unchanged per-context engine check; no model is called
    elif a.cmd == "run":
        proto, _ = _read(a.protocol)
        check_binding(proto)
        if proto.get("status") != H.LOCKED:
            raise SystemExit(f"the protocol's status is {proto.get('status')!r}: only a locked protocol, with the "
                             "owner's authorization, can be run")
        if not a.authorization.strip():
            raise SystemExit("no --authorization: the lock does not authorize a run; the owner does")
        raise SystemExit("run: the live run is not implemented before the owner approves the protocol")


if __name__ == "__main__":
    main()
