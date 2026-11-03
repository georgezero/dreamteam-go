"""Unit tests of llm-agent 6, high-impact action acknowledgment
(llm-agent-acknowledgment-workflow/v1, dreamteam.action-acknowledgment/v1):
scripted model replies and a scripted engine whose preview answers have the
real engine's shape. No model and no engine.

    python3 -B -m unittest discover -s agents/llm -p 'test_llm_agent6*.py'

The exchange with the real engine and the mock model server is tested by
internal/bench/llm_agent6_test.go.
"""

import copy
import hashlib
import json
import os
import tempfile
import unittest

import llm_agent3 as A3
import llm_agent5 as A5
import llm_agent6 as A6
import test_llm_agent4 as T4

HERE = os.path.dirname(os.path.abspath(__file__))
OPS = T4.OPS
STAFF = T4.STAFF  # a salary allocation change: not high-impact
FIRE = {"type": "terminate", "employee": "E03", "lastDay": "2026-11-02"}
FIRE2 = {"type": "terminate", "employee": "E04", "lastDay": "2026-11-02"}
HIRE = {"type": "hire", "requestId": "h-1", "role": "technologist", "skills": ["XR"], "tier": "standard",
        "startDate": "2026-11-10"}
TELE = {"type": "contract_teleradiology", "op": "start", "plan": "standard-v1", "effective": "2026-11-03",
        "scope": "all", "windows": ["weeknights"], "requestId": "tr-1"}
ROSTER = {"type": "set_roster", "op": "set", "employee": "E05", "effective": "2026-11-02", "roster": {}}
SEVERANCE = {"E03": 360000, "E04": 395000}
NAMES = {"E03": "Riley", "E04": "Sam"}


def engine_preview(actions, displaced=0):
    """A valid staffing batch's preview, in the engine's shape and order
    (internal/sim/preview.go): hires, then terminations, then rosters and
    teleradiology; the guaranteed lines in that order; displacements for
    the whole batch."""
    p = {"asOf": "2026-10-13T07:00:00", "ok": True,
         "results": [{"id": "", "batch": 0, "action": a, "ok": True, "code": "accepted", "message": "accepted"}
                     for a in actions]}
    hires, terms, teles, guar = [], [], [], []
    for a in actions:
        if a["type"] == "hire":
            offer = "XR technologist"
            hires.append({"requestId": a["requestId"], "offer": offer, "tier": "standard", "rateCents": 3600,
                          "leadDays": 28, "startDate": a["startDate"], "productiveDate": "2026-11-24"})
            guar += [{"label": f"Recruiting fee · {offer}", "cents": 300000, "when": "recognized now, paid 2026-11-01",
                      "basis": "fixed offer table; not refundable"},
                     {"label": f"Contract guarantee per full week · {offer}", "cents": 158400, "when": "from 2026-11-10",
                      "basis": "40 h × $36.00 with benefits"},
                     {"label": f"Earliest lawful exit · {offer}", "cents": 633600, "when": "last day 2026-11-24",
                      "basis": "contract guarantee to the earliest last day, plus two contracted weeks' severance"}]
    for a in actions:
        if a["type"] == "terminate":
            e = a["employee"]
            terms.append({"employee": e, "name": NAMES[e], "lastDay": a["lastDay"], "endDate": "2026-11-03",
                          "noticeDays": 20, "severanceCents": SEVERANCE[e], "severancePayday": "2026-11-13",
                          "displaced": displaced})
            guar += [{"label": f"Severance · {NAMES[e]} ({e})", "cents": SEVERANCE[e],
                      "when": "recognized now, paid 2026-11-13", "basis": "two contracted weeks"},
                     {"label": f"Contract pay to the last day · {NAMES[e]} ({e})", "cents": 400000,
                      "when": "today to " + a["lastDay"], "basis": "minimum"}]
    for a in actions:
        if a["type"] == "contract_teleradiology":
            teles.append({"op": a["op"], "plan": a["plan"], "windows": a["windows"], "scope": a["scope"],
                          "effective": a["effective"], "minimumTermEnd": "2026-12-03", "monthlyRetainerCents": 150000,
                          "guaranteedCents": 309333, "guaranteedThrough": "2026-12-03"})
            guar.append({"label": "Teleradiology retainer to the earliest end · new weeknights contract",
                         "cents": 309333, "when": "accrued daily 2026-11-03 to 2026-12-02, paid on the 1st",
                         "basis": "contract: minimum term and notice, even with no reads"})
    for a in actions:
        if a["type"] == "set_reading_allocation":
            guar.append({"label": "Salary allocation change · Dr. Avery (E07)", "cents": 1972602,
                         "when": "per 30 days from 2026-10-27 until changed (14 days' notice)", "basis": "withdrawable"})
    for k, v in (("hires", hires), ("terminations", terms), ("teleradiology", teles), ("guaranteed", guar)):
        if v:
            p[k] = v
    if displaced:
        p["displacements"] = [{"appointment": f"A{n:05d}", "patient": f"P{n:05d}", "exam": "MR Brain w/o",
                               "priority": "routine", "date": "2026-11-09", "start": "16:30", "room": "R4",
                               "reason": "unstaffed"} for n in range(displaced)]
    p["funding"] = {"commitmentCents": 0, "headroomCents": 5000000, "funded": True, "basis": "b"}
    return {"preview": p}


