"""Tests of the V0.10c experiment (feedback_experiment.py,
shared-prefix/v3): the three arms' messages, the selection rule, the
schedule, the live client, the replay checks and the summaries. No model
and no engine binary are used.

    python3 -B -m unittest discover -s analysis/repeatability
"""

import copy
import json
import os
import random
import sys
import tempfile
import unittest

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(os.path.dirname(HERE))
sys.path.insert(0, HERE)
import feedback_experiment as F  # noqa: E402
import test_decision_experiment as T2  # noqa: E402  (its scripted engine and recorded run)

X, A3, A4, S = F.X, F.A3, F.A4, F.S
V = os.path.join(ROOT, "docs", "benchmark", "v10c-dev")
HIRE, WALK = T2.HIRE, T2.WALK
VALID = {"valid": False, "results": [{"ok": False, "code": "insufficient_funds", "message": "m", "action": HIRE}]}
PREVIEW = {"preview": {"asOf": "t", "ok": False, "results": VALID["results"],
                       "funding": {"commitmentCents": 5, "headroomCents": 1, "funded": False, "basis": "b"}}}


def row(model, validation, point, rec="r1", key=None, stake=0, credit=False, rev=None, displaced=0,
        refusals=(), shown=False):
    return {"model": model, "validation": validation, "point": point, "recording": f"x/staffing-shortage-credit-{rec}-s1-decisions.jsonl",
            "contextKey": key or f"{rec}{point}", "eligible": True, "shownPreview": shown,
            "refusals": list(refusals), "displacedBookings": displaced,
            "previewFunding": {"commitmentCents": stake},
            "proposal": {"commitmentCents": stake if validation == "accepted" else 0, "creditSupported": credit,
                         "reversibility": rev or {}, "directExposure": {}, "affordability": []}}


class Arms(unittest.TestCase):
    def setUp(self):
        self.acts = [WALK, HIRE]
        self.t = F.arm_texts(self.acts, VALID, PREVIEW)

    def test_shared_final_request(self):
        for a in F.ARMS:
            self.assertTrue(self.t[a].endswith(X.FINAL_REQUEST), a)
        self.assertEqual(self.t["neutral"], X.CONTROL_TEXT)

    def test_validation_is_the_engine_answer(self):
        v = self.t["validation"]
        self.assertIn(A3.compact(VALID), v)
        self.assertTrue(v.startswith("VALIDATION. Nothing has been submitted yet."))
        self.assertIn("(actions 1)", v)
        self.assertIn(F.VALIDATION_OPS_NOTE, v)
        self.assertNotIn("preview", v.lower())
        self.assertNotIn(F.VALIDATION_OPS_NOTE, F.arm_texts([HIRE], VALID, PREVIEW)["validation"])

    def test_preview_is_llm_agent_4s_message(self):
        self.assertEqual(self.t["preview"], A4.review_text(PREVIEW, [1], True))

    def test_arms_differ_only_in_the_last_message(self):
        req = {"model": "m", "messages": [{"role": "system", "content": "s"}, {"role": "user", "content": "p"}],
               "temperature": 0, "max_tokens": 4096, "response_format": {"type": "json_object"}}
        bs = [X.arm_request(req, "prop", self.t[a]) for a in F.ARMS]
        for b in bs[1:]:
            self.assertEqual(b["messages"][:-1], bs[0]["messages"][:-1])
            self.assertEqual({k: v for k, v in b.items() if k != "messages"},
                             {k: v for k, v in bs[0].items() if k != "messages"})


