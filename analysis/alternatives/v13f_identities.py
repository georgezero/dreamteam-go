"""Rebuild the successor contract/source identity index without a transport."""
import argparse
import hashlib
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
DEST = ROOT / 'docs/v13f/CONTRACT-IDENTITIES.json'


def identities():
    paths = {
        'llm-agent-alternatives-system/v2': ['agents/llm/text/alternatives-system-v2.txt'],
        'dreamteam.initial-phase-template/v2': ['agents/llm/text/initial-v2.txt'],
        'dreamteam.correction-phase-template/v2': ['agents/llm/text/correction-v2.txt'],
        'dreamteam.phase-response-schemas/v2': ['agents/llm/text/response-schemas-v2.json'],
        'dreamteam.phase-response-parser/v2': ['agents/llm/llm_agent_alternatives_v2.py', 'internal/alternatives/turn_v2.go'],
        'llm-agent-alternatives-workflow/v2': ['agents/llm/llm_agent_alternatives_v2.py', 'docs/v13f/WORKFLOW.md'],
        'llm-agent-phase-validation-first/v2': ['agents/llm/llm_agent_alternatives_v2.py'],
        'llm-agent-prompt/v3': ['agents/llm/text/initial-v2.txt'],
        'dreamteam.phase-proposal-record/v2': ['agents/llm/llm_agent_alternatives_v2.py'],
        'dreamteam.alternatives-decision-record/v2': ['agents/llm/llm_agent_alternatives_v2.py'],
        'dreamteam.submitted-action/v1': ['internal/alternatives/submitted_v2.go'],
        'dreamteam.alternatives-view/v2': ['internal/alternatives/view_v2.go', 'docs/benchmark/v13f-dev/presentation/view-table.txt'],
        'dreamteam.alternatives-arms/v3': ['internal/alternatives/turn_v2.go'],
        'dreamteam.alternatives-turn/v2': ['internal/alternatives/turn_v2.go'],
        'dreamteam.alternatives-outcome/v2': ['internal/alternatives/turn_v2.go'],
        'dreamteam.alternatives-presentation/v3': ['internal/alternatives/text/presentation-v3.tmpl', 'docs/v13f/PRESENTATION_V3.md'],
        'dreamteam.typed-citations/v2': ['internal/alternatives/citations_v2.go'],
        'dreamteam.alternatives-outcomes/v2': ['docs/v13f/OUTCOME_DEFINITIONS.md'],
        'dreamteam.partial-run-verification/v2': ['docs/v13f/PARTIAL_RUN_PROCEDURE.md', 'analysis/alternatives/v13f.py'],
        'dreamteam-v13f-driver/2': ['analysis/alternatives/v13f.py', 'docs/v13f/DRIVER.md'],
        'dreamteam.alternatives-experiment/v2': ['docs/benchmark/v13f-dev/protocol-draft.json', 'docs/v13f/PROTOCOL_DRAFT.md'],
        'dreamteam.v13f-contexts/v1': ['docs/benchmark/v13f-dev/contexts/contexts.jsonl'],
        'dreamteam.alternatives-schedule/v2': ['docs/benchmark/v13f-dev/schedule-draft.json'],
        'dreamteam.alternatives-draw/v2': ['analysis/alternatives/v13f.py'],
        'dreamteam.alternatives-run/v2': ['analysis/alternatives/v13f.py'],
        'dreamteam.alternatives-analysis/v3': ['analysis/alternatives/v13f.py'],
        'dreamteam.v13f-fidelity/v1': ['analysis/alternatives/v13f_fidelity.py'],
        'dreamteam.review-package/v1': ['analysis/alternatives/v13f_package.py'],
    }
    proto = json.loads((ROOT / 'docs/benchmark/v13f-dev/protocol-draft.json').read_text())
    pins = json.loads((ROOT / 'docs/v13f/CONTRACT-PINS.json').read_text())
    return {
        'schema': 'dreamteam.contract-identities/v1',
        'notice': 'File hashes identify exact source bytes, not acceptance. Independent re-review: PENDING.',
        'parserVersionSha256': pins['parserVersionSha256'],
        'workflowSpecificationSha256': proto['workflow']['sha256'],
        'identities': {identity: {path: hashlib.sha256((ROOT/path).read_bytes()).hexdigest() for path in files}
                       for identity, files in paths.items()},
    }


def main():
    ap = argparse.ArgumentParser(__doc__)
    group = ap.add_mutually_exclusive_group(required=True)
    group.add_argument('--write', action='store_true')
    group.add_argument('--check', action='store_true')
    a = ap.parse_args()
    data = (json.dumps(identities(), indent=1, sort_keys=True)+'\n').encode()
    if a.write:
        DEST.write_bytes(data)
    elif DEST.read_bytes() != data:
        raise SystemExit('contract identity index differs')
    print('Successor contract identities: PASS')

if __name__ == '__main__':
    main()
