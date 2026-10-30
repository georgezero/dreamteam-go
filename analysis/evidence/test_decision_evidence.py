"""Tests of the decision evidence analysis (dreamteam.decision-evidence/v1):
each rule with its negative controls, abstention, older recording formats,
stable identifiers, ordering, and that the analysis reads its inputs
without changing them or using the network.

    python3 -B -m unittest discover -s analysis/evidence

The observation and catalog are the V0.5a live run's decision point 0
(agents/llm/testdata/v05a-point0.json): staff E01 Jordan, E03 Riley,
E05 Casey, E06 Morgan, E07 Dr. Avery, E08 Dr. Blake; cash $150,000; 9
unread studies; Tuesday, closed at 07:00.
"""

import contextlib
import copy
import hashlib
import io
import json
import os
import shutil
import socket
import tempfile
import unittest

import decision_evidence as D

A = D.A
HERE = os.path.dirname(os.path.abspath(__file__))
FIXTURE = os.path.join(HERE, "..", "..", "agents", "llm", "testdata", "v05a-point0.json")
PSET = {k: A.DEFAULTS[k] for k in A.PROMPT_SETTINGS}


def fixture():
    with open(FIXTURE, encoding="utf-8") as f:
        return json.load(f)


FX = fixture()
OBS, CATALOG, OMITTED = FX["observation"], FX["catalog"], FX["omitted"]
STAFF = D.staff_index(OBS)


def stmts(reason):
    return D.extract(reason, STAFF)[0]


def commits(reason):
    return [s for s in stmts(reason) if s["kind"] == "commit"]


def amount(text):
    a = D.amounts(text)
    assert len(a) == 1, a
    return a[0]


class Ctx:
    """A minimal analysis context for one recording."""

    def __init__(self):
        self.ctx = {"recording": {"path": "r.jsonl", "sha256": "0" * 64, "group": "t", "replayConfig": "c.json"},
                    "run": {"policy": "p", "model": "m", "scenario": "s", "seed": 1, "days": 1, "engine": "e",
                            "replayRun": None},
                    "lines": {0: 1}, "sources": lambda rec: [], "completeness": lambda rec: {}}


def record(reason, proposed, outcome=None, validation=None, refusals=None, errors=None, point=0):
    if outcome is None:
        outcome = "actions" if proposed else "no_action"
    if validation is None and proposed:
        validation = [{"valid": True, "results": [{"ok": True, "code": "accepted", "action": a} for a, in
                                                  [(x,) for x in b]]} for b, _ in A.split(proposed)]
    batches = [b for b, _ in A.split(proposed)] if outcome == "actions" else []
    return {"point": point, "time": "2026-10-13T07:00:00", "day": 1, "outcome": outcome, "reason": reason,
            "proposed": proposed, "validation": validation, "refusals": refusals or [], "errors": errors or [],
            "batches": batches, "attempts": 1}


def applied_ok(rec):
    return [{"applied": True, "results": [{"id": f"X{i}", "ok": True, "code": "accepted", "message": "ok"}
                                          for i, _ in enumerate(b)]} for b in rec["batches"]]


def run_checks(rec, applied=None, obs=OBS, catalog=CATALOG, prompt="", verified=False):
    an = D.Analysis()
    c = Ctx().ctx
    ex = D.execution(rec, applied)
    st, _ = D.extract(rec["reason"], D.staff_index(obs)) if rec["reason"] else ([], False)
    positive = [s for s in st if s["kind"] == "commit"]
    for s0 in st:
        for s in D.per_target(s0):
            D.check_statement(an, c, rec, rec["reason"], s, rec["proposed"], ex, catalog, positive, obs)
    if rec["reason"]:
        D.check_amounts(an, c, rec, rec["reason"], obs, catalog, prompt, verified, True)
        D.check_facts(an, c, rec, rec["reason"], obs, prompt, verified, True)
    return an


