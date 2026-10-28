import { entryMessages, type EntryError } from './entry-errors';
import { mountFamilyPage } from './amazon/family-page';
import type { FamilyStatus } from './amazon/family-bridge';
type State = { loading: boolean; captured: boolean; handed: boolean; title: string | null; sourceURL: string | null;
  revision: string; returnState: 'none' | 'available' | 'uncertain'; outcome: string; recoveryURL: string | null; missingFacts: { field: string; reason: string }[]; shortcut: string | null; error?: EntryError; family?: FamilyStatus };
function mountPopup() {
const element = <T extends HTMLElement>(id: string) => document.getElementById(id) as T;
const capture = element<HTMLButtonElement>('capture');
const fresh = element<HTMLButtonElement>('fresh'); const status = element('status');
const recovery = element<HTMLButtonElement>('recovery');
const newConfirm = element<HTMLButtonElement>('new-confirm-button');
let revision: string | undefined, confirmationRevision: string | undefined;
const outcomes: Record<string, string> = {
  empty: '打开 1688 或 Amazon 五站单商品详情页后采集。如之前已提交，请先在原应用页核实。', awaiting_confirmation: '已交给应用，请在应用中确认当前用户和企业。',
  processing: '应用正在导入；响应中断时请核实原操作。',
  published: '应用返回导入成功。可打开原操作查看结果。', failed: '应用返回导入失败，请查看原操作。',
  outcome_unknown: '插件暂存或操作状态待核实。保存结果请在原应用页面查看，不要重复导入。',
};
function render(state: State) {
  revision = state.revision;
  const canReturn = state.returnState === 'available';
  const familyActive=!!state.family && !canReturn;
  element('title').textContent = state.title ?? '商品采集';
  element('source').textContent = state.sourceURL ?? '';
  status.textContent = state.loading ? '正在处理当前商品…' : state.error && Object.hasOwn(entryMessages, state.error) ? entryMessages[state.error]
    : canReturn && !state.handed ? '可返回原应用核实保存结果。'
      : state.returnState === 'uncertain' ? '原标签定位待核实，请先保留原应用页面；新的采集需要明确确认。'
        : state.captured && !state.handed ? '资料已取得，正在交给应用。' : outcomes[state.outcome] ?? '请在应用中核实结果。';
  capture.hidden = state.handed || canReturn;
  capture.textContent=familyActive?'查看采集进度':'采集并在应用确认';
  if(familyActive)status.textContent=`同族采集 ${state.family!.observed}/${state.family!.total}，请回进度页查看。`;
  capture.disabled = state.loading || state.handed || state.returnState !== 'none' || state.family?.phase==='interrupted';
  fresh.hidden = !state.captured && state.returnState === 'none' && !familyActive;
  fresh.disabled = state.loading || state.error === 'RETURN_STATE_UNAVAILABLE' || (familyActive && !['interrupted','paused'].includes(state.family!.phase));
  newConfirm.disabled = fresh.disabled;
  recovery.hidden = !canReturn;
  recovery.disabled = state.loading;
  element('shortcut').textContent = state.shortcut === null ? '快捷键状态暂不可读，可使用本页按钮。'
    : state.shortcut ? `商品页快捷键：${state.shortcut} · 直接采集并在应用确认。`
      : '快捷键未分配，可在浏览器扩展的快捷键设置中分配；也可使用本页按钮。';
  const labels: Record<string,string> = { description:'商品描述',attributes:'商品属性',variants:'规格',priceFacts:'报价',images:'来源图片' };
  const missing = [...new Set(state.missingFacts.map(item => item.field.includes('currency') ? '报价币种' : /^variants[.\[]/.test(item.field) && item.field.endsWith('.images') ? '规格图片' : labels[item.field] ?? '部分商品资料'))].join('、');
  element('missing').textContent = missing ? `未取得：${missing}` : '';
}
async function action(type: string, expectedRevision?: string) {
  capture.disabled = true; recovery.disabled = true; fresh.disabled = true; newConfirm.disabled = true; status.textContent = '正在处理…';
  try {
    const response = await chrome.runtime.sendMessage(type === 'popup.new' ? { type, revision: expectedRevision } : { type });
    if (response?.state) render(response.state);
    if (!response?.ok) status.textContent = response?.code && Object.hasOwn(entryMessages, response.code)
      ? entryMessages[response.code as EntryError] : entryMessages.ACTION_UNAVAILABLE;
  } catch {
    status.textContent = '插件状态不可读。如已交给应用，请在原应用页面核实，不要直接重复采集或保存。';
    // Only an explicit successful status read can re-enable an entry.
  }
}
capture.addEventListener('click', () => void action('popup.captureAndHandoff'));
recovery.addEventListener('click', () => void action('popup.return'));
fresh.addEventListener('click', () => { if (revision) { confirmationRevision = revision; element('new-confirm').hidden = false; } });
element('new-cancel').addEventListener('click', () => { element('new-confirm').hidden = true; });
newConfirm.addEventListener('click', () => {
  if (element('new-confirm').hidden || !confirmationRevision) return;
  const expected = confirmationRevision; confirmationRevision = undefined;
  element('new-confirm').hidden = true; void action('popup.new', expected);
});
element('refresh').addEventListener('click', () => void action('popup.status'));
type LocalDraft={id:string;revision:number;sourceURL:string;phase:string;observed:number;total:number;createdAt:number;restorable:boolean};
const phaseLabel:Record<string,string>={capturing:'采集中',paused:'已暂停',ready:'待交接',bound:'待确认',sending:'发送中',unknown:'结果待核实',saved_reported:'应用报告已保存',rejected:'应用报告未保存',corrupt:'资料校验失败'};
async function draftAction(type:string,draft?:LocalDraft){
  const message=draft?{type,id:draft.id,expectedRevision:draft.revision}:{type};
  try{
    const response=await chrome.runtime.sendMessage(message);
    if(!response?.ok)throw Error(response?.code??'ACTION_UNAVAILABLE');
    element('drafts').hidden=false;const items=element('draft-items');items.replaceChildren();
    for(const d of response.drafts as LocalDraft[]){
      const row=document.createElement('section'),label=document.createElement('p');
      label.textContent=`${d.sourceURL||'无法读取的本机资料'} · ${phaseLabel[d.phase]??'待核实'}${d.total?` · ${d.observed}/${d.total} 项已保全`:''}`;row.append(label);
      const busy=['capturing','sending'].includes(d.phase);
      if(d.restorable){const restore=document.createElement('button');restore.textContent='恢复原操作';restore.disabled=busy;restore.onclick=()=>void draftAction('popup.restoreDraft',d);row.append(restore);}
      const remove=document.createElement('button');remove.textContent='清理本机副本';remove.disabled=busy;
      remove.onclick=()=>{if(window.confirm(`${['unknown','corrupt'].includes(d.phase)?'保存结果或资料完整性尚未确认，清理后可能无法恢复未保存内容。':'清理后将无法从本机恢复这份资料。'}只删除本机副本，应用数据不变。确定清理？`))void draftAction('popup.removeDraft',d);};row.append(remove);items.append(row);
    }
    element('draft-status').textContent=response.reloadRequired?'损坏资料已清理，请重新加载扩展；不会自动采集或保存。':response.drafts.length?'资料保留在当前浏览器配置中，清理浏览器扩展存储或卸载扩展仍会删除本机副本。':'暂无本机保全资料。';
  }catch(e){element('draft-status').textContent=e instanceof Error&&Object.hasOwn(entryMessages,e.message)?entryMessages[e.message as EntryError]:'本机资料暂不可读，请保留现有页面。';}
}
element('draft-list').addEventListener('click',()=>void draftAction('popup.drafts'));
void action('popup.status');

}
const mode = document.location?.hash;
if(mode?.startsWith('#amazon-family='))mountFamilyPage(mode.slice(15));
else mountPopup();
