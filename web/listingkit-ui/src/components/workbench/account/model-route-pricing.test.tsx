import {afterEach,expect,it,vi} from "vitest";
import {act,cleanup,render,screen,waitFor,within} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {QueryClient,QueryClientProvider} from "@tanstack/react-query";
import {EnterpriseModelEditor,ModelRoutePricingPage,OfficialPriceEditor,ModelRoutePricingEntry} from "./model-route-pricing";
const state=vi.hoisted(()=>({context:{permissions:["workbench.agent.configure"],roles:[],user:{id:"maintainer"},effectiveOrganization:{id:"org-a",name:"企业甲"},registerOrganizationSwitchGuard:()=>()=>{},isLoading:false,isSwitching:false,error:null as {status:number}|null,blockingError:null,selectionRequired:false,aiWorkbenchAvailable:true,aiWorkbenchPlanningReadiness:"NEEDS_CONFIGURATION",retry:vi.fn<()=>Promise<object|null>>(async()=>({}))}}));
vi.mock("@/components/providers/workbench-context-provider",()=>({useWorkbenchContext:()=>state.context}));
const scope={userId:"maintainer",organizationId:"org-a"};
const selection={schemaVersion:1,unitVersion:"ai-micropoint-v1",defaultChannel:"openai-direct",text:{id:"openai-mini",version:"approved-default-v1"},image:{id:"openai-flare",version:"approved-default-v1"},multimodal:{id:"openai-mini",version:"approved-default-v1"}};
const models=(ready=false)=>["text","image-generation","multimodal"].map(category=>({category,reference:category==="image-generation"?selection.image:selection.text,channel:"openai-direct",model:category==="image-generation"?"gpt-image-2.5-flare-2026-09-08":"gpt-5.4-mini-2026-03-17",ready,...ready?{}:{reason:category==="image-generation"?"PROTOCOL_AND_COST_BOUND_MISSING":"BINDING_NOT_ADMITTED"}}));
const clients:QueryClient[]=[];
const revision=(id:string,key="read-key")=>({
 id,key,scope:{organizationId:"org-a",capability:"ai.models.enterprise"},parent:"NONE",schemaVersion:1,
 payload:{bindingId:"",credentialVersion:"",rates:{},sharedModels:selection},digest:"digest",actorId:"maintainer",publishedAt:"2026-11-03T00:00:00Z",
});

it("keeps authorized credential settings reachable when enterprise model reads fail", async () => {
 state.context.permissions=["workbench.agent.configure","workbench.ai.credentials.manage"];
 const fetch=vi.fn(async(url:string)=>url.endsWith("/connections")
  ? Response.json({organizationId:"org-a",canManage:false,connections:[]})
  : Response.json({code:"UNAVAILABLE"},{status:503}));
 vi.stubGlobal("fetch",fetch);
 const client=new QueryClient({defaultOptions:{queries:{retry:false}}});clients.push(client);
 render(<QueryClientProvider client={client}><ModelRoutePricingEntry scope={scope}/><ModelRoutePricingPage expectedUserId="maintainer"/></QueryClientProvider>);
 expect(screen.getByRole("link",{name:"模型与服务"})).toBeVisible();
 await userEvent.click(screen.getByRole("button",{name:"提供方与 Key"}));
 expect(await screen.findByText("当前企业暂无已登记渠道，请联系系统维护人员。")).toBeVisible();
 expect(screen.queryByRole("button",{name:"录入 Key"})).not.toBeInTheDocument();
});

it("lets the administrator check models when only version history fails", async () => {
 const fetch=fixture(true),original=fetch.getMockImplementation()!;
 fetch.mockImplementation((url,init)=>url.endsWith("/history")
  ? Promise.resolve(Response.json({code:"UNAVAILABLE"},{status:503}))
  : original(url,init));
 mount(true);
 expect(await screen.findByRole("button",{name:"检查并预览"})).toBeEnabled();
 await userEvent.click(screen.getByRole("button",{name:"检查并预览"}));
 expect(await screen.findByRole("heading",{name:"确认企业模型配置"})).toBeVisible();
});

it("uses the published qualification per image model and the workbench projection for Chat", async () => {
 state.context.aiWorkbenchPlanningReadiness="UNAVAILABLE";
 const fetch=fixture(),original=fetch.getMockImplementation()!;
 const published=models(true).map(model=>model.category==="image-generation"?{...model,ready:false,reason:"OFFICIAL_PRICE_MISSING"}:model);
 fetch.mockImplementation((url,init)=>url.endsWith("/enterprise")
  ? Promise.resolve(Response.json({status:"NEEDS_CONFIGURATION",revision:revision("current"),preview:{ready:false,models:published}}))
  : original(url,init));
 mount(true);
 const analysis=screen.getByRole("region",{name:"图片分析模型"});
 const generation=screen.getByRole("region",{name:"图片生成模型"});
 expect(await within(analysis).findByText("模型配置符合条件")).toBeVisible();
 expect(within(generation).getByText("缺少适用参考价")).toBeVisible();
 expect(within(screen.getByRole("region",{name:/硕米 Chat/})).getByText("Chat 规划暂不可用")).toBeVisible();
 expect(fetch.mock.calls.some(([url])=>url.endsWith("/connections"))).toBe(false);
});

