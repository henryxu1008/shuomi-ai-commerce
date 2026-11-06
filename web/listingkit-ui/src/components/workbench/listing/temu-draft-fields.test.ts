import {it,expect} from "vitest";
import {initialTemuDraft,emptyTemuLocalSKU,emptyTemuSupplierSKU} from "./temu-draft-fields";
import {temuDraftInputSchema} from "@/lib/contracts/temu-model";

it("keeps merchant facts missing and decimal input exact in all modes",()=>{
 for(const mode of ["local","fully_managed","semi_managed"] as const){
  const value=initialTemuDraft({title:"Original title",description:"Original description"},mode);
  expect(temuDraftInputSchema.safeParse(value).success).toBe(true);
  if(value.local){expect(value.local.goodsBasic.goodsName).toBe("Original title");expect(value.local.skuList?.[0]?.quantity).toBeNull();expect(value.local.skuList?.[0]?.price.basePrice.amount).toBe("")}
  if(value.supplier){expect(value.supplier.productName).toBe("Original title");expect(value.supplier.productSkcReqs?.[0]?.productSkuReqs?.[0]?.currencyType).toBe("");expect(value.supplier.productSkcReqs?.[0]?.productSkuReqs?.[0]?.supplierPrice).toBeUndefined()}
 }
 const sku=emptyTemuLocalSKU();sku.quantity="0";sku.price.basePrice.amount="0.020000000000000001";expect(sku.quantity).toBe("0");expect(sku.price.basePrice.amount).toBe("0.020000000000000001");
 expect(emptyTemuSupplierSKU().productSkuWhExtAttrReq).toBeNull();
});
