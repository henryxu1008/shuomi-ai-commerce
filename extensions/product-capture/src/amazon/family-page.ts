import { FamilyCoordinator, type FamilySummary } from './family-coordinator';
import { extractFamilyTab } from '../capture-tab';
import { familyURL } from './family-location';
import { entryMessages, type EntryError } from '../entry-errors';
import { UUID } from '../validation';
import type { FamilyMember } from './family-discovery';
import { assertFamilyReaderOwnership, assertLocatedFamilyReader, type FamilyReaderOwner, sameAmazonProduct, sameSiteInterruption, waitForFamilyDocument } from './family-reader';
import { amazonPageSource } from './validation';

export function mountFamilyPage(revision:string) {
  if(!UUID.test(revision) || location.href!==familyURL(revision))throw Error('INVALID_MESSAGE');
  document.title='Amazon 其他变体采集';document.body.classList.add('family-page');
  document.body.replaceChildren();
  const heading=document.createElement('h1');heading.textContent='采集其他颜色／尺寸';
  const status=document.createElement('p');status.setAttribute('role','status');status.setAttribute('aria-live','polite');
  const note=document.createElement('p');note.textContent='只读取本页明确列出的同族规格。已保全的资料留在本机；关闭、刷新或后台重建后可明确恢复，不会自动续采或保存。';
  const buttons=new Map<string,HTMLButtonElement>();
  document.body.append(heading,note,status);
  for(const [id,text] of [['resume','继续采集'],['partial','仅使用已保全资料'],['cancel','停止采集并保留进度'],['return','返回应用核实'],['reader','查看需处理的商品页']]){
    const button=document.createElement('button');button.textContent=text;buttons.set(id,button);document.body.append(button);
  }
  let readerId:number|undefined,coordinatorTabId:number|undefined,windowId:number|undefined;
  let readerCreationAttempted=false,readerLocated=false;
  let initialized:Promise<{source:{id:number;url:string;documentId:string}}>|undefined;
  let sourceURL='',actionFailure='',pendingTarget='';
  let draft:{revision:number;generation:string;phase:string}|undefined;
  const request=async(type:string,extra:Record<string,unknown>={})=>{
    const response=await chrome.runtime.sendMessage({type,revision,...extra});
    if(!response?.ok)throw Error(response?.code||'FAMILY_INTERRUPTED');return response.result;
  };
  const initialize=()=>initialized??=request('family.init').then(value=>{coordinatorTabId=value.coordinatorTabId;windowId=value.windowId;sourceURL=value.source.url;draft=value.draft;if(value.reader){readerId=value.reader.tabId;readerLocated=true;pendingTarget=value.reader.sourceURL;}return value;});
  const checkpoint=async(type:string,extra:Record<string,unknown>={})=>{if(!draft)await initialize();draft=await request(type,{expectedRevision:draft!.revision,generation:draft!.generation,...extra});};
  async function readerOwner():Promise<FamilyReaderOwner> {
    // Reconfirm the existing locator before applying local first/next-load
    // guards: registration may have committed before its response was lost.
    const owner=await request('family.init');
    if(owner.coordinatorTabId!==coordinatorTabId || owner.windowId!==windowId)throw Error('FAMILY_INTERRUPTED');
    return owner;
  }
  async function read(member:FamilyMember,signal:AbortSignal) {
    await initialize();signal.throwIfAborted();
    const target=amazonPageSource(new URL(`/dp/${member.sourceID}`,sourceURL).href).sourceURL;
    if(readerId===undefined){
      if(readerCreationAttempted)throw Error('FAMILY_INTERRUPTED');
      readerCreationAttempted=true;
      const tab=await chrome.tabs.create({url:target,active:false,openerTabId:coordinatorTabId,windowId});
      if(tab.id===undefined)throw Error('FAMILY_INTERRUPTED');
      readerId=tab.id;pendingTarget=target;signal.throwIfAborted();
    }
    const id=readerId;
    if(id===undefined)throw Error('FAMILY_INTERRUPTED');
    const owner=await readerOwner();signal.throwIfAborted();
    const current=await chrome.tabs.get(id);signal.throwIfAborted();
    assertFamilyReaderOwnership(current,id,owner);
    if(readerLocated && !owner.reader)throw Error('FAMILY_INTERRUPTED');
    if(owner.reader){
      assertLocatedFamilyReader(current,id,owner);
      if(!readerLocated && owner.reader.sourceURL!==target)throw Error('FAMILY_INTERRUPTED');
      readerLocated=true;
    }
    if(current.pendingUrl && current.pendingUrl!==current.url && !sameAmazonProduct(current.pendingUrl,target) && !sameSiteInterruption(current.pendingUrl,target))throw Error('FAMILY_INTERRUPTED');
    const retry=pendingTarget===target;
    const interruption=retry && sameSiteInterruption(current.url,target);
    if(!readerLocated){
      // tabs.create already navigated. Missing metadata is only a bounded wait,
      // never proof of ownership or permission to overwrite an unknown page.
      await waitForFamilyDocument(id,target,signal,false,true,owner);
      await request('family.reader',{tabId:id,sourceURL:target});signal.throwIfAborted();
      readerLocated=true;
    }else{
      if(!interruption){
        await request('family.reader',{tabId:id,sourceURL:target});signal.throwIfAborted();
      }
      // A still-blocked owned reader is classified again only on explicit resume;
      // the locator remains the expected product, never the verification URL.
      const navigate=!retry || !sameAmazonProduct(current.url,target) && !interruption;
      pendingTarget=target;
      await waitForFamilyDocument(id,target,signal,navigate,false,await readerOwner());
    }
    const located=await readerOwner();signal.throwIfAborted();
    const ready=await chrome.tabs.get(id);signal.throwIfAborted();
    assertLocatedFamilyReader(ready,id,located);
    if(located.reader?.sourceURL!==target || !sameAmazonProduct(ready.url,target) && !sameSiteInterruption(ready.url,target))throw Error('PAGE_CHANGED');
    const result=await extractFamilyTab({id,url:target},false,signal);
    const after=await chrome.tabs.get(id);signal.throwIfAborted();
    assertLocatedFamilyReader(after,id,located);
    if(!sameAmazonProduct(after.url,target))throw Error('PAGE_CHANGED');
    return {...result,parentASIN:result.manifest?.parentASIN??null};
  }
  const coordinator=new FamilyCoordinator({
    seed:async(signal)=>{const context=await initialize();signal.throwIfAborted();if(context.source.id===undefined||!context.source.documentId)throw Error('INVALID_PAGE');const result=await extractFamilyTab(context.source,true,signal);if(!result.manifest)throw Error('FAMILY_NOT_AVAILABLE');return {anchor:result.capture,manifest:result.manifest};},
    read,
    close:async()=>{await request('family.closeReader');readerId=undefined;readerLocated=false;},
    handoff:async()=>{await checkpoint('family.finish');},
  }, {resume:()=>checkpoint('family.resume'),seed:seed=>checkpoint('family.seed',{seed}),observe:observation=>checkpoint('family.observe',{observation}),pause:async reason=>{
    try{await checkpoint('family.pause',{reason});}catch(e){
      if(!(e instanceof Error)||e.message!=='DRAFT_CHANGED')throw e;
      const current=await request('family.init');draft=current.draft;
      if(draft?.phase==='capturing')await checkpoint('family.pause',{reason});
    }
  }});
  const describe=(s:FamilySummary)=>`${s.observed}/${s.total} 项已保全（仅本页发现的规格） · ${s.phase==='paused'?'已暂停':s.phase==='collecting'?'正在采集':s.phase==='transferred'?'已交给应用，请在原应用确认保存':s.phase==='uncertain'?'交接待核实，本机资料保留，请恢复原应用':s.phase==='transferring'?'正在交给原应用':'正在初始化'}${s.reason?' · '+(Object.hasOwn(entryMessages,s.reason)?entryMessages[s.reason as EntryError]:'该商品页暂不可读取，请查看后决定继续或使用已保全资料。'):''}`;
  function render(){const s=coordinator.snapshot();status.textContent=describe(s)+(actionFailure?' · '+actionFailure:'');buttons.get('resume')!.hidden=s.phase!=='paused'||s.reason==='FAMILY_LIMIT_REACHED';buttons.get('partial')!.hidden=s.phase!=='paused'||!s.observed;buttons.get('cancel')!.hidden=!['initializing','collecting','paused'].includes(s.phase);buttons.get('return')!.hidden=!['uncertain','transferred'].includes(s.phase);buttons.get('reader')!.hidden=s.phase!=='paused'||readerId===undefined;}
  async function action(action:()=>Promise<void>){actionFailure='';try{await action();}catch{actionFailure='操作未完成，请保留此页与原应用核实；不要重复采集或保存。';}render();}
  buttons.get('resume')!.onclick=()=>void action(async()=>{
    // A checkpoint may have committed before its reply was lost, or a rebuilt
    // worker may have fenced our generation. Explicit resume reloads that fact.
    const current=await request('family.init');
    draft=current.draft;initialized=Promise.resolve(current);
    coordinator.restore(current.checkpoint,current.draft.phase);
    const pending=coordinator.resume();render();await pending;
  });
  buttons.get('partial')!.onclick=()=>void action(()=>coordinator.useObserved());
  buttons.get('cancel')!.onclick=()=>void action(()=>coordinator.cancel());
  buttons.get('reader')!.onclick=()=>void action(async()=>{if(readerId===undefined)return;const owner=await readerOwner();const tab=await chrome.tabs.get(readerId);assertLocatedFamilyReader(tab,readerId,owner);await chrome.tabs.update(readerId,{active:true});if(tab.windowId!==undefined)await chrome.windows.update(tab.windowId,{focused:true});});
  buttons.get('return')!.onclick=()=>void action(()=>request('family.return'));
  const inspect=(m:unknown,sender:chrome.runtime.MessageSender,respond:(s:FamilySummary)=>void)=>{
    if(sender.id!==chrome.runtime.id || sender.tab || sender.url && sender.url!==chrome.runtime.getURL('background.js'))return false;
    if(m && typeof m==='object' && 'type' in m && m.type==='family.inspect' && 'revision' in m && m.revision===revision){respond(coordinator.snapshot());return false;}
    return false;
  };
  chrome.runtime.onMessage.addListener(inspect);
  const ticker=setInterval(render,250);
  addEventListener('pagehide',()=>{clearInterval(ticker);chrome.runtime.onMessage.removeListener(inspect);});
  void action(async()=>{const context=await initialize();if(context.draft.phase!=='capturing'||context.checkpoint)coordinator.restore(context.checkpoint,context.draft.phase);else await coordinator.start();});render();
}
