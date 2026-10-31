import {
  fulfillmentID,
  fulfillmentInput,
  type FulfillmentInput,
  type FulfillmentScope,
} from "./fulfillment";
function key(scope: FulfillmentScope) {
  if (
    !fulfillmentID.safeParse(scope.organizationId).success ||
    !fulfillmentID.safeParse(scope.userId).success
  )
    throw new Error("Invalid scope");
  return `listingkit.fulfillment.intent:${scope.organizationId}:${scope.userId}`;
}
function parse(raw: string | null): FulfillmentInput | null {
  if (!raw || new TextEncoder().encode(raw).length > 256 * 1024) return null;
  try {
    const value = fulfillmentInput.safeParse(JSON.parse(raw));
    return value.success ? value.data : null;
  } catch {
    return null;
  }
}
export function loadFulfillmentIntent(
  scope: FulfillmentScope,
): FulfillmentInput | null {
  try {
    return parse(localStorage.getItem(key(scope)));
  } catch {
    return null;
  }
}
export async function saveFulfillmentIntent(
  scope: FulfillmentScope,
  input: FulfillmentInput | null,
  expected: FulfillmentInput | null,
): Promise<boolean> {
  try {
    if (!navigator.locks) return false;
    return await navigator.locks.request(key(scope), async () => {
      const raw = localStorage.getItem(key(scope)),
        prior = parse(raw);
      if (raw !== null && !prior) return false;
      if (input === null) {
        if (prior && JSON.stringify(prior) !== JSON.stringify(expected))
          return false;
        localStorage.removeItem(key(scope));
        return true;
      }
      const parsed = fulfillmentInput.safeParse(input);
      if (
        !parsed.success ||
        (prior && JSON.stringify(prior) !== JSON.stringify(parsed.data))
      )
        return false;
      const next = JSON.stringify(parsed.data);
      if (new TextEncoder().encode(next).length > 256 * 1024) return false;
      localStorage.setItem(key(scope), next);
      return localStorage.getItem(key(scope)) === next;
    });
  } catch {
    return false;
  }
}
