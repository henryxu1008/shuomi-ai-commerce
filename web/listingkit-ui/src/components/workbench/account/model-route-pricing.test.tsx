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

it("keeps optional publication mode when editing a historical option and clears it on discard",async()=>{
 const user=userEvent.setup(),vip={...payload,optional:true as const,bindingId:"image-vip",image:{...payload.image,model:"gpt-image-2-vip"}},old={...revision("vip-old"),payload:vip};
 const fetch=fixtures(async()=>Response.json({})),original=fetch.getMockImplementation()!;
 fetch.mockImplementation(async(url:string,init:RequestInit={})=>{
  if(url.endsWith("/bindings"))return Response.json({bindings:[...bindings.bindings,{...bindings.bindings[0],id:"image-vip",model:"gpt-image-2-vip"}]});
  if(url.endsWith("/history"))return Response.json({revisions:[old],nextCursor:""});
  if(url.endsWith("/options"))return Response.json({revisions:[old]});
  if(url.endsWith("/validate")){const input=JSON.parse(String(init.body));return Response.json({payload:{...vip,...input.optional?{optional:true}:{},rates:input.rates},expectedRevision:input.expectedRevision,digest:"preview"})}
  return original(url,init);
 });vi.stubGlobal("fetch",fetch);mount();
 await user.click(await screen.findByRole("button",{name:"以历史配置创建新预览"}));
 await screen.findByRole("button",{name:"确认发布为可选，不设为默认"});
 const rate=screen.getByLabelText(/每张图片 AI 点数/);await user.clear(rate);await user.type(rate,"9");
 await user.click(screen.getByRole("button",{name:"校验配置并预览"}));
 await screen.findByRole("button",{name:"确认发布为可选，不设为默认"});
 const last=fetch.mock.calls.filter(([u])=>u.endsWith("/validate")).at(-1)!;
 expect(JSON.parse(String(last[1]!.body)).optional).toBe(true);
 await user.click(screen.getByRole("button",{name:"放弃编辑"}));
 await user.selectOptions(screen.getByLabelText(/已准入图片绑定/),"image-vip");
 expect(screen.getByRole("checkbox",{name:"发布为可选，不设为默认"})).not.toBeChecked();
});

