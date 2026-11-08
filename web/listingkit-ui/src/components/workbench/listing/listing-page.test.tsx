import { act, cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { ListingPage } from "./listing-page";
import ListingEntry from "@/app/workbench/listing/page";
import {SupplyAPIError} from "@/lib/api/supply-chain";
import {sheinImportFixture} from "@/test/shein-import-fixture";
import ListingStage from "@/app/workbench/listing/[stage]/page";
vi.mock("next/server",()=>({connection:vi.fn()}));
const state=vi.hoisted(()=>({context:{} as Record<string,unknown>,preparation:vi.fn(),preparations:vi.fn(),stages:vi.fn(),sources:vi.fn(),history:vi.fn(),stores:vi.fn(),options:vi.fn(),write:vi.fn(),detail:vi.fn(),target:vi.fn(),publication:vi.fn(),operation:vi.fn(),items:vi.fn(),attempt:vi.fn(),record:vi.fn(),plans:vi.fn(),plan:vi.fn(),preview:vi.fn(),rules:vi.fn(),completion:vi.fn()}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>state.context}));
vi.mock("@/lib/api/supply-chain",async original=>({...await original<object>(),readSupplyPreparation:state.preparation,listSupplyPreparations:state.preparations,listSupplyStages:state.stages,listSupplySources:state.sources,listSupplyOperations:state.history,supplyCommand:state.write,readSupplyOptimizationOptions:state.options,readSupplySource:state.detail,readSupplyTarget:state.target,readSupplySavedTarget:state.target,readSupplyPublication:state.publication,readSupplyOperation:state.operation,listSupplyOperationItems:state.items,readSupplyUploadAttempt:state.attempt,readSupplyRecord:state.record,listUploadPlans:state.plans,readUploadPlan:state.plan,previewUploadPlan:state.preview,supplyRules:state.rules,readSupplyCompletionDraft:state.completion,supplyInventory:vi.fn().mockResolvedValue({assets:[]})}));
vi.mock("./temu-target-editor",()=>({TemuTargetEditor:({existing}:{existing?:{temu:{mode:string}}})=><p>TEMU 原生资料 {existing?.temu.mode}</p>}));
vi.mock("./bulk-review-panel",()=>({BulkListingReviewPanel:({items}:{items:unknown[]})=><p>实际批量提案 {items.length} 件</p>}));
vi.mock("@/lib/api/workbench-stores",async original=>({...await original<typeof import("@/lib/api/workbench-stores")>(),listWorkbenchStores:state.stores}));
const prep="11111111-1111-4111-8111-111111111111",store="22222222-2222-4222-8222-222222222222",source="33333333-3333-4333-8333-333333333333";
beforeEach(()=>{
 for(const name of ["LISTINGKIT_SUPPLY_CHAIN_ENABLED","LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED","LISTINGKIT_SUPPLY_COMPLETION_DRAFTS_ENABLED"])vi.stubEnv(name,"true");
 state.context={user:{id:"actor-a"},effectiveOrganization:{id:"org-a"},permissions:["workbench.supply.read","workbench.supply.manage","workbench.listing.submit","workbench.store.read"],supplyIntentReady:true,pendingSupplyIntent:null,pendingCollectionIntent:null,setPendingSupplyIntent:()=>true,registerOrganizationSwitchGuard:()=>()=>{}};
 state.plans.mockReset().mockResolvedValue({items:[]});state.plan.mockReset();state.preview.mockReset();
 state.rules.mockReset();state.completion.mockReset();
 state.options.mockReset().mockResolvedValue({titles:[],reason:"未启用"});
 state.stages.mockReset().mockResolvedValue({items:[],total:205,batchTotal:205,counts:{all:205,waiting:205,missing:0,ready:0,review:0,uploaded:0},nextCursor:source});
 state.preparations.mockReset().mockResolvedValue({items:[{id:prep,name:"完整批次",count:205,createdAt:"2026-10-08T00:00:00Z"}],total:1});
 state.preparation.mockReset().mockResolvedValue({id:prep,sourceBatchId:prep,sourceRevision:1,name:"完整批次",count:205,revision:1,createdAt:"2026-10-08T00:00:00Z"});
 state.sources.mockReset().mockResolvedValue({items:[],total:205,nextCursor:source});state.history.mockReset().mockResolvedValue({items:[],total:0});state.write.mockReset().mockResolvedValue({operation:{id:source,input:{preparationId:prep,expectedRevision:1,storeId:store,action:"adapt"},count:205,completed:0,status:"pending",execution:"started",createdAt:"2026-10-08T00:00:00Z"},replayed:false});
 state.stores.mockReset().mockResolvedValue({items:[{id:store,name:"美国试用店铺",platform:"shein",recordStatus:"active",connectionStatus:"connected",serviceStatus:"active"}],pagination:{page:1,pageSize:100,total:1}});
});afterEach(cleanup);
it("reads the exact historical record without loading a head, rules or execution and links to explicit report preview", async () => {
 const id="55555555-5555-4555-8555-555555555555";
 state.context.permissions=["workbench.supply.read","workbench.report.read","workbench.report.manage"];
 state.record.mockReset().mockResolvedValue({id,revision:2,source:{source:{productKey:"product-a"}},merchant:{store_id:store,site:"shein-us"},input:{draft:{product:{names:[],multi_language_name_list:[{language:"en",name:"历史标题"}],multi_language_desc_list:[]}}},result:{issues:[{field:"images",code:"missing",message:"历史缺图"}],ready_for_upload:false},createdAt:"2026-11-01T00:00:00Z"});
 const view=render(<ListingPage initialRecord={id} reportSourcesAvailable/>);
 await screen.findByText("历史标题");
 expect(state.record).toHaveBeenCalledWith({userId:"actor-a",organizationId:"org-a"},id,expect.any(AbortSignal));
 expect(state.target).not.toHaveBeenCalled();expect(state.options).not.toHaveBeenCalled();expect(state.stores).not.toHaveBeenCalled();expect(state.write).not.toHaveBeenCalled();
 expect(screen.getByRole("link",{name:"保存报告"})).toHaveAttribute("href",`/workbench/ai/reports?sourceKind=SHEIN_TARGET_RECORD&sourceId=${id}`);
 state.context.permissions=["workbench.supply.read"];
 view.rerender(<ListingPage initialRecord={id} reportSourcesAvailable/>);
 expect(screen.queryByRole("link",{name:"保存报告"})).not.toBeInTheDocument();
});
it.each([["missing","待补全"],["uploaded","已上传"]] as const)("keeps platform context through the actual %s route",async(stage,label)=>{
 render(<ListingPage initialPreparation={prep} initialStore={store} initialStage="review"/>);
 const link=await screen.findByRole("link",{name:new RegExp(label)});
 const url=new URL(link.getAttribute("href")!,"https://test.invalid");
 expect(url.pathname).toBe(`/workbench/listing/${stage}`);
 expect(Object.fromEntries(url.searchParams)).toEqual({
  preparation:prep,
  store,
  ...(stage==="missing"?{mode:"platform"}:{}),
 });
});
it("opens existing unbound drafts from the listing tabs before a store is selected",async()=>{
 render(<ListingPage initialPreparation={prep} initialStage="waiting" completionAvailable/>);
 await screen.findByText("完整批次 205 件");
 const link=screen.getByRole("link",{name:"待补全"});
 const url=new URL(link.getAttribute("href")!,"https://test.invalid");
 expect(url.pathname).toBe("/workbench/listing/missing");
 expect(Object.fromEntries(url.searchParams)).toEqual({preparation:prep});
});
it("disables selected Walmart automatic adaptation and preserves manual editing",async()=>{
 state.stores.mockResolvedValue({items:[{id:store,name:"Walmart",platform:"walmart",recordStatus:"active"}],pagination:{page:1,pageSize:100,total:1}});
 state.stages.mockResolvedValue({items:[{sourceId:source,stage:"waiting"}],total:1,batchTotal:1,counts:{all:1,waiting:1,missing:0,ready:0,review:0,uploaded:0}});
 state.detail.mockResolvedValue({source:{id:source,source:{kind:"own",productKey:"product",version:1}},product:{title:"Walmart 商品"},images:[]});
 state.target.mockResolvedValue(undefined);state.publication.mockResolvedValue({sourceId:source,storeId:store});
 render(<ListingPage initialPreparation={prep} initialStore={store} initialStage="waiting"/>);
 await userEvent.click(await screen.findByLabelText("选择 Walmart 商品"));
 expect(screen.queryByRole("button",{name:"开始适配"})).not.toBeInTheDocument();
 expect(screen.getByRole("button",{name:"预览所选 1 件"})).toBeEnabled();
 expect(screen.getByRole("button",{name:"查看 / 补全"})).toBeEnabled();expect(state.write).not.toHaveBeenCalled();
});
it("reaches Walmart repair from the listing menu and dispatches only after explicit plan approval", async () => {
 vi.stubEnv("LISTINGKIT_SUPPLY_COMPLETION_DRAFTS_ENABLED","false");
 const receipt = "44444444-4444-4444-8444-444444444444";
 const repairRecord = "55555555-5555-4555-8555-555555555555";
 const planId = "66666666-6666-4666-8666-666666666666";
 const walmartDraft = {
  selection: {feedType: "MP_ITEM_MATCH", version: "5-official", productType: ""},
  skus: [{sourceSKUKey: "source-sku", identifier: {productIdType: "GTIN", productId: "12345678901234"}, condition: "New", price: "12.50", shippingWeight: "1"}],
 };
 state.stores.mockResolvedValue({items:[{id:store,name:"Walmart",platform:"walmart",recordStatus:"active"}],pagination:{page:1,pageSize:100,total:1}});
 let repaired=false;
 state.stages.mockImplementation(async (_scope,_preparation,_store,stage) => {
  const present=repaired?stage==="waiting":stage==="uploaded";
  return {items:present?[{sourceId:source,stage:repaired?"ready":"uploaded"}]:[],total:present?1:0,batchTotal:1,counts:{all:1,waiting:0,missing:0,ready:repaired?1:0,review:0,uploaded:repaired?0:1}};
 });
 state.detail.mockResolvedValue({source:{id:source,source:{kind:"own",productKey:"product",version:"1"}},product:{title:"Walmart 商品",variants:[{source_id:"source-sku",sku:"sku",barcode:"12345678901234"}]},images:[]});
 const original = {id:source,revision:1,input:{walmart:walmartDraft},walmart:{selection:walmartDraft.selection,skus:[{sourceSKUKey:"source-sku",sellerSKU:"seller-failed"}],wireHash:"c".repeat(64),payload:btoa("{}")}};
 state.target.mockResolvedValue(original);
 const result={id:receipt,receiptId:receipt,repairableSKUs:["seller-failed"],observation:{feedId:"feed-1",observedAt:"2026-10-30T00:00:00Z",feedStatus:"PROCESSED",items:[{sku:"seller-failed",ingestionStatus:"DATA_ERROR"}]}};
 const publication={sourceId:source,storeId:store,receiptId:receipt,recordId:source,walmart:{feedId:"feed-1"},walmartResults:[result]};
 state.publication.mockResolvedValue({...publication,walmartResults:[{...result,observation:{...result.observation,partial:true}}]});
 const plan = {id:planId,revision:1,count:1,storeBindingHash:"b".repeat(64),createdAt:"2026-10-30T00:00:00Z",input:{preparationId:prep,expectedRevision:1,storeId:store,sourceIds:[source],rule:"standard",publicationMode:"platform"}};
 const previewHash = "a".repeat(64);
 state.plan.mockResolvedValue(plan);
 state.preview.mockResolvedValue({plan,items:[{item:{sourceId:source},step:"upload",facts:{hasTarget:true,target:{recordId:repairRecord,revision:2},publicationSummary:["修复原 Walmart Feed feed-1"]}}],count:1,actionable:1,counts:{upload:1},hash:previewHash});
 state.write.mockImplementation(async intent => {
  if (intent.route === "save-target") {
   repaired=true;
   const saved = {...original,id:repairRecord,revision:2,input:{walmart:intent.command.walmart}};
   state.target.mockResolvedValue(saved);
   return {record:saved};
  }
  if (intent.route === "create-upload-plan") return {plan,replayed:false};
  return {planId,operationIds:[source],replayed:false};
 });
 state.operation.mockResolvedValue({id:source,input:{action:"upload",preparationId:prep,storeId:store},status:"pending",completed:0,count:1});
 state.items.mockResolvedValue({items:[]});
 const context = {preparation:prep,store};
 const waiting = `/workbench/listing/waiting?${new URLSearchParams(context)}`;
 await expect(ListingEntry({searchParams:Promise.resolve(context)})).rejects.toMatchObject({digest:expect.stringContaining(`;${waiting};`)});
 const view=render(null);
 async function navigate(href:string){
  const url=new URL(href,"https://test.invalid");
  await act(async()=>{
   view.rerender(await ListingStage({params:Promise.resolve({stage:url.pathname.split("/").at(-1)!}),searchParams:Promise.resolve(Object.fromEntries(url.searchParams))}));
  });
 }
 await navigate(waiting);
 const uploadedHref=screen.getByRole("link",{name:/^已上传/}).getAttribute("href")!;
 await navigate(uploadedHref);
 await screen.findByText("Walmart 已受理 Feed：feed-1");
 expect(screen.queryByRole("button",{name:"查看 / 补全"})).not.toBeInTheDocument();
 expect(screen.queryByRole("button",{name:/预览.*上传/})).not.toBeInTheDocument();
 expect(screen.queryByRole("button",{name:"修复失败 SKU"})).not.toBeInTheDocument();
 state.publication.mockResolvedValue(publication);
 await userEvent.click(screen.getByRole("button",{name:"刷新"}));
 await screen.findByRole("button",{name:"修复失败 SKU"});
 const operatorPermissions=state.context.permissions;
 state.context.permissions=["workbench.supply.read","workbench.store.read"];
 await navigate(uploadedHref);
 expect(screen.queryByRole("button",{name:"修复失败 SKU"})).not.toBeInTheDocument();
 state.context.permissions=operatorPermissions;
 await navigate(uploadedHref);
 await userEvent.click(await screen.findByRole("button",{name:"修复失败 SKU"}));
 expect(screen.getByRole("button",{name:"校验并保存 Walmart 草稿"})).toBeDisabled();
 await userEvent.click(screen.getByRole("checkbox",{name:/seller-failed/}));
 await userEvent.clear(screen.getByLabelText("首次报价（USD）"));
 await userEvent.type(screen.getByLabelText("首次报价（USD）"),"13.00");
 await userEvent.click(screen.getByRole("button",{name:"校验并保存 Walmart 草稿"}));
 await waitFor(()=>expect(state.write).toHaveBeenCalledTimes(1));
 expect(state.write.mock.lastCall?.[0]).toMatchObject({route:"save-target",command:{sourceId:source,storeId:store,expectedRevision:1,walmart:{repairFrom:receipt,repairSKUs:["seller-failed"]}}});
 const handoff=await screen.findByRole("link",{name:"去待上传核对修复版"});
 expect(handoff).toHaveAttribute("href",waiting);
 expect(state.write).toHaveBeenCalledTimes(1);
 await navigate(handoff.getAttribute("href")!);
 expect(await screen.findByText("Walmart 已受理 Feed：feed-1")).toBeInTheDocument();
 await userEvent.click(await screen.findByRole("checkbox",{name:"选择 Walmart 商品"}));
 await userEvent.click(screen.getByRole("button",{name:"预览所选 1 件"}));
 const confirm = await screen.findByRole("button",{name:"确认继续 1 件"});
 expect(screen.getByText(`资料 ${repairRecord}`)).toBeInTheDocument();
 expect(state.write.mock.calls.map(([intent])=>intent.route)).toEqual(["save-target","create-upload-plan"]);
 await userEvent.click(confirm);
 await waitFor(()=>expect(state.write).toHaveBeenCalledTimes(3));
 expect(state.write.mock.lastCall?.[0]).toMatchObject({route:"continue-upload-plan",command:{planId,expectedRevision:1,previewHash}});
});
it.each(["waiting","review"] as const)("identifies same-named marketplace targets in the %s selector and confirmation",async(initialStage)=>{
 const temu="44444444-4444-4444-8444-444444444444";
 state.stores.mockResolvedValue({items:[{id:store,name:"同名店铺",platform:"shein",region:"US",recordStatus:"active",connectionStatus:"connected",serviceStatus:"active"},{id:temu,name:"同名店铺",platform:"temu",region:"GE",recordStatus:"active",connectionStatus:"connected",serviceStatus:"active"}],pagination:{page:1,pageSize:100,total:2}});
 render(<ListingPage initialPreparation={prep} initialStage={initialStage}/>);await screen.findByRole("option",{name:/同名店铺 · TEMU · GE/});
 expect(screen.getByRole("option",{name:/同名店铺 · SHEIN · US/})).toBeInTheDocument();
 await userEvent.selectOptions(screen.getByLabelText("目标店铺"),temu);
 if(initialStage==="waiting"){
  expect(screen.getByLabelText("当前目标平台")).toHaveTextContent("TEMU · GE");expect(screen.queryByText("美国站")).not.toBeInTheDocument();
 }else await userEvent.click(screen.getByRole("button",{name:"上传配置"}));
 expect(screen.getByRole("button",{name:"预览整批上传"})).toBeEnabled();
 expect(screen.getByText("同名店铺 · TEMU · GE",{selector:"p"})).toBeInTheDocument();
 expect(state.write).not.toHaveBeenCalled();
});
it("reopens a saved native TEMU record in its editor through the shared supply page",async()=>{
 state.stores.mockResolvedValue({items:[{id:store,name:"本土试用店铺",platform:"temu",recordStatus:"active",connectionStatus:"connected",serviceStatus:"active"}],pagination:{page:1,pageSize:100,total:1}});
 state.stages.mockResolvedValue({items:[{sourceId:source,stage:"missing"}],total:1,batchTotal:1,counts:{all:1,waiting:0,missing:1,ready:0,review:0,uploaded:0}});
 state.detail.mockResolvedValue({source:{id:source,source:{version:"1",kind:"own",productKey:"product-a"}},product:{title:"原生商品"},images:[]});
 state.target.mockResolvedValue({id:source,revision:1,temu:{mode:"local",readyForUpload:false,issues:[{message:"规格待核实"}]}});state.publication.mockResolvedValue({sourceId:source,storeId:store});
 render(<ListingPage initialPreparation={prep} initialStore={store}/>);await screen.findByText("原生商品");await userEvent.click(screen.getByRole("button",{name:"查看 / 补全"}));expect(await screen.findByText("TEMU 原生资料 local")).toBeInTheDocument();expect(screen.queryByText("SHEIN / 美国站")).not.toBeInTheDocument();
 expect(state.stores).toHaveBeenCalledWith(expect.not.objectContaining({platform:"shein"}),"org-a",expect.any(AbortSignal));
});
it("filters and counts the complete batch through the projection rather than the visible page",async()=>{
 const view=render(<ListingPage initialPreparation={prep} initialStage="waiting"/>);await screen.findByText("完整批次");
 await userEvent.selectOptions(await screen.findByLabelText("目标店铺"),store);
 expect(await screen.findByRole("link",{name:"待上传 (205)"})).toHaveAttribute("aria-current","page");
 state.stages.mockResolvedValue({items:[],total:0,batchTotal:205,counts:{all:205,waiting:205,missing:0,ready:0,review:0,uploaded:0}});
 const href=screen.getByRole("link",{name:"待审核 (0)"}).getAttribute("href")!;
 const url=new URL(href,"https://test.invalid");
 expect(url.pathname).toBe("/workbench/listing/review");
 view.rerender(<ListingPage initialStage="review" initialPreparation={url.searchParams.get("preparation")??undefined} initialStore={url.searchParams.get("store")??undefined}/>);
 await waitFor(()=>expect(state.stages).toHaveBeenLastCalledWith({userId:"actor-a",organizationId:"org-a"},prep,store,"review",expect.objectContaining({limit:20}),expect.any(AbortSignal)));
});
it("reads a retained batch and starts a whole-batch action without truncating it to the displayed page",async()=>{
 render(<ListingPage initialPreparation={prep}/>);await screen.findByText("完整批次");
 await userEvent.selectOptions(await screen.findByLabelText("目标店铺"),store);
 await userEvent.click(screen.getByRole("button",{name:"预览整批上传"}));
 await waitFor(()=>expect(state.write).toHaveBeenCalledOnce());
 expect(state.write.mock.calls[0][0].command).toEqual({preparationId:prep,expectedRevision:1,storeId:store,rule:"standard",publicationMode:"platform"});
 expect(state.preparation).toHaveBeenCalledWith({userId:"actor-a",organizationId:"org-a"},prep,expect.any(AbortSignal));
});
it("keeps supply writes unavailable without the management grant",async()=>{
 state.context.permissions=["workbench.supply.read","workbench.store.read"];
 render(<ListingPage initialPreparation={prep}/>);await screen.findByText("完整批次");
 await userEvent.selectOptions(await screen.findByLabelText("目标店铺"),store);
 expect(screen.getByRole("button",{name:"预览整批上传"})).toBeDisabled();
});

