import { describe,it,expect } from "vitest";
import {temuDraftInputSchema,temuIntegerSchema,ruleSelectionSchema} from "./temu-model";

describe("TEMU typed browser facts",()=>{
 it("preserves LONG IDs and incomplete facts without numeric defaults",()=>{
  const input={mode:"local",selection:{parentCatId:"0",catId:"9007199254740993"},local:{goodsBasic:{goodsName:"",catId:"9007199254740993"},goodsServicePromise:{shipmentLimitDay:null,costTemplateId:""},skuList:[]},packageUnits:{weight:"",dimension:""},images:[]};
  const parsed=temuDraftInputSchema.parse(input);
  expect(parsed.selection.catId).toBe("9007199254740993");expect(parsed.local?.goodsServicePromise.shipmentLimitDay).toBeNull();expect(parsed.local?.skuList).toEqual([]);
  expect(ruleSelectionSchema.safeParse({parentCatId:"0",catId:9007199254740993}).success).toBe(false);
  expect(temuDraftInputSchema.safeParse({...input,supplier:{}}).success).toBe(false);
 });
 it("rejects rounded, overflow and noncanonical integer encodings",()=>{
  for(const value of [9007199254740993,"01","+1","1.0","1e3","9223372036854775808","-9223372036854775809"])expect(temuIntegerSchema.safeParse(value).success).toBe(false);
  expect(temuIntegerSchema.parse("0")).toBe("0");expect(temuIntegerSchema.parse("9223372036854775807")).toBe("9223372036854775807");
 });
});