class Engine(T4.Harness):
    """The scripted engine: every batch is accepted unless one of its
    actions is in `refuse` ({type: code}); previews from engine_preview
    with `displaced` bookings. `preview`: "ok", "fail" (a protocol error),
    "orphan" (a termination missing from the answer) or "mutate" (the state
    changes). `final`: a refusal code for every validate after the
    `final_after`-th (a final check that disagrees). The benchmark's turns:
    decide at each point, then end."""

    def __init__(self, displaced=0, refuse=None, preview="ok", final=None, final_after=None, points=1):
        super().__init__(T4.fixture(), [])
        self.displaced, self.refuse, self.preview, self.final, self.final_after = (
            displaced, refuse or {}, preview, final, final_after)
        self.validations = 0
        self.turns = [{"turn": "decide", "point": p} for p in range(points)] + [{"turn": "end"}]
        self.decided = []

    def turn(self):
        return self.turns.pop(0)

    def decide(self, batches):
        self.log.append(("decide", copy.deepcopy(batches)))
        self.decided.append(batches)

    def request(self, op, args=None):
        if op == "hello":
            return {"protocol": A3.AGENT_PROTOCOL, "benchmark": {"decision": A3.POLICY_PROTOCOL,
                                                                 "permittedOps": list(A5.NEEDS)}}
        if op == "validate":
            self.log.append((op, copy.deepcopy(args["actions"])))
            self.validations += 1
            acts = args["actions"]
            code = next((self.refuse[a["type"]] for a in acts if a["type"] in self.refuse), None)
            if code is None and self.final and self.validations > self.final_after:
                code = self.final
            if code is None:
                return {"valid": True, "results": [{"ok": True, "action": a} for a in acts]}
            return {"valid": False, "results": [{"ok": False, "code": code, "message": f"{code}: refused",
                                                 "action": a} for a in acts]}
        if op == "preview":
            self.log.append((op, copy.deepcopy(args["actions"])))
            if self.preview == "fail":
                raise A3.ProtocolError("preview", {"code": "internal", "message": "preview unavailable"})
            ans = engine_preview(args["actions"], self.displaced)
            if self.preview == "orphan":
                ans["preview"].pop("terminations", None)
            if self.preview == "mutate":
                self.mutated = True
            return ans
        return super().request(op, args)


def items_of(actions, displaced=0):
    staff = [a for a in actions if a["type"] in A3.STAFFING]
    idx = [i for i, a in enumerate(actions) if a["type"] in A3.STAFFING]
    items, problems = A6.consequences(staff, idx, engine_preview(staff, displaced))
    assert not problems, problems
    return items


def ack_of(actions, displaced=0):
    """The acknowledgment that matches a proposal exactly."""
    items = items_of(actions, displaced)
    return {"proposal": A6.payload_of(actions, items)["token"], "items": [A6.shown(i) for i in items]}


def reply(reason, actions, ack=None, **kw):
    d = {"reason": reason, "actions": actions}
    if ack is not None:
        d["acknowledgment"] = ack
    d.update(kw)
    return json.dumps(d)


def settings(**kw):
    return T4.settings(**kw)


def run(replies, engine=None, retries=2, points=1):
    """The policy loop (llm_agent6.run), live: (records, engine, client,
    decisions.jsonl as written)."""
    h = engine or Engine(points=points)
    s = settings(maxRetries=retries)
    client = T4.Client(list(replies) * points)
    mem = A3.Memory(s["memoryDecisions"], s["memoryMetricPoints"], s["memoryMaxChars"])
    records = []
    with tempfile.TemporaryDirectory() as d:
        ev = A3.Evidence(d)
        for _ in A6.run(h, ev, client, s, mem, None, records):
            pass
        with open(os.path.join(d, "decisions.jsonl"), encoding="utf-8") as f:
            written = [json.loads(l) for l in f]
    return written, h, client


def replay(written, engine):
    """The same loop with the recordings in place of the model."""
    with tempfile.TemporaryDirectory() as d:
        p = os.path.join(d, "decisions.jsonl")
        with open(p, "w", encoding="utf-8") as f:
            for r in written:
                f.write(json.dumps(r) + "\n")
        rp = A3.Replayer(A6.load_recordings([p]))
        records = []
        ev = A3.Evidence(os.path.join(d, "ev"))
        for _ in A6.run(engine, ev, None, None, None, rp, records):
            pass
    return records, engine


def ops_of(h):
    return [op for op, _ in h.log]