it("confirms full-batch retail points and pins the exact selected title template quote",async()=>{
 state.options.mockResolvedValue({titles:[{agentId:"product.title.agent",modeRevision:"0",executionMode:"default",templateId:source,revision:"3",name:"美国站标题",quoteHash:"a".repeat(64),unitVersion:"ai-micropoint-v1",maximumPoints:"10000000",priceVersion:"price-1",maximumCostMicros:1000,currency:"USD"}]});
 render(<ListingPage initialPreparation={prep}/>);await screen.findByText("完整批次");
 await userEvent.selectOptions(await screen.findByLabelText("目标店铺"),store);
 await userEvent.click(screen.getByLabelText("开启智能体优化"));
 await userEvent.selectOptions(await screen.findByLabelText("标题优化模板"),`${source}:3`);
 expect(screen.getByText(/整批预算上限 2050 积分/)).toBeInTheDocument();
 await userEvent.click(screen.getByRole("button",{name:"预览整批上传"}));
 await waitFor(()=>expect(state.write).toHaveBeenCalledOnce());
 expect(state.write.mock.calls[0][0].command).toEqual({preparationId:prep,expectedRevision:1,storeId:store,rule:"standard",publicationMode:"platform",titleTemplateId:source,titleTemplateRevision:"3",titleQuoteHash:"a".repeat(64),titleModeRevision:"0"});
});
it("does not dispatch title optimization for a TEMU store with no native review consumer",async()=>{
 const temu="44444444-4444-4444-8444-444444444444";
 state.stores.mockResolvedValue({items:[{id:store,name:"SHEIN 店铺",platform:"shein",recordStatus:"active",connectionStatus:"connected",serviceStatus:"active"},{id:temu,name:"TEMU 店铺",platform:"temu",recordStatus:"active",connectionStatus:"connected",serviceStatus:"active"}],pagination:{page:1,pageSize:100,total:2}});
 state.options.mockResolvedValue({titles:[{agentId:"product.title.agent",modeRevision:"0",executionMode:"default",templateId:source,revision:"3",name:"标题模板",quoteHash:"a".repeat(64),unitVersion:"ai-micropoint-v1",maximumPoints:"10000000",priceVersion:"price-1",maximumCostMicros:1000,currency:"USD"}]});
 render(<ListingPage initialPreparation={prep} initialStore={store}/>);await screen.findByText("完整批次");
 await userEvent.click(screen.getByLabelText("开启智能体优化"));
 await userEvent.selectOptions(await screen.findByLabelText("标题优化模板"),`${source}:3`);
 await userEvent.selectOptions(screen.getByLabelText("目标店铺"),temu);
 expect(screen.getByLabelText("开启智能体优化")).toBeDisabled();
 expect(screen.getByText(/TEMU 标题优化尚未开放/)).toBeInTheDocument();
 expect(screen.queryByRole("button",{name:"重构整批"})).not.toBeInTheDocument();
 expect(screen.getByRole("button",{name:"预览整批上传"})).toBeEnabled();
 expect(state.write).not.toHaveBeenCalled();
});

