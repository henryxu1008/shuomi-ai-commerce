import {expect,it} from "vitest";
import {priceInput,canonicalPriceFacts} from "./model-route-pricing";

it("preserves the published image estimate through preview and exact fact comparison",()=>{
 const estimate={inputTokens:100,outputTokens:100,operationScope:"product-image-effects-v1",size:"1024x1024",quality:"auto",allowAutoSize:true,maximumPromptBytes:65536,maximumReferenceImages:3,maximumReferenceBytes:16777216,maximumReferencePixels:64000000};
 const draft={expectedRevision:"NO_REVISION",prices:[{id:"synthetic",region:"global",serviceType:"image-sync",image:{width:1024,height:1024,quality:"auto",referenceEstimate:estimate},provider:"openai",channel:"openai-direct",model:"gpt-image-2.5-flare-2026-09-08",currency:"USD",usageVersion:"reference-images-totals-v1",rates:{inputText:10000,cachedInputText:0,inputImage:20000,cachedInputImage:0,outputText:0,outputImage:10000},sourceUrl:"https://developers.openai.com/api/docs/pricing",verifiedAt:"2026-11-04T00:00:00Z",effectiveFrom:"2026-11-04T00:00:00Z",effectiveUntil:"2026-11-05T00:00:00Z"}],fx:[]};
 const parsed=priceInput.parse(draft);expect(parsed.prices[0].image).toEqual(draft.prices[0].image);
 expect(canonicalPriceFacts(parsed)).toContain('"inputTokens":100');
 expect(priceInput.safeParse({...draft,prices:[{...draft.prices[0],image:{...draft.prices[0].image,referenceEstimate:{...estimate,inputTokens:0}}}]}).success).toBe(false);
});