class Base(unittest.TestCase):
    def one(self, replies, engine=None, retries=2):
        written, h, client = run(replies, engine, retries)
        self.assertEqual(len(written), 1)
        r = written[0]
        # No discarded attempt: every model call is an attempt.
        self.assertEqual(len(client.payloads), len(r["attempts"]))
        # Validation, previews and acknowledgment change no state, and nothing
        # but read-only requests precede the decision.
        self.assertTrue(r["digestsBeforeSubmit"]["unchanged"] or (engine and engine.preview == "mutate"))
        self.assertFalse({op for op in ops_of(h)} - {"hello", "digests", "observe", "actions", "validate", "preview",
                                                      "decide"})
        # The record replays without the model.
        if not (engine and engine.preview == "mutate"):
            again, _ = replay(written, self.fresh(engine))
            self.assertEqual({k: v for k, v in again[0].items() if k not in ("mode", "replayed", "replayChecked")},
                             {k: v for k, v in r.items() if k != "mode"})
        return r, h, client

    @staticmethod
    def fresh(engine):
        if engine is None:
            return Engine()
        e = Engine(engine.displaced, engine.refuse, engine.preview, engine.final, engine.final_after)
        return e

    def submitted(self, h):
        return [x for op, x in h.log if op == "decide"][-1]

    def failures(self, at):
        return [f["code"] for f in at["acknowledgment"]["match"]["failures"]]


class Ordinary(Base):
    def test_01_ordinary_valid_action(self):
        r, h, client = self.one([reply("Raise Dr. Avery.", [STAFF])])
        self.assertEqual(r["outcome"], "actions")
        self.assertEqual(r["attempts"][0]["assessment"]["status"], "ordinary")
        self.assertEqual(r["attempts"][0]["assessment"]["items"], [])
        self.assertEqual(len(client.payloads), 1)
        self.assertEqual(self.submitted(h), [[STAFF]])
        self.assertIsNone(r["withheld"])
        # Operations only: no preview, nothing to acknowledge.
        r, h, _ = self.one([reply("Walk-ins.", [OPS])])
        self.assertEqual(r["attempts"][0]["assessment"]["status"], "ordinary")
        self.assertNotIn("preview", ops_of(h))
        self.assertEqual(self.submitted(h), [[OPS]])
        # No action at all.
        r, h, _ = self.one([reply("Nothing.", [])])
        self.assertEqual(r["outcome"], "no_action")


class Acknowledged(Base):
    def test_02_termination_acknowledged(self):
        r, h, client = self.one([reply("Terminate Riley.", [FIRE]),
                                 reply("Acknowledged.", [FIRE], ack_of([FIRE]))])
        a1, a2 = r["attempts"]
        self.assertEqual(a1["assessment"]["status"], "qualifying")
        it = a1["assessment"]["items"][0]
        self.assertEqual((it["id"], it["action"], it["type"], it["target"]), ("C1", 0, "terminate", "E03"))
        self.assertEqual(it["consequences"]["termination"], {"lastDay": "2026-11-02", "noticeDays": 20})
        self.assertEqual(a2["input"]["kind"], "acknowledgment")
        self.assertEqual(a2["turn"], "acknowledge")
        self.assertTrue(client.payloads[1]["messages"][-1]["content"].startswith("ACKNOWLEDGMENT REQUIRED."))
        self.assertTrue(a2["acknowledgment"]["match"]["accepted"])
        self.assertTrue(a2["finalCheck"]["agrees"])
        self.assertTrue(a2["used"])
        self.assertEqual(self.submitted(h), [[FIRE]])
        self.assertEqual(r["stages"]["acknowledged"], ["0.2"])
        self.assertEqual(r["stages"]["acknowledgmentRequested"], ["0.1"])

    def test_03_termination_not_acknowledged(self):
        r, h, _ = self.one([reply("Terminate Riley.", [FIRE]), reply("Do it.", [FIRE]), reply("Do it.", [FIRE])],
                           retries=1)
        self.assertEqual(r["outcome"], "fallback")
        self.assertEqual(self.submitted(h), [])
        self.assertEqual(self.failures(r["attempts"][1]), ["missing"])
        self.assertEqual(r["attempts"][2]["input"]["kind"], "acknowledgment_rejected")
        w = r["withheld"]
        self.assertEqual((w["reason"], w["attempt"]), ("acknowledgment_not_accepted", 1))
        self.assertEqual(w["batches"][0]["actions"], [FIRE])
        self.assertTrue(w["batches"][0]["qualifying"])

    def test_04_severance_acknowledged(self):
        ack = ack_of([FIRE])
        self.assertEqual(ack["items"][0]["consequences"]["severance"],
                         {"severanceCents": 360000, "severancePayday": "2026-11-13"})
        r, h, _ = self.one([reply("Terminate Riley.", [FIRE]), reply("Acknowledged.", [FIRE], ack)])
        self.assertTrue(r["attempts"][1]["acknowledgment"]["match"]["accepted"])
        self.assertEqual(r["attempts"][0]["assessment"]["items"][0]["source"], "preview.terminations[0]")

    def test_05_displaced_bookings_acknowledged(self):
        e = Engine(displaced=3)
        ack = ack_of([FIRE], 3)
        self.assertEqual([i["type"] for i in ack["items"]], ["terminate", "staffing_batch"])
        self.assertEqual(ack["items"][1], {"id": "C2", "actions": [0], "type": "staffing_batch",
                                           "target": "staffing batch",
                                           "consequences": {"displaced_bookings": {"count": 3}}})
        r, h, _ = self.one([reply("Terminate Riley.", [FIRE]), reply("Acknowledged.", [FIRE], ack)], e)
        self.assertEqual(self.submitted(h), [[FIRE]])
        # A roster change that displaces bookings qualifies through its batch.
        e = Engine(displaced=2)
        r, h, _ = self.one([reply("Change E05's roster.", [ROSTER]),
                            reply("Acknowledged.", [ROSTER], ack_of([ROSTER], 2))], e)
        self.assertEqual([i["type"] for i in r["attempts"][0]["assessment"]["items"]], ["staffing_batch"])
        self.assertEqual(self.submitted(h), [[ROSTER]])

    def test_06_retained_obligation_acknowledged(self):
        for act, want in ((HIRE, {"retained_obligation": {"recruitingFeeCents": 300000, "earliestExitCents": 633600}}),
                          (TELE, {"retained_obligation": {"guaranteedCents": 309333, "guaranteedThrough": "2026-12-03",
                                                          "minimumTermEnd": "2026-12-03"}})):
            with self.subTest(act["type"]):
                r, h, _ = self.one([reply("Commit.", [act]), reply("Acknowledged.", [act], ack_of([act]))])
                self.assertEqual(r["attempts"][0]["assessment"]["items"][0]["consequences"], want)
                self.assertEqual(self.submitted(h), [[act]])

    def test_07_multiple_qualifying_actions(self):
        acts = [HIRE, FIRE, FIRE2]
        e = Engine(displaced=4)
        ack = ack_of(acts, 4)
        self.assertEqual([(i["id"], i["type"], i["target"]) for i in ack["items"]],
                         [("C1", "hire", "h-1"), ("C2", "terminate", "E03"), ("C3", "terminate", "E04"),
                          ("C4", "staffing_batch", "staffing batch")])
        r, h, _ = self.one([reply("Restructure.", acts), reply("Acknowledged.", acts, ack)], e)
        self.assertEqual(self.submitted(h), [acts])

    def test_08_one_acknowledgment_missing(self):
        acts = [FIRE, FIRE2]
        ack = ack_of(acts)
        ack["items"] = ack["items"][:1]
        r, h, _ = self.one([reply("Two.", acts), reply("Ack.", acts, ack), reply("Ack.", acts, ack)], retries=1)
        self.assertEqual(self.failures(r["attempts"][1]), ["missing_item"])
        self.assertEqual(r["attempts"][1]["acknowledgment"]["match"]["failures"][0]["item"], "C2")
        self.assertEqual(self.submitted(h), [])


