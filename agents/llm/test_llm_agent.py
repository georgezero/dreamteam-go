"""Unit tests of the LLM agent: settings, parsing, the prompt and its
guidance, memory, usage and cost, and replay matching.

    python3 -B -m unittest discover -s agents/llm

The agent's exchange with the harness and a mock model server is tested by
internal/bench/llm_agent_test.go.
"""

import copy
import json
import os
import tempfile
import unittest

import llm_agent as A

TYPES = {"set_reading_allocation", "set_walkin_policy"}
HERE = os.path.dirname(os.path.abspath(__file__))


def settings_file(obj):
    fd, path = tempfile.mkstemp(suffix=".json")
    with os.fdopen(fd, "w") as f:
        json.dump(obj, f)
    return path


def fixture():
    """The V0.5a live run's decision point 0 (sustainable, seed 40601): its
    observe and actions responses, and the V0.5a prompt's size."""
    with open(os.path.join(HERE, "testdata", "v05a-point0.json"), encoding="utf-8") as f:
        return json.load(f)


BASE = {"schema": A.SETTINGS_SCHEMA, "model": "m", "baseUrl": "http://localhost:9/v1"}
PSET = {k: A.DEFAULTS[k] for k in A.PROMPT_SETTINGS}


class Settings(unittest.TestCase):
    def load(self, obj, env=None):
        p = settings_file(obj)
        try:
            return A.load_settings(p, env or {})
        finally:
            os.remove(p)

    def test_defaults_and_environment(self):
        s = self.load({"schema": A.SETTINGS_SCHEMA, "modelEnv": "M", "baseUrlEnv": "U", "apiKeyEnv": "K"},
                      {"M": "model-x", "U": "https://h.example:8443/v1/", "K": "sk-secret"})
        self.assertEqual((s["model"], s["baseUrl"], s["apiKey"]), ("model-x", "https://h.example:8443/v1", "sk-secret"))
        self.assertEqual((s["temperature"], s["maxRetries"], s["responseFormat"]), (0, 1, "json_object"))
        self.assertEqual((s["observation"], s["catalog"], s["allowDetails"], s["pricing"]),
                         ("summary", "guided", True, None))
        self.assertEqual((s["memoryDecisions"], s["memoryMetricPoints"], s["memoryMaxChars"]), (6, 6, 6000))
        self.assertEqual(s["systemPromptSource"], "built-in")

    def test_v05a_settings_still_load(self):
        p = os.path.join(HERE, "..", "..", "docs", "benchmark", "v05a-dev", "llm-settings.json")
        s = A.load_settings(p, {"DREAMTEAM_LLM_BASE_URL": "http://localhost:9/v1"})
        self.assertEqual(s["model"], "Qwen3.6-35B-A3B-oQ4e-mtp")

    def test_v05c_settings_identical(self):
        """The V0.5c settings differ only in the model, and the memory-off
        control only in memoryDecisions (docs/benchmark/v05c-dev/PLAN.md)."""
        d = os.path.join(HERE, "..", "..", "docs", "benchmark", "v05c-dev")
        got = {}
        for n in ("qwen36-35b-a3b", "gemma4-26b-a4b", "gemma4-31b", "qwen36-35b-a3b-nomem"):
            with open(os.path.join(d, f"llm-{n}.json"), encoding="utf-8") as f:
                raw = json.load(f)
            got[n] = A.load_settings(os.path.join(d, f"llm-{n}.json"), {"DREAMTEAM_LLM_BASE_URL": "http://localhost:9/v1"})
            self.assertNotIn("baseUrl", raw)
        same = lambda s: {k: v for k, v in s.items() if k not in ("model", "description")}
        ref = same(got["qwen36-35b-a3b"])
        self.assertEqual(ref["maxOutputTokens"], 4096)
        self.assertEqual(ref["memoryDecisions"], 6)
        for n in ("gemma4-26b-a4b", "gemma4-31b"):
            self.assertEqual(same(got[n]), ref, n)
        self.assertEqual(same(got["qwen36-35b-a3b-nomem"]), dict(ref, memoryDecisions=0))
        self.assertEqual(got["qwen36-35b-a3b-nomem"]["model"], got["qwen36-35b-a3b"]["model"])

    def test_refused(self):
        for obj, msg in [
            ({**BASE, "apiKey": "sk-x"}, "never put in the settings"),
            ({**BASE, "modelEnv": "M"}, "exactly one of model and modelEnv"),
            ({**BASE, "colour": 1}, "unknown settings fields"),
            ({**BASE, "schema": "x"}, "schema must be"),
            ({**BASE, "baseUrl": "http://user:pw@h/v1"}, "must not carry credentials"),
            ({**BASE, "baseUrl": "ftp://h/v1"}, "http or https"),
            ({**BASE, "maxRetries": 9}, "maxRetries must be"),
            ({**BASE, "temperature": True}, "temperature must be"),
            ({"schema": A.SETTINGS_SCHEMA, "model": "m", "baseUrlEnv": "UNSET"}, "is not set"),
            ({**BASE, "memoryMaxChars": 100}, "memoryMaxChars must be"),
            ({**BASE, "observation": "full"}, "observation must be"),
            ({**BASE, "catalog": "none"}, "catalog must be"),
            ({**BASE, "pricing": {"currency": "USD", "inputPerMillion": 1}}, "pricing must be"),
            ({**BASE, "pricing": {"currency": "usd", "inputPerMillion": 1, "outputPerMillion": 2}}, "three-letter"),
            ({**BASE, "pricing": {"currency": "USD", "inputPerMillion": -1, "outputPerMillion": 2}}, "inputPerMillion"),
        ]:
            with self.assertRaises(A.SettingsError) as e:
                self.load(obj)
            self.assertIn(msg, str(e.exception), obj)

    def test_system_prompt_beside_the_settings(self):
        d = tempfile.mkdtemp()
        with open(os.path.join(d, "prompt.txt"), "w") as f:
            f.write("Be brief.")
        p = os.path.join(d, "s.json")
        with open(p, "w") as f:
            json.dump({**BASE, "systemPromptPath": "prompt.txt"}, f)
        s = A.load_settings(p, {})
        self.assertEqual((s["systemPrompt"], s["systemPromptSource"]), ("Be brief.", "prompt.txt"))

    def test_endpoint_has_no_secrets(self):
        self.assertEqual(A.endpoint_id("https://h.example:8443/v1/?key=abc#x"), "https://h.example:8443/v1")
        self.assertEqual(A.endpoint_id("http://[::1]:8000/v1"), "http://[::1]:8000/v1")

    def test_payload_has_no_key(self):
        s = self.load({**BASE, "apiKeyEnv": "K"}, {"K": "sk-secret"})
        p = A.ModelClient(s).payload([{"role": "user", "content": "x"}])
        self.assertNotIn("sk-secret", json.dumps(p))
        self.assertEqual(p["response_format"], {"type": "json_object"})


