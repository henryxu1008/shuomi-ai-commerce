import { entryMessages, type EntryError } from './entry-errors';
type State = { loading: boolean; captured: boolean; handed: boolean; title: string | null; sourceURL: string | null;
  outcome: string; recoveryURL: string | null; missingFacts: { field: string; reason: string }[]; shortcut: string | null; error?: EntryError };
const element = <T extends HTMLElement>(id: string) => document.getElementById(id) as T;
const capture = element<HTMLButtonElement>('capture');
const fresh = element<HTMLButtonElement>('fresh'); const status = element('status');
const recovery = element<HTMLButtonElement>('recovery');
const newConfirm = element<HTMLButtonElement>('new-confirm-button');
const outcomes: Record<string, string> = {
  empty: '打开 1688 或 Amazon 五站单商品详情页后采集。如之前已提交，请先在原应用页核实。', awaiting_confirmation: '已交给应用，请在应用中确认当前用户和企业。',
  processing: '应用正在导入；响应中断时请核实原操作。',
  published: '应用返回导入成功。可打开原操作查看结果。', failed: '应用返回导入失败，请查看原操作。',
  outcome_unknown: '插件暂存或操作状态待核实。保存结果请在原应用页面查看，不要重复导入。',
};
function render(state: State) {
  element('title').textContent = state.title ?? '商品采集';
  element('source').textContent = state.sourceURL ?? '';
  status.textContent = state.loading ? '正在处理当前商品…' : state.error && Object.hasOwn(entryMessages, state.error) ? entryMessages[state.error]
    : state.captured && !state.handed ? '资料已取得，正在交给应用。' : outcomes[state.outcome] ?? '请在应用中核实结果。';
  capture.hidden = state.handed;
  capture.disabled = state.loading || state.handed;
  fresh.hidden = !state.captured;
  fresh.disabled = state.loading;
  newConfirm.disabled = state.loading;
  recovery.hidden = !state.recoveryURL;
  recovery.disabled = state.loading;
  element('shortcut').textContent = state.shortcut === null ? '快捷键状态暂不可读，可使用本页按钮。'
    : state.shortcut ? `商品页快捷键：${state.shortcut} · 直接采集并在应用确认。`
      : '快捷键未分配，可在浏览器扩展的快捷键设置中分配；也可使用本页按钮。';
  const labels: Record<string,string> = { description:'商品描述',attributes:'商品属性',variants:'规格',priceFacts:'报价',images:'来源图片' };
  const missing = [...new Set(state.missingFacts.map(item => item.field.includes('currency') ? '报价币种' : /^variants[.\[]/.test(item.field) && item.field.endsWith('.images') ? '规格图片' : labels[item.field] ?? '部分商品资料'))].join('、');
  element('missing').textContent = missing ? `未取得：${missing}` : '';
}
async function action(type: string) {
  capture.disabled = true; recovery.disabled = true; fresh.disabled = true; newConfirm.disabled = true; status.textContent = '正在处理…';
  try {
    const response = await chrome.runtime.sendMessage({ type });
    if (response?.state) render(response.state);
    if (!response?.ok) status.textContent = response?.code && Object.hasOwn(entryMessages, response.code)
      ? entryMessages[response.code as EntryError] : entryMessages.ACTION_UNAVAILABLE;
  } catch {
    status.textContent = '插件状态不可读。如已交给应用，请在原应用页面核实，不要直接重复采集或保存。';
    capture.disabled = false;
  }
}
capture.addEventListener('click', () => void action('popup.captureAndHandoff'));
recovery.addEventListener('click', () => void action('popup.return'));
fresh.addEventListener('click', () => { element('new-confirm').hidden = false; });
element('new-cancel').addEventListener('click', () => { element('new-confirm').hidden = true; });
newConfirm.addEventListener('click', () => { element('new-confirm').hidden = true; void action('popup.new'); });
element('refresh').addEventListener('click', () => void action('popup.status'));
void action('popup.status');
