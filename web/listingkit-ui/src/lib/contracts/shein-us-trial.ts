import { z } from "zod";

export const trialUUID = z.uuid().refine((value) => value === value.toLowerCase());
export const expectedTrialApprovalHeader = "X-Expected-SHEIN-Trial-Approval-ID";
const credits = z.number().int().nonnegative().max(2_147_483_647);
const receiptFields = {
  approvalId: trialUUID,
  requestId: z.string().regex(/^[A-Za-z0-9_-]{1,128}$/),
};

export const sheinTrialReceipt = z.discriminatedUnion("state", [
  z.strictObject({ ...receiptFields, requestId: z.union([receiptFields.requestId, z.literal("")]), state: z.literal("UNKNOWN"), credits: z.null(), reservedCredits: z.literal(25) }),
  z.strictObject({ ...receiptFields, state: z.literal("CONFIRMED"), credits: credits.max(25), reservedCredits: z.literal(0), publishedCount: z.number().int().min(0).max(20).optional() }),
  z.strictObject({ ...receiptFields, state: z.literal("OVER_LIMIT"), credits: credits.min(26), reservedCredits: z.literal(0), publishedCount: z.literal(0).optional() }),
]);
export const sheinTrialState = z.union([
  sheinTrialReceipt,
  z.strictObject({ approvalId: trialUUID, state: z.literal("NOT_STARTED") }),
]);
export type SheinTrialState = z.infer<typeof sheinTrialState>;
export const sheinTrialError = z.strictObject({ code: z.enum([
  "AUTHENTICATION_REQUIRED", "IDENTITY_CONTEXT_CHANGED", "ORGANIZATION_CONTEXT_CHANGED", "INVALID_REQUEST",
  "FORBIDDEN", "NOT_FOUND", "SHEIN_TRIAL_CONFLICT", "SHEIN_TRIAL_UNAVAILABLE", "APPROVAL_CONTEXT_CHANGED", "OUTCOME_UNKNOWN",
]) });
