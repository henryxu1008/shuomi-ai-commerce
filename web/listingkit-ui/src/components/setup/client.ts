export type Module = { id: string; label: string; configured: boolean; installed: string; enabled: boolean; uiEnabled: boolean | null; mutable: boolean; note: string };
export type Instance = { id: string; project: string; origin: string; backendImage: string; uiImage: string; modules?: Module[]; blocks: string[]; pending?: Plan };
export type Plan = { id: string; instanceId: string; mode: string; status: string; changes: { label: string; before: string; after: string }[]; blocks: string[]; services: string[]; updatedAt: string };
export type Environment = { dockerReady: boolean; hostFreeBytes: number | null; version: string; instances: Instance[]; profiles: { id: string; label: string; note: string }[]; blocks: string[]; pending: Plan[] };
export type SetupRequest = { mode: "fresh" | "existing"; instanceId?: string; project?: string; profile?: string; applicationPort?: number; identityPort?: number; mailPort?: number; configurationFile?: string; toolReleaseDirectory?: string; networkSubnet?: string; enabled?: Record<string, boolean> };

const messages: Record<string, string> = {
  INVALID_REQUEST: "配置未通过检查。请核对项目、端口、模块依赖及私密文件；新装组合必须使用新空项目。",
  CONFIGURATION_CHANGED: "实际配置或候选版本已改变。未覆盖现有实例，请重新检查并预览。",
  INSTANCE_BUSY: "已有本机操作正在执行，请查看其原回执。",
  ORIGINAL_OPERATION_UNRESOLVED: "原操作尚未核实。请查看原回执并选择“核实原操作”，不要重新安装。",
  CHECK_UNAVAILABLE: "检查未完成。请核对Docker、准确版本包及原部署文件；未输出配置或凭据。",
};
let csrf = "";
export async function api<T>(path: string, body?: unknown): Promise<T> {
  const response = await fetch(`/api/${path}`, { method: body === undefined ? "GET" : "POST", credentials: "same-origin", cache: "no-store", headers: body === undefined ? {} : { "Content-Type": "application/json", "X-Setup-CSRF": csrf }, ...(body === undefined ? {} : { body: JSON.stringify(body) }) });
  if (!response.ok) {
    if (response.status === 401) throw new Error("本机配对已失效，请重新启动安装管理并配对。操作回执会保留。");
    if (response.status === 403) throw new Error("配对或本机请求校验未通过。请使用本机启动窗口中的一次性配对码。");
    const error = await response.json().catch(() => ({ error: "CHECK_UNAVAILABLE" }));
    throw new Error(messages[error.error] || messages.CHECK_UNAVAILABLE);
  }
  const result = await response.json();
  if (path === "pair" || path === "session") csrf = result.csrf;
  return result;
}
