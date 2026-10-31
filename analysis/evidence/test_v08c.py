"""Tests of the V0.8c analyses: the readers of llm-agent 3's record, repeat
rule version 2, and refusal-response version 2. No model is contacted.

    python3 -I -B -m unittest discover -s analysis/evidence
"""

import json
import os
import sys
import unittest

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
sys.path.insert(0, os.path.join(HERE, "..", "..", "agents", "llm"))
import record_v3 as V3  # noqa: E402
import refusal_response as R  # noqa: E402
import repeat_v2 as P  # noqa: E402

SRC = {"path": "x.jsonl", "sha256": "0" * 64, "group": "g", "model": "m"}


def ext(emp, minutes=60, day="2026-11-17"):
    return {"type": "extend_shift", "employee": emp, "date": day, "minutes": minutes}


def at3(point, k, actions, codes=None, turn=None, details=None):
    """An llm-agent 3 attempt: actions (one operations batch) with a code
    per action ("ok" or a refusal code), "unparsed", or a details turn."""
    a = {"id": f"{point}.{k}", "point": point, "attempt": k, "turn": turn or ("first" if k == 1 else "retry"),
         "input": {"kind": "prompt" if k == 1 else "retry"}, "transport": "ok", "output": {"chars": 1},
         "parse": {"ok": True}, "reason": "r", "actions": None, "detailsRequested": details, "validation": None,
         "protocolError": None, "refusals": [], "error": None, "used": False}
    if actions == "unparsed":
        a.update(parse={"ok": False, "code": "invalid_json", "message": "x"}, reason=None, error="invalid_json: x")
        return a
    a["actions"] = actions
    if details or not actions:
        return a
    res = [{"ok": c == "ok", "code": None if c == "ok" else c,
            "message": None if c == "ok" else f"{c}: commitment ${1000 + 7 * point:,}.00 over headroom", "action": x}
           for x, c in zip(actions, codes)]
    a["validation"] = [{"actions": actions, "indices": list(range(len(actions))),
                        "answer": {"valid": all(c == "ok" for c in codes), "results": res}}]
    a["refusals"] = [{"attempt": k, "action": i, "type": x["type"], "code": c, "message": r["message"]}
                     for i, (x, c, r) in enumerate(zip(actions, codes, res)) if c != "ok"]
    if a["refusals"]:
        a["error"] = "refused: ..."
    return a


def rec3(point, attempts):
    used = [a for a in attempts if a["error"] is None and not a["detailsRequested"]]
    if used and used[-1] is attempts[-1]:
        attempts[-1]["used"] = True
    u = attempts[-1] if attempts[-1]["used"] else None
    return {"schema": V3.RECORD_SCHEMA, "point": point, "time": f"t{point}", "digests": {}, "attempts": attempts,
            "final": {"attempt": u["attempt"] if u else None,
                      "outcome": "fallback" if not u else ("actions" if u["actions"] else "no_action")},
            "batches": [u["actions"]] if u and u["actions"] else [],
            "outcome": "fallback" if not u else ("actions" if u["actions"] else "no_action"),
            "reason": "r", "refusals": [r for a in attempts for r in a["refusals"]],
            "errors": [f"attempt {a['attempt']}: {a['error']}" for a in attempts if a["error"]]}


def repeat_events(recs):
    evs, fs = P.analyze_recording(P.sequence(recs), SRC)
    return evs, fs


