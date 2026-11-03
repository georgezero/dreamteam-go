"""Tests of the V0.12b Stage B driver (stage_b.py), with mock inference
only: llm-agent 6's and llm-agent 5's scripted engines (their tests'
fixtures, whose answers have the real engine's shape) and scripted model
replies. No model and no engine binary; the real engine is exercised by
stage_b_fidelity.py in CI.

    python3 -B -m unittest discover -s analysis/acknowledgment
"""

import copy
import json
import os
import sys
import tempfile
import unittest

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(os.path.dirname(HERE))
sys.path.insert(0, HERE)
sys.path.insert(0, os.path.join(ROOT, "agents", "llm"))
sys.path.insert(0, os.path.join(ROOT, "analysis", "repeatability"))
import stage_b as B  # noqa: E402
import stage_b_fidelity as SF  # noqa: E402
import scripted_cases as SC  # noqa: E402  (V0.12a's cases)
import test_llm_agent6 as T6  # noqa: E402
import test_llm_agent5 as T5  # noqa: E402
import test_workflow_experiment as TW  # noqa: E402  (V0.11b's arm C cases: llm-agent 5)

A3, A5, A6, T4 = B.A3, B.A5, B.A6, T6.T4
FIRE, FIRE2, HIRE, TELE, STAFF, OPS = T6.FIRE, T6.FIRE2, T6.HIRE, T6.TELE, T6.STAFF, T6.OPS
r, ack = T6.reply, T6.ack_of
MODEL = B.QWEN


class FakeSession:
    """An engine session over a scripted engine: the stdio answers' shape."""

    def __init__(self, engine):
        self.engine = engine

    def request(self, rid, op, args=None):
        try:
            return {"protocol": A3.AGENT_PROTOCOL, "id": rid, "ok": True, "result": self.engine.request(op, args)}
        except A3.ProtocolError as e:
            return {"protocol": A3.AGENT_PROTOCOL, "id": rid, "ok": False,
                    "error": {"code": e.code, "message": e.message}}

    def close(self):
        pass


def settings(retries=1):
    s = T4.settings(maxRetries=retries)
    s["requestTimeoutSeconds"] = 300
    return s


def point(engine, proposal, s, pid="fake-p0", validation="accepted"):
    """A branch point on a scripted engine: its digests and the request the
    agent makes there, with `proposal` as the recorded reply."""
    pset = A3.prompt_settings(s)
    _, _, _, _, prompt, _ = A3.prepare(engine, B.FixedMemory(""), 0, pset)
    req = A3.ModelClient(s).payload([{"role": "system", "content": s["systemPrompt"]},
                                     {"role": "user", "content": prompt}])
    acts = json.loads(proposal)["actions"]
    return {"id": pid, "model": MODEL, "point": 0, "digests": engine.request("digests")["digests"],
            "validation": validation, "stageB": {"category": "test"}, "recording": "test",
            "proposal": {"content": proposal, "sha256": B.sha256(proposal), "actions": acts},
            "request": req, "sourceRequest": req}


def entry(e, arm, seq=1):
    return SF.entry_for(e, arm, seq)


def draw(arm, replies, engine_kw=None, retries=1, engine=None, limits=None, transport=None, script_kinds=None):
    """The driver's draw on a scripted engine: replies[0] is the recorded
    proposal; the rest answer the later requests (None: a timeout)."""
    eng = engine or (T6.Engine(**(engine_kw or {})) if arm == "B" else
                     T5.Harness(T4.fixture(), *(engine_kw or ([], []))))
    s = settings(retries)
    e = point(copy.deepcopy(eng), replies[0], s) if engine is None else point(eng, replies[0], s)
    tr = transport or (SF.Script(script_kinds) if script_kinds is not None else Replies(replies[1:]))
    d = B.run_draw(entry(e, arm), e, s, lambda e_, a: FakeSession(eng), B.FixedMemory(""), tr, limits)
    return d, e, tr, eng


