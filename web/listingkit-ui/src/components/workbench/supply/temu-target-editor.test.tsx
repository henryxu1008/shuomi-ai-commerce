import {cleanup,render,screen,waitFor} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {TemuTargetEditor} from "./temu-target-editor";
import type {SupplySourceDetail,TemuSupplyTarget} from "@/lib/contracts/supply-chain";
import {initialTemuDraft} from "./temu-draft-fields";
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
for(const mode of ["fully_managed","semi_managed"] as const)it(`explicitly prepares the currently verified ${mode} draft after a supplier mode change`,async()=>{
 const prior=mode==="fully_managed"?"semi_managed":"fully_managed",draft=initialTemuDraft({title:"手填标题"},prior);
 draft.selection={parentCatId:"0",catId:"0",supplierSiteId:"100"};
 draft.supplier!.productSemiManagedReq={bindSiteIds:["100"]};
 draft.supplier!.productShipmentReq={freightTemplateId:"freight-a",shipmentLimitSecond:"86400"};
 const sku=draft.supplier!.productSkcReqs![0]!.productSkuReqs![0]!;
 sku.extCode="manual-sku";sku.currencyType="CNY";sku.supplierPrice="1234";sku.siteSupplierPrices=[{siteId:"100",supplierPrice:"1234"}];sku.productSkuStockQuantityReq={warehouseStockQuantityReqs:[{warehouseId:"manual-warehouse",targetStockAvailable:"3"}]};
 const existing={input:{temu:draft},temu:{mode:prior,issues:[]},revision:1,effectiveVersion:source.source.source.version} as unknown as TemuSupplyTarget;
 state.rules.mockImplementation(async(_scope,_input)=>{if(_input.temuRules.supplierSiteId==="100")throw new Error("old site selection is not valid for current mode");return rules(mode)});
 render(<TemuTargetEditor scope={{userId:"actor",organizationId:"org"}} source={source} storeId={id} existing={existing} command={command} disabled={false} saved={0}/>);
 const reset=await screen.findByRole("button",{name:"按当前模式重新准备"});expect(state.execute).not.toHaveBeenCalled();
 await userEvent.click(reset);await screen.findByText(`TEMU · ${mode==="fully_managed"?"全托管":"半托管"}`);
 await waitFor(()=>expect(screen.getByRole("button",{name:"保存并校验平台资料"})).toBeEnabled());
 await userEvent.click(screen.getByRole("button",{name:"保存并校验平台资料"}));
 const saved=state.execute.mock.lastCall?.[1];expect(saved).toMatchObject({expectedRevision:1,effectiveVersion:source.source.source.version,temu:{mode,supplier:{productName:"手填标题",productSkcReqs:[{productSkuReqs:[{extCode:"manual-sku",currencyType:""}]}]}}});
 const supplier=saved.temu.supplier,clean=supplier.productSkcReqs[0].productSkuReqs[0];
 expect(supplier).not.toHaveProperty("productSemiManagedReq");expect(supplier).not.toHaveProperty("productShipmentReq");
 expect(clean).not.toHaveProperty("supplierPrice");expect(clean).not.toHaveProperty("siteSupplierPrices");expect(clean).not.toHaveProperty("productSkuStockQuantityReq");
});
it("returns a saved local category to the root without guessing its ancestor chain",async()=>{
 const draft=initialTemuDraft({title:"手填本土标题"},"local");draft.selection={parentCatId:"12",catId:"34"};draft.local!.goodsBasic.catId="34";
 const existing={input:{temu:draft},temu:{mode:"local",issues:[]},revision:2,effectiveVersion:source.source.source.version} as unknown as TemuSupplyTarget;
 state.rules.mockResolvedValue(rules("local"));render(<TemuTargetEditor scope={{userId:"actor",organizationId:"org"}} source={source} storeId={id} existing={existing} command={command} disabled={false} saved={0}/>);
 await screen.findByLabelText("商品标题");await waitFor(()=>expect(screen.getByRole("button",{name:"返回顶级类目"})).toBeEnabled());
 await userEvent.click(screen.getByRole("button",{name:"返回顶级类目"}));
 await waitFor(()=>expect(state.rules.mock.lastCall?.[1].temuRules).toMatchObject({parentCatId:"0",catId:"0"}));
 expect(screen.getByLabelText("商品标题")).toHaveValue("手填本土标题");expect(state.execute).not.toHaveBeenCalled();
});
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

