import "server-only";
import { orderBusinessCase } from "@/lib/contracts/order-business-case";

export function configuredOrderBusinessCase() {
  const raw = process.env.LISTINGKIT_ORDER_BUSINESS_CASE?.trim();
  if (!raw) return undefined;
  try {
    if (raw.length > 2048) return undefined;
    return orderBusinessCase.parse(JSON.parse(raw));
  } catch {
    console.warn("[orders] invalid business case reference");
    return undefined;
  }
}