class Selection(unittest.TestCase):
    def rows(self):
        G, Q = F.GEMMA, F.QWEN
        return [row(G, "accepted", 1, "G"), row(G, "refused", 2, "G"), row(G, "refused", 3, "G", key="G2x"),
                row(Q, "accepted", 1, "a", stake=500, credit=True),
                row(Q, "accepted", 2, "a", stake=900, credit=True),
                row(Q, "accepted", 3, "a", stake=800, rev={"reversible_with_retained_obligation": 1}),
                row(Q, "accepted", 4, "b", stake=100, rev={"withdrawable": 1}),
                row(Q, "accepted", 5, "c", stake=50, rev={"not_reversible": 1}),
                row(Q, "accepted", 6, "d", stake=999),
                row(Q, "refused", 1, "e", stake=10, refusals=["hire:invalid_schema"]),
                row(Q, "refused", 2, "f", stake=5, refusals=["hire:insufficient_funds"]),
                row(Q, "refused", 3, "g", stake=7, refusals=["hire:insufficient_funds"]),
                row(Q, "refused", 4, "h", stake=900, refusals=["set_roster:notice_too_short"]),
                row(Q, "refused", 5, "i", stake=1, refusals=["hire:invalid_schema"])]

    def test_rule(self):
        got = F.select(self.rows())
        g = [(r["validation"], r["point"]) for r in got if r["model"] == F.GEMMA]
        # Gemma: every distinct context, whatever the recording cap.
        self.assertEqual(sorted(g), [("accepted", 1), ("refused", 2), ("refused", 3)])
        q = [(r["validation"], r["recording"].split("credit-")[1][0], r["point"]) for r in got if r["model"] == F.QWEN]
        acc = [x for x in q if x[0] == "accepted"]
        ref = [x for x in q if x[0] == "refused"]
        # credit_supported: the larger stake (a2); retained_obligation: a3; withdrawable: b4;
        # not_reversible: c5. Recording a gives at most two.
        self.assertEqual(acc, [("accepted", "a", 2), ("accepted", "a", 3), ("accepted", "b", 4), ("accepted", "c", 5)])
        # Insufficient funds first (by stake), then by stake.
        self.assertEqual(ref, [("refused", "g", 3), ("refused", "f", 2), ("refused", "h", 4), ("refused", "e", 1)])

    def test_duplicate_and_cap(self):
        rows = self.rows()
        rows.append(row(F.GEMMA, "accepted", 9, "z", key="G1"))  # the same context as Gemma's point 1
        got = F.select(rows)
        self.assertEqual(sum(1 for r in got if r["contextKey"] == "G1"), 1)
        self.assertLessEqual(max(sum(1 for r in got if r["recording"] == x["recording"] and r["model"] == F.QWEN)
                                 for x in got), F.MAX_PER_RECORDING)

    def test_quota_not_met(self):
        rows = [r for r in self.rows() if not (r["model"] == F.QWEN and r["validation"] == "refused"
                                               and r["point"] > 2)]
        with self.assertRaisesRegex(F.ExperimentError, "2 contexts can be taken, 4 declared"):
            F.select(rows)

    def test_tags(self):
        r = row(F.QWEN, "accepted", 1, credit=True, rev={"withdrawable": 2, "not_reversible": 1}, displaced=3)
        self.assertEqual(F.tags(r), ["credit_supported", "withdrawable", "not_reversible", "displaces_bookings"])
        self.assertEqual(F.tags(row(F.QWEN, "refused", 1, refusals=["hire:insufficient_funds"])),
                         ["insufficient_funds"])


def proto(draws=None):
    return {"orderRule": F.ORDER_RULE, "modelOrder": [F.GEMMA, F.QWEN], "drawsPerArm": draws or dict(F.DRAWS),
            "stop": {"consecutiveTransportFailures": 5, "wallClockSeconds": 14400}, "request": {"timeoutSeconds": 300},
            "id": "t", "endpointEnv": "E"}


def contexts():
    out = []
    for i, m in enumerate([F.GEMMA, F.GEMMA, F.QWEN, F.QWEN, F.QWEN]):
        e = T2.context((HIRE,))
        e["id"], e["model"] = f"p{i}", m
        texts = F.arm_texts([HIRE], VALID, PREVIEW)
        e["arms"] = {a: {"text": t, "textSha256": F.sha256(t), "textChars": len(t),
                         "requestSha256": F.sha256(X.body_bytes(X.arm_request(e["request"], e["proposal"]["content"], t)))}
                     for a, t in texts.items()}
        out.append(e)
    return out


class Schedule(unittest.TestCase):
    def test_reproducible_balanced_sequential(self):
        p, cs = proto(), contexts()
        a, b = F.schedule(p, cs), F.schedule(p, copy.deepcopy(cs))
        self.assertEqual(a, b)
        self.assertEqual(len(a), 2 * 3 * 10 + 3 * 3 * 20)
        for m, n in F.DRAWS.items():
            ids = {e["id"] for e in cs if e["model"] == m}
            for d in range(1, n + 1):
                self.assertEqual(sorted((r["point"], r["arm"]) for r in a if r["model"] == m and r["draw"] == d),
                                 sorted((i, x) for i in ids for x in F.ARMS))
        self.assertEqual([r["model"] for r in a][:60], [F.GEMMA] * 60)

    def test_order_varies_between_rounds(self):
        a = F.schedule(proto(), contexts())
        firsts = {tuple((r["point"], r["arm"]) for r in a if r["model"] == F.QWEN and r["draw"] == d)
                  for d in range(1, 21)}
        self.assertGreater(len(firsts), 15)


