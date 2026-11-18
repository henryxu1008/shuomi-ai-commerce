import { z } from "zod";

const uuid = z.uuid().refine((value) => value === value.toLowerCase());
const firstPage = /^\/api\/workbench\/research\/projects\/([0-9a-f-]{36})\/amazon-first-page\/([0-9a-f-]{36})$/;
const receipt = z.strictObject({
  approvalId: uuid,
  state: z.enum(["UNKNOWN", "REJECTED", "OVER_LIMIT", "INSUFFICIENT", "PREPARED"]),
  requestId: z.string().regex(/^[A-Za-z0-9_-]{1,128}$/).optional(),
  credits: z.number().int().min(0).max(2_147_483_647).nullable(),
  jobId: uuid.optional(),
}).refine((value) => {
  if (value.credits !== null && !value.requestId) return false;
  if (value.state === "PREPARED") return !!value.jobId && !!value.requestId && (value.credits ?? 0) <= 25;
  if (value.jobId) return false;
  return value.state !== "OVER_LIMIT" || (value.credits ?? 0) > 25;
});

export function amazonFirstPagePath(path: string) {
  const match = firstPage.exec(path);
  if (!match || !uuid.safeParse(match[1]).success || !uuid.safeParse(match[2]).success) return null;
  return { project: match[1], approval: match[2] };
}

export function amazonFirstPageEndpoint(url: URL, method: string) {
  const target = amazonFirstPagePath(url.pathname);
  if (!target || url.search || (method !== "GET" && method !== "POST")) return null;
  return {
    path: "/" + target.project + "/amazon-first-page/" + target.approval,
    input: undefined,
    output: receipt.refine((value) => value.approvalId === target.approval),
    expected: false,
    emptyWrite: method === "POST",
  };
}
