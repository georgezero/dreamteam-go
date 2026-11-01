"""Tests of the V0.10b decision experiment (decision_experiment.py): the
arms' requests, the schedule, the classification, the engine sessions (a
scripted engine), the financial records, the live client and the checks
of the committed points. No model and no engine binary are used.

    python3 -B -m unittest discover -s analysis/repeatability
"""

import copy
import glob
import hashlib
import json
import os
import random
import sys
import tempfile
import unittest

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(os.path.dirname(HERE))
sys.path.insert(0, HERE)
import decision_experiment as X  # noqa: E402

A3, A4, S = X.A3, X.A4, X.S
V = os.path.join(ROOT, "docs", "benchmark", "v10b-dev")

HIRE = {"type": "hire", "requestId": "h1", "role": "technologist", "offer": "o1"}
HIRE2 = {"type": "hire", "requestId": "h2", "role": "technologist", "offer": "o2"}
WALK = {"type": "set_walkin_policy", "policy": "limited"}
TYPES = {"hire", "set_walkin_policy", "contract_teleradiology", "set_roster"}
PREVIEW = {"preview": {"ok": True, "results": [{"ok": True, "code": "accepted", "action": HIRE}]}}


def reply(actions, reason="r"):
    return json.dumps({"reason": reason, "actions": actions})


def row(content=None, status=200, error=None, parsed=True, **kw):
    return dict(kw, content=content, status=status, error=error, parsedBody=parsed)


def recordings_digest():
    h = hashlib.sha256()
    for pat in S.RECORDINGS:
        for p in sorted(glob.glob(os.path.join(ROOT, pat))):
            with open(p, "rb") as f:
                h.update(f.read())
    return h.hexdigest()


class Engine:
    """A scripted engine: one decision point per day; digests change with
    the day and with each applied batch; a hire is refused when its offer
    is "bad"."""

    def __init__(self):
        self.day, self.applied = 0, []

    def digests(self):
        return {"log": hashlib.sha256(f"{self.day}|{self.applied}".encode()).hexdigest()}

    def obs(self):
        return {"time": f"d{self.day}", "finance": {"cashCents": 100, "financing": {"solvency": {
            "currentLiabilitiesCents": 0, "reservedCents": 0, "fundingHeadroomCents": 1000,
            "creditLimitCents": 500, "creditDrawnCents": 0, "creditAvailableCents": 500}}}, "staff": []}

    def results(self, acts):
        return [{"ok": a.get("offer") != "bad", "code": "accepted" if a.get("offer") != "bad" else "invalid_args",
                 "message": ""} for a in acts]

    def __call__(self, r):
        op, args = r["op"], r.get("args") or {}
        if op == "hello":
            res = {"clock": "c", "scenario": {"fingerprint": "f"}}
        elif op == "digests":
            res = {"digests": self.digests()}
        elif op == "observe":
            res = {"observation": self.obs()}
        elif op == "actions":
            res = {"types": [{"type": t} for t in sorted(TYPES)]}
        elif op == "advance":
            self.day += 1
            res = {"clock": args.get("to")}
        elif op == "preview":
            res = {"preview": {"ok": True, "funding": {"commitmentCents": 700, "headroomCents": 1000},
                               "guaranteed": [], "hires": [{"requestId": a.get("requestId"), "startDate": "s"}
                                                           for a in args["actions"]]}}
        elif op == "validate":
            rs = self.results(args["actions"])
            res = {"valid": all(x["ok"] for x in rs), "results": rs}
        elif op == "apply":
            rs = self.results(args["actions"])
            ok = all(x["ok"] for x in rs)
            if ok:
                self.applied.append(json.dumps(args["actions"], sort_keys=True))
            res = {"applied": ok, "results": rs}
        else:
            return {"id": r["id"], "ok": True, "result": {}}
        return {"id": r["id"], "ok": True, "result": res}


def recorded_run(days=3):
    """A run with no action at every point, recorded from the scripted
    engine: (run, pairs, the digests at each point)."""
    eng, pairs, pts = Engine(), [], []

    def rec(r):
        pairs.append((r, eng(r)))
        return pairs[-1][1]
    rec({"id": "h1", "op": "hello"})
    for d in range(days):
        pts.append(rec({"id": f"g{d}", "op": "digests"})["result"]["digests"])
        rec({"id": f"o{d}", "op": "observe", "args": {"compact": True, "recentLog": 0}})
        rec({"id": f"c{d}", "op": "actions"})
        rec({"id": f"a{d}", "op": "advance", "args": {"to": f"2026-10-{13 + d}T07:00:00"}})
    rec({"id": "gz", "op": "digests"})
    return {"id": "r", "scenario": {"ref": "s"}, "seed": 1}, pairs, pts


