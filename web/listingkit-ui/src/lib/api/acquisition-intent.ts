import { z } from "zod";
import { ACQUISITION_BODY_MAX_BYTES, canonical1688Source, isAcquisitionUUID } from "../contracts/product-acquisition";
import type { AcquisitionOperation } from "./product-acquisition";

export type PendingAcquisitionIntent = AcquisitionOperation & { quotaPaused?: boolean };
const storageKey = "listingkit.acquisition.intent";
const identity = z.string().regex(/^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/);
const envelope = z.object({
  userId: identity, organizationId: identity,
  key: z.string().refine(isAcquisitionUUID),
  source: z.string().refine(source => canonical1688Source(source) === source),
  quotaPaused: z.boolean().optional(),
}).strict();
function parse(raw: string | null): PendingAcquisitionIntent | null {
  if (!raw || new TextEncoder().encode(raw).length > ACQUISITION_BODY_MAX_BYTES + 1024) return null;
  try { const result = envelope.safeParse(JSON.parse(raw)); return result.success ? result.data : null; }
  catch { return null; }
}
function sameOperation(a: PendingAcquisitionIntent, b: PendingAcquisitionIntent | null) {
  return b !== null && a.userId === b.userId && a.organizationId === b.organizationId && a.key === b.key && a.source === b.source;
}
export function loadAcquisitionIntent(): PendingAcquisitionIntent | null {
  try { return parse(localStorage.getItem(storageKey)); } catch { return null; }
}
export function saveAcquisitionIntent(intent: PendingAcquisitionIntent | null, expected: PendingAcquisitionIntent | null): boolean {
  try {
    const rawPrior = localStorage.getItem(storageKey);
    const prior = parse(rawPrior);
    // Invalid or foreign storage is never silently discarded to dispatch a new key.
    if (rawPrior !== null && !prior) return false;
    if (intent === null) {
      if (prior && !sameOperation(prior, expected)) return false;
      localStorage.removeItem(storageKey);
      return true;
    }
    const raw = JSON.stringify(intent);
    const parsed = parse(raw);
    if (!parsed || prior && !sameOperation(prior, parsed)) return false;
    localStorage.setItem(storageKey, raw);
    return true;
  } catch { return false; }
}
