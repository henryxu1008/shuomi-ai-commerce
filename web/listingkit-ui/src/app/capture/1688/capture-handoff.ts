import { isAcquisitionUUID } from "@/lib/contracts/product-acquisition";

export type CaptureHandoff = Readonly<{ kind: "handoff"; extensionId: string; handoffId: string; key: string }>;
export type CaptureEntry = CaptureHandoff | Readonly<{ kind: "recovery"; key: string }>;
export function parseCaptureEntry(rawURL: string, platform: "1688" | "amazon" = "1688"): CaptureEntry | null {
  try {
    const url = new URL(rawURL);
    if (!["1688","amazon"].includes(platform) || !["http:", "https:"].includes(url.protocol) || url.username || url.password || url.pathname !== `/capture/${platform}` || url.search || rawURL.split("#")[0]!.endsWith("?")) return null;
    const params = new URLSearchParams(url.hash.slice(1));
    const names = [...params.keys()];
    if (new Set(names).size !== names.length) return null;
    const recovery = params.get("operationKey");
    if (names.length === 1 && recovery && isAcquisitionUUID(recovery)) return { kind: "recovery", key: recovery };
    if (names.length !== 3 || !names.every((name) => ["extensionId", "handoffId", "idempotencyKey"].includes(name))) return null;
    const extensionId = params.get("extensionId") ?? "", handoffId = params.get("handoffId") ?? "", key = params.get("idempotencyKey") ?? "";
    return /^[a-p]{32}$/.test(extensionId) && isAcquisitionUUID(handoffId) && isAcquisitionUUID(key) ? { kind: "handoff", extensionId, handoffId, key } : null;
  } catch { return null; }
}