it("opens the four-entry waiting stage with its required standard rule",async()=>{
 render(<ListingPage initialPreparation={prep} initialStage="waiting"/>);await screen.findByText("完整批次");await userEvent.selectOptions(screen.getByLabelText("目标店铺"),store);
 await waitFor(()=>expect(state.stages).toHaveBeenLastCalledWith(expect.anything(),prep,store,"waiting",expect.anything(),expect.any(AbortSignal)));
 expect(screen.getByRole("link",{name:/^待上传/})).toHaveAttribute("aria-current","page");expect(screen.getByRole("button",{name:"预览整批上传"})).toBeEnabled();
 expect(screen.queryByRole("switch",{name:"启用本次规则"})).not.toBeInTheDocument();expect(state.write).not.toHaveBeenCalled();
});
it("shows the requested review list without requiring the publishing configuration panel",async()=>{
 render(<ListingPage initialPreparation={prep} initialStage="review"/>);await screen.findByText("完整批次");await userEvent.selectOptions(screen.getByLabelText("目标店铺"),store);
 await waitFor(()=>expect(state.stages).toHaveBeenLastCalledWith(expect.anything(),prep,store,"review",expect.anything(),expect.any(AbortSignal)));
 expect(screen.getByRole("link",{name:/^待审核/})).toHaveAttribute("aria-current","page");expect(screen.queryByText("发布配置流程")).not.toBeInTheDocument();
});

