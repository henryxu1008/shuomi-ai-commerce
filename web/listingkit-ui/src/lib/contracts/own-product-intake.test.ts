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

it("accepts only stock review rows whose values are absent",()=>{
 const result={requestKey:"11111111-1111-4111-8111-111111111111",phase:"human_review_required",candidate:{rows:[{attributes:{},evidence:"库存未知"}],stockReviewRows:[0]}};
 expect(intakeResult.safeParse(result).success).toBe(true);
 for(const stockReviewRows of [[1],[-1],[0,0],[256]])expect(intakeResult.safeParse({...result,candidate:{...result.candidate,stockReviewRows}}).success).toBe(false);
 expect(intakeResult.safeParse({...result,candidate:{...result.candidate,rows:[{...result.candidate.rows[0],stock:0}]}}).success).toBe(false);
});
