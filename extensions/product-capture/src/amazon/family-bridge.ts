import { z } from 'zod';
import { FamilyLocation, familyURL } from './family-location';
import type { CaptureTab, Controller } from '../controller';
import type { ReturnSession } from '../return-session';
import { UUID } from '../validation';
import { amazonPageSource } from './validation';
import type { FamilySummary } from './family-coordinator';
import { assertFamilyReaderOwnership, sameSiteInterruption } from './family-reader';
import { DraftStore } from '../draft-store';
import type { FamilySeed } from './family-checkpoint';
import type { AmazonCapturePayload } from './wire';

const common={revision:z.string().regex(UUID)};
const checkpoint={...common,expectedRevision:z.number().int().positive(),generation:z.string().regex(UUID)};
const message=z.discriminatedUnion('type',[
  z.strictObject({...common,type:z.literal('family.init')}),
  z.strictObject({...checkpoint,type:z.literal('family.cancel')}),
  z.strictObject({...checkpoint,type:z.literal('family.resume')}),
  z.strictObject({...checkpoint,type:z.literal('family.pause'),reason:z.string().max(100)}),
  z.strictObject({...checkpoint,type:z.literal('family.seed'),seed:z.unknown()}),
  z.strictObject({...checkpoint,type:z.literal('family.observe'),observation:z.unknown()}),
  z.strictObject({...common,type:z.literal('family.closeReader')}),
  z.strictObject({...common,type:z.literal('family.return')}),
  z.strictObject({...common,type:z.literal('family.reader'),tabId:z.number().int().min(0),sourceURL:z.string().max(2048)}),
  z.strictObject({...checkpoint,type:z.literal('family.finish')}),
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
  private selected?:string;
  readonly ready:Promise<void>;
  constructor(private readonly controller:()=>Controller,private readonly application:ReturnSession,private readonly drafts:DraftStore,private readonly app:string){
    this.ready=Promise.all([this.location.ready,drafts.ready]).then(async()=>{const d=await drafts.latest();if(d?.mode==='family'&&!d.payload&&d.app===app)this.selected=d.id;});
  }
  get active(){return this.location.unavailable || !!this.location.current || !!this.selected;}
  get revision(){return this.location.unavailable?undefined:this.location.current?.revision??this.selected;}
  get busy(){return !!this.opening || this.finishing;}
  async select(id:string) {
    if(this.busy)throw Error('ACTION_IN_PROGRESS');
    const d=await this.drafts.get(id);
    if(!d||d.mode!=='family'||d.payload||d.app!==this.app||d.phase!=='paused')throw Error('DRAFT_CHANGED');
    await this.closeReader();await this.location.clear();this.selected=id;this.attempted=false;
  }
  async forget(id:string) {if(this.selected===id){await this.closeReader();await this.location.clear();this.selected=undefined;}}
  private async contexts() {
    const r=this.location.current;if(!r)return [];
    return chrome.runtime.getContexts({contextTypes:['TAB'],documentUrls:[familyURL(r.revision)]});
  }
  async begin(tab:CaptureTab,revision:string) {
    if(this.opening || this.active)throw Error('ACTION_IN_PROGRESS');
    this.attempted=false;
    this.opening=this.serialized(async()=>{
      if(!tab.url)throw Error('INVALID_PAGE');
      const draft=await this.drafts.create(amazonPageSource(tab.url).sourceURL,'family',this.app,revision);
      this.selected=draft.id;
      await this.location.prepare(tab,revision);
      const created=await chrome.tabs.create({url:familyURL(revision)});
      if(created.id===undefined)throw Error('FAMILY_INTERRUPTED');
      await this.location.noteTab(created.id);
    }).finally(()=>{this.opening=undefined;});
    return this.opening;
  }
  async status():Promise<FamilyStatus|undefined> {
    if(this.location.unavailable)return {phase:'uncertain',observed:0,total:0,reason:'RETURN_STATE_UNAVAILABLE'};
    const r=this.location.current;
    const draft=this.selected?await this.drafts.get(this.selected):undefined;
    if(!r)return draft?{phase:'paused',observed:draft.family?.count??0,total:draft.family?.manifest.members.length??0,reason:draft.reason}:undefined;
    const basic:FamilyStatus={phase:'initializing',observed:0,total:0,reason:''};
    try {
      const contexts=await this.contexts();
      if(contexts.length!==1){
        if(r.phase==='pending' && r.tabId!==undefined){try{const tab=await chrome.tabs.get(r.tabId);if(tab.url===familyURL(r.revision)||tab.pendingUrl===familyURL(r.revision))return basic;}catch{/* closed */}}
        return {...basic,phase:'paused',observed:draft?.family?.count??0,total:draft?.family?.manifest.members.length??0,reason:'DRAFT_INTERRUPTED'};
      }
      if(r.phase==='bound' && (contexts[0].documentId!==r.documentId || contexts[0].tabId!==r.tabId))return {...basic,phase:'interrupted',reason:'FAMILY_INTERRUPTED'};
      if(r.phase==='pending')return basic;
      const value=await chrome.runtime.sendMessage({type:'family.inspect',revision:r.revision});
      const view=summary.parse(value);
      return draft?.family?{...view,observed:draft.family.count,total:draft.family.manifest.members.length,...(draft.phase==='paused'?{phase:'paused' as const,reason:draft.reason}:{} )}:view;
    }catch{return {...basic,phase:'uncertain',reason:'FAMILY_INTERRUPTED'};}
  }
  async focus() {
    let r=this.location.current;
    const draft=this.selected?await this.drafts.get(this.selected):undefined;
    if(!r && draft){await this.location.restore(draft.sourceURL,draft.id);r=this.location.current;}
    if(!r)throw Error('FAMILY_INTERRUPTED');
    const contexts=await this.contexts();
    const context=contexts.length===1?contexts[0]:undefined;
    const id=context?.tabId??r.tabId;
    if(context && id!==undefined){
      const tab=await chrome.tabs.get(id);
      if(tab.url!==familyURL(r.revision) && tab.pendingUrl!==familyURL(r.revision))throw Error('FAMILY_INTERRUPTED');
      await chrome.tabs.update(id,{active:true});if(tab.windowId!==undefined)await chrome.windows.update(tab.windowId,{focused:true});return;
    }
    // Browser-restored old IDs are not authority. An explicit restore creates a
    // new progress tab from the durable manifest, without reading a source page.
    if(!draft)throw Error('FAMILY_INTERRUPTED');
    await this.location.clear();await this.location.restore(draft.sourceURL,draft.id);
    const created=await chrome.tabs.create({url:familyURL(draft.id)});if(created.id===undefined)throw Error('FAMILY_INTERRUPTED');await this.location.noteTab(created.id);
  }
  private async sender(sender:chrome.runtime.MessageSender,revision:string,initialize=false) {
    const r=this.location.current;
    if(!r || r.revision!==revision || sender.id!==chrome.runtime.id || sender.url!==familyURL(revision) || !sender.documentId)throw Error('INVALID_MESSAGE');
    const contexts=await this.contexts();
    if(contexts.length!==1)throw Error('INVALID_MESSAGE');
    const c=contexts[0];
    if(c.documentId!==sender.documentId || c.documentUrl!==sender.url || c.tabId<0 || c.frameId!==0 || sender.tab && sender.tab.id!==c.tabId)throw Error('INVALID_MESSAGE');
    if(r.phase==='bound' && (r.tabId!==c.tabId || !initialize&&r.documentId!==c.documentId) || r.tabId!==undefined && r.tabId!==c.tabId)throw Error('ENTRY_CHANGED');
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
    const context=await this.sender(sender,m.revision,m.type==='family.init');
    if(m.type==='family.init'){
      let draft=await this.drafts.get(m.revision);if(!draft || draft.app!==this.app)throw Error('DRAFT_CHANGED');
      const previous=this.location.current;
      if(previous?.phase==='bound'&&previous.documentId!==context.documentId){
        if(draft.phase==='capturing')draft=await this.drafts.pause(draft.id,draft.revision,draft.generation,'DRAFT_INTERRUPTED');
        await this.location.rebind(context.tabId,context.documentId!);
      }else await this.location.bind(context.tabId,context.documentId!);
      const r=this.location.current!;
      return {source:{id:r.sourceTabId,url:r.sourceURL,documentId:r.sourceDocumentId},coordinatorTabId:context.tabId,windowId:context.windowId,reader:r.phase==='bound'?r.reader:undefined,
        draft:{revision:draft.revision,generation:draft.generation,phase:draft.phase},checkpoint:await this.drafts.checkpoint(draft.id)};
    }
    const r=this.location.current;if(r?.phase!=='bound')throw Error('ENTRY_CHANGED');
    if(m.type==='family.reader'){
      const request=m as {tabId:number;sourceURL:string}; const tab=await chrome.tabs.get(request.tabId);
      const retry=r.reader?.sourceURL===request.sourceURL;
      const canonical=(url:string|undefined)=>{try{return url?amazonPageSource(url).sourceURL:undefined;}catch{return undefined;}};
      const sameTarget=(url:string|undefined)=>canonical(url)===request.sourceURL;
      const registered=!!r.reader && canonical(tab.url)===r.reader.sourceURL;
      const first=!r.reader;
      const interruption=(url:string|undefined)=>sameSiteInterruption(url,request.sourceURL);
      const initial=first && (sameTarget(tab.url) || interruption(tab.url)
        || (!tab.url || tab.url==='about:blank') && (sameTarget(tab.pendingUrl) || interruption(tab.pendingUrl)));
      assertFamilyReaderOwnership(tab,request.tabId,{coordinatorTabId:r.tabId,windowId:context.windowId,reader:r.reader});
      if(tab.pendingUrl && tab.pendingUrl!==tab.url && tab.pendingUrl!==request.sourceURL && !((first||retry) && (sameTarget(tab.pendingUrl)||interruption(tab.pendingUrl)))
        || !initial && !registered && !(retry && (tab.url===r.reader?.previousURL || interruption(tab.url))))throw Error('FAMILY_INTERRUPTED');
      await this.location.reader(request.tabId,request.sourceURL);return {};
    }
    if(m.type==='family.return'){if(this.application.state!=='available')throw Error('HANDOFF_UNAVAILABLE');await this.controller().returnToApplication();return {};}
    if(m.type==='family.closeReader'){await this.closeReader();return {};}
    if(m.type==='family.resume'||m.type==='family.seed'||m.type==='family.observe'||m.type==='family.pause'||m.type==='family.cancel'){
      let draft=await this.drafts.get(m.revision);if(!draft || draft.revision!==m.expectedRevision || draft.generation!==m.generation)throw Error('DRAFT_CHANGED');
      if(m.type==='family.resume'){
        if(draft.phase==='paused')draft=await this.drafts.resume(draft.id,draft.revision,draft.generation);
        else if(draft.phase!=='capturing')throw Error('DRAFT_CHANGED');
      }else if(m.type==='family.seed')draft=await this.drafts.seed(draft.id,draft.revision,draft.generation,m.seed as FamilySeed);
      else if(m.type==='family.observe')draft=await this.drafts.observe(draft.id,draft.revision,draft.generation,m.observation as {capture:AmazonCapturePayload;parentASIN:string|null});
      else {
      if(this.finishing || this.application.state!=='none' || this.controller().isBusy())throw Error('HANDOFF_UNAVAILABLE');
      if(draft.phase==='capturing')draft=await this.drafts.pause(draft.id,draft.revision,draft.generation,m.type==='family.pause'?m.reason:'');
      if(m.type==='family.cancel')await this.closeReader();
      }
      return {revision:draft.revision,generation:draft.generation,phase:draft.phase};
    }
    if(this.attempted || this.application.state!=='none')throw Error('HANDOFF_UNAVAILABLE');
    this.finishing=true;
    try {
    if(m.type!=='family.finish')throw Error('INVALID_MESSAGE');
    const draft=await this.drafts.finishFamily(m.revision,m.expectedRevision,m.generation);
    this.attempted=true;
    await this.controller().acceptPrepared(draft.id);
    await this.controller().handoff();
    await this.location.clear();
    this.selected=undefined;
    return {};
    }finally{this.finishing=false;}
  }
  private async closeReader() {
    const r=this.location.current;if(r?.phase!=='bound' || !r.reader)return;
    try {
      const tab=await chrome.tabs.get(r.reader.tabId);
      const owner=await chrome.tabs.get(r.tabId);
      if(tab.openerTabId===r.tabId && tab.windowId===owner.windowId && (tab.url===r.reader.sourceURL || tab.url===r.reader.previousURL || tab.url==='about:blank') && (!tab.pendingUrl || tab.pendingUrl===r.reader.sourceURL))await chrome.tabs.remove(r.reader.tabId);
    }catch{ /* A closed or user-navigated tab is never replaced or guessed. */ }
    await this.location.forgetReader();
  }
  abandonInterrupted() {return this.serialized(()=>this.abandon());}
  private async abandon() {
    if(this.busy || this.controller().isBusy())throw Error('ACTION_IN_PROGRESS');
    const r=this.location.current;if(!r){this.selected=undefined;return;}
    const d=this.selected?await this.drafts.get(this.selected):undefined;
    if(d?.phase==='capturing')throw Error('ACTION_IN_PROGRESS');
    await this.closeReader();await this.location.clear();
    this.selected=undefined;
  }
  closed(tabId:number){return this.serialized(async()=>{
    const r=this.location.current;if(r?.tabId!==tabId)return;
    const d=await this.drafts.get(r.revision);
    if(d?.phase==='capturing')await this.drafts.pause(d.id,d.revision,d.generation,'DRAFT_INTERRUPTED');
    await this.closeReader();
  });}
}

export function isFamilyCommand(value:unknown) {
  return !!value && typeof value==='object' && 'type' in value && typeof value.type==='string' && value.type.startsWith('family.') && value.type!=='family.inspect';
}
export type InspectReply=FamilySummary;
