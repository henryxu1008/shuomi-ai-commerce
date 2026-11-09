import {describe,it,expect} from "vitest";
import {sheinSalesSelectionSchema} from "./shein-sales-mapping";
import {targetInputSchema,supplyPath} from "./supply-chain";
import {sheinImportFixture} from "@/test/shein-import-fixture";
describe("SHEIN sales mapping boundary",()=>{
 it("requires explicit zero indices and strict selections",()=>{
  const s={dimensions:[{sourceName:"颜色",attributeId:12,main:true}],values:[],rows:[{sourceIndex:0,skcIndex:0,skuIndex:0}],images:[]};
  expect(sheinSalesSelectionSchema.safeParse(s).success).toBe(true);
  expect(sheinSalesSelectionSchema.safeParse({...s,rows:[{skcIndex:0,skuIndex:0}]}).success).toBe(false);
  expect(sheinSalesSelectionSchema.safeParse({...s,returnTo:"injected"}).success).toBe(false);
 });
 it("keeps the complete one-use binding in persisted target DTOs",()=>{
  const f=sheinImportFixture(),input={...f.preview.input,sheinImport:undefined,sheinSalesMapping:{...f.preview.input.sheinImport,version:"shein-sales-mapping-v1",inventoryHash:"b".repeat(64),selection:{dimensions:[{sourceName:"颜色",attributeId:12,main:true}],values:[],rows:[{sourceIndex:0,skcIndex:0,skuIndex:0}],images:[]}}};
  expect(targetInputSchema.parse(input)).toHaveProperty("sheinSalesMapping.inventoryHash","b".repeat(64));
  expect(targetInputSchema.safeParse({...input,sheinImport:f.preview.input.sheinImport}).success).toBe(false);
  expect(supplyPath("POST",["supply-preparations","shein-sales-mapping-preview"])).toBe("shein-sales-mapping-preview");
 });
});