class Mismatch(Base):
    def rejected(self, acts, ack, engine=None):
        r, h, _ = self.one([reply("Propose.", acts), reply("Ack.", acts, ack)], engine, retries=0)
        self.assertEqual(self.submitted(h), [])
        self.assertEqual(r["withheld"]["reason"], "acknowledgment_not_accepted")
        return r["attempts"][1]["acknowledgment"]["match"]["failures"]

    def test_09_wrong_employee(self):
        ack = ack_of([FIRE])
        ack["items"][0]["target"] = "E04"
        self.assertEqual([f["code"] for f in self.rejected([FIRE], ack)], ["wrong_target"])

    def test_10_wrong_action_type(self):
        ack = ack_of([FIRE])
        ack["items"][0]["type"] = "hire"
        self.assertEqual([f["code"] for f in self.rejected([FIRE], ack)], ["wrong_type"])
        ack = ack_of([FIRE])
        ack["items"][0]["action"] = 1
        self.assertEqual([f["code"] for f in self.rejected([FIRE], ack)], ["wrong_action"])

    def test_11_wrong_severance_amount(self):
        for bad in (350000, 360000.0, "360000", True):
            with self.subTest(bad):
                ack = ack_of([FIRE])
                ack["items"][0]["consequences"]["severance"]["severanceCents"] = bad
                f = self.rejected([FIRE], ack)
                self.assertEqual([(x["code"], x["field"], x["expected"]) for x in f],
                                 [("wrong_value", "severance.severanceCents", 360000)])
        ack = ack_of([FIRE])
        del ack["items"][0]["consequences"]["severance"]
        self.assertEqual([f["code"] for f in self.rejected([FIRE], ack)], ["wrong_consequence_types"])

    def test_12_wrong_displaced_booking_count(self):
        ack = ack_of([FIRE], 3)
        ack["items"][1]["consequences"]["displaced_bookings"]["count"] = 2
        f = self.rejected([FIRE], ack, Engine(displaced=3))
        self.assertEqual([(x["code"], x["item"], x["got"]) for x in f], [("wrong_value", "C2", 2)])

    def test_15_generic_acknowledgment_rejected(self):
        for ack in ("I understand.", {"understood": True}, {"proposal": "x", "items": "all"}, [], True):
            with self.subTest(ack):
                self.assertEqual([f["code"] for f in self.rejected([FIRE], ack)], ["malformed"])
        # Agreement in prose, with no acknowledgment, is not one.
        r, h, _ = self.one([reply("Terminate Riley.", [FIRE]),
                            reply("I understand the severance and accept the consequences.", [FIRE])], retries=0)
        self.assertEqual(self.failures(r["attempts"][1]), ["missing"])
        self.assertEqual(self.submitted(h), [])
        # Extra keys or an extra item are not accepted either.
        ack = ack_of([FIRE])
        ack["items"][0]["approved"] = True
        self.assertEqual([f["code"] for f in self.rejected([FIRE], ack)], ["malformed_entry"])
        ack = ack_of([FIRE])
        ack["items"].append(dict(ack["items"][0], id="C9"))
        self.assertEqual([f["code"] for f in self.rejected([FIRE], ack)], ["unknown_item"])
        ack = ack_of([FIRE])
        ack["items"].append(ack["items"][0])
        self.assertEqual([f["code"] for f in self.rejected([FIRE], ack)], ["duplicate_item"])


