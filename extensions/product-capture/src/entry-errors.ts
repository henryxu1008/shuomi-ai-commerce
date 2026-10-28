export const entryMessages = {
  DRAFT_CHANGED: '本机草稿已变化，请刷新状态后恢复；已保全资料未删除。',
  DRAFT_STORAGE_FULL: '本机草稿空间不足，采集已暂停；此前已保全资料仍在。请先核实并清理不再需要的本地副本。',
  DRAFT_CORRUPT: '本机草稿未通过完整性检查，已停止恢复，不会覆盖原资料。',
  DRAFT_SCOPE_CHANGED: '请回到原账号、企业和成员恢复本机资料。',
  DRAFT_BODY_MISMATCH: '资料与原发送内容不一致，已停止保存，本机原资料保留。',
  DRAFT_INTERRUPTED: '采集已暂停，已保全的进度仍在本机；可明确继续。',
  AMAZON_VERIFICATION_REQUIRED: 'Amazon 要求真人校验。本次已暂停，请查看需处理的商品页，处理后回到原规格，再明确继续采集。',
  AMAZON_SIGN_IN_REQUIRED: 'Amazon 要求登录。本次已暂停，请在需处理的商品页登录并回到原规格，再明确继续采集。',
  PAGE_NOT_READY: '商品页面尚未就绪，本次已暂停；页面显示完整后可明确继续。',
  CAPTURE_TOO_LARGE: '该商品页超过现有采集资源上限，本次已暂停，可使用已采数据或取消。',
  CAPTURE_TIMEOUT: '等待商品页或解析超时，本次已暂停；请查看页面后决定继续或取消。',
  UNSUPPORTED_PAGE: '当前页面没有可读取的商品结构，本次已暂停。',
  SOURCE_CHANGED: '商品身份或规格发生变化，本次已暂停，不会合并不匹配的资料。',
  INVALID_CAPTURE: '商品资料未通过校验，本次已暂停，不会交给应用保存。',
  FAMILY_NOT_AVAILABLE: '未找到明确的同族规格关系，可取消后选择仅采当前规格。',
  FAMILY_INVALID: '规格关系不明确或发生变化，本次已暂停；不会猜测组合。',
  FAMILY_INTERRUPTED: '变体采集进度已中断，请保留原应用核实；不要重复保存。',
  FAMILY_LIMIT_REACHED: '已达到本次采集资源上限，可明确使用已采数据或取消。',
  INVALID_PAGE: '请在受支持的 Amazon 或 1688 单商品详情页触发采集。',
  CAPTURE_REJECTED: '未取得可用商品资料，请检查页面是否加载完成或需要登录。',
  PAGE_CHANGED: '采集时商品页面已变化，本次没有交给应用，请在当前商品页重新触发。',
  CAPTURE_EXPIRED: '插件暂存已失效。若曾交给应用，请回到原应用标签核实；不要直接重复保存。',
  HANDOFF_UNAVAILABLE: '原应用交接待核实，请保留现场与已有应用页面，不要重复采集或保存。',
  ORIGINAL_TAB_UNAVAILABLE: '无法返回原应用标签，请检查并保留原页面，不要重新采集或保存。',
  ACTION_IN_PROGRESS: '当前操作尚未结束，请等待后再开始新的采集。',
  ENTRY_CHANGED: '采集操作已变化，请核对当前状态后重新选择；本次没有开始新的采集。',
  RETURN_STATE_UNAVAILABLE: '原标签定位暂不可读，请保留原应用页面核实，不要重复采集或保存。',
  INVALID_MESSAGE: '此操作入口不可用，请重新打开当前版本的采集扩展。',
  ACTION_UNAVAILABLE: '采集未完成，请检查当前商品页；已交给应用的操作请在原页面核实。',
} as const;
export type EntryError = keyof typeof entryMessages;
export function entryError(error: unknown): EntryError {
  return error instanceof Error && Object.hasOwn(entryMessages, error.message) ? error.message as EntryError : 'ACTION_UNAVAILABLE';
}

// Only deterministic capture failures cross an injected document boundary.
// Arbitrary messages (including uppercase internal errors) are never forwarded.
const captureCodes = new Set<string>([
  'AMAZON_VERIFICATION_REQUIRED', 'AMAZON_SIGN_IN_REQUIRED', 'PAGE_NOT_READY',
  'CAPTURE_TOO_LARGE', 'CAPTURE_TIMEOUT', 'UNSUPPORTED_PAGE', 'SOURCE_CHANGED',
  'PAGE_CHANGED', 'INVALID_PAGE', 'INVALID_CAPTURE', 'CAPTURE_REJECTED',
  'FAMILY_NOT_AVAILABLE', 'FAMILY_INVALID', 'FAMILY_INTERRUPTED', 'FAMILY_LIMIT_REACHED',
  'RETURN_STATE_UNAVAILABLE', 'ENTRY_CHANGED',
  'DRAFT_CHANGED', 'DRAFT_STORAGE_FULL', 'DRAFT_CORRUPT', 'DRAFT_INTERRUPTED',
]);
export function captureErrorCode(code: unknown): EntryError {
  return typeof code === 'string' && captureCodes.has(code) ? code as EntryError : 'CAPTURE_REJECTED';
}
export function captureException(error: unknown): EntryError {
  return captureErrorCode(error instanceof Error ? error.message : undefined);
}
