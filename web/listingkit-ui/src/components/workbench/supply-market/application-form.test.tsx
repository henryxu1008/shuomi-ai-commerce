import {cleanup, render, screen, waitFor} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {QueryClient, QueryClientProvider} from "@tanstack/react-query";
import {afterEach, beforeEach, expect, it, vi} from "vitest";
import {listOwnProducts, readCollectionBatch} from "@/lib/api/product-collection";
import {readMarketChoice} from "@/lib/api/supply-market";
import {SupplyApplicationForm} from "./applications";

const state = vi.hoisted(() => ({
  context:{user:{id:"member-a"},effectiveOrganization:{id:"org-a"},isLoading:false,isSwitching:false,error:null,blockingError:null},
  execute:vi.fn(),locked:false,
}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>state.context}));
vi.mock("@/lib/api/product-collection",()=>({listOwnProducts:vi.fn(),readCollectionBatch:vi.fn()}));
vi.mock("@/lib/api/supply-market",async original=>({...await original<typeof import("@/lib/api/supply-market")>(),readMarketChoice:vi.fn()}));
vi.mock("./shared",async original=>({...await original<typeof import("./shared")>(),useMarketCommands:()=>({execute:state.execute,locked:state.locked,saved:null,notice:null})}));
vi.mock("./qualifications",()=>({Qualifications:({onIDs,onLocked}:{onIDs:(ids:string[])=>void;onLocked:(locked:boolean)=>void})=><><button type="button" onClick={()=>onIDs([fileId])}>保存测试资质</button><button type="button" onClick={()=>onLocked(true)}>暂存附件待确认</button><button type="button" onClick={()=>onLocked(false)}>继续确认原附件</button></>}));
const id="11111111-1111-4111-8111-111111111111",other="22222222-2222-4222-8222-222222222222",foreign="33333333-3333-4333-8333-333333333333",batch="44444444-4444-4444-8444-444444444444",fileId="55555555-5555-4555-8555-555555555555";
const item=(itemId=id,kind="own")=>({id:itemId,batchId:batch,revision:1,createdAt:"2026-10-15T00:00:00Z",title:itemId===id?"已优化自有外套":"未优化自有商品",source:{kind:kind as "own",productKey:"own-"+itemId,publicationId:batch,version:"1"}});
const selection={itemId:id,expectedRevision:1,originalPublicationId:batch,originalVersion:"1",effectiveVersion:"2",applyId:fileId};
const choice=(itemId=id,optimized=true):Awaited<ReturnType<typeof readMarketChoice>>=>{
  const product={title:itemId===id?"已优化自有外套":"未优化自有商品",images:["https://images.example.org/own.png"],attributes:{},variants:[]};
  return optimized?{selection:{...selection,itemId},optimized:true,product}:{selection:{...selection,itemId,effectiveVersion:"1",applyId:undefined},optimized:false,ineligibleReason:"not_optimized",product};
};
beforeEach(()=>{
  state.context.effectiveOrganization.id="org-a";state.context.isLoading=false;state.context.isSwitching=false;state.locked=false;state.execute.mockReset();
  vi.mocked(listOwnProducts).mockReset().mockResolvedValue({items:[item(),item(other),item(foreign,"acquisition")],total:3});
  vi.mocked(readMarketChoice).mockReset().mockImplementation(async(_scope,itemId)=>choice(itemId,itemId===id));
  vi.mocked(readCollectionBatch).mockReset().mockResolvedValue({id:batch,name:"新品开发",kind:"own",revision:1,count:3,supplyTransferSupported:true,createdAt:"2026-10-15T00:00:00Z"});
});
afterEach(cleanup);
function page(official=false){const client=new QueryClient({defaultOptions:{queries:{retry:false,gcTime:0}}});return {client,...render(<QueryClientProvider client={client}><SupplyApplicationForm official={official}/></QueryClientProvider>)};}
async function advance(){const user=userEvent.setup();await user.click(await screen.findByRole("radio",{name:"选择 已优化自有外套"}));await user.click(screen.getByRole("button",{name:"下一步：填写供货信息"}));return user;}
it("shows the Figma selection step and only optimized own candidates",async()=>{
  page();await screen.findByRole("radio",{name:"选择 已优化自有外套"});
  expect(screen.getByRole("heading",{level:1,name:"选择申请商品"})).toBeVisible();
  expect(screen.getByRole("table",{name:"自有商品申请资格"})).toBeVisible();
  expect(screen.getByRole("radio",{name:"选择 未优化自有商品"})).toBeDisabled();
  expect(screen.getAllByText("新品开发")).toHaveLength(2);
  expect(screen.getByRole("img",{name:"已优化自有外套"})).toHaveAttribute("src","https://images.example.org/own.png");
  expect(readMarketChoice).toHaveBeenCalledTimes(2);
  expect(readMarketChoice).not.toHaveBeenCalledWith(expect.anything(),foreign,expect.anything());
  expect(screen.getByRole("button",{name:"下一步：填写供货信息"})).toBeDisabled();
  expect(screen.queryByRole("heading",{name:"供货与交付"})).not.toBeInTheDocument();
});
it("keeps the existing single-item command through supply, qualifications and return",async()=>{
  page();const user=await advance();
  expect(screen.getByRole("heading",{level:1,name:"填写供货信息"})).toBeVisible();
  expect(screen.getByRole("heading",{name:"合作与资质"})).toBeVisible();
  await user.clear(screen.getByLabelText("可供库存（件）"));await user.type(screen.getByLabelText("可供库存（件）"),"12");
  await user.type(screen.getByLabelText("发货省份"),"浙江");await user.type(screen.getByLabelText("发货城市"),"杭州");
  await user.click(screen.getByRole("button",{name:"保存测试资质"}));
  await user.click(screen.getByRole("checkbox",{name:/我确认这是本人/}));
  await user.click(screen.getByRole("button",{name:"返回选择商品"}));
  await user.click(screen.getByRole("button",{name:"下一步：填写供货信息"}));
  expect(screen.getByLabelText("可供库存（件）")).toHaveValue(12);
  await user.click(screen.getByRole("button",{name:"提交申请"}));
  expect(state.execute).toHaveBeenCalledExactlyOnceWith({action:"submit_selected",selection,supply:{stock:12,capacity:"",minimumQuantity:1,leadDays:0,province:"浙江",city:"杭州",priceNote:"",afterSaleNote:""},fileIds:[fileId],disclosure:true});
});
it("explains an empty own library and links to the real add-product page",async()=>{
  vi.mocked(listOwnProducts).mockResolvedValue({items:[],total:0});page();
  await screen.findByText("暂无自有商品");
  expect(screen.getByRole("link",{name:"添加自有商品"})).toHaveAttribute("href","/workbench/data/mine/own/new");
  expect(screen.getByRole("button",{name:"下一步：填写供货信息"})).toBeDisabled();
});
it("shows missing optimization as a disabled product instead of hiding it",async()=>{
  vi.mocked(listOwnProducts).mockResolvedValue({items:[item(other)],total:1});page();
  expect(await screen.findByRole("radio",{name:"选择 未优化自有商品"})).toBeDisabled();
  expect(screen.getByText("未完成智能优化")).toBeVisible();
  expect(screen.getByRole("button",{name:"下一步：填写供货信息"})).toBeDisabled();
  expect(state.execute).not.toHaveBeenCalled();
});
it("blocks the supply step while eligibility is unknown or a refresh fails",async()=>{
  const {client}=page();const user=userEvent.setup();
  await user.click(await screen.findByRole("radio",{name:"选择 已优化自有外套"}));
  vi.mocked(readMarketChoice).mockRejectedValue(new Error("unavailable"));
  await client.invalidateQueries({queryKey:["supply-market","member-a","org-a","application-products"]});
  await screen.findByRole("button",{name:"重新读取"});
  expect(screen.getByRole("button",{name:"下一步：填写供货信息"})).toBeDisabled();
  expect(screen.queryByRole("radio")).not.toBeInTheDocument();
  expect(state.execute).not.toHaveBeenCalled();
});
it("clears selection when searching another page of own products",async()=>{
  page();const user=userEvent.setup();await user.click(await screen.findByRole("radio",{name:"选择 已优化自有外套"}));
  await user.type(screen.getByRole("textbox",{name:"商品关键词"}),"新搜索");await user.click(screen.getByRole("button",{name:"搜索"}));
  await waitFor(()=>expect(listOwnProducts).toHaveBeenLastCalledWith({userId:"member-a",organizationId:"org-a"},{keyword:"新搜索",after:undefined,limit:20},expect.any(AbortSignal)));
  expect(screen.getByRole("button",{name:"下一步：填写供货信息"})).toBeDisabled();
});
it("unmounts the old supply form when the enterprise changes",async()=>{
  const {rerender,client}=page();await advance();
  state.context.effectiveOrganization.id="org-b";
  rerender(<QueryClientProvider client={client}><SupplyApplicationForm/></QueryClientProvider>);
  await screen.findByRole("heading",{name:"选择申请商品",level:1});
  expect(screen.queryByRole("button",{name:"提交申请"})).not.toBeInTheDocument();
  expect(screen.getByRole("button",{name:"下一步：填写供货信息"})).toBeDisabled();
  expect(state.execute).not.toHaveBeenCalled();
});
it("leaves qualification recovery usable while pending attachments block navigation",async()=>{
  page();const user=await advance();
  await user.click(screen.getByRole("button",{name:"暂存附件待确认"}));
  expect(screen.getByRole("button",{name:"提交申请"})).toBeDisabled();
  expect(screen.getByRole("button",{name:"返回选择商品"})).toBeDisabled();
  expect(screen.getByRole("button",{name:"继续确认原附件"})).toBeEnabled();
  await user.click(screen.getByRole("button",{name:"继续确认原附件"}));
  expect(screen.getByRole("button",{name:"返回选择商品"})).toBeEnabled();
});
it("blocks a stale item revision instead of turning it into a candidate",async()=>{
  vi.mocked(readMarketChoice).mockResolvedValue({...choice(),selection:{...selection,expectedRevision:2}});page();
  await screen.findByRole("button",{name:"重新读取"});
  expect(screen.queryByRole("radio")).not.toBeInTheDocument();
  expect(screen.getByRole("button",{name:"下一步：填写供货信息"})).toBeDisabled();
});
it("requires and allows a fresh selection after the product version changes",async()=>{
  const {client}=page();const user=userEvent.setup();
  await user.click(await screen.findByRole("radio",{name:"选择 已优化自有外套"}));
  vi.mocked(listOwnProducts).mockResolvedValue({items:[{...item(),revision:2}],total:1});
  vi.mocked(readMarketChoice).mockResolvedValue({...choice(),selection:{...selection,expectedRevision:2}});
  await client.invalidateQueries({queryKey:["supply-market","member-a","org-a","application-products"]});
  const radio=await screen.findByRole("radio",{name:"选择 已优化自有外套"});
  expect(radio).not.toBeChecked();
  expect(screen.getByRole("button",{name:"下一步：填写供货信息"})).toBeDisabled();
  await user.click(radio);
  expect(screen.getByRole("button",{name:"下一步：填写供货信息"})).toBeEnabled();
});
it("aborts the candidate read and removes the old page during an enterprise switch",async()=>{
  const {client,rerender}=page();await screen.findByRole("radio",{name:"选择 已优化自有外套"});
  vi.mocked(readMarketChoice).mockImplementation((_scope,_id,signal)=>new Promise((_resolve,reject)=>signal?.addEventListener("abort",()=>reject(new Error("aborted")))));
  void client.invalidateQueries({queryKey:["supply-market","member-a","org-a","application-products"]});
  await waitFor(()=>expect(vi.mocked(readMarketChoice).mock.calls.length).toBeGreaterThan(2));
  const signal=vi.mocked(readMarketChoice).mock.calls.at(-1)![2]!;
  state.context.isSwitching=true;rerender(<QueryClientProvider client={client}><SupplyApplicationForm/></QueryClientProvider>);
  await waitFor(()=>expect(signal.aborted).toBe(true));
  expect(screen.queryByRole("table")).not.toBeInTheDocument();
});
it("retains the separate platform official-draft entry",async()=>{
  page(true);await screen.findByRole("combobox",{name:"申请商品"});
  expect(screen.getByRole("heading",{level:1,name:"建立自营发布草稿"})).toBeVisible();
  expect(screen.getByRole("button",{name:"保存发布草稿"})).toBeDisabled();
  expect(screen.queryByRole("button",{name:"下一步：填写供货信息"})).not.toBeInTheDocument();
});
it.each(["not_optimized","incomplete_product"] as const)("keeps a no-image %s product visible beside an eligible own product",async(reason)=>{
  vi.mocked(readMarketChoice).mockImplementation(async(_scope,itemId)=>itemId===id?choice():reason==="not_optimized"?{selection:{...selection,itemId,effectiveVersion:"1",applyId:undefined},optimized:false,ineligibleReason:reason}:{selection:{...selection,itemId},optimized:true,ineligibleReason:reason});
  page();const user=userEvent.setup();
  expect(await screen.findByRole("radio",{name:"选择 未优化自有商品"})).toBeDisabled();
  expect(screen.getByText(reason==="not_optimized"?"未完成智能优化":"商品资料不符合申请要求")).toBeVisible();
  expect(screen.getByText("暂无图片")).toBeVisible();
  expect(screen.getByText("本页 1 件可申请")).toBeVisible();
  await user.click(screen.getByRole("radio",{name:"选择 已优化自有外套"}));
  expect(screen.getByRole("button",{name:"下一步：填写供货信息"})).toBeEnabled();
  expect(state.execute).not.toHaveBeenCalled();
});
it("blocks an open supply form when the same exact optimized choice loses completeness",async()=>{
  const {client}=page();await advance();
  vi.mocked(readMarketChoice).mockResolvedValue({selection,optimized:true,ineligibleReason:"incomplete_product"});
  await client.invalidateQueries({queryKey:["supply-market","member-a","org-a","application-products"]});
  await screen.findByText("商品或申请资格已变化，请返回选择商品并重新确认。");
  expect(screen.getByLabelText("可供库存（件）")).toBeDisabled();
  expect(screen.getByRole("button",{name:"提交申请"})).toBeDisabled();
  expect(state.execute).not.toHaveBeenCalled();
});
it("keeps unoptimized complete products usable for the official draft only",async()=>{
  vi.mocked(readMarketChoice).mockResolvedValue(choice(other,false));page(true);const user=userEvent.setup();
  await user.selectOptions(await screen.findByRole("combobox",{name:"申请商品"}),other);
  await screen.findByText("自有商品原始资料");
  await user.click(screen.getByRole("checkbox",{name:/我确认这是本人/}));
  expect(screen.getByRole("button",{name:"保存发布草稿"})).toBeEnabled();
});
it("shows an incomplete official candidate without crashing or allowing save",async()=>{
  vi.mocked(readMarketChoice).mockResolvedValue({selection:{...selection,itemId:other,effectiveVersion:"1",applyId:undefined},optimized:false,ineligibleReason:"not_optimized"});
  page(true);const user=userEvent.setup();await user.selectOptions(await screen.findByRole("combobox",{name:"申请商品"}),other);
  await screen.findByText("商品资料不符合发布要求，请先完善自有商品。");
  await user.click(screen.getByRole("checkbox",{name:/我确认这是本人/}));
  expect(screen.getByRole("button",{name:"保存发布草稿"})).toBeDisabled();
  expect(state.execute).not.toHaveBeenCalled();
});
it("blocks an official cached choice after a failed refresh",async()=>{
  vi.mocked(readMarketChoice).mockResolvedValue(choice(other,false));const {client}=page(true);const user=userEvent.setup();
  await user.selectOptions(await screen.findByRole("combobox",{name:"申请商品"}),other);await screen.findByText("自有商品原始资料");
  await user.click(screen.getByRole("checkbox",{name:/我确认这是本人/}));
  expect(screen.getByRole("button",{name:"保存发布草稿"})).toBeEnabled();
  vi.mocked(readMarketChoice).mockRejectedValue(new Error("unavailable"));
  await client.invalidateQueries({queryKey:["supply-market","member-a","org-a","choice",other]});
  await screen.findByRole("button",{name:"重新读取"});
  expect(screen.getByRole("button",{name:"保存发布草稿"})).toBeDisabled();
  expect(state.execute).not.toHaveBeenCalled();
});
