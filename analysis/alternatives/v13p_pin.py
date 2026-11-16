#!/usr/bin/env python3
"""DT13O-01 and DT13O-04 witnesses: strict policy and protocol parsing, before
(V0.13o) and after (V0.13p).

Genuine-provenance recordings (scripted model replies and health, a monotonic
clock and a linked SYSTEM_UTC clock; no model endpoint) of a wall-clock halt,
classified and analysed through the ordinary command line. Every mutation
recomputes every dependent hash, commitment and the seal.

* DT13O-01 (a): an acceptance file with a second R-HALT-003 (60 then 120) and
  a protocol re-pinned to those bytes with 120 s; archive 60.1 s beyond the
  timestamp bound. Byte-identical verifier copies next to that file: V0.13o
  accepts it (120 s applied); V0.13p refuses it (R-POLICY-001). Both
  recorders refuse the policy.
* DT13O-01 (b): an acceptance file with a duplicate top-level key and a
  protocol re-pinned to it. The V0.13o recorder accepts the binding; the
  V0.13p recorder refuses it; both verifiers refuse the archive.
* DT13O-04: protocol text with "toleranceSeconds": 120 then 60, bound by
  SHA-256 to the archive. V0.13o reads it last-key-wins and accepts; V0.13p
  refuses it (R-PROTO-001), and its recorder entry point refuses the bytes.
* DT13M-01 regression: the original ambient-120 witness, refused (R-PIN-001).
* Controls recorded through v13p.run_schedule (exact protocol bytes): a
  genuine halt and +59.9 s accepted, +60.1 s rejected (R-HALT-003).

    python3 -I -B analysis/alternatives/v13p_pin.py --binary ./dt --scratch DIR --out FILE
"""
import argparse
import copy
import hashlib
import json
from pathlib import Path
import shutil
import sys

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
import v13m_dt13l as D  # noqa: E402  (V0.13m-lite witness helpers)
import v13o  # noqa: E402
import v13o_pin as B  # noqa: E402  (V0.13o witness helpers, unchanged)
import v13p  # noqa: E402

SCHEMA = "dreamteam.v13p-parsing-witnesses/v1"
MODULES = ("v13m_verify.py", "v13o_verify.py", "v13p_verify.py", "bindings_v2.json")
RAW = (HERE / "acceptance_v1.json").read_bytes()


def ambient(scratch, name, data):
    """Byte-identical verifier copies next to an acceptance file with these bytes."""
    d = scratch / ("ambient-" + name)
    d.mkdir(parents=True, exist_ok=True)
    for m in MODULES:
        shutil.copyfile(HERE / m, d / m)
    (d / "acceptance_v1.json").write_bytes(data)
    return d


def dup_rule():
    doc = json.loads(RAW)
    r = copy.deepcopy(next(x for x in doc["relationships"] if x["id"] == "R-HALT-003"))
    r["toleranceSeconds"] = 120
    doc["relationships"].append(r)
    return (json.dumps(doc, indent=1, ensure_ascii=False) + "\n").encode("utf-8")


def dup_top():
    return RAW.replace(b'{\n "schema"', b'{\n "schema": "x",\n "schema"', 1)


def refuses(fn):
    try:
        fn()
        return False
    except Exception:  # noqa: BLE001  (any refusal counts; the message is recorded by the tests)
        return True


def v13o_binds(p, f):
    """Whether the V0.13o recorder, next to acceptance file f, accepts p's binding (v13o.check_binding)."""
    try:
        return json.dumps(v13o.acceptance_binding(f), sort_keys=True) == json.dumps(p["acceptance"], sort_keys=True)
    except Exception:  # noqa: BLE001
        return False