class Proposals(Base):
    def test_13_stale_acknowledgment(self):
        # Riley is proposed, then Sam: an acknowledgment of Riley's proposal
        # given for Sam's is stale.
        stale = ack_of([FIRE])
        r, h, _ = self.one([reply("Riley.", [FIRE]), reply("Sam instead.", [FIRE2]),
                            reply("Ack.", [FIRE2], stale), reply("Ack.", [FIRE2], stale)], retries=2)
        codes = self.failures(r["attempts"][2])
        self.assertEqual(codes[0], "stale_proposal")
        self.assertIn("wrong_target", codes)
        self.assertEqual(self.submitted(h), [])
        self.assertEqual(r["withheld"]["batches"][0]["actions"], [FIRE2])

    def test_14_revised_action_after_acknowledgment(self):
        # A correct acknowledgment of Riley's termination, sent with Sam's:
        # the proposal changed, so it is not accepted; Sam's is assessed and
        # acknowledged on its own.
        r, h, client = self.one([reply("Riley.", [FIRE]), reply("Sam instead.", [FIRE2], ack_of([FIRE])),
                                 reply("Ack.", [FIRE2], ack_of([FIRE2]))])
        a2 = r["attempts"][1]
        self.assertEqual(self.failures(a2), ["proposal_changed"])
        self.assertEqual(a2["assessment"]["status"], "qualifying")
        self.assertNotEqual(a2["assessment"]["payload"], r["attempts"][0]["assessment"]["payload"])
        self.assertTrue(r["attempts"][2]["acknowledgment"]["match"]["accepted"])
        self.assertEqual(self.submitted(h), [[FIRE2]])
        self.assertEqual(r["stages"]["acknowledgmentRequested"], ["0.1", "0.2"])

    def test_16_unparseable_acknowledgment(self):
        r, h, client = self.one([reply("Riley.", [FIRE]), "I acknowledge the severance.",
                                 reply("Ack.", [FIRE], ack_of([FIRE]))])
        self.assertEqual(r["attempts"][1]["parse"]["code"], "invalid_json")
        self.assertIsNone(r["attempts"][1]["acknowledgment"])
        self.assertEqual(r["attempts"][2]["input"]["kind"], "retry")
        self.assertTrue(r["attempts"][2]["acknowledgment"]["match"]["accepted"])
        self.assertEqual(self.submitted(h), [[FIRE]])
        # Unparseable to the end: nothing is submitted.
        r, h, _ = self.one([reply("Riley.", [FIRE]), "yes", "yes"], retries=1)
        self.assertEqual(self.submitted(h), [])
        self.assertEqual(r["withheld"]["reason"], "acknowledgment_not_accepted")

    def test_17_model_timeout(self):
        r, h, _ = self.one([reply("Riley.", [FIRE]), None, reply("Ack.", [FIRE], ack_of([FIRE]))])
        self.assertEqual(r["attempts"][1]["transport"], "timeout")
        self.assertEqual(r["attempts"][2]["input"]["kind"], "resend")
        self.assertEqual(r["attempts"][2]["input"]["sha256"], r["attempts"][1]["input"]["sha256"])
        self.assertEqual(self.submitted(h), [[FIRE]])
        r, h, _ = self.one([reply("Riley.", [FIRE]), None, None], retries=1)
        self.assertEqual(self.submitted(h), [])
        self.assertIsNotNone(r["withheld"])

    def test_20_mixed_operations_staffing(self):
        acts = [OPS, FIRE]
        ack = ack_of(acts)
        self.assertEqual(ack["items"][0]["action"], 1)
        r, h, _ = self.one([reply("Walk-ins and Riley.", acts), reply("Ack.", acts, ack)])
        self.assertEqual(self.submitted(h), [[OPS], [FIRE]])
        self.assertEqual(len(r["attempts"][1]["finalCheck"]["validation"]), 2)
        # A failed acknowledgment withholds the qualifying batch, and the
        # operations batch is not submitted on its own.
        r, h, _ = self.one([reply("Walk-ins and Riley.", acts), reply("Ack.", acts)], retries=0)
        self.assertEqual(self.submitted(h), [])
        w = r["withheld"]["batches"]
        self.assertEqual([(b["kind"], b["qualifying"], b["items"]) for b in w],
                         [("operations", False, []), ("staffing", True, ["C1"])])

    def test_21_refused_action(self):
        e = Engine(refuse={"terminate": "invalid_parameters"})
        r, h, _ = self.one([reply("Riley.", [FIRE]), reply("Nothing.", [])], e)
        self.assertTrue(r["attempts"][0]["refusals"])
        self.assertIsNone(r["attempts"][0]["assessment"])
        self.assertNotIn("preview", ops_of(h))
        self.assertEqual(r["attempts"][1]["input"]["kind"], "validation")
        self.assertEqual(r["outcome"], "no_action")

    def test_22_retry_budget_exhausted(self):
        bad = ack_of([FIRE])
        bad["items"][0]["target"] = "E99"
        r, h, _ = self.one([reply("Riley.", [FIRE])] + [reply("Ack.", [FIRE], bad)] * 3, retries=2)
        self.assertEqual(len(r["attempts"]), 4)
        self.assertEqual(self.submitted(h), [])
        self.assertEqual(r["withheld"]["reason"], "acknowledgment_not_accepted")
        # A revision answering the last request, with no retry left.
        r, h, _ = self.one([reply("Riley.", [FIRE]), reply("Sam.", [FIRE2])], retries=0)
        self.assertEqual(r["withheld"]["batches"][0]["actions"], [FIRE2])
        self.assertEqual(self.submitted(h), [])