def found(an, rule, status="supported"):
    return [f for f in an.findings if f["category"] == rule and f["status"] == status]


EXT = lambda emp, minutes=60: {"type": "extend_shift", "employee": emp, "minutes": minutes, "date": "2026-10-13"}
PROMPT = A.build_prompt(0, OBS, OMITTED, CATALOG, "MEMORY", PSET)[0]


class Statements(unittest.TestCase):
    def test_names_resolve_to_staff(self):
        s = commits("I will extend Dr. Avery's shift and Casey's shift to clear the backlog.")
        self.assertEqual(s[0]["type"], "extend_shift")
        self.assertEqual(s[0]["targets"], ["E07"])

    def test_name_list(self):
        s = commits("I will extend Riley and Morgan by 2 hours each to open slots.")
        self.assertEqual(s[0]["targets"], ["E03", "E06"])
        self.assertEqual(len(D.per_target(s[0])), 2)

    def test_coordinated_verb(self):
        s = commits("I will extend Casey's shift today to increase MRI capacity, and extend Morgan's shift to "
                    "increase US capacity.")
        self.assertEqual([x["targets"] for x in s], [["E05"], ["E06"]])

    def test_broad_strategic_language_is_no_statement(self):
        for r in ("We need to reduce costs.", "The goal is to improve turnaround.",
                  "Extending her shift by 60 minutes will allow her to clear the backlog.",
                  "I need to reduce labour costs to avoid insolvency.", "I will monitor the situation.",
                  "I will focus on imaging capacity to reduce future backlog."):
            self.assertEqual(commits(r), [], r)

    def test_ambiguous_intent_is_flagged(self):
        s = commits("If the backlog grows, I will extend Dr. Avery's shift.")
        self.assertIn("hedged", s[0]["flags"])
        s = commits("I will extend Riley or Sam, whichever is on site.")
        self.assertIn("hedged", s[0]["flags"])
        s = commits("I will extend them by 60 minutes.")
        self.assertIn("unresolved_target", s[0]["flags"])
        s = commits("I will extend Dr. Avery's shift tomorrow.")
        self.assertIn("deferred", s[0]["flags"])

    def test_revised_text(self):
        st, revised = D.extract("I will extend Dr. Blake. Wait, he is off today. I will do nothing.", STAFF)
        self.assertTrue(revised)
        self.assertTrue(all("revised_in_text" in s["flags"] for s in st))

    def test_unknown_name_unresolved(self):
        s = commits("I will extend Quinn's shift.")
        self.assertIn("unresolved_target", s[0]["flags"])

    def test_teleradiology_hire_is_not_a_staff_hire(self):
        s = commits("I will hire a teleradiologist for the weekend.")
        self.assertIn("unresolved_target", s[0]["flags"])


