import {afterEach,beforeEach,expect,it,vi} from "vitest";
import {cleanup,fireEvent,render,screen,waitFor,within} from "@testing-library/react";
import {QueryClient,QueryClientProvider} from "@tanstack/react-query";
import {BusinessTaskPage} from "./task-page";
import {ConfigurationError} from "@/lib/api/agent-configuration";

const f=vi.hoisted(()=>({request:vi.fn(),tasks:vi.fn(),context:{} as Record<string,unknown>}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>f.context}));
vi.mock("@/lib/api/agent-configuration",async original=>({...await original<typeof import("@/lib/api/agent-configuration")>(),configurationRequest:f.request}));
vi.mock("@/lib/api/ai-workbench",async original=>({...await original<typeof import("@/lib/api/ai-workbench")>(),requestAIWorkbench:f.tasks}));
const runId="550e8400-e29b-41d4-a716-446655440001", objectId="550e8400-e29b-41d4-a716-446655440002", key="550e8400-e29b-41d4-a716-446655440003";
const time="2026-10-18T01:00:00Z";
function entry(id="product.content.agent",name="商品内容与本地化"){return {agent:{agentId:id,activation:"DISABLED",revision:"1",activationEpoch:"1",defaultTemplate:null,updatedAt:time},name,description:"",definitionVersion:"v1.0.0",parameterSchema:id==="product.image.agent"?"image-config-v1":"content-config-v1",canUse:false,canConfigure:false,canReadRuns:true,capabilities:[]};}
const content={runId,operationId:objectId,contextKind:"own_product",requestKey:key,productKey:"我的水杯",targetPlatform:"product",phase:"running",revision:"1",startedAt:time};
const image={runId,contextKind:"supply",contextId:objectId,status:"completed",targetPlatform:"product",createdAt:time};
let client:QueryClient;
const tree=()=> <QueryClientProvider client={client}><BusinessTaskPage {...{nativeRunsAvailable:true}} /></QueryClientProvider>;
beforeEach(()=>{
 client=new QueryClient({defaultOptions:{queries:{retry:false}}});
 f.context={user:{id:"owner"},effectiveOrganization:{id:"org"},roles:[],permissions:["workbench.agent.read","workbench.agent.use","local_agent.write","listingkit.image_agent.read"],aiWorkbenchAvailable:false};
 f.tasks.mockReset();f.request.mockReset().mockImplementation(async(_scope,path:string,schema)=>schema.parse(path.startsWith("market")?{items:[entry(),entry("product.image.agent","商品图片")],nextCursor:""}:path.startsWith("product.content")?{items:[content],nextCursor:""}:{items:[image],nextCursor:""}));
});
afterEach(()=>{cleanup();client.clear();});
it("reads content and image runs with Chat disabled and canUse false",async()=>{
 render(tree());
 expect(await screen.findByRole("button",{name:/商品内容与本地化.*我的水杯/})).toBeVisible();
 expect(await screen.findByRole("button",{name:/商品图片.*图片运行已完成/})).toBeVisible();
 expect(screen.getByRole("link",{name:"执行中"})).toBeVisible();
 expect(f.tasks).not.toHaveBeenCalled();
 expect(f.request.mock.calls.every(c=>!c[3]?.method||c[3].method==="GET")).toBe(true);
});
it("keeps identical run IDs from different owners and opens the exact own-product context",async()=>{
 render(tree());fireEvent.click(await screen.findByRole("button",{name:/商品内容与本地化.*我的水杯/}));
 const detail=screen.getByRole("region",{name:"任务详情"});
 expect(within(detail).getByRole("link",{name:"打开原业务工作区"})).toHaveAttribute("href",expect.stringContaining(`/workbench/data/mine/own/${objectId}/optimize?`));
 expect(screen.getByRole("button",{name:/商品图片.*图片运行已完成/})).toBeVisible();
});
it("does not route an unrecognized source to acquisition",async()=>{
 f.request.mockImplementation(async(_s,path:string,schema)=>schema.parse(path.startsWith("market")?{items:[entry()],nextCursor:""}:{items:[{...content,contextKind:"future_source"}],nextCursor:""}));
 render(tree());fireEvent.click(await screen.findByRole("button",{name:/商品内容与本地化.*我的水杯/}));
 expect(screen.getByRole("link",{name:"打开原业务工作区"})).toHaveAttribute("href","/workbench/agents/mine/product.content.agent");
 expect(screen.getByText(/该上下文暂无直达入口/)).toBeVisible();
});
it("keeps successful sources visible when another source fails",async()=>{
 const original=f.request.getMockImplementation()!;
 f.request.mockImplementation((scope,path:string,schema,init)=>path.startsWith("product.image")?Promise.reject(new ConfigurationError("DEPENDENCY_UNAVAILABLE",503)):original(scope,path,schema,init));
 render(tree());
 expect(await screen.findByRole("button",{name:/商品内容与本地化.*我的水杯/})).toBeVisible();
 expect(await screen.findByText(/商品图片：/)).toBeVisible();
 expect(screen.getAllByText("—")).toHaveLength(3);
});
it("never requests runs denied by the authoritative catalog",async()=>{
 f.request.mockResolvedValue({items:[{...entry(),canReadRuns:false}],nextCursor:""});
 render(tree());await screen.findByText("当前没有可读取的智能体运行");
 expect(f.request).toHaveBeenCalledTimes(1);
});
it("removes old native details when organization or permission identity changes",async()=>{
 const view=render(tree());fireEvent.click(await screen.findByRole("button",{name:/商品内容与本地化.*我的水杯/}));
 f.context={...f.context,effectiveOrganization:{id:"other"}};
 f.request.mockImplementation(()=>new Promise(()=>{}));view.rerender(tree());
 await waitFor(()=>expect(screen.queryByText("运行编号")).not.toBeInTheDocument());
 expect(screen.queryByText("我的水杯")).not.toBeInTheDocument();
});
it("paginates only the selected source and continues an authorized empty page",async()=>{
 const cursor="550e8400-e29b-41d4-a716-446655440099";
 const original=f.request.getMockImplementation()!;
 f.request.mockImplementation((scope,path:string,schema,init)=>path.startsWith("product.content")?Promise.resolve(schema.parse(path.includes("cursor=")?{items:[content],nextCursor:""}:{items:[],nextCursor:cursor})):original(scope,path,schema,init));
 render(tree());
 fireEvent.click(await screen.findByRole("button",{name:"加载更多商品内容与本地化"}));
 expect(await screen.findByRole("button",{name:/商品内容与本地化.*我的水杯/})).toBeVisible();
 expect(f.request.mock.calls.filter(c=>c[1].startsWith("product.image"))).toHaveLength(1);
 expect(f.request.mock.calls.some(c=>c[1]===`product.content.agent/recent-runs?pageSize=20&cursor=${cursor}`)).toBe(true);
});
it("hides all previously loaded rows of a source when a later page denies access",async()=>{
 const cursor="550e8400-e29b-41d4-a716-446655440099";
 const original=f.request.getMockImplementation()!;
 f.request.mockImplementation((scope,path:string,schema,init)=>path.startsWith("product.content")?path.includes("cursor=")?Promise.reject(new ConfigurationError("FORBIDDEN",403)):Promise.resolve(schema.parse({items:[content],nextCursor:cursor})):original(scope,path,schema,init));
 render(tree());fireEvent.click(await screen.findByRole("button",{name:/商品内容与本地化.*我的水杯/}));
 fireEvent.click(screen.getByRole("button",{name:"加载更多商品内容与本地化"}));
 await screen.findByText(/商品内容与本地化：/);
 expect(screen.queryByRole("button",{name:/我的水杯/})).not.toBeInTheDocument();
 expect(screen.queryByText("运行编号")).not.toBeInTheDocument();
 expect(screen.getByRole("button",{name:/商品图片.*图片运行已完成/})).toBeVisible();
});
it("does not label generated text as an unresolved review or an applied product",async()=>{
 const original=f.request.getMockImplementation()!;
 f.request.mockImplementation((scope,path:string,schema,init)=>path.startsWith("product.content")?Promise.resolve(schema.parse({items:[{...content,phase:"human_review_required"}],nextCursor:""})):original(scope,path,schema,init));
 render(tree());expect(await screen.findByRole("button",{name:/商品内容与本地化.*建议已生成/})).toBeVisible();
 expect(screen.queryByRole("button",{name:/商品内容与本地化.*待确认/})).not.toBeInTheDocument();
 expect(screen.getAllByText("—")).toHaveLength(3);
});
it("does not attempt discovery without the existing use permission",async()=>{
 f.context={...f.context,permissions:["workbench.agent.read"]};render(tree());
 expect(screen.getByText("暂未启用")).toBeVisible();expect(f.request).not.toHaveBeenCalled();
});
it("reauthorizes every loaded page before restoring a source after an access denial",async()=>{
 const cursor="550e8400-e29b-41d4-a716-446655440099";
 const original=f.request.getMockImplementation()!;let denied=true,firstReads=0;
 f.request.mockImplementation((scope,path:string,schema,init)=>{
  if(!path.startsWith("product.content"))return original(scope,path,schema,init);
  if(path.includes("cursor="))return denied?Promise.reject(new ConfigurationError("FORBIDDEN",403)):Promise.resolve(schema.parse({items:[],nextCursor:""}));
  firstReads++;return Promise.resolve(schema.parse({items:firstReads===1?[content]:[],nextCursor:cursor}));
 });
 render(tree());await screen.findByRole("button",{name:/商品内容与本地化.*我的水杯/});
 fireEvent.click(screen.getByRole("button",{name:"加载更多商品内容与本地化"}));
 const retry=await screen.findByRole("button",{name:"重试商品内容与本地化"});denied=false;fireEvent.click(retry);
 await waitFor(()=>expect(screen.queryByRole("button",{name:"重试商品内容与本地化"})).not.toBeInTheDocument());
 expect(firstReads).toBe(2);expect(screen.queryByRole("button",{name:/我的水杯/})).not.toBeInTheDocument();
});
