import { describe, expect, it } from "vitest";
import { browserCaptureSchema, BROWSER_CAPTURE_MAX_BYTES } from "./browser-capture";
import { browserCaptureFixture } from "./browser-capture.fixture";

type MutablePayload = { [key: string]: unknown; evidence: Record<string, unknown> & { priceFacts: Array<Record<string, unknown>> } };

describe("Browser transport contract", () => {
  it("preserves strict v2 variant images without accepting a mixed version", () => {
    const original = browserCaptureFixture();
    const input = {captureVersion:2, evidence:{...original.evidence,schemaVersion:2,parserVersion:"1688-browser-dom/v3",variants:[{sourceID:"blue",sku:null,title:null,attributes:[{name:"颜色",value:"蓝色"}],price:null,images:[{url:"https://cbu01.alicdn.com/blue.jpg",role:"source"}]}]}};
    expect(browserCaptureSchema.parse(input)).toEqual(input);
    expect(browserCaptureSchema.safeParse({...input,captureVersion:1}).success).toBe(false);
    expect(browserCaptureSchema.safeParse({...input,evidence:{...input.evidence,schemaVersion:1}}).success).toBe(false);
    expect(browserCaptureSchema.safeParse({...input,evidence:{...input.evidence,variants:input.evidence.variants.map(v=>({...v,images:null}))}}).success).toBe(false);
  });
  it("retains original decimal/string/array lexemes without computing a digest", () => {
    const input = browserCaptureFixture();
    expect(browserCaptureSchema.parse(input)).toEqual(input);
    expect(BROWSER_CAPTURE_MAX_BYTES).toBe(2 * 1024 * 1024);
  });
  it.each([
    (p: MutablePayload) => { p.actor = "untrusted"; },
    (p: MutablePayload) => { p.evidence.orgId = "untrusted"; },
    (p: MutablePayload) => { p.evidence.attributes = null; },
    (p: MutablePayload) => { delete p.evidence.title; },
    (p: MutablePayload) => { p.evidence.title = "\ud800"; },
    (p: MutablePayload) => { p.evidence.title = "x".repeat(8193); },
    (p: MutablePayload) => { p.evidence.offerID = "42"; },
    (p: MutablePayload) => { p.evidence.parserVersion = "untrusted/v1"; },
    (p: MutablePayload) => { p.evidence.priceFacts[0].amount = 12; },
    (p: MutablePayload) => { p.evidence.images = Array(257).fill({ url: "x", role: "source" }); },
    (p: MutablePayload) => { p.evidence.variants = Array(5).fill({ sourceID: null, sku: null, title: null, price: null, attributes: Array(256).fill({ name: "x", value: "y" }) }); },
  ])("rejects malformed or authority-bearing payload %#", (change) => {
    const input = browserCaptureFixture(); change(input);
    expect(browserCaptureSchema.safeParse(input).success).toBe(false);
  });
  it("accepts paired unicode and optional minQuantity without changing evidence", () => {
    const input = browserCaptureFixture(); input.evidence.title = "\ud83d\ude00\ufffd";
    const price: Record<string, unknown> = input.evidence.priceFacts[0]!;
    delete price.minQuantity;
    expect(browserCaptureSchema.parse(input)).toEqual(input);
  });
});
