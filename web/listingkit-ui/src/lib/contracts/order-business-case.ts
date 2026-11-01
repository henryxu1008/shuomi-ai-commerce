import { z } from "zod";
import { fulfillmentID } from "@/lib/api/fulfillment";

// A reference to existing synthetic facts, never a source of quantities/status.
export const orderBusinessCase = z.object({
  synthetic: z.literal(true),
  title: z.string().trim().min(1).max(120),
  organizationId: fulfillmentID,
  storeId: fulfillmentID,
  sourceId: fulfillmentID,
  kind: z.enum(["consumer_order", "platform_supply"]),
}).strict();
export type OrderBusinessCase = z.infer<typeof orderBusinessCase>;
