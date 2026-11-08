#!/usr/bin/env python3
"""Contradictory-evidence archives (Phase 7, DT13H-04/05): genuine V0.13i
archives are recorded, then each is altered the way a careless or deliberate
forger could, re-sealing every dependent hash, count and mirror the forger
could recompute, so that only the targeted contradiction remains. Every
mutant must classify INVALID with the targeted invariant among its
violations, and analysis must refuse it.

    python3 -I -B analysis/alternatives/v13i_mutations.py --binary ./dt --scratch DIR --out FILE
"""
import argparse
import base64
import copy
import hashlib
import json
import os
from pathlib import Path
import shutil
import sys

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))
import v13i_scenarios as S  # noqa: E402

SCHEMA = "dreamteam.v13i-mutations/v1"
D = S.D


def record_bases(binary, scratch):
    """Genuine archives: COMPLETE (both arms), PARTIAL health stop, PARTIAL
    aborted workflow with a recorded call, PARTIAL halt row, PARTIAL cleanup."""
    bases = {}
    both = S.Fixture(binary, arms=("control", "treatment"))
    one = S.Fixture(binary)

    def rec(name, fixture, **kw):
        out = os.path.join(scratch, "base-" + name)
        shutil.rmtree(out, ignore_errors=True)
        fixture.record(out, **kw)
        v = fixture.classify(out)
        if v["state"] == "INVALID":
            raise SystemExit(f"base {name} is not valid: {v['violations']}")
        bases[name] = (fixture, out, v["state"])
    clock = S.Clock()
    rec("complete", both, health=S.Health(clock), clock=clock)
    clock = S.Clock()
    rec("health_stop", one, health=S.Health(clock, modes={1: "refused"}), clock=clock)
    clock = S.Clock()

    calls = [0]

    def second_call_fails(payload):
        calls[0] += 1
        if calls[0] == 2:
            return None, "", "timeout: scripted"
        if calls[0] == 3:
            raise RuntimeError("scripted transport failure after a failed exchange")
        return D.scripted([S.F.ABANDON] * 4)(payload)
    rec("aborted_workflow", one, health=S.Health(clock), clock=clock, model=second_call_fails)
    clock = S.Clock()
    rec("halt_row", one.with_limits(maxModelCallsPerDraw=0), health=S.Health(clock), clock=clock)
    clock = S.Clock()
    pool = one.ctx.opener(binary)
    rec("cleanup_failed", one, health=S.Health(clock), clock=clock,
        opener=S.FailingOpener(pool, close_error=RuntimeError("scripted pool release failure")))
    return bases


class Arc:
    def __init__(self, out):
        self.live = Path(out) / "live"
        self.files = {p.name: json.loads(p.read_text()) for p in self.live.iterdir() if p.suffix == ".json"}

    @property
    def doc(self):
        return self.files["run.json"]

    @property
    def rows(self):
        return self.files["draws.json"]

    def write(self, reseal=True, commits=True, counters=False, event=True):
        doc, rows = self.doc, self.rows
        if commits:
            k = 0
            for e in doc["journal"]:
                if e.get("event") == "commit_draw" and k < len(rows):
                    e.update(rowSha256=D.sha256(D.canon(rows[k])), modelCalls=len(rows[k]["calls"]),
                             status=rows[k]["status"], draw=rows[k]["seq"])
                    k += 1
        if counters:
            completed = sum(r["status"] == "completed" for r in rows)
            doc["counters"]["drawsCompleted"] = completed
        if event and doc.get("failureEvent") is not None:
            self.files["failure-event.json"] = copy.deepcopy(doc["failureEvent"])
        for name, value in self.files.items():
            if name != "run.json":
                (self.live / name).write_text(json.dumps(value, sort_keys=True, ensure_ascii=False, indent=1) + "\n")
        if reseal and isinstance(doc.get("finalization"), dict):
            for n in list(doc["finalization"].get("artifacts", {})):
                p = self.live / n
                doc["finalization"]["artifacts"][n] = hashlib.sha256(p.read_bytes()).hexdigest() if p.exists() else None
        (self.live / "run.json").write_text(json.dumps(doc, sort_keys=True, ensure_ascii=False, indent=1) + "\n")