class FailClosed(Base):
    def test_18_consequence_retrieval_failure(self):
        r, h, _ = self.one([reply("Riley.", [FIRE])], Engine(preview="orphan"))
        self.assertEqual(r["attempts"][0]["assessment"]["status"], "unavailable")
        self.assertEqual(r["completeness"]["stopped"], "consequences_unavailable")
        self.assertEqual(r["withheld"]["reason"], "consequences_unavailable")
        self.assertEqual(self.submitted(h), [])

    def test_19_preview_unavailable(self):
        for acts in ([FIRE], [ROSTER]):
            with self.subTest(acts[0]["type"]):
                r, h, _ = self.one([reply("Change.", acts)], Engine(preview="fail"))
                self.assertEqual(r["attempts"][0]["assessment"]["error"]["code"], "internal")
                self.assertEqual(r["withheld"]["reason"], "consequences_unavailable")
                self.assertEqual(self.submitted(h), [])
        # The state changed during the preview: withheld too.
        written, h, _ = run([reply("Riley.", [FIRE])], Engine(preview="mutate"))
        self.assertEqual(written[0]["withheld"]["reason"], "consequences_unavailable")
        self.assertEqual(self.submitted(h), [])

    def test_23_no_state_mutation(self):
        r, h, _ = self.one([reply("Riley.", [FIRE]), reply("Ack.", [FIRE], ack_of([FIRE]))])
        self.assertEqual(ops_of(h)[-1], "decide")
        self.assertTrue(all(op in ("hello", "digests", "observe", "actions", "validate", "preview")
                            for op in ops_of(h)[:-1]))
        a1, a2 = r["attempts"]
        self.assertTrue(a1["assessment"]["digestsUnchanged"])
        self.assertTrue(a2["finalCheck"]["digestsUnchanged"])
        self.assertTrue(r["digestsBeforeSubmit"]["unchanged"])

    def test_24_final_validation_disagreement(self):
        # The final check's validate answers differ from those acknowledged.
        r, h, _ = self.one([reply("Riley.", [FIRE]), reply("Ack.", [FIRE], ack_of([FIRE]))],
                           Engine(final="insufficient_funds", final_after=1))
        a2 = r["attempts"][1]
        self.assertTrue(a2["acknowledgment"]["match"]["accepted"])
        self.assertFalse(a2["finalCheck"]["agrees"])
        self.assertEqual(r["withheld"]["reason"], "final_check_disagrees")
        self.assertEqual(self.submitted(h), [])
        self.assertEqual(r["stages"]["acknowledged"], ["0.2"])
        self.assertEqual(r["stages"]["validated"], ["0.1"])  # the final validation refused it
        self.assertEqual(r["stages"]["submitted"]["batches"], 0)