class Replies:
    """A transport of fixed reply texts (None: a timeout)."""

    def __init__(self, replies):
        self.replies, self.payloads = list(replies), []

    def __call__(self, payload):
        self.payloads.append(json.loads(json.dumps(payload)))
        x = self.replies.pop(0)
        if x is None:  # the scripted clients' timeout (test_llm_agent4.Client)
            return None, "", "timeout: no answer within 1 s"
        return 200, SF.body_of(payload, x), None


def bare(arm, replies, engine_kw=None, retries=1):
    """The agent itself (its ask(), a plain client, its Evidence)."""
    eng = T6.Engine(**(engine_kw or {})) if arm == "B" else T5.Harness(T4.fixture(), *(engine_kw or ([], [])))
    s = settings(retries)
    client = T4.Client(list(replies))
    with tempfile.TemporaryDirectory() as d:
        ev = A3.Evidence(d)
        if arm == "B":
            b, rec = A6.ask(eng, ev, A6.LiveModel(client, s), s, B.FixedMemory(""), 0)
            rec["workflow"] = "acknowledgment"
        else:
            b, rec = A5.ask(eng, ev, client, s, B.FixedMemory(""), 0, workflow="validation-first-preview")
    return b, rec, [p["messages"] for p in client.payloads], eng


def engine_ops(log):
    return [(op, x) for op, x in log if op != "decide"]


class Fidelity(unittest.TestCase):
    """For the same replies and engine answers, the driver's draw is the
    agent's own run: the same messages, record, batches and engine
    requests (Phase 3)."""

    def same_as_agent(self, arm, replies, engine_kw=None, retries=1):
        b, rec, msgs, beng = bare(arm, replies, engine_kw, retries)
        d, e, tr, deng = draw(arm, replies, engine_kw, retries)
        self.assertEqual(d["status"], "completed")
        self.assertEqual([p["messages"] for p in tr.payloads], msgs[1:])
        self.assertEqual(A3.ModelClient(settings(retries)).payload(msgs[0]), e["sourceRequest"])
        self.assertTrue(A6.same(json.loads(json.dumps(d["record"])), json.loads(json.dumps(rec))))
        self.assertEqual(d["batches"], b)
        # the driver's engine requests: a digests before and after the agent's own
        dlog, blog = engine_ops(deng.log), engine_ops(beng.log)
        self.assertEqual(dlog[1:-1], blog)
        self.assertEqual(len(d["calls"]), len(msgs) - 1)
        return d

    def test_every_v012a_scripted_case(self):
        n = 0
        for name, (replies, kw, retries) in SC.CASES.items():
            if not isinstance(replies[0], str) or "actions" not in json.loads(replies[0]):
                continue  # a recorded proposal is always a proposal
            with self.subTest(name):
                self.same_as_agent("B", replies, kw, retries)
                n += 1
        self.assertEqual(n, len(SC.CASES))

    def test_v011b_arm_c_cases_for_arm_a(self):
        d_ = TW.d
        cases = [([d_("100%", [TW.STAFF]), d_("keep", [TW.STAFF])], ["ok", "ok"], [TW.C]),
                 ([d_("100%", [TW.STAFF]), d_("63%", [TW.STAFF_LESS])], ["ok", "ok"], [TW.C]),
                 ([d_("100%", [TW.STAFF]), d_("none", [])], ["ok"], [TW.C]),
                 ([d_("100%", [TW.STAFF]), d_("63%", [TW.STAFF_LESS]), d_("keep", [TW.STAFF_LESS])],
                  ["insufficient_funds", "ok", "ok"], [TW.C]),
                 ([d_("both", [TW.OPS, TW.STAFF]), d_("both", [TW.STAFF, TW.OPS])], ["ok", "ok", "ok", "ok"], [TW.C]),
                 ([d_("100%", [TW.STAFF])], ["ok"], [TW.N]),
                 ([d_("100%", [TW.STAFF])], ["ok"], ["fail"]),
                 ([d_("100%", [TW.STAFF]), None, d_("keep", [TW.STAFF])], ["ok", "ok"], [TW.C])]
        for replies, answers, pv in cases:
            with self.subTest(answers):
                self.same_as_agent("A", replies, (answers, pv), retries=1)

    def test_driver_engine_requests_are_the_agents_plus_two_digests(self):
        d, _, _, eng = draw("B", [r("Riley.", [FIRE]), r("Ack.", [FIRE], ack([FIRE]))])
        ops = [x["op"] for x in d["engine"]]
        self.assertEqual(ops[0], "digests")
        self.assertIn("preview", ops)
        self.assertEqual(d["digestsAfter"], d["initialDigests"])


