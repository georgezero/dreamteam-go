#!/usr/bin/env python3
"""DT13M-01 witnesses: the acceptance standard pinned in the protocol, before
(V0.13m-lite) and after (V0.13o).

Genuine-provenance recordings (scripted model replies and health, a
monotonic clock and a linked SYSTEM_UTC clock; no model endpoint) of a
wall-clock halt, analysed through the ordinary command line:

* the reviewer's diagnostic: a byte-identical verifier copy whose scratch
  acceptance data says 120 s instead of 60 s, applied to an archive whose
  elapsedSeconds exceeds the timestamp bound by 60.1 s. V0.13m-lite accepts
  it; V0.13o refuses it (R-PIN-001) because the protocol pins the original
  acceptance file;
* the tolerance boundary under the unmodified standard (+59.9 s accepted,
  +60.1 s rejected), before and after;
* protocols that omit or alter the binding (no field, another file hash,
  another tolerance, a weakened operational-assumption statement), recorded
  through the V0.13m-lite recorder (the V0.13o recorder refuses each of
  them) and refused by the V0.13o verifier.

Every mutation recomputes every dependent hash, commitment and the seal.

    python3 -I -B analysis/alternatives/v13o_pin.py --binary ./dt --scratch DIR --out FILE
"""
import argparse
import copy
import datetime
import json
from pathlib import Path
import shutil
import subprocess
import sys

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
import v13m_dt13l as D  # noqa: E402  (V0.13m-lite witness helpers)
import v13o  # noqa: E402

SCHEMA = "dreamteam.v13o-pin-witnesses/v1"
MODULES = ("v13m_verify.py", "v13o_verify.py", "bindings_v2.json")


def setup(impl, scratch, name, draft):
    """D.cli_setup for a chosen protocol drafter (one context, one model, two draws)."""
    K, S = impl.K, impl.S
    rows, _ = K.H.load_contexts(S.CONTEXTS)
    rows = [r for r in rows if r["id"] == "S1-a"]
    cd = scratch / "cli-contexts"
    cd.mkdir(parents=True, exist_ok=True)
    raw = "".join(json.dumps(r, sort_keys=True) + "\n" for r in rows)
    (cd / "contexts.jsonl").write_text(raw)
    p, _ = draft(rows, S.D.sha256(raw))
    p["modelOrder"] = [K.QWEN]
    p["drawsPerUnit"] = {K.QWEN: {"aa": 1, "unit": 1}}
    return finish(impl, scratch, name, p, rows, cd)


def finish(impl, scratch, name, p, rows, cd):
    K, S = impl.K, impl.S
    path = scratch / (name + "-protocol.json")
    D.write(path, p)
    sched = K.H.schedule(p, rows)
    ctx = K.Contexts(p, rows, None, {K.H.SB.SETTINGS_ENV: K.H.SB.REPLAY_URL})
    replies = [S.F.ABANDON if e["arm"] == "control" else S.F.SEL("no_action") for e in sched]
    return dict(name=name, path=path, cd=cd, rows=rows, p=p, ph=S.D.sha256(path.read_bytes()), sched=sched, ctx=ctx,
                replies=replies)


def record_halt(impl, binary, cli, out, recorder):
    """A genuine wall-clock halt well past the request-start threshold, UTC linked to the monotonic clock."""
    K, S = impl.K, impl.S
    stop = cli["p"]["stop"]["wallClockSeconds"]
    value = stop - cli["p"]["request"]["timeoutSeconds"] + 3600.7
    m = D.Mono()
    utc = D.LinkedClock(m)

    def hop(b, phase, c):
        if (b, phase) == ("draw_guard", "during") and m.value == 0.0:
            m.value = value
    recorder(cli["p"], cli["ph"], cli["sched"], cli["ctx"], cli["ctx"].opener(binary),
             {K.QWEN: S.Model(cli["replies"])}, str(out), "model-free DT13M-01 witness",
             state_fn=S.Health(utc), utc=utc, now=m, hook=hop)
    return out


def _t(value):
    return datetime.datetime.fromisoformat(value[:-1] + "+00:00")


def excess(a, beyond):
    """Shift every halt's elapsedSeconds so it exceeds the timestamp upper bound
    (stop transition - begin_preflight) by `beyond` seconds before tolerance."""
    j = a.doc["journal"]
    ev = [e["event"] for e in j]
    upper = (_t(j[ev.index("stop")]["timestamp"]) - _t(j[ev.index("begin_preflight")]["timestamp"])).total_seconds()
    e = a.doc["stop"]["evidence"]["elapsedSeconds"]
    D.shift_elapsed(a, round(upper - e + beyond, 6))


def cli(verifier, cli, out, analysis, ambient=None):
    """Classify and analyse through the ordinary command line. With `ambient`,
    the verifier modules are byte-identical copies in that directory, next to
    its acceptance file (the reviewer's construction)."""
    if ambient is None:
        base = [sys.executable, "-I", "-B", str(HERE / verifier)]
    else:
        boot = ("import sys,runpy;sys.path.insert(0,%r);sys.argv=[%r]+sys.argv[1:];runpy.run_path(%r,run_name='__main__')"
                % (str(HERE), str(ambient / verifier), str(ambient / verifier)))
        base = [sys.executable, "-I", "-B", "-c", boot]
    args = ["--contexts", str(cli["cd"]), "--protocol", str(cli["path"]), "--dir", str(out)]
    c = subprocess.run(base + ["classify"] + args, capture_output=True, text=True, timeout=900)
    try:
        v = json.loads(c.stdout)
        cls = {"state": v["state"], "violations": sorted({x["invariant"] for x in v["violations"]})}
    except ValueError:
        cls = {"state": None, "error": D.scrub((c.stderr.strip().splitlines() or [""])[-1])[:300]}
    r = subprocess.run(base + ["analyze", "--binary", cli["binary"], "--out", str(analysis)] + args,
                       capture_output=True, text=True, timeout=900)
    return {"classification": cls["state"], "violations": cls.get("violations", []), "classifyExit": c.returncode,
            "analysis": {"accepted": r.returncode == 0, "exit": r.returncode,
                         "stdout": D.scrub(r.stdout.strip())[:200],
                         "refusal": D.scrub((r.stderr.strip().splitlines() or [""])[-1])[:300]}}