def body(obj_bytes):
    return {"bytes": len(obj_bytes), "sha256": hashlib.sha256(obj_bytes).hexdigest(),
            "base64": base64.b64encode(obj_bytes).decode()}


def health(a, k=0):
    return a.doc["healthChecks"][k]["serverState"]["health"]


def set_initial_health(a, raw, category, outcome):
    h = health(a)
    h.update(body=body(raw), category=category, outcome=outcome)
    a.files["server-before.json"] = copy.deepcopy(a.doc["healthChecks"][0]["serverState"])


def first(a, event):
    return next(e for e in a.doc["journal"] if e["event"] == event)


def m(name, base, expect, fn, **write):
    return {"name": name, "base": base, "expect": expect, "fn": fn, "write": write}


MUTATIONS = [
    # --- DT13H-04 and Phase 7 contradictions ---------------------------------------------
    m("completed_row_with_cleanup_failure", "complete", {"L02"},
      lambda a: a.rows[0].update(cleanupFailure={"type": "RuntimeError", "message": "session close failed"})),
    m("complete_with_unhealthy_initial_observation", "complete", {"L07"},
      lambda a: set_initial_health(a, b'{"status":"unhealthy"}', "unhealthy_status", "UNHEALTHY")),
    m("complete_with_unhealthy_server_before_file_only", "complete", {"L07"},
      lambda a: a.files["server-before.json"]["health"].update(outcome="UNHEALTHY", category="unhealthy_status",
                                                              body=body(b'{"status":"unhealthy"}'))),
    m("engine_open_failure_with_workflow_evidence", "aborted_workflow", {"L06"},
      lambda a: a.files["aborted-draw.json"].update(failureStage="engine_open")),
    m("engine_open_failure_marked_ready_without_calls", "aborted_workflow", {"L06"},
      lambda a: a.files["aborted-draw.json"].update(failureStage="engine_open", calls=[], modelCallsCompleted=0)),
    m("draw_setup_failure_with_engine_log", "aborted_workflow", {"L06"},
      lambda a: a.files["aborted-draw.json"].update(failureStage="draw_setup", engineInitialization="NOT_STARTED")),
    m("impossible_run_model_calls", "complete", {"L03"}, lambda a: a.doc["counters"].update(modelCalls=999)),
    m("impossible_per_draw_calls", "complete", {"L04"},
      lambda a: (a.rows[0].update(calls=[dict(c, n=i + 1) for i, c in enumerate(a.rows[0]["calls"] * 7)],
                                  modelCallsCompleted=len(a.rows[0]["calls"]) * 7),
                 a.doc["counters"].update(modelCalls=sum(len(r["calls"]) for r in a.rows)))),
    m("failure_event_call_count", "health_stop", {"L19"}, lambda a: a.doc["failureEvent"].update(modelCalls=3)),
    m("duplicate_completed_draws", "complete", {"L01"}, lambda a: a.rows.__setitem__(1, copy.deepcopy(a.rows[0]))),
    m("missing_schedule_entry", "complete", {"L01"}, lambda a: a.rows.pop(0)),
    m("truncated_ledger_claiming_complete", "complete", {"L17", "L10"}, lambda a: a.rows.pop()),
    m("aborted_status_row_in_ledger", "complete", {"L01"}, lambda a: a.rows[1].update(status="aborted")),
    m("aborted_artifact_beside_complete", "complete", {"L05", "L16"},
      lambda a: a.files.__setitem__("aborted-draw.json", dict(copy.deepcopy(a.rows[1]), status="aborted"))),
    m("stop_code_differs_in_failure_event_file", "health_stop", {"L16"},
      lambda a: a.files["failure-event.json"]["stop"].update(code="operator"), event=False),
    m("stop_code_differs_from_journal", "health_stop", {"L19"},
      lambda a: (a.doc["stop"].update(code="operator"), a.doc["failureEvent"]["stop"].update(code="operator"))),
    m("stop_state_differs_from_journal", "health_stop", {"L19"},
      lambda a: (a.doc["stop"].update(state="READY"), a.doc["failureEvent"].update(state="READY"),
                 a.doc["failureEvent"]["stop"].update(state="READY"))),
    m("second_stop_transition", "health_stop", {"L14", "L19"},
      lambda a: a.doc["journal"].insert(4, dict(first(a, "stop"), n=0))),
    m("halt_row_with_operator_stop", "halt_row", {"L06"},
      lambda a: (a.doc["stop"].update(code="operator", evidence=None, failure={"type": "KeyboardInterrupt", "message": ""}),
                 a.doc["failureEvent"].update(stop=copy.deepcopy(a.doc["stop"])),
                 next(e for e in a.doc["journal"] if e["event"] == "stop").update(code="operator"))),
    m("illegal_transition", "complete", {"L14"}, lambda a: a.doc["journal"][2].update(event="commit_draw")),
    m("unknown_event", "complete", {"L14"}, lambda a: a.doc["journal"][2].update(event="resume")),
    m("unknown_state", "complete", {"L14"}, lambda a: a.doc["journal"][2].update(to="WAITING")),
    m("transitions_reordered", "complete", {"L14"},
      lambda a: a.doc["journal"].__setitem__(slice(3, 5), list(reversed(a.doc["journal"][3:5])))),
    m("journal_without_finalization", "complete", {"L14"},
      lambda a: (a.doc["journal"].pop(), a.doc.update(state="FINALIZING", status="FINALIZING"))),
    m("unsupported_terminal_status", "complete", {"L14"}, lambda a: a.doc.update(status="ABORTED")),
    m("terminal_state_invalid", "complete", {"L14"}, lambda a: a.doc.update(state="INVALID", status="INVALID")),
    m("partial_claimed_complete", "health_stop", {"L14", "L10", "L09"},
      lambda a: (a.doc.update(state="COMPLETE", status="COMPLETE"), a.doc["journal"][-1].update(event="finalize_complete", to="COMPLETE"))),
    m("complete_with_cleanup_failure", "complete", {"L08"},
      lambda a: a.doc["cleanup"].update(status="FAILED", failure={"type": "RuntimeError", "message": "x"})),
    m("cleanup_failure_replaces_primary", "health_stop", {"L19", "L08"},
      lambda a: (a.doc["cleanup"].update(status="FAILED", failure={"type": "RuntimeError", "message": "release"}),
                 a.doc["stop"].update(code="driver", state="CLEANUP", failure={"type": "RuntimeError", "message": "release"}))),
    m("cleanup_created_stop_mislabelled", "cleanup_failed", {"L06"},
      lambda a: (a.doc["stop"].update(code="operator"), a.doc["failureEvent"]["stop"].update(code="operator"))),
    m("server_before_byte_changed_unsealed", "complete", {"L16"},
      lambda a: a.files["server-before.json"].update(serverTitle="changed"), reseal=False),
    m("ledger_changed_unsealed", "complete", {"L16", "L17"},
      lambda a: a.rows[0]["record"].update(retryRequests=a.rows[0]["record"]["retryRequests"] + 1), reseal=False, commits=False),
    m("finalization_hash_forged", "complete", {"L16"},
      lambda a: a.doc["finalization"]["artifacts"].update({"draws.json": "0" * 64}), reseal=False),
    m("commit_row_hash_stale", "complete", {"L17"},
      lambda a: a.rows[0]["record"].update(retryRequests=a.rows[0]["record"]["retryRequests"] + 1), commits=False),
    m("counters_not_conserved", "health_stop", {"L03"}, lambda a: a.doc["counters"].update(drawsNotRun=7)),
    m("reconciled_commit_without_stop", "complete", {"L17"}, lambda a: first(a, "commit_draw").update(reconciled=True)),
    m("resume_permitted", "complete", {"L18"}, lambda a: a.doc.update(resume="allowed")),
    m("run_identity_not_uuid", "complete", {"L15"}, lambda a: a.doc.update(runId="run-1")),
    m("lifecycle_contract_differs", "complete", {"L15"}, lambda a: a.doc["lifecycle"].update(sha256="0" * 64)),
    m("recording_claims_verification", "complete", {"L15"}, lambda a: a.doc.update(verificationStatus="VERIFIED")),
    # --- DT13H-01 health evidence ---------------------------------------------------------
    m("healthy_label_on_contradictory_body", "complete", {"L07"},
      lambda a: set_initial_health(a, b'{"status":"healthy","error":"URLError"}', "healthy", "HEALTHY")),
    m("healthy_with_exception", "complete", {"L07"},
      lambda a: (health(a).update(exception={"type": "URLError", "reasonType": "ConnectionRefusedError"}),
                 a.files.__setitem__("server-before.json", copy.deepcopy(a.doc["healthChecks"][0]["serverState"])))),
    m("health_field_missing", "health_stop", {"L07"},
      lambda a: (a.doc["healthChecks"][0]["serverState"].pop("health"),
                 a.files.__setitem__("server-before.json", copy.deepcopy(a.doc["healthChecks"][0]["serverState"])))),
    m("unknown_read_failure_category", "health_stop", {"L07"},
      lambda a: (health(a).update(category="invented_failure"),
                 a.files.__setitem__("server-before.json", copy.deepcopy(a.doc["healthChecks"][0]["serverState"])))),
    m("read_failure_class_mismatch", "health_stop", {"L07"},
      lambda a: (health(a).update(exception={"type": "TimeoutError", "reasonType": None}),
                 a.files.__setitem__("server-before.json", copy.deepcopy(a.doc["healthChecks"][0]["serverState"])))),
    m("health_stop_on_healthy_observation", "complete", {"L07", "L14"},
      lambda a: a.doc["journal"][2].update(event="stop", to="CLEANUP")),
    m("missing_round_health_check", "complete", {"L07", "L14"},
      lambda a: [a.doc["journal"].remove(e) for e in [x for x in a.doc["journal"] if x["event"] in ("begin_health_check", "health_passed")][2:4]]),
    # --- DT13H-05 timestamps ------------------------------------------------------------------
    m("null_journal_timestamp", "complete", {"L13"}, lambda a: a.doc["journal"][3].update(timestamp=None)),
    m("unparseable_journal_timestamp", "complete", {"L13"}, lambda a: a.doc["journal"][3].update(timestamp="not-a-time")),
    m("impossible_calendar_timestamp", "complete", {"L13"}, lambda a: a.doc["journal"][3].update(timestamp="2026-02-30T00:00:00Z")),
    m("journal_time_reversed", "complete", {"L13"}, lambda a: a.doc["journal"][3].update(timestamp="2025-01-01T00:00:00Z")),
    m("run_started_after_ended", "complete", {"L13"},
      lambda a: (a.doc.update(started="2027-01-01T00:00:00Z"), a.doc["journal"][0].update(timestamp="2027-01-01T00:00:00Z"))),
    m("run_started_missing", "complete", {"L13"}, lambda a: a.doc.update(started=None)),
    m("run_ended_null", "complete", {"L13"}, lambda a: a.doc.update(ended=None)),
    m("failure_event_timestamp_null", "health_stop", {"L19"}, lambda a: a.doc["failureEvent"].update(timestamp=None)),
    m("observation_end_before_start", "complete", {"L07"},
      lambda a: (health(a).update(ended="2025-01-01T00:00:00Z"),
                 a.files.__setitem__("server-before.json", copy.deepcopy(a.doc["healthChecks"][0]["serverState"])))),
    m("call_sent_missing", "complete", {"L13"}, lambda a: a.rows[0]["calls"][0].update(sent=None)),
    m("call_sent_outside_draw", "complete", {"L13"}, lambda a: a.rows[0]["calls"][0].update(sent="2030-01-01T00:00:00Z")),
]