class Outcomes(unittest.TestCase):
    """Phase 9, 1–14: the outcome of each kind of draw."""

    def cls(self, replies, kw=None, retries=1, arm="B"):
        d, e, _, _ = draw(arm, replies, kw, retries)
        return B.classify(d, e), d

    def test_01_02_accepted_acknowledgment_proceeded(self):
        x, d = self.cls([r("Riley.", [FIRE]), r("Ack.", [FIRE], ack([FIRE]))])
        self.assertEqual((x["class"], x["outcome"], x["submitted"]), ("A_proceeded", "kept", True))
        self.assertEqual((x["acknowledgment"]["accepted"], x["acknowledgment"]["accurate"]), (1, 1))
        self.assertTrue(x["acknowledgment"]["finalCheck"])
        self.assertEqual(x["modelCalls"], 1)

    def test_kept_whatever_the_recorded_key_order(self):
        # the Stage B contexts file stores the recorded actions with their keys
        # sorted; an unchanged proposal is still kept
        d, e, _, _ = draw("B", [r("Riley.", [FIRE]), r("Ack.", [FIRE], ack([FIRE]))])
        e = copy.deepcopy(e)
        e["proposal"]["actions"] = json.loads(json.dumps(e["proposal"]["actions"], sort_keys=True))
        self.assertNotEqual(list(e["proposal"]["actions"][0]), list(FIRE))
        self.assertEqual(B.classify(d, e)["outcome"], "kept")

    def test_03_acknowledged_and_revised(self):
        x, _ = self.cls([r("Both.", [FIRE, FIRE2]), r("Riley only.", [FIRE]), r("Ack.", [FIRE], ack([FIRE]))], retries=1)
        self.assertEqual((x["class"], x["outcome"]), ("B_revised", "revised"))
        self.assertFalse(x["accurateBeforeChange"])
        # an accurate acknowledgment of the first proposal, with changed actions
        x, _ = self.cls([r("Riley.", [FIRE]), r("Sam instead.", [FIRE2], ack([FIRE])), r("Ack.", [FIRE2], ack([FIRE2]))],
                        retries=2)
        self.assertEqual(x["class"], "B_revised")
        self.assertTrue(x["accurateBeforeChange"])

    def test_04_acknowledged_and_abandoned(self):
        x, d = self.cls([r("Riley.", [FIRE]), r("No.", [])])
        self.assertEqual((x["class"], x["outcome"], x["submitted"], x["noAction"]), ("C_abandoned", "abandoned", False, True))

    def test_05_failed_acknowledgment(self):
        bad = SC.changed([FIRE], item=0, path=("consequences", "severance", "severanceCents"), value=1)
        x, d = self.cls([r("Riley.", [FIRE]), r("Ack.", [FIRE], bad), r("Ack.", [FIRE], bad)])
        self.assertEqual((x["class"], x["outcome"], x["submitted"]), ("D_withheld", "withheld", False))
        self.assertEqual(x["acknowledgment"]["failureCodes"], ["wrong_value"])
        self.assertEqual(d["batches"], [])

    def test_06_no_acknowledgment_required(self):
        x, d = self.cls([r("Raise.", [STAFF])])
        self.assertEqual((x["class"], x["outcome"], x["modelCalls"]), ("E_not_required", "kept", 0))
        self.assertEqual(d["batches"], [[STAFF]])

    def test_07_missing_consequence_field(self):
        # the engine's: a termination missing from the preview (fail-closed)
        x, d = self.cls([r("Riley.", [FIRE])], {"preview": "orphan"})
        self.assertEqual((x["class"], x["acknowledgment"]["withheld"], x["modelCalls"]),
                         ("D_withheld", "consequences_unavailable", 0))
        # the model's: an item without its consequences
        no_c = ack([FIRE])
        del no_c["items"][0]["consequences"]
        x, _ = self.cls([r("Riley.", [FIRE]), r("Ack.", [FIRE], no_c), r("Ack.", [FIRE], no_c)])
        self.assertEqual((x["class"], x["acknowledgment"]["failureCodes"]), ("D_withheld", ["malformed_entry"]))

    def test_08_stale_acknowledgment(self):
        x, _ = self.cls([r("Riley.", [FIRE]), r("Sam.", [FIRE2]), r("Ack.", [FIRE2], ack([FIRE])),
                         r("Ack.", [FIRE2], ack([FIRE]))], retries=2)
        self.assertIn("stale_proposal", x["acknowledgment"]["failureCodes"])
        self.assertEqual(x["class"], "D_withheld")

    def test_09_wrong_acknowledgment_value(self):
        bad = SC.changed([FIRE], item=0, path=("consequences", "termination", "noticeDays"), value=19)
        x, _ = self.cls([r("Riley.", [FIRE]), r("Ack.", [FIRE], bad), r("Ack.", [FIRE], ack([FIRE]))])
        self.assertEqual((x["class"], x["acknowledgment"]["failureCodes"], x["retries"]),
                         ("A_proceeded", ["wrong_value"], 1))

    def test_10_refused_proposal(self):
        for arm, kw in (("B", {"refuse": {"terminate": "notice_too_short"}}), ("A", (["notice_too_short"] * 2, []))):
            with self.subTest(arm):
                x, d = self.cls([r("Riley.", [FIRE]), r("Riley.", [FIRE])], kw, arm=arm)
                self.assertEqual((x["outcome"], x["submitted"], x["inputs"]), ("refused", False, ["validation"]))
                if arm == "B":
                    self.assertEqual(x["class"], "E_not_required")

    def test_11_corrected_proposal(self):
        eng = T6.Engine(refuse={"hire": "lead_too_short"})
        d, e, _, _ = draw("B", [r("Hire.", [HIRE, FIRE]), r("Riley only.", [FIRE]), r("Ack.", [FIRE], ack([FIRE]))],
                          engine=eng, retries=2)
        x = B.classify(d, e)
        self.assertEqual((x["outcome"], x["class"]), ("revised", "A_proceeded"))
        self.assertEqual(x["inputs"], ["validation", "acknowledgment"])

    def test_12_retry_exhausted(self):
        replies, kw, _ = SC.CASES["retries_exhausted"]
        x, d = self.cls(replies[:3], kw, retries=1)
        self.assertEqual((x["class"], x["stopped"]), ("D_withheld", "retries"))

    def test_13_invalid_model_output(self):
        x, _ = self.cls([r("Riley.", [FIRE]), "garbage", "more garbage"])
        self.assertEqual((x["class"], x["outcome"]), ("D_withheld", "withheld"))
        x, _ = self.cls([r("Riley.", [FIRE]), "garbage"], {"refuse": {"terminate": "x"}})
        self.assertEqual(x["outcome"], "invalid")

    def test_14_model_timeout(self):
        x, d = self.cls([r("Riley.", [FIRE]), None, r("Ack.", [FIRE], ack([FIRE]))])
        self.assertEqual((x["class"], x["inputs"], x["modelCalls"]), ("A_proceeded", ["acknowledgment", "resend"], 2))
        self.assertIsNone(d["calls"][0]["status"])
        x, _ = self.cls([r("Riley.", [FIRE]), None, None])
        self.assertEqual(x["class"], "D_withheld")

    def test_arm_a_preview_classes(self):
        cases = [([TW.d("x", [STAFF]), TW.d("keep", [STAFF])], "proceeded"),
                 ([TW.d("x", [STAFF]), TW.d("none", [])], "abandoned"),
                 ([TW.d("x", [STAFF]), TW.d("less", [TW.STAFF_LESS])], "revised"),
                 ([TW.d("x", [STAFF]), "garbage", "garbage"], "failed")]
        for replies, want in cases:
            with self.subTest(want):
                d, e, _, _ = draw("A", replies, (["ok", "ok"], [TW.C]))
                self.assertEqual(B.classify(d, e)["preview"]["class"], want)