class Decisions(unittest.TestCase):
    def test_accepted(self):
        a = {"type": "set_walkin_policy", "policy": "accept"}
        for text in [json.dumps({"reason": "r", "actions": [a]}),
                     "```json\n" + json.dumps({"reason": "r", "actions": [a]}) + "\n```",
                     "<think>hmm</think>\n" + json.dumps({"reason": "r", "actions": [a]}),
                     '{"reason": "", "actions": []}']:
            d = A.parse_decision(text, TYPES)
            self.assertEqual(set(d), {"reason", "actions"})
        d = A.parse_decision('{"reason": "r", "actions": [], "details": ["hire"]}', TYPES, details_allowed=True)
        self.assertEqual(d["details"], ["hire"])

    def test_refused(self):
        for text, code in [
            ("hire more", "invalid_json"),
            ("[]", "invalid_decision"),
            ('{"reason": "r"}', "invalid_decision"),
            ('{"reason": "r", "actions": [], "note": 1}', "invalid_decision"),
            ('{"reason": 3, "actions": []}', "invalid_decision"),
            ('{"reason": "r", "actions": {}}', "invalid_decision"),
            ('{"reason": "r", "actions": [1]}', "invalid_action"),
            ('{"reason": "r", "actions": [{"type": "buy_scanner"}]}', "unknown_action"),
            (json.dumps({"reason": "r", "actions": [{"type": "set_walkin_policy"}] * 65}), "invalid_decision"),
            ('{"reason": "r", "actions": []} trailing', "invalid_json"),
            ('{"reason": "r", "actions": [], "details": ["hire"]}', "invalid_decision"),  # not allowed now
        ]:
            with self.assertRaises(ValueError) as e:
                A.parse_decision(text, TYPES)
            self.assertTrue(str(e.exception).startswith(code), (text, str(e.exception)))
        for text in ['{"reason": "r", "actions": [{"type": "set_walkin_policy"}], "details": ["hire"]}',
                     '{"reason": "r", "actions": [], "details": []}',
                     '{"reason": "r", "actions": [], "details": [1]}']:
            with self.assertRaises(ValueError):
                A.parse_decision(text, TYPES, details_allowed=True)

    def test_split(self):
        acts = [{"type": "hire"}, {"type": "extend_shift"}, {"type": "set_roster"}, {"type": "set_walkin_policy"}]
        self.assertEqual(A.split(acts), [([acts[1], acts[3]], [1, 3]), ([acts[0], acts[2]], [0, 2])])
        self.assertEqual(A.split([]), [])

    def test_completion(self):
        body = json.dumps({"choices": [{"message": {"content": "{}"}}], "usage": {"total_tokens": 5}, "model": "x"})
        self.assertEqual(A.completion(body), ("{}", {"total_tokens": 5}, "x"))
        for body, code in [("x", "response_not_json"), ("{}", "response_shape"),
                           (json.dumps({"choices": [{"message": {"content": " "}}]}), "empty_content")]:
            with self.assertRaises(ValueError) as e:
                A.completion(body)
            self.assertTrue(str(e.exception).startswith(code))