class ActionRules(unittest.TestCase):
    def test_explicit_action_omitted(self):
        rec = record("I will extend Dr. Avery's shift by 120 minutes. I will also extend Riley's shift.",
                     [EXT("E07", 120)])
        an = run_checks(rec, applied_ok(rec))
        f = found(an, "EXPLICIT_ACTION_OMITTED")
        self.assertEqual(len(f), 1)
        self.assertIn("E03", f[0]["basis"])
        self.assertTrue(f[0]["reasonSpan"]["text"].startswith("I will also extend Riley"))

    def test_action_present_is_consistent(self):
        rec = record("I will extend Riley's shift by 60 minutes.", [EXT("E03")])
        an = run_checks(rec, applied_ok(rec))
        self.assertEqual(an.findings, [])
        self.assertEqual(an.checks["EXPLICIT_ACTION_OMITTED"]["consistent"], 1)
        self.assertEqual(an.checks["STATED_ACTION_NOT_EXECUTED"]["consistent"], 1)

    def test_similar_but_distinct_target_is_omission_not_match(self):
        rec = record("I will extend Casey's shift.", [EXT("E11")])
        an = run_checks(rec, applied_ok(rec))
        self.assertEqual(len(found(an, "EXPLICIT_ACTION_OMITTED")), 1)

    def test_opposite_direction_allocation(self):
        a = {"type": "set_reading_allocation", "op": "set", "employee": "E07", "siteAllocationBP": 4000,
             "effective": "2026-10-27"}
        rec = record("I will increase Dr. Avery's site allocation to cover the evening gap.", [a])
        an = run_checks(rec, applied_ok(rec))
        f = found(an, "EXPLICIT_OPPOSITE_ACTION")
        self.assertEqual(len(f), 1)
        self.assertEqual(f[0]["observation"]["value"], 6000)
        a2 = dict(a, siteAllocationBP=8000)
        rec2 = record("I will increase Dr. Avery's site allocation to cover the evening gap.", [a2])
        self.assertEqual(found(run_checks(rec2, applied_ok(rec2)), "EXPLICIT_OPPOSITE_ACTION"), [])

    def test_opposite_contract_direction(self):
        a = {"type": "contract_teleradiology", "op": "start", "plan": "standard-v1", "effective": "2026-11-02"}
        rec = record("I will cancel the teleradiology contract T00001 to stop the retainer.", [a])
        f = found(run_checks(rec, applied_ok(rec)), "EXPLICIT_OPPOSITE_ACTION")
        self.assertEqual(len(f), 1)
        self.assertEqual(f[0]["rule"]["subrule"], "contract_direction")

    def test_stated_no_action_with_actions(self):
        rec = record("The backlog will clear. I will take no action now.", [EXT("E07")])
        f = found(run_checks(rec, applied_ok(rec)), "EXPLICIT_OPPOSITE_ACTION")
        self.assertEqual(f[0]["rule"]["subrule"], "no_action_stated")
        # Negative controls: a qualified or scoped no-action statement, or one beside an action.
        for r in ("I will take no further action. ", "I will take no action on staffing.",
                  "I will take no other action. I will extend Dr. Avery's shift."):
            rec = record(r, [EXT("E07")])
            an = run_checks(rec, applied_ok(rec))
            self.assertEqual(found(an, "EXPLICIT_OPPOSITE_ACTION"), [], r)

    def test_negated_action_submitted(self):
        rec = record("I will not extend Blake, Sam, or Alex as they are already at close.", [EXT("E08")])
        f = found(run_checks(rec, applied_ok(rec)), "EXPLICIT_OPPOSITE_ACTION")
        self.assertEqual(f[0]["rule"]["subrule"], "negated_action")
        rec = record("I will not extend Blake.", [EXT("E07")])
        self.assertEqual(found(run_checks(rec, applied_ok(rec)), "EXPLICIT_OPPOSITE_ACTION"), [])

    def test_rejected_action(self):
        a = EXT("E08", 120)
        val = [{"valid": False, "results": [{"ok": False, "code": "not_on_shift", "action": a}]}]
        rec = record("I will extend Dr. Blake's shift by 120 minutes.", [a], outcome="fallback", validation=val,
                     refusals=[{"attempt": 1, "action": 0, "type": "extend_shift", "code": "not_on_shift",
                                "message": "Dr. Blake is not on shift today."}])
        f = found(run_checks(rec), "STATED_ACTION_NOT_EXECUTED")
        self.assertEqual(len(f), 1)
        self.assertIn("refused_at_validate", f[0]["basis"])
        self.assertEqual(found(run_checks(rec), "EXPLICIT_ACTION_OMITTED"), [])

    def test_refused_at_apply(self):
        rec = record("I will extend Dr. Avery's shift.", [EXT("E07")])
        app = [{"applied": True, "results": [{"id": "X1", "ok": False, "code": "after_close", "message": "no"}]}]
        f = found(run_checks(rec, app), "STATED_ACTION_NOT_EXECUTED")
        self.assertIn("refused_at_apply", f[0]["basis"])

    def test_batch_partner_refused(self):
        a, b = {"type": "set_worklist_policy", "policy": "oldest_first"}, EXT("E08")
        val = [{"valid": False, "results": [{"ok": True, "code": "accepted"}, {"ok": False, "code": "not_on_shift"}]}]
        rec = record("I will set the worklist policy to oldest_first.", [a, b], outcome="fallback", validation=val)
        f = found(run_checks(rec), "STATED_ACTION_NOT_EXECUTED")
        self.assertIn("not_submitted_other_refused", f[0]["basis"])

    def test_successfully_executed(self):
        rec = record("I will extend Dr. Avery's shift.", [EXT("E07")])
        an = run_checks(rec, applied_ok(rec))
        self.assertEqual(found(an, "STATED_ACTION_NOT_EXECUTED"), [])
        self.assertEqual(an.checks["STATED_ACTION_NOT_EXECUTED"]["consistent"], 1)

    def test_apply_unknown_is_insufficient(self):
        rec = record("I will extend Dr. Avery's shift.", [EXT("E07")])
        f = found(run_checks(rec, None), "STATED_ACTION_NOT_EXECUTED", "insufficient_evidence")
        self.assertEqual(len(f), 1)

    def test_hedged_omission_is_unresolved(self):
        rec = record("I will extend Dr. Blake's shift if he is on site.", [])
        an = run_checks(rec)
        self.assertEqual(found(an, "EXPLICIT_ACTION_OMITTED"), [])
        self.assertEqual(len(found(an, "EXPLICIT_ACTION_OMITTED", "unresolved")), 1)