FILE_MUTATIONS = [
    ("marker_present", "complete", {"L16"}, lambda live: (live / "FINALIZING").write_text("{}")),
    ("stray_temporary_file", "complete", {"L16"}, lambda live: (live / ".metadata-x").write_text("{}")),
    ("legacy_stop_event_file", "health_stop", {"L16"}, lambda live: (live / "stop-event.json").write_text("{}")),
    ("missing_manifest", "complete", {"L16"}, lambda live: (live / "run.json").unlink()),
    ("truncated_manifest", "complete", {"L16"}, lambda live: (live / "run.json").write_text("{")),
    ("missing_ledger", "complete", {"L16"}, lambda live: (live / "draws.json").unlink()),
    ("truncated_ledger", "complete", {"L16"}, lambda live: (live / "draws.json").write_text("[")),
    ("missing_failure_event", "health_stop", {"L16"}, lambda live: (live / "failure-event.json").unlink()),
    ("mismatched_failure_event", "health_stop", {"L16"}, lambda live: (live / "failure-event.json").write_text("{}\n")),
    ("missing_aborted_artifact", "aborted_workflow", {"L05", "L16"}, lambda live: (live / "aborted-draw.json").unlink()),
    ("missing_server_before", "complete", {"L07", "L16"}, lambda live: (live / "server-before.json").unlink()),
]