it("navigates dedicated stage tabs with the current batch and store",async()=>{
 render(<ListingPage initialPreparation={prep} initialStore={store} initialStage="review"/>);await screen.findByText("完整批次");expect(screen.getByRole("link",{name:/待上传/})).toHaveAttribute("href",`/workbench/listing/waiting?preparation=${prep}&store=${store}`);
});

it("keeps the selected saved operation when changing a dedicated stage",async()=>{
 const operation="44444444-4444-4444-8444-444444444444";
 state.operation.mockReset().mockResolvedValue({id:operation,input:{preparationId:prep,expectedRevision:1,storeId:store,action:"adapt"},count:205,completed:0,status:"pending",execution:"started",createdAt:"2026-10-08T00:00:00Z"});
 state.items.mockReset().mockResolvedValue({items:[],total:0});
 render(<ListingPage initialPreparation={prep} initialStore={store} initialStage="review" initialOperation={operation}/>);
 await waitFor(()=>expect(state.operation).toHaveBeenCalledWith({userId:"actor-a",organizationId:"org-a"},operation,expect.any(AbortSignal)));
 expect(screen.getByRole("link",{name:/待上传/})).toHaveAttribute("href",`/workbench/listing/waiting?preparation=${prep}&store=${store}&operation=${operation}`);
 expect(state.write).not.toHaveBeenCalled();
});