class Prompt(unittest.TestCase):
    """The targeted prompt on a recorded observation (owner's tests 6, 7
    and the size comparison)."""

    def setUp(self):
        self.f = fixture()
        self.obs, self.omitted, self.cat = self.f["observation"], self.f["omitted"], self.f["catalog"]
        m = A.Memory(6, 6, 6000)
        m.observe(0, self.obs)
        self.mem = m.render(A.pending(self.obs, self.cat))
        self.prompt, self.left = A.build_prompt(0, self.obs, self.omitted, self.cat, self.mem, PSET)

    def test_smaller_than_v05a(self):
        v1 = self.f["v1Prompt"]["userChars"]
        self.assertEqual(v1, 67767)
        self.assertLess(len(self.prompt), 0.45 * v1, len(self.prompt))

    def test_constraints_shown(self):
        p = self.prompt
        for want in ["Operating hours 08:00–20:00; scheduled close today 20:00", "Today is 2026-10-13 (Tue), now 07:00",
                     "earliestEffective 2026-10-27 (in 14 days)",  # set_reading_allocation: the past_date refusal
                     "standardLeadDays 14–28; premiumLeadDays 7–21; earliestStartDate 2026-10-14 (in 1 days)",
                     "earliestLastDay 2026-10-27 (in 14 days)", "earliestEffective 2026-11-02 (in 20 days)",
                     "noticeDays 14; minTermDays 30; cancelNoticeDays 30",
                     '"capAt":"23:30"', '"maxTechnologists":8',
                     "Rule: Headcount caps count recruiting, onboarding, active and notice staff.",
                     "cannot share a batch with V0.1 actions (mixed_batch)",
                     "as two batches at this decision point, operations first",
                     "A: Mon–Fri 07:45–16:15 @desk break 12:00–12:30; Sat,Sun off | B: same as A"]:
            self.assertIn(want, p)
        # Where the catalog gives no date, the prompt says so.
        self.assertIn("## set_walkin_policy (operations batch)", p)
        self.assertIn("state no dates or lead times for this type", p)

    def test_every_action_type_and_its_fields(self):
        for t in self.cat["types"]:
            self.assertIn(f"## {t['type']} (", self.prompt)
            for field in t["schema"]["properties"]:
                self.assertIn(field, self.prompt.split(f"## {t['type']} (")[1].split("\n## ")[0], (t["type"], field))
        self.assertEqual({t["type"] for t in self.cat["types"]} - set(A.STAFFING),
                         {"extend_shift", "set_walkin_policy", "set_worklist_policy"})
        for t in A.STAFFING:  # the agent's staffing set is the catalog's
            self.assertIn(t, self.cat["batch"][2])

    def test_details_and_full_catalog(self):
        for t in self.cat["types"]:
            d = A.details_text(None, [t["type"]], self.obs, self.omitted, self.cat, 10 ** 6)
            self.assertIn(A.compact(t), d)
        d = A.details_text(None, ["finance.cashHistory", "nonsense"], self.obs, self.omitted, self.cat, 10 ** 6)
        self.assertIn(A.compact(self.obs["finance"]["cashHistory"]), d)
        self.assertIn("nonsense: not an action type", d)
        full, _ = A.build_prompt(0, self.obs, self.omitted, self.cat, self.mem, {**PSET, "catalog": "full"})
        self.assertIn(A.compact(self.cat["types"]), full)
        compact_obs, _ = A.build_prompt(0, self.obs, self.omitted, self.cat, self.mem, {**PSET, "observation": "compact"})
        self.assertIn(A.compact(self.obs), compact_obs)

    def test_left_out_is_named(self):
        left = self.prompt.split("\nLEFT OUT (")[1]
        for path, keep in A.SUMMARY_CUTS:
            v = A.full_path(self.obs, path)
            if v not in (None, [], {}) and (keep is None or len(v) > keep):
                self.assertTrue(path in left, path)
        for k in self.omitted:
            if self.omitted[k]:
                self.assertTrue(k in left, k)
        s, _ = A.summarize(self.obs)
        self.assertEqual(len(s["worklist"]["items"]), 5)
        self.assertEqual(self.obs["worklist"]["unread"], 9)  # the input is not changed

    def test_bounded(self):
        none, _ = A.build_prompt(0, self.obs, self.omitted, self.cat, self.mem, {**PSET, "maxPromptChars": 1000})
        self.assertIsNone(none)

    def test_no_seed_in_prompt(self):
        self.assertNotIn("seed", A.SYSTEM_PROMPT.lower())
        self.assertNotIn("seed", self.prompt.lower())
        self.assertNotIn("t90", self.prompt.lower())