class Lenient(SF.Script):
    """A script whose acknowledgment kinds, with no request to answer, keep
    the actions instead."""

    def __call__(self, payload):
        if self.kinds and self.kinds[0].startswith("ack") and SF.request_items(payload["messages"])[0] is None:
            self.kinds[0] = "keep"
        return super().__call__(payload)


class CallBound(unittest.TestCase):
    """At most 2 model calls per draw, whatever the replies (maxRetries 1):
    the protocol's limit of 4 is never the binding one."""

    def test_every_reply_sequence(self):
        kinds = ["ack", "ack_wrong", "ack_stale", "ack_missing", "abandon", "garbage", "timeout", "keep", "details"]
        worst = 0
        starts = [("B", [FIRE], {}), ("B", [FIRE, FIRE2], {}), ("B", [FIRE], {"refuse": {"terminate": "x"}}),
                  ("A", [STAFF], (["ok"] * 4, [TW.C] * 2)), ("A", [STAFF], (["x", "ok", "ok"], [TW.C] * 2))]
        s = settings()
        for arm, acts, kw in starts:
            def eng():
                return T6.Engine(**kw) if arm == "B" else T5.Harness(T4.fixture(), *kw)
            e = point(eng(), r("P.", acts), s)
            for a in kinds:
                for b in kinds:
                    for c in kinds:
                        if arm == "A" and any(k.startswith("ack") for k in (a, b, c)):
                            continue
                        x = eng()
                        d = B.run_draw(entry(e, arm), e, s, lambda e_, a_: FakeSession(x), B.FixedMemory(""),
                                       Lenient([a, b, c]), None)
                        worst = max(worst, len(d["calls"]))
        self.assertEqual(worst, 2)