it("publishes VIP as an optional version using its own head without switching default",async()=>{
 const user=userEvent.setup(),fetch=fixtures(async init=>{const input=JSON.parse(String(init.body));return Response.json({receipt:{...revision("vip-r2",new Headers(init.headers).get("Idempotency-Key")!),payload:{...payload,optional:true,bindingId:input.bindingId,image:{...payload.image,model:"gpt-image-2-vip"}},parent:"NONE"}})});
 const original=fetch.getMockImplementation()!;
 fetch.mockImplementation(async(url:string,init:RequestInit={})=>{
  if(url.endsWith("/bindings"))return Response.json({bindings:[{...bindings.bindings[0],id:"image-vip",model:"gpt-image-2-vip"}]});
  if(url.endsWith("/options"))return Response.json({revisions:[]});
  if(url.endsWith("/validate")){const input=JSON.parse(String(init.body));return Response.json({payload:{...payload,optional:true,bindingId:"image-vip",image:{...payload.image,model:"gpt-image-2-vip"}},expectedRevision:input.expectedRevision,digest:"preview"})}
  return original(url,init);
 });vi.stubGlobal("fetch",fetch);mount();
 await user.selectOptions(await screen.findByLabelText(/已准入图片绑定/),"image-vip");
 await user.click(screen.getByRole("checkbox",{name:"发布为可选，不设为默认"}));
 await user.click(screen.getByRole("button",{name:"校验配置并预览"}));
 await user.click(await screen.findByRole("button",{name:"确认发布为可选，不设为默认"}));
 await screen.findByRole("heading",{name:"发布回执"});
 const body=JSON.parse(String(fetch.mock.calls.find(([u])=>u.endsWith("/publish"))![1]!.body));
 expect(body.optional).toBe(true);expect(body.expectedRevision).toBe("NONE");
 expect(screen.getByText("当前版本：r1 · 已发布")).toBeVisible();
});
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
afterEach(()=>{document.documentElement.className="";cleanup();clients.splice(0).forEach(c=>c.clear());sessionStorage.clear();vi.unstubAllGlobals();state.context.permissions=["listingkit.platform_admin","listingkit.admin.write"]});
it.each(["light","dark"])("shows fixed VIP constraints without publishing the existing default in %s theme",async theme=>{
 document.documentElement.className=theme;
 const user=userEvent.setup(),fetch=fixtures(async()=>Response.json({}));
 const original=fetch.getMockImplementation()!;
 fetch.mockImplementation(async(url:string,init:RequestInit={})=>{
  if(url.endsWith("/bindings"))return Response.json({bindings:[...bindings.bindings,{...bindings.bindings[0],id:"image-vip",model:"gpt-image-2-vip"}]});
  if(url.endsWith("/validate")){const input=JSON.parse(String(init.body));return Response.json({payload:{...payload,bindingId:"image-vip",image:{...payload.image,model:"gpt-image-2-vip"}},expectedRevision:input.expectedRevision,digest:"vip-preview"})}
  if(url.endsWith("/imageagent.generation"))return Response.json({status:"READY",revision:{...revision(),payload:{...payload,image:{...payload.image,provider:"bailian",model:"qwen-image-3.0-pro",protocol:"dashscope-image-sync-v1"}}}});
  return original(url,init);
 });vi.stubGlobal("fetch",fetch);mount();
 const select=await screen.findByLabelText(/已准入图片绑定/);
 await user.selectOptions(select,"image-vip");
 expect(screen.getByText(/固定质量：medium/)).toBeVisible();
 expect(screen.getAllByText(/原图 1024×1024/).length).toBeGreaterThan(0);
 expect(fetch.mock.calls.some(([url])=>url.endsWith("/publish")||url.endsWith("/validate"))).toBe(false);
 expect(screen.getByRole("heading",{name:"当前已发布配置"})).toBeVisible();
 expect(screen.getByText("bailian / qwen-image-3.0-pro / dashscope-image-sync-v1")).toBeVisible();
 await user.click(screen.getByRole("button",{name:"校验配置并预览"}));
 await screen.findByRole("heading",{name:"确认发布这个用途的新配置"});
 expect(screen.getAllByText(/固定质量：medium/)).toHaveLength(2);
 const submitted=JSON.parse(String(fetch.mock.calls.find(([url])=>url.endsWith("/validate"))![1]!.body));
 expect(submitted).toEqual({expectedRevision:"r1",bindingId:"image-vip",credentialVersion:"credential-a",rates:{pointsPerImage:"8"}});
 expect(fetch.mock.calls.some(([url])=>url.endsWith("/publish"))).toBe(false);
});
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
it("keeps an unconfigured service unavailable after platform permission passes",async()=>{
 const fetch=vi.fn(async(url:string)=>{expect(url).toBe("/api/model-route-pricing");return Response.json({code:"UNAVAILABLE"},{status:503})});vi.stubGlobal("fetch",fetch);mount(true);
 await screen.findByRole("heading",{name:"配置服务未接线或暂不可用"});expect(screen.queryByText("无平台维护权限或当前企业不可用")).not.toBeInTheDocument();
 expect(fetch).toHaveBeenCalledTimes(1);expect(fetch.mock.calls[0]?.[0]).toBe("/api/model-route-pricing");expect(screen.queryByRole("button",{name:"确认发布"})).not.toBeInTheDocument();
});
it("does not advertise configured uses when the authorized capability list is empty",async()=>{
 const fetch=vi.fn(async()=>Response.json({organizationId:"org-a",capabilities:[]}));vi.stubGlobal("fetch",fetch);mount(true);
 await screen.findByRole("heading",{name:"当前企业暂无已接线用途"});expect(fetch).toHaveBeenCalledTimes(1);expect(screen.queryByRole("button",{name:"确认发布"})).not.toBeInTheDocument();
});
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
