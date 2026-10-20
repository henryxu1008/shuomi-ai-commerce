import { z } from "zod";
import { customId, customVersion, customInput, customUpdate } from "@/lib/api/agent-customization";
import { retainPendingPayload, restorePendingPayload, forgetPendingPayload } from "../resources/pending-payload";
export type CustomIntent = {
    key: string;
    body: string;
    path: string;
    version?: string;
};
export const customizationPending = z.strictObject({ key: customId, path: z.string().max(64), version: customVersion.optional(), digest: z.string().regex(/^[a-f0-9]{64}$/), body: z.string().max(131072).optional() }).refine(v => v.path === "" && !v.version || /^\/[a-f0-9-]{36}\/progress$/.test(v.path) && customId.safeParse(v.path.split("/")[1]).success && !!v.version);
export type CustomPending = z.infer<typeof customizationPending>;
// Large files remain in feature-local memory across navigation. A small session
// marker survives reload and blocks a new command if its bytes are unavailable.
// Small commands can be recovered in full without storing large contact files.
export async function freezeCustomization(storageKey: string, i: CustomIntent): Promise<CustomPending> {
    const bytes = new TextEncoder().encode(i.body);
    if (bytes.length > 9 * 1024 * 1024)
        throw new Error("Command exceeds bound");
    const parsed = JSON.parse(i.body);
    if (!(i.path === "" ? customInput : customUpdate).safeParse(parsed).success)
        throw new Error("Invalid command");
    return customizationPending.parse({ ...i, ...await retainPendingPayload(storageKey,i.key,i.body) });
}
export function restoreCustomization(storageKey: string, record: CustomPending | null): CustomIntent | null {
    if (!record)
        return null;
    const body = restorePendingPayload(storageKey,record.key,record.digest,record.body);
    return body === undefined ? null : { key: record.key, path: record.path, version: record.version, body };
}
export function forgetCustomization(storageKey: string, key: string) { forgetPendingPayload(storageKey,key); }
