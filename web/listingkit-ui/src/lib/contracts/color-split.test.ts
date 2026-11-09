import {describe,it,expect} from "vitest";
import {colorSplitRequestSchema,supplyPath,supplyUsesKey,supplyMutates} from "./supply-chain";
import {parseSupplyIntent} from "../api/supply-intent";
const id="00000000-0000-4000-8000-000000000001";
describe("color split contract",()=>{
 it("binds the real routes and a single durable creation key",()=>{
  expect(supplyPath("POST",["supply-preparations","shein-color-split-preview"])).toBe("color-split-preview");
  expect(supplyPath("POST",["supply-preparations","shein-color-splits"])).toBe("create-color-split");
  expect(supplyPath("GET",["supply-preparations","shein-color-split-commands",id])).toBe("color-split-command");
  expect(supplyUsesKey("create-color-split")).toBe(true);expect(supplyMutates("color-split-preview")).toBe(false);
 });
 it("never interprets omitted source or target row positions as zero",()=>{
  const r={sourceId:id,baseRecordId:id,completionDraft:{recordId:id,revision:1},selection:{colorAttributeId:12,skus:[{skcIndex:0,skuIndex:0}],sharedImages:[]}};
  expect(colorSplitRequestSchema.safeParse(r).success).toBe(false);
  expect(parseSupplyIntent(JSON.stringify({userId:"u",organizationId:"o",route:"create-color-split",key:id,command:r}))).toBeNull();
 });
});
