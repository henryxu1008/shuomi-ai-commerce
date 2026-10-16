// Extracted from agent-customization/pending: bounded bytes are a recovery clue,
// never authorization or a business fact. Each caller validates its own intent.
const payloads = new Map<string, { digest: string; body: string }>();
const memoryBound = 18 * 1024 * 1024;
const memoryKey = (storageKey: string, key: string) =>
  JSON.stringify([storageKey, key]);

export async function retainPendingPayload(
  storageKey: string,
  key: string,
  body: string,
) {
  const bytes = new TextEncoder().encode(body);
  if (bytes.length > 9 * 1024 * 1024) throw new Error("Command exceeds bound");
  const digest = Array.from(
    new Uint8Array(await crypto.subtle.digest("SHA-256", bytes)),
    (v) => v.toString(16).padStart(2, "0"),
  ).join("");
  const identity = memoryKey(storageKey, key),
    original = payloads.get(identity);
  if (original && original.digest !== digest)
    throw new Error("Original command differs");
  const inline = body.length <= 131072 ? body : undefined;
  if (inline === undefined) {
    let size = body.length;
    for (const [other, value] of payloads)
      if (other !== identity) size += value.body.length;
    for (const [other, value] of payloads) {
      if (size <= memoryBound) break;
      if (other !== identity) {
        payloads.delete(other);
        size -= value.body.length;
      }
    }
    payloads.set(identity, { digest, body });
  }
  return { digest, body: inline };
}
export function restorePendingPayload(
  storageKey: string,
  key: string,
  digest: string,
  inline?: string,
) {
  const cached = payloads.get(memoryKey(storageKey, key));
  return inline ?? (cached?.digest === digest ? cached.body : undefined);
}
export function forgetPendingPayload(storageKey: string, key: string) {
  payloads.delete(memoryKey(storageKey, key));
}