class RepeatV2(unittest.TestCase):
    def test_immediate_and_later_day_repeats_with_amounts(self):
        recs = [rec3(0, [at3(0, 1, [ext("E07")], ["insufficient_funds"]),
                         at3(0, 2, [ext("E07")], ["insufficient_funds"])]),
                rec3(1, [at3(1, 1, [ext("E07")], ["insufficient_funds"]), at3(1, 2, [], [])])]
        evs, fs = repeat_events(recs)
        self.assertEqual([e["status"] for e in evs], ["repeated_refused", "repeated_refused", "not_repeated"])
        self.assertTrue(evs[0]["repeat"]["samePoint"])
        self.assertFalse(evs[1]["repeat"]["samePoint"])
        # The messages state different amounts: v1's key differs, v2's does not.
        self.assertFalse(evs[1]["repeat"]["sameMessage"])
        self.assertEqual(len(fs), 1)
        self.assertEqual((fs[0]["count"], fs[0]["withinPoint"], fs[0]["acrossPoints"], fs[0]["distinctMessages"]),
                         (3, 1, 1, 2))

    def test_target_parameters_and_no_longer_applicable(self):
        recs = [rec3(0, [at3(0, 1, [ext("E07"), ext("E08")], ["after_close", "after_close"]),
                         at3(0, 2, [ext("E09"), ext("E07", 30)], ["ok", "ok"])]),
                rec3(1, [at3(1, 1, [ext("E08")], ["ok"])])]
        evs, fs = repeat_events(recs)
        e07, e08 = evs
        # A different target, and other values for the same target, are not repeats.
        self.assertEqual(e07["status"], "not_repeated")
        self.assertEqual(e07["related"], 1)
        # The identical action accepted later: the refusal no longer applied.
        self.assertEqual(e08["status"], "repeated_accepted")
        self.assertEqual(fs, [])

    def test_abstains_on_missing_evidence(self):
        v2 = {"point": 0, "attempts": 2, "outcome": "actions", "proposed": [ext("E09")],
              "validation": [{"valid": True, "results": [{"ok": True}]}],
              "refusals": [{"attempt": 1, "action": 0, "type": "extend_shift", "code": "after_close", "message": "m"}],
              "errors": ["attempt 1: refused: action 0: after_close: m"], "batches": [[ext("E09")]]}
        nxt = dict(v2, point=1)
        evs, _ = P.analyze_recording(P.sequence([v2, nxt]), SRC)
        # The refused attempt's own actions are not recorded.
        self.assertEqual([e["status"] for e in evs], ["insufficient_evidence", "insufficient_evidence"])
        known = {"point": 0, "attempts": 1, "outcome": "fallback", "proposed": [ext("E07")],
                 "validation": [{"valid": False, "results": [{"ok": False, "code": "after_close", "message": "m"}]}],
                 "refusals": [{"attempt": 1, "action": 0, "type": "extend_shift", "code": "after_close", "message": "m"}],
                 "errors": ["attempt 1: refused: action 0: after_close: m"], "batches": []}
        evs, _ = P.analyze_recording(P.sequence([known, nxt]), SRC)
        # Known, but a later attempt's actions are not recorded: unresolved.
        self.assertEqual(evs[0]["status"], "unresolved")
        self.assertEqual(evs[0]["laterUnrecorded"], 1)

    def test_versus_v1(self):
        recs = [rec3(0, [at3(0, 1, [ext("E07")], ["insufficient_funds"])]),
                rec3(1, [at3(1, 1, [ext("E07")], ["insufficient_funds"])])]
        evs, _ = repeat_events(recs)
        c = P.compare([], evs)
        self.assertEqual((c["v1Links"], c["v2Links"], c["v2OnlyMessageDiffers"]), (0, 1, 1))
        link = ((0, 1, "extend_shift", "insufficient_funds"), (1, 1, "extend_shift", "insufficient_funds"))
        self.assertEqual(P.compare([link], evs)["both"], 1)


