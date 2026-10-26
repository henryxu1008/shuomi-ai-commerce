import { FamilyCoordinator, type FamilySummary } from './family-coordinator';
import { extractFamilyTab } from '../capture-tab';
import { familyURL } from './family-location';
import { entryMessages, type EntryError } from '../entry-errors';
import { UUID } from '../validation';
import type { FamilyMember } from './family-discovery';
import { sameAmazonProduct, sameSiteInterruption, waitForFamilyDocument } from './family-reader';
import { amazonPageSource } from './validation';

export function mountFamilyPage(revision:string) {
  if(!UUID.test(revision) || location.href!==familyURL(revision))throw Error('INVALID_MESSAGE');
  document.title='Amazon 其他变体采集';document.body.classList.add('family-page');
  document.body.replaceChildren();
  const heading=document.createElement('h1');heading.textContent='采集其他颜色／尺寸';
  const status=document.createElement('p');status.setAttribute('role','status');status.setAttribute('aria-live','polite');
  const note=document.createElement('p');note.textContent='只读取本页明确列出的同族规格。完成后到原应用确认保存一次；关闭或刷新此页会中断未交接的采集。';
  const buttons=new Map<string,HTMLButtonElement>();
  document.body.append(heading,note,status);
  for(const [id,text] of [['resume','继续采集'],['partial','仅使用已采数据'],['cancel','取消本次采集'],['return','返回应用核实'],['reader','查看需处理的商品页']]){
    const button=document.createElement('button');button.textContent=text;buttons.set(id,button);document.body.append(button);
  }
  let readerId:number|undefined,coordinatorTabId:number|undefined,windowId:number|undefined;
  let readerCreationAttempted=false,readerLocated=false;
  let initialized:Promise<{source:{id:number;url:string;documentId:string}}>|undefined;
  let sourceURL='',actionFailure='',pendingTarget='';
  const request=async(type:string,extra:Record<string,unknown>={})=>{
    const response=await chrome.runtime.sendMessage({type,revision,...extra});
    if(!response?.ok)throw Error(response?.code||'FAMILY_INTERRUPTED');return response.result;
  };
  const initialize=()=>initialized??=request('family.init').then(value=>{coordinatorTabId=value.coordinatorTabId;windowId=value.windowId;sourceURL=value.source.url;return value;});
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
    const current=await chrome.tabs.get(id);signal.throwIfAborted();
    if(current.openerTabId!==coordinatorTabId || current.windowId!==windowId
      || current.pendingUrl && current.pendingUrl!==current.url && !sameAmazonProduct(current.pendingUrl,target) && !sameSiteInterruption(current.pendingUrl,target))throw Error('FAMILY_INTERRUPTED');
    const retry=pendingTarget===target;
    const interruption=retry && sameSiteInterruption(current.url,target);
    if(!readerLocated){
      // tabs.create already navigated. Missing metadata is only a bounded wait,
      // never proof of ownership or permission to overwrite an unknown page.
      await waitForFamilyDocument(id,target,signal,false,true);
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
      await waitForFamilyDocument(id,target,signal,navigate);
    }
    const result=await extractFamilyTab({id,url:target},false,signal);
    return {...result,parentASIN:result.manifest?.parentASIN??null};
  }
  const coordinator=new FamilyCoordinator({
    seed:async(signal)=>{const context=await initialize();signal.throwIfAborted();const result=await extractFamilyTab(context.source,true,signal);if(!result.manifest)throw Error('FAMILY_NOT_AVAILABLE');return {anchor:result.capture,manifest:result.manifest};},
    read,
    close:async()=>{await request('family.closeReader');readerId=undefined;readerLocated=false;},
    handoff:async payload=>{await request('family.finish',{payload});},
  });
  const describe=(s:FamilySummary)=>`${s.observed}/${s.total} 项已观察（仅本页发现的规格） · ${s.phase==='paused'?'已暂停':s.phase==='collecting'?'正在采集':s.phase==='transferred'?'已交给应用，请在原应用确认保存':s.phase==='uncertain'?'交接待核实，请保留原应用，不要重采或重复保存':s.phase==='cancelled'?'本次采集已取消':s.phase==='transferring'?'正在交给原应用':'正在初始化'}${s.reason?' · '+(Object.hasOwn(entryMessages,s.reason)?entryMessages[s.reason as EntryError]:'该商品页暂不可读取，请查看后决定继续、使用已采数据或取消。'):''}`;
  function render(){const s=coordinator.snapshot();status.textContent=describe(s)+(actionFailure?' · '+actionFailure:'');buttons.get('resume')!.hidden=s.phase!=='paused'||s.reason==='FAMILY_LIMIT_REACHED';buttons.get('partial')!.hidden=s.phase!=='paused'||!s.observed;buttons.get('cancel')!.hidden=!['initializing','collecting','paused'].includes(s.phase);buttons.get('return')!.hidden=!['uncertain','transferred'].includes(s.phase);buttons.get('reader')!.hidden=s.phase!=='paused'||readerId===undefined;}
  async function action(action:()=>Promise<void>){actionFailure='';try{await action();}catch{actionFailure='操作未完成，请保留此页与原应用核实；不要重复采集或保存。';}render();}
  buttons.get('resume')!.onclick=()=>void action(()=>{if(!coordinator.snapshot().observed)initialized=undefined;return coordinator.resume();});
  buttons.get('partial')!.onclick=()=>void action(()=>coordinator.useObserved());
  buttons.get('cancel')!.onclick=()=>void action(async()=>{await coordinator.cancel();await request('family.cancel');});
  buttons.get('reader')!.onclick=()=>void action(async()=>{if(readerId===undefined)return;const tab=await chrome.tabs.get(readerId);if(tab.openerTabId!==coordinatorTabId||tab.windowId!==windowId)throw Error('FAMILY_INTERRUPTED');await chrome.tabs.update(readerId,{active:true});if(tab.windowId!==undefined)await chrome.windows.update(tab.windowId,{focused:true});});
  buttons.get('return')!.onclick=()=>void action(()=>request('family.return'));
  const inspect=(m:unknown,sender:chrome.runtime.MessageSender,respond:(s:FamilySummary)=>void)=>{
    if(sender.id!==chrome.runtime.id || sender.tab || sender.url && sender.url!==chrome.runtime.getURL('background.js'))return false;
    if(m && typeof m==='object' && 'type' in m && m.type==='family.inspect' && 'revision' in m && m.revision===revision){respond(coordinator.snapshot());return false;}
    return false;
  };
  chrome.runtime.onMessage.addListener(inspect);
  const ticker=setInterval(render,250);
  addEventListener('pagehide',()=>{clearInterval(ticker);chrome.runtime.onMessage.removeListener(inspect);});
  void action(()=>coordinator.start());render();
}