def fake_open(run, pairs):
    def opener(binary, run_dir):
        return X.Session(run, pairs, Engine())
    return opener


def context(proposal_actions=(HIRE,), digests=None):
    req = {"model": "m", "messages": [{"role": "system", "content": "s"}, {"role": "user", "content": "p"}],
           "temperature": 0, "max_tokens": 4096, "response_format": {"type": "json_object"}}
    prop = reply(list(proposal_actions))
    texts = {"control": X.CONTROL_TEXT, "treatment": X.treatment_text(PREVIEW, list(proposal_actions))}
    arms = {a: {"text": t, "textSha256": X.sha256(t), "textChars": len(t),
                "requestSha256": X.sha256(X.body_bytes(X.arm_request(req, prop, t)))} for a, t in texts.items()}
    return {"id": "pt", "model": "gemma4-26b-a4b", "recording": "r.jsonl", "point": 1, "digests": digests,
            "request": req, "proposal": {"content": prop, "actions": list(proposal_actions)}, "arms": arms}


class Requests(unittest.TestCase):
    def setUp(self):
        self.e = context((WALK, HIRE))
        self.c = X.arm_request(self.e["request"], self.e["proposal"]["content"], self.e["arms"]["control"]["text"])
        self.t = X.arm_request(self.e["request"], self.e["proposal"]["content"], self.e["arms"]["treatment"]["text"])

    def test_identical_shared_state(self):
        # Every message before the arm's own is the same, and the settings.
        self.assertEqual(self.c["messages"][:-1], self.t["messages"][:-1])
        self.assertEqual({k: v for k, v in self.c.items() if k != "messages"},
                         {k: v for k, v in self.t.items() if k != "messages"})
        self.assertEqual(list(self.c), list(self.e["request"]))

    def test_identical_initial_proposal(self):
        for b in (self.c, self.t):
            self.assertEqual(b["messages"][-2], {"role": "assistant", "content": self.e["proposal"]["content"]})

    def test_control_construction(self):
        text = self.c["messages"][-1]["content"]
        self.assertEqual(text, X.CONTROL_TEXT)
        self.assertTrue(text.startswith("CONTINUE. Nothing has been submitted yet.\n\nNow reply"))
        self.assertTrue(self.t["messages"][-1]["content"].endswith(X.FINAL_REQUEST))
        self.assertNotIn("preview", text.lower())
        self.assertIn('{"reason": ..., "actions": [...]}', text)

    def test_treatment_construction(self):
        text = self.t["messages"][-1]["content"]
        self.assertEqual(text, A4.review_text(PREVIEW, [1], True))
        self.assertIn(A4.OPS_NOTE, text)
        self.assertEqual(X.treatment_text(PREVIEW, [HIRE]), A4.review_text(PREVIEW, [0], False))
        self.assertIn(A3.compact(PREVIEW), text)


class Schedule(unittest.TestCase):
    def setUp(self):
        self.proto, _, _, self.contexts = X.load_protocol(os.path.join(V, "protocol.json"))

    def test_reproducible_and_balanced(self):
        a, b = X.schedule(self.proto, self.contexts), X.schedule(self.proto, list(self.contexts))
        self.assertEqual(a, b)
        self.assertEqual(len(a), 320)
        self.assertEqual([r["seq"] for r in a], list(range(1, 321)))
        for m in self.proto["modelOrder"]:
            ids = {e["id"] for e in self.contexts if e["model"] == m}
            for d in range(1, 21):
                cells = [(r["point"], r["arm"]) for r in a if r["model"] == m and r["draw"] == d]
                self.assertEqual(sorted(cells), sorted((p, x) for p in ids for x in X.ARMS))
        # Gemma's block first, then Qwen's.
        self.assertEqual([r["model"] for r in a], sorted((r["model"] for r in a), key=self.proto["modelOrder"].index))

    def test_committed_schedule(self):
        with open(os.path.join(V, "schedule.json"), encoding="utf-8") as f:
            doc = json.load(f)
        _, psha, man, _ = X.load_protocol(os.path.join(V, "protocol.json"))
        self.assertEqual(doc, json.loads(json.dumps(X.schedule_doc(self.proto, psha, man,
                                                                   X.schedule(self.proto, self.contexts)))))


