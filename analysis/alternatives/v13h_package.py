"""Build the acceptance-review material from complete delivered scripted interactions."""
import argparse
from collections import Counter
import ast
import hashlib
import json
from pathlib import Path

ROOT=Path(__file__).resolve().parents[2]
DEST=ROOT/'docs/v13h/review-package'
DEFAULT=ROOT/'docs/benchmark/v13h-dev/fidelity'
PACKAGE_SCHEMA='dreamteam.review-package/v3'


def artifacts(source):
    out={}
    index=[]
    paths=sorted((source/'interactions').glob('*.json'))
    if not paths:raise ValueError('no completed delivered interactions')
    required=set()
    for path in paths:
        doc=json.loads(path.read_text())
        rec=doc['record']
        # Only actual delivered messages and actual scripted replies enter the
        # model-visible package. Engine records, identities, labels and outcomes
        # remain in the separate record-only index and evidence directory.
        deliveries=[]
        for call in rec['deliveries']:
            deliveries.append({'messages':call['messages'],'scriptedResponse':call['content'],
                               'transportStatus':call['status'],'transportError':call['error']})
        name='transcripts/'+path.name
        out[name]=(json.dumps({'notice':'SCRIPTED-MODEL OUTPUT: complete delivered requests and scripted responses',
                               'deliveries':deliveries},indent=1,ensure_ascii=False)+'\n').encode()
        required.add((doc['context'],doc['arm']))
        index.append({'transcript':name,'case':doc['case'],'context':doc['context'],'arm':doc['arm'],
                      'result':rec['result'],'exposure':rec['exposure'],
                      'sourceSha256':hashlib.sha256(path.read_bytes()).hexdigest()})
    contexts=[json.loads(line) for line in (ROOT/'docs/benchmark/v13g-dev/contexts/contexts.jsonl').read_text().splitlines()]
    for c in contexts:
        for arm in ('control','treatment'):
            if (c['id'],arm) not in required:raise ValueError('missing complete context interaction: '+c['id']+' '+arm)
    active=['agents/llm/text/alternatives-system-v2.txt','agents/llm/text/initial-v2.txt',
            'agents/llm/text/correction-v2.txt','agents/llm/text/response-schemas-v2.json',
            'internal/alternatives/text/presentation-v3.tmpl','docs/benchmark/v13g-dev/presentation/view-table.txt',
            'internal/alternatives/text/selection-v2.txt']
    legacy=['internal/alternatives/text/control-v1.tmpl','internal/alternatives/text/presentation-v1.tmpl',
            'internal/alternatives/text/presentation-v2.tmpl','internal/alternatives/text/selection-v1.txt']
    sources=[]
    for status,files in [('ACTIVE',active),('LEGACY-INACTIVE',legacy)]:
        for f in files:
            target='sources/'+status+'/'+Path(f).name
            out[target]=(ROOT/f).read_bytes()
            sources.append({'status':status,'source':f,'packagePath':target,
                            'legacyInactiveSections':['rows labelled LEGACY-INACTIVE (internal binding only)'] if f.endswith('view-table.txt') else []})
    legacy_agent=ROOT/'agents/llm/llm_agent3.py'
    assignment=next(node for node in ast.parse(legacy_agent.read_text()).body
                    if isinstance(node,ast.Assign) and any(isinstance(t,ast.Name) and t.id=='SYSTEM_PROMPT' for t in node.targets))
    target='sources/LEGACY-INACTIVE/system-prompt-v3.txt'
    out[target]=ast.literal_eval(assignment.value).encode()
    sources.append({'status':'LEGACY-INACTIVE','source':'agents/llm/llm_agent3.py#SYSTEM_PROMPT',
                    'packagePath':target})
    out['RECORD_ONLY_INDEX.json']=(json.dumps({'schema':PACKAGE_SCHEMA,'notice':'RECORD ONLY: never appended to model-visible messages',
        'interactions':index,'sources':sources},indent=1,sort_keys=True)+'\n').encode()
    items=['neutrality','readability','financial terminology','target identification','continuing obligations and no action',
           'bookings displaced','slots closed and replacement capacity','missing information','withdrawal and reversibility',
           'potentially misleading comparisons','information boundary','declared treatment bundle and complete interactions']
    questionnaire=['# Reviewer questionnaire\n','HUMAN REVIEW PENDING\n']
    for field in ('Reviewer name','Reviewer identity','Independence declaration','Date','Written findings'):
        questionnaire.append(field+': PENDING\n')
    for i,item in enumerate(items,1):questionnaire.append(f'{i}. {item}\n\nResponse: PENDING\n')
    out['QUESTIONNAIRE.md']='\n'.join(questionnaire).encode()
    out['README.md']=b'''# V0.13h human presentation review package

HUMAN REVIEW PENDING. Independent re-review: PENDING.

Every transcript is labelled SCRIPTED-MODEL OUTPUT. Its deliveries contain
complete model-visible request messages and the actual scripted response,
including the initial system instructions, public context and proposal.
Numeric spelling and 64/65-action validation paths are included.
Read FAILURE_FEEDBACK_DISCLOSURE.md and the added initial invalid-number interaction.
RECORD_ONLY_LIFECYCLE adds health-read, interruption and cleanup examples.
RECORD_ONLY_PARTIALS contains actual scripted initialization-failure and
between-draw health-stop examples with analysis; none is a completed draw.
Repeated requests on transport retry are retained as delivered. A null
response marks an actual scripted transport failure. No response or reviewer
finding has been invented. Adversarial scripts can contain invented attack
strings; they are supplied test inputs, not ground truth or evaluation results.

The package covers both arms at all nine historical contexts and all twelve
sensitivity members, plus validation correction, correction failure, deliberate
empty revision, selection, abandonment, combined inspection, mixed operations
and staffing, partial termination, roster changes, unsupported mixed capacity,
retry exhaustion and refusal paths. The scripted initial proposals are replayed;
they were not newly sampled under the successor system instruction.

Start with the S1-a treatment interaction in RECORD_ONLY_INDEX.json. Inspect
proposal and no_action together: 2,481 closed slots and zero displaced bookings
are distinct quantities. Continuing guarantees and conditional severance are
shown against the same no-action baseline for each employee. Exposure is not
payable-now severance. Capacity is not demand. Then read the multi-employee,
roster, mixed staffing and limited-history teleradiology interactions.

Sources are labelled ACTIVE or LEGACY-INACTIVE. The legacy source files, including the original built-in system instruction,
are reference material, not additional messages delivered by this workflow.
Selection v2 remains ACTIVE with its unchanged historical contract identity. The successor view inherits
unchanged item grammar where used; runtime replaces legacy reversibility,
capacity, retained guarantees and fee-proxy fragments with the typed sections.

Evaluation-only context labels, outcome fields, ground truth and protocol values
remain outside model-visible messages, in RECORD_ONLY_INDEX.json and the evidence
directory. They are not appended to a conversation. The transcript filenames and
this guide are reviewer navigation metadata.

Complete QUESTIONNAIRE.md in a separate review response, with identity,
independence declaration, date and written findings. Judge understandability and
balance as well as coverage of the complete interaction. No acceptance is
requested for the historical V0.13e package. All reviewer fields remain PENDING.

Regenerate with analysis/alternatives/v13h_package.py --write; verify with --check.
SHA256SUMS covers every included file except itself.
'''
    for path in sorted((source/'partials').rglob('*.json')):
        out['RECORD_ONLY_PARTIALS/'+str(path.relative_to(source/'partials'))]=path.read_bytes()
    if not any(name.startswith('RECORD_ONLY_PARTIALS/') for name in out):raise ValueError('missing partial-run examples')
    for path in sorted((source/'lifecycle-examples').rglob('*.json')):
        out['RECORD_ONLY_LIFECYCLE/'+str(path.relative_to(source/'lifecycle-examples'))]=path.read_bytes()
    old=ROOT/'docs/v13g/review-package/transcripts'
    unchanged=Counter(p.read_bytes() for p in old.glob('*.json'))
    current=Counter(data for name,data in out.items() if name.startswith('transcripts/'))
    if unchanged-current:raise ValueError('a predecessor delivered transcript changed or is missing')
    out['FAILURE_FEEDBACK_DISCLOSURE.md']=b'''# Invalid-number feedback disclosure

Human review PENDING. The seven named static active sources retain their bytes.
Generated invalid-number feedback changed in the predecessor numeric fix.
The successor adds an initial minutes:30.1 scripted proposal and records the
complete delivered retry interaction. Its actual delivered user message is:

> Your reply could not be used: action number must be an in-range integer. Reply again with one JSON object {"reason": ..., "actions": [...]} that follows the action fields, targets and dates exactly, or with "actions": [] for no change.

The fixed 21-context schedule has not been shown to deliver this changed
initial-parser diagnostic. This is a general failure-path disclosure, not an
additional experimental draw. Numeric validation and agent text are unchanged.
All 92 predecessor delivered transcripts and all static sources are preserved
byte for byte as package members; the additional interaction changes the exact
human-review package. Human review of this exact successor package is required.
Lifecycle examples use explicitly synthetic identities and timestamps for
reproducibility. External causes remain UNVERIFIED; examples are record-only.
'''
    out['SHA256SUMS']=''.join(hashlib.sha256(data).hexdigest()+'  '+name+'\n' for name,data in sorted(out.items())).encode()
    return out


def main(argv=None):
    ap=argparse.ArgumentParser(__doc__)
    ap.add_argument('--source',type=Path,default=DEFAULT)
    group=ap.add_mutually_exclusive_group(required=True)
    group.add_argument('--write',action='store_true')
    group.add_argument('--check',action='store_true')
    a=ap.parse_args(argv)
    expected=artifacts(a.source)
    if a.write:
        for name,data in expected.items():
            p=DEST/name;p.parent.mkdir(parents=True,exist_ok=True);p.write_bytes(data)
    else:
        actual={str(p.relative_to(DEST)) for p in DEST.rglob('*') if p.is_file()}
        if actual!=set(expected):raise ValueError('package file set differs')
        for name,data in expected.items():
            if (DEST/name).read_bytes()!=data:raise ValueError('package bytes differ: '+name)
    print(f'Package: {len(expected)} files; HUMAN REVIEW PENDING; independent re-review PENDING')

if __name__=='__main__':main()
