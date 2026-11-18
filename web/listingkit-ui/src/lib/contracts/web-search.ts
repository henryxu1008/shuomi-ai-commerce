import { z } from "zod";
import { routeFact } from "./model-route-pricing";
import { isAcquisitionUUID } from "./product-acquisition";

const rev = z.number().int().nonnegative().max(Number.MAX_SAFE_INTEGER);
const owner = z.strictObject({ organizationId: routeFact, actorId: routeFact, memberId: routeFact });
const envelope = owner.extend({ requestKey: z.string() });
const conversation = owner.extend({ conversationId: z.string().refine(isAcquisitionUUID) });
const usage = z.union([z.literal(0), z.literal(1)]).nullable();
const source = z.enum(["platform", "owned"]);
const keyword = z.string().min(1).max(400);
const digest = z.string().regex(/^[0-9a-f]{64}$/);
const validity = z.strictObject({ revision: rev, enabled: z.boolean(), keyPresent: z.boolean(), updatedAt: z.string().datetime({ offset: true }).nullable() });
const publication = z.strictObject({
  revision: rev, name: z.string().max(128), country: z.string().max(3), language: z.string().max(8),
  maxUsdMicros: rev, totalLimit: rev, tenantLimit: rev, organizations: z.array(routeFact).max(64).nullable(), published: z.boolean(),
});
const selection = z.strictObject({ revision: rev, source: z.union([source, z.literal("")]) });
const option = z.strictObject({ source: z.union([source, z.literal("")]), name: z.string().max(128), available: z.boolean(), limit: rev, preview: digest.optional() });
export const searchReceipt = z.strictObject({
  key: z.string().refine(isAcquisitionUUID), query: keyword, source, status: z.enum(["held", "claimed", "not_sent", "unknown", "complete", "no_results"]),
  startedAt: z.string().datetime({ offset: true }), usage,
});
export const searchSettingsResponse = envelope.extend({ result: z.strictObject({ selection, options: z.array(option).max(2) }) });
export const searchSelectionResponse = envelope.extend({ result: selection });
export const searchPlatformResponse = envelope.extend({ result: publication, canWrite: z.boolean(), organizations: z.array(routeFact).nullable() });
export const searchPublicationResponse = envelope.extend({ result: publication });
export const searchKeyResponse = envelope.extend({ result: validity });
export const searchChatRead = conversation.extend({ option, result: z.array(searchReceipt).max(20) });
export const searchChatCommand = conversation.extend({ result: z.strictObject({
  receipt: searchReceipt,
  results: z.array(z.strictObject({
    title: z.string().max(400), snippet: z.string().max(1000),
    url: z.string().max(2048).url().refine(value => /^https?:\/\//.test(value)),
  })).max(5).optional(),
}) });
export const searchChatReceipt = conversation.extend({ result: searchReceipt });

const withMember = { expectedMemberId: routeFact };

export function publicSearchEndpoint(parts: string[], method: string) {
  const [family, action, segment, key] = parts;
  if (family !== "search" || (method !== "GET" && method !== "POST")) return null;
  const write = method === "POST";
  if (action === "chat" && isAcquisitionUUID(segment) && (parts.length === 3 || !write && parts.length === 4 && isAcquisitionUUID(key))) {
    return {
      path: "/api/v1/workbench/chat/conversations/" + segment + "/public-search" + (key ? "/" + key : ""),
      input: write ? z.strictObject({ query: keyword, preview: digest, ...withMember }) : undefined,
      output: write ? searchChatCommand : key ? searchChatReceipt : searchChatRead,
      conversationId: segment,
      receiptKey: key,
    };
  }
  if (!["enterprise", "platform", "platform-key", "owned-key"].includes(action)) return null;
  const receipt = !write && parts.length === 4 && segment === "receipts" && isAcquisitionUUID(key);
  if (!receipt && parts.length !== 2) return null;
  const input = action === "platform" ? publication.extend({ organizations: z.array(routeFact).max(64), ...withMember })
    : action === "enterprise" ? selection.extend({ source, ...withMember })
    : validity.pick({ revision: true, enabled: true }).extend({ apiKey: z.string().max(4096).optional(), ...withMember });
  const output = action === "platform" ? write || receipt ? searchPublicationResponse : searchPlatformResponse
    : action === "enterprise" ? write || receipt ? searchSelectionResponse : searchSettingsResponse
    : searchKeyResponse;
  return { path: "/api/v1/model-route-pricing/" + parts.join("/"), input: write ? input : undefined, output, conversationId: undefined, receiptKey: receipt ? key : undefined };
}
