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
