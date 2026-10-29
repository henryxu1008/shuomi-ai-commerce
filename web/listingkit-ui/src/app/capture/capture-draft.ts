import { z } from 'zod';
import type { CaptureHandoff } from './1688/capture-handoff';
export type CaptureDraftScope = Readonly<{ userId:string; organizationId:string; memberId:string }>;
const phase = z.enum(['capturing','paused','ready','bound','sending','unknown','saved_reported','rejected']);
const intent = z.strictObject({body:z.string().max(2*1024*1024),digest:z.string().regex(/^[a-f0-9]{64}$/),bytes:z.number().int().min(1).max(2*1024*1024)});
export type CaptureDraft<T> = { payload:T; phase:z.infer<typeof phase>; canSubmit:boolean;intent?:z.infer<typeof intent> };
export const captureHandoffExpiredMessage='交接已过期（CAPTURE_EXPIRED）。请打开原商品采集插件的「本机保全资料」，找到本次商品并点击「恢复原操作」。恢复沿用原资料和原键，无需重新采集；恢复不会自动保存，请按原操作状态核实或确认。';
export const isCaptureHandoffExpired=(error:unknown)=>error instanceof Error&&error.message==='CAPTURE_EXPIRED';
export class CaptureDraftClient {
  constructor(readonly entry:CaptureHandoff,readonly scope:CaptureDraftScope,private readonly signal:AbortSignal,private readonly stillCurrent:()=>boolean) {}
  private check() { this.signal.throwIfAborted(); if(!this.stillCurrent())throw Error('IDENTITY_CONTEXT_CHANGED'); }
  private async request(type:string,extra:Record<string,unknown>={}) {
    this.check();
    const response = await new Promise<unknown>((resolve,reject)=>{
      type Runtime={lastError?:unknown;sendMessage:(id:string,m:unknown,callback:(response:unknown)=>void)=>void};
      const runtime=(globalThis as typeof globalThis&{chrome?:{runtime?:Runtime}}).chrome?.runtime;
      if(!runtime){reject(Error('DRAFT_UNAVAILABLE'));return;}
      let settled=false;
      const finish=(ok:boolean,value?:unknown)=>{if(settled)return;settled=true;clearTimeout(timer);this.signal.removeEventListener('abort',abort);if(ok)resolve(value);else reject(Error('DRAFT_UNAVAILABLE'));};
      const abort=()=>finish(false);const timer=setTimeout(abort,10000);this.signal.addEventListener('abort',abort,{once:true});
      try{runtime.sendMessage(this.entry.extensionId,{version:2,type,handoffId:this.entry.handoffId,idempotencyKey:this.entry.key,scope:this.scope,...extra},value=>finish(!runtime.lastError,value));}catch{finish(false);}
    });
    this.check();
    const error=z.object({version:z.literal(2),code:z.string()}).strict().safeParse(response);
    if(error.success)throw Error(['DRAFT_CHANGED','DRAFT_SCOPE_CHANGED','DRAFT_BODY_MISMATCH','DRAFT_STORAGE_FULL','DRAFT_CORRUPT','CAPTURE_EXPIRED'].includes(error.data.code)?error.data.code:'DRAFT_UNAVAILABLE');
    return response;
  }
  private metadata(value:unknown) { return z.strictObject({version:z.literal(2),type:z.literal('capture.metadata'),phase,idempotencyKey:z.literal(this.entry.key)}).parse(value); }
  async read<T>(schema:z.ZodType<T>):Promise<CaptureDraft<T>> {
    for(let attempt=0;;attempt++){
      try{this.metadata(await this.request('capture.meta'));break;}catch(e){if(attempt===2||!(e instanceof Error)||e.message!=='DRAFT_UNAVAILABLE')throw e;await new Promise(r=>setTimeout(r,250));}
    }
    const value=z.strictObject({version:z.literal(2),type:z.literal('capture.draft'),phase,canSubmit:z.boolean(),idempotencyKey:z.literal(this.entry.key),payload:schema,intent:intent.optional()}).parse(await this.request('capture.read'));
    if(value.intent){
      const bytes=new TextEncoder().encode(value.intent.body);
      const digest=Array.from(new Uint8Array(await crypto.subtle.digest('SHA-256',bytes)),b=>b.toString(16).padStart(2,'0')).join('');
      if(bytes.length!==value.intent.bytes||digest!==value.intent.digest)throw Error('DRAFT_CORRUPT');
    }
    this.check();return value;
  }
  async bind(body:string) { return this.metadata(await this.request('capture.bind',{body})); }
  async send() { return this.metadata(await this.request('capture.send')); }
  async report(status:'unknown'|'rejected'|'saved_reported',operationId?:string,code?:string) { return this.metadata(await this.request('capture.status',{phase:status,...(operationId?{operationId}:{}),...(code?{code}:{})})); }
  async armRetry() { return this.metadata(await this.request('capture.armRetry')); }
  async retry() { return this.metadata(await this.request('capture.retry')); }
}