class Amounts(unittest.TestCase):
    def test_parse(self):
        a = amount("Cash is healthy ($55.7k).")
        self.assertEqual((a["cents"], a["precisionCents"]), (5570000, 10000))
        a = amount("losing money (-$11k MTD)")
        self.assertEqual((a["cents"], a["negative"]), (-1100000, True))
        a = amount("cash at $62,841")
        self.assertEqual((a["cents"], a["precisionCents"]), (6284100, 100))
        self.assertEqual(amount("about $12,000")["comparator"], "approx")

    def test_bindings(self):
        for text, q in (("Cash is healthy ($150k).", "cash"), ("with cash at $28,917 and", "cash"),
                        ("losing money (-$11k MTD).", "profit_mtd"),
                        ("operating loss month-to-date (-$9,707)", "profit_mtd"),
                        ("payroll due in 4 days ($40,653)", "payroll_due"), ("a $21,336 payroll due", "payroll_due"),
                        ("significant receivables ($235,636)", "receivables"),
                        ("the $3,100/mo teleradiology retainer", "teleradiology_retainer")):
            self.assertEqual(D.bind(amount(text), text)[0], q, text)
        for text in ("saves ~$3,800/week in pay", "this action costs ~$462", "cash dropped from $90,000"):
            self.assertIsNone(D.bind(amount(text), text)[0], text)

    def test_correct_amount(self):
        rec = record("Cash is healthy ($150,000) and no action is needed.", [])
        an = run_checks(rec, prompt=PROMPT, verified=True)
        self.assertEqual(an.findings, [])
        self.assertEqual(an.checks["FINANCIAL_AMOUNT_MISMATCH"]["consistent"], 1)
        for r in ("Cash is healthy ($150k).", "Cash is about $149,000.", "Cash is over $100,000."):
            an = run_checks(record(r, []), prompt=PROMPT, verified=True)
            self.assertEqual(found(an, "FINANCIAL_AMOUNT_MISMATCH"), [], r)

    def test_incorrect_amount(self):
        an = run_checks(record("Cash is critically low ($2,698).", []), prompt=PROMPT, verified=True)
        f = found(an, "FINANCIAL_AMOUNT_MISMATCH")
        self.assertEqual(len(f), 1)
        self.assertEqual(f[0]["rule"]["subrule"], "cash")

    def test_cents_versus_dollars(self):
        an = run_checks(record("Cash is $15,000,000, more than enough.", []), prompt=PROMPT, verified=True)
        f = found(an, "FINANCIAL_AMOUNT_MISMATCH")
        self.assertEqual(f[0]["rule"]["subrule"], "cash:cents_read_as_dollars")

    def test_sign(self):
        mtd = OBS["finance"]["monthToDate"]["operatingProfitCents"]
        self.assertLess(mtd, 0)
        d = abs(mtd) // 100
        for r in (f"a significant loss (${d:,} MTD)", f"losing money (-${d:,} MTD)"):
            self.assertEqual(found(run_checks(record(r, []), prompt=PROMPT, verified=True),
                                   "FINANCIAL_AMOUNT_MISMATCH"), [], r)
        self.assertEqual(len(found(run_checks(record("Cash is -$150,000.", []), prompt=PROMPT, verified=True),
                                   "FINANCIAL_AMOUNT_MISMATCH")), 1)

    def test_unbound_amount_is_unresolved(self):
        an = run_checks(record("This action costs ~$462 but helps.", []), prompt=PROMPT, verified=True)
        self.assertEqual(len(found(an, "FINANCIAL_AMOUNT_MISMATCH", "unresolved")), 1)

    def test_missing_observation(self):
        an = run_checks(record("Cash is critically low ($2,698).", []), prompt=None, verified=False)
        self.assertEqual(len(found(an, "FINANCIAL_AMOUNT_MISMATCH", "insufficient_evidence")), 1)
        an = run_checks(record("There are 30 unread studies.", []), prompt=None, verified=False)
        self.assertEqual(len(found(an, "FACTUAL_OBSERVATION_MISMATCH", "insufficient_evidence")), 1)