it("opens the batch review dialog with the selected actual review items",async()=>{
 state.context.permissions=[...(state.context.permissions as string[]),"listingkit.admin.write"];
 const review={sourceId:source,recordId:prep,recordRevision:1,status:"review",resultReference:store};
 state.stages.mockResolvedValue({items:[{sourceId:source,stage:"review",review}],total:1,batchTotal:1,counts:{all:1,waiting:0,missing:0,ready:0,review:1,uploaded:0}});
 state.detail.mockResolvedValue({source:{id:source,source:{kind:"own",productKey:"product",version:1}},product:{title:"待审核商品"},images:[]});
 state.target.mockResolvedValue({revision:1,result:{issues:[],ready_for_upload:false}});
 state.publication.mockResolvedValue({sourceId:source,storeId:store});
 render(<ListingPage initialPreparation={prep} initialStore={store} initialStage="review"/>);
 await userEvent.click(await screen.findByLabelText("选择 待审核商品"));
 await userEvent.click(screen.getByRole("button",{name:"批量审核通过"}));
 expect(await screen.findByRole("dialog",{name:"批量审核通过"})).toBeInTheDocument();
 expect(screen.getByText("实际批量提案 1 件")).toBeInTheDocument();expect(state.write).not.toHaveBeenCalled();
});
it("reloads the full-batch projection when the source filter changes",async()=>{
 render(<ListingPage initialPreparation={prep} initialStore={store}/>);await screen.findByText("完整批次");
 await userEvent.selectOptions(screen.getByLabelText("商品来源"),"acquisition");
 await waitFor(()=>expect(state.stages).toHaveBeenLastCalledWith(expect.anything(),prep,store,"all",expect.objectContaining({sourceKind:"acquisition"}),expect.any(AbortSignal)));
});

