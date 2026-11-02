import {describe,it,expect} from "vitest";
import {imageAdviceSchema} from "./image-advice";

export const adviceFixture=()=>({
 schemaVersion:"image-advice-v1",
 binding:{productKey:"product",catalogVersion:"1",publicationId:"publication",targetPlatform:"shein",inventoryDigest:"a".repeat(64)},
 evidence:{toolId:"product.image.analyze",toolVersion:"v1.0.0",callId:"call",digest:"b".repeat(64)},
 facts:[{assetId:"approved",role:"main",approved:true,width:0,height:0,dimensionsStatus:"unknown"}],
 checks:[{id:"check",code:"dimensions_unknown",status:"unknown",assetIds:["approved"],ruleRef:"",reason:"真实尺寸缺少依据"}],
 suggestions:[{kind:"verify_fact",checkIds:["check"],assetIds:["approved"],reason:"请在原图片入口核对"}],
 unresolved:["真实尺寸尚未核实"]
});
describe("image advice read contract",()=>{
 it("retains unknown facts without inventing invalidity or Store requirements",()=>{
  const value=imageAdviceSchema.parse(adviceFixture());
  expect(value.checks[0].status).toBe("unknown");
  expect(value.binding.target).toBeUndefined();
 });
 it("rejects execution payloads, unsupported versions and missing references",()=>{
  expect(imageAdviceSchema.safeParse({...adviceFixture(),execute:true}).success).toBe(false);
  const a=adviceFixture();a.suggestions[0].assetIds=["unread"];
  expect(imageAdviceSchema.safeParse(a).success).toBe(false);
  expect(imageAdviceSchema.safeParse({...adviceFixture(),schemaVersion:"future"}).success).toBe(false);
 });
 it("binds native advice to its current exact version and preserves an unset saved category",()=>{
  const original=adviceFixture();
  const native={...original,binding:{...original.binding,target:{recordId:"record",revision:"1",storeId:"store",site:"shein-us",categoryId:0,ruleDigest:"a".repeat(64),sourceId:"source",originalPublicationId:"publication",originalVersion:"1",applyReceiptId:""}},evidence:{...original.evidence,toolId:"listing.materials.inspect",toolVersion:"v3.0.0"}};
  expect(imageAdviceSchema.safeParse(native).success).toBe(true);
  expect(imageAdviceSchema.safeParse({...native,evidence:{...native.evidence,toolVersion:"v2.0.0"}}).success).toBe(false);
  expect(imageAdviceSchema.safeParse({...native,evidence:original.evidence}).success).toBe(false);
 });
});