class Facts(unittest.TestCase):
    def test_facts_parsed(self):
        f = D.facts("The practice is currently closed (Wednesday 07:00) with a backlog of 14 studies, "
                    "4 overdue and 2 overdue 'soon' reports. Lost demand is stable at 153.")
        self.assertEqual([(x["kind"], x["value"]) for x in f],
                         [("closed", True), ("weekday", "Wed"), ("unread", 14), ("overdue", 4),
                          ("overdue_priority", 2), ("lost_total", 153)])

    def test_subsets_and_past_values_are_not_claims(self):
        self.assertEqual(D.facts("The XR backlog of 6 studies is small."), [])
        self.assertEqual(D.facts("It decreased from 25 unread to 3."), [])
        self.assertEqual([x["value"] for x in D.facts("It decreased from 25 to 3 unread studies.")], [3])

    def test_fact_consistent_and_mismatch(self):
        an = run_checks(record("The practice is closed today (Tuesday) with 9 unread studies.", []),
                        prompt=PROMPT, verified=True)
        self.assertEqual(an.findings, [])
        self.assertEqual(an.checks["FACTUAL_OBSERVATION_MISMATCH"]["consistent"], 3)
        an = run_checks(record("Today (Saturday) there are 30 unread studies.", []), prompt=PROMPT, verified=True)
        f = found(an, "FACTUAL_OBSERVATION_MISMATCH")
        self.assertEqual(sorted(x["rule"]["subrule"] for x in f), ["unread", "weekday"])


