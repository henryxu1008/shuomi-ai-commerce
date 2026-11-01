import { z } from "zod";
import { readBoundedStrictJSON } from "./strict-json-response";
export class KernelError extends Error { constructor(public code: string, public status = 0) { super(code); } }
export async function kernelRequest<T>(userId: string, path: string, schema: z.ZodType<T>, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers); headers.set("X-Expected-User-ID", userId); headers.set("Accept", "application/json");
  try {
    const response = await fetch("/api/account/kernel-releases" + path, { ...init, headers, credentials: "same-origin", cache: "no-store", redirect: "error" });
    const raw = await readBoundedStrictJSON(response, 256 * 1024, init.signal ?? undefined);
    if (!response.ok) { const error = z.object({ code: z.string() }).safeParse(raw); throw new KernelError(error.success ? error.data.code : "UNAVAILABLE", response.status); }
    if (response.status !== 200) throw new Error(); return schema.parse(raw);
  } catch (error) { if (error instanceof KernelError) throw error; throw new KernelError(init.method === "PUT" ? "OUTCOME_UNKNOWN" : "UNAVAILABLE"); }
}
export function kernelMessage(error: unknown) {
  const code = error instanceof KernelError ? error.code : "UNAVAILABLE";
  if (code === "NOT_SENT") return "本次登记尚未发送。输入保留，服务恢复后可按原编号重试。";
  return ({ FORBIDDEN: "无平台维护权限。", AUTHENTICATION_REQUIRED: "登录已失效，请重新登录。", IDENTITY_CONTEXT_CHANGED: "登录身份已变化，请重新打开页面。", INVALID_REQUEST: "参数不符合登记要求，请检查后重试。", KERNEL_RELEASE_CONFLICT: "此编号已有不同元数据，请核对原记录。", KERNEL_RELEASE_WITHDRAWN: "此编号已下架，不能重新登记。", KERNEL_RELEASE_NOT_FOUND: "尚未读到本次记录，登记结果仍待核实。", READBACK_MISMATCH: "读回元数据与本次登记不一致，请核对原编号。", OUTCOME_UNKNOWN: "登记结果待核实。保留原编号和参数，请查询本次登记。" } as Record<string, string>)[code] ?? "服务暂不可用。输入和原编号保留，请稍后重新读取。";
}
