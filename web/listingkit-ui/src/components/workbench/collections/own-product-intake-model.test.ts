import {expect,it} from "vitest";
import {newVariantRow,variantCombinationKey} from "./own-product-model";
import {applyIntakeVariants} from "./own-product-intake-model";

const blackM={颜色:"黑",尺码:"M"},blackL={颜色:"黑",尺码:"L"},whiteM={颜色:"白",尺码:"M"};
const candidate={rows:[
 {attributes:blackM,sku:"BLK-M",evidence:"黑 M",stock:0},
 {attributes:blackL,sku:"BLK-L",evidence:"黑 L"},
 {attributes:whiteM,sku:"WHT-M",evidence:"白 M",image:2},
]};

it("uses only explicit combinations and keeps omitted stock blank",()=>{
 const result=applyIntakeVariants(candidate,[newVariantRow()],["common","white"],true,new Set());
 expect(result.rows.map(row=>[row.sku,row.stock,row.image])).toEqual([["BLK-M","0",""],["BLK-L","",""],["WHT-M","","white"]]);
 expect([...result.assigned]).toEqual(["white"]);
});

it("preserves manual values and respects a previously removed row",()=>{
 const first={...newVariantRow(blackM),sku:"MANUAL-M",stock:"8",image:"common"};
 const result=applyIntakeVariants(candidate,[first],["common","white"],false,new Set([variantCombinationKey(blackL)]));
 expect(result.rows.map(row=>[row.sku,row.stock,row.image])).toEqual([["MANUAL-M","8","common"],["WHT-M","",""]]);
 expect(result.rows[0].id).toBe(first.id);
 expect(result.assigned.size).toBe(0);
});
