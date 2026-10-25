import { z } from 'zod';
import { FamilyLocation, familyURL } from './family-location';
import type { CaptureTab, Controller } from '../controller';
import type { ReturnSession } from '../return-session';
import { UUID } from '../validation';
import { validateAmazonCapture } from './validation';
import type { FamilySummary } from './family-coordinator';

const common={revision:z.string().regex(UUID)};
const message=z.discriminatedUnion('type',[
  z.strictObject({...common,type:z.literal('family.init')}),
  z.strictObject({...common,type:z.literal('family.cancel')}),
  z.strictObject({...common,type:z.literal('family.closeReader')}),
  z.strictObject({...common,type:z.literal('family.return')}),
  z.strictObject({...common,type:z.literal('family.reader'),tabId:z.number().int().min(0),sourceURL:z.string().max(2048)}),
  z.strictObject({...common,type:z.literal('family.finish'),payload:z.unknown()}),
]);
const summary=z.strictObject({phase:z.enum(['initializing','collecting','paused','transferring','transferred','uncertain','cancelled','interrupted']),observed:z.number().int().min(0).max(256),total:z.number().int().min(0).max(256),reason:z.string().max(100)});
export type FamilyStatus=z.infer<typeof summary>;
export class FamilyBridge {
  private opening?:Promise<void>;
  private finishing=false;
  private attempted=false;
  private queue:Promise<unknown>=Promise.resolve();
  private serialized<T>(action:()=>Promise<T>):Promise<T>{const next=this.queue.then(action);this.queue=next.catch(()=>{});return next;}
  readonly location=new FamilyLocation(chrome.storage?.session);
  constructor(private readonly controller:()=>Controller,private readonly application:ReturnSession){}
  get ready(){return this.location.ready;}
  get active(){return this.location.unavailable || !!this.location.current;}
  get revision(){return this.location.unavailable?undefined:this.location.current?.revision;}
  get busy(){return !!this.opening || this.finishing;}
  private async contexts() {
    const r=this.location.current;if(!r)return [];
    return chrome.runtime.getContexts({contextTypes:['TAB'],documentUrls:[familyURL(r.revision)]});
  }
  async begin(tab:CaptureTab,revision:string) {
    if(this.opening || this.active)throw Error('ACTION_IN_PROGRESS');
    this.attempted=false;
    this.opening=this.serialized(async()=>{
      await this.location.prepare(tab,revision);
      const created=await chrome.tabs.create({url:familyURL(revision)});
      if(created.id===undefined)throw Error('FAMILY_INTERRUPTED');
      await this.location.noteTab(created.id);
    }).finally(()=>{this.opening=undefined;});
    return this.opening;
  }
  async status():Promise<FamilyStatus|undefined> {
    if(this.location.unavailable)return {phase:'uncertain',observed:0,total:0,reason:'RETURN_STATE_UNAVAILABLE'};
    const r=this.location.current;if(!r)return;
    const basic:FamilyStatus={phase:'initializing',observed:0,total:0,reason:''};
    try {
      const contexts=await this.contexts();
      if(contexts.length!==1){
        if(r.phase==='pending' && r.tabId!==undefined){try{const tab=await chrome.tabs.get(r.tabId);if(tab.url===familyURL(r.revision)||tab.pendingUrl===familyURL(r.revision))return basic;}catch{/* closed */}}
        return {...basic,phase:'interrupted',reason:'FAMILY_INTERRUPTED'};
      }
      if(r.phase==='bound' && (contexts[0].documentId!==r.documentId || contexts[0].tabId!==r.tabId))return {...basic,phase:'interrupted',reason:'FAMILY_INTERRUPTED'};
      if(r.phase==='pending')return basic;
      const value=await chrome.runtime.sendMessage({type:'family.inspect',revision:r.revision});
      return summary.parse(value);
    }catch{return {...basic,phase:'uncertain',reason:'FAMILY_INTERRUPTED'};}
  }
  async focus() {
    const r=this.location.current;if(!r)throw Error('FAMILY_INTERRUPTED');
    const contexts=await this.contexts();
    const context=contexts.length===1?contexts[0]:undefined;
    const id=context?.tabId??r.tabId;
    if(id===undefined || r.phase==='bound' && context?.documentId!==r.documentId)throw Error('FAMILY_INTERRUPTED');
    const tab=await chrome.tabs.get(id);
    if(tab.url!==familyURL(r.revision) && tab.pendingUrl!==familyURL(r.revision))throw Error('FAMILY_INTERRUPTED');
    await chrome.tabs.update(id,{active:true});if(tab.windowId!==undefined)await chrome.windows.update(tab.windowId,{focused:true});
  }
  private async sender(sender:chrome.runtime.MessageSender,revision:string) {
    const r=this.location.current;
    if(!r || r.revision!==revision || sender.id!==chrome.runtime.id || sender.url!==familyURL(revision) || !sender.documentId)throw Error('INVALID_MESSAGE');
    const contexts=await this.contexts();
    if(contexts.length!==1)throw Error('INVALID_MESSAGE');
    const c=contexts[0];
    if(c.documentId!==sender.documentId || c.documentUrl!==sender.url || c.tabId<0 || c.frameId!==0 || sender.tab && sender.tab.id!==c.tabId)throw Error('INVALID_MESSAGE');
    if(r.phase==='bound' && (r.tabId!==c.tabId || r.documentId!==c.documentId) || r.tabId!==undefined && r.tabId!==c.tabId)throw Error('ENTRY_CHANGED');
    return c;
  }
  handle(value:unknown,sender:chrome.runtime.MessageSender) {return this.serialized(()=>this.handleOne(value,sender));}
  private async handleOne(value:unknown,sender:chrome.runtime.MessageSender) {
    const m=message.parse(value);
    if(m.type==='family.return' && !this.active){
      if(this.application.state!=='available' || this.application.revision!==m.revision || sender.id!==chrome.runtime.id || sender.url!==familyURL(m.revision) || !sender.documentId)throw Error('INVALID_MESSAGE');
      const contexts=await chrome.runtime.getContexts({documentIds:[sender.documentId],contextTypes:['TAB']});
      if(contexts.length!==1 || contexts[0].documentUrl!==sender.url || contexts[0].frameId!==0)throw Error('INVALID_MESSAGE');
      await this.controller().returnToApplication();return {};
    }
    const context=await this.sender(sender,m.revision);
    if(m.type==='family.init'){
      await this.location.bind(context.tabId,context.documentId!);
      const r=this.location.current!;
      return {source:{id:r.sourceTabId,url:r.sourceURL,documentId:r.sourceDocumentId},coordinatorTabId:context.tabId,windowId:context.windowId};
    }
    const r=this.location.current;if(r?.phase!=='bound')throw Error('ENTRY_CHANGED');
    if(m.type==='family.reader'){
      const request=m as {tabId:number;sourceURL:string}; const tab=await chrome.tabs.get(request.tabId);
      if(tab.openerTabId!==r.tabId || r.reader && r.reader.tabId!==request.tabId
        || tab.url!=='about:blank' && tab.url!==r.reader?.sourceURL)throw Error('FAMILY_INTERRUPTED');
      await this.location.reader(request.tabId,request.sourceURL);return {};
    }
    if(m.type==='family.return'){if(this.application.state!=='available')throw Error('HANDOFF_UNAVAILABLE');await this.controller().returnToApplication();return {};}
    if(m.type==='family.closeReader'){await this.closeReader();return {};}
    if(m.type==='family.cancel'){
      if(this.finishing || this.application.state!=='none' || this.controller().isBusy())throw Error('HANDOFF_UNAVAILABLE');
      await this.closeReader();await this.location.clear();return {};
    }
    if(this.attempted || this.application.state!=='none')throw Error('HANDOFF_UNAVAILABLE');
    this.attempted=true;this.finishing=true;
    try {
    const capture=validateAmazonCapture((m as {payload:unknown}).payload);
    if(capture.evidence.sourceURL!==r.sourceURL || capture.evidence.observation.scope!=='partial_family')throw Error('INVALID_CAPTURE');
    this.controller().acceptPrepared(capture);
    await this.controller().handoff();
    await this.location.clear();
    return {};
    }finally{this.finishing=false;}
  }
  private async closeReader() {
    const r=this.location.current;if(r?.phase!=='bound' || !r.reader)return;
    try {
      const tab=await chrome.tabs.get(r.reader.tabId);
      if(tab.openerTabId===r.tabId && (tab.url===r.reader.sourceURL || tab.url==='about:blank') && (!tab.pendingUrl || tab.pendingUrl===r.reader.sourceURL))await chrome.tabs.remove(r.reader.tabId);
    }catch{ /* A closed or user-navigated tab is never replaced or guessed. */ }
    await this.location.forgetReader();
  }
  abandonInterrupted() {return this.serialized(()=>this.abandon());}
  private async abandon() {
    if(this.busy || this.controller().isBusy())throw Error('ACTION_IN_PROGRESS');
    const r=this.location.current;if(!r)return;
    const contexts=await this.contexts();
    if(this.application.state!=='available' && contexts.some(c=>r.phase==='bound'?c.documentId===r.documentId:true))throw Error('ACTION_IN_PROGRESS');
    await this.closeReader();await this.location.clear();
  }
  closed(tabId:number){return this.serialized(async()=>{const r=this.location.current;if(r?.tabId===tabId)await this.closeReader();});}
}

export function isFamilyCommand(value:unknown) {
  return !!value && typeof value==='object' && 'type' in value && typeof value.type==='string' && value.type.startsWith('family.') && value.type!=='family.inspect';
}
export type InspectReply=FamilySummary;