class Limits(unittest.TestCase):
    """Phase 9, 15–17: the inference budget, the wall clock, a partial run."""

    def test_15_call_budget(self):
        lim = B.Limits(4, 1, 14400, 300, now=lambda: 0.0)
        d, _, _, _ = draw("B", [r("Riley.", [FIRE]), "garbage", r("Ack.", [FIRE], ack([FIRE]))], limits=lim)
        self.assertEqual((d["status"], d["halt"]["code"], len(d["calls"]), d["record"]),
                         ("interrupted", "call_budget", 1, None))

    def test_per_draw_limit(self):
        lim = B.Limits(1, 100, 14400, 300, now=lambda: 0.0)
        d, _, _, _ = draw("B", [r("Riley.", [FIRE]), "garbage", r("Ack.", [FIRE], ack([FIRE]))], limits=lim)
        self.assertEqual(d["halt"]["code"], "draw_call_limit")

    def test_16_wall_clock(self):
        t = [0.0]
        lim = B.Limits(4, 100, 14400, 300, now=lambda: t[0])

        def slow(payload):
            t[0] += 14200
            return 200, SF.body_of(payload, "garbage"), None
        d, _, _, _ = draw("B", [r("Riley.", [FIRE])], limits=lim, transport=slow)
        self.assertEqual((d["status"], d["halt"]["code"], len(d["calls"])), ("interrupted", "wall_clock", 1))
        with self.assertRaises(B.Halt):
            lim.before_draw()


