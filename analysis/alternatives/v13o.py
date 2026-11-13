#!/usr/bin/env python3
"""V0.13o protocol successor: the V0.13m-lite recorder, unchanged, with the
acceptance standard pinned in the protocol (DT13M-01).

The protocol (dreamteam.alternatives-experiment/v10, id v13o-alt-1) is the
V0.13m-lite protocol plus one field, ``acceptance``: the identity, file and
SHA-256 of dreamteam.acceptance-standard/v1 (analysis/alternatives/
acceptance_v1.json, unchanged) and the clock-reconciliation policy it
contains, stated as an operational assumption:

* the 60-second tolerance reconciles a halt's elapsedSeconds with the
  recorded wall-clock timestamps only; it does not change the four-hour
  request-start guard;
* a discrepancy above the tolerance makes the archive INVALID, with no
  retry, resumption or replacement draw;
* it is an operational assumption about clock agreement, not a measured
  clock-accuracy guarantee.

A run archive binds the protocol by SHA-256, so it binds the acceptance
standard transitively. The recorder refuses a protocol whose binding does not
match the acceptance file; the verifier (v13o_verify.py) refuses an archive
whose protocol does not bind the acceptance standard it applies.

Recording, the driver, the lifecycle contract, the seal, the schedule order
and every model-visible input are V0.13m-lite's, unchanged.

    python3 -I -B analysis/alternatives/v13o.py protocol --contexts DIR --out FILE
    python3 -I -B analysis/alternatives/v13o.py schedule --protocol FILE --contexts DIR --out FILE
    python3 -I -B analysis/alternatives/v13o.py check --binary ./dt --protocol FILE --contexts DIR
"""
import argparse
import hashlib
import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import v13m as K  # noqa: E402  (the V0.13m-lite recorder, unchanged)

H, sha256, write_json = K.H, K.sha256, K.write_json

PROTOCOL_SCHEMA = "dreamteam.alternatives-experiment/v10"
PROTOCOL_ID = "v13o-alt-1"
ACCEPTANCE_ID = "dreamteam.acceptance-standard/v1"
ACCEPTANCE_FILE = "analysis/alternatives/acceptance_v1.json"
CLOCK_RULE = "R-HALT-003"
CLOCK_POLICY = {
    "rule": CLOCK_RULE,
    "scope": ("reconciles each halt's evidence elapsedSeconds with the recorded wall-clock (SYSTEM_UTC) journal and "
              "call-journal timestamps only; it does not change the four-hour request-start guard "
              "(stop.wallClockSeconds, request.timeoutSeconds), which is applied without tolerance"),
    "onExcess": "INVALID: a discrepancy above toleranceSeconds rejects the archive, with no retry, resumption or "
                "replacement draw",
    "basis": "operational assumption that the monotonic and UTC clocks of the recording host agree within "
             "toleranceSeconds over the run; it is not a measured clock-accuracy guarantee",
}


def acceptance_binding(path=None):
    """The protocol's acceptance field, computed from the acceptance file's exact bytes."""
    raw = open(path or os.path.join(HERE, "acceptance_v1.json"), "rb").read()
    doc = json.loads(raw)
    if doc.get("schema") != ACCEPTANCE_ID:
        raise K.ExperimentError("acceptance standard identity")
    rule = [r for r in doc["relationships"] if r.get("id") == CLOCK_RULE]
    if len(rule) != 1 or type(rule[0].get("toleranceSeconds")) is not int:
        raise K.ExperimentError("the acceptance standard has no single integer R-HALT-003 tolerance")
    return {"id": ACCEPTANCE_ID, "file": ACCEPTANCE_FILE, "sha256": hashlib.sha256(raw).hexdigest(),
            "clockReconciliation": dict(CLOCK_POLICY, toleranceSeconds=rule[0]["toleranceSeconds"])}


def draft_protocol(rows, contexts_sha):
    proto, sched = K.draft_protocol(rows, contexts_sha)
    proto.update(schema=PROTOCOL_SCHEMA, id=PROTOCOL_ID, acceptance=acceptance_binding())
    return proto, sched


def check_binding(proto):
    """Refuse a protocol that does not bind the acceptance standard on disk."""
    # Canonical JSON text: 60.0 or true is not 60 (strict typed equality).
    if json.dumps(proto.get("acceptance"), sort_keys=True) != json.dumps(acceptance_binding(), sort_keys=True):
        raise K.ExperimentError("the protocol does not bind the acceptance standard (DT13M-01): "
                                "acceptance identity, file hash or clock-reconciliation policy differs")


def run_schedule(proto, proto_sha, sched, ctxs, opener, transports, out, authorization, **kw):
    """K.run_schedule, only for a protocol that binds the acceptance standard."""
    check_binding(proto)
    return K.run_schedule(proto, proto_sha, sched, ctxs, opener, transports, out, authorization, **kw)


# --- commands (V0.13m-lite's, for this protocol) -------------------------------------

def cmd_protocol(a):
    rows, csha = H.load_contexts(a.contexts)
    proto, sched = draft_protocol(rows, csha)
    write_json(a.out, proto)
    print(f"protocol {proto['id']}: {proto['draws']} draws, at most {proto['maxModelCalls']} model calls; "
          f"acceptance {proto['acceptance']['sha256'][:12]} tolerance "
          f"{proto['acceptance']['clockReconciliation']['toleranceSeconds']} s")


def cmd_schedule(a):
    proto, raw = json.load(open(a.protocol, encoding="utf-8")), open(a.protocol, "rb").read()
    rows, csha = H.load_contexts(a.contexts)
    if csha != proto["contextsFileSha256"]:
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
        check_binding(json.load(open(a.protocol, encoding="utf-8")))
        H.cmd_check(a)  # unchanged per-context engine check; no model is called
    elif a.cmd == "run":
        proto = json.load(open(a.protocol, encoding="utf-8"))
        check_binding(proto)
        if proto.get("status") != H.LOCKED:
            raise SystemExit(f"the protocol's status is {proto.get('status')!r}: only a locked protocol, with the "
                             "owner's authorization, can be run")
        if not a.authorization.strip():
            raise SystemExit("no --authorization: the lock does not authorize a run; the owner does")
        raise SystemExit("run: the live run is not implemented before the owner approves the protocol")


if __name__ == "__main__":
    main()
