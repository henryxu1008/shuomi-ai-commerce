import { expect,it } from "vitest";
import { readFileSync } from "node:fs";
import { amazonCaptureResultSchema,amazonCaptureSchema,freezeAmazonCapture } from "./amazon-browser-capture";
const source=JSON.parse(readFileSync("../../internal/product/sourcing/testdata/amazon-capture-v2.json","utf8"));
const receipt=JSON.parse(readFileSync("../../internal/product/sourcing/testdata/amazon-capture-result-v1.json","utf8"));
it("matches Go canonical digest and preserves saved variant images in strict readback",async()=>{
 const frozen=await freezeAmazonCapture(source);expect(frozen.digest).toBe(receipt.payloadSHA256);
 const parsed=amazonCaptureResultSchema.parse(receipt);expect(parsed.snapshot.variants?.[0].images?.[0].url).toBe(source.evidence.variants[0].images[0].url);expect(parsed.catalogVersion).toBe("1");
 expect(parsed.snapshot).toEqual(receipt.snapshot);
 expect(()=>amazonCaptureResultSchema.parse({...receipt,snapshot:{...receipt.snapshot,unexpected:true}})).toThrow();
});
it('freezes both offer conditions with deterministic fields',async()=>{
 const input=structuredClone(source);input.evidence.variants[0].price={amount:'35.98',currency:'USD',condition:'new',alternative:{amount:'31.99',currency:null,condition:'used',conditionNote:'Very Good'}};
 const frozen=await freezeAmazonCapture(input);
 expect(frozen.digest).toBe('da9e2e942f912cc06c7b124676149456b602739e98d3243ae1ebc14f504eb3f9');
 expect(frozen.body).toContain('"price":{"amount":"35.98","currency":"USD","minQuantity":null,"condition":"new","alternative":{"amount":"31.99","currency":null,"minQuantity":null,"condition":"used","conditionNote":"Very Good"}}');
});
it('counts nested alternative offers toward the common aggregate limit',()=>{
 const input=structuredClone(source),e=input.evidence;e.observation={scope:'partial_family',parentASIN:'B012345679'};
 e.variants=Array.from({length:171},(_,i)=>({sourceID:`B${String(i).padStart(9,'0')}`,sku:null,title:null,attributes:[{name:'Color',value:`c${i}`}],images:[],price:{amount:'35.98',currency:'USD',condition:'new',alternative:{amount:'31.99',currency:'USD',condition:'used'}}}));
 e.attributes=[];e.priceFacts=[];e.images=[];e.warnings=[];e.missingFacts=[];expect(amazonCaptureSchema.safeParse(input).success).toBe(true);
 e.attributes=Array.from({length:256},(_,i)=>({name:`a${i}`,value:'x'}));e.images=Array.from({length:256},()=>({url:'https://m.media-amazon.com/images/I/fixture.jpg',role:'gallery'}));expect(amazonCaptureSchema.safeParse(input).success).toBe(false);
});
