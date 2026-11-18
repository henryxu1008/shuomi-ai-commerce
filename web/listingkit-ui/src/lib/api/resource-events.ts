import { isAcquisitionUUID } from "../contracts/product-acquisition";
import {z} from "zod";
import {aiPointUnit,displayPoints,signedDisplayPoints} from "../contracts/ai-points";
import {
  readCommercialBilling,
  CommercialBillingReadError,
} from "./commercial-billing";
const id = z.string().regex(/^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/);
const signed = z.string().refine((v) => {
  if (!/^(0|-?[1-9][0-9]{0,18})$/.test(v)) return false;
  const n = BigInt(v);
  return (
    n >= BigInt("-9223372036854775808") && n <= BigInt("9223372036854775807")
  );
});
const nonnegative = signed.refine((v) => /^(0|[1-9][0-9]*)$/.test(v));
const timestamp = z
  .string()
  .max(40)
  .regex(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d{1,9})?Z$/)
  .refine((v) => Number.isFinite(Date.parse(v)));
const resourceType = z.enum(["store_renewal_period", "ai_point", "data_row"]);
const text = z
  .string()
  .min(1)
  .max(512)
  .refine((v) => !/[\u0000-\u001f\u007f-\u009f]/.test(v));
const modelPurpose = z.enum(["chat_intent", "chat_planning", "own_product_intake", "skill_authoring"]);
const event = z
  .object({
    event_id: id,
    operation_id: text,
    resource_type: resourceType,
 unit_version:aiPointUnit.optional(),
    quantity: displayPoints,
    available_delta: signedDisplayPoints,
    allocated_delta: signedDisplayPoints,
    reserved_delta: signedDisplayPoints,
    consumed_delta: signedDisplayPoints,
    available_after: displayPoints,
    allocated_after: displayPoints,
    reserved_after: displayPoints,
    consumed_after: displayPoints,
    reason: text,
    source_type: text,
    source_identity: text,
    purpose: modelPurpose.optional(),
    occurred_at: timestamp,
  })
  .strict().refine(value=>value.resource_type==="ai_point"?value.unit_version==="ai-micropoint-v1":[value.quantity,value.available_after,value.allocated_after,value.reserved_after,value.consumed_after].every(v=>nonnegative.safeParse(v).success)&&[value.available_delta,value.allocated_delta,value.reserved_delta,value.consumed_delta].every(v=>signed.safeParse(v).success)&&value.unit_version===undefined)
  .refine((value) => !value.purpose || (value.resource_type === "ai_point" && value.source_type === "model_invocation_v1"));
const page = z
  .object({
    organization_id: id,
    items: z.array(event).max(50),
    next_cursor: z.string().min(1).max(1024).nullable(),
  })
  .strict()
  .refine(
    (v) => new Set(v.items.map((e) => e.event_id)).size === v.items.length,
  );
export type ResourceEventPage = z.infer<typeof page>;
export type ResourceEventFilters = {
  deductionsOnly?: boolean;
  resourceType?: z.infer<typeof resourceType>;
  from?: string;
  until?: string;
  cursor?: string;
};
export const RESOURCE_EVENTS_MAX_BYTES = 128 * 1024;
export function parseResourceEvents(v: unknown) {
  const result = page.safeParse(v);
  return result.success ? result.data : null;
}
export function getResourceEvents(
  user: string,
  org: string,
  filters: ResourceEventFilters = {},
  signal?: AbortSignal,
) {
  if (
    (filters.resourceType &&
      !resourceType.safeParse(filters.resourceType).success) ||
    (filters.from && !timestamp.safeParse(filters.from).success) ||
    (filters.until && !timestamp.safeParse(filters.until).success) ||
    (filters.from &&
      filters.until &&
      Date.parse(filters.from) >= Date.parse(filters.until)) ||
    (filters.cursor && filters.cursor.length > 1024)
  )
    throw new CommercialBillingReadError(400, "INVALID_REQUEST");
  const query = new URLSearchParams({ limit: "50", include_purpose: "true" });
  if (filters.resourceType) query.set("resource_type", filters.resourceType);
  if (filters.deductionsOnly) query.set("deductions_only", "true");
  if (filters.from) query.set("from", filters.from);
  if (filters.until) query.set("until", filters.until);
  if (filters.cursor) query.set("cursor", filters.cursor);
  return readCommercialBilling(
    `/api/workbench/commercial/resources/events?${query}`,
    user,
    org,
    parseResourceEvents,
    signal,
    RESOURCE_EVENTS_MAX_BYTES,
  );
}

const ownedModelUsage = z.strictObject({
  organization_id: id,
  items: z.array(z.strictObject({
    invocation_id: id, started_at: timestamp, purpose: text, model: text,
    status: z.enum(["observed", "not_sent", "unknown"]),
    input_tokens: z.number().int().nonnegative().max(1048576).optional(),
    output_tokens: z.number().int().nonnegative().max(65536).optional(),
  }).refine(item => item.status === "observed"
    ? item.input_tokens !== undefined && item.output_tokens !== undefined
    : item.input_tokens === undefined && item.output_tokens === undefined)).max(30),
  next_cursor: z.string().regex(/^[A-Za-z0-9_-]{1,240}$/).nullable(),
}).refine(v => new Set(v.items.map(item => item.invocation_id)).size === v.items.length);

export function parseOwnedModelUsage(payload: unknown) {
  const parsed = ownedModelUsage.safeParse(payload);
  return parsed.success ? parsed.data : null;
}

export function getOwnedModelUsage(user: string, org: string, cursor: string, signal?: AbortSignal) {
  const query = cursor ? "?" + new URLSearchParams({ cursor }) : "";
  return readCommercialBilling(
    "/api/workbench/commercial/resources/owned-model-usage" + query,
    user, org, parseOwnedModelUsage, signal, 32 * 1024,
  );
}

const objectId = z.string().refine(isAcquisitionUUID);
const objectName = text.refine((value) => value.trim() === value && [...value].length <= 240);
const businessObject = z.discriminatedUnion("kind", [
  z.object({ kind: z.literal("store"), name: objectName, id: objectId }).strict(),
  z.object({ kind: z.literal("amazon_product"), name: objectName }).strict(),
  z.object({ kind: z.literal("task"), name: objectName, id: objectId }).strict(),
]);
const usageObjectResponse = z.object({
  organization_id: id,
  event_id: objectId,
  object: businessObject.nullable(),
  purpose: modelPurpose.optional(),
}).strict().refine(({ object, purpose }) => object === null || purpose === undefined);

export type ResourceBusinessObject = z.infer<typeof businessObject>;

export function parseResourceBusinessObject(payload: unknown, eventId: string) {
  const result = usageObjectResponse.safeParse(payload);
  return result.success && result.data.event_id === eventId ? result.data : null;
}

export function readResourceBusinessObject(
  userId: string,
  organizationId: string,
  eventId: string,
  signal?: AbortSignal,
) {
  if (!isAcquisitionUUID(eventId)) throw new CommercialBillingReadError(400, "INVALID_REQUEST");
  return readCommercialBilling(
    `/api/workbench/commercial/resources/events/${eventId}/object`,
    userId,
    organizationId,
    (payload) => parseResourceBusinessObject(payload, eventId),
    signal,
    4096,
  );
}