def run(binary, scratch, analyze=True, names=None):
    bases = record_bases(binary, scratch)
    results = {"bases": {k: v[2] for k, v in bases.items()}, "mutants": []}
    for mu in MUTATIONS + [{"name": n, "base": b, "expect": e, "file": f} for n, b, e, f in FILE_MUTATIONS]:
        if names is not None and mu["name"] not in names:
            continue
        fixture, src, _ = bases[mu["base"]]
        out = os.path.join(scratch, "mutant-" + mu["name"])
        shutil.rmtree(out, ignore_errors=True)
        shutil.copytree(src, out)
        if "file" in mu:
            mu["file"](Path(out) / "live")
        else:
            a = Arc(out)
            mu["fn"](a)
            a.write(**mu["write"])
        v = fixture.classify(out)
        found = sorted({x["invariant"] for x in v["violations"]})
        refused = None
        if analyze:
            try:
                fixture.analyze(out)
                refused = False
            except D.ExperimentError:
                refused = True
        ok = v["state"] == "INVALID" and bool(set(mu["expect"]) & set(found)) and refused is not False
        results["mutants"].append({"name": mu["name"], "base": mu["base"], "expected": sorted(mu["expect"]),
                                   "classification": v["state"], "violations": found, "analysisRefused": refused,
                                   "agree": ok})
        shutil.rmtree(out, ignore_errors=True)
    results["summary"] = {"mutants": len(results["mutants"]), "agree": sum(x["agree"] for x in results["mutants"])}
    return results


def main(argv=None):
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    ap.add_argument("--binary", required=True)
    ap.add_argument("--scratch", required=True)
    ap.add_argument("--out", required=True)
    a = ap.parse_args(argv)
    os.makedirs(a.scratch, exist_ok=True)
    r = run(a.binary, a.scratch)
    D.write_json(a.out, dict({"schema": SCHEMA, "notice": "Scripted archives only; no inference."}, **r))
    print(f"mutations: {r['summary']['agree']} of {r['summary']['mutants']} rejected as expected")
    for x in r["mutants"]:
        if not x["agree"]:
            print("DISAGREE", x["name"], x["classification"], x["violations"])
    if r["summary"]["agree"] != r["summary"]["mutants"]:
        raise SystemExit(1)


if __name__ == "__main__":
    main()
