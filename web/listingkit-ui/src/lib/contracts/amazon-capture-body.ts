// Shared versioned transport serialization. Both the application freezer and
// the extension's durable intent check use these exact Go JSON encoding rules.
type Offer = { amount: string; currency: string | null; minQuantity?: string | null; condition?: 'new' | 'used'; conditionNote?: string };
type Price = Offer & { alternative?: Offer };
export function amazonCaptureBody<T extends { evidence: { variants: { price: Price | null }[]; priceFacts: Price[] } }>(input: T): string {
  const value = structuredClone(input);
  const offer = (p: Offer) => ({ amount: p.amount, currency: p.currency, minQuantity: p.minQuantity ?? null, ...(p.condition ? { condition: p.condition } : {}), ...(p.conditionNote ? { conditionNote: p.conditionNote } : {}) });
  const price = (p: Price) => ({ ...offer(p), ...(p.alternative ? { alternative: offer(p.alternative) } : {}) });
  for (const v of value.evidence.variants) if (v.price) v.price = price(v.price);
  value.evidence.priceFacts = value.evidence.priceFacts.map(price);
  return JSON.stringify(value).replace(/[<>&\u2028\u2029]/g, c => `\\u${c.charCodeAt(0).toString(16).padStart(4, '0')}`);
}
