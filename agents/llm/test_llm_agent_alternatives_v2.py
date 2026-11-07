"""Shared phase contract checks without any transport."""
import importlib.util
from pathlib import Path
import sys
import unittest

HERE=Path(__file__).resolve().parent
sys.path.insert(0,str(HERE))
import llm_agent_alternatives_v2 as W

class Phases(unittest.TestCase):
    def test_shared_system_has_all_phases_and_precedence(self):
        for n in range(1,5):self.assertIn('Phase '+str(n)+':',W.SYSTEM_PROMPT)
        self.assertIn('takes precedence',W.SYSTEM_PROMPT)
        self.assertIn('an empty array for deliberate no action',W.SYSTEM_PROMPT)
    def test_initial_parser_matches_shared_schema(self):
        d=W.parse_initial('{"reason":"no change","actions":[]}',set(),False)
        self.assertEqual(d['actions'],[])
        with self.assertRaises(ValueError):W.parse_initial('{"reason":"x","actions":[],"actions":[]}',set(),False)
        with self.assertRaises(ValueError):W.parse_initial('{"reason":"x","actions":[],"details":["staff"]}',set(),False)
