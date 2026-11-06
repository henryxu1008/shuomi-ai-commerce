import { expect,it } from "vitest";
import { officialDraftInputSchema,supplyPath,operationInputSchema,operationSchema,parseSupplyResponse,uploadAttemptSchema } from "./supply-chain";
import { initialListingDraft } from "@/components/workbench/listing/draft-fields";
import type { SupplySourceDetail } from "./supply-chain";

it("admits plan-owned operation metadata only in the response contract",()=>{
 const id="11111111-1111-4111-8111-111111111111";
 const input={preparationId:id,expectedRevision:1,storeId:id,action:"upload",sourceIds:[id]};
 const owned={planId:id,planCommandKey:id,planRequestHash:"a".repeat(64),targetApprovals:[{sourceId:id,recordId:id,revision:2}]};
 const operation={id,input:{...input,...owned},count:1,completed:0,status:"pending",execution:"started",createdAt:"2026-11-03T00:00:00Z"};
 expect(operationSchema.safeParse(operation).success).toBe(true);
 expect(operationInputSchema.safeParse(input).success).toBe(true);
 for(const [key,value] of Object.entries(owned))expect(operationInputSchema.safeParse({...input,[key]:value}).success).toBe(false);
 for(const invalid of [{planId:"not-an-id"},{planCommandKey:"not-a-key"},{planRequestHash:"not-a-hash"},{targetApprovals:[{sourceId:id,recordId:id,revision:0}]},{targetApprovals:[{sourceId:id,recordId:"other",revision:1}]}]){
  expect(operationSchema.safeParse({...operation,input:{...operation.input,...invalid}}).success).toBe(false);
 }
});
it("admits not_sent only for the original TikTok step and rejects a foreign platform",()=>{
 const id="11111111-1111-4111-8111-111111111111";
 const view={recordId:id,attemptId:id,effectKind:"image",platform:"tiktokshop",status:"not_sent",message:"本次未发送"};
 expect(uploadAttemptSchema.safeParse(view).success).toBe(true);
 for(const platform of [undefined,"temu","walmart"]){expect(uploadAttemptSchema.safeParse({...view,platform}).success).toBe(false)}
});
it("allows one approved image to be assigned to every real SKU in a large batch",()=>{
 const draft=initialListingDraft({product:{}} as SupplySourceDetail);
 draft.images=Array.from({length:205},(_,sku)=>({group:"sku" as const,skc:0,sku,asset_id:"approved-original",sort:1,type:1}));
 expect(officialDraftInputSchema.safeParse(draft).success).toBe(true);
});
it("requires the confirmed template version and quote for title optimization",()=>{
 const id="11111111-1111-4111-8111-111111111111";
 const input={preparationId:id,expectedRevision:1,titleModeRevision:"0",storeId:id,action:"optimize",titleTemplateId:id};
 expect(operationInputSchema.safeParse(input).success).toBe(false);
 expect(operationInputSchema.safeParse({...input,titleTemplateRevision:"1",titleQuoteHash:"a".repeat(64)}).success).toBe(true);
 expect(supplyPath("GET",["supply-preparations","optimization-options"])).toBe("optimization-options");
});
it("admits private retained preparation and operation history reads",()=>{
 const id="11111111-1111-4111-8111-111111111111";
 expect(supplyPath("GET",["supply-preparations",id])).toBe("preparation");expect(supplyPath("GET",["supply-preparations",id,"operations"])).toBe("operations");
});

it("binds manual TikTok status to the original source, store and product",()=>{
 const id="11111111-1111-4111-8111-111111111111",other="22222222-2222-4222-8222-222222222222",hash="a".repeat(64),time="2026-10-14T00:00:00Z";
 expect(supplyPath("GET",["supply-preparations","sources",id,"publications",other,"status"])).toBe("publication-status");
 const payload={sourceId:id,storeId:other,receiptId:id,observedAt:time,tiktok:{draftHash:hash,localSaveMode:"AS_DRAFT",local:{id:"local",skus:[{id:"sku",seller_sku:"SKU"}],responseHash:hash}},tiktokStatus:{productId:"local",status:"PENDING",auditStatus:"PRE_APPROVED",reasons:["KYC_PENDING"],observedAt:time}};
 expect(parseSupplyResponse("publication-status",payload,`${id}:${other}`)).toBeTruthy();
 expect(parseSupplyResponse("publication-status",payload,`${other}:${id}`)).toBeNull();
 expect(parseSupplyResponse("publication-status",{...payload,tiktokStatus:{...payload.tiktokStatus,productId:"wrong"}})).toBeNull();
 expect(parseSupplyResponse("publication-status",{...payload,tiktokStatus:{...payload.tiktokStatus,status:"UNKNOWN"}})).toBeNull();
});
