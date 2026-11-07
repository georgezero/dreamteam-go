"""Pin final remediation contracts and verify all preserved predecessor bytes."""
import argparse
import hashlib
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
DEST = ROOT / 'docs/v13g'


def digest(path):
    return hashlib.sha256((ROOT / path).read_bytes()).hexdigest()


def artifacts():
    paths = {
        'dreamteam.phase-response-parser/v3': ['internal/alternatives/validation_v3.go', 'agents/llm/llm_agent_alternatives_v3.py'],
        'dreamteam.protocol-batch-decoder/export-1': ['internal/agent/batch_export.go'],
        'llm-agent-alternatives-workflow/v3': ['agents/llm/llm_agent_alternatives_v3.py', 'docs/v13g/WORKFLOW.md'],
        'dreamteam.execution-representation/v1': ['analysis/alternatives/v13g.py', 'docs/v13g/RECORDING.md'],
        'dreamteam.phase-proposal-record/v3': ['agents/llm/llm_agent_alternatives_v3.py'],
        'dreamteam.alternatives-decision-record/v3': ['agents/llm/llm_agent_alternatives_v3.py'],
        'llm-agent-phase-validation-first/v3': ['agents/llm/llm_agent_alternatives_v3.py'],
        'dreamteam.submitted-action/v2': ['internal/alternatives/submitted_v3.go'],
        'dreamteam.alternatives-reply/v3': ['internal/alternatives/ops_v3.go'],
        'dreamteam.alternatives-outcome/v3': ['internal/alternatives/validation_v3.go'],
        'dreamteam.alternatives-outcomes/v3': ['docs/v13g/OUTCOME_DEFINITIONS.md'],
        'dreamteam-v13g-driver/3': ['analysis/alternatives/v13g.py', 'docs/v13g/DRIVER.md'],
        'dreamteam.alternatives-draw/v3': ['analysis/alternatives/v13g.py'],
        'dreamteam.alternatives-run/v3': ['analysis/alternatives/v13g.py'],
        'dreamteam.alternatives-analysis/v4': ['analysis/alternatives/v13g.py'],
        'dreamteam.alternatives-experiment/v3': ['docs/benchmark/v13g-dev/protocol-draft.json', 'docs/v13g/PROTOCOL_DRAFT.md'],
        'dreamteam.alternatives-schedule/v3': ['docs/benchmark/v13g-dev/schedule-draft.json'],
        'dreamteam.v13g-contexts/v1': ['docs/benchmark/v13g-dev/contexts/contexts.jsonl'],
        'dreamteam.alternatives-partial-run/v3': ['docs/v13g/PARTIAL_RUN_PROCEDURE.md', 'analysis/alternatives/v13g.py'],
        'dreamteam.between-draw-stop/v1': ['analysis/alternatives/v13g.py'],
        'dreamteam.v13g-fidelity/v1': ['analysis/alternatives/v13g_fidelity.py'],
        'dreamteam.review-package/v2': ['analysis/alternatives/v13g_package.py'],
        'llm-agent-alternatives-system/v2': ['agents/llm/text/alternatives-system-v2.txt'],
        'dreamteam.phase-response-schemas/v2': ['agents/llm/text/response-schemas-v2.json'],
        'dreamteam.alternatives-presentation/v3': ['internal/alternatives/text/presentation-v3.tmpl', 'docs/v13g/PRESENTATION_V3.md'],
        'dreamteam.alternatives-view/v2': ['docs/benchmark/v13g-dev/presentation/view-table.txt', 'internal/alternatives/view_v2.go'],
    }
    files = sorted(set(p for ps in paths.values() for p in ps) | {
        'agents/llm/text/initial-v2.txt', 'agents/llm/text/correction-v2.txt',
        'docs/v13g/KNOWN_LIMITATIONS.md', 'docs/v13g/FIDELITY.md', 'docs/v13g/RUN_PROCEDURE.md'})
    parser = 'dreamteam.phase-response-parser/v3'
    pins = {'schema':'dreamteam.remediation-contract-pins/v2', 'parserVersion':parser,
            'parserVersionSha256':hashlib.sha256(parser.encode()).hexdigest(), 'files':{p:digest(p) for p in files}}
    proto = json.loads((ROOT/'docs/benchmark/v13g-dev/protocol-draft.json').read_text())
    index = {'schema':'dreamteam.contract-identities/v2', 'notice':'Independent re-review PENDING; human review PENDING.',
             'parserVersionSha256':pins['parserVersionSha256'], 'workflowSpecificationSha256':proto['workflow']['sha256'],
             'identities':{identity:{p:digest(p) for p in ps} for identity,ps in paths.items()}}
    index['identityHashEncoding']='SHA-256 of UTF-8 JSON {identity,files}, sorted keys, compact separators'
    index['identitySha256']={identity:hashlib.sha256(json.dumps({'identity':identity,'files':files},sort_keys=True,separators=(',',':')).encode()).hexdigest()
                             for identity,files in index['identities'].items()}
    return {'CONTRACT-PINS.json':pins, 'CONTRACT-IDENTITIES.json':index}


def main():
    p = argparse.ArgumentParser(__doc__)
    g = p.add_mutually_exclusive_group(required=True)
    g.add_argument('--write',action='store_true')
    g.add_argument('--check',action='store_true')
    a = p.parse_args()
    for name,obj in artifacts().items():
        data=(json.dumps(obj,indent=1,sort_keys=True)+'\n').encode()
        if a.write:(DEST/name).write_bytes(data)
        elif (DEST/name).read_bytes()!=data:raise SystemExit('contract mismatch: '+name)
    for line in (DEST/'BASELINE-HASHES.sha256').read_text().splitlines():
        want,path=line.split('  ',1)
        if digest(path)!=want:raise SystemExit('preserved bytes changed: '+path)
    print('Successor identities and preserved bytes: PASS')


if __name__=='__main__':main()