class BranchPoints(unittest.TestCase):
    def test_committed_points_identity(self):
        before = recordings_digest()
        m, contexts = X.load_points(os.path.join(V, "points"))
        self.assertEqual([c["id"] for c in X.select(copy.deepcopy(m["candidates"]))], [e["id"] for e in contexts])
        for e in contexts:
            rec = S.recording_point(ROOT, e["recording"], e["point"])
            X.check_context(e, rec, e["preview"]["answer"])
            self.assertEqual(e["contextKey"], X.context_key(e["model"], rec, rec["attempts"][e["attempt"] - 1]))
            self.assertEqual(e["digests"], rec["digests"])
        self.assertEqual(before, recordings_digest())

    def test_missing_evidence(self):
        m, contexts = X.load_points(os.path.join(V, "points"))
        e = contexts[0]
        rec = S.recording_point(ROOT, e["recording"], e["point"])
        bad = copy.deepcopy(e)
        bad["proposal"]["content"] += " "
        with self.assertRaisesRegex(X.ExperimentError, "proposal"):
            X.check_context(bad, rec, e["preview"]["answer"])
        bad = copy.deepcopy(e)
        bad["request"]["messages"][1]["content"] += " "
        with self.assertRaisesRegex(X.ExperimentError, "messages"):
            X.check_context(bad, rec, e["preview"]["answer"])
        with self.assertRaisesRegex(X.ExperimentError, "preview"):
            X.check_context(e, rec, {"preview": {"ok": False}})
        few = [c for c in copy.deepcopy(m["candidates"]) if not (c["model"] == "gemma4-26b-a4b" and
                                                                  c["stratum"] == "refused" and c["point"] == 27)]
        with self.assertRaisesRegex(X.ExperimentError, "2 distinct eligible contexts, 3 declared"):
            X.select(few)

    def test_duplicate_context_taken_once(self):
        m, _ = X.load_points(os.path.join(V, "points"))
        dup = [c for c in m["candidates"] if c["model"] == "gemma4-26b-a4b" and c["stratum"] == "accepted"]
        self.assertEqual(len(dup), 2)
        self.assertEqual(len({c["contextKey"] for c in dup}), 1)
        self.assertEqual(sum(1 for s in m["selected"] if s["contextKey"] == dup[0]["contextKey"]), 1)


class Classify(unittest.TestCase):
    PROP = [WALK, HIRE]

    def test_kept(self):
        self.assertEqual(X.classify(row(reply([HIRE, WALK])), self.PROP, TYPES)[0], "kept")

    def test_revised(self):
        o, why, acts = X.classify(row(reply([WALK, HIRE2])), self.PROP, TYPES)
        self.assertEqual((o, why, acts), ("revised", None, [WALK, HIRE2]))
        ch = X.changes(self.PROP, acts)
        self.assertEqual(ch["changed"], [{"type": "hire", "fields": ["offer"]}])
        self.assertEqual((ch["unchanged"], ch["added"], ch["removed"]), (1, [], []))
        self.assertEqual(X.changes(self.PROP, [WALK])["removed"], ["hire"])

    def test_abandoned(self):
        self.assertEqual(X.classify(row(reply([])), self.PROP, TYPES)[:2], ("abandoned", None))

    def test_invalid(self):
        cases = [(row("not json"), "invalid_json"), (row(json.dumps({"actions": []})), "invalid_decision"),
                 (row(json.dumps({"reason": "", "actions": [], "details": ["x"]})), "invalid_decision"),
                 (row(reply([{"type": "fly"}])), "unknown_action"), (row(None), "no_content"),
                 (row("x", status=None, error="timeout: T"), "request_failed"),
                 (row("x", status=500), "request_failed"), (row("x", parsed=False), "request_failed")]
        for r, why in cases:
            self.assertEqual(X.classify(r, self.PROP, TYPES), ("invalid", why, None), why)