def obs_at(day, accepted=0, rejected=0, recent=None, cash=100000, unread=3, extra=None):
    o = {"time": f"2026-10-{day:02d}T07:00:00", "date": f"2026-10-{day:02d}", "day": day,
         "actions": {"attempted": accepted + rejected, "accepted": accepted, "rejected": rejected, "recent": recent},
         "finance": {"cashCents": cash, "monthToDate": {"operatingProfitCents": -500}},
         "worklist": {"unread": unread, "items": [{"overdue": True}, {"overdue": False}]},
         "demand": {"modalities": [{"sinceStart": {"lost": day}}]},
         "flow": {"sinceStart": {"examsDone": 10 * day, "overtimeMin": day, "noCoverage": 0}},
         "staff": []}
    o.update(extra or {})
    return o


def rec_at(point, day, outcome="actions", batches=None, refusals=None, reason="r"):
    return {"point": point, "time": f"2026-10-{day:02d}T07:00:00", "outcome": outcome, "reason": reason,
            "batches": batches if batches is not None else [], "refusals": refusals or [], "errors": []}


class MemoryTests(unittest.TestCase):
    """Owner's tests 1–4."""

    def test_updates(self):
        m = A.Memory(6, 6, 6000)
        m.observe(0, obs_at(1))
        self.assertIn("No earlier decisions in this run.", m.render([]))
        act = {"type": "extend_shift", "employee": "E07", "minutes": 120}
        m.record(rec_at(0, 1, batches=[[act]], reason="backlog",
                        refusals=[{"attempt": 1, "action": 1, "type": "extend_shift", "code": "after_close",
                                   "message": "Dr. Blake's shift today already runs to the scheduled close."}]))
        t = m.render([])
        self.assertIn("point 0 (10-01 07:00): actions — \"backlog\"", t)
        self.assertIn("applied: not yet observed", t)
        self.assertIn("refused by validate first: extend_shift after_close", t)
        self.assertIn("extend_shift after_close (point 0)", t)
        # The next observation shows the outcome, from the action log.
        result = {"time": "2026-10-01T07:00:00", "ok": True, "code": "accepted", "message": "Shift extended."}
        m.observe(1, obs_at(2, accepted=1, recent=[result], cash=90000))
        t = m.render([])
        self.assertIn("applied: 1 accepted, 0 refused; accepted: Shift extended.", t)
        self.assertIn("1 | 10-02 07:00 | $900 | -$5 | 3 | 1 | 2 | 20 | 2 | 0", t)
        self.assertIn('submitted: {"type":"extend_shift","employee":"E07","minutes":120}', t)

    def test_refused_on_apply(self):
        m = A.Memory(6, 6, 6000)
        m.observe(0, obs_at(1, accepted=2))
        m.record(rec_at(0, 1, batches=[[{"type": "hire"}]]))
        m.observe(1, obs_at(2, accepted=2, rejected=1))
        self.assertIn("applied: 0 accepted, 1 refused", m.render([]))

    def test_bounded_over_90_days(self):
        m = A.Memory(6, 6, 6000)
        sizes = []
        for p in range(120):
            day = 1 + p % 28
            m.observe(p, obs_at(day, accepted=p, rejected=p))
            m.record(rec_at(p, day, reason="x" * 3000,
                            batches=[[{"type": "extend_shift", "employee": f"E{i:02d}", "minutes": 30} for i in range(20)]],
                            refusals=[{"type": f"t{p % 11}", "code": f"c{p % 13}", "message": "m" * 500}] * 5))
            t = m.render([f"pending {i} " + "y" * 400 for i in range(40)])
            sizes.append(len(t))
            self.assertLessEqual(len(t), 6000)
            self.assertLessEqual(len(m.entries), 6)
            self.assertLessEqual(len(m.rows), 6)
            self.assertLessEqual(len(m.lessons), 8)
        self.assertIn("Earlier, points 0–", t)
        self.assertEqual(m.earlier["points"], 114)
        self.assertEqual(m.earlier["submitted"], 114 * 20)
        tiny = A.Memory(50, 30, 500)
        for p in range(90):
            tiny.observe(p, obs_at(1 + p % 28))
            tiny.record(rec_at(p, 1 + p % 28, reason="z" * 200))
            self.assertLessEqual(len(tiny.render(["p" * 300] * 12)), 500)

    def test_resets(self):
        a = A.Memory(6, 6, 6000)
        a.observe(0, obs_at(1))
        a.record(rec_at(0, 1))
        b = A.Memory(6, 6, 6000)
        b.observe(0, obs_at(1))
        self.assertIn("No earlier decisions in this run.", b.render([]))
        self.assertNotEqual(a.render([]), b.render([]))

    def test_off(self):
        """memoryDecisions 0 (the V0.5c memory-off control): no earlier
        decision, refusal or metric reaches the prompt, and the rest of the
        prompt is the memory-on prompt's."""
        on, off = A.Memory(6, 6, 6000), A.Memory(0, 6, 6000)
        for p in range(10):
            for m in (on, off):
                m.observe(p, obs_at(1 + p, accepted=p, rejected=1))
                m.record(rec_at(p, 1 + p, refusals=[{"type": "extend_shift", "code": "after_close",
                                                     "message": "ends after the close"}]))
        text = off.render(["set_reading_allocation E07: pending"])
        self.assertEqual(text, "MEMORY (llm-agent-memory/v1): off in this run (memoryDecisions 0); nothing from "
                               "earlier decision points is carried over.\nPending changes now (from the observation "
                               "and catalog):\n- set_reading_allocation E07: pending")
        for gone in ("after_close", "point 3", "Metrics", "Earlier", "Recent"):
            self.assertNotIn(gone, text)
        self.assertIn("after_close", on.render([]))
        f = fixture()
        pset = dict(PSET, memoryDecisions=0)
        a, _ = A.build_prompt(9, f["observation"], f["omitted"], f["catalog"], on.render([]), PSET)
        b, _ = A.build_prompt(9, f["observation"], f["omitted"], f["catalog"], off.render([]), pset)
        strip = lambda t: t.replace(on.render([]), "").replace(off.render([]), "")
        self.assertEqual(strip(a), strip(b))

    def test_pending_from_public_data_only(self):
        f = fixture()
        cat = copy.deepcopy(f["catalog"])
        for t in cat["types"]:
            if t["type"] == "set_reading_allocation":
                t["targets"][0]["versions"].append({"siteBP": 7000, "effective": "2026-10-30", "inEffect": False,
                                                    "pending": True})
        obs = copy.deepcopy(f["observation"])
        obs["staff"].append({"id": "E12", "name": "Kim", "role": "technologist", "employment": "onboarding",
                             "startDate": "2026-11-10", "hourlyCents": 4000})
        self.assertEqual(A.pending(f["observation"], f["catalog"]), [])
        pend = A.pending(obs, cat)
        self.assertEqual(pend[0], 'set_reading_allocation E07: {"siteBP":7000,"effective":"2026-10-30",'
                                  '"inEffect":false,"pending":true}')
        self.assertIn('"employment":"onboarding","startDate":"2026-11-10"', pend[1])
        # The memory sees only what it is given: a field the observation
        # does not have never appears, whatever the agent is handed.
        m = A.Memory(6, 6, 6000)
        m.observe(0, obs_at(1, extra={"futureEvents": ["SECRET-ARRIVAL"], "seed": 424242}))
        m.record(rec_at(0, 1, outcome="no_action"))
        t = m.render(pend)
        self.assertNotIn("SECRET-ARRIVAL", t)
        self.assertNotIn("424242", t)
        self.assertIn("2026-10-30", t)