class Repeats(unittest.TestCase):
    def ref(self, attempt, msg="Dr. Blake is not on shift on 2026-10-13.", code="not_on_shift", typ="extend_shift"):
        return {"attempt": attempt, "action": 0, "type": typ, "code": code, "message": msg}

    def run_repeats(self, recs):
        an = D.Analysis()
        D.check_repeats(an, Ctx().ctx, recs, {})
        return an

    def test_repeated_identical_refusal(self):
        recs = [record("r", [], outcome="fallback", refusals=[self.ref(1), self.ref(2)], point=0),
                record("r", [], outcome="fallback",
                       refusals=[self.ref(1, "Dr. Blake is not on shift on 2026-10-14.")], point=1)]
        f = found(self.run_repeats(recs), "REPEATED_INVALID_PROPOSAL")
        self.assertEqual(len(f), 1)
        self.assertEqual((f[0]["repeat"]["count"], f[0]["repeat"]["withinPoint"]), (3, 1))
        self.assertEqual(len(f[0]["repeat"]["acrossPoints"]), 1)

    def test_similar_but_distinct_refusals(self):
        recs = [record("r", [], outcome="fallback", refusals=[
            self.ref(1), self.ref(2, "Dr. Avery is not on shift on 2026-10-13."),
            self.ref(2, code="after_close"), self.ref(2, typ="set_roster")])]
        an = self.run_repeats(recs)
        self.assertEqual(found(an, "REPEATED_INVALID_PROPOSAL"), [])
        self.assertEqual(an.checks["REPEATED_INVALID_PROPOSAL"]["consistent"], 4)

    def test_batch_rejected_not_counted(self):
        recs = [record("r", [], outcome="fallback", refusals=[self.ref(1, "", "batch_rejected"),
                                                               self.ref(2, "", "batch_rejected")])]
        self.assertEqual(self.run_repeats(recs).findings, [])

    def test_older_format_refusals_from_errors(self):
        r = {"point": 0, "time": "t", "outcome": "fallback", "reason": "x", "proposed": [],
             "errors": ["attempt 1: refused: action 0: after_close: The shift ends at close.",
                        "attempt 2: refused: action 0: after_close: The shift ends at close."]}
        self.assertEqual(len(D.refusals_of(r)), 2)
        f = found(self.run_repeats([r]), "REPEATED_INVALID_PROPOSAL")
        self.assertEqual(f[0]["repeat"]["count"], 2)
        self.assertIsNone(D.refusals_of({"point": 0}))


class OlderFormats(unittest.TestCase):
    def test_v05a_record(self):
        # llm-agent 1: no memory, refusals, promptSettings or agentVersion.
        path = os.path.join(HERE, "..", "..", "docs", "benchmark", "v05a-dev", "llm-decisions.jsonl")
        with open(path, encoding="utf-8") as f:
            recs = [json.loads(l) for l in f]
        self.assertEqual(D.rebuild_prompts(recs, {}), {})
        for r in recs:
            an = run_checks(dict(r, refusals=r.get("refusals")), None, prompt=None, verified=False)
            for f in an.findings:
                self.assertIn(f["status"], ("unresolved", "insufficient_evidence"))

    def test_missing_reason(self):
        rec = record(None, [EXT("E07")])
        an = run_checks(rec, applied_ok(rec))
        self.assertEqual(an.findings, [])


def write(path, obj, lines=False):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w", encoding="utf-8") as f:
        if lines:
            for o in obj:
                f.write(json.dumps(o, separators=(",", ":")) + "\n")
        else:
            json.dump(obj, f)


def tree_digest(root):
    h = hashlib.sha256()
    for d, _, files in sorted(os.walk(root)):
        for n in sorted(files):
            p = os.path.join(d, n)
            h.update(p.encode())
            with open(p, "rb") as f:
                h.update(f.read())
    return h.hexdigest()


