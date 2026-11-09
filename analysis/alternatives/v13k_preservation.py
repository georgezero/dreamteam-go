#!/usr/bin/env python3
"""B-HANDOFF-001: actual filesystem and completed-return fault regressions.
Scripted replies and a real local engine; no inference.
"""
import argparse
import copy
import errno
import json
import linecache
import os
from pathlib import Path
import sys

sys.path.insert(0, str(Path(__file__).resolve().parent))
import v13k as K
import v13k_scenarios as S
import lifecycle_v3_spec as LS

CASES = LS.rule("B-HANDOFF-001")["cases"]


def one(binary, scratch, kind):
    fixture = S.Fixture(binary)
    out = Path(scratch) / kind
    clock = S.Clock()
    armed, fired, after_sync = [False], [False], [False]

    class FS(K.OsFS):
        def fail(self, operation):
            fired[0] = True
            raise OSError(errno.EIO, "completed draw " + operation)

        def fsync(self, fd):
            if armed[0] and kind == "fsync_failure" and not fired[0]:
                self.fail("fsync failure")
            super().fsync(fd)
            if armed[0]:
                after_sync[0] = True

        def close(self, fd):
            super().close(fd)
            if armed[0] and kind == "close_failure_after_fsync" and not fired[0]:
                assert after_sync[0]
                self.fail("close failure after successful fsync")

        def replace(self, src, dst):
            if armed[0] and kind == "rename_failure" and not fired[0]:
                self.fail("rename failure")
            super().replace(src, dst)

    recorder = K.Recorder(fixture.proto, fixture.proto_sha, fixture.sched, fixture.ctx,
                          fixture.ctx.opener(binary), {S.D.QWEN: S.Model(fixture.replies())}, str(out),
                          "scripted completed draw preservation", state_fn=S.Health(clock), utc=clock,
                          now=S.Monotonic(), run_id=S.RUN_ID, provenance=S.SYNTHETIC, fs=FS())

    def hook(b, p, ctx):
        if b == "persist:ledger" and p == "before":
            armed[0] = True
        if b == "persist:ledger" and p == "after":
            armed[0] = False
        if kind == "ledger_commit_failure" and b == "persist:manifest" and p == "file_fsync" \
                and ctx.get("event") == "commit_draw" and not fired[0]:
            fired[0] = True
            raise OSError(errno.EIO, "completed draw ledger commit failure")
        if kind == "error_handling_failure":
            if b == "draw_execute" and p == "after" and not fired[0]:
                fired[0] = True
                raise RuntimeError("completed draw before ledger handoff")
            if recorder.stopping and b == "persist:ledger" and p == "file_fsync":
                raise OSError(errno.EIO, "completed draw failure during error handling")
    recorder.hook = hook
    captured = []

    def tracer(frame, event, arg):
        if event == "line" and frame.f_code is K.H.run_draw.__code__ \
                and linecache.getline(frame.f_code.co_filename, frame.f_lineno).strip() == "return out" \
                and not fired[0]:
            fired[0] = True
            captured.append(copy.deepcopy(frame.f_locals["out"]))
            raise RuntimeError("completed draw interrupted immediately before return")
        return tracer
    previous = sys.gettrace()
    if kind == "interruption_before_return":
        sys.settrace(tracer)
    try:
        result = S.attempt(recorder.run)
    finally:
        sys.settrace(previous)
    verdict = fixture.classify(str(out))
    live = out / "live"
    rows = json.loads((live / "draws.json").read_text())
    retained = list(rows)
    pending = []
    failures = []
    for p in live.iterdir():
        if p.name not in ("pending-draw.json", "persistence-failure.json") and not p.name.startswith(".tmp-"):
            continue
        try:
            doc = json.loads(p.read_bytes())
        except (ValueError, UnicodeDecodeError):
            continue
        candidates = doc if isinstance(doc, list) else [doc.get("draw"), doc.get("pending")]
        pending.extend(d for d in candidates if isinstance(d, dict) and d.get("status") == "completed")
        if p.name == "persistence-failure.json":
            failures.append(doc)
    retained.extend(pending)
    complete = {d["seq"]: d for d in retained if d.get("status") == "completed"}
    agree = fired[0] and 1 in complete and bool(complete[1]["record"]) and bool(complete[1]["calls"])
    # Independently replay the retained completed result, including a pending row of an INVALID archive.
    replay = False
    if 1 in complete:
        row = complete[1]
        pool = fixture.ctx.opener(binary)
        try:
            K.H.replay_draw(row, fixture.ctx.by_id[row["context"]], fixture.ctx.settings[row["model"]],
                            pool, fixture.ctx.memory(fixture.ctx.by_id[row["context"]], row["model"]))
            replay = True
        finally:
            pool.close()
    agree = agree and replay and (verdict["state"] == "PARTIAL" if kind == "interruption_before_return"
                                 else verdict["state"] == "INVALID" and not (live / K.SEAL).exists())
    if captured:
        agree = agree and K.canon(complete[1]) == K.canon(captured[0])
    return {"case": kind, "binding": "B-HANDOFF-001", "fired": fired[0], "classification": verdict["state"],
            "recorder": S.recorder_label(result), "completedRetained": len(complete), "pendingCopies": len(pending),
            "journalDispatches": sum(r["record"] == "dispatch" for r in S.journal(str(out))),
            "failedCandidateRetained": any(p.name.startswith(".tmp-") for p in live.iterdir()),
            "completedReplay": replay, "failureEvidence": failures, "agree": bool(agree)}


def run(binary, scratch):
    Path(scratch).mkdir(parents=True, exist_ok=True)
    cases = [one(binary, scratch, k) for k in CASES]
    return {"schema": "dreamteam.v13k-completed-preservation/v1", "cases": cases,
            "summary": {"cases": len(cases), "agree": sum(c["agree"] for c in cases)}}


def main():
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--binary", required=True)
    ap.add_argument("--scratch", required=True)
    ap.add_argument("--out", required=True)
    a = ap.parse_args()
    doc = run(os.path.abspath(a.binary), os.path.abspath(a.scratch))
    # Full pending row data are retained in scratch; the committed summary keeps the exact errors and counts.
    for case in doc["cases"]:
        for report in case["failureEvidence"]:
            row = report.pop("pending", None)
            report["pendingRowSha256"] = K.sha256(K.canon(row)) if row else None
            report["candidates"] = ["failed ledger candidate" for _ in report["candidates"]]
    K.write_json(a.out, doc)
    print(json.dumps(doc["summary"]))
    if not all(c["agree"] for c in doc["cases"]):
        raise SystemExit(1)


if __name__ == "__main__":
    main()
