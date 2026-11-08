"""Run-level failure regressions. All responses are scripted; HTTP is loopback only."""
import copy
import json
import os
from pathlib import Path
import sys
import tempfile
import threading
import time
import unittest
from unittest import mock
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

sys.path.insert(0, str(Path(__file__).resolve().parent))
import v13h as D
import v13g as G
import v13g_fidelity as F

BINARY = os.environ.get('DREAMTEAM_BINARY')


class HealthEvidence(unittest.TestCase):
    def test_error_shapes_are_closed(self):
        for health in ({'error':[]},{'error':{}},{'error':None},{'error':'Unknown'},
                       {'error':'URLError','status':'healthy'},{'error':'URLError','extra':True}):
            self.assertIsNone(D.health_failure(health),health)
        self.assertIsNone(D.health_failure({'status':'healthy'}))
        self.assertEqual(D.health_failure({'error':'URLError'}),'READ_FAILED')
        self.assertEqual(D.health_failure({'status':'unhealthy'}),'UNHEALTHY')


@unittest.skipUnless(BINARY, 'DREAMTEAM_BINARY is not set')
class Lifecycle(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        rows, h = D.load_contexts(str(Path(D.ROOT)/'docs/benchmark/v13g-dev/contexts'))
        cls.proto, _ = D.draft_protocol(rows, h)
        cls.ctx = D.Contexts(cls.proto, [r for r in rows if r['kind']=='constructed'], None,
                            {D.SB.SETTINGS_ENV:D.SB.REPLAY_URL})

    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(prefix='v13h-')
        self.addCleanup(self.tmp.cleanup)
        self.out = Path(self.tmp.name)
        self.sched = [{'seq':i,'model':D.QWEN,'context':'S1-a','arm':'control','draw':i} for i in (1,2)]

    def run_case(self, state=None, opening=None, closing=None, transport=None, module=D):
        pool=self.ctx.opener(BINARY)
        class Pool:
            def __call__(inner,c,arm):
                if opening: raise opening
                return pool(c,arm)
            def close(inner):
                pool.close()
                if closing: raise closing
        doc=module.run_schedule(self.proto,'scripted',self.sched,self.ctx,Pool(),
                               {D.QWEN:transport or D.scripted([F.ABANDON]*3)},str(self.out),'scripted test',
                               state_fn=state,now=lambda:0.,utc=lambda:'2026-01-01T00:00:00Z')
        rows=D.SB.load_jsonl(str(self.out/'live/draws.jsonl')) if (self.out/'live/draws.jsonl').exists() else []
        ap=self.out/'live/aborted-draw.json'
        aborted=json.loads(ap.read_text()) if ap.exists() else None
        return doc,rows,aborted

    def verify(self, result):
        doc,rows,aborted=result
        D.check_run_doc(doc,rows,self.sched,'scripted',self.proto,aborted)
        D.check_stop_artifact(str(self.out/'live'),doc)
        return D.analyze(BINARY,{},self.ctx,self.proto,self.sched,rows,doc,'scripted',aborted)

    def health_at(self, n, health=None, failure=None):
        count=[0]
        def state():
            count[0]+=1
            # Manifest must already exist, even at the first health check.
            doc=json.loads((self.out/'live/run.json').read_text())
            self.assertEqual(doc['status'],'PARTIAL')
            if count[0]==n:
                if failure: raise failure
                return {'health':health}
            return {'health':{'status':'healthy'}}
        return state

    def test_01_server_unavailable_before_run(self):
        r=self.run_case(state=self.health_at(1,{'error':'URLError'}))
        self.assertEqual((r[0]['drawsCompleted'],r[0]['modelCalls']),(0,0))
        self.assertEqual(self.verify(r)['failureVerification']['status'],'UNVERIFIED')

    def test_02_server_unavailable_mid_run(self):
        r=self.run_case(state=self.health_at(3,{'error':'URLError'}))
        self.assertEqual((r[0]['drawsCompleted'],r[0]['modelCalls']),(1,1))
        self.verify(r)

    def http_health(self, body=b'{}', delay=0):
        class Handler(BaseHTTPRequestHandler):
            def do_GET(inner):
                time.sleep(delay)
                inner.send_response(200);inner.end_headers()
                try: inner.wfile.write(body)
                except (BrokenPipeError, ConnectionResetError): pass
            def log_message(self,*args): pass
        server=ThreadingHTTPServer(('localhost',0),Handler)
        thread=threading.Thread(target=server.serve_forever,daemon=True);thread.start()
        try: return D.SB.RR.get('http://localhost:'+str(server.server_port),'/health',timeout=.05)
        finally: server.shutdown();server.server_close();thread.join()

    def test_03_health_timeout(self):
        health=self.http_health(delay=.15)
        self.assertEqual(health,{'error':'TimeoutError'})
        self.verify(self.run_case(state=self.health_at(1,health)))

    def test_04_malformed_health_response(self):
        health=self.http_health(b'{')
        self.assertEqual(health,{'error':'JSONDecodeError'})
        self.verify(self.run_case(state=self.health_at(1,health)))
        self.assertEqual(D.health_failure([]),'MALFORMED')
        self.assertEqual(D.health_failure({}),'MALFORMED')

    def test_05_unhealthy_response(self):
        health=self.http_health(b'{"status":"unhealthy"}')
        self.verify(self.run_case(state=self.health_at(1,health)))

    def test_06_first_health_interruption(self):
        r=self.run_case(state=self.health_at(1,failure=KeyboardInterrupt('first health')))
        self.assertEqual(r[0]['failureEvent']['stage'],'initial_health')
        self.assertIsNone(r[2]);self.verify(r)

    def test_07_later_health_interruption(self):
        r=self.run_case(state=self.health_at(3,failure=KeyboardInterrupt('later health')))
        self.assertEqual(r[0]['drawsCompleted'],1)
        self.assertEqual(r[0]['stopped']['code'],'operator');self.verify(r)

    def test_08_engine_open_failure(self):
        r=self.run_case(opening=RuntimeError('engine open'))
        self.assertEqual(r[2]['failureStage'],'engine_open');self.verify(r)

    def test_09_draw_level_failure(self):
        def fail(_):raise RuntimeError('draw failure')
        r=self.run_case(transport=fail)
        self.assertEqual(r[2]['failure']['message'],'draw failure');self.verify(r)

    def test_10_cleanup_failure(self):
        r=self.run_case(closing=KeyboardInterrupt('cleanup'))
        self.assertEqual((r[0]['drawsCompleted'],r[0]['status']),(2,'PARTIAL'))
        self.verify(r)

    def test_11_primary_plus_cleanup_failure(self):
        r=self.run_case(opening=RuntimeError('primary'),closing=KeyboardInterrupt('cleanup'))
        self.assertEqual(r[0]['failureEvent']['failure']['message'],'primary')
        self.assertEqual(r[0]['cleanup']['failure']['message'],'cleanup');self.verify(r)

    def test_12_missing_manifest(self):
        self.run_case(state=self.health_at(1,{'error':'URLError'}))
        (self.out/'live/run.json').unlink()
        with self.assertRaises(FileNotFoundError):D.load_run(str(self.out))

    def test_13_truncated_manifest(self):
        self.run_case(state=self.health_at(1,{'error':'URLError'}))
        (self.out/'live/run.json').write_text('{')
        with self.assertRaises(json.JSONDecodeError):D.load_run(str(self.out))

    def test_14_failed_atomic_replacement(self):
        path=str(self.out/'run.json');D.atomic_json(path,{'status':'PARTIAL'})
        for target,error in (('replace',OSError('replace failed')),('fsync',KeyboardInterrupt('interrupted write'))):
            with mock.patch.object(D.os,target,side_effect=error):
                with self.assertRaises(type(error)):D.atomic_json(path,{'status':'COMPLETE'})
            self.assertEqual(json.loads(Path(path).read_text()),{'status':'PARTIAL'})
        self.assertEqual(list(self.out.iterdir()),[Path(path)])

    def mutate(self, fn, opening=None):
        r=self.run_case(state=None if opening else self.health_at(1,{'error':'URLError'}),opening=opening)
        self.verify(r)
        bad=list(copy.deepcopy(r));fn(bad)
        with self.assertRaises(D.ExperimentError):D.check_run_doc(bad[0],bad[1],self.sched,'scripted',self.proto,bad[2])

    def test_15_unknown_error_type(self):
        def change(r):
            for stop in (r[0]['stopped'],r[0]['failureEvent']['stop']):stop['state']['health']={'error':'InventedError'}
        self.mutate(change)
        self.assertIsNone(D.health_failure({'error':'URLError','status':'unhealthy'}))

    def test_16_invented_stop_code(self):self.mutate(lambda r:r[0]['stopped'].update(code='invented'))
    def test_17_wrong_event_stage(self):self.mutate(lambda r:r[0]['failureEvent'].update(stage='draw'))
    def test_18_wrong_scheduled_draw(self):self.mutate(lambda r:r[0]['failureEvent']['nextDraw'].update(seq=999))
    def test_19_negative_draw_count(self):self.mutate(lambda r:r[0].update(drawsCompleted=-1))
    def test_20_impossible_call_count(self):self.mutate(lambda r:r[0].update(modelCalls=999))
    def test_21_missing_aborted_artifact(self):self.mutate(lambda r:r.__setitem__(2,None),RuntimeError('open'))
    def test_22_contradictory_cleanup_event(self):self.mutate(lambda r:r[0]['cleanup'].update(status='FAILED'))

    def test_23_genuine_partial_analysis(self):
        r=self.run_case(state=self.health_at(3,{'error':'URLError'}))
        a=self.verify(r)
        self.assertEqual((a['runStatus'],len(a['draws'])),('PARTIAL',1))

    def test_24_partial_comparison_suppression(self):
        a=self.verify(self.run_case(closing=RuntimeError('close')))
        self.assertEqual((a['armDifferences'],a['pairs']),([],[]))

    def test_25_exact_model_free_completed_replay(self):
        r=self.run_case()
        a=self.verify(r)
        self.assertEqual((a['runStatus'],a['completedVerification']),('COMPLETE','VERIFIED'))
        c=self.ctx.by_id['S1-a'];pool=self.ctx.opener(BINARY)
        try:
            for row in r[1]:
                D.replay_draw(row,c,self.ctx.settings[D.QWEN],pool,self.ctx.memory(c,D.QWEN))
                G.replay_draw(dict(row,schema=G.DRAW_SCHEMA),c,self.ctx.settings[D.QWEN],pool,self.ctx.memory(c,D.QWEN))
        finally:pool.close()

    def test_connection_refused_production_getter(self):
        server=ThreadingHTTPServer(('localhost',0),BaseHTTPRequestHandler)
        port=server.server_port;server.server_close()
        health=D.SB.RR.get('http://localhost:'+str(port),'/health',timeout=.05)
        self.assertEqual(health,{'error':'URLError'})
        self.verify(self.run_case(state=self.health_at(1,health)))

    def test_no_resume_and_unfinalized_rejected(self):
        r=self.run_case()
        with self.assertRaises(D.ExperimentError):self.run_case()
        r[0]['lifecycle']='IN_PROGRESS';r[0]['status']='PARTIAL'
        with self.assertRaises(D.ExperimentError):D.check_run_doc(r[0],r[1],self.sched,'scripted',self.proto)

    def test_event_artifact_required(self):
        r=self.run_case(state=self.health_at(1,{'error':'URLError'}))
        (self.out/'live/failure-event.json').unlink()
        with self.assertRaises(D.ExperimentError):D.check_stop_artifact(str(self.out/'live'),r[0])

    def test_preserved_protocol_and_schedule(self):
        rows,h=G.load_contexts(str(Path(D.ROOT)/'docs/benchmark/v13g-dev/contexts'))
        old,sched=G.draft_protocol(rows,h)
        new,current=D.draft_protocol(rows,h)
        self.assertEqual(sched,current)
        self.assertEqual((len(rows),len(current),new['maxModelCalls']),(21,262,802))
        ignore={'id','schema','driver'}
        self.assertEqual({k:v for k,v in old.items() if k not in ignore},
                         {k:v for k,v in new.items() if k not in ignore})
        self.assertIsNone(D.health_failure({'status':'healthy'}))

    def test_primary_survives_session_cleanup(self):
        pool=self.ctx.opener(BINARY)
        class Session:
            def __init__(inner,s):inner.s=s
            def request(inner,*args,**kwargs):return inner.s.request(*args,**kwargs)
            def close(inner):
                inner.s.close()
                raise RuntimeError('session cleanup')
        def opener(c,arm):return Session(pool(c,arm))
        def broken(_):raise ValueError('primary workflow')
        c=self.ctx.by_id['S1-a']
        try:
            with self.assertRaises(ValueError) as caught:
                D.run_draw(self.sched[0],c,self.ctx.settings[D.QWEN],opener,self.ctx.memory(c,D.QWEN),broken,None)
            r=caught.exception.partialDraw
            self.assertEqual(r['failure']['message'],'primary workflow')
            self.assertEqual(r['cleanupFailure']['message'],'session cleanup')
        finally:pool.close()

    def test_failed_checkpoint_stays_partial_and_propagates(self):
        replace=D.os.replace;calls=[0]
        def fail(source,target):
            if Path(target).name=='run.json':
                calls[0]+=1
                if calls[0]==2:raise OSError('checkpoint replacement')
            return replace(source,target)
        with mock.patch.object(D.os,'replace',side_effect=fail):
            with self.assertRaisesRegex(OSError,'checkpoint replacement'):
                self.run_case(state=lambda:{'health':{'status':'healthy'}},closing=KeyboardInterrupt('cleanup'))
        doc=json.loads((self.out/'live/run.json').read_text())
        self.assertEqual((doc['status'],doc['lifecycle']),('PARTIAL','IN_PROGRESS'))
        with self.assertRaises(D.ExperimentError):D.check_run_doc(doc,[],self.sched,'scripted',self.proto)

    def test_setup_exception_is_analyzable(self):
        with mock.patch.object(D,'Limits',side_effect=RuntimeError('limits initialization')):
            r=self.run_case()
        self.assertEqual(r[0]['failureEvent']['stage'],'run_setup')
        self.verify(r)

    def test_contradictory_side_artifacts_rejected(self):
        self.run_case(state=self.health_at(1,{'error':'URLError'}))
        D.load_run(str(self.out))
        export=self.out/'live/draws.jsonl'
        export.write_text('{"invented":true}\n')
        with self.assertRaises(D.ExperimentError):D.load_run(str(self.out))
        export.write_text('')
        before=self.out/'live/server-before.json'
        before.write_text('{"health":{"status":"healthy"}}')
        with self.assertRaises(D.ExperimentError):D.load_run(str(self.out))
        before.write_text('{"health":{"error":"URLError"}}')
        (self.out/'live/stop-event.json').write_text('{}')
        with self.assertRaises(D.ExperimentError):D.load_run(str(self.out))

    def test_no_boundary_stop_after_completed_schedule(self):
        doc,rows,aborted=self.run_case(closing=RuntimeError('cleanup'))
        for stage in ('initial_health','between_health','draw_guard','draw'):
            bad=copy.deepcopy(doc);bad['failureEvent']['stage']=stage
            with self.assertRaises(D.ExperimentError):D.check_run_doc(bad,rows,self.sched,'scripted',self.proto,aborted)
        # Removing a completed row cannot invent an early cleanup-only stop.
        bad=copy.deepcopy(doc);bad.update(drawsCompleted=1,drawsNotRun=1,modelCalls=1)
        bad['failureEvent'].update(completed=1,modelCalls=1,nextDraw=self.sched[1])
        with self.assertRaises(D.ExperimentError):D.check_run_doc(bad,rows[:1],self.sched,'scripted',self.proto)


if __name__=='__main__':unittest.main()