it.each([true,false])("distinguishes same-name candidates by publication and model qualification (%s)",async readyIsPublished=>{
 const fetch=fixture(),original=fetch.getMockImplementation()!;
 const textOption=models().find(model=>model.category==="text")!;
 const unready={...textOption,reason:"CREDENTIAL_NOT_ADMITTED"};
 const ready={...textOption,reference:{id:"registered-text",version:"configured-v1"},ready:true,reason:undefined};
 const current=revision("current");
 current.payload.sharedModels={...selection,text:readyIsPublished?ready.reference:unready.reference};
 const other={...textOption,reference:{id:"another-text",version:"configured-v1"},channel:"other"};
 fetch.mockImplementation((url,init)=>{
  if(url.endsWith("/options"))return Promise.resolve(Response.json({organizationId:"org-a",channels:["openai-direct","other"],models:[...models().filter(model=>model.category!=="text"),unready,ready,other]}));
  if(url.endsWith("/enterprise"))return Promise.resolve(Response.json({status:"NEEDS_CONFIGURATION",revision:current,preview:{ready:false,models:models().map(model=>model.category==="text"?(readyIsPublished?ready:unready):model)}}));
  return original(url,init);
 });
 mount();
 const modelsField=await screen.findByRole("combobox",{name:"文本模型"});
 const options=within(modelsField);
 const currentOption=options.getByRole("option",{name:readyIsPublished?/当前已发布 · 模型配置符合条件/:/当前已发布 · 渠道暂不可用/});
 const alternative=options.getByRole("option",{name:readyIsPublished?/渠道暂不可用/:/模型配置符合条件/});
 expect(options.getAllByRole("option")).toHaveLength(3);
 expect(modelsField).toHaveValue((currentOption as HTMLOptionElement).value);
 await userEvent.selectOptions(modelsField,alternative);
 expect(modelsField).toHaveValue((alternative as HTMLOptionElement).value);
 expect(currentOption).toHaveTextContent("当前已发布");
 expect(options.getByRole("option",{name:/ · other/})).toBeInTheDocument();
});

it("limits a model's unavailability to its own option",async()=>{
 const fetch=fixture(),original=fetch.getMockImplementation()!;
 const unready=models().map(model=>model.category==="multimodal"?{...model,reason:"VISION_CONSUMER_NOT_IMPLEMENTED"}:model);
 const registered=models(true).filter(model=>model.category!=="image-generation").map(model=>({
  ...model,reference:{id:`registered-${model.category}`,version:"configured-v1"},
 }));
 fetch.mockImplementation((url,init)=>url.endsWith("/options")
  ? Promise.resolve(Response.json({organizationId:"org-a",channels:["openai-direct"],models:[...unready,...registered]}))
  : original(url,init));
 mount();
 const text=within(await screen.findByRole("combobox",{name:"文本模型"}));
 const vision=within(screen.getByRole("combobox",{name:"多模态理解模型"}));
 expect(text.getByRole("option",{name:/当前企业暂不能使用此选项/})).toBeInTheDocument();
 expect(vision.getByRole("option",{name:/此选项暂不能用于图片分析/})).toBeInTheDocument();
 expect(text.getByRole("option",{name:/模型配置符合条件/})).toBeInTheDocument();
 expect(vision.getByRole("option",{name:/模型配置符合条件/})).toBeInTheDocument();
});

it("shows the image generation reason and returns focus to that status after inspection",async()=>{
 const fetch=fixture(),original=fetch.getMockImplementation()!;
 fetch.mockImplementation((url,init)=>url.endsWith("/enterprise")
  ? Promise.resolve(Response.json({status:"NEEDS_CONFIGURATION",revision:revision("current"),preview:{ready:false,models:models()}}))
  : original(url,init));
 mount(true);
 const user=userEvent.setup();
 const generation=within(screen.getByRole("region",{name:"图片生成模型"}));
 expect(await generation.findByText("图片生成服务尚未配置")).toBeVisible();
 const view=await generation.findByRole("button",{name:"查看模型"});
 const field=await screen.findByRole("combobox",{name:"图片生成模型"});
 await user.click(view);
 expect(field).toHaveFocus();
 await user.click(screen.getByRole("button",{name:"返回使用概览"}));
 expect(generation.getByRole("heading",{name:"图片生成模型"})).toHaveFocus();
 const refresh=screen.getByRole("button",{name:"重新读取状态"});
 await user.click(refresh);
 await waitFor(()=>expect(refresh).toBeEnabled());
 expect(field).not.toHaveFocus();
});

