import {act,cleanup,fireEvent,render,screen,waitFor} from "@testing-library/react";
import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {SupplyAPIError,type SupplyIntent} from "@/lib/api/supply-chain";
import {CompletionStructuredPage} from "./completion-structured-page";
import StructuredRoute from "@/app/workbench/supply/mine/[stage]/[sourceId]/structure/page";

const f=vi.hoisted(()=>({context:{user:{id:"actor"},effectiveOrganization:{id:"org"},permissions:["workbench.supply.read","workbench.supply.manage"],supplyIntentReady:true,pendingSupplyIntent:null as SupplyIntent|null,setPendingSupplyIntent:vi.fn<(v:SupplyIntent|null)=>boolean>(),registerOrganizationSwitchGuard:()=>()=>{}},source:vi.fn(),head:vi.fn(),save:vi.fn(),verify:vi.fn()}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>f.context}));
vi.mock("next/server",()=>({connection:async()=>{}}));
vi.mock("next/navigation",()=>({notFound:()=>{throw new Error("NOT_FOUND")}}));
vi.mock("@/lib/api/supply-chain",async original=>({...await original<object>(),readSupplySource:f.source,readSupplyCompletionDraft:f.head,supplyCommand:f.save,readSupplyCommand:f.verify}));
const id="11111111-1111-4111-8111-111111111111",batch="22222222-2222-4222-8222-222222222222",other="33333333-3333-4333-8333-333333333333";
const source={id,preparationId:batch,collectionItemId:batch,collectionRevision:1,source:{kind:"own",version:"1",productKey:"fixed",publicationId:batch}};
const detail={source,images:[],product:{title:"来源标题",description:"来源描述",attributes:[{name:"材质",value:"棉"}],variants:[{sku:"",title:"第一款",price:{currency:"CNY",amount:0}},{sku:"",title:"第二款",price:{currency:"CNY",amount:10},stock:3}]}};
const content={title:"已保存标题",description:"已保存描述",structured:{attributes:[],variants:[{sourceIndex:0,price:null,stock:0},{sourceIndex:1,price:{currency:"CNY",amount:10},stock:null}]}};
const head={id,draftId:batch,revision:3,source,content,status:"pending_completion",createdAt:"2026-10-15T00:00:00Z"};
const props={sourceId:id,preparationId:batch,itemId:batch};
beforeEach(()=>{vi.clearAllMocks();f.context.effectiveOrganization.id="org";f.context.permissions=["workbench.supply.read","workbench.supply.manage"];f.context.pendingSupplyIntent=null;f.context.supplyIntentReady=true;f.context.setPendingSupplyIntent.mockImplementation(v=>{f.context.pendingSupplyIntent=v;return true});f.source.mockResolvedValue(detail);f.head.mockResolvedValue(head);f.save.mockResolvedValue({record:{...head,revision:4},replayed:false})});
afterEach(cleanup);
it("guards the independent route and keeps only the approved navigation parameters",async()=>{
 vi.stubEnv("LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED","true");vi.stubEnv("LISTINGKIT_SUPPLY_COMPLETION_DRAFTS_ENABLED","true");
 const route=(params={stage:"missing",sourceId:id},query:Record<string,string|string[]|undefined>={preparation:batch,item:batch})=>StructuredRoute({params:Promise.resolve(params),searchParams:Promise.resolve(query)});
 expect((await route()).props).toEqual(props);
 await expect(route({stage:"adapted",sourceId:id})).rejects.toThrow("NOT_FOUND");await expect(route({stage:"missing",sourceId:"wrong"})).rejects.toThrow("NOT_FOUND");
 for(const query of [{item:[batch,batch]},{preparation:"wrong"},{returnTo:"https://example.test"}])await expect(route(undefined,query)).rejects.toThrow("NOT_FOUND");
 vi.stubEnv("LISTINGKIT_SUPPLY_COMPLETION_DRAFTS_ENABLED","false");await expect(route()).rejects.toThrow("NOT_FOUND");vi.unstubAllEnvs();
});
it("keeps the original pending save locked and verifies its existing key",async()=>{
 f.save.mockRejectedValueOnce(new SupplyAPIError("OUTCOME_UNKNOWN",503));render(<CompletionStructuredPage {...props}/>);await screen.findByRole("heading",{name:"候选商品属性"});fireEvent.click(save());await screen.findByText("结果待核实");expect(save()).toBeDisabled();
 const pending=f.context.pendingSupplyIntent as SupplyIntent|null;expect(pending).not.toBeNull();f.head.mockResolvedValue({...head,revision:4});f.verify.mockResolvedValue({record:{...head,revision:4},replayed:true});
 fireEvent.click(screen.getByRole("button",{name:"核实原操作"}));await waitFor(()=>expect(f.verify).toHaveBeenCalledWith(expect.objectContaining({key:pending!.key,command:pending!.command}),expect.any(AbortSignal)));await waitFor(()=>expect(save()).toBeEnabled());expect(f.save).toHaveBeenCalledTimes(1);
});
const save=()=>screen.getByRole("button",{name:"保存商品属性和 SKU"});
it("reads exact saved empty/zero/null values and preserves all return context",async()=>{
 render(<CompletionStructuredPage {...props}/>);await screen.findByRole("heading",{name:"候选商品属性"});
 expect(screen.queryByRole("dialog")).not.toBeInTheDocument();expect(screen.queryByRole("textbox",{name:"属性名称 1"})).not.toBeInTheDocument();
 expect(screen.getByRole("spinbutton",{name:"候选价格 SKU 1"})).toHaveValue(null);expect(screen.getByRole("spinbutton",{name:"候选库存 SKU 1"})).toHaveValue(0);expect(screen.getByRole("spinbutton",{name:"候选库存 SKU 2"})).toHaveValue(null);
 expect(screen.getByRole("link",{name:"返回草稿"})).toHaveAttribute("href",`/workbench/supply/mine/missing/${id}?preparation=${batch}&item=${batch}`);
 expect(screen.getByRole("link",{name:"返回商品"})).toHaveAttribute("href",`/workbench/data/mine/own/${batch}`);expect(screen.getByRole("link",{name:"返回待补全列表"})).toHaveAttribute("href",`/workbench/supply/mine/missing?preparation=${batch}`);
 fireEvent.click(save());await waitFor(()=>expect(f.save).toHaveBeenCalledWith(expect.objectContaining({command:{sourceId:id,expectedRevision:3,content}}),expect.any(AbortSignal)));
});
it("prefills only the fixed source as unsaved and keeps own zero stock explicit",async()=>{
 f.head.mockRejectedValue(new SupplyAPIError("NOT_FOUND",404));render(<CompletionStructuredPage {...props}/>);
 expect(await screen.findByRole("textbox",{name:"属性名称 1"})).toHaveValue("材质");expect(screen.getByText(/来源预填，尚未保存结构化候选资料/)).toBeVisible();
 expect(screen.getByRole("spinbutton",{name:"候选价格 SKU 1"})).toHaveValue(0);expect(screen.getByRole("spinbutton",{name:"候选库存 SKU 1"})).toHaveValue(0);
});
it("adds and clears candidate fields without changing SKU identity or the saved text",async()=>{
 render(<CompletionStructuredPage {...props}/>);await screen.findByRole("heading",{name:"候选商品属性"});fireEvent.click(screen.getByRole("button",{name:"添加属性"}));
 fireEvent.change(screen.getByRole("textbox",{name:"属性名称 1"}),{target:{value:"材质"}});fireEvent.change(screen.getByRole("textbox",{name:"属性值 1"}),{target:{value:"混纺"}});
 fireEvent.change(screen.getByRole("spinbutton",{name:"候选价格 SKU 2"}),{target:{value:""}});fireEvent.change(screen.getByRole("spinbutton",{name:"候选库存 SKU 2"}),{target:{value:"0"}});fireEvent.click(save());
 await waitFor(()=>expect(f.save).toHaveBeenCalledWith(expect.objectContaining({command:{sourceId:id,expectedRevision:3,content:{...content,structured:{attributes:[{name:"材质",value:"混纺"}],variants:[content.structured.variants[0],{sourceIndex:1,price:null,stock:0}]}}}}),expect.any(AbortSignal)));
});
it("locks a CAS conflict until explicit reload and new confirmation",async()=>{
 f.save.mockRejectedValueOnce(new SupplyAPIError("REVISION_CONFLICT",409));render(<CompletionStructuredPage {...props}/>);await screen.findByRole("heading",{name:"候选商品属性"});fireEvent.click(save());await screen.findByText(/当前版本已变化/);expect(save()).toBeDisabled();
 f.head.mockResolvedValue({...head,revision:4,content:{...content,title:"另一页新标题"}});fireEvent.click(screen.getByRole("button",{name:"重新读取保存版本"}));await waitFor(()=>expect(save()).toBeEnabled());fireEvent.click(save());
 await waitFor(()=>expect(f.save).toHaveBeenLastCalledWith(expect.objectContaining({command:expect.objectContaining({expectedRevision:4,content:expect.objectContaining({title:"另一页新标题"})})}),expect.any(AbortSignal)));
});
it("refuses wrong navigation, readonly edits, and late responses after scope changes",async()=>{
 const wrong=render(<CompletionStructuredPage {...props} itemId={other}/>);await screen.findByText("结构化草稿读取失败");expect(screen.queryByRole("spinbutton")).not.toBeInTheDocument();wrong.unmount();
 f.context.permissions=["workbench.supply.read"];const ro=render(<CompletionStructuredPage {...props}/>);await screen.findByRole("heading",{name:"候选商品属性"});expect(save()).toBeDisabled();expect(screen.getByRole("spinbutton",{name:"候选库存 SKU 1"})).toBeDisabled();ro.unmount();
 f.context.permissions=["workbench.supply.read","workbench.supply.manage"];let resolve!:(value:unknown)=>void;f.source.mockReturnValueOnce(new Promise(r=>{resolve=r}));const view=render(<CompletionStructuredPage {...props}/>);f.context.effectiveOrganization.id="changed";f.source.mockReturnValue(new Promise(()=>{}));view.rerender(<CompletionStructuredPage {...props}/>);await act(async()=>resolve(detail));expect(screen.queryByRole("spinbutton")).not.toBeInTheDocument();
});

it("retains the exact rich description and format when saving only attributes and SKUs",async()=>{
 const rich={...head,content:{...content,description:"<p><strong>保留排版</strong></p>",descriptionFormat:"html-v1" as const}};
 f.head.mockResolvedValue(rich);render(<CompletionStructuredPage {...props}/>);await screen.findByRole("heading",{name:"候选商品属性"});
 fireEvent.click(save());await waitFor(()=>expect(f.save).toHaveBeenCalledWith(expect.objectContaining({command:{sourceId:id,expectedRevision:3,content:rich.content}}),expect.any(AbortSignal)));
});