def tiny(tmp, n_points=2, draws=2, engines=None):
    """A protocol of n_points scripted points, its points and schedule
    files, and its Points stand-in."""
    s = settings()
    es = []
    engines = engines or [T6.Engine() for _ in range(n_points)]
    props = [r("Riley.", [FIRE]), r("Raise.", [STAFF])]
    for k in range(n_points):
        e = point(engines[k], props[k % 2], s, pid=f"fake-p{k}")
        e["model"] = MODEL
        es.append(e)
    man = {"selected": [{"id": e["id"], "model": MODEL} for e in es], "contextsFileSha256": "test"}
    proto = {"id": "test", "modelOrder": [MODEL], "drawsPerArm": {MODEL: draws}, "orderRule": "dreamteam-v12b-order/1",
             "draws": n_points * 2 * draws, "maxModelCallsPerDraw": 4, "maxModelCalls": 1000,
             "stop": {"wallClockSeconds": 14400, "consecutiveTransportFailures": 5}, "request": {"timeoutSeconds": 300},
             "endpointEnv": "X"}
    sched = B.schedule(proto, man, es)
    path = os.path.join(tmp, "protocol.json")
    with open(path, "w") as f:
        json.dump(proto, f)
    B.write_json(os.path.join(tmp, "schedule.json"), {"schedule": sched})

    class Pts:
        by_id = {e["id"]: e for e in es}
        settings = {MODEL: s}
        mem = {e["id"]: "" for e in es}

        def opener(self, binary):
            eng = {e["id"]: engines[k] for k, e in enumerate(es)}
            return B.Pool(lambda e, arm: FakeSession(eng[e["id"]]))
    return proto, man, es, path, Pts(), sched


def healthy(base):
    return {"time": "t", "health": {"status": "healthy"}, "serverVersion": "v", "openapiSha256": "o", "models": []}