class RefusalResponseV2(unittest.TestCase):
    def test_rr1_d1_parsed_reply_keeps_its_attempt(self):
        """A refused proposal then an unparseable retry (llm-agent 2 format):
        v1 compared the proposal with itself; v2 looks at the next point."""
        p0 = {"point": 0, "attempts": 2, "outcome": "fallback", "proposed": [ext("E07")],
              "validation": [{"valid": False, "results": [{"ok": False, "code": "lead_too_short", "message": "m"}]}],
              "refusals": [{"attempt": 1, "action": 0, "type": "extend_shift", "code": "lead_too_short", "message": "m"}],
              "errors": ["attempt 1: refused: action 0: lead_too_short: m", "attempt 2: invalid_json: x"],
              "batches": []}
        p1 = {"point": 1, "attempts": 1, "outcome": "no_action", "proposed": [], "validation": None, "refusals": [],
              "errors": [], "batches": []}
        v1 = R.analyze_sequence([p0, p1], None, SRC)
        v2 = R.analyze_sequence_v2([p0, p1], None, SRC)
        self.assertEqual(v1[0]["category"], "identical_repeated")
        self.assertEqual(v1[0]["response"], {"point": 0, "attempt": 2})
        self.assertEqual((v2[0]["category"], v2[0]["response"], v2[0]["skippedUnparsed"]),
                         ("no_action", {"point": 1, "attempt": 1}, 1))

    def test_v3_skips_unparsed_and_details(self):
        recs = [rec3(0, [at3(0, 1, [ext("E07")], ["insufficient_funds"]), at3(0, 2, "unparsed"),
                         at3(0, 3, [ext("E07", 30)], ["ok"])]),
                rec3(1, [at3(1, 1, [], [], turn="first", details=["finance"]), at3(1, 2, [ext("E08")], ["ok"])])]
        evs = R.analyze_sequence_v2(recs, None, SRC)
        self.assertEqual([(e["category"], e["skippedUnparsed"]) for e in evs], [("modified_proposal", 1)])
        recs[0]["attempts"][2] = at3(0, 3, "unparsed")
        evs = R.analyze_sequence_v2(recs, None, SRC)
        # The details request at point 1 is not a proposal.
        self.assertEqual((evs[0]["category"], evs[0]["response"]), ("different_proposal", {"point": 1, "attempt": 2}))

    def test_unrecorded_next_proposal_is_unavailable(self):
        p0 = {"point": 0, "attempts": 1, "outcome": "fallback", "proposed": [ext("E07")],
              "validation": [{"valid": False, "results": [{"ok": False, "code": "after_close", "message": "m"}]}],
              "refusals": [{"attempt": 1, "action": 0, "type": "extend_shift", "code": "after_close", "message": "m"}],
              "errors": ["attempt 1: refused: action 0: after_close: m"], "batches": []}
        p1 = {"point": 1, "attempts": 2, "outcome": "actions", "proposed": [ext("E09")],
              "validation": [{"valid": True, "results": [{"ok": True}]}],
              "refusals": [{"attempt": 1, "action": 0, "type": "extend_shift", "code": "after_close", "message": "m"}],
              "errors": ["attempt 1: refused: action 0: after_close: m"], "batches": [[ext("E09")]]}
        e = R.analyze_sequence_v2([p0, p1], None, SRC)[0]
        self.assertEqual((e["category"], e["unavailable"]),
                         ("evidence_unavailable", "the next proposal's actions are not recorded"))
        # Version 1 used the next point's final reply.
        self.assertEqual(R.analyze_sequence([p0, p1], None, SRC)[0]["category"], "different_proposal")


class V2View(unittest.TestCase):
    """The llm-agent 2 view of an llm-agent 3 record is what llm-agent 2
    recorded for the same decision point."""

    def test_same_as_llm_agent_2(self):
        sys.path.insert(0, os.path.join(HERE, "..", "..", "agents", "llm"))
        import test_llm_agent3 as T
        for case in T.CASES:
            with self.subTest(case):
                r2, _, _, _ = T.run(T.A2, case)
                r3, _, _, _ = T.run(T.A3, case)
                for (_, x), (_, y) in zip(r2, r3):
                    v = V3.v2_view(json.loads(json.dumps(y)))
                    x = json.loads(json.dumps(x))
                    self.assertEqual(set(v), set(x))
                    for k in x:
                        if k != "agentVersion":  # 3, not 2: the view says which agent wrote it
                            self.assertEqual(v[k], x[k], k)

    def test_attempt_view_leaves_out_details(self):
        r = rec3(0, [at3(0, 1, [], [], details=["x"]), at3(0, 2, [ext("E07")], ["ok"])])
        self.assertEqual([a["attempt"] for a in V3.attempt_view(r)["attempts"]], [2])


if __name__ == "__main__":
    unittest.main()