class LiveClient(unittest.TestCase):
    def state(self, version="0.7.0"):
        return lambda base: {"time": "t", "health": {"status": "healthy"}, "serverVersion": version,
                             "openapiSha256": "o", "models": []}

    def go(self, post, state=None, draws=None):
        p, cs = proto(draws or {F.GEMMA: 1, F.QWEN: 1}), contexts()
        with tempfile.TemporaryDirectory() as d:
            doc = F.run(p, "sha", {"contextsFileSha256": "c"}, cs, d, "http://stub", post_fn=post,
                        state_fn=state or self.state(), now=lambda: 0)
            rows = F.load_jsonl(os.path.join(d, "live", "requests.jsonl"))
        return p, cs, doc, rows

    def ok(self, base, data, timeout):
        return 200, json.dumps({"choices": [{"message": {"content": T2.reply([])}, "finish_reason": "stop"}]}), None, 1.0

    def test_failures_recorded_and_stop(self):
        _, _, doc, rows = self.go(lambda b, d, t: (None, "", "connection: URLError", 0.1))
        self.assertEqual((doc["requestsSent"], doc["stopped"]), (5, "5 consecutive transport failures"))
        self.assertTrue(all(r["error"] for r in rows))

    def test_replay_checks(self):
        p, cs, doc, rows = self.go(self.ok)
        self.assertEqual(doc["requestsSent"], 15)
        by = F.check_rows(p, cs, rows)
        self.assertEqual(sum(len(v) for v in by.values()), 15)
        bad = copy.deepcopy(rows)
        bad[0]["body"] = bad[0]["body"].replace("stop", "length")
        with self.assertRaisesRegex(F.ExperimentError, "response is not the recorded one"):
            F.check_rows(p, cs, bad)
        bad = copy.deepcopy(rows)
        bad[1]["requestSha256"] = "0" * 64
        with self.assertRaisesRegex(F.ExperimentError, "not the declared request"):
            F.check_rows(p, cs, bad)
        bad = copy.deepcopy(rows)
        bad[2]["content"] = "x"
        with self.assertRaisesRegex(F.ExperimentError, "content"):
            F.check_rows(p, cs, bad)

    def test_server_change_stops(self):
        n = [0]

        def state(base):
            n[0] += 1
            return self.state("0.7.1" if n[0] > 2 else "0.7.0")(base)
        _, _, doc, rows = self.go(self.ok, state, draws={F.GEMMA: 2, F.QWEN: 2})
        self.assertTrue(doc["stopped"].startswith("server configuration changed"))
        self.assertEqual(len(rows), 6)  # Gemma's first round: 2 points, 3 arms


class Summary(unittest.TestCase):
    def draws(self, cs, seed=0):
        rnd, out = random.Random(seed), []
        for e in cs:
            for a in F.ARMS:
                for d in range(1, F.DRAWS[e["model"]] + 1):
                    o = rnd.choice(F.OUTCOMES)
                    out.append({"point": e["id"], "arm": a, "draw": d, "outcome": o, "invalidReason": None,
                                "final": "invalid" if o == "invalid" else "accepted",
                                "actionsKey": None if o == "invalid" else o, "rawSha256": o, "parsedSha256": o,
                                "commitmentCents": None if o == "invalid" else 100,
                                "distinctions": {"proposalChanged": o != "kept", "becameValid": False,
                                                 "commitmentChanged": False, "exposureChanged": False}})
        return out

    def test_per_point_and_primary(self):
        cs = [dict(e, validation="refused") for e in contexts()]
        ds = self.draws(cs)
        s = F.summarize(cs, ds, proto())
        for r in s["points"]:
            self.assertEqual(set(r["comparisons"]), {"preview-validation", "validation-neutral", "preview-neutral"})
            for a in F.ARMS:
                self.assertEqual(sum(v["count"] for v in r["arms"][a]["outcomes"].values()), r["draws"])
            self.assertEqual(r["draws"], F.DRAWS[r["model"]])
        self.assertIn(f"{F.QWEN}/refused", s["pooled"])
        sh = list(ds)
        random.Random(3).shuffle(sh)
        self.assertEqual(json.dumps(s, sort_keys=True), json.dumps(F.summarize(cs, sh, proto()), sort_keys=True))

    def test_not_run_in_denominator(self):
        cs = [dict(contexts()[0], validation="accepted")]
        ds = [x for x in self.draws(cs) if x["draw"] <= 3]
        a = F.summarize(cs, ds, proto())["points"][0]["arms"]["preview"]
        self.assertEqual((a["draws"], a["observed"], a["notRun"]), (10, 3, 7))


class Committed(unittest.TestCase):
    def test_committed_points(self):
        if not os.path.exists(os.path.join(V, "points", "manifest.json")):
            self.skipTest("no points committed yet")
        before = T2.recordings_digest()
        m, cs = F.load_points(os.path.join(V, "points"))
        with open(os.path.join(V, "points", "inventory.json"), encoding="utf-8") as f:
            inv = json.load(f)
        self.assertEqual(m["inventorySha256"], F.sha256(F.canon(inv)))
        self.assertEqual([r["id"] for r in F.select(copy.deepcopy(inv["contexts"]))], [e["id"] for e in cs])
        for e in cs:
            rec = S.recording_point(ROOT, e["recording"], e["point"])
            F.check_context(e, rec, e["validate"]["answer"], e["preview"]["answer"])
            self.assertEqual(e["digests"], rec["digests"])
            self.assertEqual(e["validate"]["answer"]["results"], e["preview"]["answer"]["preview"]["results"])
        self.assertEqual(before, T2.recordings_digest())

    def test_committed_responses_replay(self):
        p = os.path.join(V, "live", "requests.jsonl")
        if not os.path.exists(p):
            self.skipTest("no live run committed yet")
        pr, _, _, cs = F.load_protocol(os.path.join(V, "protocol.json"))
        rows = F.load_jsonl(p)
        self.assertEqual(sum(len(v) for v in F.check_rows(pr, cs, rows).values()), len(rows))


if __name__ == "__main__":
    unittest.main()
