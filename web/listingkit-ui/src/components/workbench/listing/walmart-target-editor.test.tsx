import {cleanup,render,screen} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {afterEach,beforeEach,it,expect,vi} from "vitest";
import {WalmartTargetEditor} from "./walmart-target-editor";
import type {SupplySourceDetail} from "@/lib/contracts/supply-chain";
import type {ListingCommandState} from "./use-listing-command";
const state=vi.hoisted(()=>({rules:vi.fn(),inventory:vi.fn(),completion:vi.fn(),execute:vi.fn()}));
vi.mock("@/lib/api/supply-chain",async original=>({...await original<object>(),walmartSupplyRules:state.rules,supplyInventory:state.inventory,readSupplyCompletionDraft:state.completion}));
const id="11111111-1111-4111-8111-111111111111";
const source:SupplySourceDetail={source:{id,preparationId:id,collectionItemId:id,collectionRevision:1,source:{productKey:"product",publicationId:id,version:"1",kind:"own"}},product:{title:"Source title",variants:[{source_id:"source-sku",sku:"sku",barcode:"12345678901234",stock:42,price:{amount:99}}]},images:[]};
beforeEach(()=>{state.execute.mockReset();state.rules.mockResolvedValue({walmartCatalog:{feedType:"MP_ITEM_MATCH",version:"5-official",productTypes:[]}});state.inventory.mockResolvedValue({assets:[]})});afterEach(cleanup);
it("saves a local match draft with explicit price and source identity, without publishing",async()=>{
 render(<WalmartTargetEditor scope={{userId:"actor",organizationId:"org"}} source={source} storeId={id} command={{execute:state.execute} as unknown as ListingCommandState} disabled={false} saved={0}/>);
 expect(screen.getByLabelText("首次报价（USD）")).toHaveValue("");expect(screen.getByLabelText("商品成色")).toHaveValue("");
 await userEvent.click(screen.getByRole("button",{name:"读取当前可用类目"}));await screen.findByText("规则版本：5-official");await userEvent.type(screen.getByLabelText("商品成色"),"New");await userEvent.type(screen.getByLabelText("首次报价（USD）"),"12.50");await userEvent.type(screen.getByLabelText("运输重量（按规格单位）"),"1");
 await userEvent.click(screen.getByRole("button",{name:"校验并保存 Walmart 草稿"}));expect(state.execute).toHaveBeenCalledOnce();expect(state.execute.mock.lastCall).toEqual(["save-target",{sourceId:id,storeId:id,expectedRevision:0,effectiveVersion:"1",walmart:{selection:{feedType:"MP_ITEM_MATCH",version:"5-official",productType:""},skus:[{sourceSKUKey:"source-sku",identifier:{productIdType:"GTIN",productId:"12345678901234"},condition:"New",price:"12.50",shippingWeight:"1"}]}}]);
});


it("carries only saved candidate text into a Walmart submission draft",async()=>{
 state.rules.mockResolvedValue({walmartCatalog:{feedType:"MP_ITEM",version:"5-official",productTypes:["Cable"]}});
 state.completion.mockResolvedValue({id,revision:3,content:{title:"候选标题",description:"候选描述",internalNote:"WORKBENCH_ONLY_WALMART_NOTE"}});
 render(<WalmartTargetEditor scope={{userId:"actor",organizationId:"org"}} source={source} storeId={id} command={{execute:state.execute} as unknown as ListingCommandState} disabled={false} saved={0} completionAvailable/>);
 await userEvent.selectOptions(screen.getByLabelText("刊登方式"),"MP_ITEM");
 await userEvent.click(screen.getByRole("button",{name:"读取当前可用类目"}));
 await userEvent.selectOptions(await screen.findByRole("combobox",{name:"商品类型"}),"Cable");
 await userEvent.click(await screen.findByRole("button",{name:"采用已保存标题和描述"}));
 await userEvent.type(screen.getByLabelText("商品成色"),"New");
 await userEvent.type(screen.getByLabelText("首次报价（USD）"),"12.50");
 await userEvent.type(screen.getByLabelText("运输重量（按规格单位）"),"1");
 await userEvent.click(screen.getByRole("button",{name:"校验并保存 Walmart 草稿"}));
 const payload=state.execute.mock.lastCall?.[1];
 expect(payload).toMatchObject({completionDraft:{recordId:id,revision:3},walmart:{skus:[{visible:{productName:"候选标题",shortDescription:"候选描述"}}]}});
 expect(JSON.stringify(payload)).not.toMatch(/internalNote|WORKBENCH_ONLY_WALMART_NOTE/);
});
