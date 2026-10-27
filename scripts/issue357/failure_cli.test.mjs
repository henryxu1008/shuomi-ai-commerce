import test from 'node:test';
import assert from 'node:assert/strict';
import {execFile} from 'node:child_process';
import {mkdtemp,readFile,stat} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join,resolve} from 'node:path';
import {fileURLToPath} from 'node:url';

const cli=fileURLToPath(new URL('../issue357-runtime.mjs',import.meta.url));
const preload=new URL('fixtures/failure_preload.mjs',import.meta.url).href;

for(const scenario of ['compose-cleanup-failure','compose-cleanup-success','build-cleanup-failure','compose-port-failure','evidence-write-failure']) {
 test(`launcher protects original failure evidence: ${scenario}`,{skip:process.platform!=='win32'},async()=>{
  const root=await mkdtemp(join(tmpdir(),'issue357-failure-test-'));
  const result=await new Promise(resolveResult=>execFile(process.execPath,['--import',preload,cli,'start','--current-application'],{
   windowsHide:true,timeout:15000,env:{...process.env,TEMP:root,TMP:root,TMPDIR:root,FAILURE_FIXTURE:scenario},
  },(error,stdout,stderr)=>resolveResult({code:error?.code??0,stdout,stderr})));
  const output=result.stdout+result.stderr;
  assert.equal(result.code,1,output);
  assert.match(output,/FIXTURE_LOADED/);
  assert.match(output,/FIXTURE_CLEANUP/,'existing cleanup still runs after failure');
  assert.doesNotMatch(output,/SYNTHETIC_PRIVATE|UNEXPECTED_COMMAND/);
  const directory=resolve(root,'task-processor-issue357',/runId=([a-f0-9-]{36})/.exec(output)?.[1]??'missing');
  const manifest=JSON.parse(await readFile(join(directory,'manifest.json'),'utf8'));
  assert.equal(manifest.failure,scenario.startsWith('build')?'PROCESS_FAILED:go:1':'PROCESS_FAILED:docker:1');
  assert.doesNotMatch(JSON.stringify(manifest),/SYNTHETIC_PRIVATE|privateOutput/);
  if(scenario==='evidence-write-failure') {
   assert.match(output,/FAILURE_EVIDENCE_INCOMPLETE/);
   assert.match(output,/CLEANUP_INCOMPLETE/);
   assert.match(result.stderr,/PROCESS_FAILED:docker:1/,'diagnostic I/O must not replace the original failure');
   return;
  }
  const file=join(directory,'failure-details.json');
  const evidence=JSON.parse(await readFile(file,'utf8'));
  assert.equal(evidence.runId,manifest.runId);
  assert.equal(evidence.sourceSha,manifest.sourceSha);
  const first=evidence.failures[0];
  assert.equal(first.phase,'initialize');
  assert.equal(first.stage,scenario.startsWith('build')?'compile-runtime':'docker-compose-up');
  assert.equal(first.error.privateOutput,'SYNTHETIC_PRIVATE_STDERR\n');
  assert.match(first.error.stack,/PROCESS_FAILED/);
  if(scenario.endsWith('cleanup-success')) {
   assert.equal(evidence.failures.length,1,'successful cleanup retains initialization diagnostics');
   assert.equal(manifest.status,'destroyed');
   assert.doesNotMatch(output,/CLEANUP_INCOMPLETE/);
  } else {
   assert.equal(evidence.failures.length,2);
   const second=evidence.failures[1];
   assert.equal(second.phase,'cleanup');
   if(scenario==='compose-port-failure') {
    assert.equal(second.stage,'verify-ports');
    assert.equal(second.error.code,'EADDRINUSE');
    assert.equal(second.error.syscall,'listen');
    assert.equal(second.error.address,'127.0.0.1');
    assert.equal(second.error.port,12001);
    assert.equal(second.error.message,'SYNTHETIC_PRIVATE_PORT_FAILURE');
   } else {
    assert.equal(second.stage,'unlink-ui-dependencies');
    assert.equal(second.error.message,'SYNTHETIC_PRIVATE_UNLINK_FAILURE');
    assert.equal(second.error.code,'EPERM');
    assert.equal(second.error.errno,-4048);
    assert.equal(second.error.syscall,'unlink');
    assert.equal(second.error.path,join(directory,'ui','node_modules'));
    assert.match(second.error.stack,/SYNTHETIC_PRIVATE_UNLINK_FAILURE/);
   }
  }
  const receipt=JSON.parse(await readFile(join(root,'protection-receipt.json'),'utf8'));
  assert.equal(receipt.writes.length,evidence.failures.length);
  assert.ok(receipt.writes.every(write=>write.mode===0o600&&write.protectedDirectory));
  if(process.platform!=='win32')assert.equal((await stat(file)).mode&0o777,0o600);
 });
}

for(const scenario of ['snapshot-cleanup-success','snapshot-evidence-write-failure']) {
 test(`seeded cleanup retains the snapshot failure: ${scenario}`,{skip:process.platform!=='win32'},async()=>{
  const root=await mkdtemp(join(tmpdir(),'issue357-snapshot-test-'));
  const id='12345678-1234-4234-8234-123456789abc';
  const result=await new Promise(resolveResult=>execFile(process.execPath,['--import',preload,cli,'destroy','--run',id],{
   windowsHide:true,timeout:15000,env:{...process.env,TEMP:root,TMP:root,TMPDIR:root,FAILURE_FIXTURE:scenario},
  },(error,stdout,stderr)=>resolveResult({code:error?.code??0,stdout,stderr})));
  const output=result.stdout+result.stderr;
  assert.equal(result.code,1,output);
  assert.match(output,/FIXTURE_CLEANUP/,'snapshot failure must not stop the existing cleanup path');
  assert.doesNotMatch(output,/SYNTHETIC_PRIVATE|UNEXPECTED_COMMAND/);
  const directory=join(root,'task-processor-issue357',id);
  const audit=JSON.parse(await readFile(join(directory,'cleanup.json'),'utf8'));
  assert.equal(audit.passed,false);
  assert.deepEqual(audit.zeroWrite,{passed:false,reason:'snapshot_failed'});
  assert.equal(JSON.parse(await readFile(join(directory,'manifest.json'),'utf8')).status,'destroyed');
  if(scenario.endsWith('evidence-write-failure')) {
   assert.match(output,/FAILURE_EVIDENCE_INCOMPLETE/);
  } else {
   const evidence=JSON.parse(await readFile(join(directory,'failure-details.json'),'utf8'));
   assert.equal(evidence.failures.length,1);
   assert.equal(evidence.failures[0].phase,'cleanup');
   assert.equal(evidence.failures[0].stage,'snapshot');
   assert.match(evidence.failures[0].error.message,/PROCESS_FAILED/);
   assert.equal(evidence.failures[0].error.privateOutput,'SYNTHETIC_PRIVATE_SNAPSHOT_STDERR\n');
  }
 });
}
