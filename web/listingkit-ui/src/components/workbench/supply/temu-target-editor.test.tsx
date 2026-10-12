import {cleanup,render,screen,waitFor} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {TemuTargetEditor} from "./temu-target-editor";
import type {SupplySourceDetail} from "@/lib/contracts/supply-chain";
import type {SupplyCommandState} from "./use-supply-command";
const state=vi.hoisted(()=>({rules:vi.fn(),inventory:vi.fn(),execute:vi.fn(),completion:vi.fn()}));
vi.mock("@/lib/api/supply-chain",async original=>({...await original<object>(),temuSupplyRules:state.rules,supplyInventory:state.inventory,readSupplyCompletionDraft:state.completion}));
const id="11111111-1111-4111-8111-111111111111";
const source:SupplySourceDetail={source:{id,preparationId:id,collectionItemId:id,collectionRevision:1,source:{productKey:"product-a",publicationId:id,version:"9007199254740993",kind:"own"}},product:{title:"原始商品",description:"原始描述",variants:[{sku:"source-sku",price:{amount:99},stock:42}]},images:[{id,url:"https://example.org/main.jpg",width:900,height:900}]};
const command={execute:state.execute} as unknown as SupplyCommandState;
beforeEach(()=>{state.rules.mockReset();state.execute.mockReset();state.inventory.mockResolvedValue({scope:{tenant_id:"org",product_key:"product-a",target_platform:"temu",source_snapshot_version:"9007199254740993"},assets:[]});state.completion.mockReset()});
afterEach(cleanup);

it("edits each composition value separately with exact selected IDs",async()=>{
 const current=rules("local");state.rules.mockResolvedValue({...current,temu:{...current.temu,localAttributes:{attributeList:[{refPid:"9007199254740993",attributeName:"组成",required:true,attributeRules:{controlType:"16",chooseMaxNum:"2",transnationalAttribute:false,attributeValueRules:{valueRule:"1",valuePrecision:"2"}},attributeValueDetail:[{groupId:"main",attributeValueList:[{vid:"9007199254740994",value:"棉",childAttribute:[]},{vid:"9007199254740995",value:"聚酯",childAttribute:[]}]}],attributeValueUnitList:[]}]}}});
 render(<TemuTargetEditor scope={{userId:"actor",organizationId:"org"}} source={source} storeId={id} command={command} disabled={false} saved={0}/>);await screen.findByLabelText("商品标题");
 await userEvent.click(screen.getByRole("button",{name:"添加 组成 属性值"}));await userEvent.selectOptions(screen.getByLabelText("组成 值 1"),"9007199254740994");await userEvent.type(screen.getByLabelText("组成 数值 1"),"60");
 await userEvent.click(screen.getByRole("button",{name:"添加 组成 属性值"}));await userEvent.selectOptions(screen.getByLabelText("组成 值 2"),"9007199254740995");await userEvent.type(screen.getByLabelText("组成 数值 2"),"40");
 await userEvent.click(screen.getByRole("button",{name:"保存并校验平台资料"}));expect(state.execute.mock.lastCall?.[1].temu.local.goodsProperty).toEqual([{refPid:"9007199254740993",vid:"9007199254740994",numberInputValue:"60"},{refPid:"9007199254740993",vid:"9007199254740995",numberInputValue:"40"}]);
});
function rules(mode:string){return {merchant:{merchant_mode:mode},temu:{mode,selection:{parentCatId:"0",catId:"0"},localCategories:[],supplierCategories:[],freightTemplates:[],issues:[],...(mode==="local"?{localTarget:{region:{regionId:"75",siteId:"165",label:"Georgia"},issues:["site_specification_unverified"]}}:{semiSites:[]})}}}
it("saves incomplete native facts without inventing price, inventory or site specs",async()=>{
 state.rules.mockResolvedValue(rules("local"));render(<TemuTargetEditor scope={{userId:"actor",organizationId:"org"}} source={source} storeId={id} command={command} disabled={false} saved={0}/>);
 await screen.findByLabelText("商品标题");expect(screen.getByLabelText("SKU 1 库存数量")).toHaveValue("");expect(screen.getByLabelText("SKU 1 基础价格")).toHaveValue("");expect(screen.getByText(/站点规格尚未核实/)).toBeInTheDocument();
 await userEvent.type(screen.getByLabelText("SKU 1 库存数量"),"0");await userEvent.type(screen.getByLabelText("SKU 1 基础价格"),"10.2300");
 await userEvent.click(screen.getByRole("button",{name:"保存并校验平台资料"}));await waitFor(()=>expect(state.execute).toHaveBeenCalled());
 const value=state.execute.mock.lastCall?.[1];expect(value).toMatchObject({effectiveVersion:"9007199254740993",temu:{mode:"local",local:{goodsBasic:{goodsName:"原始商品"},skuList:[{quantity:"0",price:{basePrice:{amount:"10.2300",currency:""}}}]}}});expect(value).not.toHaveProperty("draft");expect(value.temu.packageUnits).toEqual({weight:"",dimension:""});
});
for(const mode of ["fully_managed","semi_managed"] as const)it(`uses verified ${mode} store mode and keeps supplier pricing distinct`,async()=>{
 state.rules.mockResolvedValue(rules(mode));render(<TemuTargetEditor scope={{userId:"actor",organizationId:"org"}} source={source} storeId={id} command={command} disabled={false} saved={0}/>);await screen.findByLabelText("商品标题");
 expect(screen.queryByLabelText("店铺模式")).not.toBeInTheDocument();expect(screen.queryByLabelText("SKU 1 基础价格")).not.toBeInTheDocument();expect(screen.getByText(/全托管与半托管/)).toBeInTheDocument();
 await userEvent.click(screen.getByRole("button",{name:"保存并校验平台资料"}));expect(state.execute.mock.lastCall?.[1].temu).toMatchObject({mode,supplier:{cat1Id:null,productSkcReqs:[{productSkuReqs:[{currencyType:""}]}]}});
});