it("does not use ready candidates when the current selection cannot be read",async()=>{
 const fetch=fixture(true),original=fetch.getMockImplementation()!;
 fetch.mockImplementation((url,init)=>url.endsWith("/enterprise")
  ? Promise.resolve(Response.json({code:"NEEDS_CONFIGURATION"},{status:409}))
  : original(url,init));
 mount(true);
 const analysis=screen.getByRole("region",{name:"图片分析模型"});
 expect(await within(analysis).findByText("模型配置暂时无法确认")).toBeVisible();
 expect(within(analysis).queryByText("模型配置符合条件")).not.toBeInTheDocument();
 expect(screen.queryByRole("button",{name:"检查并预览"})).not.toBeInTheDocument();
});

it.each([true,false])("locates a connection only for a unique exact model match (%s)",async unique=>{
 state.context.permissions=["workbench.agent.configure","workbench.ai.credentials.manage"];
 const fetch=fixture(true),original=fetch.getMockImplementation()!;
 const published=models(true).map(model=>model.category==="multimodal"?{...model,ready:false,reason:"CREDENTIAL_NOT_ADMITTED"}:model);
 const exactModel={...selection.multimodal,model:"vision",category:"multimodal"};
 const connection=(connectionId:string,provider:string,model:typeof exactModel)=>({
  organizationId:"org-a",connectionId,provider,channel:"openai-direct",region:"global",account:"",revision:"0",enabled:false,keyPresent:false,models:[model],
 });
 fetch.mockImplementation((url,init)=>{
  if(url.endsWith("/enterprise")) return Promise.resolve(Response.json({status:"NEEDS_CONFIGURATION",revision:revision("current"),preview:{ready:false,models:published}}));
  if(url.endsWith("/connections")) return Promise.resolve(Response.json({organizationId:"org-a",canManage:true,connections:[
   connection("first","correct",exactModel), connection("second","other",unique?{...exactModel,category:"text"}:exactModel),
  ]}));
  return original(url,init);
 });
 mount(true);
 const analysis=screen.getByRole("region",{name:"图片分析模型"});
 await userEvent.click(await within(analysis).findByRole("button",{name:"查看渠道配置"}));
 const heading=await screen.findByRole("heading",{name:unique?"correct · openai-direct":"渠道凭据"});
 await waitFor(()=>expect(heading).toHaveFocus());
 expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
 expect(fetch.mock.calls.filter(([url])=>url.includes("/connections")).map(([url])=>url)).toEqual(["/api/model-route-pricing/connections"]);
});

it.each([true,false])("preserves the receipt and checks current configuration immediately after publish (%s)", async refreshed=>{
 let saved=false;
 const fetch=fixture(true,async init=>{
  saved=true;
  return Response.json({receipt:revision("saved",new Headers(init.headers).get("Idempotency-Key")!)});
 });
 const original=fetch.getMockImplementation()!;
 fetch.mockImplementation((url,init)=>saved&&url.endsWith("/enterprise")
  ? Promise.resolve(refreshed?Response.json({status:"READY",revision:revision("saved"),preview:{ready:true,models:models(true)}}):Response.json({code:"UNAVAILABLE"},{status:503}))
  : original(url,init));
 mount(true);
 await userEvent.click(await screen.findByRole("button",{name:"检查并预览"}));
 await userEvent.click(await screen.findByRole("button",{name:"确认发布三类模型"}));
 expect(await screen.findByRole("heading",{name:"发布回执"})).toBeVisible();
 expect(await screen.findByText(refreshed?"当前版本：saved · 已发布":"本次已保存，当前状态暂时无法确认。")).toBeVisible();
 expect(state.context.retry).toHaveBeenCalledTimes(1);
 expect(fetch.mock.calls.filter(([url])=>url.endsWith("/enterprise"))).toHaveLength(2);
});

it("keeps a save receipt and hides stale configuration when the workbench refresh fails",async()=>{
 state.context.retry.mockImplementation(async()=>{state.context.error={status:503};return null});
 const fetch=fixture(true,async init=>Response.json({receipt:revision("saved",new Headers(init.headers).get("Idempotency-Key")!)}));
 mount(true);
 await userEvent.click(await screen.findByRole("button",{name:"检查并预览"}));
 await userEvent.click(await screen.findByRole("button",{name:"确认发布三类模型"}));
 expect(await screen.findByText("本次已保存，当前状态暂时无法确认。")).toBeVisible();
 expect(screen.getByRole("heading",{name:"发布回执"})).toBeVisible();
 expect(screen.getByText("Chat 规划状态暂时无法确认")).toBeVisible();
 expect(screen.queryByLabelText("文本模型")).not.toBeInTheDocument();
 expect(fetch.mock.calls.filter(([url])=>url.endsWith("/publish"))).toHaveLength(1);
});