class Usage(unittest.TestCase):
    """Owner's tests 8 and 9."""

    class Ev:
        def __init__(self, calls):
            self.calls = calls

    def call(self, turn, usage=None, content=None, latency=100, error=None, sent="x" * 400):
        return {"turn": turn, "usage": usage, "content": content, "latencyMs": latency, "error": error,
                "servedModel": "served-x" if content else None, "request": {"messages": [{"content": sent}]}}

    def test_totals_and_cost(self):
        calls = [self.call("first", {"prompt_tokens": 1000, "completion_tokens": 50}, "{}", 100),
                 self.call("retry", {"prompt_tokens": 1200, "completion_tokens": 30}, "{}", 300),
                 self.call("details", None, "y" * 40, 200),  # answered without counts: estimated
                 self.call("retry", None, None, 900, "timeout: no answer")]  # unknown
        recs = [{"outcome": "actions", "refusals": [{"code": "x"}]}, {"outcome": "no_action"}, {"outcome": "fallback"}]
        settings = {"model": "m", "pricing": {"currency": "USD", "inputPerMillion": 2.0, "outputPerMillion": 8.0}}
        s = A.summary(self.Ev(calls), recs, settings)
        self.assertEqual((s["requests"], s["retries"], s["detailRequests"], s["failedRequests"]), (4, 2, 1, 1))
        self.assertEqual(s["tokens"]["reported"], {"requests": 2, "prompt": 2200, "completion": 80, "total": 2280})
        self.assertEqual(s["tokens"]["estimated"], {"requests": 1, "prompt": 100, "completion": 10, "total": 110})
        self.assertEqual(s["tokens"]["unknown"], {"requests": 1})
        self.assertEqual(s["latencyMs"], {"total": 1500, "mean": 375, "median": 250, "max": 900})
        self.assertEqual(s["decisions"], {"valid": 2, "withActions": 1, "noAction": 1, "refused": 1, "fallback": 1})
        self.assertEqual(s["servedModels"], ["served-x"])
        c = s["cost"]
        self.assertAlmostEqual(c["estimated"], (2300 * 2.0 + 90 * 8.0) / 1e6)
        self.assertIn("estimate, not provider billing", c["basis"])
        self.assertIn("reported and estimated tokens", c["basis"])
        self.assertEqual(c["requestsWithUnknownUsage"], 1)
        text = A.usage_text(s)
        self.assertIn("cost, estimated       0.005320 USD", text)
        self.assertIn("(from characters, not counted)", text)

    def test_no_prices(self):
        s = A.summary(self.Ev([]), [], {"model": "m", "pricing": None})
        self.assertIsNone(s["cost"])
        self.assertEqual(s["latencyMs"], {"total": 0, "mean": None, "median": None, "max": None})
        self.assertIn("no prices configured", A.usage_text(s))


