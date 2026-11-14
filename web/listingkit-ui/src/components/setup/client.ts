type Module = { id: string; label: string; configured: boolean; installed: string; enabled: boolean; uiEnabled: boolean | null; mutable: boolean; note: string };
export type Instance = { id: string; project: string; origin: string; backendImage: string; uiImage: string; modules?: Module[]; blocks: string[]; pending?: Plan };
export type Plan = { id: string; instanceId: string; name?: string; createdAt?: string; revision?: number; archived?: boolean; stage?: string; reason?: string; attempts?: { number: number; status: string; stage: string; reason: string; updatedAt: string }[]; mode: string; status: string; changes: { label: string; before: string; after: string }[]; blocks: string[]; services: string[]; updatedAt: string };
export type RecoveryPreview = { id: string; operationId: string; action: "archive" | "retry"; changes: Plan["changes"]; blocks: string[]; expiresAt: string };
export type Environment = { dockerReady: boolean; hostFreeBytes: number | null; version: string; instances: Instance[]; profiles: { id: string; label: string; note: string }[]; blocks: string[]; pending: Plan[]; history?: Plan[] };
export type SetupRequest = { mode: "fresh" | "existing"; instanceId?: string; project?: string; profile?: string; applicationPort?: number; identityPort?: number; mailPort?: number; configurationFile?: string; toolReleaseDirectory?: string; networkSubnet?: string; enabled?: Record<string, boolean> };

const messages: Record<string, string> = {
	RECOVERY_PREVIEW_EXPIRED: "检查结果已过期。请重新检查这次安装，再确认操作。",
  INVALID_REQUEST: "设置未通过检查。请核对安装名称、端口、功能所需配置和配置文件。首次安装必须使用未被占用的新名称。",
  CONFIGURATION_CHANGED: "系统设置或安装版本已有变化，请重新检查并预览。本次未覆盖已有系统。",
  INSTANCE_BUSY: "另一个操作正在执行，请先查看它的操作记录。",
  ORIGINAL_OPERATION_UNRESOLVED: "上次操作的结果尚未确认。请查看操作记录并选择“检查操作结果”，避免重复安装。",
  CHECK_UNAVAILABLE: "检查未完成。请确认Docker已启动、安装包完整，并请维护人员核对已有系统的安装文件。",
};
let csrf = "";
export async function api<T>(path: string, body?: unknown): Promise<T> {
  const response = await fetch(`/api/${path}`, { method: body === undefined ? "GET" : "POST", credentials: "same-origin", cache: "no-store", headers: body === undefined ? {} : { "Content-Type": "application/json", "X-Setup-CSRF": csrf }, ...(body === undefined ? {} : { body: JSON.stringify(body) }) });
  if (!response.ok) {
    if (response.status === 401) throw new Error("连接已失效，请重新启动安装工具并配对。操作记录会保留。");
    if (response.status === 403) throw new Error("配对或本机请求校验未通过。请使用本机启动窗口中的一次性配对码。");
    const error = await response.json().catch(() => ({ error: "CHECK_UNAVAILABLE" }));
    throw new Error(messages[error.error] || messages.CHECK_UNAVAILABLE);
  }
  const result = await response.json();
  if (path === "pair" || path === "session") csrf = result.csrf;
  return result;
}