class Continuation(unittest.TestCase):
    def setUp(self):
        self.run, self.pairs, self.pts = recorded_run()
        self.orig = X.open_session
        X.open_session = fake_open(self.run, self.pairs)

    def tearDown(self):
        X.open_session = self.orig

    def evidence(self, acts):
        return X.decision_evidence("dt", "d", context(digests=self.pts[1]), acts)

    def test_validation_and_apply_agree(self):
        x = self.evidence([WALK, HIRE])
        self.assertEqual(x["outcome"], "accepted")
        self.assertEqual([a["applied"] for a in x["applied"]], [True, True])
        self.assertTrue(x["validateApplyAgree"])
        self.assertNotEqual(x["digestsAfterDecision"], self.pts[1])

    def test_no_action_suffix(self):
        none = self.evidence([])
        self.assertEqual((none["outcome"], none["applied"], none["validateApplyAgree"]), ("no_action", [], None))
        self.assertEqual(none["digestsAfterDecision"], self.pts[1])
        self.assertEqual(none["suffix"]["days"], 2)  # the advances after point 1
        refused = self.evidence([dict(HIRE, offer="bad")])
        self.assertEqual((refused["outcome"], refused["applied"]), ("refused", []))
        self.assertEqual(refused["suffix"]["endDigests"], none["suffix"]["endDigests"])
        acted = self.evidence([HIRE])
        self.assertNotEqual(acted["suffix"]["endDigests"], none["suffix"]["endDigests"])

    def test_financial_commitment_extraction(self):
        x = self.evidence([WALK, HIRE])
        fe = x["financialEvidence"]
        self.assertEqual(fe["affordability"], ["no_new_commitment", "funded_with_credit"])
        self.assertEqual(fe["commitmentCents"], 700)
        self.assertTrue(fe["creditSupported"])
        self.assertEqual(fe["reversibility"], {"reversible_no_recorded_cost": 1,
                                               "reversible_with_retained_obligation": 1})
        refused = self.evidence([dict(HIRE, offer="bad")])["financialEvidence"]
        self.assertEqual((refused["affordability"], refused["commitmentCents"], refused["creditSupported"]),
                         (["not_tested"], 0, False))

    def test_point_not_reached(self):
        with self.assertRaisesRegex(X.ExperimentError, "not reached"):
            X.decision_evidence("dt", "d", context(digests={"log": "nowhere"}), [])

    def test_recorded_answers_checked(self):
        pairs = copy.deepcopy(self.pairs)
        pairs[1][1]["result"]["digests"] = {"log": "changed"}
        X.open_session = fake_open(self.run, pairs)
        with self.assertRaisesRegex(X.ExperimentError, "answered differently"):
            X.decision_evidence("dt", "d", context(digests=self.pts[1]), [])


class LiveClient(unittest.TestCase):
    def setUp(self):
        self.proto, self.psha, self.man, self.contexts = X.load_protocol(os.path.join(V, "protocol.json"))
        self.proto = dict(self.proto, drawsPerArm=2)

    def state(self, version="0.7.0"):
        return lambda base: {"time": "t", "health": {"status": "healthy"}, "serverVersion": version,
                             "openapiSha256": "o", "models": [{"id": "Qwen3.6-35B-A3B-oQ4e-mtp"}]}

    def go(self, post, state=None):
        with tempfile.TemporaryDirectory() as d:
            doc = X.run(self.proto, self.psha, self.man, self.contexts, d, "http://stub", post_fn=post,
                        state_fn=state or self.state(), now=lambda: 0)
            rows = X.load_jsonl(os.path.join(d, "live", "requests.jsonl"))
        return doc, rows

    def test_request_failure_handling(self):
        n = [0]

        def post(base, data, timeout):
            n[0] += 1
            if n[0] in (2, 3):
                return None, "", "timeout: TimeoutError", 300.0
            return 200, json.dumps({"choices": [{"message": {"content": "{}"}, "finish_reason": "stop"}]}), None, 1.0
        doc, rows = self.go(post)
        self.assertEqual((doc["requestsSent"], doc["stopped"]), (32, None))
        self.assertEqual([r["error"] for r in rows[1:3]], ["timeout: TimeoutError"] * 2)
        self.assertEqual(X.classify(rows[1], [], TYPES)[:2], ("invalid", "request_failed"))
        doc, rows = self.go(lambda base, data, timeout: (None, "", "connection: URLError", 0.1))
        self.assertEqual((doc["requestsSent"], doc["stopped"]), (5, "5 consecutive transport failures"))

    def test_sequential_requests_are_the_declared_ones(self):
        seen = []

        def post(base, data, timeout):
            seen.append(X.sha256(data))
            return 200, json.dumps({"choices": [{"message": {"content": "{}"}, "finish_reason": "stop"}]}), None, 1.0
        doc, rows = self.go(post)
        self.assertEqual(seen, [r["requestSha256"] for r in rows])
        by = X.check_rows(self.proto, self.contexts, rows)
        self.assertEqual(sum(len(v) for v in by.values()), 32)
        bad = copy.deepcopy(rows)
        bad[0]["requestSha256"] = "0" * 64
        with self.assertRaisesRegex(X.ExperimentError, "not the declared request"):
            X.check_rows(self.proto, self.contexts, bad)
        with self.assertRaisesRegex(X.ExperimentError, "schedule"):
            X.check_rows(self.proto, self.contexts, rows[1:])

    def test_server_change_stops(self):
        calls = [0]

        def state(base):
            calls[0] += 1
            return self.state("0.7.1" if calls[0] > 2 else "0.7.0")(base)
        doc, rows = self.go(lambda b, d, t: (200, "{}", None, 1.0), state)
        self.assertTrue(doc["stopped"].startswith("server configuration changed"))
        self.assertEqual(len(rows), 8)  # Gemma's first round: 4 points, 2 arms