it("retains confirmed UNKNOWN readback across progress refresh while other items remain pending verification",async()=>{
 const operation="44444444-4444-4444-8444-444444444444",attempt="55555555-5555-4555-8555-555555555555";
 const op={id:operation,input:{preparationId:prep,expectedRevision:1,storeId:store,action:"upload"},count:2,completed:2,status:"completed",execution:"started",createdAt:"2026-10-08T00:00:00Z"};
 const original={sourceId:source,recordId:prep,recordRevision:1,status:"unknown",resultReference:attempt,note:"原请求响应丢失"};
 const other={sourceId:store,recordId:source,recordRevision:1,status:"unknown",resultReference:operation};
 const product={spu_name:"confirmed-spu",skc_list:[{skc_name:"confirmed-skc",sku_list:[{sku_code:"confirmed-sku",supplier_sku:"sku-a"}]}]};
 state.operation.mockResolvedValue(op);state.history.mockResolvedValue({items:[op],total:1});state.items.mockResolvedValue({items:[original,other],total:2});
 state.attempt.mockResolvedValue({recordId:prep,attemptId:attempt,effectKind:"publish",status:"outcome_unknown",message:"请核实原商品"});
 state.write.mockImplementation(async()=>{state.items.mockResolvedValue({items:[{...original,confirmedProduct:product},other],total:2});return {recordId:prep,publishedRecordId:prep,attemptId:attempt,status:"succeeded",currentRecordUploaded:true,product,message:"原上传已核实"}});
 const view=render(<ListingPage initialPreparation={prep} initialStore={store} initialOperation={operation}/>);
 const buttons=await screen.findAllByRole("button",{name:"查看待核实结果"});await userEvent.click(buttons[0]);
 await userEvent.type(await screen.findByLabelText("原商品 SPU"),"confirmed-spu");await userEvent.click(screen.getByRole("button",{name:"从官方查询并核实"}));
 expect(await screen.findByText("原执行 UNKNOWN，已核实成功")).toBeInTheDocument();expect(screen.getByText("SPU：confirmed-spu")).toBeInTheDocument();expect(screen.getAllByRole("button",{name:"查看待核实结果"})).toHaveLength(1);
 view.unmount();render(<ListingPage initialPreparation={prep} initialStore={store} initialOperation={operation}/>);
 expect(await screen.findByText("原执行 UNKNOWN，已核实成功")).toBeInTheDocument();expect(screen.getByText("SPU：confirmed-spu")).toBeInTheDocument();expect(screen.getAllByRole("button",{name:"查看待核实结果"})).toHaveLength(1);expect(state.write).toHaveBeenCalledOnce();
});

