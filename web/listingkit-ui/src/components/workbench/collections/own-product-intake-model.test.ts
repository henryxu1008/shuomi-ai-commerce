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

it("keeps suggestions unconfirmed while adopting other valid fields",()=>{
 const suggested={rows:[{attributes:blackM,sku:"BLK-M",evidence:"black M",priceSuggestion:{amount:10,currency:"CNY",evidence:"CNY 10-20"}}]};
 const result=applyIntakeVariants(suggested,[newVariantRow()],[],false,new Set());
 expect(result.rows[0]).toMatchObject({sku:"BLK-M",currency:"",price:""});
});

it("adopts only explicitly confirmed complete prices, preserving unknown and zero",()=>{
 const suggested={rows:[
  {attributes:blackM,evidence:"black M",priceSuggestion:{amount:10,currency:"CNY",evidence:"CNY 10"}},
  {attributes:blackL,evidence:"black L"},
  {attributes:whiteM,evidence:"white M",priceSuggestion:{amount:0,currency:"CNY",evidence:"CNY 0"}},
 ]};
 const first={...newVariantRow(blackM),currency:"USD",price:"25.50"};
 const result=applyIntakeVariants(suggested,[first],[],false,new Set(),new Set([0,2]));
 expect(result.rows.map(row=>[row.currency,row.price])).toEqual([["USD","25.50"],["",""],["CNY","0"]]);
 expect(result.rows[0].id).toBe(first.id);
});

it.each([{currency:"USD",price:""},{currency:"",price:"15"}])("never combines a suggestion with one manual price field: %s",prior=>{
 const suggested={rows:[{attributes:blackM,evidence:"black M",priceSuggestion:{amount:10,currency:"CNY",evidence:"CNY 10"}}]};
 const result=applyIntakeVariants(suggested,[{...newVariantRow(blackM),...prior}],[],false,new Set(),new Set([0]));
 expect(result.rows[0]).toMatchObject(prior);
});

it("does not adopt a suggestion missing currency even with a confirmation index",()=>{
 const suggested={rows:[{attributes:blackM,evidence:"black M",priceSuggestion:{amount:10,evidence:"价格10"}}]};
 const result=applyIntakeVariants(suggested,[newVariantRow()],[],false,new Set(),new Set([0]));
 expect(result.rows[0]).toMatchObject({currency:"",price:""});
});

it("keeps removed variants removed and repeated adoption preserves existing prices",()=>{
 const suggested={rows:[
  {attributes:blackM,evidence:"black M",priceSuggestion:{amount:10,currency:"CNY",evidence:"CNY 10"}},
  {attributes:blackL,evidence:"black L",priceSuggestion:{amount:20,currency:"CNY",evidence:"CNY 20"}},
 ]};
 const removed=new Set([variantCombinationKey(blackL)]);
 const first=applyIntakeVariants(suggested,[newVariantRow()],[],false,removed,new Set([0,1]));
 const second=applyIntakeVariants(suggested,first.rows,[],false,removed,new Set([0,1]));
 expect(second.rows).toEqual(first.rows);
 expect(second.rows).toHaveLength(1);
 expect(second.rows[0]).toMatchObject({currency:"CNY",price:"10"});
});
