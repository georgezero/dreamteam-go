"""llm-agent 6: the engine's consequence fields are required, typed and
consistent, or the proposal is withheld (V0.12b, audit finding F1).

    python3 -I -B -m unittest discover -s agents/llm -p 'test_llm_agent6*.py'

Every case starts from a preview answer the real engine sent in V0.11b
(docs/benchmark/v11b-dev/live/draws.jsonl): one field is removed, nulled,
retyped or made inconsistent, in the engine's own structure
(internal/api/api.go: StaffingPreview, TerminationPreview, HirePreview,
MoneyLine, TelePreview, DisplacementPreview). For each invalid case:
nothing is submitted, no state changes, the failure is recorded, the model
is never asked to acknowledge, and the replay reproduces the refusal.
"""

import copy
import json
import unittest

import llm_agent3 as A3
import llm_agent6 as A6
import test_llm_agent6 as T

ANSWERS = T.recorded_previews()
TERM100 = ANSWERS[9]   # terminate E04, 100 displaced bookings
TERM2 = ANSWERS[8]     # two terminations, 6 displaced bookings
HIRE = ANSWERS[5]      # one hire: its three guaranteed lines first
TELE = ANSWERS[0]      # a teleradiology start
TELE2 = ANSWERS[11]    # a teleradiology cancel and a start


class RealEngine(T.Engine):
    """The scripted engine answering preview with a real recorded answer
    for its actions, edited: `edit(preview)` from the `edit_from`-th
    preview on; `raw(text)` edits the answer's JSON text, which is then
    read as llm-agent 6 reads the engine's lines."""

    def __init__(self, base, edit=None, edit_from=1, raw=None):
        super().__init__()
        self.base, self.edit, self.edit_from, self.raw = base, edit, edit_from, raw
        self.previews = 0

    def request(self, op, args=None):
        if op != "preview":
            return super().request(op, args)
        self.log.append((op, copy.deepcopy(args["actions"])))
        self.previews += 1
        acts, ans = self.base
        assert args["actions"] == acts, (args["actions"], acts)
        ans = copy.deepcopy(ans)
        if self.edit and self.previews >= self.edit_from:
            self.edit(ans["preview"])
        if self.raw:
            ans = A6.json_loads(self.raw(json.dumps(ans)))
        return ans


def ack_for(base):
    acts, ans = base
    items, problems = A6.consequences(acts, list(range(len(acts))), ans)
    assert not problems and items, problems
    return {"proposal": A6.payload_of(acts, items)["token"], "items": [A6.shown(i) for i in items]}


def drop(key):
    return lambda o: o.pop(key)


def put(key, value):
    return lambda o: o.__setitem__(key, value)


