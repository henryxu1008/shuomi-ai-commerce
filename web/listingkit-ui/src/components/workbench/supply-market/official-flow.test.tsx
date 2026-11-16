import {act,cleanup,render,screen,waitFor} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {QueryClient,QueryClientProvider} from "@tanstack/react-query";
import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {SupplyMarketPage} from "./market";
import {SupplyApplicationForm,SupplyApplicationsPage} from "./applications";
import {SupplyRecordPage} from "./record";
import {MarketAPIError} from "@/lib/api/supply-market";
import type {MarketReceipt} from "@/lib/contracts/supply-market";

const state=vi.hoisted(()=>({context:{user:{id:"actor"},effectiveOrganization:{id:"org"},permissions:["workbench.collection.read","workbench.collection.manage","workbench.supply-market.apply"]},platform:vi.fn(),own:vi.fn(),choice:vi.fn(),record:vi.fn(),releases:vi.fn(),execute:vi.fn(),push:vi.fn(),saved:null as MarketReceipt|null}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>state.context}));
vi.mock("next/navigation",()=>({useRouter:()=>({push:state.push})}));
vi.mock("@/lib/api/product-collection",()=>({listOwnProducts:state.own}));
vi.mock("@/lib/api/supply-market",async original=>({...await original<object>(),listMarket:async()=>({items:[],total:0}),listMarketRecords:state.platform,readMarketChoice:state.choice,readMarketRecord:state.record,listRecordReleases:state.releases,listMarketEvents:async()=>({items:[],total:0})}));
vi.mock("./shared",async original=>({...await original<object>(),useMarketCommands:()=>({execute:state.execute,saved:state.saved,savedAction:state.saved?"create_official_draft":undefined,locked:false,notice:null})}));
const id="11111111-1111-4111-8111-111111111111",recordId="22222222-2222-4222-8222-222222222222",releaseId="33333333-3333-4333-8333-333333333333";
const product={title:"自营外套",description:"棉质外套",brand:"自营品牌",attributes:{材质:"棉"},images:["https://images.example.org/coat.png"],variants:[{sourceId:"red",sku:"RED",attributes:{颜色:"红"},currency:"CNY",price:0,stock:10}]};
const supply={stock:12,capacity:"",minimumQuantity:1,leadDays:2,province:"上海市",city:"上海市",priceNote:"",afterSaleNote:"支持售后"};
const selection={itemId:id,expectedRevision:3,originalPublicationId:id,originalVersion:"1",effectiveVersion:"1"};
function show(node:React.ReactNode){const client=new QueryClient({defaultOptions:{queries:{retry:false,gcTime:0}}});return {client,...render(<QueryClientProvider client={client}>{node}</QueryClientProvider>)}}
beforeEach(()=>{vi.clearAllMocks();state.saved=null;state.context.permissions=["workbench.collection.read","workbench.collection.manage","workbench.supply-market.apply"];state.platform.mockResolvedValue({items:[],total:0});state.own.mockResolvedValue({items:[{id,title:product.title,source:{kind:"own"}}],total:1});state.choice.mockResolvedValue({selection,product,optimized:false,ineligibleReason:"not_optimized"});state.record.mockResolvedValue({id:recordId,kind:"official",product,supply,stage:"DRAFT",revision:4});state.releases.mockResolvedValue({items:[],total:0});});
afterEach(cleanup);
it("exposes add and management from the official market for a verified specialist",async()=>{
 show(<SupplyMarketPage channel="official"/>);
 expect(await screen.findByRole("link",{name:"添加自营商品"})).toHaveAttribute("href","/workbench/supply/official/new");
 expect(screen.getByRole("link",{name:"管理自营商品"})).toHaveAttribute("href","/workbench/supply/operator?kind=official");
});
it("does not advertise publishing to an enterprise administrator without platform access",async()=>{
 state.platform.mockRejectedValue(new MarketAPIError("PERMISSION_DENIED",403));show(<SupplyMarketPage channel="official"/>);
 await screen.findByText("暂无商品");expect(screen.queryByRole("link",{name:"添加自营商品"})).not.toBeInTheDocument();
});
it("blocks new product entry when the enterprise has not granted supply permission",async()=>{
 state.context.permissions=["workbench.collection.read","workbench.collection.manage"];
 show(<SupplyApplicationForm official/>);
 await screen.findByText("当前企业下没有自营供货权限");
 expect(state.own).not.toHaveBeenCalled();expect(screen.queryByRole("link",{name:"新建商品"})).not.toBeInTheDocument();
});
it("hides add but retains management for a specialist without enterprise supply permission",async()=>{
 state.context.permissions=["workbench.collection.read","workbench.collection.manage"];
 show(<SupplyMarketPage channel="official"/>);
 await screen.findByRole("link",{name:"管理自营商品"});
 expect(screen.queryByRole("link",{name:"添加自营商品"})).not.toBeInTheDocument();
});
it("checks platform access before showing the product and supply form",async()=>{
 state.platform.mockRejectedValue(new MarketAPIError("PERMISSION_DENIED",403));show(<SupplyApplicationForm official/>);
 await screen.findByText("当前账号不能发布自营商品");expect(state.own).not.toHaveBeenCalled();expect(screen.queryByLabelText("现有库存（件）")).not.toBeInTheDocument();
});
it("offers the existing product editor with a fixed continuation and goes directly from a saved draft to preview",async()=>{
 const view=show(<SupplyApplicationForm official/>);
 expect(await screen.findByRole("link",{name:"新建商品"})).toHaveAttribute("href","/workbench/data/mine/own/new?continuation=official");
 const user=userEvent.setup();await user.selectOptions(await screen.findByRole("combobox",{name:"自有商品"}),id);await screen.findByText("自有商品原始资料");
 await user.clear(screen.getByLabelText("现有库存（件）"));await user.type(screen.getByLabelText("现有库存（件）"),"12");
 await user.selectOptions(screen.getByRole("combobox",{name:"发货省份"}),"上海市");await user.selectOptions(screen.getByRole("combobox",{name:"发货城市"}),"上海市");
 await user.click(screen.getByRole("checkbox",{name:/我确认这是本人/}));await user.click(screen.getByRole("button",{name:"保存草稿并预览"}));
 expect(state.execute).toHaveBeenCalledWith(expect.objectContaining({action:"create_official_draft",selection}));
 expect(state.push).not.toHaveBeenCalled();state.saved={operationId:recordId,recordId,revision:1,replayed:false};
 view.rerender(<QueryClientProvider client={view.client}><SupplyApplicationForm official/></QueryClientProvider>);
 await waitFor(()=>expect(state.push).toHaveBeenCalledWith(`/workbench/supply/operator/${recordId}`));
 expect(state.execute).toHaveBeenCalledTimes(1);
});
it("previews the exact draft product and publishes only on explicit confirmation",async()=>{
 show(<SupplyRecordPage id={recordId} admin expectedUserId="actor"/>);
 expect(await screen.findByRole("heading",{name:"自营商品发布预览"})).toBeVisible();
 expect(screen.getByText("棉质外套")).toBeVisible();expect(screen.getByText("RED")).toBeVisible();expect(screen.getByText("CNY 0")).toBeVisible();
 expect(state.execute).not.toHaveBeenCalled();await userEvent.click(screen.getByRole("button",{name:"确认发布"}));
 expect(state.execute).toHaveBeenCalledExactlyOnceWith({action:"publish",id:recordId,expectedRevision:4});
});
it("uses actual releases to show success even though the official record remains DRAFT",async()=>{
 state.releases.mockResolvedValue({items:[{id:releaseId,active:true,publishedAt:"2026-11-16T00:00:00Z"}],total:1});show(<SupplyRecordPage id={recordId} admin expectedUserId="actor"/>);
 expect(await screen.findByText("已发布到硕米自营")).toBeVisible();expect(screen.queryByRole("button",{name:"确认发布"})).not.toBeInTheDocument();
 expect(screen.getByRole("link",{name:"查看市场商品"})).toHaveAttribute("href",`/workbench/supply/products/${releaseId}`);
});
it("selects the exact newly saved item even when it is outside the first library page",async()=>{
 state.own.mockResolvedValue({items:[],total:35});show(<SupplyApplicationForm official initialItemId={id}/>);
 expect(await screen.findByRole("option",{name:product.title})).toHaveValue(id);
 expect(screen.getByRole("combobox",{name:"自有商品"})).toHaveValue(id);
 expect(state.choice).toHaveBeenCalledWith({userId:"actor",organizationId:"org"},id,expect.any(AbortSignal));
 expect(state.execute).not.toHaveBeenCalled();
});
it("holds the form while platform permission is unknown and provides retry on a failed read",async()=>{
 let fail!:(e:Error)=>void;state.platform.mockReturnValue(new Promise((_resolve,reject)=>{fail=reject}));show(<SupplyApplicationForm official/>);
 await screen.findByText("正在确认自营发布权限");expect(state.own).not.toHaveBeenCalled();
 await act(async()=>{fail(new MarketAPIError("DEPENDENCY_UNAVAILABLE",503))});await screen.findByRole("button",{name:"重新读取"});expect(state.own).not.toHaveBeenCalled();
});
it("removes the form after a failed platform refresh instead of keeping cached access",async()=>{
 const {client}=show(<SupplyApplicationForm official/>);await screen.findByRole("combobox",{name:"自有商品"});
 state.platform.mockRejectedValue(new MarketAPIError("PERMISSION_DENIED",403));await act(async()=>{await client.invalidateQueries({queryKey:["supply-market","actor","","records"]})});
 await screen.findByText("当前账号不能发布自营商品");expect(screen.queryByLabelText("现有库存（件）")).not.toBeInTheDocument();
});
it("does not enable publishing when the actual release state cannot be read",async()=>{
 state.releases.mockRejectedValue(new MarketAPIError("DEPENDENCY_UNAVAILABLE",503));show(<SupplyRecordPage id={recordId} admin expectedUserId="actor"/>);
 expect(await screen.findByRole("button",{name:"确认发布"})).toBeDisabled();await screen.findByRole("button",{name:"重新读取"});expect(state.execute).not.toHaveBeenCalled();
});
it("opens official management directly and shows actual release state rather than DRAFT",async()=>{
 state.platform.mockResolvedValue({items:[{id:recordId,kind:"official",product,stage:"DRAFT",createdAt:"2026-11-16T00:00:00Z"}],total:1});
 state.releases.mockResolvedValue({items:[{id:releaseId,active:true}],total:1});show(<SupplyApplicationsPage admin expectedUserId="actor" initialKind="official"/>);
 await screen.findByText("已发布");expect(screen.queryByText("草稿 · 待发布")).not.toBeInTheDocument();
 expect(screen.getByRole("tab",{name:"自营商品"})).toHaveAttribute("aria-selected","true");
 expect(screen.getByRole("link",{name:"查看并继续发布"})).toHaveAttribute("href",`/workbench/supply/operator/${recordId}`);
 expect(screen.queryByRole("button",{name:"已结束"})).not.toBeInTheDocument();
 expect(state.platform).toHaveBeenCalledWith({userId:"actor",organizationId:""},{kind:"official",ended:false,after:undefined},true,expect.any(AbortSignal));
});
