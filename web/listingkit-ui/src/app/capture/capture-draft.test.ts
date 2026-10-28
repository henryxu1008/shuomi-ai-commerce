import { expect, it, vi, afterEach } from 'vitest';
import { browserCaptureFixture } from '@/lib/contracts/browser-capture.fixture';
import { browserCaptureSchema } from '@/lib/contracts/browser-capture';
import { CaptureDraftClient } from './capture-draft';
const entry = { kind:'handoff' as const, extensionId:'a'.repeat(32),handoffId:'11111111-1111-4111-8111-111111111111',key:'22222222-2222-4222-8222-222222222222' };
const scope={userId:'actor',organizationId:'org',memberId:'member'};
afterEach(()=>vi.unstubAllGlobals());
it('reads minimal metadata before requesting product content and carries the exact verified scope',async()=>{
  const sendMessage=vi.fn((_id,m,cb)=>cb(m.type==='capture.meta'?{version:2,type:'capture.metadata',phase:'ready',idempotencyKey:entry.key}:{version:2,type:'capture.draft',phase:'ready',canSubmit:true,idempotencyKey:entry.key,payload:browserCaptureFixture()}));
  vi.stubGlobal('chrome',{runtime:{sendMessage}});
  const client=new CaptureDraftClient(entry,scope,new AbortController().signal,()=>true);
  expect((await client.read(browserCaptureSchema)).payload).toEqual(browserCaptureFixture());
  expect(sendMessage.mock.calls.map(c=>c[1].type)).toEqual(['capture.meta','capture.read']);
  expect(sendMessage.mock.calls.every(c=>JSON.stringify(c[1].scope)===JSON.stringify(scope))).toBe(true);
});
it('discards a late product body after the trusted context changes',async()=>{
  let current=true,finish!:(value:unknown)=>void;
  vi.stubGlobal('chrome',{runtime:{sendMessage:(_id:string,m:{type:string},cb:(v:unknown)=>void)=>{if(m.type==='capture.meta')cb({version:2,type:'capture.metadata',phase:'ready',idempotencyKey:entry.key});else finish=cb;}}});
  const client=new CaptureDraftClient(entry,scope,new AbortController().signal,()=>current);
  const pending=client.read(browserCaptureSchema);await vi.waitFor(()=>expect(finish).toBeTypeOf('function'));
  current=false;finish({version:2,type:'capture.draft',phase:'ready',canSubmit:true,idempotencyKey:entry.key,payload:browserCaptureFixture()});
  await expect(pending).rejects.toThrow('IDENTITY_CONTEXT_CHANGED');
});