it("keeps semi-managed site prices, explicit zero warehouse stock and packaging units separate",async()=>{
 const current=rules("semi_managed");state.rules.mockResolvedValue({...current,temu:{...current.temu,semiSites:[{siteId:"100",label:"US",opening:"open"},{siteId:"101",label:"CA",opening:"open"}]}});
 render(<TemuTargetEditor scope={{userId:"actor",organizationId:"org"}} source={source} storeId={id} command={command} disabled={false} saved={0}/>);await screen.findByLabelText("商品标题");
 await userEvent.click(screen.getByLabelText("销售站点 US"));await userEvent.click(screen.getByLabelText("销售站点 CA"));
 await userEvent.type(screen.getByLabelText("SKC 1 SKU 1 站点 US 供货价（整数分）"),"1234");await userEvent.type(screen.getByLabelText("SKC 1 SKU 1 站点 CA 供货价（整数分）"),"2345");
 expect(screen.getByLabelText("SKC 1 SKU 1 包装重量（mg）")).toHaveValue("");
 await userEvent.type(screen.getByLabelText("SKC 1 SKU 1 包装重量（mg）"),"150000");
 await userEvent.click(screen.getByRole("button",{name:"SKC 1 SKU 1 添加仓库库存"}));await userEvent.type(screen.getByLabelText("SKC 1 SKU 1 仓库 1 编号"),"warehouse-a");await userEvent.type(screen.getByLabelText("SKC 1 SKU 1 仓库 1 初始数量"),"0");
 await userEvent.click(screen.getByRole("button",{name:"保存并校验平台资料"}));
 const supplier=state.execute.mock.lastCall?.[1].temu.supplier;
 expect(supplier.productSemiManagedReq.bindSiteIds).toEqual(["100","101"]);
 expect(supplier.productSkcReqs[0].productSkuReqs[0]).toMatchObject({siteSupplierPrices:[{siteId:"100",supplierPrice:"1234"},{siteId:"101",supplierPrice:"2345"}],productSkuWhExtAttrReq:{productSkuWeightReq:{value:"150000"}},productSkuStockQuantityReq:{warehouseStockQuantityReqs:[{warehouseId:"warehouse-a",targetStockAvailable:"0"}]}});
 expect(supplier.productSkcReqs[0].productSkuReqs[0]).not.toHaveProperty("supplierPrice");expect(supplier).not.toHaveProperty("productComplianceStatementReq");
});
it("approves only explicitly selected source images under original TEMU provenance",async()=>{
 state.rules.mockResolvedValue(rules("local"));render(<TemuTargetEditor scope={{userId:"actor",organizationId:"org"}} source={source} storeId={id} command={command} disabled={false} saved={0}/>);await screen.findByLabelText("商品标题");
 await userEvent.click(screen.getByLabelText("使用来源图 1"));await userEvent.click(screen.getByRole("button",{name:"确认完整图片选择"}));expect(state.execute).toHaveBeenCalledWith("approve",{selection:{itemId:id,originalPublicationId:id,originalSnapshotVersion:"9007199254740993",effectiveCatalogVersion:"9007199254740993",targetPlatform:"temu"},images:[{id,role:"gallery"}],approved:[]});
});
it("imports completion text only after an explicit action",async()=>{
 state.rules.mockResolvedValue(rules("local"));state.completion.mockResolvedValue({revision:3,content:{title:"候选标题",description:"候选描述"}});render(<TemuTargetEditor scope={{userId:"actor",organizationId:"org"}} source={source} storeId={id} command={command} disabled={false} saved={0} completionAvailable/>);const bring=await screen.findByRole("button",{name:"带入待补全草稿 R3"});await screen.findByLabelText("商品标题");expect(screen.getByLabelText("商品标题")).toHaveValue("原始商品");await userEvent.click(bring);expect(screen.getByLabelText("商品标题")).toHaveValue("候选标题");
});
it("uses exact predefined variation IDs when the current template disallows custom values",async()=>{
 const current={...rules("local"),temu:{...rules("local").temu,localVariations:{catId:"34",language:"en",variations:[{parentSpecId:"9007199254740994",parentSpecName:"Color",required:true,variationType:"0",specList:[{specId:"9007199254740993",specName:"Blue",parentSpecId:"9007199254740994",parentSpecName:"Color",subGroupId:null,subGroupName:""}]}]}}};state.rules.mockResolvedValue(current);
 render(<TemuTargetEditor scope={{userId:"actor",organizationId:"org"}} source={source} storeId={id} command={command} disabled={false} saved={0}/>);await screen.findByLabelText("商品标题");await userEvent.click(screen.getByRole("button",{name:"添加规格"}));await userEvent.selectOptions(screen.getByLabelText("规格 1"),"9007199254740994");await userEvent.selectOptions(screen.getByLabelText("规格 1 值"),"9007199254740993");await userEvent.click(screen.getByRole("button",{name:"保存并校验平台资料"}));expect(state.execute.mock.lastCall?.[1].temu.local.skuList[0].specDetails).toEqual([{parentSpecId:"9007199254740994",specId:"9007199254740993",specName:"Blue"}]);
});