class Replay(unittest.TestCase):
    def rec(self, point, state, batches=()):
        return {"point": point, "digests": {"state": state}, "batches": list(batches)}

    def test_choice_and_checks(self):
        r1 = ("a", "h1", [self.rec(0, "s0"), self.rec(1, "s1", [[{"type": "x"}]])])
        r2 = ("b", "h2", [self.rec(0, "t0")])
        rp = A.Replayer([r1, r2])
        self.assertEqual(rp.next(0, {"digests": {"state": "s0"}})["point"], 0)
        with self.assertRaises(A.ReplayError):
            rp.next(1, {"digests": {"state": "other"}})
        rp = A.Replayer([r1, r2])
        rp.next(0, {"digests": {"state": "s0"}})
        with self.assertRaises(A.ReplayError):
            rp.finish()  # point 1 not reached
        with self.assertRaises(A.ReplayError):
            A.Replayer([r1]).next(0, {"digests": {"state": "zz"}})
        clash = ("c", "h3", [self.rec(0, "s0", [[{"type": "y"}]])])
        with self.assertRaises(A.ReplayError):
            A.Replayer([r1, clash]).next(0, {"digests": {"state": "s0"}})

    def write(self, recs):
        p = settings_file({})
        with open(p, "w") as f:
            for r in recs:
                f.write(json.dumps(r) + "\n")
        return p

    def test_versions(self):
        # V0.5a's recording (no memory) loads; so does this version's.
        v05a = os.path.join(HERE, "..", "..", "docs", "benchmark", "v05a-dev", "llm-decisions.jsonl")
        self.assertEqual(len(A.load_recordings([v05a])[0][2]), 7)
        ok = self.write([{**self.rec(0, "s"), "memoryFormat": A.MEMORY_FORMAT, "promptVersion": A.PROMPT_VERSION}])
        self.assertEqual(len(A.load_recordings([ok])), 1)
        # Another memory format or prompt version: an explicit refusal.
        for fmt, pv in [("llm-agent-memory/v9", A.PROMPT_VERSION), (A.MEMORY_FORMAT, "llm-agent-prompt/v1")]:
            p = self.write([{**self.rec(0, "s"), "memoryFormat": fmt, "promptVersion": pv}])
            with self.assertRaises(A.SettingsError) as e:
                A.load_recordings([p])
            self.assertIn("version incompatible", str(e.exception))


if __name__ == "__main__":
    unittest.main()
