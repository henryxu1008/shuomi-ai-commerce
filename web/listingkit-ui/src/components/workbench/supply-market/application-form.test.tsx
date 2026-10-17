import {act,cleanup, render, screen, waitFor} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {QueryClient, QueryClientProvider} from "@tanstack/react-query";
import {afterEach, beforeEach, expect, it, vi} from "vitest";
import {listOwnProducts, readCollectionBatch} from "@/lib/api/product-collection";
import {readMarketChoice} from "@/lib/api/supply-market";
import {SupplyApplicationForm} from "./applications";
import type {ApplicationResult} from "./use-selected-batch";
import type {MarketReceipt} from "@/lib/contracts/supply-market";

const state = vi.hoisted(() => ({
  context:{user:{id:"member-a"},effectiveOrganization:{id:"org-a"},isLoading:false,isSwitching:false,error:null,blockingError:null},
  execute:vi.fn(),start:vi.fn(),locked:false,finishIDs:null as ((ids:string[])=>void)|null,results:[] as ApplicationResult[],saved:null as MarketReceipt|null,
}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>state.context}));
vi.mock("@/lib/api/product-collection",()=>({listOwnProducts:vi.fn(),readCollectionBatch:vi.fn()}));
vi.mock("@/lib/api/supply-market",async original=>({...await original<typeof import("@/lib/api/supply-market")>(),readMarketChoice:vi.fn()}));
vi.mock("./shared",async original=>({...await original<typeof import("./shared")>(),useMarketCommands:()=>({execute:state.execute,locked:state.locked,saved:state.saved,notice:null})}));
vi.mock("./use-selected-batch",()=>({useSelectedBatch:()=>({start:state.start,reset:()=>{state.results=[]},locked:state.locked,journal:null,results:state.results,error:null,pendingError:false,busy:false})}));
vi.mock("./qualifications",()=>({Qualifications:({onIDs,onLocked}:{onIDs:(ids:string[])=>void;onLocked:(locked:boolean)=>void})=><><button type="button" onClick={()=>onIDs([fileId])}>保存测试资质</button><button type="button" onClick={()=>{state.finishIDs=onIDs}}>保存延迟测试资质</button><button type="button" onClick={()=>onLocked(true)}>暂存附件待确认</button><button type="button" onClick={()=>onLocked(false)}>继续确认原附件</button></>}));
const id="11111111-1111-4111-8111-111111111111",other="22222222-2222-4222-8222-222222222222",foreign="33333333-3333-4333-8333-333333333333",batch="44444444-4444-4444-8444-444444444444",fileId="55555555-5555-4555-8555-555555555555";
const item=(itemId=id,kind="own")=>({id:itemId,batchId:batch,revision:1,createdAt:"2026-10-15T00:00:00Z",title:itemId===id?"已优化自有外套":"未优化自有商品",source:{kind:kind as "own",productKey:"own-"+itemId,publicationId:batch,version:"1"}});
const selection={itemId:id,expectedRevision:1,originalPublicationId:batch,originalVersion:"1",effectiveVersion:"2",applyId:fileId};
const choice=(itemId=id,optimized=true):Awaited<ReturnType<typeof readMarketChoice>>=>{
  const product={title:itemId===id?"已优化自有外套":"未优化自有商品",images:["https://images.example.org/own.png"],attributes:{},variants:[]};
  return optimized?{selection:{...selection,itemId},optimized:true,product}:{selection:{...selection,itemId,effectiveVersion:"1",applyId:undefined},optimized:false,ineligibleReason:"not_optimized",product};
};
beforeEach(()=>{
  state.context.effectiveOrganization.id="org-a";state.context.isLoading=false;state.context.isSwitching=false;state.locked=false;state.execute.mockReset();state.start.mockReset();state.results=[];state.finishIDs=null;state.saved=null;
  vi.mocked(listOwnProducts).mockReset().mockResolvedValue({items:[item(),item(other),item(foreign,"acquisition")],total:3});
  vi.mocked(readMarketChoice).mockReset().mockImplementation(async(_scope,itemId)=>choice(itemId,itemId===id));
  vi.mocked(readCollectionBatch).mockReset().mockResolvedValue({id:batch,name:"新品开发",kind:"own",revision:1,count:3,supplyTransferSupported:true,createdAt:"2026-10-15T00:00:00Z"});
});
afterEach(cleanup);
function page(official=false){const client=new QueryClient({defaultOptions:{queries:{retry:false,gcTime:0}}});return {client,...render(<QueryClientProvider client={client}><SupplyApplicationForm official={official}/></QueryClientProvider>)};}
async function advance(){const user=userEvent.setup();await user.click(await screen.findByRole("checkbox",{name:"选择 已优化自有外套"}));await user.click(screen.getByRole("button",{name:"下一步：填写供货信息"}));return user;}
it("selects several eligible products and retains them across a search",async()=>{
  vi.mocked(readMarketChoice).mockImplementation(async(_scope,itemId)=>choice(itemId,true));
  page();const user=userEvent.setup();
  await user.click(await screen.findByRole("checkbox",{name:"选择 已优化自有外套"}));
  await user.click(screen.getByRole("checkbox",{name:"选择 未优化自有商品"}));
  expect(screen.getByText("已选择 2 件")).toBeVisible();
  await user.type(screen.getByRole("textbox",{name:"商品关键词"}),"外套");
  await user.click(screen.getByRole("button",{name:"搜索"}));
  await waitFor(()=>expect(listOwnProducts).toHaveBeenLastCalledWith(expect.anything(),{keyword:"外套",after:undefined,limit:20},expect.any(AbortSignal)));
  expect(screen.getByText("已选择 2 件")).toBeVisible();
});
it("keeps common defaults and independent per-product quantities, delivery and price",async()=>{
  vi.mocked(readMarketChoice).mockImplementation(async(_scope,itemId)=>choice(itemId,true));
  page();const user=userEvent.setup();
  await user.click(await screen.findByRole("checkbox",{name:"选择 已优化自有外套"}));
  await user.click(screen.getByRole("checkbox",{name:"选择 未优化自有商品"}));
  await user.click(screen.getByRole("button",{name:"下一步：填写供货信息"}));
  await user.clear(screen.getByLabelText("可供库存（件）"));await user.type(screen.getByLabelText("可供库存（件）"),"12");
  await user.selectOptions(screen.getByRole("combobox",{name:"发货省份"}),"浙江省");await user.selectOptions(screen.getByRole("combobox",{name:"发货城市"}),"杭州市");
  await user.type(screen.getByLabelText("公共售后政策"),"公共售后");
  await user.click(screen.getAllByRole("button",{name:"保存测试资质"})[0]!);
  await user.selectOptions(screen.getByRole("combobox",{name:"正在编辑商品"}),other);
  expect(screen.getByLabelText("可供库存（件）")).toHaveValue(0);
  await user.clear(screen.getByLabelText("可供库存（件）"));await user.type(screen.getByLabelText("可供库存（件）"),"37");
  await user.type(screen.getByLabelText("供货价说明"),"按箱报价");
  await user.click(screen.getByRole("checkbox",{name:"继承公共发货地"}));
  await user.click(screen.getByRole("checkbox",{name:"继承公共售后政策"}));
  await user.clear(screen.getByLabelText("商品售后政策"));await user.type(screen.getByLabelText("商品售后政策"),"独立售后");
  await user.selectOptions(screen.getAllByRole("combobox",{name:"发货省份"})[1]!,"广东省");await user.selectOptions(screen.getAllByRole("combobox",{name:"发货城市"})[1]!,"广州市");
  await user.selectOptions(screen.getAllByRole("combobox",{name:"发货省份"})[0]!,"北京市");await user.selectOptions(screen.getAllByRole("combobox",{name:"发货城市"})[0]!,"北京市");
  expect(screen.getAllByRole("combobox",{name:"发货城市"})[1]).toHaveValue("广州市");
  await user.click(screen.getByRole("button",{name:"核对本批申请"}));
  await screen.findByRole("heading",{name:"确认本批申请",level:1});
  expect(readMarketChoice).toHaveBeenCalledTimes(4);
  await user.click(screen.getByRole("checkbox",{name:/我确认以上/}));await user.click(screen.getByRole("button",{name:"提交 2 件申请"}));
  const commands=state.start.mock.calls[0]![0] as {command:{selection:typeof selection;supply:Record<string,unknown>;fileIds:string[]}}[];
  expect(commands[0]!.command).toMatchObject({selection,supply:{stock:12,province:"北京市",city:"北京市",afterSaleNote:"公共售后",priceNote:""},fileIds:[fileId]});
  expect(commands[1]!.command).toMatchObject({selection:{...selection,itemId:other},supply:{stock:37,province:"广东省",city:"广州市",afterSaleNote:"独立售后",priceNote:"按箱报价"},fileIds:[fileId]});
});
it("refuses to review a failed or changed exact choice without consuming the old snapshot",async()=>{
  page();const user=await advance();
  await user.clear(screen.getByLabelText("可供库存（件）"));await user.type(screen.getByLabelText("可供库存（件）"),"12");
  await user.selectOptions(screen.getByRole("combobox",{name:"发货省份"}),"浙江省");await user.selectOptions(screen.getByRole("combobox",{name:"发货城市"}),"杭州市");
  await user.click(screen.getAllByRole("button",{name:"保存测试资质"})[0]!);
  vi.mocked(readMarketChoice).mockResolvedValue({...choice(),selection:{...selection,effectiveVersion:"3"}});
  await user.click(screen.getByRole("button",{name:"核对本批申请"}));await screen.findByRole("alert");
  expect(screen.queryByRole("button",{name:"提交 1 件申请"})).not.toBeInTheDocument();
  vi.mocked(readMarketChoice).mockRejectedValue(new Error("unavailable"));
  await user.click(screen.getByRole("button",{name:"核对本批申请"}));await screen.findByText("供应市场暂时不可用，请稍后重新读取。");
  expect(state.start).not.toHaveBeenCalled();
});
it("does not overwrite a newer supply edit when an earlier attachment callback completes",async()=>{
  page();const user=await advance();
  await user.clear(screen.getByLabelText("可供库存（件）"));await user.type(screen.getByLabelText("可供库存（件）"),"12");
  await user.click(screen.getAllByRole("button",{name:"保存延迟测试资质"})[1]!);
  await user.clear(screen.getByLabelText("可供库存（件）"));await user.type(screen.getByLabelText("可供库存（件）"),"24");
  await act(async()=>state.finishIDs!([fileId]));
  expect(screen.getByLabelText("可供库存（件）")).toHaveValue(24);
});
it("keeps cross-page selection bounded to twenty and permits explicit removal",async()=>{
  const ids=Array.from({length:21},()=>crypto.randomUUID());
  vi.mocked(listOwnProducts).mockResolvedValueOnce({items:ids.slice(0,20).map(value=>item(value)),total:21,nextCursor:ids[19]}).mockResolvedValue({items:[item(ids[20])],total:21});
  vi.mocked(readMarketChoice).mockImplementation(async(_scope,itemId)=>({selection:{...selection,itemId},optimized:true,product:{title:"商品"+(ids.indexOf(itemId)+1),images:["https://images.example.org/own.png"]}}));
  page();const user=userEvent.setup();await screen.findByRole("checkbox",{name:"选择 商品1"});
  for(const checkbox of screen.getAllByRole("checkbox"))await user.click(checkbox);
  expect(screen.getByText("已选择 20 件")).toBeVisible();await user.click(screen.getByRole("button",{name:"下一页"}));
  const extra=await screen.findByRole("checkbox",{name:"选择 商品21"});expect(extra).toBeDisabled();
  await user.click(screen.getByRole("button",{name:"移除 商品1"}));expect(extra).toBeEnabled();await user.click(extra);
  expect(screen.getByText("已选择 20 件")).toBeVisible();expect(state.start).not.toHaveBeenCalled();
});
it("rebuilds only definite failures from restored results, preserving the final original payload",async()=>{
  const successfulKey=crypto.randomUUID(),failedKey=crypto.randomUUID();
  const intent=(key:string,itemId:string)=>({key,admin:false,command:{action:"submit_selected" as const,selection:{...selection,itemId},supply:{stock:37,minimumQuantity:2,leadDays:3,province:"浙江省",city:"杭州市",priceNote:"原价格",afterSaleNote:"原售后"},fileIds:[fileId],disclosure:true as const}});
  state.results=[{title:"已优化自有外套",key:successfulKey,intent:intent(successfulKey,id),state:"submitted",receipt:{operationId:crypto.randomUUID(),recordId:crypto.randomUUID(),revision:1,replayed:true}},{title:"未优化自有商品",key:failedKey,intent:intent(failedKey,other),state:"failed"}];
  vi.mocked(readMarketChoice).mockImplementation(async(_scope,itemId)=>choice(itemId,true));
  page();const user=userEvent.setup();await user.click(await screen.findByRole("button",{name:"修改明确失败商品"}));
  expect(await screen.findByRole("heading",{name:"填写供货信息",level:1})).toBeVisible();
  expect(screen.getByText("已选择 1 件自有商品")).toBeVisible();expect(screen.getByLabelText("可供库存（件）")).toHaveValue(37);
  expect(screen.queryByRole("option",{name:"已优化自有外套"})).not.toBeInTheDocument();
  expect(screen.getByLabelText("商品售后政策")).toHaveValue("原售后");expect(state.start).not.toHaveBeenCalled();
});
it("shows the recovered single application's receipt and prevents an unintended new application",async()=>{
  state.saved={operationId:crypto.randomUUID(),recordId:id,revision:1,replayed:true};page();
  expect(await screen.findByText("原操作已确认。")).toBeVisible();
  expect(screen.getByRole("link",{name:"查看原操作结果"})).toHaveAttribute("href","/workbench/supply/applications/"+id);
  expect(await screen.findByRole("checkbox",{name:"选择 已优化自有外套"})).toBeDisabled();
  expect(screen.getByRole("button",{name:"下一步：填写供货信息"})).toBeDisabled();expect(state.start).not.toHaveBeenCalled();
});
it("opens My Data for a recovered select-release Collection receipt",async()=>{
  state.saved={operationId:crypto.randomUUID(),releaseId:other,collection:{operationId:crypto.randomUUID(),batchId:batch,revision:1,replayed:true},revision:1,replayed:true};page();
  expect(screen.getByRole("link",{name:"查看原操作结果"})).toHaveAttribute("href","/workbench/data/mine");
  expect(screen.getByRole("button",{name:"下一步：填写供货信息"})).toBeDisabled();expect(state.start).not.toHaveBeenCalled();
});
it("shows the Figma selection step and only optimized own candidates",async()=>{
  page();await screen.findByRole("checkbox",{name:"选择 已优化自有外套"});
  expect(screen.getByRole("heading",{level:1,name:"选择申请商品"})).toBeVisible();
  expect(screen.getByRole("table",{name:"自有商品申请资格"})).toBeVisible();
  expect(screen.getByRole("checkbox",{name:"选择 未优化自有商品"})).toBeDisabled();
  expect(screen.getAllByText("新品开发")).toHaveLength(2);
  expect(screen.getByRole("img",{name:"已优化自有外套"})).toHaveAttribute("src","https://images.example.org/own.png");
  expect(readMarketChoice).toHaveBeenCalledTimes(2);
  expect(readMarketChoice).not.toHaveBeenCalledWith(expect.anything(),foreign,expect.anything());
  expect(screen.getByRole("button",{name:"下一步：填写供货信息"})).toBeDisabled();
  expect(screen.queryByRole("heading",{name:"供货与交付"})).not.toBeInTheDocument();
});
it("reviews the final supply and explicitly confirms each independent command",async()=>{
  page();const user=await advance();
  expect(screen.getByRole("heading",{level:1,name:"填写供货信息"})).toBeVisible();
  expect(screen.getByRole("heading",{name:"逐商品供货与资质"})).toBeVisible();
  await user.clear(screen.getByLabelText("可供库存（件）"));await user.type(screen.getByLabelText("可供库存（件）"),"12");
  await user.selectOptions(screen.getByRole("combobox",{name:"发货省份"}),"浙江省");await user.selectOptions(screen.getByRole("combobox",{name:"发货城市"}),"杭州市");
  await user.click(screen.getAllByRole("button",{name:"保存测试资质"})[0]!);
  await user.click(screen.getByRole("button",{name:"返回选择商品"}));
  await user.click(screen.getByRole("button",{name:"下一步：填写供货信息"}));
  expect(screen.getByLabelText("可供库存（件）")).toHaveValue(12);
  await user.click(screen.getByRole("button",{name:"核对本批申请"}));
  await screen.findByRole("heading",{name:"确认本批申请",level:1});
  expect(screen.getByRole("button",{name:"提交 1 件申请"})).toBeDisabled();
  await user.click(screen.getByRole("checkbox",{name:/我确认以上/}));
  await user.click(screen.getByRole("button",{name:"返回修改供货信息"}));
  await user.click(screen.getByRole("button",{name:"核对本批申请"}));
  await screen.findByRole("heading",{name:"确认本批申请",level:1});
  expect(screen.getByRole("button",{name:"提交 1 件申请"})).toBeDisabled();
  await user.click(screen.getByRole("checkbox",{name:/我确认以上/}));
  await user.click(screen.getByRole("button",{name:"提交 1 件申请"}));
  expect(state.start).toHaveBeenCalledExactlyOnceWith([{title:"已优化自有外套",command:{action:"submit_selected",selection,supply:{stock:12,capacity:"",minimumQuantity:1,leadDays:0,province:"浙江省",city:"杭州市",priceNote:"",afterSaleNote:""},fileIds:[fileId],disclosure:true}}]);
});
it("requires an explicit province and clears the city when the province changes",async()=>{
  page();const user=await advance();
  const province=screen.getByRole("combobox",{name:"发货省份"}),city=screen.getByRole("combobox",{name:"发货城市"});
  expect(province).toHaveValue("");expect(province).toBeRequired();
  expect(city).toHaveValue("");expect(city).toBeDisabled();expect(city).toBeRequired();
  expect(screen.getByRole("option",{name:"请先选择省份"})).toBeVisible();
  await user.selectOptions(province,"浙江省");expect(city).toBeEnabled();
  await user.selectOptions(city,"杭州市");
  await user.selectOptions(province,"广东省");
  expect(city).toHaveValue("");expect(city).toBeInvalid();
  expect(screen.queryByRole("option",{name:"杭州市"})).not.toBeInTheDocument();
  await user.selectOptions(province,"");expect(city).toBeDisabled();expect(city).toHaveValue("");
  expect(state.execute).not.toHaveBeenCalled();
});
it.each([["北京市","北京市"],["重庆市","重庆市"],["湖北省","仙桃市"],["新疆维吾尔自治区","石河子市"]])("offers actual city names for %s",async(provinceName,cityName)=>{
  page();const user=await advance();
  await user.selectOptions(screen.getByRole("combobox",{name:"发货省份"}),provinceName);
  await user.selectOptions(screen.getByRole("combobox",{name:"发货城市"}),cityName);
  expect(screen.getByRole("combobox",{name:"发货城市"})).toHaveValue(cityName);
  expect(screen.queryByRole("option",{name:/^(县|省直辖县|自治区直辖县级行政区划)$/})).not.toBeInTheDocument();
  expect(state.execute).not.toHaveBeenCalled();
});
it("explains an empty own library and links to the real add-product page",async()=>{
  vi.mocked(listOwnProducts).mockResolvedValue({items:[],total:0});page();
  await screen.findByText("暂无自有商品");
  expect(screen.getByRole("link",{name:"添加自有商品"})).toHaveAttribute("href","/workbench/data/mine/own/new");
  expect(screen.getByRole("button",{name:"下一步：填写供货信息"})).toBeDisabled();
});
it("shows missing optimization as a disabled product instead of hiding it",async()=>{
  vi.mocked(listOwnProducts).mockResolvedValue({items:[item(other)],total:1});page();
  expect(await screen.findByRole("checkbox",{name:"选择 未优化自有商品"})).toBeDisabled();
  expect(screen.getByText("未完成智能优化")).toBeVisible();
  expect(screen.getByRole("button",{name:"下一步：填写供货信息"})).toBeDisabled();
  expect(state.execute).not.toHaveBeenCalled();
});
it("blocks the supply step while eligibility is unknown or a refresh fails",async()=>{
  const {client}=page();const user=userEvent.setup();
  await user.click(await screen.findByRole("checkbox",{name:"选择 已优化自有外套"}));
  vi.mocked(readMarketChoice).mockRejectedValue(new Error("unavailable"));
  await client.invalidateQueries({queryKey:["supply-market","member-a","org-a","application-products"]});
  await screen.findByRole("button",{name:"重新读取"});
  expect(screen.getByRole("button",{name:"下一步：填写供货信息"})).toBeDisabled();
  expect(screen.queryByRole("checkbox")).not.toBeInTheDocument();
  expect(state.execute).not.toHaveBeenCalled();
});
it("keeps selection when searching another page of own products",async()=>{
  page();const user=userEvent.setup();await user.click(await screen.findByRole("checkbox",{name:"选择 已优化自有外套"}));
  await user.type(screen.getByRole("textbox",{name:"商品关键词"}),"新搜索");await user.click(screen.getByRole("button",{name:"搜索"}));
  await waitFor(()=>expect(listOwnProducts).toHaveBeenLastCalledWith({userId:"member-a",organizationId:"org-a"},{keyword:"新搜索",after:undefined,limit:20},expect.any(AbortSignal)));
  await waitFor(()=>expect(screen.getByRole("button",{name:"下一步：填写供货信息"})).toBeEnabled());
});
it("unmounts the old supply form when the enterprise changes",async()=>{
  const {rerender,client}=page();await advance();
  state.context.effectiveOrganization.id="org-b";
  rerender(<QueryClientProvider client={client}><SupplyApplicationForm/></QueryClientProvider>);
  await screen.findByRole("heading",{name:"选择申请商品",level:1});
  expect(screen.queryByRole("button",{name:"核对本批申请"})).not.toBeInTheDocument();
  expect(screen.getByRole("button",{name:"下一步：填写供货信息"})).toBeDisabled();
  expect(state.execute).not.toHaveBeenCalled();
});
it("leaves qualification recovery usable while pending attachments block navigation",async()=>{
  page();const user=await advance();
  await user.click(screen.getAllByRole("button",{name:"暂存附件待确认"})[0]!);
  expect(screen.getByRole("button",{name:"核对本批申请"})).toBeDisabled();
  expect(screen.getByRole("button",{name:"返回选择商品"})).toBeDisabled();
  expect(screen.getAllByRole("button",{name:"继续确认原附件"})[0]!).toBeEnabled();
  await user.click(screen.getAllByRole("button",{name:"继续确认原附件"})[0]!);
  expect(screen.getByRole("button",{name:"返回选择商品"})).toBeEnabled();
});
it("blocks a stale item revision instead of turning it into a candidate",async()=>{
  vi.mocked(readMarketChoice).mockResolvedValue({...choice(),selection:{...selection,expectedRevision:2}});page();
  await screen.findByRole("button",{name:"重新读取"});
  expect(screen.queryByRole("checkbox")).not.toBeInTheDocument();
  expect(screen.getByRole("button",{name:"下一步：填写供货信息"})).toBeDisabled();
});
it("requires and allows a fresh selection after the product version changes",async()=>{
  const {client}=page();const user=userEvent.setup();
  await user.click(await screen.findByRole("checkbox",{name:"选择 已优化自有外套"}));
  vi.mocked(listOwnProducts).mockResolvedValue({items:[{...item(),revision:2}],total:1});
  vi.mocked(readMarketChoice).mockResolvedValue({...choice(),selection:{...selection,expectedRevision:2}});
  await client.invalidateQueries({queryKey:["supply-market","member-a","org-a","application-products"]});
  const radio=await screen.findByRole("checkbox",{name:"选择 已优化自有外套"});
  expect(radio).toBeChecked();
  expect(screen.getByRole("button",{name:"下一步：填写供货信息"})).toBeDisabled();
  await user.click(radio);
  await user.click(radio);
  expect(screen.getByRole("button",{name:"下一步：填写供货信息"})).toBeEnabled();
});
it("aborts the candidate read and removes the old page during an enterprise switch",async()=>{
  const {client,rerender}=page();await screen.findByRole("checkbox",{name:"选择 已优化自有外套"});
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
  expect(await screen.findByRole("checkbox",{name:"选择 未优化自有商品"})).toBeDisabled();
  expect(screen.getByText(reason==="not_optimized"?"未完成智能优化":"商品资料不符合申请要求")).toBeVisible();
  expect(screen.getByText("暂无图片")).toBeVisible();
  expect(screen.getByText("本页 1 件可申请")).toBeVisible();
  await user.click(screen.getByRole("checkbox",{name:"选择 已优化自有外套"}));
  expect(screen.getByRole("button",{name:"下一步：填写供货信息"})).toBeEnabled();
  expect(state.execute).not.toHaveBeenCalled();
});
it("blocks an open supply form when the same exact optimized choice loses completeness",async()=>{
  const {client}=page();await advance();
  vi.mocked(readMarketChoice).mockResolvedValue({selection,optimized:true,ineligibleReason:"incomplete_product"});
  await client.invalidateQueries({queryKey:["supply-market","member-a","org-a","application-products"]});
  await screen.findByText("商品或申请资格已变化，请返回选择商品并重新确认。");
  expect(screen.getByLabelText("可供库存（件）")).toBeDisabled();
  expect(screen.getByRole("button",{name:"核对本批申请"})).toBeDisabled();
  expect(state.execute).not.toHaveBeenCalled();
});
it("keeps unoptimized complete products usable for the official draft only",async()=>{
  vi.mocked(readMarketChoice).mockResolvedValue(choice(other,false));page(true);const user=userEvent.setup();
  await user.selectOptions(await screen.findByRole("combobox",{name:"申请商品"}),other);
  await screen.findByText("自有商品原始资料");
  await user.clear(screen.getByLabelText("现有库存（件）"));await user.type(screen.getByLabelText("现有库存（件）"),"12");
  await user.selectOptions(screen.getByRole("combobox",{name:"发货省份"}),"上海市");
  await user.selectOptions(screen.getByRole("combobox",{name:"发货城市"}),"上海市");
  await user.click(screen.getByRole("checkbox",{name:/我确认这是本人/}));
  expect(screen.getByRole("button",{name:"保存发布草稿"})).toBeEnabled();
  await user.click(screen.getByRole("button",{name:"保存发布草稿"}));
  expect(state.execute).toHaveBeenCalledExactlyOnceWith({action:"create_official_draft",selection:{...selection,itemId:other,effectiveVersion:"1",applyId:undefined},supply:{stock:12,capacity:"",minimumQuantity:1,leadDays:0,province:"上海市",city:"上海市",priceNote:"",afterSaleNote:""},disclosure:true});
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
