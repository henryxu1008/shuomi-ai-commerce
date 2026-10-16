import { z } from "zod";
import {
  toolEndpoint,
  toolVersion,
  maxToolCreateBytes,
} from "@/lib/contracts/tool-market";
import {
  retainPendingPayload,
  restorePendingPayload,
  forgetPendingPayload,
} from "../resources/pending-payload";
const pathFields = z.strictObject({
  path: z.string().max(256),
  key: z.uuid(),
  revision: toolVersion.optional(),
  absent: z.boolean().optional(),
});
function validPath(i: z.infer<typeof pathFields>) {
  const route = toolEndpoint(
    new URL("/api/tool-market/" + i.path, "https://pending.invalid"),
    i.path.startsWith("activations/") ? "PUT" : "POST",
  );
  return route?.input &&
    route.operation &&
    !(i.absent && i.revision) &&
    !!(i.absent || i.revision) &&
    !(route.operation === "create" && !i.absent) &&
    !(route.operation === "progress" && i.absent)
    ? route
    : null;
}
const intentSchema = pathFields
  .extend({ body: z.string().max(maxToolCreateBytes) })
  .refine((i) => {
    const route = validPath(i);
    if (!route) return false;
    try {
      return (
        route.input!.safeParse(JSON.parse(i.body)).success &&
        (route.operation === "create" ||
          new TextEncoder().encode(i.body).length <= 16384)
      );
    } catch {
      return false;
    }
  });
export type ToolIntent = z.infer<typeof intentSchema>;
export const toolPendingSchema = pathFields
  .extend({
    digest: z.string().regex(/^[a-f0-9]{64}$/),
    body: z.string().max(131072).optional(),
  })
  .refine((i) => {
    const { digest, ...intent } = i;
    void digest;
    return (
      !!validPath(i) &&
      (i.body === undefined || intentSchema.safeParse(intent).success)
    );
  });
export async function freezeToolIntent(storageKey: string, intent: ToolIntent) {
  const valid = intentSchema.parse(intent);
  return toolPendingSchema.parse({
    ...valid,
    ...(await retainPendingPayload(storageKey, valid.key, valid.body)),
  });
}
export function restoreToolIntent(
  storageKey: string,
  record: z.infer<typeof toolPendingSchema> | null,
): ToolIntent | null {
  if (!record) return null;
  const body = restorePendingPayload(
    storageKey,
    record.key,
    record.digest,
    record.body,
  );
  if (body === undefined) return null;
  const { digest: _digest, ...fields } = record;
  void _digest;
  const parsed = intentSchema.safeParse({ ...fields, body });
  return parsed.success ? parsed.data : null;
}
export function forgetToolIntent(storageKey: string, key: string) {
  forgetPendingPayload(storageKey, key);
}