def run(binary, scratch):
    scratch = Path(scratch)
    impl = D.Impl("v13m", scratch)
    K = impl.K
    base = B.setup(impl, scratch, "v13p", v13p.draft_protocol)
    base["binary"] = binary
    results = []
    identical = []

    def case(finding, name, expected, verifier, c, out, amb=None):
        r = B.cli(verifier, c, out, scratch / "analysis" / (verifier[:-3] + "-" + name), amb)
        r.update(finding=finding, case=name, verifier=verifier[:-10], expected=expected,
                 ambientAcceptance=None if amb is None else amb.name[len("ambient-"):])
        results.append(r)
        return r

    def strict_recorder(path):
        return lambda p, ph, *a, **k: v13p.run_schedule(path.read_bytes(), *a, **k)

    # Controls: recorded through the V0.13p entry point (exact protocol bytes).
    halt = B.record_halt(impl, binary, base, scratch / "v13p-halt", strict_recorder(base["path"]))
    case("control", "genuine-linked-halt", "accepted", "v13p_verify.py", base, halt)
    for beyond, expect in ((59.9, "accepted"), (60.1, "rejected")):
        out = scratch / ("v13p-excess-%s" % beyond)
        D.mutate(impl, halt, out, lambda a, b=beyond: B.excess(a, b))
        case("tolerance", "elapsed-beyond-timestamps-by-%s-s" % beyond, expect, "v13p_verify.py", base, out)

    # DT13M-01 regression: the reviewer's ambient-120 construction.
    amb = ambient(scratch, "tolerance-120", B.ambient_dir(scratch, 120).joinpath("acceptance_v1.json").read_bytes())
    for v in ("v13o_verify.py", "v13p_verify.py"):
        case("DT13M-01", "ambient-acceptance-120-s-excess-60.1-s", "rejected", v, base, scratch / "v13p-excess-60.1", amb)

    # DT13O-01 (a) and (b): acceptance files other than the committed bytes, protocol re-pinned.
    for name, data, tol, beyond, gap in (("duplicate-rule-60-then-120", dup_rule(), 120, 60.1, "v13o"),
                                         ("duplicate-top-level-key", dup_top(), 60, None, None)):
        amb = ambient(scratch, name, data)
        p = copy.deepcopy(base["p"])
        p["acceptance"]["sha256"] = hashlib.sha256(data).hexdigest()
        p["acceptance"]["clockReconciliation"]["toleranceSeconds"] = tol
        c = B.finish(impl, scratch, "repinned-" + name, p, base["rows"], base["cd"])
        c["binary"] = binary
        out = B.record_halt(impl, binary, c, scratch / ("repinned-" + name), K.run_schedule)
        if beyond is not None:
            moved = scratch / ("repinned-%s-excess-%s" % (name, beyond))
            D.mutate(impl, out, moved, lambda a, b=beyond: B.excess(a, b))
            out = moved
        f = amb / "acceptance_v1.json"
        recorders = {"v13o": v13o_binds(p, f), "v13p": not refuses(lambda: v13p.check_binding(p, f))}
        for v in ("v13o_verify.py", "v13p_verify.py"):
            expect = "accepted (gap)" if gap == v[:4] else "rejected"
            r = case("DT13O-01", "repinned-" + name, expect, v, c, out, amb)
            r["recorderAcceptsBinding"] = recorders[v[:4]]
        identical.append(all((HERE / m).read_bytes() == (amb / m).read_bytes() for m in MODULES))

    # DT13O-04: duplicated protocol text, bound to the archive by SHA-256.
    text = base["path"].read_bytes().replace(b'"toleranceSeconds": 60', b'"toleranceSeconds": 120,\n  "toleranceSeconds": 60', 1)
    c = B.finish(impl, scratch, "dup-protocol-text", json.loads(text), base["rows"], base["cd"])
    c["path"].write_bytes(text)
    c["ph"] = hashlib.sha256(text).hexdigest()
    c["binary"] = binary
    out = B.record_halt(impl, binary, c, scratch / "dup-protocol-text", K.run_schedule)
    strict_entry_refuses = refuses(lambda: v13p.run_schedule(text, c["sched"], None, None, {}, str(scratch / "never"), ""))
    for v in ("v13o_verify.py", "v13p_verify.py"):
        r = case("DT13O-04", "protocol-text-tolerance-120-then-60", "accepted (gap)" if v[:4] == "v13o" else "rejected",
                 v, c, out)
        r["recorderAcceptsProtocol"] = not refuses(lambda: v13o.check_binding(json.loads(text))) \
            if v[:4] == "v13o" else not strict_entry_refuses
    return results, all(identical)


def meets(r):
    if not B.meets(r):
        return False
    if r["finding"] == "DT13O-01":
        # V0.13o: (a) recorder refuses, verifier accepts; (b) recorder accepts, verifier refuses. V0.13p: both refuse.
        if r["verifier"] == "v13p":
            return r["recorderAcceptsBinding"] is False
        return r["recorderAcceptsBinding"] is (not r["expected"].startswith("accepted"))
    if r["finding"] == "DT13O-04":
        return r["recorderAcceptsProtocol"] is (r["verifier"] == "v13o")
    return True


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    ap.add_argument("--binary", required=True)
    ap.add_argument("--scratch", required=True)
    ap.add_argument("--out", required=True)
    a = ap.parse_args()
    results, identical = run(str(Path(a.binary).resolve()), a.scratch)
    for r in results:
        r["meetsExpectation"] = meets(r)
    doc = {"schema": SCHEMA, "findings": ["DT13O-01", "DT13O-04", "DT13M-01"], "verifierCopiesByteIdentical": identical,
           "acceptanceSha256": v13p.acceptance_binding()["sha256"],
           "protocolSha256": hashlib.sha256((HERE.parents[1] / "docs/benchmark/v13o-dev/protocol-draft.json")
                                            .read_bytes()).hexdigest(),
           "cases": results, "allMeetExpectation": all(r["meetsExpectation"] for r in results)}
    D.write(a.out, doc)
    for r in results:
        print(f"{'ok ' if r['meetsExpectation'] else 'BAD'} {r['verifier']:5} {r['case']:45} "
              f"{r['classification']!s:8} analysis {'accepted' if r['analysis']['accepted'] else 'refused'} "
              f"{','.join(r['violations'])}")
    if not doc["allMeetExpectation"]:
        raise SystemExit("a V0.13p witness does not meet its expectation")


if __name__ == "__main__":
    main()