it("derives exact supplier property IDs only from explicitly chosen preset specs",async()=>{
 const current=rules("fully_managed");state.rules.mockResolvedValue({...current,temu:{...current.temu,supplierAttributes:{properties:[{refPid:"9007199254740993",pid:"11",templatePid:"21",parentSpecId:"9007199254740994",name:"颜色",required:true,isSale:true,mainSale:true,controlType:"9",values:[{vid:"41",specId:"9007199254740995",value:"蓝色"}],group:{id:"9007199254740996",name:"官方组"}}]}}});
 render(<TemuTargetEditor scope={{organizationId:"org",userId:"actor"}} source={source} storeId={id} command={command} disabled={false} saved={0}/>);
 await screen.findByText("TEMU · 全托管");
 await userEvent.selectOptions(screen.getByRole("combobox",{name:"SKC 1 主规格 颜色"}),"9007199254740995");
 await userEvent.selectOptions(screen.getByRole("combobox",{name:"SKC 1 SKU 1 规格 颜色"}),"9007199254740995");
 await userEvent.click(screen.getByRole("button",{name:"保存并校验平台资料"}));
 expect(state.execute.mock.lastCall?.[1].temu.supplier.productSpecPropertyReqs).toEqual([{pid:"11",templatePid:"21",refPid:"9007199254740993",propName:"颜色",vid:"41",propValue:"蓝色",valueUnit:"",specId:"9007199254740995",parentSpecId:"9007199254740994",valueGroupId:"9007199254740996",valueGroupName:"官方组"}]);
});
it("saves the full supplier category path and explicit zero unused levels",async()=>{
 state.rules.mockImplementation((_scope,request)=>{const current=rules("fully_managed");return Promise.resolve({...current,temu:{...current.temu,supplierCategories:request.temuRules.parentCatId==="0"?[{catId:"12",catName:"一级",isLeaf:false}]:[{catId:"34",catName:"末级",isLeaf:true}]}})});
 render(<TemuTargetEditor scope={{organizationId:"org",userId:"actor"}} source={source} storeId={id} command={command} disabled={false} saved={0}/>);
 await screen.findByText("TEMU · 全托管");
 await userEvent.selectOptions(screen.getByRole("combobox",{name:"官方类目"}),"12");
 await screen.findByRole("option",{name:"末级"});
 await waitFor(()=>expect(screen.getByRole("combobox",{name:"官方类目"})).not.toBeDisabled());
 await userEvent.selectOptions(screen.getByRole("combobox",{name:"官方类目"}),"34");
 await userEvent.click(screen.getByRole("button",{name:"保存并校验平台资料"}));
 const saved=state.execute.mock.lastCall?.[1].temu;
 expect(saved.selection.supplierCategoryPath).toEqual(["12","34","0","0","0","0","0","0","0","0"]);
 expect(saved.supplier).toMatchObject({cat1Id:"12",cat2Id:"34",cat3Id:"0",cat10Id:"0"});
});

it("selects existing size charts across pages and saves bounded reread hints",async()=>{
 const chart=(businessId:string,name:string)=>({businessId,name,reusable:false});
 state.rules.mockImplementation((_scope,request)=>{const current=rules("fully_managed"),offset=request.temuRules.supplierSizeOffset??"0";return Promise.resolve({...current,temu:{...current.temu,supplierSizeRequirement:{catId:"34",observed:true,class:{classId:"7"}},supplierSizeCharts:{offset,pageSize:"50",totalCount:"100",sizeSpecDataList:offset==="0"?[chart("100","表A")]:[chart("200","表B")]},supplierSizeChartPages:offset==="0"&&request.temuRules.supplierSizeOffsets?.includes("50")?[{offset:"50",pageSize:"50",totalCount:"100",sizeSpecDataList:[chart("200","表B")]}]:offset==="50"?[{offset:"0",pageSize:"50",totalCount:"100",sizeSpecDataList:[chart("100","表A")]}]:[]}})});
 render(<TemuTargetEditor scope={{organizationId:"org",userId:"actor"}} source={source} storeId={id} command={command} disabled={false} saved={0}/>);
 await userEvent.click(await screen.findByRole("checkbox",{name:"表A"}));
 await waitFor(()=>expect(screen.getByRole("button",{name:"尺码表下一页"})).not.toBeDisabled());
 await userEvent.click(screen.getByRole("button",{name:"尺码表下一页"}));
 await userEvent.click(await screen.findByRole("checkbox",{name:"表B"}));
 await waitFor(()=>expect(state.rules.mock.lastCall?.[1].temuRules.supplierSizeOffsets).toContain("50"));
 await waitFor(()=>expect(screen.getByRole("button",{name:"尺码表上一页"})).not.toBeDisabled());
 await userEvent.click(screen.getByRole("button",{name:"尺码表上一页"}));
 expect(await screen.findByRole("checkbox",{name:"表A"})).toBeChecked();
 await userEvent.click(screen.getByRole("button",{name:"保存并校验平台资料"}));
 expect(state.execute.mock.lastCall?.[1].temu.supplier.sizeTemplateIds).toEqual(["100","200"]);
 expect(state.execute.mock.lastCall?.[1].temu.selection.supplierSizeOffsets).toEqual(["50",...Array(15).fill("0")]);
 state.rules.mockResolvedValue(rules("fully_managed"));
 await userEvent.click(screen.getByRole("button",{name:"刷新官方规则"}));
 await screen.findByText("尺码表要求尚未核实，商品可保存。");
 await userEvent.click(screen.getByRole("button",{name:"移除尺码表 100"}));
 await waitFor(()=>expect(screen.getByRole("button",{name:"移除尺码表 200"})).not.toBeDisabled());
 await userEvent.click(screen.getByRole("button",{name:"移除尺码表 200"}));
 await userEvent.click(screen.getByRole("button",{name:"保存并校验平台资料"}));
 expect(state.execute.mock.lastCall?.[1].temu.supplier.sizeTemplateIds).toEqual([]);
 expect(state.execute.mock.lastCall?.[1].temu.selection.supplierSizeOffsets).toEqual(Array(16).fill("0"));
});