it("shows four configuration steps before a batch is selected",async()=>{
 render(<ListingPage initialStage="waiting"/>);
 for(const name of ["1. 选择商品","2. 选择目标店铺","3. 配置规则与智能体","4. 确认提交"]){
  expect(screen.getByRole("heading",{name})).toBeInTheDocument();
 }
 expect(screen.getByRole("button",{name:"选择商品"})).toBeEnabled();
 expect(screen.getByRole("button",{name:"预览整批上传"})).toBeDisabled();
 await userEvent.click(screen.getByRole("button",{name:"选择商品"}));
 await userEvent.click(await screen.findByRole("button",{name:"选择批次"}));
 expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
 await screen.findByRole("button",{name:"更换商品批次"});
 expect(state.write).not.toHaveBeenCalled();
});

it("shows four configuration steps in product/store/rules/confirmation order",async()=>{
 render(<ListingPage initialPreparation={prep} initialStore={store} initialStage="waiting"/>);
 await screen.findByText("完整批次");
 const titles=screen.getAllByRole("heading",{level:2}).map(v=>v.textContent);
 expect(titles).toContain("2. 选择目标店铺");
 expect(titles).toContain("3. 配置规则与智能体");
 expect(titles).toContain("4. 确认提交");
 expect(screen.queryByRole("tab",{name:/已适配/})).not.toBeInTheDocument();
});