class Runs(unittest.TestCase):
    def go(self, tmp, transport, now=None, auth="test", **kw):
        proto, man, es, path, pts, sched = tiny(tmp, **kw)
        out = os.path.join(tmp, "out")
        doc = B.run(proto, "sha", man, es, out, None, None, auth, lambda st: transport, {}, state_fn=healthy,
                    now=now or (lambda: 0.0), utc=lambda: "t", lock_check=False, proto_path=path, pts=pts,
                    client_info={"python": "test"})
        rows = B.load_jsonl(os.path.join(out, "live", "draws.jsonl"))
        return doc, rows, sched, pts, out

    def test_complete_run(self):
        with tempfile.TemporaryDirectory() as tmp:
            doc, rows, sched, pts, out = self.go(tmp, SF.mock_transport("t"))
            self.assertEqual((doc["status"], doc["drawsCompleted"], len(rows)), ("COMPLETE", 8, 8))
            B.check_rows(sched, rows)
            self.assertEqual([x["seq"] for x in rows], list(range(1, 9)))

    def test_17_partial_run_and_no_resume(self):
        with tempfile.TemporaryDirectory() as tmp:
            t = [0.0]

            def tr(payload):
                t[0] += 5000
                return 200, SF.body_of(payload, "garbage"), None
            doc, rows, sched, pts, out = self.go(tmp, tr, now=lambda: t[0])
            self.assertEqual(doc["status"], "PARTIAL")
            self.assertEqual(doc["stopped"]["code"], "wall_clock")
            self.assertLess(doc["drawsCompleted"], 8)
            self.assertEqual(doc["drawsNotRun"] + doc["drawsCompleted"] + (1 if doc["drawInterrupted"] else 0), 8)
            B.check_rows(sched, rows)
            summ = B.summarize({"drawsPerArm": {MODEL: 2}}, list(pts.by_id.values()), [], complete=False)
            self.assertIsNone(summ["pooled"])
            self.assertIsNone(summ["points"][0]["BminusA"])
            with self.assertRaises(B.ExperimentError):  # never resumed into the same directory
                B.run({}, "sha", {}, [], out, None, None, "x", None, {}, proto_path="x")

    def test_authorization_required(self):
        with tempfile.TemporaryDirectory() as tmp:
            with self.assertRaises(B.ExperimentError):
                self.go(tmp, SF.mock_transport("t"), auth=" ")

    def test_15_budget_stops_the_run(self):
        with tempfile.TemporaryDirectory() as tmp:
            proto, man, es, path, pts, sched = tiny(tmp)
            proto["maxModelCalls"] = 1
            doc = B.run(proto, "sha", man, es, os.path.join(tmp, "o"), None, None, "t",
                        lambda st: SF.mock_transport("t"), {}, state_fn=healthy, now=lambda: 0.0, utc=lambda: "t",
                        lock_check=False, proto_path=path, pts=pts)
            self.assertEqual((doc["status"], doc["stopped"]["code"]), ("PARTIAL", "call_budget"))

    def test_transport_failures_stop(self):
        with tempfile.TemporaryDirectory() as tmp:
            doc, rows, _, _, _ = self.go(tmp, lambda p: (None, "", "connection: refused"))
            self.assertEqual((doc["status"], doc["stopped"]["code"]), ("PARTIAL", "transport"))

    def test_unhealthy_server_stops_before_any_draw(self):
        with tempfile.TemporaryDirectory() as tmp:
            proto, man, es, path, pts, sched = tiny(tmp)
            doc = B.run(proto, "sha", man, es, os.path.join(tmp, "o"), None, None, "t", lambda st: None, {},
                        state_fn=lambda b: {"health": {"status": "down"}}, now=lambda: 0.0, utc=lambda: "t",
                        lock_check=False, proto_path=path, pts=pts)
            self.assertEqual((doc["status"], doc["drawsCompleted"]), ("PARTIAL", 0))