class Recording(Base):
    def test_25_complete_recording(self):
        acts = [OPS, FIRE]
        r, h, client = self.one([reply("Walk-ins and Riley.", acts), reply("Ack.", acts, ack_of(acts))])
        self.assertEqual((r["schema"], r["agentVersion"], r["workflow"], r["workflowVersion"], r["contract"]),
                         (A6.RECORD_SCHEMA, "6", "acknowledgment", A6.WORKFLOW_VERSION, A6.CONTRACT))
        a1, a2 = r["attempts"]
        self.assertEqual(a1["actions"], acts)                                        # initial proposal
        self.assertEqual([v["actions"] for v in a1["validation"]], [[OPS], [FIRE]])  # validation requests
        self.assertTrue(all(v["answer"]["valid"] for v in a1["validation"]))         # validation results
        asm = a1["assessment"]
        self.assertEqual(asm["answerSha256"], A3.sha256(A3.compact(asm["answer"])))  # the engine's answer
        self.assertEqual(asm["items"][0]["source"], "preview.terminations[0]")      # consequence source
        self.assertEqual(asm["items"][0]["consequences"]["severance"]["severanceCents"], 360000)  # values
        self.assertEqual(asm["payload"], A6.payload_of(acts, asm["items"]))          # payload hash
        req = client.payloads[1]["messages"][-1]["content"]                          # the request
        self.assertEqual(a2["input"]["sha256"], A3.sha256(req))
        self.assertEqual(req, A6.request_text(asm))
        self.assertEqual(a2["output"]["text"], reply("Ack.", acts, ack_of(acts)))   # the model's acknowledgment
        self.assertEqual(a2["acknowledgment"]["given"], ack_of(acts))               # parsed
        self.assertEqual(a2["acknowledgment"]["match"], {"accepted": True, "failures": []})  # matching
        self.assertEqual([v["answer"]["valid"] for v in a2["finalCheck"]["validation"]], [True, True])  # final
        self.assertEqual(r["batches"], [[OPS], [FIRE]])                              # submitted
        self.assertEqual(r["stages"], {"proposed": ["0.1", "0.2"], "validated": ["0.1", "0.2"], "refused": [],
                                       "previewed": ["0.1"], "acknowledgmentRequested": ["0.1"],
                                       "acknowledged": ["0.2"],
                                       "submitted": {"batches": 2, "actions": 2, "outcome": "actions"}})
        self.assertIn("state", r["digests"])                                          # state digests
        self.assertTrue(r["digestsBeforeSubmit"]["unchanged"])
        self.assertEqual(r["completeness"]["attempts"], 2)

    def test_26_tampered_recording_rejected(self):
        written, _, _ = run([reply("Riley.", [FIRE]), reply("Ack.", [FIRE], ack_of([FIRE]))])

        def tamper(f):
            w = copy.deepcopy(written)
            f(w[0])
            return w

        def text(r, ack):
            o = json.loads(r["attempts"][1]["output"]["text"])
            o["acknowledgment"] = ack
            t = json.dumps(o)
            r["attempts"][1]["output"] = {"sha256": A3.sha256(t), "chars": len(t), "text": t}
        bad_ack = ack_of([FIRE])
        bad_ack["items"][0]["consequences"]["severance"]["severanceCents"] = 1
        cases = {
            "consequence value": lambda r: r["attempts"][0]["assessment"]["items"][0]["consequences"]["severance"]
            .update(severanceCents=1),
            "payload hash": lambda r: r["attempts"][0]["assessment"]["payload"].update(sha256="0" * 64),
            "parsed acknowledgment": lambda r: r["attempts"][1]["acknowledgment"]["given"]["items"][0]
            .update(target="E04"),
            "match result": lambda r: r["attempts"][1]["acknowledgment"]["match"].update(accepted=False),
            "submitted action": lambda r: r["batches"][0][0].update(employee="E04"),
            "output text": lambda r: r["attempts"][1]["output"].update(text="{}"),
            "re-hashed acknowledgment": lambda r: text(r, bad_ack),
            "stage": lambda r: r["stages"].update(acknowledged=[]),
        }
        for name, f in cases.items():
            with self.subTest(name):
                with self.assertRaises(A3.ReplayError):
                    replay(tamper(f), Engine())
        for name, f in {"workflow": lambda r: r.update(workflowSha256="0" * 64),
                        "contract": lambda r: r.update(contractSha256="0" * 64),
                        "selection": lambda r: r.update(workflow="validation-first"),
                        "schema": lambda r: r.update(schema=A5.RECORD_SCHEMA)}.items():
            with self.subTest(name):
                with self.assertRaises(A3.SettingsError):
                    replay(tamper(f), Engine())

    def test_27_model_free_replay(self):
        replies = [reply("Walk-ins and Riley.", [OPS, FIRE]), reply("Ack.", [OPS, FIRE], ack_of([OPS, FIRE], 2))]
        written, h, _ = run(replies, Engine(displaced=2, points=2), points=2)
        self.assertEqual(len(written), 2)
        again, e = replay(written, Engine(displaced=2, points=2))
        self.assertEqual(e.decided, h.decided)
        self.assertEqual([{k: v for k, v in r.items() if k not in ("mode", "replayed", "replayChecked")}
                          for r in again], [{k: v for k, v in r.items() if k != "mode"} for r in written])
        self.assertEqual(again[0]["replayChecked"], {"attempts": 2, "assessments": 1, "acknowledgments": 1})
        # A recording replays only from its own start.
        with self.assertRaises(A3.ReplayError):
            replay(written, Engine(displaced=3, points=2))


def recorded_previews():
    """The real engine's preview answers recorded in V0.11b (live/draws.jsonl):
    (actions, answer), each distinct pair once."""
    path = os.path.join(HERE, "..", "..", "docs", "benchmark", "v11b-dev", "live", "draws.jsonl")
    out, seen = [], set()

    def walk(o):
        if isinstance(o, dict):
            if o.get("op") == "preview" and isinstance(o.get("answer"), dict) and (o.get("args") or {}).get("actions"):
                res = o["answer"].get("result")
                key = json.dumps([o["args"]["actions"], res], sort_keys=True)
                if res and key not in seen:
                    seen.add(key)
                    out.append((o["args"]["actions"], res))
            for v in o.values():
                walk(v)
        elif isinstance(o, list):
            for v in o:
                walk(v)
    with open(path, encoding="utf-8") as f:
        for line in f:
            if '"preview"' in line:
                walk(json.loads(line))
    return out