it("allows a read-only user to select a batch without enabling upload commands",async()=>{
 state.context.permissions=["workbench.supply.read"];
 render(<ListingPage initialStage="waiting"/>);
 expect(screen.getByRole("button",{name:"选择商品"})).toBeEnabled();
 await userEvent.click(screen.getByRole("button",{name:"选择商品"}));
 await userEvent.click(await screen.findByRole("button",{name:"选择批次"}));
 expect(await screen.findByRole("button",{name:"更换商品批次"})).toBeEnabled();
 expect(screen.getByRole("button",{name:"预览整批上传"})).toBeDisabled();
 expect(state.write).not.toHaveBeenCalled();
});

it("keeps uploaded records read-only while retaining the original product and verification entry",async()=>{
 state.plan.mockRejectedValue(new Error("uploaded must not mount execution plan"));
 state.stages.mockResolvedValue({items:[{sourceId:source,stage:"uploaded"}],total:1,batchTotal:1,counts:{all:1,waiting:0,missing:0,ready:0,review:0,uploaded:1}});
 state.detail.mockResolvedValue({source:{id:source,preparationId:prep,source:{kind:"own",productKey:"product",version:1}},product:{title:"原上传商品"},images:[]});
 state.target.mockResolvedValue({id:source,revision:1,result:{issues:[],ready_for_upload:true}});
 state.publication.mockResolvedValue({sourceId:source,storeId:store,recordId:source,receiptId:prep,product:{spu_name:"original-spu",skc_list:[]}});
 render(<ListingPage initialPreparation={prep} initialStore={store} initialStage="uploaded" initialPlan={prep}/>);
 await screen.findByText("原上传商品");
 for(const name of ["查看 / 补全","上传配置","配置所选商品上传","检查进度并继续","停止后续步骤","新建上传计划"])expect(screen.queryByRole("button",{name})).not.toBeInTheDocument();
 expect(screen.getByRole("button",{name:"查看商品"})).toBeEnabled();expect(state.write).not.toHaveBeenCalled();
});

it("opens the first SHEIN platform editor from the actual waiting route",async()=>{
 const fixture=sheinImportFixture();
 const detail={...fixture.detail,source:{...fixture.detail.source,id:source,preparationId:prep},product:{...fixture.detail.product,title:"合成首次建档商品"}};
 state.detail.mockResolvedValue(detail);
 state.stages.mockImplementation(async (_scope,_preparation,_store,stage)=>({
  items:stage==="waiting"?[{sourceId:source,stage:"waiting"}]:[],
  total:stage==="waiting"?1:0,batchTotal:1,
  counts:{all:1,waiting:1,missing:0,ready:0,review:0,uploaded:0},
 }));
 state.target.mockRejectedValue(new SupplyAPIError("NOT_FOUND",404));
 state.publication.mockResolvedValue({sourceId:source,storeId:store});
 state.rules.mockResolvedValue(fixture.rules);
 state.completion.mockResolvedValue({...fixture.candidate,source:detail.source});
 const page=await ListingStage({
  params:Promise.resolve({stage:"waiting"}),
  searchParams:Promise.resolve({preparation:prep,store}),
 });
 render(page);
 await userEvent.click(await screen.findByRole("button",{name:"查看 / 补全"}));
 const dialog=await screen.findByRole("dialog",{name:"商品资料与平台适配"});
 expect(dialog).toBeVisible();
 expect(await screen.findByLabelText("商品标题内容 1")).toHaveValue("合成首次建档商品");
 await waitFor(()=>expect(state.rules).toHaveBeenCalledWith(
  {userId:"actor-a",organizationId:"org-a"},
  expect.objectContaining({sourceId:source,storeId:store,expectedRevision:0}),
  expect.any(AbortSignal),
 ));
 expect(state.write).not.toHaveBeenCalled();
});