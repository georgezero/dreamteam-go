"""Unit tests of the LLM agent's settings, parsing and prompt bounds.

    python3 -B -m unittest discover -s agents/llm

The agent's exchange with the harness and a mock model server is tested by
internal/bench/llm_agent_test.go.
"""

import json
import os
import tempfile
import unittest

import llm_agent as A

TYPES = {"set_reading_allocation", "set_walkin_policy"}


def settings_file(obj):
    fd, path = tempfile.mkstemp(suffix=".json")
    with os.fdopen(fd, "w") as f:
        json.dump(obj, f)
    return path


BASE = {"schema": A.SETTINGS_SCHEMA, "model": "m", "baseUrl": "http://localhost:9/v1"}


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
        self.assertEqual(s["systemPromptSource"], "built-in")

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
        ]:
            with self.assertRaises(ValueError) as e:
                A.parse_decision(text, TYPES)
            self.assertTrue(str(e.exception).startswith(code), (text, str(e.exception)))

    def test_completion(self):
        body = json.dumps({"choices": [{"message": {"content": "{}"}}], "usage": {"total_tokens": 5}})
        self.assertEqual(A.completion(body), ("{}", {"total_tokens": 5}))
        for body, code in [("x", "response_not_json"), ("{}", "response_shape"),
                           (json.dumps({"choices": [{"message": {"content": " "}}]}), "empty_content")]:
            with self.assertRaises(ValueError) as e:
                A.completion(body)
            self.assertTrue(str(e.exception).startswith(code))


class Prompt(unittest.TestCase):
    obs = {"time": "2026-10-13T07:00:00", "day": 1, "weekday": "Tuesday", "worklist": {"oldestMin": 30}}
    catalog = {"batch": {"max": 64}, "types": [
        {"type": "hire", "schema": {"x": "y" * 5000}, "constraints": ["c"], "targets": {"offers": []}},
        {"type": "set_walkin_policy", "schema": {"p": 1}, "constraints": ["c"], "targets": []}]}

    def test_bounded(self):
        full, left = A.build_prompt(3, self.obs, self.catalog, 100000)
        self.assertEqual(left, [])
        self.assertIn("Decision point 3", full)
        small, left = A.build_prompt(3, self.obs, self.catalog, 3000)
        self.assertEqual(left, ["hire"])
        self.assertLessEqual(len(small), 3000)
        self.assertIn('"set_walkin_policy","schema":{"p":1}', small)
        none, _ = A.build_prompt(3, self.obs, self.catalog, 100)
        self.assertIsNone(none)

    def test_no_seed_in_prompt(self):
        self.assertNotIn("seed", A.SYSTEM_PROMPT.lower())


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


if __name__ == "__main__":
    unittest.main()