class RealEngineAnswers(unittest.TestCase):
    """The consequence items read from the real engine's preview answers
    recorded in V0.11b: terminations with displaced bookings, hires (the
    fee and exit derived from the engine's line order), teleradiology
    starts and allocation changes."""

    def test_recorded_previews(self):
        got = {}
        for actions, ans in recorded_previews():
            idx = list(range(len(actions)))
            items, problems = A6.consequences(actions, idx, ans)
            self.assertEqual(problems, [], actions)
            p = ans["preview"]
            for it in items:
                got[it["type"]] = got.get(it["type"], 0) + 1
                if it["type"] == "terminate":
                    t = next(x for x in p["terminations"] if x["employee"] == it["target"])
                    self.assertEqual(it["consequences"]["severance"]["severanceCents"], t["severanceCents"])
                    self.assertEqual(it["consequences"]["termination"]["lastDay"], t["lastDay"])
                elif it["type"] == "hire":
                    h = next(x for x in p["hires"] if x["requestId"] == it["target"])
                    fee = next(g for g in p["guaranteed"] if g["label"] == "Recruiting fee · " + h["offer"])
                    self.assertEqual(it["consequences"]["retained_obligation"]["recruitingFeeCents"], fee["cents"])
                elif it["type"] == "contract_teleradiology":
                    k = int(it["source"].split("[")[1].rstrip("]"))
                    x = p["teleradiology"][k]
                    a = actions[it["action"]]
                    self.assertEqual((x["op"], x["plan"], x["effective"]), ("start", a["plan"], a["effective"]))
                    self.assertEqual(it["consequences"]["retained_obligation"]["guaranteedCents"], x["guaranteedCents"])
                elif it["type"] == "staffing_batch":
                    self.assertEqual(it["consequences"]["displaced_bookings"]["count"], len(p["displacements"]))
                    self.assertEqual(it["actions"], idx)
            if all(a["type"] == "set_reading_allocation" for a in actions):
                self.assertEqual(items, [])  # an allocation change is not high-impact
        self.assertTrue(all(got.get(k) for k in ("terminate", "hire", "contract_teleradiology", "staffing_batch")), got)


class Compatibility(unittest.TestCase):
    def test_28_historical_compatibility(self):
        # llm-agent 5 (the file, as at V0.11.0) and its workflow identities are unchanged.
        with open(os.path.join(HERE, "llm_agent5.py"), "rb") as f:
            self.assertEqual(hashlib.sha256(f.read()).hexdigest(),
                             "f89dd4e1950662c5e5b3d4e1dd48b8ee23b9ed7e3e4080dbb0ed136267c26cb0")
        self.assertEqual(A5.WORKFLOW_SHA256, "3ee7d66f4f10fd93e05a908725c792b16b76d01dc29095360dae9d67c0c09d0a")
        self.assertEqual(A5.VALIDATION_FIRST_SHA256,
                         "a0204f628c7631d43ebfbb48e3b6e31071b67252daa76f141e3ad3a43ff5f184")
        self.assertEqual(A5.DEFAULT_WORKFLOW, "validation-first")
        # Each version refuses the other's records.
        written, _, _ = run([reply("Riley.", [FIRE]), reply("Ack.", [FIRE], ack_of([FIRE]))])
        with tempfile.TemporaryDirectory() as d:
            p6 = os.path.join(d, "v6.jsonl")
            with open(p6, "w") as f:
                f.write(json.dumps(written[0]) + "\n")
            with self.assertRaisesRegex(A3.SettingsError, "version incompatible"):
                A5.load_recordings([p6])
            v5 = dict(written[0], schema=A5.RECORD_SCHEMA, agentVersion="5")
            p5 = os.path.join(d, "v5.jsonl")
            with open(p5, "w") as f:
                f.write(json.dumps(v5) + "\n")
            with self.assertRaisesRegex(A3.SettingsError, "version 5 recordings replay with"):
                A6.load_recordings([p5])

    def test_29_cross_platform_determinism(self):
        # The scripted evidence made again here is byte-identical to the
        # committed files; CI runs this on linux/amd64, linux/arm64 and
        # darwin/arm64, and compares its own copies across them.
        import importlib.util
        root = os.path.dirname(os.path.dirname(HERE))
        spec = importlib.util.spec_from_file_location(
            "ack_scripted_cases", os.path.join(root, "analysis", "acknowledgment", "scripted_cases.py"))
        m = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(m)
        with tempfile.TemporaryDirectory() as d:
            import contextlib
            import io
            with contextlib.redirect_stdout(io.StringIO()):
                m.main(["--out", d])
            for f in ("scripted-cases.json", "scripted-cases.md"):
                with open(os.path.join(d, f), "rb") as a, \
                        open(os.path.join(root, "docs", "benchmark", "v12a-dev", f), "rb") as b:
                    self.assertEqual(a.read(), b.read(), f)

    def test_identities_pinned(self):
        self.assertEqual(A6.CONTRACT, "dreamteam.action-acknowledgment/v1")
        self.assertEqual(A6.CONTRACT_SHA256, CONTRACT_SHA256)
        self.assertEqual(A6.WORKFLOW_VERSION, "llm-agent-acknowledgment-workflow/v1")
        self.assertEqual(A6.WORKFLOW_SHA256, WORKFLOW_SHA256)


CONTRACT_SHA256 = "33c116da979a5a1067a4f19163369a1092394ca9bfe9ec4aade2dab8cf9c4c9b"
WORKFLOW_SHA256 = "b9cb79828f2657de91de69b1fa30019752ab73611d80fb6854c8f7a6488595a6"

if __name__ == "__main__":
    unittest.main()
