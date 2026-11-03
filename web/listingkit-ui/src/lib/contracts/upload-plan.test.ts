import {expect,it} from "vitest";
import {uploadPlanInputSchema,uploadPlanPreviewSchema} from "./upload-plan";
import {supplyPath,supplyUsesKey} from "./supply-chain";

const id="11111111-1111-4111-8111-111111111111";
it("admits durable plan routes and keeps preview separate from keyed mutation",()=>{
 expect(supplyPath("POST",["supply-preparations","upload-plans"])).toBe("create-upload-plan");
 expect(supplyPath("POST",["supply-preparations","upload-plans",id,"continue"])).toBe("continue-upload-plan");
 expect(supplyPath("GET",["supply-preparations","upload-plans",id,"commands",id])).toBe("upload-plan-command");
 expect(supplyUsesKey("continue-upload-plan")).toBe(true);
 expect(supplyUsesKey("upload-plan-preview")).toBe(false);
});
it("rejects invented rule capabilities and incomplete model quote bindings",()=>{
 const input={preparationId:id,expectedRevision:1,storeId:id,rule:"standard",publicationMode:"platform"};
 expect(uploadPlanInputSchema.safeParse(input).success).toBe(true);
 expect(uploadPlanInputSchema.safeParse({...input,rule:"invented"}).success).toBe(false);
 expect(uploadPlanInputSchema.safeParse({...input,titleTemplateId:id}).success).toBe(false);
 expect(uploadPlanInputSchema.safeParse({...input,titleTemplateId:id,titleTemplateRevision:"1",titleQuoteHash:"a".repeat(64),titleModeRevision:"0"}).success).toBe(true);
});
it("retains full preview counts while limiting displayed rows",()=>{
 const plan={id,input:{preparationId:id,expectedRevision:1,storeId:id,rule:"standard",publicationMode:"platform"},storeBindingHash:"a".repeat(64),count:1005,revision:1,createdAt:"2026-11-03T00:00:00Z"};
 const preview={plan,items:[],count:1005,actionable:1005,counts:{adapt:1005},hash:"b".repeat(64)};
 expect(uploadPlanPreviewSchema.safeParse(preview).success).toBe(true);
 expect(uploadPlanPreviewSchema.safeParse({...preview,counts:{adapt:1000}}).success).toBe(false);
});