class Corpus(unittest.TestCase):
    """A two-point recording with its replay, built with the agent's own
    functions so that the prompts verify."""

    def setUp(self):
        self.dir = tempfile.mkdtemp()
        d = self.dir
        digests = [dict(FX["clock"]["digests"], run=f"r{i}") for i in range(2)]
        obs = [OBS, copy.deepcopy(OBS)]
        obs[1]["time"], obs[1]["date"], obs[1]["day"] = "2026-10-14T07:00:00", "2026-10-14", 2
        recs = [
            {"point": 0, "time": OBS["time"], "day": 1, "digests": digests[0], "mode": "live", "agentVersion": "2",
             "promptVersion": A.PROMPT_VERSION, "memoryFormat": A.MEMORY_FORMAT, "promptSettings": PSET,
             "attempts": 2, "retries": 1, "detailsRequested": None, "errors": [
                 "attempt 1: refused: action 0: not_on_shift: Dr. Blake is not on shift on 2026-10-13."],
             "refusals": [{"attempt": 1, "action": 0, "type": "extend_shift", "code": "not_on_shift",
                           "message": "Dr. Blake is not on shift on 2026-10-13."}],
             "outcome": "actions", "reason": "Cash is $15,000,000. I will extend Dr. Avery's shift. I will also "
                                             "extend Riley's shift.",
             "proposed": [EXT("E07")], "validation": [{"valid": True, "results": [{"ok": True, "code": "accepted"}]}],
             "batches": [[EXT("E07")]]},
            {"point": 1, "time": obs[1]["time"], "day": 2, "digests": digests[1], "mode": "live", "agentVersion": "2",
             "promptVersion": A.PROMPT_VERSION, "memoryFormat": A.MEMORY_FORMAT, "promptSettings": PSET,
             "attempts": 1, "retries": 0, "detailsRequested": None, "errors": [], "refusals": [
                 {"attempt": 1, "action": 0, "type": "extend_shift", "code": "not_on_shift",
                  "message": "Dr. Blake is not on shift on 2026-10-14."}],
             "outcome": "fallback", "reason": "I will extend Dr. Blake's shift.", "proposed": [EXT("E08")],
             "validation": [{"valid": False, "results": [{"ok": False, "code": "not_on_shift"}]}], "batches": []},
        ]
        mem = A.Memory(PSET["memoryDecisions"], PSET["memoryMetricPoints"], PSET["memoryMaxChars"])
        for r, o in zip(recs, obs):
            mem.observe(r["point"], o)
            mt = mem.render(A.pending(o, CATALOG))
            prompt, _ = A.build_prompt(r["point"], o, OMITTED, CATALOG, mt, PSET)
            r["promptSha256"], r["memorySha256"] = A.sha256(prompt), A.sha256(mt)
            mem.record(r)
        self.rec_path = os.path.join(d, "repo", "rec-decisions.jsonl")
        write(self.rec_path, recs, lines=True)
        cfg = {"external": [{"id": "m1", "command": ["python3", "agent.py", "--replay", self.rec_path]}]}
        self.cfg_path = os.path.join(d, "repo", "replay.json")
        write(self.cfg_path, cfg)
        run = os.path.join(d, "replays", "out1", "runs", "s-1-m1")
        t = [{"id": "h1", "op": "hello"}, {"ok": True, "result": {}}]
        for i, o in enumerate(obs):
            t += [{"id": f"d{i}", "op": "digests"}, {"ok": True, "result": {"digests": digests[i]}},
                  {"id": f"o{i}", "op": "observe"}, {"ok": True, "result": {"observation": o, "omitted": OMITTED}},
                  {"id": f"a{i}", "op": "actions"}, {"ok": True, "result": CATALOG}]
        write(os.path.join(run, "transcript.jsonl"), t, lines=True)
        write(os.path.join(run, "decisions.json"), [
            {"point": 0, "batches": [{"applied": True, "results": [{"id": "X1", "ok": True, "code": "accepted",
                                                                     "message": "Shift extended"}]}]},
            {"point": 1, "batches": []}])
        write(os.path.join(run, "run.json"), {"scenario": {"id": "s"}, "seed": 1, "days": 2, "engine": "e"})
        write(os.path.join(run, "policy-evidence", "decisions.jsonl"),
              [{"point": 0, "replayed": os.path.basename(self.rec_path)}], lines=True)
        self.corpus = os.path.join(d, "repo", "corpus.json")
        write(self.corpus, {"schema": D.CORPUS_SCHEMA, "id": "t", "groups": [
            {"id": "g", "role": "primary", "replays": [{"config": self.cfg_path, "out": "out1"}]}]})

    def tearDown(self):
        shutil.rmtree(self.dir)

    def analyze(self):
        return D.analyze(self.corpus, os.path.join(self.dir, "replays"))

    def test_pipeline(self):
        doc, traces, seen = self.analyze()
        self.assertEqual(seen, [self.rec_path])
        cov = doc["coverage"][0]
        self.assertEqual((cov["points"], cov["promptVerified"], cov["applyResults"]), (2, 2, 1))
        cats = sorted((f["category"], f["status"], f["rule"]["subrule"]) for f in doc["findings"])
        self.assertIn(("EXPLICIT_ACTION_OMITTED", "supported", "extend_shift"), cats)
        self.assertIn(("FINANCIAL_AMOUNT_MISMATCH", "supported", "cash:cents_read_as_dollars"), cats)
        self.assertIn(("STATED_ACTION_NOT_EXECUTED", "supported", "extend_shift"), cats)
        rep = [f for f in doc["findings"] if f["category"] == "REPEATED_INVALID_PROPOSAL"][0]
        self.assertEqual(rep["repeat"]["count"], 2)
        self.assertTrue(rep["repeat"]["acrossPoints"][0]["refusalShownInMemory"])
        self.assertEqual(traces[0]["direct"][0]["id"], "X1")
        self.assertEqual(traces[0]["subsequent"]["nextPoint"], 1)
        self.assertIn("not attributed", traces[0]["subsequent"]["attribution"].replace("not a consequence",
                                                                                         "not attributed"))
        for f in doc["findings"]:
            for k in ("id", "category", "rule", "status", "basis", "recording", "run", "decision", "sources",
                      "reasonSpan", "actions", "execution", "completeness"):
                self.assertIn(k, f)

    def test_stable_identifiers_and_ordering(self):
        a, _, _ = self.analyze()
        b, _, _ = self.analyze()
        self.assertEqual(json.dumps(a, sort_keys=True), json.dumps(b, sort_keys=True))
        ids = [f["id"] for f in a["findings"]]
        self.assertEqual(len(ids), len(set(ids)))
        # The same recording content at another path gives the same identifiers.
        self.assertTrue(all(i.startswith("de1-") and len(i) == 20 for i in ids))
        keys = [(f["recording"]["group"], f["recording"]["path"], f["decision"]["point"] if f["decision"] else -1,
                 f["category"]) for f in a["findings"]]
        self.assertEqual(keys, sorted(keys))

    def test_no_mutation_and_no_network(self):
        before = tree_digest(self.dir)
        real = socket.socket

        def refuse(*a, **k):
            raise AssertionError("the analysis opened a socket")
        socket.socket = refuse
        try:
            out = os.path.join(self.dir, "..", os.path.basename(self.dir) + "-out")
            with contextlib.redirect_stdout(io.StringIO()):
                D.main(["--corpus", self.corpus, "--replays", os.path.join(self.dir, "replays"), "--out", out])
        finally:
            socket.socket = real
        self.assertEqual(tree_digest(self.dir), before)
        with open(os.path.join(out, "evidence.json"), encoding="utf-8") as f:
            doc = json.load(f)
        self.assertEqual(doc["schema"], D.SCHEMA)
        shutil.rmtree(out)

    def test_missing_replay(self):
        shutil.rmtree(os.path.join(self.dir, "replays"))
        doc, _, _ = self.analyze()
        self.assertEqual(doc["coverage"][0]["replayMissing"], 1)
        for f in doc["findings"]:
            if f["category"] in ("FINANCIAL_AMOUNT_MISMATCH", "FACTUAL_OBSERVATION_MISMATCH"):
                self.assertNotEqual(f["status"], "supported")


if __name__ == "__main__":
    unittest.main()
