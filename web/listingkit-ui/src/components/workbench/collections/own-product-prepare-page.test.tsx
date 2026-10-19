import {cleanup,fireEvent,render,screen,waitFor} from "@testing-library/react";
import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {SupplyAPIError} from "@/lib/api/supply-chain";
import {OwnProductPreparePage} from "./own-product-prepare-page";
import {parseSkillLaunch} from "@/lib/contracts/skill-launch";
const f=vi.hoisted(()=>({context:{user:{id:"actor"},effectiveOrganization:{id:"org"},permissions:["workbench.collection.read","workbench.supply.read","workbench.supply.manage"],supplyIntentReady:true,pendingSupplyIntent:null as unknown,setPendingSupplyIntent:vi.fn<(intent:unknown)=>boolean>(),registerOrganizationSwitchGuard:()=>()=>{}},item:vi.fn(),batch:vi.fn(),sources:vi.fn(),save:vi.fn(),recover:vi.fn(),push:vi.fn(),order:[] as string[]}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>f.context}));
vi.mock("next/navigation",()=>({useRouter:()=>({push:f.push})}));
vi.mock("@/lib/api/product-collection",async original=>({...await original<object>(),readCollectionItem:f.item,readCollectionBatch:f.batch}));
vi.mock("@/lib/api/supply-chain",async original=>({...await original<object>(),listSupplySources:f.sources,supplyCommand:f.save,readSupplyCommand:f.recover}));
vi.mock("@/lib/api/skill",()=>({skillRequest:vi.fn(async()=>({items:[{skill:{id:"55555555-5555-4555-8555-555555555555",owner:{kind:"PLATFORM"}},version:2,package:{name:"standard",target:"product-image",digest:"a".repeat(64)}}]}))}));
const itemId="11111111-1111-4111-8111-111111111111",batchId="22222222-2222-4222-8222-222222222222",preparationId="33333333-3333-4333-8333-333333333333",sourceId="44444444-4444-4444-8444-444444444444";
const detail=(version="2")=>({item:{id:itemId,batchId,revision:2,createdAt:"2026-10-14T00:00:00Z",source:{kind:"own",productKey:"own-product",publicationId:sourceId,version}},product:{title:`商品版本${version}`,description:"原始描述",attributes:[{name:"材质",value:"棉"}],images:[{url:"https://images.example.org/red.png"}],variants:[{source_id:"red",sku:"RED",title:"红色",price:{currency:"CNY",amount:0},stock:0}]}});
const receipt={preparation:{id:preparationId,sourceBatchId:batchId,sourceRevision:5,name:"自有商品库",count:1,revision:1,createdAt:"2026-10-14T00:00:00Z"},replayed:false};
const retained={id:sourceId,preparationId,collectionItemId:itemId,collectionRevision:2,source:detail().item.source};
const launch={userId:"actor",organizationId:"org",target:"product-image" as const,selection:{skillId:"55555555-5555-4555-8555-555555555555",version:2,digest:"a".repeat(64)}};
it("uses explicit single-item transfer then opens originals with the exact skill",async()=>{
 f.context.permissions.push("listingkit.image_agent.read","listingkit.image_agent.write");
 render(<OwnProductPreparePage itemId={itemId} launch={launch}/>);
 const confirm=await screen.findByRole("button",{name:"确认转入并选择原图"});await waitFor(()=>expect(confirm).toBeEnabled());expect(f.save).not.toHaveBeenCalled();
 fireEvent.click(confirm);await waitFor(()=>expect(f.push).toHaveBeenCalled());const url=new URL(f.push.mock.lastCall![0],"https://test");expect(url.pathname).toBe(`/workbench/data/mine/own/${itemId}/images`);expect(url.searchParams.get("source")).toBe(sourceId);expect(url.searchParams.get("preparation")).toBe(preparationId);expect(parseSkillLaunch(url.searchParams)).toEqual(launch);
});
it("retains an old pending transfer command while carrying a new navigation choice",async()=>{
 f.context.permissions.push("listingkit.image_agent.read","listingkit.image_agent.write");const intent={userId:"actor",organizationId:"org",route:"transfer",key:sourceId,command:{batchId,expectedRevision:1,itemIds:[itemId]}};f.context.pendingSupplyIntent=intent;
 f.recover.mockResolvedValue({...receipt,preparation:{...receipt.preparation,sourceRevision:1},replayed:true});f.sources.mockResolvedValue({items:[{...retained,source:{...retained.source,version:"1"}}],total:1});
 render(<OwnProductPreparePage itemId={itemId} launch={launch}/>);fireEvent.click(await screen.findByRole("button",{name:"核实原操作"}));await waitFor(()=>expect(f.push).toHaveBeenCalled());expect(f.recover.mock.lastCall![0]).toEqual(intent);expect(f.save).not.toHaveBeenCalled();
});
beforeEach(()=>{vi.clearAllMocks();f.order=[];f.context.pendingSupplyIntent=null;f.context.effectiveOrganization.id="org";f.context.setPendingSupplyIntent.mockImplementation(v=>{f.context.pendingSupplyIntent=v;return true});f.item.mockImplementation(async()=>{f.order.push("item");return detail()});f.batch.mockImplementation(async()=>{f.order.push("batch");return{id:batchId,revision:5,count:2,supplyTransferSupported:true}});f.sources.mockResolvedValue({items:[retained],total:1});f.save.mockResolvedValue(receipt)});
afterEach(cleanup);
it("reads the batch revision before the displayed item and transfers only that item",async()=>{
 f.item.mockImplementationOnce(async()=>{f.order.push("item");return detail("1")});
 render(<OwnProductPreparePage itemId={itemId}/>);
 expect(await screen.findByText("商品版本2")).toBeVisible();expect(f.order).toEqual(["item","batch","item"]);expect(screen.queryByText("商品版本1")).not.toBeInTheDocument();
 expect(screen.getByText("CNY 0")).toBeVisible();expect(screen.getByText("0 件")).toBeVisible();expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
 fireEvent.click(screen.getByRole("button",{name:"确认转入并填写草稿"}));
 await waitFor(()=>expect(f.save).toHaveBeenCalledWith(expect.objectContaining({route:"transfer",command:{batchId,expectedRevision:5,itemIds:[itemId]}}),expect.any(AbortSignal)));
 await waitFor(()=>expect(f.push).toHaveBeenCalledWith(`/workbench/supply/mine/missing/${sourceId}?preparation=${preparationId}&item=${itemId}`));
});
it("requires a fresh confirmation after a conflict",async()=>{
 f.save.mockRejectedValue(new SupplyAPIError("REVISION_CONFLICT",409));render(<OwnProductPreparePage itemId={itemId}/>);await screen.findByText("商品版本2");fireEvent.click(screen.getByRole("button",{name:"确认转入并填写草稿"}));
 await screen.findByText("资料已发生变化，请重新读取后补齐。");expect(screen.getByRole("button",{name:"确认转入并填写草稿"})).toBeDisabled();expect(f.push).not.toHaveBeenCalled();
 fireEvent.click(screen.getByRole("button",{name:"重新读取商品"}));await waitFor(()=>expect(screen.getByRole("button",{name:"确认转入并填写草稿"})).toBeEnabled());
});
it("recovers an original single-item transfer even after the current product changed",async()=>{
 const intent={userId:"actor",organizationId:"org",route:"transfer",key:sourceId,command:{batchId,expectedRevision:1,itemIds:[itemId]}};f.context.pendingSupplyIntent=intent;
 f.recover.mockResolvedValue({...receipt,preparation:{...receipt.preparation,sourceRevision:1},replayed:true});f.sources.mockResolvedValue({items:[{...retained,source:{...retained.source,version:"1"}}],total:1});
 render(<OwnProductPreparePage itemId={itemId}/>);fireEvent.click(await screen.findByRole("button",{name:"核实原操作"}));
 await waitFor(()=>expect(f.push).toHaveBeenCalledWith(`/workbench/supply/mine/missing/${sourceId}?preparation=${preparationId}&item=${itemId}`));expect(f.recover.mock.lastCall![0]).toEqual(intent);expect(f.save).not.toHaveBeenCalled();
});
it("keeps a successful transfer and retries only source reading when that read fails",async()=>{
 f.sources.mockRejectedValueOnce(new SupplyAPIError("DEPENDENCY_UNAVAILABLE",503));render(<OwnProductPreparePage itemId={itemId}/>);await screen.findByText("商品版本2");fireEvent.click(screen.getByRole("button",{name:"确认转入并填写草稿"}));
 fireEvent.click(await screen.findByRole("button",{name:"重新读取已转入来源"}));await waitFor(()=>expect(f.push).toHaveBeenCalled());expect(f.save).toHaveBeenCalledOnce();
});
it("rejects a mismatched retained item instead of opening its editor",async()=>{
 f.sources.mockResolvedValue({items:[{...retained,collectionItemId:batchId}],total:1});render(<OwnProductPreparePage itemId={itemId}/>);await screen.findByText("商品版本2");fireEvent.click(screen.getByRole("button",{name:"确认转入并填写草稿"}));
 await screen.findByRole("button",{name:"重新读取已转入来源"});expect(f.push).not.toHaveBeenCalled();
});
it("rejects a fresh transfer source that differs from the confirmed version",async()=>{
 f.sources.mockResolvedValue({items:[{...retained,source:{...retained.source,version:"3"}}],total:1});render(<OwnProductPreparePage itemId={itemId}/>);await screen.findByText("商品版本2");fireEvent.click(screen.getByRole("button",{name:"确认转入并填写草稿"}));
 await screen.findByRole("button",{name:"重新读取已转入来源"});expect(f.push).not.toHaveBeenCalled();
});
