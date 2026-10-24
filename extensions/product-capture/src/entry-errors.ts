export const entryMessages = {
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
