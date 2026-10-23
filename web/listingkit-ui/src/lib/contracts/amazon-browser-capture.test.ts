import { expect,it } from "vitest";
import { readFileSync } from "node:fs";
import { amazonCaptureResultSchema,freezeAmazonCapture } from "./amazon-browser-capture";
const source=JSON.parse(readFileSync("../../internal/product/sourcing/testdata/amazon-capture-v2.json","utf8"));
const receipt=JSON.parse(readFileSync("../../internal/product/sourcing/testdata/amazon-capture-result-v1.json","utf8"));
it("matches Go canonical digest and preserves saved variant images in strict readback",async()=>{
 const frozen=await freezeAmazonCapture(source);expect(frozen.digest).toBe(receipt.payloadSHA256);
 const parsed=amazonCaptureResultSchema.parse(receipt);expect(parsed.snapshot.variants?.[0].images?.[0].url).toBe(source.evidence.variants[0].images[0].url);expect(parsed.catalogVersion).toBe("1");
 expect(parsed.snapshot).toEqual(receipt.snapshot);
 expect(()=>amazonCaptureResultSchema.parse({...receipt,snapshot:{...receipt.snapshot,unexpected:true}})).toThrow();
});
