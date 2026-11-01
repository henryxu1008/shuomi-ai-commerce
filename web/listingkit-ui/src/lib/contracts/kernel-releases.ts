import { z } from "zod";

export const kernelID = z.string().regex(/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/, "请输入规范的小写 UUID。");
const numeric = "(?:0|[1-9][0-9]*)";
const identifier = "(?:0|[1-9][0-9]*|[0-9A-Za-z-]*[A-Za-z-][0-9A-Za-z-]*)";
const semver = new RegExp(`^${numeric}\\.${numeric}\\.${numeric}(?:-${identifier}(?:\\.${identifier})*)?(?:\\+[0-9A-Za-z-]+(?:\\.[0-9A-Za-z-]+)*)?$`);
export const kernelClientVersion = z.string().max(64, "版本号不能超过 64 个字符。").regex(semver, "请输入完整的客户端版本号，例如 1.8.0。");
function compareNumeric(a: string, b: string) { return a.length - b.length || (a < b ? -1 : a > b ? 1 : 0); }
export function compareKernelVersions(a: string, b: string): number {
  const [ac, ...ar] = a.split("+", 1)[0].split("-"), [bc, ...br] = b.split("+", 1)[0].split("-");
  const ap = ar.length ? ar.join("-") : undefined, bp = br.length ? br.join("-") : undefined;
  const aa = ac.split("."), bb = bc.split(".");
  for (let i = 0; i < 3; i++) { const n = compareNumeric(aa[i], bb[i]); if (n) return n; }
  if (ap === undefined || bp === undefined) return ap === bp ? 0 : ap === undefined ? 1 : -1;
  const ax = ap.split("."), bx = bp.split(".");
  for (let i = 0; i < Math.max(ax.length, bx.length); i++) {
    if (ax[i] === undefined || bx[i] === undefined) return ax.length - bx.length;
    if (ax[i] === bx[i]) continue;
    const an = /^[0-9]+$/.test(ax[i]), bn = /^[0-9]+$/.test(bx[i]);
    return an && bn ? compareNumeric(ax[i], bx[i]) : an !== bn ? an ? -1 : 1 : ax[i] < bx[i] ? -1 : 1;
  }
  return 0;
}
const platform = { os: z.enum(["windows", "linux", "darwin"]), arch: z.enum(["amd64", "arm64"]) };
const version = z.string().min(1, "请填写内核版本。").max(64, "内核版本不能超过 64 个字符。").refine(v => v.trim() === v && !/[\u0000-\u001f\u007f-\u009f]/.test(v), "内核版本不能包含首尾空格或控制字符。");
const range = { minClientVersion: kernelClientVersion, maxClientVersionExclusive: z.union([z.literal(""), kernelClientVersion]) };
const validRange = (p: { minClientVersion: string; maxClientVersionExclusive: string }) => !p.maxClientVersionExclusive || semver.test(p.minClientVersion) && semver.test(p.maxClientVersionExclusive) && compareKernelVersions(p.minClientVersion, p.maxClientVersionExclusive) < 0;
export function kernelCompatible(client: string, p: { minClientVersion: string; maxClientVersionExclusive: string }) {
  return semver.test(client) && semver.test(p.minClientVersion) && (!p.maxClientVersionExclusive || semver.test(p.maxClientVersionExclusive)) && compareKernelVersions(client, p.minClientVersion) >= 0 && (!p.maxClientVersionExclusive || compareKernelVersions(client, p.maxClientVersionExclusive) < 0);
}
const fixedURL = z.string().max(2048).refine(v => {
  try { const u = new URL(v); return v === u.href && u.protocol === "https:" && !!u.hostname && !u.username && !u.password && !u.search && !v.includes("?") && !v.includes("#") && u.pathname !== "/"; } catch { return false; }
}, "请使用不含凭据、查询参数或片段的固定 HTTPS 文件地址。");
const metadata = { archiveFormat: z.enum(["zip", "tar.gz", "tar.xz"]), size: z.number().int("大小须为整数字节。").min(1, "大小须为正整数。").max(2 ** 31, "大小不能超过 2147483648 字节。"), sha256: z.string().regex(/^[0-9a-f]{64}$/, "请输入 64 位小写十六进制 SHA256。"), downloadUrl: fixedURL };
export const kernelPublish = z.strictObject({ version, ...platform, ...range, ...metadata }).refine(validRange, { message: "结束版本必须大于最低版本", path: ["maxClientVersionExclusive"] });
export const kernelQuery = z.strictObject({ ...platform, clientVersion: kernelClientVersion });
export const kernelSummary = z.strictObject({ releaseId: kernelID, version, ...platform, ...range, recommended: z.boolean() }).refine(validRange);
export const kernelPage = z.strictObject({ releases: z.array(kernelSummary).max(100), nextCursor: z.union([z.literal(""), kernelID]) });
export const kernelArtifact = z.strictObject({ ...kernelSummary.shape, ...metadata, downloadOrigins: z.array(z.string().max(256)).min(1).max(8) }).refine(validRange);
export const kernelResult = z.strictObject({ releaseId: kernelID, state: z.enum(["published", "withdrawn"]) });
export const kernelIntent = z.strictObject({ key: kernelID, clientVersion: kernelClientVersion, payload: kernelPublish }).refine(p => kernelCompatible(p.clientVersion, p.payload), { message: "读回客户端版本必须在兼容范围内", path: ["clientVersion"] });
export type KernelPublish = z.infer<typeof kernelPublish>;
export type KernelQuery = z.infer<typeof kernelQuery>;
export type KernelIntent = z.infer<typeof kernelIntent>;
export type KernelArtifact = z.infer<typeof kernelArtifact>;
export function matchesKernelArtifact(id: string, input: KernelPublish, readback: KernelArtifact) {
  return readback.releaseId === id && !readback.recommended && Object.entries(input).every(([k, value]) => readback[k as keyof KernelPublish] === value);
}
