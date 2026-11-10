#!/usr/bin/env python3
"""Original DT13J counterexamples, re-expressed as engineering regressions.
Every archive forgery recomputes commit hashes, failure mirrors and the seal.
"""
import argparse
import copy
import json
import os
from pathlib import Path
import shutil
import sys
import urllib.error

sys.path.insert(0, str(Path(__file__).resolve().parent))
import v13l as K
import v13l_mutations as M
import v13l_scenarios as S
import v13l_witnesses as W


def bases(binary, scratch):
    f = S.Fixture(binary, arms=("control", "treatment"))
    out = {}
    for name, kw in (("engine_close", {"opener": W.FailPool(f.ctx.opener(binary), "engine_close")}),
                     ("after_schedule", {"hook": M.At("safe_point", "before_cleanup")}),
                     ("unrecorded", {"draw_runner": M.unrecorded_runner})):
        p = str(Path(scratch) / ("base-" + name))
        c = S.Clock()
        f.record(p, clock=c, health=S.Health(c), **kw)
        f.analyze(p)
        out[name] = p
    return f, out


def erase_calls(a):
    r = a.files["aborted-draw.json"]
    r.update(calls=[], modelCallsCompleted=0)
    a.calls = []
    a.doc["counters"]["modelCalls"] = 0
    a.doc["failureEvent"]["modelCalls"] = 0


def abandoned_proposal(a):
    r = a.files["aborted-draw.json"]
    batches = copy.deepcopy(r["record"]["proposalSubmittedAction"]["batches"])
    rep = json.dumps(batches)
    r.update(batches=batches, submittedRepresentation=rep, submittedRepresentationSha256=K.sha256(rep))


def no_record(a):
    import v13l_verify as V
    r = a.files["aborted-draw.json"]
    r.update(record=None, batches=None, submittedRepresentation=None, submittedRepresentationSha256=None)
    r["process"] = V.Classifier(None, {}, None, [])._derived_process(r, {})


def too_many(a):
    original = copy.deepcopy(a.calls[:2])
    a.calls = []
    for n in range(1, 5):
        for rec in original:
            a.calls.append(dict(rec, k=n, n=n, timestamp=original[0]["timestamp"]))
    a.doc["counters"]["modelCalls"] = 4
    a.doc["failureEvent"]["modelCalls"] = 4


def duplicate_interrupt(a):
    e = a.doc["interrupts"]["events"][0]
    a.doc["interrupts"]["events"].append(dict(e, n=2))


def reverse_interrupts(a):
    e = a.doc["interrupts"]["events"][0]
    e["safePoint"] = "before_finalization"
    a.doc["interrupts"]["events"].append(dict(e, n=2, safePoint="before_cleanup"))


def early_after(a):
    e = a.doc["interrupts"]["events"][0]
    e.update(safePoint="after_cleanup", timestamp=M.first(a, "schedule_done")["timestamp"])


CASES = [
    ("DT13J-01/final_digest", "B-ENGINE-002", "engine_close", lambda a: a.files["aborted-draw.json"]["digestsAfter"].update(state="0" * 64)),
    ("DT13J-01/abandoned_proposal_submitted", "B-BATCH-001", "engine_close", abandoned_proposal),
    ("DT13J-01/call_erasure_retained_exchange", "B-ABORT-002", "engine_close", erase_calls),
    ("DT13J-01/record_and_halt_missing", "B-ABORT-001", "engine_close", no_record),
    ("DT13J-01/initial_engine_digest", "B-ENGINE-001", "engine_close", lambda a: a.files["aborted-draw.json"]["engine"][0]["answer"]["result"]["digests"].update(state="0" * 64)),
    ("DT13J-02/duplicate_before_cleanup", "B-INTERRUPT-001", "after_schedule", duplicate_interrupt),
    ("DT13J-02/reversed_safe_points", "B-INTERRUPT-001", "after_schedule", reverse_interrupts),
    ("DT13J-02/after_cleanup_before_completion", "B-CLEANUP-001", "after_schedule", early_after),
    ("DT13J-03/null_calls_over_context_bound", "B-CALL-004", "unrecorded", too_many),
    ("DT13J-03/body_without_status_or_error", "B-CALL-002", "unrecorded", lambda a: a.calls[1].update(status=None, error=None)),
    ("inventory-health/duplicate_status", "B-HEALTH-003", "after_schedule", lambda a: M.set_initial_health(a, b'{"status":"unhealthy","status":"healthy"}', "healthy", "HEALTHY")),
    ("inventory-health/nonfinite_json", "B-HEALTH-003", "after_schedule", lambda a: M.set_initial_health(a, b'{"status":"healthy","load":NaN}', "healthy", "HEALTHY")),
    ("DT13J-03/null_calls_false_hex_request_hash", "B-ABORT-002", "unrecorded", lambda a: a.calls[1].update(requestSha256="f" * 64)),
    ("DT13J-03/nonhex_request_hash", "B-CALL-002", "unrecorded", lambda a: a.calls[1].update(requestSha256="z" * 64)),
]


def run(binary, scratch):
    Path(scratch).mkdir(parents=True, exist_ok=True)
    f, sources = bases(binary, scratch)
    cases = []
    for name, binding, base, mutate in CASES:
        out = str(Path(scratch) / name.replace("/", "-"))
        shutil.copytree(sources[base], out)
        arc = M.Arc(out)
        mutate(arc)
        arc.write()
        v = f.classify(out)
        try:
            f.analyze(out)
            accepted = True
        except K.ExperimentError:
            accepted = False
        cases.append({"name": name, "binding": binding, "classification": v["state"],
                      "fullAnalysisAccepted": accepted, "agree": not accepted})
    for wrapped in (False, True):
        impostor = type("OSError", (ValueError,), {"__module__": "builtins"})("different class object")
        error = urllib.error.URLError(impostor) if wrapped else impostor
        def urlopen(*args, **kwargs):
            raise error
        def state():
            return {"health": K.probe_health("http://localhost:9", urlopen=urlopen)}
        out = str(Path(scratch) / ("impostor-wrapped" if wrapped else "impostor-unwrapped"))
        result = f.record(out, health=state)
        analysis = f.analyze(out)
        stop = result["manifest"]["stop"]
        cases.append({"name": "DT13J-05/" + ("wrapped" if wrapped else "unwrapped"), "binding": "B-HEALTH-004",
                      "classification": analysis["runStatus"], "stopCode": stop["code"],
                      "normalized": K.normalize_exception(error, K.LS.load()),
                      "agree": stop["code"] == "driver" and K.normalize_exception(error, K.LS.load()) is None})
    return {"schema": "dreamteam.v13l-regressions/v1", "cases": cases,
            "summary": {"cases": len(cases), "agree": sum(x["agree"] for x in cases)}}


def main():
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--binary", required=True)
    ap.add_argument("--scratch", required=True)
    ap.add_argument("--out", required=True)
    a = ap.parse_args()
    doc = run(os.path.abspath(a.binary), os.path.abspath(a.scratch))
    K.write_json(a.out, doc)
    print(json.dumps(doc["summary"]))
    if not all(x["agree"] for x in doc["cases"]):
        raise SystemExit(1)


if __name__ == "__main__":
    main()
