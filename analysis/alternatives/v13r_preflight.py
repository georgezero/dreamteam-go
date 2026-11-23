#!/usr/bin/env python3
"""V0.13r runner preflight: the lock checks a future live runner must pass
before it records anything (LC-02, DT13O-05). It is not a runner and calls no
model.

``preflight`` refuses (L.LockError) unless:

* the lock parses strictly, is complete, LOCKED, and its purpose matches;
* its Git identity checks out (lock commit, manifest bytes, approval record,
  human-review record, engine source tree);
* the repository modules loaded in this process are exactly the lock's
  recorder modules, byte for byte;
* the protocol is the locked protocol's exact bytes, parses strictly
  (R-PROTO-001), and binds the acceptance standard (R-POLICY-001, V0.13o);
* the acceptance file, schedule, engine binary, model configuration,
  presentation, outcome definition and design values are the locked ones.

TEST ONLY locks may be preflighted in a process that also loads unpinned
test-harness modules; every pinned module must still match. A RUN lock never
allows that.

Only after preflight may anything call ``v13p.run_schedule``. It is
reached only through ``record_after_preflight``, which records the lock binding
at the start of the run's authorization (the locked analysis requires it).
In V0.13r, ``record_after_preflight`` refuses a RUN lock: no live runner
exists.

    python3 -I -B analysis/alternatives/v13r_preflight.py check --lock FILE --lock-commit SHA \\
        --protocol FILE --schedule FILE --contexts DIR --binary ./dt [--test-only]
"""
import argparse
import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import v13r_lock as L  # noqa: E402  (the lock contract)
import v13p as P  # noqa: E402  (the V0.13p recorder entry point, unchanged)

K, H = P.K, P.H
ACCEPTANCE = os.path.join(HERE, "acceptance_v1.json")


class Preflight:
    """A passed preflight: the exact protocol bytes, the schedule and the lock binding."""

    def __init__(self, protocol_raw, proto, sched, rows, identity, bind, unpinned):
        self.protocol_raw, self.proto, self.sched, self.rows = protocol_raw, proto, sched, rows
        self.identity, self.binding, self.unpinned = identity, bind, unpinned


def preflight(lock, lock_commit, protocol, schedule, contexts, binary, repo=L.ROOT, test_only=False):
    raw = L.read(lock)
    doc = L.load(raw)
    L.need((doc["purpose"] == L.TEST) == test_only, "LK-PURPOSE",
           "a TEST ONLY lock requires test_only, and test_only accepts only a TEST ONLY lock")
    ident = L.verify_commit(doc, raw, lock_commit, repo)
    expected = doc["recorder"]["modules"]
    loaded = L.loaded_modules(HERE)
    bad = sorted(k for k in expected if loaded.get(k) != expected[k])
    L.need(not bad, "LK-MODULE", f"recorder modules missing or differing from the lock: {bad}")
    unpinned = sorted(set(loaded) - set(expected))
    L.need(not unpinned or doc["purpose"] == L.TEST, "LK-MODULE", f"unpinned modules loaded: {unpinned}")
    praw = L.read(protocol)
    try:
        proto = P.load_protocol(praw)
    except K.ExperimentError as x:
        raise L.LockError("LK-PROTOCOL", str(x)) from None
    rows, csha = H.load_contexts(contexts)
    L.need(csha == proto.get("contextsFileSha256"), "LK-PROTOCOL", "the contexts file is not the protocol's")
    sched = H.schedule(proto, rows)
    psha = L.sha(praw)
    L.need(psha == doc["protocol"]["sha256"], "LK-PROTOCOL", "the protocol bytes are not the locked protocol")
    L.check_content(doc, praw, proto, L.read(schedule), sched, binary, ACCEPTANCE)
    try:
        P.check_binding(proto)
    except K.ExperimentError as x:
        raise L.LockError("LK-ACCEPTANCE", str(x)) from None
    return Preflight(praw, proto, sched, rows, ident, L.binding(ident, psha), unpinned)


def record_after_preflight(pf, ctxs, opener, transports, out, owner_authorization, **kw):
    """v13p.run_schedule for a preflighted lock, with the lock binding recorded.
    Refuses a RUN lock in V0.13r (no live runner is authorized or built)."""
    L.need(isinstance(pf, Preflight), "LK-RUNNER", "no preflight")
    L.need(pf.identity["purpose"] == L.TEST, "LK-RUNNER",
           "recording under a RUN lock is not implemented: the live runner is not built (V0.13r)")
    L.need(isinstance(owner_authorization, str) and owner_authorization.strip(), "LK-RUNNER",
           "no run authorization")
    return P.run_schedule(pf.protocol_raw, pf.sched, ctxs, opener, transports, out,
                          pf.binding + " " + owner_authorization.strip(), **kw)


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    sub = ap.add_subparsers(dest="cmd", required=True)
    p = sub.add_parser("check")
    for k in ("--lock", "--lock-commit", "--protocol", "--schedule", "--contexts", "--binary"):
        p.add_argument(k, required=True)
    p.add_argument("--repo", default=L.ROOT)
    p.add_argument("--test-only", action="store_true")
    a = ap.parse_args(argv)
    try:
        try:
            pf = preflight(a.lock, a.lock_commit, a.protocol, a.schedule, a.contexts, a.binary, a.repo, a.test_only)
        except OSError as x:
            raise L.LockError("LK-IO", f"cannot read an input: {x.strerror}: {os.path.basename(str(x.filename))}") \
                from None
    except L.LockError as x:
        print(json.dumps({"preflight": "REFUSED", "code": x.code, "message": x.message}, sort_keys=True))
        raise SystemExit(1)
    print(json.dumps({"preflight": "PASSED", "lock": pf.identity, "draws": len(pf.sched),
                      "unpinnedModules": pf.unpinned}, sort_keys=True))


if __name__ == "__main__":
    main()
