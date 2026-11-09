import {expect,it} from "vitest";
import {intakeResult,intakeValidationMessage} from "./own-product-intake";

it("admits only safe candidate error identifiers",()=>{
 const result={requestKey:"11111111-1111-4111-8111-111111111111",phase:"stopped",stopReason:"invalid_model_output",validationError:{field:"rows[1].stock",code:"stock_evidence"}};
 const parsed=intakeResult.parse(result);
 expect(intakeValidationMessage(parsed.validationError!)).toContain("第 2 个 SKU 的库存");
 for(const validationError of [
  {field:"rows[256].stock",code:"stock_evidence"},
  {field:"raw model content",code:"stock_evidence"},
  {field:"rows[1].stock",code:"untrusted model explanation"},
  {...result.validationError,raw:"never expose this"},
 ])expect(intakeResult.safeParse({...result,validationError}).success).toBe(false);
});

it("accepts suggestions but rejects model-authored confirmation and old direct prices",()=>{
 const row={attributes:{颜色:"黑"},evidence:"黑",priceSuggestion:{amount:0,currency:"CNY",evidence:"CNY 0"}};
 const result={requestKey:"11111111-1111-4111-8111-111111111111",phase:"human_review_required",candidate:{rows:[row]}};
 expect(intakeResult.parse(result).candidate?.rows?.[0].priceSuggestion?.amount).toBe(0);
 for(const invalid of [
  {...row,price:0,currency:"CNY"},
  {...row,priceSuggestion:{...row.priceSuggestion,confirmed:true}},
  {...row,priceSuggestion:{currency:"CNY",evidence:"CNY 0"}},
  {...row,priceSuggestion:{...row.priceSuggestion,amount:null}},
  {...row,priceSuggestion:{...row.priceSuggestion,amount:-1}},
 ])expect(intakeResult.safeParse({...result,candidate:{rows:[invalid]}}).success).toBe(false);
});

it("requires a source-bearing safe-integer stock suggestion, rejecting retired fields",()=>{
 const row={attributes:{},evidence:"库存未知",stockSuggestion:{quantity:0,evidence:"库存未知"}};
 const result={requestKey:"11111111-1111-4111-8111-111111111111",phase:"human_review_required",candidate:{rows:[row]}};
 expect(intakeResult.parse(result).candidate?.rows?.[0].stockSuggestion?.quantity).toBe(0);
 for(const stockSuggestion of [null,{evidence:"库存未知"},{quantity:null,evidence:"库存未知"},{quantity:-1,evidence:"库存未知"},{quantity:0.5,evidence:"库存未知"},{quantity:Number.MAX_SAFE_INTEGER+1,evidence:"库存未知"},{quantity:0,evidence:""},{...row.stockSuggestion,confirmed:true}])
  expect(intakeResult.safeParse({...result,candidate:{rows:[{...row,stockSuggestion}]}}).success).toBe(false);
 expect(intakeResult.safeParse({...result,candidate:{rows:[{...row,stock:0}]}}).success).toBe(false);
 expect(intakeResult.safeParse({...result,candidate:{...result.candidate,stockReviewRows:[0]}}).success).toBe(false);
});