it("reads again after a publish receipt arrives during an earlier manual refresh",async()=>{
 let saved=false;
 let finishPublish:(response:Response)=>void=()=>{};
 let finishFirstRead:(context:object)=>void=()=>{};
 const firstRead=new Promise<object>(resolve=>{finishFirstRead=resolve});
 state.context.retry.mockImplementationOnce(()=>firstRead).mockImplementationOnce(async()=>{
  state.context.aiWorkbenchPlanningReadiness="AVAILABLE";
  return {};
 });
 const fetch=fixture(true,init=>new Promise<Response>(resolve=>{
  finishPublish=response=>{
   saved=true;
   resolve(response);
  };
  expect(new Headers(init.headers).get("Idempotency-Key")).toBeTruthy();
 }));
 const original=fetch.getMockImplementation()!;
 fetch.mockImplementation((url,init)=>saved&&url.endsWith("/enterprise")
  ? Promise.resolve(Response.json({status:"READY",revision:revision("saved"),preview:{ready:true,models:models(true)}}))
  : original(url,init));
 mount(true);
 const user=userEvent.setup();
 await user.click(await screen.findByRole("button",{name:"检查并预览"}));
 await user.click(await screen.findByRole("button",{name:"确认发布三类模型"}));
 const write=fetch.mock.calls.find(([url])=>url.endsWith("/publish"));
 expect(write).toBeDefined();
 await user.click(screen.getByRole("button",{name:"重新读取状态"}));
 await waitFor(()=>expect(fetch.mock.calls.filter(([url])=>url.endsWith("/enterprise"))).toHaveLength(2));
 await act(async()=>finishPublish(Response.json({receipt:revision("saved",new Headers(write?.[1]?.headers).get("Idempotency-Key")!)})));
 expect(await screen.findByRole("heading",{name:"发布回执"})).toBeVisible();
 await act(async()=>finishFirstRead({}));
 expect(await screen.findByText("企业已具备 Chat 规划条件")).toBeVisible();
 expect(screen.getByText("当前版本：saved · 已发布")).toBeVisible();
 expect(state.context.retry).toHaveBeenCalledTimes(2);
});

it("keeps the model choice across settings areas and clears the old business return on enterprise switch",async()=>{
 state.context.permissions=["workbench.agent.configure","workbench.ai.credentials.manage"];
 const fetch=fixture(true),original=fetch.getMockImplementation()!;
 fetch.mockImplementation((url,init)=>{
  if(url.endsWith("/connections")) return Promise.resolve(Response.json({organizationId:"org-a",canManage:true,connections:[]}));
  if(url.endsWith("/options")) return Promise.resolve(Response.json({organizationId:"org-a",channels:["openai-direct","other"],models:models(true)}));
  return original(url,init);
 });
 const client=new QueryClient({defaultOptions:{queries:{retry:false}}});clients.push(client);
 const navigation={category:"text" as const,organizationId:"org-a",returnTo:"/workbench/ai/chat/new"};
 const page=<QueryClientProvider client={client}><ModelRoutePricingPage expectedUserId="maintainer" navigation={navigation}/></QueryClientProvider>;
 const view=render(page);
 const user=userEvent.setup();
 expect(screen.getByRole("link",{name:"返回原业务页面"})).toHaveAttribute("href",navigation.returnTo);
 await waitFor(()=>expect(screen.getByRole("combobox",{name:"文本模型"})).toHaveFocus());
 await user.click(screen.getByRole("button",{name:"返回使用概览"}));
 expect(screen.getByRole("heading",{name:/硕米 Chat/})).toHaveFocus();
 await user.selectOptions(await screen.findByLabelText("默认渠道"),"other");
 await user.click(screen.getByRole("button",{name:"提供方与 Key"}));
 await screen.findByText(/暂无已登记渠道/);
 await user.click(screen.getByRole("button",{name:"企业模型"}));
 expect(await screen.findByLabelText("默认渠道")).toHaveValue("other");
 state.context.effectiveOrganization={id:"org-b",name:"企业乙"};
 view.rerender(<QueryClientProvider client={client}><ModelRoutePricingPage expectedUserId="maintainer" navigation={navigation}/></QueryClientProvider>);
 expect(screen.queryByRole("link",{name:"返回原业务页面"})).not.toBeInTheDocument();
 expect(await screen.findByLabelText("默认渠道")).toHaveValue("openai-direct");
});

it("opens the existing model settings entry for a credential-only manager without model configuration access",async()=>{
 state.context.permissions=["workbench.ai.credentials.manage"];
 const fetch=vi.fn(async()=>Response.json({organizationId:"org-a",canManage:true,connections:[]}));
 vi.stubGlobal("fetch",fetch);
 const client=new QueryClient({defaultOptions:{queries:{retry:false}}});clients.push(client);
 render(<QueryClientProvider client={client}><ModelRoutePricingEntry scope={scope}/></QueryClientProvider>);
 expect(await screen.findByRole("link",{name:"模型与服务"})).toBeVisible();
 expect(fetch.mock.calls).toHaveLength(0);
});