def ambient_dir(scratch, tolerance):
    d = scratch / ("ambient-tolerance-%d" % tolerance)
    d.mkdir(parents=True, exist_ok=True)
    for m in MODULES:
        shutil.copyfile(HERE / m, d / m)
    acc = json.loads((HERE / "acceptance_v1.json").read_text())
    next(x for x in acc["relationships"] if x["id"] == "R-HALT-003")["toleranceSeconds"] = tolerance
    (d / "acceptance_v1.json").write_text(json.dumps(acc, indent=1) + "\n")
    return d


def run(binary, scratch):
    scratch = Path(scratch)
    impl = D.Impl("v13m", scratch)
    K = impl.K
    before = setup(impl, scratch, "v13m", K.draft_protocol)
    after = setup(impl, scratch, "v13o", v13o.draft_protocol)
    results = []
    for c in (before, after):
        c["binary"] = binary
    amb = ambient_dir(scratch, 120)
    identical = all((HERE / m).read_bytes() == (amb / m).read_bytes() for m in MODULES)

    def case(finding, name, expected, impl_name, c, out, ambient=None):
        verifier = "v13m_verify.py" if impl_name == "v13m" else "v13o_verify.py"
        r = cli(verifier, c, out, scratch / "analysis" / (impl_name + "-" + name), ambient)
        r.update(finding=finding, case=name, implementation=impl_name, expected=expected,
                 ambientTolerance=None if ambient is None else 120)
        results.append(r)

    for impl_name, c in (("v13m", before), ("v13o", after)):
        recorder = K.run_schedule if impl_name == "v13m" else v13o.run_schedule
        base = record_halt(impl, binary, c, scratch / (impl_name + "-halt"), recorder)
        case("control", "genuine-linked-halt", "accepted", impl_name, c, base)
        for beyond, expect in ((59.9, "accepted"), (60.1, "rejected")):
            out = scratch / ("%s-excess-%s" % (impl_name, beyond))
            D.mutate(impl, base, out, lambda a, b=beyond: excess(a, b))
            case("tolerance", "elapsed-beyond-timestamps-by-%s-s" % beyond, expect, impl_name, c, out)
        out = scratch / ("%s-excess-60.1" % impl_name)
        # The reviewer's diagnostic: unchanged verifier code, ambient acceptance data 120 s.
        case("DT13M-01", "ambient-acceptance-120-s-excess-60.1-s",
             "accepted (gap)" if impl_name == "v13m" else "rejected", impl_name, c, out, amb)

    # Protocols that omit or alter the binding: recorded through the V0.13m-lite recorder.
    tampered = {
        "binding-absent": lambda p: p.pop("acceptance"),
        "acceptance-sha-altered": lambda p: p["acceptance"].update(sha256="0" * 64),
        "tolerance-120": lambda p: p["acceptance"]["clockReconciliation"].update(toleranceSeconds=120),
        "operational-assumption-weakened": lambda p: p["acceptance"]["clockReconciliation"].update(
            basis="measured clock-accuracy guarantee"),
        "scope-widened-to-guard": lambda p: p["acceptance"]["clockReconciliation"].update(
            scope="also relaxes the four-hour request-start guard by toleranceSeconds"),
    }
    for name, fn in tampered.items():
        p = copy.deepcopy(after["p"])
        fn(p)
        c = finish(impl, scratch, "tampered-" + name, p, after["rows"], after["cd"])
        c["binary"] = binary
        try:
            v13o.check_binding(p)
            refused = None
        except K.ExperimentError as x:
            refused = str(x)
        out = record_halt(impl, binary, c, scratch / ("tampered-" + name), K.run_schedule)
        case("DT13M-01", "protocol-" + name, "rejected", "v13o", c, out)
        results[-1]["v13oRecorderRefusesProtocol"] = refused is not None
    return results, identical


def meets(r):
    accepted = r["analysis"]["accepted"]
    if r["expected"].startswith("accepted"):
        return accepted and r["classification"] != "INVALID"
    return not accepted and r["classification"] == "INVALID"


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    ap.add_argument("--binary", required=True)
    ap.add_argument("--scratch", required=True)
    ap.add_argument("--out", required=True)
    a = ap.parse_args()
    binary = str(Path(a.binary).resolve())
    results, identical = run(binary, a.scratch)
    for r in results:
        r["meetsExpectation"] = meets(r) and r.get("v13oRecorderRefusesProtocol", True)
    doc = {"schema": SCHEMA, "finding": "DT13M-01", "verifierCopiesByteIdentical": identical,
           "acceptanceSha256": v13o.acceptance_binding()["sha256"],
           "cases": results, "allMeetExpectation": all(r["meetsExpectation"] for r in results)}
    D.write(a.out, doc)
    for r in results:
        print(f"{'ok ' if r['meetsExpectation'] else 'BAD'} {r['implementation']:5} {r['case']:45} "
              f"{r['classification']!s:8} analysis {'accepted' if r['analysis']['accepted'] else 'refused'}")
    if not doc["allMeetExpectation"]:
        raise SystemExit("a DT13M-01 witness does not meet its expectation")


if __name__ == "__main__":
    main()
