import {afterEach,expect,it,vi} from "vitest";
import {cleanup,render,screen,waitFor} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {QueryClient,QueryClientProvider} from "@tanstack/react-query";
import {ModelRoutePricingPage,RoutePricingEditor} from "./model-route-pricing";
const state=vi.hoisted(()=>({context:{permissions:["listingkit.platform_admin","listingkit.admin.write"],roles:[],user:{id:"maintainer"},effectiveOrganization:{id:"org-a",name:"企业甲"},registerOrganizationSwitchGuard:()=>()=>{},isLoading:false,isSwitching:false,error:null,blockingError:null,selectionRequired:false}}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>state.context}));
const scope={userId:"maintainer",organizationId:"org-a"};
const payload={bindingId:"image-a",credentialVersion:"credential-a",rates:{pointsPerImage:"8"},image:{provider:"grsai",model:"gpt-image-2.5",protocol:"grsai-json-sync-v1",endpointDigest:"digest-a",credentialReference:"db:1:1:image-a",configurationVersion:"r1",priceVersion:"r1",points:8}};
const revision=(id="r1",key="old")=>({id,scope:{organizationId:"org-a",capability:"imageagent.generation"},parent:"NONE",schemaVersion:1,payload,digest:"digest",actorId:"maintainer",publishedAt:"2026-10-19T00:00:00Z",key});
const bindings={bindings:[{id:"image-a",credentialVersion:"credential-a",provider:"grsai",model:"gpt-image-2.5",protocol:"grsai-json-sync-v1",maximumPointsPerImage:20}]};
const clients:QueryClient[]=[];
function mount(page=false){const client=new QueryClient({defaultOptions:{queries:{retry:false}}});clients.push(client);return render(<QueryClientProvider client={client}>{page?<ModelRoutePricingPage expectedUserId="maintainer"/>:<RoutePricingEditor scope={scope} family="image" capability="imageagent.generation"/>}</QueryClientProvider>)}
function fixtures(publish:(init:RequestInit)=>Promise<Response>,current="r1"){
 return vi.fn(async(url:string,init:RequestInit={})=>{
  if(url.endsWith("/bindings"))return Response.json(bindings);
  if(url.endsWith("/history"))return Response.json({revisions:[revision()],nextCursor:""});
  if(url.endsWith("/validate")){const input=JSON.parse(String(init.body));return Response.json({payload:{...payload,rates:input.rates},expectedRevision:input.expectedRevision,digest:"preview"})}
  if(url.endsWith("/publish"))return publish(init);
  if(url.includes("/receipts/"))return Response.json({receipt:revision("r2",url.split("/").pop())});
  return Response.json({status:"READY",revision:revision(current)});
 });
}
afterEach(()=>{cleanup();clients.splice(0).forEach(c=>c.clear());sessionStorage.clear();vi.unstubAllGlobals();state.context.permissions=["listingkit.platform_admin","listingkit.admin.write"]});
it("requires preview before publish and distinguishes receipt from actual current",async()=>{
 const user=userEvent.setup();const fetch=fixtures(async init=>Response.json({receipt:revision("r2",new Headers(init.headers).get("Idempotency-Key")!)}),"r3");vi.stubGlobal("fetch",fetch);mount();
 const rate=await screen.findByLabelText(/每张图片 AI 点数/);await user.clear(rate);await user.type(rate,"10");
 expect(screen.queryByRole("button",{name:"确认发布"})).not.toBeInTheDocument();await user.click(screen.getByRole("button",{name:"校验配置并预览"}));await screen.findByRole("heading",{name:"确认发布这个用途的新配置"});
 expect(fetch.mock.calls.filter(([u])=>u.endsWith("/publish"))).toHaveLength(0);await user.click(screen.getByRole("button",{name:"确认发布"}));await screen.findByRole("heading",{name:"发布回执"});expect(screen.getByText("本次已发布，当前已有更新。旧回执不代表当前配置。")).toBeVisible();expect(fetch.mock.calls.filter(([u])=>u.endsWith("/publish"))).toHaveLength(1);
});
it("persists original key on unknown, restores after remount and only queries its receipt",async()=>{
 const user=userEvent.setup(),fetch=fixtures(async()=>{throw new Error("lost")});vi.stubGlobal("fetch",fetch);const view=mount();await screen.findByLabelText(/每张图片 AI 点数/);
 await user.click(screen.getByRole("button",{name:"校验配置并预览"}));await screen.findByRole("heading",{name:"确认发布这个用途的新配置"});await user.click(screen.getByRole("button",{name:"确认发布"}));await screen.findByRole("heading",{name:"发布结果待核实"});
 const key=new Headers(fetch.mock.calls.find(([u])=>u.endsWith("/publish"))![1]!.headers).get("Idempotency-Key");expect(key).toBeTruthy();expect(screen.getByRole("button",{name:"确认发布"})).toBeDisabled();view.unmount();mount();await screen.findByRole("heading",{name:"发布结果待核实"});await user.click(screen.getByRole("button",{name:"查询本次发布结果"}));await screen.findByRole("heading",{name:"发布回执"});expect(fetch.mock.calls.some(([u])=>u.endsWith("/receipts/"+key))).toBe(true);expect(fetch.mock.calls.filter(([u])=>u.endsWith("/publish"))).toHaveLength(1);
});
it("denies ordinary enterprise admins before management reads",async()=>{state.context.permissions=["listingkit.admin.write"];const fetch=vi.fn();vi.stubGlobal("fetch",fetch);mount(true);await screen.findByText("无平台维护权限或当前企业不可用");await waitFor(()=>expect(fetch).not.toHaveBeenCalled())});
it("rejects zero rate without sending validation and keeps edits",async()=>{const user=userEvent.setup(),fetch=fixtures(async()=>Response.json({}));vi.stubGlobal("fetch",fetch);mount();const rate=await screen.findByLabelText(/每张图片 AI 点数/);await user.clear(rate);await user.type(rate,"0");await user.click(screen.getByRole("button",{name:"校验配置并预览"}));expect(await screen.findByRole("alert")).toHaveTextContent("费率");expect(rate).toHaveValue("0");expect(fetch.mock.calls.some(([u])=>u.endsWith("/validate"))).toBe(false)});
it("keeps edits while a failed refresh clears authority to publish stale current",async()=>{
 const user=userEvent.setup(),fetch=fixtures(async()=>Response.json({}));vi.stubGlobal("fetch",fetch);mount();const rate=await screen.findByLabelText(/每张图片 AI 点数/);await user.clear(rate);await user.type(rate,"12");
 fetch.mockImplementation(async()=>Response.json({code:"UNAVAILABLE"},{status:503}));await user.click(screen.getByRole("button",{name:"读取当前与历史"}));await screen.findByText(/当前配置读取失败。编辑与原回执保留/);expect(rate).toHaveValue("12");expect(screen.getByRole("button",{name:"校验配置并预览"})).toBeDisabled();expect(screen.queryByRole("heading",{name:"当前已发布配置"})).not.toBeInTheDocument();
});
it("still queries the pending original receipt when current reads fail after remount",async()=>{
 const user=userEvent.setup();sessionStorage.setItem(`resource.pending:${JSON.stringify([scope.userId,scope.organizationId,"model-route-pricing","image","imageagent.generation"])}`,JSON.stringify({key:"86d5eace-722b-4d5f-8875-8532c3d0af2f",input:{expectedRevision:"NONE",bindingId:"image-a",credentialVersion:"credential-a",rates:{pointsPerImage:"8"}}}));
 const fetch=vi.fn(async(url:string)=>url.includes("/receipts/")?Response.json({receipt:revision("r2","86d5eace-722b-4d5f-8875-8532c3d0af2f")}):Response.json({code:"UNAVAILABLE"},{status:503}));vi.stubGlobal("fetch",fetch);mount();await screen.findByText(/读取失败不代表发布失败/);await user.click(screen.getByRole("button",{name:"查询本次发布结果"}));await screen.findByText(/本次发布已记录：r2。实际当前配置仍无法读取/);expect(fetch.mock.calls.filter(([url])=>url.endsWith("/publish"))).toHaveLength(0);
});
it("separates text currency cost from token points through preview and receipt",async()=>{
 const user=userEvent.setup();const p={bindingId:"text-a",credentialVersion:"credential-a",rates:{inputCost:"1.000000",outputCost:"2.000000",inputPoints:"100",outputPoints:"200"},text:{ClientName:"text-a",ProviderID:"fixture",AdapterKind:"openai-compatible",ModelID:"model-a",EndpointIdentityDigest:"endpoint",CredentialVersion:"credential-a",RoutingPolicyVersion:"r1",AdapterPolicyVersion:"adapter-v1",PromptVersion:"content-v1",OutputSchemaVersion:"schema-v1",UsageMappingVersion:"usage-v1",CostPricingVersion:"r1",PointTariff:{priceVersion:"r1",inputPointsPerMillionTokens:100,outputPointsPerMillionTokens:200},Currency:"CNY",InputMicrosPerMillion:1000000,OutputMicrosPerMillion:2000000,MaximumPromptTokens:100,MaximumCompletionTokens:100,MaximumInputBytes:4096,MaximumOutputBytes:4096,DeadlineBound:1000000000}};
 const row={...revision(),scope:{organizationId:"org-a",capability:"product.content.decision"},payload:p};
 const fetch=vi.fn(async(url:string,init:RequestInit={})=>{
  if(url.endsWith("/bindings"))return Response.json({bindings:[{id:"text-a",credentialVersion:"credential-a",provider:"fixture",model:"model-a",protocol:"openai-compatible",usageMappingVersion:"usage-v1",currency:"CNY",maximumCostMicros:10000,maximumPromptTokens:100,maximumCompletionTokens:100,maximumInputBytes:4096,maximumOutputBytes:4096,deadlineSeconds:1}]});
  if(url.endsWith("/history"))return Response.json({revisions:[row],nextCursor:""});
  if(url.endsWith("/validate")){const input=JSON.parse(String(init.body));return Response.json({payload:{...p,rates:input.rates},expectedRevision:input.expectedRevision,digest:"preview"})}
  if(url.endsWith("/publish"))return Response.json({receipt:{...row,id:"r2",key:new Headers(init.headers).get("Idempotency-Key")}});
  return Response.json({status:"READY",revision:row});
 });vi.stubGlobal("fetch",fetch);const client=new QueryClient();clients.push(client);render(<QueryClientProvider client={client}><RoutePricingEditor scope={scope} family="text" capability="product.content.decision"/></QueryClientProvider>);
 const cost=await screen.findByLabelText(/输入估算成本/);expect(screen.getByText(/用量映射 usage-v1/)).toBeVisible();await user.clear(cost);await user.type(cost,"0.000001");await user.click(screen.getByRole("button",{name:"校验配置并预览"}));await screen.findByRole("heading",{name:"确认发布这个用途的新配置"});await user.click(screen.getByRole("button",{name:"确认发布"}));await screen.findByRole("heading",{name:"发布回执"});
 const posted=JSON.parse(String(fetch.mock.calls.find(([u])=>u.endsWith("/publish"))![1]!.body));expect(posted.rates).toEqual({...p.rates,inputCost:"0.000001"});expect(screen.queryByText(/供应商成本：尚未接入/)).not.toBeInTheDocument();expect(fetch.mock.calls.every(([u])=>u.startsWith("/api/model-route-pricing/"))).toBe(true);
});