it("opens only credential management for its independent grant and drops the secret dialog on revocation",async()=>{
 Object.defineProperty(HTMLDialogElement.prototype,"showModal",{configurable:true,value:function(this:HTMLDialogElement){this.open=true}});
 Object.defineProperty(HTMLDialogElement.prototype,"close",{configurable:true,value:function(this:HTMLDialogElement){this.open=false}});
 state.context.permissions=["workbench.ai.credentials.manage"];
 const fetch=vi.fn(async(url:string)=>{if(!url.endsWith("/connections"))throw new Error("unexpected model or price request");return Response.json({organizationId:"org-a",canManage:true,connections:[{organizationId:"org-a",connectionId:"shared",provider:"openai",channel:"openai-direct",region:"global",account:"",revision:"0",enabled:false,keyPresent:false,models:[]}]})});vi.stubGlobal("fetch",fetch);
 const user=userEvent.setup(),client=new QueryClient({defaultOptions:{queries:{retry:false}}});clients.push(client);
 const view=render(<QueryClientProvider client={client}><ModelRoutePricingPage expectedUserId="maintainer"/></QueryClientProvider>);
 await user.click(await screen.findByRole("button",{name:"录入 Key"}));await user.type(screen.getByLabelText("新的渠道 Key"),"synthetic-not-sent");
 expect(fetch.mock.calls.every(([url])=>String(url).endsWith("/connections"))).toBe(true);
 state.context.permissions=[];view.rerender(<QueryClientProvider client={client}><ModelRoutePricingPage expectedUserId="maintainer"/></QueryClientProvider>);
 expect(screen.queryByLabelText("新的渠道 Key")).not.toBeInTheDocument();expect(JSON.stringify({...sessionStorage})).not.toContain("synthetic-not-sent");
 expect(screen.getByText("无模型配置权限或当前企业不可用")).toBeVisible();
});
function mount(page=false){const client=new QueryClient({defaultOptions:{queries:{retry:false}}});clients.push(client);return render(<QueryClientProvider client={client}>{page?<ModelRoutePricingPage expectedUserId="maintainer"/>:<EnterpriseModelEditor scope={scope}/>}</QueryClientProvider>)}
function fixture(ready=false,publish?:(init:RequestInit)=>Promise<Response>){const fetch=vi.fn(async(url:string,init:RequestInit={})=>{if(url.endsWith("/options"))return Response.json({organizationId:"org-a",channels:["openai-direct"],models:models(ready)});if(url.endsWith("/history"))return Response.json({revisions:[],nextCursor:""});if(url.endsWith("/validate")){const input=JSON.parse(String(init.body));return Response.json({...input,preview:{ready,models:models(ready)}})}if(url.endsWith("/publish"))return publish?publish(init):Response.json({code:"NEEDS_CONFIGURATION"},{status:409});return Response.json({status:"UNCONFIGURED",revision:null,preview:null})});vi.stubGlobal("fetch",fetch);return fetch}
afterEach(()=>{cleanup();for(const client of clients)client.clear();clients.length=0;sessionStorage.clear();vi.unstubAllGlobals();state.context.permissions=["workbench.agent.configure"];state.context.effectiveOrganization={id:"org-a",name:"企业甲"};state.context.aiWorkbenchAvailable=true;state.context.aiWorkbenchPlanningReadiness="NEEDS_CONFIGURATION";state.context.error=null;state.context.retry.mockReset();state.context.retry.mockResolvedValue({})});
it("shows the active settings area and keeps enterprise editing apart from platform prices", async () => {
 state.context.permissions = ["workbench.agent.configure", "listingkit.platform_admin", "listingkit.admin.write"];
 fixture();
 mount(true);
 const user = userEvent.setup();
 const enterprise = screen.getByRole("button", { name: "企业模型" });
 const prices = screen.getByRole("button", { name: "参考价格与汇率" });
 expect(enterprise).toHaveAttribute("aria-pressed", "true");
 expect(await screen.findByLabelText("文本模型")).toBeVisible();
 await user.click(prices);
 expect(prices).toHaveAttribute("aria-pressed", "true");
 expect(enterprise).toHaveAttribute("aria-pressed", "false");
 expect(screen.queryByLabelText("文本模型")).not.toBeInTheDocument();
 expect(screen.getByRole("region", { name: "美元汇率自动更新" })).toBeVisible();
 expect(screen.getByRole("region", { name: "模型参考价" })).toBeVisible();
 expect(screen.getByRole("region", { name: "人民币汇率" })).toBeVisible();
});
it("moves keyboard focus to the preview after checking the model selection", async () => {
 fixture();
 mount();
 await userEvent.click(await screen.findByRole("button", { name: "检查并预览" }));
 expect(await screen.findByRole("heading", { name: "确认企业模型配置" })).toHaveFocus();
});
it("uses enterprise configure permission and removes all editable retail point prices",async()=>{fixture();mount(true);expect(await screen.findByLabelText("文本模型")).toBeVisible();expect(screen.getByRole("combobox",{name:"图片生成模型"})).toBeVisible();expect(screen.getByLabelText("多模态理解模型")).toBeVisible();expect(screen.queryByText("每张图片 AI 点数")).not.toBeInTheDocument();expect(screen.queryByText("输入 AI 点数")).not.toBeInTheDocument();expect(screen.queryByRole("button",{name:"参考价格与汇率"})).not.toBeInTheDocument()});
it("saves complete model choices without presenting an unready category as executable",async()=>{const user=userEvent.setup(),fetch=fixture();mount();await user.click(await screen.findByRole("button",{name:"检查并预览"}));const publish=await screen.findByRole("button",{name:"确认发布三类模型"});expect(publish).toBeEnabled();expect(screen.getAllByText(/图片生成服务尚未配置/).length).toBeGreaterThan(0);expect(fetch.mock.calls.some(([url])=>url.endsWith("/publish"))).toBe(false);await user.click(publish);expect(fetch.mock.calls.filter(([url])=>url.endsWith("/publish"))).toHaveLength(1)});
it("publishes all categories in one original-key command after a ready preview",async()=>{const user=userEvent.setup(),fetch=fixture(true,async init=>Response.json({receipt:{id:"r1",scope:{organizationId:"org-a",capability:"ai.models.enterprise"},parent:"NONE",schemaVersion:1,payload:{bindingId:"",credentialVersion:"",rates:{},sharedModels:selection},digest:"digest",actorId:"maintainer",publishedAt:"2026-11-03T00:00:00Z",key:new Headers(init.headers).get("Idempotency-Key")}}));mount();await user.click(await screen.findByRole("button",{name:"检查并预览"}));await user.click(await screen.findByRole("button",{name:"确认发布三类模型"}));await screen.findByRole("heading",{name:"发布回执"});const calls=fetch.mock.calls.filter(([url])=>url.endsWith("/publish"));expect(calls).toHaveLength(1);expect(JSON.parse(String(calls[0][1]?.body))).toEqual({expectedRevision:"NONE",selection});expect(new Headers(calls[0][1]?.headers).get("Idempotency-Key")).toBeTruthy()});
it("preserves the original key after an uncertain publish and a missing receipt",async()=>{const user=userEvent.setup(),fetch=fixture(true,async()=>{throw new Error("ack lost")});mount();await user.click(await screen.findByRole("button",{name:"检查并预览"}));await user.click(await screen.findByRole("button",{name:"确认发布三类模型"}));await screen.findByRole("heading",{name:"发布结果待核实"});await user.click(screen.getByRole("button",{name:"查询本次发布结果"}));await screen.findByRole("heading",{name:"发布结果待核实"});expect(screen.getByRole("button",{name:"确认发布三类模型"})).toBeDisabled();expect(fetch.mock.calls.filter(([url])=>url.endsWith("/publish"))).toHaveLength(1)});
it("does not clear an uncertain original command for a receipt with different model choices",async()=>{const user=userEvent.setup(),fetch=fixture(true,async()=>{throw new Error("ack lost")}),original=fetch.getMockImplementation()!;fetch.mockImplementation(async(url,init={})=>url.includes("/receipts/")?Response.json({receipt:{id:"r1",scope:{organizationId:"org-a",capability:"ai.models.enterprise"},parent:"NONE",schemaVersion:1,payload:{bindingId:"",credentialVersion:"",rates:{},sharedModels:{...selection,image:{id:"different",version:"v1"}}},digest:"digest",actorId:"maintainer",publishedAt:"2026-11-03T00:00:00Z",key:url.split("/").at(-1)}}):original(url,init));mount();await user.click(await screen.findByRole("button",{name:"检查并预览"}));await user.click(await screen.findByRole("button",{name:"确认发布三类模型"}));await screen.findByRole("heading",{name:"发布结果待核实"});await user.click(screen.getByRole("button",{name:"查询本次发布结果"}));await screen.findByText("发布结果待核实，请使用原请求键查询回执。");expect(screen.queryByRole("heading",{name:"发布回执"})).not.toBeInTheDocument();expect(screen.getByRole("heading",{name:"发布结果待核实"})).toBeVisible();expect(fetch.mock.calls.filter(([url])=>url.endsWith("/publish"))).toHaveLength(1)});
it("permits a new initial command after a definite initial rejection",async()=>{const user=userEvent.setup(),fetch=fixture(true,async()=>Response.json({code:"PERMISSION_DENIED"},{status:403}));mount();await user.click(await screen.findByRole("button",{name:"检查并预览"}));const publish=await screen.findByRole("button",{name:"确认发布三类模型"});await user.click(publish);await screen.findByRole("alert");expect(screen.queryByRole("heading",{name:"发布结果待核实"})).not.toBeInTheDocument();expect(publish).toBeEnabled();await user.click(publish);const attempts=fetch.mock.calls.filter(([url])=>url.endsWith("/publish"));expect(attempts).toHaveLength(2);expect(new Headers(attempts[0][1]?.headers).get("Idempotency-Key")).not.toBe(new Headers(attempts[1][1]?.headers).get("Idempotency-Key"))});
it("keeps the original uncertain command when its receipt lookup is later denied",async()=>{const user=userEvent.setup(),fetch=fixture(true,async()=>{throw new Error("ack lost")}),original=fetch.getMockImplementation()!;fetch.mockImplementation(async(url,init={})=>url.includes("/receipts/")?Response.json({code:"PERMISSION_DENIED"},{status:403}):original(url,init));mount();await user.click(await screen.findByRole("button",{name:"检查并预览"}));await user.click(await screen.findByRole("button",{name:"确认发布三类模型"}));await screen.findByRole("heading",{name:"发布结果待核实"});await user.click(screen.getByRole("button",{name:"查询本次发布结果"}));await screen.findByRole("alert");expect(screen.getByRole("heading",{name:"发布结果待核实"})).toBeVisible();expect(screen.getByRole("button",{name:"确认发布三类模型"})).toBeDisabled();expect(fetch.mock.calls.filter(([url])=>url.endsWith("/publish"))).toHaveLength(1)});
it("blocks the page on organization switch or missing configure permission",()=>{state.context.permissions=[];fixture();mount(true);expect(screen.getByText("无模型配置权限或当前企业不可用")).toBeVisible();expect(screen.queryByRole("button",{name:"检查并预览"})).not.toBeInTheDocument()});
it("opens platform prices without conferring enterprise configuration permission",async()=>{state.context.permissions=["listingkit.platform_admin","listingkit.admin.write"];const fetch=fixture();mount(true);expect(await screen.findByRole("heading",{name:"模型参考价与人民币汇率"})).toBeVisible();await waitFor(()=>expect(screen.getByRole("button",{name:"添加参考价目"})).toBeEnabled());expect(screen.queryByLabelText("文本模型")).not.toBeInTheDocument();expect(fetch.mock.calls.every(([url])=>!url.includes("/enterprise"))).toBe(true)});
it("keeps a platform reader without write authority read-only",async()=>{state.context.permissions=["listingkit.platform_admin"];fixture();mount(true);expect(await screen.findByRole("heading",{name:"模型参考价与人民币汇率"})).toBeVisible();expect(screen.getByRole("button",{name:"添加参考价目"})).toBeDisabled();expect(screen.queryByLabelText("文本模型")).not.toBeInTheDocument()});
it("shows the platform maintenance entry without probing any configuration API",async()=>{state.context.permissions=["listingkit.platform_admin"];const fetch=fixture(),client=new QueryClient({defaultOptions:{queries:{retry:false}}});clients.push(client);render(<QueryClientProvider client={client}><ModelRoutePricingEntry scope={scope}/></QueryClientProvider>);expect(await screen.findByRole("link",{name:"模型与服务"})).toBeVisible();expect(fetch).not.toHaveBeenCalled()});
it("shows an automatic USD conflict as stopped and needing review",async()=>{const fetch=fixture(),original=fetch.getMockImplementation()!;fetch.mockImplementation(async(url,init={})=>url.endsWith("/automatic-fx")?Response.json({state:"USD_CONFLICT",checkedAt:"2026-11-12T00:00:00Z",stopRequested:false}):original(url,init));const client=new QueryClient({defaultOptions:{queries:{retry:false}}});clients.push(client);render(<QueryClientProvider client={client}><OfficialPriceEditor scope={scope} canPublish/></QueryClientProvider>);expect(await screen.findByText(/人工美元汇率发生冲突，已停止自动发布并需核对/)).toBeVisible()});
it("explains the weekly reference rate and its waiting state",async()=>{const fetch=fixture(),original=fetch.getMockImplementation()!;fetch.mockImplementation(async(url,init={})=>url.endsWith("/automatic-fx")?Response.json({state:"WAITING",checkedAt:"2026-11-12T00:00:00Z",stopRequested:false}):original(url,init));const client=new QueryClient({defaultOptions:{queries:{retry:false}}});clients.push(client);render(<QueryClientProvider client={client}><OfficialPriceEditor scope={scope} canPublish/></QueryClientProvider>);expect(await screen.findByText(/等待下一次周更新/)).toBeVisible();expect(screen.getByText(/美元自动汇率每周更新一次/)).toBeVisible()});
it("enables only a server-confirmed prepared schedule and keeps an unknown start from being retried",async()=>{
 const fetch=fixture(),original=fetch.getMockImplementation()!;
 fetch.mockImplementation(async(url,init={})=>url.endsWith("/automatic-fx/start")?Promise.reject(new Error("ack lost")):url.endsWith("/automatic-fx")?Response.json({state:"PREPARED",checkedAt:"2026-11-12T00:00:00Z",stopRequested:false}):original(url,init));
 const user=userEvent.setup(),client=new QueryClient({defaultOptions:{queries:{retry:false}}});clients.push(client);render(<QueryClientProvider client={client}><OfficialPriceEditor scope={scope} canPublish/></QueryClientProvider>);
 await user.click(await screen.findByRole("button",{name:"启用每周自动更新"}));
 expect(await screen.findByText(/启用结果待核实/)).toBeVisible();expect(screen.queryByRole("button",{name:"启用每周自动更新"})).not.toBeInTheDocument();
 expect(fetch.mock.calls.filter(([url])=>url.endsWith("/automatic-fx/start"))).toHaveLength(1);
});
it("rechecks the current status before starting from a stale browser",async()=>{
 const fetch=fixture(),original=fetch.getMockImplementation()!;let reads=0;
 fetch.mockImplementation(async(url,init={})=>url.endsWith("/automatic-fx")?Response.json({state:++reads===1?"PREPARED":"CHECKED",checkedAt:"2026-11-12T00:00:00Z",stopRequested:false}):original(url,init));
 const user=userEvent.setup(),client=new QueryClient({defaultOptions:{queries:{retry:false}}});clients.push(client);render(<QueryClientProvider client={client}><OfficialPriceEditor scope={scope} canPublish/></QueryClientProvider>);
 await user.click(await screen.findByRole("button",{name:"启用每周自动更新"}));expect(await screen.findByText(/启用结果待核实/)).toBeVisible();
 expect(fetch.mock.calls.filter(([url])=>url.endsWith("/automatic-fx/start"))).toHaveLength(0);
});
it("never rebases an edited price draft onto a newly published revision",async()=>{
 const fetch=fixture(),original=fetch.getMockImplementation()!;let updated=false;
 fetch.mockImplementation(async(url,init={})=>updated&&url.endsWith("/platform-prices")?Response.json({status:"READY",preview:null,revision:{id:"new-revision",scope:{organizationId:"platform-ai-models",capability:"ai.models.official-prices"},parent:"NONE",schemaVersion:1,payload:{bindingId:"",credentialVersion:"",rates:{},platformPrices:{schemaVersion:1,unitVersion:"ai-micropoint-v1",prices:[],fx:[]}},digest:"digest",actorId:"admin",publishedAt:"2026-11-12T00:00:00Z",key:"key"}}):original(url,init));
 const user=userEvent.setup(),client=new QueryClient({defaultOptions:{queries:{retry:false}}});clients.push(client);render(<QueryClientProvider client={client}><OfficialPriceEditor scope={scope} canPublish/></QueryClientProvider>);
 await user.click(await screen.findByRole("button",{name:"添加汇率"}));updated=true;await user.click(screen.getByRole("button",{name:"重新读取参考价与汇率"}));
 expect(await screen.findByText(/草稿基于旧版本/)).toBeVisible();expect(screen.getByRole("button",{name:"校验参考价与汇率"})).toBeDisabled();expect(fetch.mock.calls.some(([url])=>url.endsWith("/validate"))).toBe(false);
 await user.click(screen.getByRole("button",{name:"放弃草稿并载入当前版本"}));expect(screen.getByRole("button",{name:"校验参考价与汇率"})).toBeEnabled();
});
it("keeps displayed official amounts bound to the surviving price after deleting an earlier row",async()=>{fixture();const user=userEvent.setup(),client=new QueryClient({defaultOptions:{queries:{retry:false}}});clients.push(client);render(<QueryClientProvider client={client}><OfficialPriceEditor scope={scope} canPublish/></QueryClientProvider>);const add=await screen.findByRole("button",{name:"添加参考价目"});await user.click(add);await user.click(add);const rates=screen.getAllByLabelText("文字输入 / 百万 Token（参考原币）");await user.clear(rates[0]);await user.type(rates[0],"1.25");await user.clear(rates[1]);await user.type(rates[1],"2.5");await user.click(screen.getAllByRole("button",{name:"删除此价目"})[0]);expect(screen.getByLabelText("文字输入 / 百万 Token（参考原币）")).toHaveValue("2.5")});

it("offers Anthropic in the existing reference price form", async () => {
 fixture();
 const user=userEvent.setup(), client=new QueryClient({defaultOptions:{queries:{retry:false}}});
 clients.push(client);
 render(<QueryClientProvider client={client}><OfficialPriceEditor scope={scope} canPublish/></QueryClientProvider>);
 await user.click(await screen.findByRole("button",{name:"添加参考价目"}));
 await user.selectOptions(screen.getByLabelText("模型厂商"),"anthropic");
 expect(screen.getByLabelText("模型厂商")).toHaveValue("anthropic");
 expect(screen.getAllByText("参考价目 1")).toHaveLength(1);
});