class Recorded(unittest.TestCase):
    """Phase 9, 18–25: the recorded draws against the schedule; replay."""

    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        doc, self.rows, self.sched, self.pts, _ = Runs.go(None, self.tmp.name, SF.mock_transport("t"))

    def tearDown(self):
        self.tmp.cleanup()

    def bad(self, rows):
        with self.assertRaises(B.ExperimentError):
            B.check_rows(self.sched, rows)

    def test_18_missing_draw(self):
        self.bad(self.rows[:2] + self.rows[3:])

    def test_19_duplicate_draw(self):
        self.bad(self.rows[:3] + [self.rows[2]] + self.rows[3:])

    def test_unlisted_draw(self):
        self.bad(self.rows + [dict(self.rows[-1], seq=99)])

    def test_20_incorrect_arm(self):
        x = copy.deepcopy(self.rows)
        x[0]["arm"] = "A" if x[0]["arm"] == "B" else "B"
        self.bad(x)
        e = self.pts.by_id[self.rows[0]["point"]]
        with self.assertRaises(B.ExperimentError):
            B.run_draw(dict(self.sched[0], arm="C"), e, settings(), lambda *a: None, B.FixedMemory(""), None, None)

    def test_21_incorrect_initial_proposal(self):
        e = copy.deepcopy(self.pts.by_id[self.sched[0]["point"]])
        e["proposal"]["content"] = r("Sam.", [FIRE2])
        with self.assertRaises(B.ExperimentError):
            B.run_draw(self.sched[0], e, settings(), lambda *a: FakeSession(T6.Engine()), B.FixedMemory(""), None, None)
        x = copy.deepcopy(self.rows)
        x[0]["expected"]["proposalSha256"] = "0" * 64
        self.bad(x)
        # the agent's first request is not the recorded one (another prompt)
        e = copy.deepcopy(self.pts.by_id[self.sched[0]["point"]])
        with self.assertRaises(B.ExperimentError):
            B.run_draw(self.sched[0], e, settings(), lambda *a: FakeSession(T6.Engine()), B.FixedMemory("other memory"),
                       None, None)

    def test_22_incorrect_state(self):
        s = copy.deepcopy(self.sched[0])
        s["expected"]["digestsSha256"] = "0" * 64
        e = self.pts.by_id[s["point"]]
        with self.assertRaises(B.ExperimentError):
            B.run_draw(s, e, settings(), lambda *a: FakeSession(T6.Engine()), B.FixedMemory(""), None, None)
        eng = T6.Engine()
        eng.mutated = True
        with self.assertRaises(B.ExperimentError):
            B.run_draw(self.sched[0], e, settings(), lambda *a: FakeSession(eng), B.FixedMemory(""), None, None)

    def replay(self, row):
        e = self.pts.by_id[row["point"]]
        return B.replay_draw(row, e, settings(), lambda *a: FakeSession(T6.Engine()), B.FixedMemory(""))

    def test_23_tampered_recording(self):
        row = next(x for x in self.rows if x["calls"])
        t = copy.deepcopy(row)
        t["calls"][0]["body"] += " "
        with self.assertRaises(B.ExperimentError):
            self.replay(t)
        t = copy.deepcopy(row)
        t["record"]["outcome"] = "fallback"
        with self.assertRaises(B.ExperimentError):
            self.replay(t)
        t = copy.deepcopy(row)
        t["engine"][0]["answer"]["result"]["digests"]["state"] = "0" * 64
        with self.assertRaises(B.ExperimentError):
            self.replay(t)
        t = copy.deepcopy(row)
        t["calls"].append(dict(t["calls"][-1]))
        with self.assertRaises(B.ExperimentError):
            self.replay(t)

    def test_24_model_free_replay(self):
        for row in self.rows:
            again = self.replay(row)
            self.assertEqual(B.deterministic(again), B.deterministic(row))

    def test_25_deterministic(self):
        with tempfile.TemporaryDirectory() as tmp:
            _, rows2, _, _, _ = Runs.go(None, tmp, SF.mock_transport("t"))
        self.assertEqual([B.deterministic(x) for x in rows2], [B.deterministic(x) for x in self.rows])
        self.assertEqual(B.sha256(B.canon([B.deterministic(x) for x in rows2])),
                         B.sha256(B.canon([B.deterministic(x) for x in self.rows])))


class Schedule(unittest.TestCase):
    def test_committed_schedule(self):
        p = os.path.join(ROOT, "docs", "benchmark", "v12b-dev", "protocol.json")
        proto, sha, man, contexts = B.load_protocol(p)
        sched = B.schedule(proto, man, contexts)
        self.assertEqual(len(sched), 420)
        self.assertEqual(sum(1 for x in sched if x["arm"] == "A"), 210)
        for x in sched:
            self.assertEqual(set(x), {"seq", "model", "point", "arm", "draw", "expected"})
            self.assertEqual(set(x["expected"]), {"digestsSha256", "requestSha256", "proposalSha256"})
        B.check_schedule(proto, sched, os.path.join(os.path.dirname(p), "schedule.json"))

    def test_source_request_keeps_the_clients_key_order(self):
        p = os.path.join(ROOT, "docs", "benchmark", "v12b-dev", "protocol.json")
        _, _, _, contexts = B.load_protocol(p)
        for e in contexts:
            self.assertEqual(list(e["sourceRequest"]), ["model", "messages", "temperature", "max_tokens",
                                                        "response_format"])


if __name__ == "__main__":
    unittest.main()