class Fields(T.Base):
    @staticmethod
    def fresh(engine):
        return RealEngine(engine.base, engine.edit, engine.edit_from, engine.raw)

    def withheld(self, base, edit=None, where=None, raw=None, extra=()):
        """A proposal of base's actions (with extra operations actions) whose
        preview is edited: withheld as consequences_unavailable before any
        acknowledgment, with the problem naming `where`."""
        acts = list(extra) + base[0]
        e = RealEngine(base, edit, raw=raw)
        r, h, client = self.one([T.reply("Proceed.", acts)], e)
        a = r["attempts"][0]["assessment"]
        self.assertEqual(a["status"], "unavailable", a["problems"])
        self.assertTrue(any(where in p for p in a["problems"]), (where, a["problems"]))
        self.assertIsNone(a["payload"])
        self.assertEqual(r["withheld"]["reason"], "consequences_unavailable")
        self.assertEqual(r["completeness"]["stopped"], "consequences_unavailable")
        self.assertEqual(r["batches"], [])
        self.assertEqual(self.submitted(h), [])
        self.assertEqual(len(client.payloads), 1)  # never asked to acknowledge
        self.assertTrue(a["digestsUnchanged"])
        self.assertTrue(r["digestsBeforeSubmit"]["unchanged"])
        self.assertEqual(r["stages"]["acknowledgmentRequested"], [])
        return r

    def accepted(self, base, edit=None):
        acts = base[0]
        e = RealEngine(base, edit)
        ans = copy.deepcopy(base[1])
        if edit:
            edit(ans["preview"])
        items, problems = A6.consequences(acts, list(range(len(acts))), ans)
        self.assertEqual(problems, [])
        ack = {"proposal": A6.payload_of(acts, items)["token"], "items": [A6.shown(i) for i in items]}
        r, h, _ = self.one([T.reply("Proceed.", acts), T.reply("Acknowledged.", acts, ack)], e)
        self.assertTrue(r["attempts"][1]["acknowledgment"]["match"]["accepted"])
        self.assertTrue(r["attempts"][1]["finalCheck"]["agrees"])
        self.assertEqual(self.submitted(h), [acts])
        return r, items

    # 1-3: the termination entry.
    def test_01_missing_termination_field(self):
        for f in ("lastDay", "noticeDays", "severancePayday", "employee", "endDate"):
            with self.subTest(f):
                self.withheld(TERM100, lambda p, f=f: p["terminations"][0].pop(f), "terminations[0]")

    def test_02_null_severance(self):
        self.withheld(TERM100, lambda p: put("severanceCents", None)(p["terminations"][0]),
                      "terminations[0].severanceCents")

    def test_03_wrong_severance_type(self):
        for v in ("395000", 395000.0, True, -395000, [395000], {"cents": 395000}):
            with self.subTest(repr(v)):
                self.withheld(TERM100, lambda p, v=v: put("severanceCents", v)(p["terminations"][0]),
                              "terminations[0].severanceCents")
        # Non-finite numbers: the engine's lines are refused before reading
        # (an exchange error: the run stops, nothing submitted); and as
        # values they are not integers.
        for t in ("NaN", "Infinity", "-Infinity"):
            with self.assertRaises(ValueError):
                A6.json_loads('{"severanceCents": %s}' % t)
        for v in (float("nan"), float("inf")):
            acts, ans = copy.deepcopy(TERM100)
            ans["preview"]["terminations"][0]["severanceCents"] = v
            problems = A6.consequences(acts, [0], ans)[1]
            self.assertTrue(any("terminations[0].severanceCents: not" in p for p in problems), problems)
        # Dates that are not dates.
        for v in ("2026-13-45", "20261120", 20261120, ""):
            with self.subTest(repr(v)):
                self.withheld(TERM100, lambda p, v=v: put("lastDay", v)(p["terminations"][0]),
                              "terminations[0].lastDay")

    # 4-5: displaced bookings.
    def test_04_missing_displacement_count(self):
        # The termination's own count of the batch's displacements.
        self.withheld(TERM100, lambda p: p["terminations"][0].pop("displaced"), "terminations[0].displaced")
        # The displacement list gone while the termination still counts 100.
        self.withheld(TERM100, drop("displacements"), "displacements")
        # A list that is not a list, or entries that are not displacements.
        self.withheld(TERM100, put("displacements", None), "displacements")
        self.withheld(TERM100, put("displacements", {"count": 100}), "displacements")
        self.withheld(TERM100, lambda p: p["displacements"][5].pop("appointment"), "displacements[5]")
        self.withheld(TERM100, lambda p: p["displacements"].append(dict(p["displacements"][0])), "displacements")
        self.withheld(TERM100, lambda p: p["displacements"].pop(), "terminations[0].displaced")

    def test_05_negative_displacement_count(self):
        self.withheld(TERM100, lambda p: put("displaced", -100)(p["terminations"][0]), "terminations[0].displaced")
        # Two terminations must report the same batch count.
        self.withheld(TERM2, lambda p: put("displaced", 5)(p["terminations"][1]), "terminations[1].displaced")

    # 6-9: the teleradiology guarantee.
    def test_06_missing_guarantee_amount(self):
        self.withheld(TELE, lambda p: p["teleradiology"][0].pop("guaranteedCents"), "teleradiology[0].guaranteedCents")
        self.withheld(TELE2, lambda p: p["teleradiology"][1].pop("guaranteedCents"),
                      "teleradiology[1].guaranteedCents")

    def test_07_wrong_guarantee_type(self):
        for v in ("306998", 306998.5, None, False):
            with self.subTest(repr(v)):
                self.withheld(TELE, lambda p, v=v: put("guaranteedCents", v)(p["teleradiology"][0]),
                              "teleradiology[0].guaranteedCents")

    def test_08_missing_guaranteed_through(self):
        self.withheld(TELE, lambda p: p["teleradiology"][0].pop("guaranteedThrough"),
                      "teleradiology[0].guaranteedThrough")

    def test_09_missing_minimum_term_end(self):
        self.withheld(TELE, lambda p: p["teleradiology"][0].pop("minimumTermEnd"), "teleradiology[0].minimumTermEnd")
        self.withheld(TELE, lambda p: put("minimumTermEnd", None)(p["teleradiology"][0]),
                      "teleradiology[0].minimumTermEnd")

    # 10-11: the hire.
    def test_10_missing_hire_fee(self):
        self.withheld(HIRE, lambda p: p["guaranteed"][0].pop("cents"), "guaranteed[0].cents")
        self.withheld(HIRE, lambda p: p["guaranteed"][2].pop("cents"), "guaranteed[2].cents")

    def test_11_incorrect_hire_cost_structure(self):
        self.withheld(HIRE, lambda p: put("cents", "300000")(p["guaranteed"][0]), "guaranteed[0].cents")
        self.withheld(HIRE, lambda p: put("cents", {"amount": 1})(p["guaranteed"][2]), "guaranteed[2].cents")
        self.withheld(HIRE, lambda p: p["guaranteed"].pop(2), "guaranteed")
        self.withheld(HIRE, lambda p: p["guaranteed"].insert(0, p["guaranteed"].pop(2)), "guaranteed")
        self.withheld(HIRE, lambda p: p["hires"][0].pop("offer"), "hires[0].offer")
        self.withheld(HIRE, put("guaranteed", None), "guaranteed")

    # 12-13: duplicates and consequences nothing in the batch accounts for.
    def test_12_duplicate_consequence_field(self):
        # The engine's JSON line with a key twice: the last value would be
        # read silently; the reader marks it and the proposal is withheld.
        sev = TERM100[1]["preview"]["terminations"][0]["severanceCents"]
        self.withheld(TERM100, raw=lambda t: t.replace(f'"severanceCents": {sev}',
                                                       f'"severanceCents": 1, "severanceCents": {sev}', 1),
                      where="duplicate")
        self.withheld(TELE, raw=lambda t: t.replace('"guaranteedThrough": ',
                                                    '"guaranteedThrough": "2026-11-01", "guaranteedThrough": ', 1),
                      where="duplicate")
        # Duplicate keys anywhere else in an engine line are read as before.
        self.assertEqual(A6.json_loads('{"a": 1, "a": 2}'), {"a": 2})

    def test_13_unknown_required_consequence(self):
        # A retained obligation in the answer that no action of the batch is.
        def extra_hire(p):
            h = dict(p["hires"][0], requestId="someone-else")
            p["hires"].append(h)
        self.withheld(HIRE, extra_hire, "hires")

        def extra_tele(p):
            p["teleradiology"].append(dict(p["teleradiology"][0]))
        self.withheld(TELE, extra_tele, "teleradiology")
        # A termination of someone not in the batch.
        self.withheld(TERM100, lambda p: p["terminations"].append(dict(p["terminations"][0], employee="E99")),
                      "terminations")
        # An entry of an unknown kind for a teleradiology action.
        self.withheld(TELE, lambda p: put("op", "renew")(p["teleradiology"][0]), "teleradiology[0]")
        # A section that is not a list.
        self.withheld(TERM100, put("terminations", {"E04": {}}), "terminations")
        self.withheld(TERM100, put("results", None), "results")

    # 14-15: what stays valid.
    def test_14_valid_zero_value_consequence(self):
        r, items = self.accepted(TERM100, lambda p: p["terminations"][0].update(severanceCents=0))
        self.assertEqual(items[0]["consequences"]["severance"]["severanceCents"], 0)
        r, items = self.accepted(TELE, lambda p: p["teleradiology"][0].update(guaranteedCents=0))
        self.assertEqual(items[0]["consequences"]["retained_obligation"]["guaranteedCents"], 0)
        # No displacement at all: no list, every termination counts 0.
        r, items = self.accepted(TERM100, lambda p: (p.pop("displacements"),
                                                     p["terminations"][0].update(displaced=0)))
        self.assertEqual([i["type"] for i in items], ["terminate"])

    def test_15_valid_complete_preview(self):
        for name, base in (("termination, 100 displaced", TERM100), ("two terminations", TERM2), ("hire", HIRE),
                           ("teleradiology", TELE), ("cancel and start", TELE2)):
            with self.subTest(name):
                r, items = self.accepted(base)
                self.assertTrue(items)
        # Every recorded real answer still reads without a problem.
        for acts, ans in ANSWERS:
            self.assertEqual(A6.consequences(acts, list(range(len(acts))), ans)[1], [], acts)

    # 16-18: the workflow around it.
    def test_16_missing_field_in_mixed_batch(self):
        r = self.withheld(TERM100, lambda p: p["terminations"][0].pop("severanceCents"),
                          "terminations[0].severanceCents", extra=[T.OPS])
        self.assertEqual([b["actions"] for b in r["withheld"]["batches"]], [[T.OPS], TERM100[0]])

    def test_17_preview_changes_between_acknowledgment_and_submission(self):
        acts = TERM100[0]
        e = RealEngine(TERM100, lambda p: p["terminations"][0].pop("severanceCents"), edit_from=2)
        r, h, _ = self.one([T.reply("Proceed.", acts), T.reply("Acknowledged.", acts, ack_for(TERM100))], e)
        a2 = r["attempts"][1]
        self.assertTrue(a2["acknowledgment"]["match"]["accepted"])
        self.assertFalse(a2["finalCheck"]["agrees"])
        self.assertEqual(r["withheld"]["reason"], "final_check_disagrees")
        self.assertEqual(self.submitted(h), [])

    def test_18_missing_consequence_data_during_replay(self):
        acts = TERM100[0]
        # Recorded with a complete preview; the engine now omits a field: the
        # replay is refused, not re-certified.
        written, _, _ = T.run([T.reply("Proceed.", acts), T.reply("Acknowledged.", acts, ack_for(TERM100))],
                              RealEngine(TERM100))
        self.assertEqual(len(written[0]["batches"]), 1)
        with self.assertRaises(A3.ReplayError):
            T.replay(written, RealEngine(TERM100, lambda p: p["terminations"][0].pop("severanceCents")))
        # Recorded withheld; replayed against a complete preview: refused too.
        written, _, _ = T.run([T.reply("Proceed.", acts)],
                              RealEngine(TERM100, lambda p: p["terminations"][0].pop("severanceCents")))
        self.assertEqual(written[0]["withheld"]["reason"], "consequences_unavailable")
        with self.assertRaises(A3.ReplayError):
            T.replay(written, RealEngine(TERM100))
        # Against the same incomplete preview, the refusal replays exactly.
        again, _ = T.replay(written, RealEngine(TERM100, lambda p: p["terminations"][0].pop("severanceCents")))
        self.assertEqual({k: v for k, v in again[0].items() if k not in ("mode", "replayed", "replayChecked")},
                         {k: v for k, v in written[0].items() if k != "mode"})


if __name__ == "__main__":
    unittest.main()