class Statistics(unittest.TestCase):
    def draws(self, seed=0):
        rnd = random.Random(seed)
        out = []
        for p in ("a", "b"):
            for arm in X.ARMS:
                for d in range(1, 21):
                    o = rnd.choice(X.OUTCOMES)
                    out.append({"point": p, "arm": arm, "draw": d, "outcome": o, "invalidReason": None,
                                "final": "no_action", "actionsKey": None if o == "invalid" else o,
                                "rawSha256": o, "parsedSha256": o, "commitmentChangeCents": 0})
        return out

    def test_intervals(self):
        self.assertEqual(X.wilson(0, 20), [0.0, 0.1611])
        self.assertEqual(X.wilson(20, 20), [0.8389, 1.0])
        self.assertEqual(X.newcombe(10, 20, 10, 20)["difference"], 0.0)
        lo, hi = X.newcombe(15, 20, 5, 20)["interval"]
        self.assertTrue(0 < lo < 0.5 < hi < 1)

    def test_stable_output_ordering(self):
        ctx = [{"id": "a", "model": "gemma4-26b-a4b", "stratum": "accepted"},
               {"id": "b", "model": "qwen36-35b-a3b", "stratum": "refused"}]
        ds = self.draws()
        sh = list(ds)
        random.Random(1).shuffle(sh)
        a = X.summarize(ctx, ds, {"drawsPerArm": 20})
        b = X.summarize(ctx, sh, {"drawsPerArm": 20})
        self.assertEqual(json.dumps(a, sort_keys=True), json.dumps(b, sort_keys=True))
        for r in a["points"]:
            for arm in X.ARMS:
                self.assertEqual(sum(v["count"] for v in r["arms"][arm]["outcomes"].values()), 20)
        self.assertEqual(sum(r["blocked"]["rounds"] for r in a["points"]), 40)

    def test_not_run_draws_stay_in_the_denominator(self):
        ctx = [{"id": "a", "model": "gemma4-26b-a4b", "stratum": "accepted"}]
        ds = [x for x in self.draws() if x["point"] == "a" and x["draw"] <= 5]
        s = X.summarize(ctx, ds, {"drawsPerArm": 20})["points"][0]["arms"]["control"]
        self.assertEqual((s["draws"], s["observed"], s["notRun"]), (20, 5, 15))


class Replay(unittest.TestCase):
    def test_committed_analysis_replays(self):
        """The committed analysis, if any, is made from the recorded
        responses alone: every request rebuilt from the points."""
        p = os.path.join(V, "live", "requests.jsonl")
        if not os.path.exists(p):
            self.skipTest("no live run committed yet")
        proto, _, _, contexts = X.load_protocol(os.path.join(V, "protocol.json"))
        rows = X.load_jsonl(p)
        by = X.check_rows(proto, contexts, rows)
        self.assertEqual(sum(len(v) for v in by.values()), len(rows))


if __name__ == "__main__":
    unittest.main()
