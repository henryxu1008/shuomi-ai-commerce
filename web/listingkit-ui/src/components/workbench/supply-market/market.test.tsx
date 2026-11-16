import {cleanup,fireEvent,render,screen,waitFor,within} from "@testing-library/react";
import {QueryClient,QueryClientProvider} from "@tanstack/react-query";
import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {SupplyMarketPage,SupplyProductPage} from "./market";
import {shippingRegions} from "@/test/fixtures/shipping-regions";
const calls=vi.hoisted(()=>({list:vi.fn(),read:vi.fn(),execute:vi.fn(),saved:null as null|{releaseId:string;collection:object}}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>({user:{id:"actor"},permissions:[]})}));
vi.mock("@/lib/api/supply-market",async original=>({...await original<object>(),listMarket:calls.list,readMarket:calls.read,listMarketRecords:async()=>{throw new Error("no platform access")}}));
vi.mock("./shared",async original=>({...await original<object>(),MarketBoundary:({children}:{children:(scope:{userId:string;organizationId:string})=>unknown})=>children({userId:"actor",organizationId:"org"}),useMarketCommands:()=>({execute:calls.execute,locked:false,saved:calls.saved,notice:null})}));
const sample={added:false,prices:[],merchant:{state:"unverified"},id:"11111111-1111-4111-8111-111111111111",revision:3,active:true,publishedAt:"2026-11-15T00:00:00Z",product:{title:"【合成示例】折叠收纳箱",description:"仅供流程试用",images:["https://example.com/sample.png"]},supply:{province:"浙江省",city:"杭州市",stock:60,minimumQuantity:2,leadDays:3}};
function show(node:React.ReactNode,seedRegions=true){const client=new QueryClient({defaultOptions:{queries:{retry:false}}});if(seedRegions)client.setQueryData(["region-catalog","actor","shipping","CN"],shippingRegions);return render(<QueryClientProvider client={client}>{node}</QueryClientProvider>)}
beforeEach(()=>{vi.clearAllMocks();calls.saved=null;calls.list.mockResolvedValue({items:[],total:0});});
afterEach(()=>{cleanup();vi.unstubAllGlobals()});
it.each([["official","硕米自营"],["selected","硕米优选"]] as const)("shows focused channel navigation and My Data for %s",async(channel,title)=>{
 calls.list.mockResolvedValue({items:[{...sample,channel}],total:1});show(<SupplyMarketPage channel={channel}/>);
 const nav=screen.getByRole("navigation",{name:"货源渠道"});
 expect(within(nav).getAllByRole("link")).toHaveLength(2);
 expect(within(nav).getByRole("link",{name:title})).toHaveAttribute("aria-current","page");
 expect(within(screen.getByRole("group",{name:"页面操作"})).getByRole("link",{name:"我的数据"})).toHaveAttribute("href","/workbench/data/mine");
 expect(screen.queryByRole("link",{name:"货盘集成"})).not.toBeInTheDocument();
 expect(await screen.findByRole("link",{name:"查看商品"})).toHaveAttribute("href",`/workbench/supply/products/${sample.id}`);
 expect(calls.list).toHaveBeenCalledWith({userId:"actor",organizationId:"org"},{kind:channel,keyword:"",after:undefined},expect.any(AbortSignal));
 expect(calls.execute).not.toHaveBeenCalled();
});
it.each([["official","硕米自营"],["selected","硕米优选"]] as const)("returns details to the original %s channel and selects only on click",async(channel,title)=>{
 calls.read.mockResolvedValue({...sample,channel});show(<SupplyProductPage id={sample.id}/>);
 expect(await screen.findByRole("link",{name:`返回${title}`})).toHaveAttribute("href",`/workbench/supply/${channel}`);
 expect(calls.execute).not.toHaveBeenCalled();
 fireEvent.click(screen.getByRole("button",{name:"加入我的数据"}));
 expect(calls.execute).toHaveBeenCalledWith({action:"select_release",id:sample.id,expectedRevision:sample.revision});
});
it("explains empty searches and clears the filter without changing channels",async()=>{
 show(<SupplyMarketPage channel="selected"/>);await screen.findByText("暂无商品");
 fireEvent.change(screen.getByRole("textbox",{name:"商品关键词"}),{target:{value:"不存在"}});fireEvent.click(screen.getByRole("button",{name:"搜索"}));
 await screen.findByText("没有找到匹配商品");fireEvent.click(screen.getByRole("button",{name:"清除筛选"}));
 await waitFor(()=>expect(screen.getByRole("textbox",{name:"商品关键词"})).toHaveValue(""));
 expect(within(screen.getByRole("navigation",{name:"货源渠道"})).getByRole("link",{name:"硕米优选"})).toHaveAttribute("aria-current","page");
});
it("does not mark another product as joined from a recovered selection receipt",async()=>{
 calls.saved={releaseId:"22222222-2222-4222-8222-222222222222",collection:{}};
 calls.read.mockResolvedValue({...sample,channel:"official"});show(<SupplyProductPage id={sample.id}/>);
 expect(await screen.findByRole("button",{name:"加入我的数据"})).toBeEnabled();
 expect(screen.queryByText("已加入我的数据")).not.toBeInTheDocument();
});
it.each(["official","selected"] as const)("shows price and directly adds the exact %s release",async channel=>{
 calls.list.mockResolvedValue({items:[{...sample,channel,prices:[{currency:"CNY",minimum:22,maximum:37}],added:false,merchant:{state:"unverified"}}],total:1});
 calls.execute.mockResolvedValue({releaseId:sample.id,collection:{itemId:sample.id}});
 show(<SupplyMarketPage channel={channel}/>);
 expect(await screen.findByText("¥22.00 – ¥37.00")).toBeInTheDocument();
 fireEvent.click(screen.getByRole("button",{name:"加入我的数据"}));
 await waitFor(()=>expect(calls.execute).toHaveBeenCalledWith({action:"select_release",id:sample.id,expectedRevision:sample.revision}));
 expect(await screen.findByRole("link",{name:"查看我的数据"})).toHaveAttribute("href","/workbench/data/mine");
});
it("shows confirmed membership and verified merchant name without another add button",async()=>{
 calls.list.mockResolvedValue({items:[{...sample,channel:"selected",prices:[{currency:"CNY",minimum:22,maximum:22}],added:true,merchant:{state:"verified",name:"已认证示例企业"}}],total:1});
 show(<SupplyMarketPage channel="selected"/>);
 expect(await screen.findByText("已认证示例企业")).toBeInTheDocument();
 expect(screen.getByRole("img",{name:"已加入我的数据"})).toHaveAttribute("title","已加入我的数据");
 expect(screen.getByRole("link",{name:"查看我的数据"})).toBeInTheDocument();
 expect(screen.queryByRole("button",{name:"加入我的数据"})).not.toBeInTheDocument();
 expect(calls.execute).not.toHaveBeenCalled();
});
it("does not present an unknown membership or unavailable merchant as a confirmed fact",async()=>{
 calls.list.mockResolvedValue({items:[{...sample,channel:"selected",prices:[],added:null,merchant:{state:"unavailable"}}],total:1});
 show(<SupplyMarketPage channel="selected"/>);
 expect(await screen.findByRole("button",{name:"暂不可加入"})).toBeDisabled();
 expect(screen.getByText("商家信息暂不可用")).toBeInTheDocument();
 expect(screen.getByText("价格待确认")).toBeInTheDocument();
 expect(screen.queryByRole("img",{name:"已加入我的数据"})).not.toBeInTheDocument();
});
it("applies supply filters together, preserves zero days and resets paging when cleared",async()=>{
 calls.list.mockImplementation(async(_scope,q)=>({items:[],total:2,...(!q.after?{nextCursor:sample.id}:{})}));
 show(<SupplyMarketPage channel="selected"/>);
 await screen.findByText("暂无商品");fireEvent.click(screen.getByRole("button",{name:"下一页"}));
 await waitFor(()=>expect(calls.list.mock.calls.at(-1)![1].after).toBe(sample.id));
 fireEvent.change(screen.getByRole("combobox",{name:"发货省份"}),{target:{value:"浙江省"}});
 fireEvent.change(screen.getByRole("combobox",{name:"发货城市"}),{target:{value:"杭州市"}});
 fireEvent.change(screen.getByRole("combobox",{name:"发货省份"}),{target:{value:"广东省"}});
 expect(screen.getByRole("combobox",{name:"发货城市"})).toHaveValue("");
 expect(screen.getByRole("option",{name:"深圳市"})).toBeInTheDocument();
 expect(screen.queryByRole("option",{name:"杭州市"})).not.toBeInTheDocument();
 fireEvent.change(screen.getByRole("combobox",{name:"发货省份"}),{target:{value:"浙江省"}});
 fireEvent.change(screen.getByRole("combobox",{name:"发货城市"}),{target:{value:"杭州市"}});
 fireEvent.change(screen.getByRole("spinbutton",{name:"最低价格"}),{target:{value:"0"}});
 fireEvent.change(screen.getByRole("spinbutton",{name:"最高价格"}),{target:{value:"50"}});
 fireEvent.change(screen.getByRole("spinbutton",{name:"起订量不超过（件）"}),{target:{value:"2"}});
 fireEvent.change(screen.getByRole("spinbutton",{name:"供货周期不超过（天）"}),{target:{value:"0"}});
 fireEvent.click(screen.getByRole("button",{name:"搜索"}));
 await waitFor(()=>expect(calls.list.mock.calls.at(-1)![1]).toEqual({kind:"selected",keyword:"",province:"浙江省",city:"杭州市",minPrice:0,maxPrice:50,priceCurrency:"CNY",maxMOQ:2,maxLeadDays:0,after:undefined}));
 fireEvent.click(screen.getByRole("button",{name:"清除筛选"}));
 await waitFor(()=>expect(calls.list.mock.calls.at(-1)![1]).toEqual({kind:"selected",keyword:"",after:undefined}));
 expect(screen.getByRole("combobox",{name:"发货省份"})).toHaveValue("");
 expect(screen.getByRole("combobox",{name:"发货城市"})).toHaveValue("");
 expect(screen.getByRole("combobox",{name:"发货城市"})).toBeDisabled();
 expect(screen.getByRole("spinbutton",{name:"最低价格"})).toHaveValue(null);
});
it("highlights variant prices as ranges within each currency",async()=>{
 calls.read.mockResolvedValue({...sample,channel:"official",product:{...sample.product,variants:[{currency:"CNY",price:22},{currency:"CNY",price:37},{currency:"USD",price:9},{currency:"CNY"}]}});
 show(<SupplyProductPage id={sample.id}/>);
 const prices=await screen.findByRole("group",{name:"商品价格"});
 expect(within(prices).getByText("¥22.00 – ¥37.00")).toBeInTheDocument();
 expect(within(prices).getByText("US$9.00")).toBeInTheDocument();
 expect(calls.execute).not.toHaveBeenCalled();
});
it("keeps a zero price and displays an unfamiliar currency without guessing",async()=>{
 calls.read.mockResolvedValue({...sample,channel:"selected",product:{...sample.product,variants:[{currency:"CNY",price:0},{currency:"积分",price:12}]}});
 show(<SupplyProductPage id={sample.id}/>);
 const prices=await screen.findByRole("group",{name:"商品价格"});
 expect(within(prices).getByText("¥0.00")).toBeInTheDocument();
 expect(within(prices).getByText("积分 12.00")).toBeInTheDocument();
});
it.each(["500件起订，价格另议",undefined])("shows the original note or no-price text without interpreting numbers: %s",async(priceNote)=>{
 calls.read.mockResolvedValue({...sample,channel:"official",supply:{...sample.supply,priceNote}});
 show(<SupplyProductPage id={sample.id}/>);
 const prices=await screen.findByRole("group",{name:"商品价格"});
 expect(within(prices).getByText(priceNote??"暂未提供")).toBeInTheDocument();
});

it("keeps keyword search available when the shipping catalog cannot load",async()=>{
 vi.stubGlobal("fetch",vi.fn().mockRejectedValue(new Error("region directory unavailable")));
 show(<SupplyMarketPage channel="official"/>,false);
 expect(await screen.findByText("发货地选项暂时无法读取")).toBeVisible();
 expect(screen.getByRole("combobox",{name:"发货省份"})).toBeDisabled();
 fireEvent.change(screen.getByRole("textbox",{name:"商品关键词"}),{target:{value:"收纳箱"}});
 fireEvent.click(screen.getByRole("button",{name:"搜索"}));
 await waitFor(()=>expect(calls.list.mock.calls.at(-1)![1]).toEqual({kind:"official",keyword:"收纳箱",after:undefined}));
});
