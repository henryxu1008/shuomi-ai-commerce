import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";

import { AIWorkbenchError } from "@/lib/api/ai-workbench";
import { ChatPage } from "./chat-page";

const fixture = vi.hoisted(() => ({
  request: vi.fn(), templates:vi.fn(), knowledge:vi.fn(), push: vi.fn(), search: new URLSearchParams(),
  context: { user: { id: "user-a" }, effectiveMemberId:"member-a", effectiveOrganization: { id: "org-a", capabilities: { "workbench.chat.use": true } }, roles: ["listingkit_operator"], permissions: ["product_sourcing.write","local_agent.write","listingkit.image_agent.read","listingkit.image_agent.write","workbench.agent.read","workbench.agent.use","workbench.chat.read","workbench.chat.use","workbench.task.read","workbench.knowledge.read","workbench.store.read","workbench.store.create","workbench.store.update","workbench.store.lifecycle","workbench.source_account.read","workbench.source_account.manage","workbench.organization_member.read","workbench.commercial.read"],
    aiWorkbenchExecutionKind:undefined as "product.title.optimize"|"product.content.localize"|undefined,aiWorkbenchExecutionReadiness:"AVAILABLE",aiWorkbenchAvailable: true, aiWorkbenchPlanningReadiness: "AVAILABLE", aiWorkbenchStoreRouteReadiness:"AVAILABLE", aiWorkbenchTitleReadiness: "AVAILABLE", isLoading: false, isSwitching: false, selectionRequired: false, error: null, blockingError: null },
}));
vi.mock("@/lib/api/ai-workbench", async original => ({ ...await original<typeof import("@/lib/api/ai-workbench")>(), requestAIWorkbench: fixture.request }));
vi.mock("@/lib/api/agent-configuration",async original=>({...await original<object>(),configurationRequest:fixture.templates}));
vi.mock("@/lib/api/knowledge",async original=>({...await original<object>(),knowledgeRequest:fixture.knowledge}));
vi.mock("@/components/providers/workbench-context-provider", () => ({ useWorkbenchContext: () => fixture.context }));
vi.mock("../pod/collection-picker",()=>({CollectionPicker:({onSelect,disabled}:{onSelect:(item:unknown)=>void;disabled:boolean})=><button disabled={disabled} onClick={()=>onSelect({id:operationId,title:"Imported own product",source:{kind:"own",productKey:"product",version:"1",publicationId:operationId}})}>选择我的商品</button>}));
vi.mock("next/navigation", () => ({ useRouter: () => ({ push: fixture.push }), useSearchParams: () => fixture.search }));

const conversationId = "550e8400-e29b-41d4-a716-446655440000";
const scopeConfirmation = "a".repeat(64);
const scopeOptions = {options:[{scope:{version:"workbench-scope-v1",kind:"general"},confirmation:scopeConfirmation}],next:"",incomplete:false};
async function selectGeneralScope() {
  await waitFor(() => expect((screen.getByLabelText("工作范围") as HTMLSelectElement).options.length).toBe(2));
  fireEvent.change(screen.getByLabelText("工作范围"), {target:{value:scopeConfirmation}});
}
const operationId = "11111111-1111-4111-8111-111111111111";
const conversation = { ID: conversationId, Scope: { OrganizationID: "org-a", ActorID: "user-a" }, Title: "", Favorite: false,
  Archived: false, MetadataRevision: 1, NextSequence: 1, CreatedAt: "2026-10-03T00:00:00Z", UpdatedAt: "2026-10-03T00:00:00Z" };
let client: QueryClient;
function tree() { return <QueryClientProvider client={client}><ChatPage mode="detail" conversationId={conversationId} /></QueryClientProvider>; }

it("hides store counts on focus until the live history is reauthorized", async () => {
 const stamp="2026-10-03T00:00:00Z";
 const reference={ID:operationId,ConversationID:conversationId,Sequence:1,Author:"ASSISTANT",Content:"店铺概况",CreatedAt:stamp,kind:"STORE_OVERVIEW_REF",storeQueryStatus:"ALL_NON_DELETED",currentStoreOverview:{total:3,platforms:{shein:3},statuses:{active:3},operatingTypeUnknown:3,readStartedAt:stamp}};
 let resolveNext:(value:unknown)=>void=()=>{};
 fixture.request.mockImplementationOnce(async()=>({conversation,messages:[reference],proposals:[],before:""})).mockImplementationOnce(()=>new Promise(resolve=>{resolveNext=resolve}));
 render(tree());
 expect(await screen.findByText(/当前可查看 3 家店铺/)).toBeInTheDocument();
 fireEvent.blur(window); fireEvent.focus(window);
 await waitFor(()=>expect(screen.queryByText(/当前可查看 3 家店铺/)).toBeNull());
 await act(async()=>resolveNext({conversation,messages:[{...reference,currentStoreOverview:undefined,storeOverviewUnavailable:true}],proposals:[],before:""}));
 expect(await screen.findByText(/当前无权查看或店铺数据暂不可用/)).toBeInTheDocument();
 expect(screen.queryByText(/当前可查看 3 家店铺/)).toBeNull();
});

it("discards an initial history response started before the latest focus", async () => {
 const stamp="2026-10-03T00:00:00Z";
 const reference={ID:operationId,ConversationID:conversationId,Sequence:1,Author:"ASSISTANT",Content:"店铺概况",CreatedAt:stamp,kind:"STORE_OVERVIEW_REF",storeQueryStatus:"ALL_NON_DELETED",currentStoreOverview:{total:3,platforms:{shein:3},statuses:{active:3},operatingTypeUnknown:3,readStartedAt:stamp}};
 let resolveOld:(value:unknown)=>void=()=>{},resolveFresh:(value:unknown)=>void=()=>{};
 fixture.request.mockImplementationOnce(()=>new Promise(resolve=>{resolveOld=resolve})).mockImplementationOnce(()=>new Promise(resolve=>{resolveFresh=resolve}));
 render(tree());
 await waitFor(()=>expect(fixture.request).toHaveBeenCalledTimes(1));
 fireEvent.blur(window);fireEvent.focus(window);
 await act(async()=>resolveOld({conversation,messages:[reference],proposals:[],before:""}));
 await waitFor(()=>expect(fixture.request).toHaveBeenCalledTimes(2));
 expect(screen.queryByText(/当前可查看 3 家店铺/)).toBeNull();
 await act(async()=>resolveFresh({conversation,messages:[{...reference,currentStoreOverview:undefined,storeOverviewUnavailable:true}],proposals:[],before:""}));
 expect(await screen.findByText(/当前无权查看或店铺数据暂不可用/)).toBeInTheDocument();
 expect(screen.queryByText(/当前可查看 3 家店铺/)).toBeNull();
});

it.each(["project","store"] as const)("explains the store status breakdown and %s context boundary",async kind=>{
 const stamp="2026-10-03T00:00:00Z", fixed={...conversation,businessScope:{version:"workbench-scope-v1",kind,...(kind==="project"?{projectId:"project-a"}:{storeId:"store-a"})}};
 const reference={ID:operationId,ConversationID:conversationId,Sequence:1,Author:"ASSISTANT",Content:"店铺概况",CreatedAt:stamp,kind:"STORE_OVERVIEW_REF",storeQueryStatus:"ALL_NON_DELETED",currentStoreOverview:{total:3,platforms:{shein:3},statuses:{active:1,disabled:1,deleting:1},operatingTypeUnknown:3,readStartedAt:stamp}};
 fixture.request.mockResolvedValue({conversation:fixed,messages:[reference],proposals:[],before:""});render(tree());
 expect(await screen.findByText(/启用 1 家.*停用 1 家.*删除中 1 家/)).toBeInTheDocument();
 expect(screen.getByText(/查看当前企业所有可见店铺，请进入通用业务会话/)).toBeInTheDocument();
});

it("displays zero for each status on an empty store snapshot",async()=>{
 const stamp="2026-10-03T00:00:00Z";
 const reference={ID:operationId,ConversationID:conversationId,Sequence:1,Author:"ASSISTANT",Content:"店铺概况",CreatedAt:stamp,kind:"STORE_OVERVIEW_REF",storeQueryStatus:"ALL_NON_DELETED",currentStoreOverview:{total:0,platforms:{},statuses:{},operatingTypeUnknown:0,readStartedAt:stamp}};
 fixture.request.mockResolvedValue({conversation,messages:[reference],proposals:[],before:""});render(tree());
 expect(await screen.findByText(/启用 0 家.*停用 0 家.*删除中 0 家/)).toBeInTheDocument();
});

it("does not redisplay cached store counts when reopening fails", async () => {
 const stamp="2026-10-03T00:00:00Z";
 const reference={ID:operationId,ConversationID:conversationId,Sequence:1,Author:"ASSISTANT",Content:"店铺概况",CreatedAt:stamp,kind:"STORE_OVERVIEW_REF",storeQueryStatus:"ALL_NON_DELETED",currentStoreOverview:{total:3,platforms:{shein:3},statuses:{active:3},operatingTypeUnknown:3,readStartedAt:stamp}};
 fixture.request.mockResolvedValueOnce({conversation,messages:[reference],proposals:[],before:""});
 render(tree());expect(await screen.findByText(/当前可查看 3 家店铺/)).toBeInTheDocument();
 cleanup();
 let rejectNext:(reason:unknown)=>void=()=>{};
 fixture.request.mockImplementationOnce(()=>new Promise((_,reject)=>{rejectNext=reject}));
 render(tree());
 expect(screen.queryByText(/当前可查看 3 家店铺/)).toBeNull();
 await act(async()=>rejectNext(new AIWorkbenchError("FORBIDDEN")));
 expect(await screen.findByText(/会话不可用/)).toBeInTheDocument();
 expect(screen.queryByText(/当前可查看 3 家店铺/)).toBeNull();
});

it("hides old store counts when the same user rejoins with a new membership",async()=>{
 const stamp="2026-10-03T00:00:00Z",ref={ID:operationId,ConversationID:conversationId,Sequence:1,Author:"ASSISTANT",Content:"店铺概况",CreatedAt:stamp,kind:"STORE_OVERVIEW_REF",storeQueryStatus:"ALL_NON_DELETED",currentStoreOverview:{total:3,platforms:{shein:3},statuses:{active:3},operatingTypeUnknown:3,readStartedAt:stamp}};
 let resolveNext:(value:unknown)=>void=()=>{};fixture.request.mockResolvedValueOnce({conversation,messages:[ref],proposals:[],before:""}).mockImplementationOnce(()=>new Promise(resolve=>{resolveNext=resolve}));
 const view=render(tree());expect(await screen.findByText(/当前可查看 3 家店铺/)).toBeInTheDocument();fixture.context.effectiveMemberId="member-b";view.rerender(tree());expect(screen.queryByText(/当前可查看 3 家店铺/)).toBeNull();
 await act(async()=>resolveNext({conversation,messages:[{...ref,currentStoreOverview:undefined,storeOverviewUnavailable:true}],proposals:[],before:""}));expect(await screen.findByText(/当前无权查看或店铺数据暂不可用/)).toBeInTheDocument();
});

it("routes a general question only on send and retries the original key", async () => {
 fixture.context.aiWorkbenchExecutionKind="product.content.localize";
 const general={...conversation,businessScope:{version:"workbench-scope-v1",kind:"general"}};
 let calls=0;
 fixture.request.mockImplementation(async ({route})=>{
   if(route==="conversation-read")return {conversation:general,messages:[],proposals:[],before:""};
   if(route==="store-route"){if(calls++===0)throw new AIWorkbenchError("OUTCOME_UNKNOWN");return {state:"COMPLETE"};}
   throw new Error("unexpected route");
 });
 render(tree());
 await waitFor(()=>expect(screen.getByLabelText("需求")).toBeEnabled());
 fireEvent.change(screen.getByLabelText("需求"),{target:{value:"我有几个店铺，分别是什么类型？"}});
 expect(fixture.request.mock.calls.every(([input])=>input.route!=="store-route")).toBe(true);
 fireEvent.click(screen.getByRole("button",{name:"发送消息"}));
 await screen.findByRole("button",{name:"重试同一消息"});
 fireEvent.click(screen.getByRole("button",{name:"重试同一消息"}));
 await waitFor(()=>expect(fixture.request.mock.calls.filter(([input])=>input.route==="store-route")).toHaveLength(2));
 const sent=fixture.request.mock.calls.map(([input])=>input).filter(input=>input.route==="store-route");
 expect(sent[1].key).toBe(sent[0].key);
 expect(sent[0].body).toEqual({content:"我有几个店铺，分别是什么类型？"});
});

it("allows a configured store question when product planning is unavailable",async()=>{
 fixture.context.aiWorkbenchExecutionKind="product.content.localize";fixture.context.aiWorkbenchPlanningReadiness="UNAVAILABLE";fixture.context.aiWorkbenchStoreRouteReadiness="AVAILABLE";
 const general={...conversation,businessScope:{version:"workbench-scope-v1",kind:"general"}};
 fixture.request.mockImplementation(async({route})=>route==="conversation-read"?{conversation:general,messages:[],proposals:[],before:""}:route==="store-route"?{state:"COMPLETE"}:Promise.reject(new Error("unexpected")));
 render(tree());await waitFor(()=>expect(screen.getByLabelText("需求")).toBeEnabled());fireEvent.change(screen.getByLabelText("需求"),{target:{value:"我有几个店铺"}});fireEvent.click(screen.getByRole("button",{name:"发送消息"}));await waitFor(()=>expect(fixture.request.mock.calls.some(([input])=>input.route==="store-route")).toBe(true));
});

it("unlocks the original question only after a definite unavailable-route receipt",async()=>{
 fixture.context.aiWorkbenchExecutionKind="product.content.localize";
 const general={...conversation,businessScope:{version:"workbench-scope-v1",kind:"general"}};
 fixture.request.mockImplementation(async({route})=>route==="conversation-read"?{conversation:general,messages:[],proposals:[],before:""}:route==="store-route"?{state:"FAILED_BEFORE_DISPATCH"}:Promise.reject(new Error("unexpected")));
 render(tree());await waitFor(()=>expect(screen.getByLabelText("需求")).toBeEnabled());fireEvent.change(screen.getByLabelText("需求"),{target:{value:"我有几个店铺"}});fireEvent.click(screen.getByRole("button",{name:"发送消息"}));
 expect(await screen.findByText(/请求尚未发送到模型/)).toBeInTheDocument();expect(screen.getByLabelText("需求")).toHaveValue("我有几个店铺");expect(screen.getByLabelText("需求")).toBeEnabled();
});

it("uses a separate idempotent no-model store overview action", async () => {
 const general={...conversation,businessScope:{version:"workbench-scope-v1",kind:"general"}};
 let calls=0;
 fixture.request.mockImplementation(async ({route})=>{
   if(route==="conversation-read")return {conversation:general,messages:[],proposals:[],before:""};
   if(route==="store-overview"){if(calls++===0)throw new AIWorkbenchError("OUTCOME_UNKNOWN");return {state:"COMPLETE",replay:true};}
   throw new Error("unexpected model call");
 });
 render(tree());
 fireEvent.click(await screen.findByRole("button",{name:"查看店铺概况"}));
 fireEvent.click(await screen.findByRole("button",{name:"核对原店铺查询"}));
 await waitFor(()=>expect(fixture.request.mock.calls.filter(([input])=>input.route==="store-overview")).toHaveLength(2));
 const sent=fixture.request.mock.calls.map(([input])=>input).filter(input=>input.route==="store-overview");
 expect(sent[1].key).toBe(sent[0].key);expect(sent[0].body).toEqual({status:"ALL_NON_DELETED"});
 expect(fixture.request.mock.calls.every(([input])=>input.route!=="store-route" && input.route!=="message")).toBe(true);
});

it("does not carry a pending store query key into a different conversation",async()=>{
 const secondId="33333333-3333-4333-8333-333333333333",pendingKey="44444444-4444-4444-8444-444444444444";
 sessionStorage.setItem(`ai-workbench:store-overview:${JSON.stringify(["user-a","org-a",conversationId])}`,pendingKey);
 fixture.request.mockImplementation(async({route,path})=>{if(route!=="conversation-read")throw new Error("unexpected");return {conversation:{...conversation,ID:path.includes(secondId)?secondId:conversationId,businessScope:{version:"workbench-scope-v1",kind:"general"}},messages:[],proposals:[],before:""}});
 const view=render(tree());expect(await screen.findByRole("button",{name:"核对原店铺查询"})).toBeEnabled();
 view.rerender(<QueryClientProvider client={client}><ChatPage mode="detail" conversationId={secondId}/></QueryClientProvider>);
 await waitFor(()=>expect(screen.getByRole("button",{name:"查看店铺概况"})).toBeEnabled());expect(screen.queryByRole("button",{name:"核对原店铺查询"})).toBeNull();
});

it("inserts final speech at the current caret without losing edits or sending a message", async () => {
  fixture.context.aiWorkbenchExecutionKind = "product.content.localize";
  fixture.request.mockImplementation(async ({route}) => route === "scope-options" ? scopeOptions : Promise.reject(new Error("unexpected request")));
  const instance: {onresult?: (event: unknown) => void; onend?: () => void; start: ReturnType<typeof vi.fn>; abort: ReturnType<typeof vi.fn>; stop: ReturnType<typeof vi.fn>} = {start:vi.fn(), abort:vi.fn(), stop:vi.fn()};
  function FakeRecognition() { return instance; }
  vi.stubGlobal("SpeechRecognition", FakeRecognition);
  try {
    render(<QueryClientProvider client={client}><ChatPage mode="new" /></QueryClientProvider>);
    const input = await screen.findByLabelText("需求") as HTMLTextAreaElement;
    await waitFor(() => expect(input).toBeEnabled());
    fireEvent.change(input, {target:{value:"前后"}});
    fireEvent.click(screen.getByRole("button", {name:"开始语音输入"}));
    expect(instance?.start).toHaveBeenCalledTimes(1);
    fireEvent.change(input, {target:{value:"前修改后"}});
    input.setSelectionRange(1,3);
    const result = Object.assign([{transcript:"语音"}], {isFinal:true});
    act(() => instance?.onresult?.({resultIndex:0,results:[result]}));
    expect(input).toHaveValue("前修改语音后");
    expect(sessionStorage.getItem('ai-workbench:compose:["user-a","org-a","general"]')).toBe("前修改语音后");
    act(() => instance?.onresult?.({resultIndex:0,results:[result]}));
    expect(input).toHaveValue("前修改语音后");
    const storage = vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => { throw new Error("storage unavailable"); });
    const large = Object.assign([{transcript:"字".repeat(3000)}], {isFinal:true});
    act(() => instance?.onresult?.({resultIndex:1,results:[result,large]}));
    expect(input.value).toContain("字".repeat(3000));
    expect(screen.getByText(/无法保存草稿/)).toBeVisible();
    expect(screen.getByText(/不能超过 8192 字节/)).toBeVisible();
    storage.mockRestore();
    fireEvent.click(screen.getByRole("button", {name:"停止语音输入"}));
    expect(instance?.stop).toHaveBeenCalledTimes(1);
    const beforeFinal = input.value;
    act(() => instance?.onresult?.({resultIndex:2,results:[result,large,result]}));
    expect(input.value.length).toBe(beforeFinal.length + 2);
    act(() => instance?.onend?.());
    const beforeLate = input.value;
    act(() => instance?.onresult?.({resultIndex:3,results:[result,large,result,result]}));
    expect(input.value).toBe(beforeLate);
    expect(fixture.request.mock.calls.every(([value]) => value.route === "scope-options")).toBe(true);
  } finally { cleanup(); vi.unstubAllGlobals(); }
});

it("accepts a local draft before creating a general conversation without sending a message", async () => {
  fixture.context.aiWorkbenchExecutionKind = "product.content.localize";
  fixture.request.mockImplementation(async ({ route }) => route === "scope-options" ? scopeOptions : route === "conversation-create" ? { conversation } : Promise.reject(new Error("unexpected request")));
  render(<QueryClientProvider client={client}><ChatPage mode="new" /></QueryClientProvider>);
  await waitFor(()=>expect(screen.getByLabelText("需求")).toBeEnabled());
  fireEvent.change(screen.getByLabelText("需求"), { target: { value: "请整理商品描述" } });
  await waitFor(() => expect(screen.getByRole("button", { name: "开始对话" })).toBeEnabled());
  expect(screen.queryByLabelText("范围类型")).toBeNull();
  expect(fixture.request.mock.calls.every(([input]) => input.route === "scope-options")).toBe(true);
  fireEvent.click(screen.getByRole("button", { name: "开始对话" }));
  await waitFor(() => expect(fixture.push).toHaveBeenCalled());
  expect(fixture.request.mock.calls.map(([input]) => input.route)).toEqual(["scope-options", "conversation-create"]);
});

it("restores the unsent input after an unknown create and hands it to the same conversation", async () => {
  fixture.context.aiWorkbenchExecutionKind = "product.content.localize";
  let attempts = 0;
  fixture.request.mockImplementation(async ({ route }) => {
    if(route === "scope-options") return scopeOptions;
    if(route === "conversation-read") return {conversation,messages:[],proposals:[],before:""};
    if(route === "conversation-create") { if(attempts++ === 0) throw new AIWorkbenchError("OUTCOME_UNKNOWN"); return {conversation}; }
    throw new Error("unexpected request");
  });
  render(<QueryClientProvider client={client}><ChatPage mode="new" /></QueryClientProvider>);
  await waitFor(()=>expect(screen.getByLabelText("需求")).toBeEnabled());
  fireEvent.change(screen.getByLabelText("需求"),{target:{value:"请整理商品描述"}});
  await waitFor(()=>expect(screen.getByRole("button",{name:"开始对话"})).toBeEnabled());
  fireEvent.click(screen.getByRole("button",{name:"开始对话"}));
  await screen.findByRole("button",{name:"核对原创建请求"});
  expect(screen.getByLabelText("需求")).toHaveValue("请整理商品描述");
  fireEvent.click(screen.getByRole("button",{name:"核对原创建请求"}));
  await waitFor(()=>expect(fixture.push).toHaveBeenCalled());
  const creates=fixture.request.mock.calls.map(([input])=>input).filter(input=>input.route==="conversation-create");
  expect(creates[1].key).toBe(creates[0].key);
  cleanup(); render(tree());
  await waitFor(()=>expect(screen.getByLabelText("需求")).toHaveValue("请整理商品描述"));
  expect(fixture.request.mock.calls.every(([input])=>input.route!=="message")).toBe(true);
});

it.each(["project", "store"] as const)("uses only the authorized %s background without auto-sending", async kind => {
  fixture.context.aiWorkbenchExecutionKind="product.content.localize";
  const field=kind==="project"?"projectId":"storeId", value="fixture-one";
  fixture.search.set(field,value);
  const selection=kind==="project"?{kind,projectId:value}:{kind,storeId:value};
  fixture.request.mockImplementation(async ({route})=>route==="scope-options"?{options:[{scope:{version:"workbench-scope-v1",...selection},confirmation:scopeConfirmation,...(kind==="project"?{projectName:"测试项目",storeName:"关联店铺"}:{storeName:"测试店铺"})}],next:"",incomplete:false}:route==="conversation-create"?{conversation}:Promise.reject(new Error("unexpected request")));
  render(<QueryClientProvider client={client}><ChatPage mode="new" /></QueryClientProvider>);
  await waitFor(()=>expect(screen.getByLabelText("需求")).toBeEnabled());
  fireEvent.change(screen.getByLabelText("需求"),{target:{value:"请整理商品资料"}});
  await screen.findByText(kind==="project"?"工作背景：测试项目 · 关联店铺":"工作背景：测试店铺");
  fireEvent.click(screen.getByRole("button",{name:"开始对话"}));
  await waitFor(()=>expect(fixture.push).toHaveBeenCalled());
  const calls=fixture.request.mock.calls.map(([input])=>input);
  expect(calls.find(input=>input.route==="scope-options").path).toContain(`${field}=${value}`);
  expect(calls.find(input=>input.route==="conversation-create").body).toEqual({scopeSelection:selection,scopeConfirmation});
  expect(calls.every(input=>input.route!=="message")).toBe(true);
  fixture.search.delete(field);
});

it("keeps the input editable and does not create when browser storage fails", async () => {
  fixture.context.aiWorkbenchExecutionKind="product.content.localize";
  fixture.request.mockImplementation(async ({route})=>route==="scope-options"?scopeOptions:Promise.reject(new Error("unexpected request")));
  render(<QueryClientProvider client={client}><ChatPage mode="new" /></QueryClientProvider>);
  await waitFor(()=>expect(screen.getByLabelText("需求")).toBeEnabled());
  fireEvent.change(screen.getByLabelText("需求"),{target:{value:"保留的文字"}});
  await waitFor(()=>expect(screen.getByRole("button",{name:"开始对话"})).toBeEnabled());
  const original=Storage.prototype.setItem;
  const storage=vi.spyOn(Storage.prototype,"setItem").mockImplementation(function(this:Storage,key,value){if(key.includes("pending-create"))throw new Error("storage unavailable");return original.call(this,key,value);});
  fireEvent.click(screen.getByRole("button",{name:"开始对话"}));
  expect(await screen.findByText(/无法保存待恢复的请求/)).toBeVisible();
  expect(screen.getByLabelText("需求")).toHaveValue("保留的文字");
  expect(screen.getByLabelText("需求")).toBeEnabled();
  expect(fixture.request.mock.calls.every(([input])=>input.route!=="conversation-create"&&input.route!=="message")).toBe(true);
  storage.mockRestore();
});

it("retries an existing pre-update create with the original key", async () => {
  fixture.context.aiWorkbenchExecutionKind="product.content.localize";
  const body={scopeSelection:{kind:"general"},scopeConfirmation};
  sessionStorage.setItem("ai-workbench:pending-create:user-a:org-a",JSON.stringify({key:operationId,body}));
  fixture.request.mockImplementation(async ({route})=>route==="scope-options"?scopeOptions:route==="conversation-create"?{conversation}:Promise.reject(new Error("unexpected request")));
  render(<QueryClientProvider client={client}><ChatPage mode="new" /></QueryClientProvider>);
  const button=await screen.findByRole("button",{name:"核对原创建请求"});
  await waitFor(()=>expect(button).toBeEnabled()); fireEvent.click(button);
  await waitFor(()=>expect(fixture.push).toHaveBeenCalled());
  expect(fixture.request.mock.calls.find(([input])=>input.route==="conversation-create")?.[0]).toMatchObject({key:operationId,body});
});

it("fails closed on an unreadable pending create record", async () => {
  fixture.context.aiWorkbenchExecutionKind="product.content.localize";
  sessionStorage.setItem("ai-workbench:pending-create:user-a:org-a","{");
  fixture.request.mockImplementation(async ({route})=>route==="scope-options"?scopeOptions:Promise.reject(new Error("unexpected request")));
  render(<QueryClientProvider client={client}><ChatPage mode="new" /></QueryClientProvider>);
  await screen.findByText(/原创建请求记录不可读取/);
  fireEvent.change(screen.getByLabelText("需求"),{target:{value:"保留需求"}});
  expect(screen.getByRole("button",{name:"开始对话"})).toBeDisabled();
  expect(sessionStorage.getItem("ai-workbench:pending-create:user-a:org-a")).toBe("{");
  expect(fixture.request.mock.calls.every(([input])=>input.route!=="conversation-create")).toBe(true);
});

it("does not expose an ambiguous legacy pending record across colliding identities", async () => {
  fixture.context.aiWorkbenchExecutionKind="product.content.localize";
  fixture.context.user.id="u"; fixture.context.effectiveOrganization.id="o:x";
  sessionStorage.setItem("ai-workbench:pending-create:u:o:x",JSON.stringify({key:operationId,body:{scopeSelection:{kind:"general"},scopeConfirmation},draft:"别人的私密需求",context:"general"}));
  fixture.request.mockImplementation(async ({route})=>route==="scope-options"?scopeOptions:Promise.reject(new Error("unexpected request")));
  render(<QueryClientProvider client={client}><ChatPage mode="new" /></QueryClientProvider>);
  await screen.findByText(/身份无法确认/);
  expect(screen.queryByDisplayValue("别人的私密需求")).toBeNull();
  expect(screen.getByRole("button",{name:"开始对话"})).toBeDisabled();
  fixture.context.user.id="user-a";
});

it("does not reveal an ambiguous old pending message to a colliding identity", async () => {
  fixture.context.aiWorkbenchExecutionKind="product.content.localize";
  fixture.context.user.id="u";fixture.context.effectiveOrganization.id="o:x";
  sessionStorage.setItem(`ai-workbench:pending-message:u:o:x:${conversationId}`,JSON.stringify({key:operationId,body:{content:"私密旧消息",operationId,targetPlatform:"shein"}}));
  fixture.request.mockImplementation(async ({route})=>route==="conversation-read"?{conversation,messages:[],proposals:[],before:""}:Promise.reject(new Error("unexpected request")));
  render(tree());await screen.findByText(/原消息恢复记录不可读取或身份无法确认/);
  expect(screen.queryByDisplayValue("私密旧消息")).toBeNull();
  expect(fixture.request.mock.calls.every(([input])=>input.route!=="message")).toBe(true);
  fixture.context.user.id="user-a";
});

it("preserves a newer unsent draft when an older pending message completes", async () => {
  fixture.context.aiWorkbenchExecutionKind="product.content.localize";
  const draftKey=`ai-workbench:conversation-draft:${JSON.stringify(["user-a","org-a",conversationId])}`;
  sessionStorage.setItem(draftKey,JSON.stringify({id:conversationId,text:"新草稿 Y"}));
  sessionStorage.setItem(`ai-workbench:pending-message:user-a:org-a:${conversationId}`,JSON.stringify({key:operationId,body:{content:"旧消息 X",taskSelection:{subject:{kind:"own_product",id:operationId,catalogVersion:"3",publicationId:"effective-publication"},capability:"product.content.localize",parameters:{targetLanguage:"en-US",description:true}}}}));
  fixture.request.mockImplementation(async ({route})=>route==="conversation-read"?{conversation,messages:[],proposals:[],before:""}:route==="message"?{state:"COMPLETE"}:Promise.reject(new Error("unexpected request")));
  render(tree());
  const button=await screen.findByRole("button",{name:"重试同一消息"}); fireEvent.click(button);
  await screen.findByText("新草稿 Y");
  fireEvent.click(screen.getByRole("button",{name:"使用保留的需求"}));
  expect(screen.getByLabelText("需求")).toHaveValue("新草稿 Y");
  expect(JSON.parse(sessionStorage.getItem(draftKey)!).text).toBe("新草稿 Y");
});

it.each(["COMPLETE","FAILED_BEFORE_DISPATCH","PLANNER_INVALID_OUTPUT","PLANNER_UNKNOWN"] as const)("keeps a separate same-text draft after pending message ends as %s", async state => {
  fixture.context.aiWorkbenchExecutionKind="product.content.localize";
  const key=`ai-workbench:conversation-draft:${JSON.stringify(["user-a","org-a",conversationId])}`;
  sessionStorage.setItem(key,JSON.stringify({id:conversationId,text:"相同内容"}));
  sessionStorage.setItem(`ai-workbench:pending-message:user-a:org-a:${conversationId}`,JSON.stringify({key:operationId,body:{content:"相同内容",taskSelection:{subject:{kind:"own_product",id:operationId,catalogVersion:"3",publicationId:"effective-publication"},capability:"product.content.localize",parameters:{targetLanguage:"en-US",description:true}}}}));
  fixture.request.mockImplementation(async ({route})=>route==="conversation-read"?{conversation,messages:[],proposals:[],before:""}:route==="message"?{state}:Promise.reject(new Error("unexpected request")));
  render(tree()); fireEvent.click(await screen.findByRole("button",{name:"重试同一消息"}));
  await waitFor(()=>expect(screen.queryByRole("button",{name:"重试同一消息"})).toBeNull());
  expect(JSON.parse(sessionStorage.getItem(key)!).id).toBe(conversationId);
  if(state!=="COMPLETE"){
    fireEvent.change(screen.getByLabelText("需求"),{target:{value:"旧需求的修改"}});
    cleanup(); render(tree());
    await waitFor(()=>expect(screen.getByLabelText("需求")).toHaveValue("旧需求的修改"));
    expect(screen.getByText("相同内容")).toBeVisible();
  }
});

it.each(["INVALID_REQUEST","PROPOSAL_STALE"] as const)("preserves both drafts after the old message is explicitly rejected with %s", async code => {
  fixture.context.aiWorkbenchExecutionKind="product.content.localize";
  const key=`ai-workbench:conversation-draft:${JSON.stringify(["user-a","org-a",conversationId])}`;
  sessionStorage.setItem(key,JSON.stringify({id:conversationId,text:"相同内容"}));
  sessionStorage.setItem(`ai-workbench:pending-message:user-a:org-a:${conversationId}`,JSON.stringify({key:operationId,body:{content:"相同内容",taskSelection:{subject:{kind:"own_product",id:operationId,catalogVersion:"3",publicationId:"effective-publication"},capability:"product.content.localize",parameters:{targetLanguage:"en-US",description:true}}}}));
  fixture.request.mockImplementation(async ({route})=>route==="conversation-read"?{conversation,messages:[],proposals:[],before:""}:Promise.reject(new AIWorkbenchError(code,false)));
  render(tree()); fireEvent.click(await screen.findByRole("button",{name:"重试同一消息"}));
  await waitFor(()=>expect(screen.queryByRole("button",{name:"重试同一消息"})).toBeNull());
  fireEvent.change(screen.getByLabelText("需求"),{target:{value:"修改旧需求"}});
  cleanup(); render(tree());
  await waitFor(()=>expect(screen.getByLabelText("需求")).toHaveValue("修改旧需求"));
  expect(screen.getByText("相同内容")).toBeVisible();
  expect(JSON.parse(sessionStorage.getItem(key)!).id).toBe(conversationId);
});

it("keeps the original message key when completed draft cleanup fails", async () => {
  fixture.context.aiWorkbenchExecutionKind="product.content.localize";
  const draftKey=`ai-workbench:conversation-draft:${JSON.stringify(["user-a","org-a",conversationId])}`;
  const pendingKey=`ai-workbench:pending-message-v2:${JSON.stringify(["user-a","org-a",conversationId])}`;
  sessionStorage.setItem(draftKey,JSON.stringify({id:operationId,text:"待核对文字"}));
  sessionStorage.setItem(pendingKey,JSON.stringify({key:conversationId,owner:{userId:"user-a",organizationId:"org-a"},conversationId,draftId:operationId,draftSlot:"handoff",body:{content:"待核对文字",taskSelection:{subject:{kind:"own_product",id:operationId,catalogVersion:"3",publicationId:"effective-publication"},capability:"product.content.localize",parameters:{targetLanguage:"en-US",description:true}}}}));
  fixture.request.mockImplementation(async ({route})=>route==="conversation-read"?{conversation,messages:[],proposals:[],before:""}:route==="message"?{state:"COMPLETE"}:Promise.reject(new Error("unexpected request")));
  render(tree()); const original=Storage.prototype.removeItem;
  const storage=vi.spyOn(Storage.prototype,"removeItem").mockImplementation(function(this:Storage,k){if(k===draftKey)throw new Error("unavailable");return original.call(this,k);});
  const button=await screen.findByRole("button",{name:"重试同一消息"});await waitFor(()=>expect(button).toBeEnabled());fireEvent.click(button);
  await screen.findByText(/无法完成草稿清理/);
  expect(JSON.parse(sessionStorage.getItem(pendingKey)!).key).toBe(conversationId);
  storage.mockRestore();cleanup();render(tree());
  expect(await screen.findByRole("button",{name:"重试同一消息"})).toBeVisible();
});

it("blocks retry while draft storage cannot be read and keeps the original message", async () => {
  fixture.context.aiWorkbenchExecutionKind="product.content.localize";
  const key=`ai-workbench:pending-message:user-a:org-a:${conversationId}`;
  const draftKey=`ai-workbench:conversation-draft:${JSON.stringify(["user-a","org-a",conversationId])}`;
  sessionStorage.setItem(key,JSON.stringify({key:operationId,body:{content:"旧需求",taskSelection:{subject:{kind:"own_product",id:operationId,catalogVersion:"3",publicationId:"effective-publication"},capability:"product.content.localize",parameters:{targetLanguage:"en-US",description:true}}}}));
  fixture.request.mockImplementation(async ({route})=>route==="conversation-read"?{conversation,messages:[],proposals:[],before:""}:Promise.reject(new Error("unexpected request")));
  const original=Storage.prototype.getItem;
  const storage=vi.spyOn(Storage.prototype,"getItem").mockImplementation(function(this:Storage,k){if(k===draftKey)throw new Error("unavailable");return original.call(this,k);});
  render(tree());await screen.findByText(/本地草稿不可读取/);
  expect(screen.getByRole("button",{name:"重试同一消息"})).toBeDisabled();
  expect(sessionStorage.getItem(key)).not.toBeNull(); storage.mockRestore();
  fireEvent.click(screen.getByRole("button",{name:"重新读取草稿"}));
  await waitFor(()=>expect(screen.getByRole("button",{name:"重试同一消息"})).toBeEnabled());
  expect(fixture.request.mock.calls.every(([input])=>input.route!=="message")).toBe(true);
});

it("does not permit editing until an unreadable existing draft is recovered", async () => {
  fixture.context.aiWorkbenchExecutionKind="product.content.localize";
  const key=`ai-workbench:conversation-draft:${JSON.stringify(["user-a","org-a",conversationId])}`;
  sessionStorage.setItem(key,JSON.stringify({id:operationId,text:"保留的 Y"}));
  fixture.request.mockImplementation(async ({route})=>route==="conversation-read"?{conversation,messages:[],proposals:[],before:""}:Promise.reject(new Error("unexpected request")));
  const original=Storage.prototype.getItem;
  const storage=vi.spyOn(Storage.prototype,"getItem").mockImplementation(function(this:Storage,k){if(k===key)throw new Error("unavailable");return original.call(this,k);});
  render(tree());await screen.findByText(/本地草稿不可读取/);
  expect(screen.getByLabelText("需求")).toBeDisabled();storage.mockRestore();
  fireEvent.click(screen.getByRole("button",{name:"重新读取草稿"}));
  await waitFor(()=>expect(screen.getByLabelText("需求")).toHaveValue("保留的 Y"));
  expect(screen.getByLabelText("需求")).toBeEnabled();
  expect(JSON.parse(sessionStorage.getItem(key)!).text).toBe("保留的 Y");
});

it.each(["FAILED_BEFORE_DISPATCH","PLANNER_INVALID_OUTPUT","PLANNER_UNKNOWN"] as const)("retains an old pending input after %s and refresh", async state => {
  fixture.context.aiWorkbenchExecutionKind="product.content.localize";
  sessionStorage.setItem(`ai-workbench:pending-message:user-a:org-a:${conversationId}`,JSON.stringify({key:operationId,body:{content:"旧需求",taskSelection:{subject:{kind:"own_product",id:operationId,catalogVersion:"3",publicationId:"effective-publication"},capability:"product.content.localize",parameters:{targetLanguage:"en-US",description:true}}}}));
  fixture.request.mockImplementation(async ({route})=>route==="conversation-read"?{conversation,messages:[],proposals:[],before:""}:route==="message"?{state}:Promise.reject(new Error("unexpected request")));
  render(tree());const button=await screen.findByRole("button",{name:"重试同一消息"});await waitFor(()=>expect(button).toBeEnabled());fireEvent.click(button);
  await waitFor(()=>expect(screen.queryByRole("button",{name:"重试同一消息"})).toBeNull());
  cleanup();render(tree());await waitFor(()=>expect(screen.getByLabelText("需求")).toHaveValue("旧需求"));
});

it("does not reveal a stored project name after its scope becomes unreadable", async () => {
  fixture.context.aiWorkbenchExecutionKind="product.content.localize";
  fixture.search.set("projectId","fixture-one");
  sessionStorage.setItem(`ai-workbench:pending-create-v2:${JSON.stringify(["user-a","org-a"])}`,JSON.stringify({key:operationId,body:{scopeSelection:{kind:"project",projectId:"fixture-one"},scopeConfirmation},draft:"尚未发送",context:"project:fixture-one",owner:{userId:"user-a",organizationId:"org-a"},background:"保密项目名称"}));
  fixture.request.mockRejectedValue(new AIWorkbenchError("FORBIDDEN"));
  render(<QueryClientProvider client={client}><ChatPage mode="new" /></QueryClientProvider>);
  await screen.findByText(/工作背景暂不可用/);
  expect(screen.queryByText(/保密项目名称/)).toBeNull();
  expect(screen.getByText(/原请求待核对/)).toBeVisible();
  fixture.search.delete("projectId");
});

it("warns if editing a handed-off draft cannot be stored", async () => {
  fixture.context.aiWorkbenchExecutionKind="product.content.localize";
  const key=`ai-workbench:conversation-draft:${JSON.stringify(["user-a","org-a",conversationId])}`;
  sessionStorage.setItem(key,JSON.stringify({id:operationId,text:"原草稿"}));
  fixture.request.mockImplementation(async ({route})=>route==="conversation-read"?{conversation,messages:[],proposals:[],before:""}:Promise.reject(new Error("unexpected request")));
  render(tree()); await waitFor(()=>expect(screen.getByLabelText("需求")).toHaveValue("原草稿"));
  const original=Storage.prototype.setItem;
  const storage=vi.spyOn(Storage.prototype,"setItem").mockImplementation(function(this:Storage,k,v){if(k===key)throw new Error("quota");return original.call(this,k,v);});
  fireEvent.change(screen.getByLabelText("需求"),{target:{value:"修改后的草稿"}});
  expect(screen.getByText(/刷新会丢失修改/)).toBeVisible();
  expect(screen.getByLabelText("需求")).toHaveValue("修改后的草稿");
  expect(JSON.parse(sessionStorage.getItem(key)!).text).toBe("原草稿"); storage.mockRestore();
});

beforeEach(() => {
  client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  fixture.context.effectiveOrganization = { id: "org-a", capabilities: { "workbench.chat.use": true } };
  fixture.context.roles = ["listingkit_operator"]; fixture.context.permissions = ["product_sourcing.write","local_agent.write","listingkit.image_agent.read","listingkit.image_agent.write","workbench.agent.read","workbench.agent.use","workbench.chat.read","workbench.chat.use","workbench.task.read","workbench.knowledge.read","workbench.store.read","workbench.store.create","workbench.store.update","workbench.store.lifecycle","workbench.source_account.read","workbench.source_account.manage","workbench.organization_member.read","workbench.commercial.read"];
  fixture.context.isSwitching = false;
  fixture.context.aiWorkbenchAvailable = true;
  fixture.context.effectiveMemberId = "member-a";
  fixture.context.aiWorkbenchExecutionKind=undefined;fixture.context.aiWorkbenchExecutionReadiness="AVAILABLE";
  fixture.context.aiWorkbenchPlanningReadiness = "AVAILABLE";
  fixture.context.aiWorkbenchStoreRouteReadiness = "AVAILABLE";
  fixture.context.aiWorkbenchTitleReadiness = "AVAILABLE";
  fixture.request.mockReset(); fixture.push.mockReset(); fixture.templates.mockReset();fixture.knowledge.mockReset();sessionStorage.clear();
  fixture.templates.mockResolvedValue({items:[],nextCursor:""});
  fixture.knowledge.mockResolvedValue({items:[],pagination:{page:1,pageSize:100,total:0}});
  fixture.request.mockImplementation(async ({ route }) => {
    if (route === "conversation-read") return { conversation, messages: [], proposals: [], before: "" };
    throw new AIWorkbenchError("OUTCOME_UNKNOWN");
  });
});
afterEach(() => { cleanup(); client.clear(); sessionStorage.clear(); });
it("preserves a valid acquisition context across conversation view links", () => {
  fixture.search.set("operationId",operationId);
  render(<QueryClientProvider client={client}><ChatPage mode="new" /></QueryClientProvider>);
  const nav=within(screen.getByRole("navigation",{name:"会话视图"}));
  for(const [view,label] of [["new","新建会话"],["recent","最近会话"],["saved","收藏会话"],["archived","归档会话"]]) expect(nav.getByRole("link",{name:label})).toHaveAttribute("href",`/workbench/ai/chat/${view}?operationId=${operationId}`);
  fixture.search.delete("operationId");
});
it.each(["home", "new", "recent", "saved", "archived"] as const)("provides all conversation views from %s", mode => {
  fixture.request.mockImplementation(async ({ route }) => route === "conversation-list" ? {conversations:[],next:""} : route === "scope-options" ? scopeOptions : Promise.reject(new AIWorkbenchError("INVALID_REQUEST")));
  render(<QueryClientProvider client={client}><ChatPage mode={mode} /></QueryClientProvider>);
  const nav = screen.getByRole("navigation", {name:"会话视图"});
  for (const [view,label] of [["new","新建会话"],["recent","最近会话"],["saved","收藏会话"],["archived","归档会话"]]) {
    const link=within(nav).getByRole("link",{name:label});
    expect(link).toHaveAttribute("href",`/workbench/ai/chat/${view}`);
    if (mode===view) expect(link).toHaveAttribute("aria-current","page");
    else expect(link).not.toHaveAttribute("aria-current");
  }
  expect(fixture.request.mock.calls.every(([input]) => input.route !== "conversation-create")).toBe(true);
});
it("keeps new conversation creation separate from the history list", async () => {
  fixture.request.mockImplementation(async ({ route }) => {
    if (route === "conversation-list") return { conversations: [{ ...conversation, Title: "已有讨论" }], next: "" };
    if (route === "scope-options") return scopeOptions;
    if (route === "conversation-create") return { conversation };
    throw new AIWorkbenchError("INVALID_REQUEST");
  });
  render(<QueryClientProvider client={client}><ChatPage mode="new" /></QueryClientProvider>);
  expect(screen.getByRole("heading", { name: "开始商品任务会话" })).toBeVisible();
  expect(screen.queryByRole("region", { name: "会话列表" })).toBeNull();
  await selectGeneralScope();
  expect(fixture.request.mock.calls.every(([input])=>input.route === "scope-options")).toBe(true);
  fireEvent.click(screen.getByRole("button", { name: "新建会话" }));
  await waitFor(() => expect(fixture.push).toHaveBeenCalled());
  expect(fixture.request.mock.calls.map(([input]) => input.route)).toEqual(["scope-options", "conversation-create"]);
  expect(fixture.request.mock.calls.at(-1)?.[0].body).toEqual({scopeSelection:{kind:"general"},scopeConfirmation});
});

it("unlocks selection after proof that an unavailable scope never created a conversation", async () => {
  const post = vi.fn().mockRejectedValueOnce(new AIWorkbenchError("NOT_FOUND", undefined, false)).mockResolvedValue({conversation});
  fixture.request.mockImplementation(async input => input.route === "scope-options" ? scopeOptions : post(input));
  render(<QueryClientProvider client={client}><ChatPage mode="new" /></QueryClientProvider>);
  await selectGeneralScope();
  fireEvent.click(screen.getByRole("button",{name:"新建会话"}));
  await waitFor(()=>expect(post).toHaveBeenCalledTimes(1));
  await waitFor(()=>expect(screen.getByLabelText("范围类型")).toBeEnabled());
  await selectGeneralScope();
  fireEvent.click(screen.getByRole("button",{name:"新建会话"}));
  await waitFor(()=>expect(fixture.push).toHaveBeenCalled());
  expect(post.mock.calls[1][0].key).not.toEqual(post.mock.calls[0][0].key);
});

it("keeps the exact create payload when the outcome is unknown", async () => {
  const post = vi.fn().mockRejectedValueOnce(new AIWorkbenchError("OUTCOME_UNKNOWN")).mockResolvedValue({conversation});
  fixture.request.mockImplementation(async input => input.route === "scope-options" ? scopeOptions : post(input));
  render(<QueryClientProvider client={client}><ChatPage mode="new" /></QueryClientProvider>);
  await selectGeneralScope();
  fireEvent.click(screen.getByRole("button",{name:"新建会话"}));
  fireEvent.click(await screen.findByRole("button",{name:"核对原创建请求"}));
  await waitFor(()=>expect(fixture.push).toHaveBeenCalled());
  expect(post.mock.calls[1][0]).toMatchObject({key:post.mock.calls[0][0].key,body:post.mock.calls[0][0].body});
});

it.each(["recent", "saved"] as const)("filters only loaded %s titles while retaining pagination and its scope", async mode => {
  const first = { ...conversation, Title: "中文内容整理", Favorite: mode === "saved" };
  const older = { ...first, ID: "d5d9d1ca-1db3-43af-9649-dcdf3663745b", Title: "English content" };
  fixture.request.mockImplementation(async ({ path }) => path.includes("after=older")
    ? { conversations: [older], next: "" } : { conversations: [first], next: "older" });
  render(<QueryClientProvider client={client}><ChatPage mode={mode} /></QueryClientProvider>);
  await screen.findByRole("link", { name: /中文内容整理/ });
  fireEvent.change(screen.getByRole("searchbox", { name: "筛选已加载的会话标题" }), { target: { value: " ENGLISH " } });
  expect(screen.queryByRole("link", { name: /中文内容整理/ })).toBeNull();
  expect(screen.getByText("已加载的会话中没有匹配标题")).toBeVisible();
  fireEvent.click(screen.getByRole("button", { name: "加载更早会话" }));
  expect(await screen.findByRole("link", { name: /English content/ })).toBeVisible();
  expect(fixture.request.mock.calls.filter(([input]) => input.route === "conversation-list")).toHaveLength(2);
  expect(fixture.request.mock.calls.at(-1)![0]).toMatchObject({ scope: { userId: "user-a", organizationId: "org-a" } });
  if (mode === "saved") expect(fixture.request.mock.calls.at(-1)![0].path).toContain("saved=true");
  fireEvent.click(screen.getByRole("button", { name: "清空筛选" }));
  expect(screen.getByRole("link", { name: /中文内容整理/ })).toBeVisible();
  expect(screen.getByRole("link", { name: /English content/ })).toBeVisible();
});

it.each(["INVALID_REQUEST","PROPOSAL_STALE"])("releases a definitively rejected %s message for correction with a new key",async(code)=>{
 fixture.request.mockImplementation(async({route})=>{if(route==="conversation-read")return {conversation,messages:[],proposals:[],before:""};throw new AIWorkbenchError(code,false)});
 const view=render(tree());await screen.findByText("开始讨论");
 fireEvent.change(screen.getByLabelText("采集操作 ID"),{target:{value:operationId}});fireEvent.change(screen.getByLabelText("需求"),{target:{value:"保留需求"}});
 fireEvent.click(screen.getByRole("button",{name:"发送消息"}));
 await waitFor(()=>expect(screen.getByLabelText("需求")).toBeEnabled());expect(screen.getByLabelText("需求")).toHaveValue("保留需求");
 const first=fixture.request.mock.calls.find(([v])=>v.route==="message")![0];
 view.unmount();render(tree());await screen.findByText("开始讨论");await waitFor(()=>expect(screen.getByLabelText("需求")).toBeEnabled());
 fireEvent.change(screen.getByLabelText("采集操作 ID"),{target:{value:operationId}});fireEvent.change(screen.getByLabelText("需求"),{target:{value:"修正需求"}});await waitFor(()=>expect(screen.getByRole("button",{name:"发送消息"})).toBeEnabled());fireEvent.click(screen.getByRole("button",{name:"发送消息"}));
 await waitFor(()=>expect(fixture.request.mock.calls.filter(([v])=>v.route==="message")).toHaveLength(2));
 expect(fixture.request.mock.calls.filter(([v])=>v.route==="message")[1][0].key).not.toBe(first.key);
});
it("opens optional settings without asking for internal identifiers", async () => {
  render(tree());
  await screen.findByText("开始讨论");
  const summary = screen.getByText("模板与知识库（可选）");
  const disclosure = summary.closest("details")!;
  fireEvent.click(summary);
  await waitFor(() => expect(disclosure).toHaveAttribute("open"));
  expect(disclosure).toHaveAttribute("open");
  expect(screen.getByLabelText("知识库")).toBeVisible();
  expect(screen.queryByLabelText("知识库 ID（可选）")).not.toBeInTheDocument();
});
it("sends the chosen template version and knowledge without exposing their identifiers",async()=>{
 const templateId="22222222-2222-4222-8222-222222222222",baseId="33333333-3333-4333-8333-333333333333";
 fixture.templates.mockResolvedValue({items:[{templateId,agentId:"product.title.agent",name:"标题模板",version:"2",lifecycle:"ACTIVE",targetPlatform:"shein"}],nextCursor:""});
 fixture.knowledge.mockResolvedValue({items:[{id:baseId,name:"品牌知识",state:"ACTIVE"}],pagination:{page:1,pageSize:100,total:1}});
 render(tree());await screen.findByText("开始讨论");
 fireEvent.change(screen.getByLabelText("采集操作 ID"),{target:{value:operationId}});
 fireEvent.change(screen.getByLabelText("需求"),{target:{value:"整理商品标题"}});
 fireEvent.click(screen.getByText("模板与知识库（可选）"));
 expect(await screen.findByRole("option",{name:"标题模板"})).toBeVisible();
 expect(await screen.findByRole("option",{name:"品牌知识"})).toBeVisible();
 fireEvent.change(screen.getByLabelText("模板"),{target:{value:`${templateId}@2`}});
 fireEvent.change(screen.getByLabelText("知识库"),{target:{value:baseId}});
 await waitFor(()=>expect(screen.getByRole("button",{name:"发送消息"})).toBeEnabled());
 fireEvent.click(screen.getByRole("button",{name:"发送消息"}));
 await waitFor(()=>expect(fixture.request.mock.calls.some(([v])=>v.route==="message")).toBe(true));
 expect(fixture.request.mock.calls.find(([v])=>v.route==="message")![0].body).toMatchObject({templateId,templateRevision:"2",knowledgeBaseId:baseId});
 expect(screen.queryByText(templateId)).not.toBeInTheDocument();
});
it.each(["INVALID_REQUEST","PROPOSAL_STALE"])("retains the original intent for %s without owner rejection evidence",async(code)=>{
 fixture.request.mockImplementation(async({route})=>{if(route==="conversation-read")return {conversation,messages:[],proposals:[],before:""};throw new AIWorkbenchError(code)});
 render(tree());await screen.findByText("开始讨论");fireEvent.change(screen.getByLabelText("采集操作 ID"),{target:{value:operationId}});fireEvent.change(screen.getByLabelText("需求"),{target:{value:"原需求"}});fireEvent.click(screen.getByRole("button",{name:"发送消息"}));
 await screen.findByRole("button",{name:"重试同一消息"});expect(screen.getByLabelText("需求")).toBeDisabled();
 const original=fixture.request.mock.calls.find(([v])=>v.route==="message")![0];fireEvent.click(screen.getByRole("button",{name:"重试同一消息"}));
 await waitFor(()=>expect(fixture.request.mock.calls.filter(([v])=>v.route==="message")).toHaveLength(2));expect(fixture.request.mock.calls.filter(([v])=>v.route==="message")[1][0]).toMatchObject({key:original.key,body:original.body});
});

it("shows the unavailable state without sending Chat requests when the module is absent", () => {
  fixture.context.aiWorkbenchAvailable = false;
  render(tree());
  expect(screen.getByText("当前应用未启用硕米 Chat 与业务任务。")).toBeVisible();
  expect(fixture.request).not.toHaveBeenCalled();
});

it("allows conversation creation but does not offer planning while the model needs configuration", async () => {
  fixture.context.aiWorkbenchPlanningReadiness = "NEEDS_CONFIGURATION";
  const view = render(<QueryClientProvider client={client}><ChatPage mode="home" /></QueryClientProvider>);
  await screen.findByText("开始一项新需求");
  expect(screen.getByRole("button", { name: "进入新建会话 →" })).toBeEnabled();
  expect(screen.getByText(/当前企业的规划模型需要配置/)).toBeVisible();
  view.rerender(tree());
  await screen.findByText("开始讨论");
  expect(screen.queryByRole("button", { name: "发送消息" })).toBeNull();
  expect(screen.queryByLabelText("需求")).toBeNull();
  expect(fixture.request.mock.calls.every(call => call[0].route === "conversation-list" || call[0].route === "conversation-read")).toBe(true);
});

it.each(["AVAILABLE", "UNAVAILABLE"])("creates a real conversation with planning readiness %s", async readiness => {
  fixture.context.aiWorkbenchPlanningReadiness = readiness;
  fixture.search = new URLSearchParams(`operationId=${operationId}`);
  fixture.request.mockImplementation(async ({ route }) => {
    if (route === "conversation-list") return { conversations: [], next: "" };
    if (route === "scope-options") return scopeOptions;
    if (route === "conversation-create") return { conversation };
    throw new AIWorkbenchError("INVALID_REQUEST");
  });
  render(<QueryClientProvider client={client}><ChatPage mode="new" /></QueryClientProvider>);
  await selectGeneralScope();
  fireEvent.click(await screen.findByRole("button", { name: "新建会话" }));
  await waitFor(() => expect(fixture.push).toHaveBeenCalledWith(`/workbench/ai/chat/${conversationId}?operationId=${operationId}`));
  fixture.search = new URLSearchParams();
});

it("loads favorites before pagination so an older saved conversation is visible", async () => {
  const saved = { ...conversation, ID: "d5d9d1ca-1db3-43af-9649-dcdf3663745b", Title: "重要会话", Favorite: true };
  fixture.request.mockImplementation(async ({ route, path }) => {
    if (route !== "conversation-list") throw new AIWorkbenchError("INVALID_REQUEST");
    return path.includes("saved=true") ? { conversations: [saved], next: "" }
      : { conversations: [conversation], next: conversation.ID };
  });
  render(<QueryClientProvider client={client}><ChatPage mode="saved" /></QueryClientProvider>);
  expect(await screen.findByRole("link", { name: /重要会话/ })).toBeVisible();
  expect(fixture.request).toHaveBeenCalledWith(expect.objectContaining({ path: expect.stringContaining("saved=true") }));
});

it("lists archived conversations through the scoped archive filter", async () => {
  const archived = { ...conversation, Title: "已归档的标题讨论", Archived: true };
  fixture.request.mockImplementation(async ({ route, path }) => {
    if (route !== "conversation-list") throw new AIWorkbenchError("INVALID_REQUEST");
    return path.includes("archived=true") ? { conversations: [archived], next: "" }
      : { conversations: [], next: "" };
  });
  render(<QueryClientProvider client={client}><ChatPage mode="archived" /></QueryClientProvider>);
  expect(await screen.findByRole("link", { name: /已归档的标题讨论/ })).toHaveAttribute("href", `/workbench/ai/chat/${conversationId}`);
  expect(screen.getByRole("link", { name: "返回最近会话" })).toHaveAttribute("href", "/workbench/ai/chat/recent");
  expect(fixture.request).toHaveBeenCalledWith(expect.objectContaining({ path: expect.stringContaining("archived=true") }));
});

it.each(["recent", "saved", "archived"] as const)("keeps the first %s page visible after loading older conversations", async mode => {
  const first = { ...conversation, ID: "d5d9d1ca-1db3-43af-9649-dcdf3663745b", Title: "较新的会话", Favorite: mode === "saved", Archived: mode === "archived" };
  const older = { ...first, ID: "5892474d-1c8a-47e4-9550-45ed946e8197", Title: "较早的会话" };
  fixture.request.mockImplementation(async ({ route, path }) => {
    if (route !== "conversation-list") throw new AIWorkbenchError("INVALID_REQUEST");
    return path.includes("after=older") ? { conversations: [older], next: "" } : { conversations: [first], next: "older" };
  });
  render(<QueryClientProvider client={client}><ChatPage mode={mode} /></QueryClientProvider>);
  expect(await screen.findByRole("link", { name: /较新的会话/ })).toBeVisible();
  fireEvent.click(screen.getByRole("button", { name: "加载更早会话" }));
  expect(await screen.findByRole("link", { name: /较早的会话/ })).toBeVisible();
  expect(screen.getByRole("link", { name: /较新的会话/ })).toBeVisible();
  fireEvent.click(screen.getByRole("button", { name: "刷新" }));
  await waitFor(() => expect(fixture.request.mock.calls.filter(call => call[0].route === "conversation-list" && !call[0].path.includes("after=")).length).toBeGreaterThan(1));
  expect(screen.getByRole("link", { name: /较新的会话/ })).toBeVisible();
});

it("keeps loaded conversations visible when an older page fails", async () => {
  const first = { ...conversation, Title: "已有会话" };
  fixture.request.mockImplementation(async ({ route, path }) => {
    if (route !== "conversation-list") throw new AIWorkbenchError("INVALID_REQUEST");
    if (path.includes("after=older")) throw new AIWorkbenchError("DEPENDENCY_UNAVAILABLE");
    return { conversations: [first], next: "older" };
  });
  render(<QueryClientProvider client={client}><ChatPage mode="recent" /></QueryClientProvider>);
  expect(await screen.findByRole("link", { name: /已有会话/ })).toBeVisible();
  fireEvent.click(screen.getByRole("button", { name: "加载更早会话" }));
  expect(await screen.findByText(/服务暂不可用/)).toBeVisible();
  expect(screen.getByRole("link", { name: /已有会话/ })).toBeVisible();
});

it("retains the newest message and proposal while loading older messages", async () => {
  const newest = { ID: operationId, ConversationID: conversationId, Sequence: 102, Author: "USER", Content: "最新需求", CreatedAt: "2026-10-03T00:00:00Z" };
  const oldest = { ...newest, ID: "11111111-1111-4111-8111-111111111112", Sequence: 1, Content: "最早需求" };
  const proposal = { id: "d5d9d1ca-1db3-43af-9649-dcdf3663745b", digest: "a".repeat(64), sourceSequence: 102,
    goalSummary: "最新标题方案", productKey: "product", targetPlatform: "shein", humanReviewRequired: true, detailsAvailable: true, titleProfileReady: true, executionAuthorized: true };
  fixture.request.mockImplementation(async ({ route, path }) => {
    if (route !== "conversation-read") throw new AIWorkbenchError("INVALID_REQUEST");
    return path.includes("before=53") ? { conversation, messages: [oldest], proposals: [proposal], before: "" }
      : { conversation, messages: [newest], proposals: [proposal], before: "53" };
  });
  render(tree());
  expect(await screen.findByText("最新需求")).toBeVisible();
  expect(screen.getByText("最新标题方案")).toBeVisible();
  fireEvent.click(screen.getByRole("button", { name: "加载更早消息" }));
  expect(await screen.findByText("最早需求")).toBeVisible();
  expect(screen.getByText("最新需求")).toBeVisible();
  expect(screen.getByText("最新标题方案")).toBeVisible();
  expect(await screen.findByRole("button", { name: "确认并创建任务" })).toBeEnabled();
});

it("gates an existing proposal on title readiness independently of planning readiness", async () => {
  fixture.context.aiWorkbenchPlanningReadiness = "NEEDS_CONFIGURATION";
  fixture.context.aiWorkbenchTitleReadiness = "NEEDS_CONFIGURATION";
  const user = { ID: operationId, ConversationID: conversationId, Sequence: 1, Author: "USER", Content: "已有需求", CreatedAt: "2026-10-03T00:00:00Z" };
  const proposal = { id: "d5d9d1ca-1db3-43af-9649-dcdf3663745b", digest: "a".repeat(64), sourceSequence: 1,
    goalSummary: "已准备的方案", productKey: "product", targetPlatform: "shein", humanReviewRequired: true, detailsAvailable: true, titleProfileReady: true, executionAuthorized: true };
  fixture.request.mockImplementation(async ({ route }) => route === "conversation-read"
    ? { conversation, messages: [user], proposals: [proposal], before: "" }
    : Promise.reject(new AIWorkbenchError("INVALID_REQUEST")));
  const view = render(tree());
  expect(await screen.findByRole("button", { name: "确认并创建任务" })).toBeDisabled();
  expect(screen.getByText(/执行模型需要配置/)).toBeVisible();
  fixture.context.aiWorkbenchTitleReadiness = "AVAILABLE";
  view.rerender(tree());
  expect(await screen.findByRole("button", { name: "确认并创建任务" })).toBeEnabled();
  expect(screen.queryByRole("button", { name: "发送消息" })).toBeNull();
});

it("disables confirmation when the proposal's frozen title profile no longer matches the ready route", async () => {
  const proposal = { id: "d5d9d1ca-1db3-43af-9649-dcdf3663745b", digest: "a".repeat(64), sourceSequence: 1,
    goalSummary: "旧模型方案", productKey: "product", targetPlatform: "shein", humanReviewRequired: true,
    detailsAvailable: true, titleProfileReady: false, executionAuthorized: true };
  fixture.request.mockImplementation(async ({ route }) => route === "conversation-read"
    ? { conversation, messages: [{ ID: operationId, ConversationID: conversationId, Sequence: 1, Author: "USER", Content: "需求", CreatedAt: "2026-10-03T00:00:00Z" }], proposals: [proposal], before: "" }
    : Promise.reject(new AIWorkbenchError("INVALID_REQUEST")));
  render(tree());
  expect(await screen.findByRole("button", { name: "确认并创建任务" })).toBeDisabled();
  expect(screen.getByText(/提案的执行配置已变化/)).toBeVisible();
});

it("keeps planning available but disables execution when this actor lacks the current enterprise execution role", async () => {
  const proposal = { id: "d5d9d1ca-1db3-43af-9649-dcdf3663745b", digest: "a".repeat(64), sourceSequence: 1,
    goalSummary: "有规划权限的方案", productKey: "product", targetPlatform: "shein", humanReviewRequired: true,
    detailsAvailable: true, titleProfileReady: true, executionAuthorized: false };
  fixture.request.mockImplementation(async ({ route }) => route === "conversation-read"
    ? { conversation, messages: [{ ID: operationId, ConversationID: conversationId, Sequence: 1, Author: "USER", Content: "需求", CreatedAt: "2026-10-03T00:00:00Z" }], proposals: [proposal], before: "" }
    : Promise.reject(new AIWorkbenchError("FORBIDDEN")));
  render(tree());
  expect(await screen.findByRole("button", { name: "确认并创建任务" })).toBeDisabled();
  expect(screen.getByText(/需要当前企业的执行权限/)).toBeVisible();
  expect(screen.getByRole("button", { name: "发送消息" })).toBeVisible();
  expect(fixture.request).toHaveBeenCalledTimes(1);
});

it("shows chat history without mutation controls when the current organization lacks chat use", async () => {
  fixture.context.roles = ["listingkit_viewer"]; fixture.context.permissions = ["workbench.task.read","workbench.store.read","workbench.source_account.read","workbench.organization_member.read","workbench.commercial.read"];
  fixture.context.effectiveOrganization = { id: "org-a", capabilities: { "workbench.chat.use": false } };
  const proposal = { id: "d5d9d1ca-1db3-43af-9649-dcdf3663745b", digest: "a".repeat(64), sourceSequence: 1,
    goalSummary: "只读方案", productKey: "product", targetPlatform: "shein", humanReviewRequired: true, detailsAvailable: true, titleProfileReady: true, executionAuthorized: false };
  fixture.request.mockImplementation(async ({ route }) => route === "conversation-read" ? { conversation,
    messages: [{ ID: operationId, ConversationID: conversationId, Sequence: 1, Author: "USER", Content: "已有内容", CreatedAt: "2026-10-03T00:00:00Z" }],
    proposals: [proposal], before: "" } : Promise.reject(new AIWorkbenchError("FORBIDDEN")));
  render(tree());
  expect(await screen.findByText("已有内容")).toBeVisible();
  expect(screen.getByText("只读方案")).toBeVisible();
  for (const name of ["收藏", "归档", "发送消息", "确认并创建任务"]) expect(screen.queryByRole("button", { name })).toBeNull();
  expect(screen.queryByLabelText("需求")).toBeNull();
  expect(fixture.request).toHaveBeenCalledTimes(1);
});

it("restores an archived conversation using its current metadata revision", async () => {
  let current = { ...conversation, Archived: true, MetadataRevision: 2 };
  fixture.request.mockImplementation(async ({ route, method, body, revision }) => {
    if (route === "conversation-read") return { conversation: current, messages: [], proposals: [], before: "" };
    if (route === "conversation-metadata" && method === "PATCH") {
      expect(body).toEqual({ archived: false });
      expect(revision).toBe(2);
      current = { ...current, Archived: false, MetadataRevision: 3 };
      return { conversation: current };
    }
    throw new AIWorkbenchError("INVALID_REQUEST");
  });
  render(tree());
  expect(await screen.findByText("会话已归档")).toBeVisible();
  fireEvent.click(screen.getByRole("button", { name: "恢复会话" }));
  expect(await screen.findByLabelText("需求")).toBeVisible();
  expect(screen.queryByText("会话已归档")).toBeNull();
});

it("edits a conversation title through the existing revision-checked metadata route", async () => {
  let current = { ...conversation, Title: "原始标题", MetadataRevision: 4 };
  fixture.request.mockImplementation(async ({ route, method, body, revision }) => {
    if (route === "conversation-read") return { conversation: current, messages: [], proposals: [], before: "" };
    if (route === "conversation-metadata" && method === "PATCH") {
      expect(body).toEqual({ title: "修改后的标题" });
      expect(revision).toBe(4);
      current = { ...current, Title: "修改后的标题", MetadataRevision: 5 };
      return { conversation: current };
    }
    throw new AIWorkbenchError("INVALID_REQUEST");
  });
  render(tree());
  fireEvent.click(await screen.findByRole("button", { name: "修改标题" }));
  fireEvent.change(screen.getByRole("textbox", { name: "会话标题" }), { target: { value: "修改后的标题" } });
  fireEvent.click(screen.getByRole("button", { name: "保存标题" }));
  expect(await screen.findByText("修改后的标题")).toBeVisible();
  expect(fixture.request).toHaveBeenCalledWith(expect.objectContaining({ route: "conversation-metadata", method: "PATCH", revision: 4 }));
});

it("hides creation for a read-only organization and restores it after switching to a writable one", async () => {
  fixture.context.roles = ["listingkit_viewer"]; fixture.context.permissions = ["workbench.task.read","workbench.store.read","workbench.source_account.read","workbench.organization_member.read","workbench.commercial.read"];
  fixture.context.effectiveOrganization = { id: "org-a", capabilities: { "workbench.chat.use": false } };
  fixture.request.mockImplementation(async ({ route }) => route === "conversation-list" ? { conversations: [], next: "" }
    : Promise.reject(new AIWorkbenchError("FORBIDDEN")));
  const view = render(<QueryClientProvider client={client}><ChatPage mode="home" /></QueryClientProvider>);
  await screen.findByText("开始一项新需求");
  expect(screen.queryByRole("button", { name: "进入新建会话 →" })).toBeNull();
  fixture.context.roles = ["listingkit_operator"]; fixture.context.permissions = ["product_sourcing.write","local_agent.write","listingkit.image_agent.read","listingkit.image_agent.write","workbench.agent.read","workbench.agent.use","workbench.chat.read","workbench.chat.use","workbench.task.read","workbench.knowledge.read","workbench.store.read","workbench.store.create","workbench.store.update","workbench.store.lifecycle","workbench.source_account.read","workbench.source_account.manage","workbench.organization_member.read","workbench.commercial.read"];
  fixture.context.effectiveOrganization = { id: "org-b", capabilities: { "workbench.chat.use": true } };
  view.rerender(<QueryClientProvider client={client}><ChatPage mode="home" /></QueryClientProvider>);
  expect(await screen.findByRole("button", { name: "进入新建会话 →" })).toBeVisible();
  fixture.context.effectiveOrganization = { id: "org-b", capabilities: { "workbench.chat.use": false } };
  view.rerender(<QueryClientProvider client={client}><ChatPage mode="home" /></QueryClientProvider>);
  expect(screen.queryByRole("button", { name: "进入新建会话 →" })).toBeNull();
});

it("explains a terminal plan failure before dispatch and keeps the goal for a new attempt", async () => {
  fixture.request.mockImplementation(async ({ route }) => {
    if (route === "conversation-read") return { conversation, messages: [], proposals: [], before: "" };
    if (route === "message") return { state: "FAILED_BEFORE_DISPATCH" };
    throw new AIWorkbenchError("INVALID_REQUEST");
  });
  render(tree());
  await screen.findByText("开始讨论");
  fireEvent.change(screen.getByLabelText("采集操作 ID"), { target: { value: operationId } });
  fireEvent.change(screen.getByLabelText("需求"), { target: { value: "优化标题" } });
  fireEvent.click(screen.getByRole("button", { name: "发送消息" }));
  expect(await screen.findByText(/规划尚未发送到模型/)).toBeInTheDocument();
  expect(screen.getByLabelText("需求")).toHaveValue("优化标题");
  expect(screen.getByRole("button", { name: "发送消息" })).toBeEnabled();
});

it("explains a settled invalid planner response and releases the composer for a new intent", async () => {
  fixture.request.mockImplementation(async ({ route }) => {
    if (route === "conversation-read") return { conversation, messages: [], proposals: [], before: "" };
    if (route === "message") return { state: "PLANNER_INVALID_OUTPUT" };
    throw new AIWorkbenchError("INVALID_REQUEST");
  });
  render(tree());
  await screen.findByText("开始讨论");
  fireEvent.change(screen.getByLabelText("采集操作 ID"), { target: { value: operationId } });
  fireEvent.change(screen.getByLabelText("需求"), { target: { value: "优化标题" } });
  fireEvent.click(screen.getByRole("button", { name: "发送消息" }));
  expect(await screen.findByText(/模型返回的规划内容格式无效/)).toBeInTheDocument();
  expect(screen.getByLabelText("需求")).toHaveValue("优化标题");
  expect(screen.getByRole("button", { name: "发送消息" })).toBeEnabled();
});

it("keeps the exact message key and body across a lost response and page reload", async () => {
  const view = render(tree());
  await screen.findByText("开始讨论");
  fireEvent.change(screen.getByLabelText("采集操作 ID"), { target: { value: operationId } });
  fireEvent.change(screen.getByLabelText("需求"), { target: { value: "优化标题" } });
  fireEvent.click(screen.getByRole("button", { name: "发送消息" }));
  await screen.findByText(/结果暂无法确认/);
  const first = fixture.request.mock.calls.find(call => call[0].route === "message")?.[0];
  expect(first).toMatchObject({ key: expect.any(String), body: { content: "优化标题", operationId, targetPlatform: "shein" } });
  view.unmount();

  render(tree());
  await screen.findByRole("button", { name: "重试同一消息" });
  expect(screen.getByLabelText("需求")).toBeDisabled();
  expect(fixture.request.mock.calls.filter(call => call[0].route === "message")).toHaveLength(1);
  await waitFor(()=>expect(screen.getByRole("button", { name: "重试同一消息" })).toBeEnabled());
  fireEvent.click(screen.getByRole("button", { name: "重试同一消息" }));
  await waitFor(() => expect(fixture.request.mock.calls.filter(call => call[0].route === "message")).toHaveLength(2));
  const second = fixture.request.mock.calls.filter(call => call[0].route === "message")[1][0];
  expect(second.key).toBe(first.key);
  expect(second.body).toEqual(first.body);
});

it("unmounts the old organization view and never sends its pending draft to the new organization", async () => {
  const view = render(tree());
  await screen.findByText("开始讨论");
  fireEvent.change(screen.getByLabelText("采集操作 ID"), { target: { value: operationId } });
  fireEvent.change(screen.getByLabelText("需求"), { target: { value: "企业甲内容" } });
  fireEvent.click(screen.getByRole("button", { name: "发送消息" }));
  await screen.findByText(/结果暂无法确认/);
  fixture.context.isSwitching = true;
  view.rerender(tree());
  expect(screen.queryByDisplayValue("企业甲内容")).toBeNull();
  fixture.context.isSwitching = false;
  fixture.context.effectiveOrganization = { id: "org-b", capabilities: { "workbench.chat.use": true } };
  fixture.request.mockImplementation(async ({ route }) => route === "conversation-read"
    ? { conversation: { ...conversation, Scope: { OrganizationID: "org-b", ActorID: "user-a" } }, messages: [], proposals: [], before: "" }
    : Promise.reject(new AIWorkbenchError("OUTCOME_UNKNOWN")));
  view.rerender(tree());
  await screen.findByText("开始讨论");
  expect(screen.queryByDisplayValue("企业甲内容")).toBeNull();
  expect(screen.getByRole("button", { name: "发送消息" })).toBeDisabled();
});

it.each(["product.title.optimize", "product.content.localize"])("reads the committed %s task from the owner after browser storage is lost", async kind => {
  const taskId = "5892474d-1c8a-47e4-9550-45ed946e8197";
  fixture.context.aiWorkbenchExecutionKind = kind as "product.title.optimize" | "product.content.localize";
  sessionStorage.setItem(`ai-workbench:confirm:user-a:org-a:${conversationId}`, JSON.stringify({[operationId]:{
    key:operationId,taskId:"11111111-1111-4111-8111-111111111111",
  }}));
  fixture.request.mockImplementation(async ({ route }) => {
    if (route === "conversation-read") return { conversation, messages: [{ ID: operationId, ConversationID: conversationId, Sequence: 1,
      Author: "USER", Content: "生成商品内容", CreatedAt: "2026-10-03T00:00:00Z" }], proposals: [{
      kind, id: operationId, digest: "a".repeat(64), sourceSequence: 1, goalSummary: "生成商品内容", confirmedTaskId: taskId,
      humanReviewRequired: true, detailsAvailable: false, titleProfileReady: false, executionAuthorized: false,
    }], before: "" };
    throw new AIWorkbenchError("INVALID_REQUEST");
  });
  render(tree());
  expect(await screen.findByRole("link", { name: "查看已确认任务" })).toHaveAttribute("href", `/workbench/ai/tasks/${taskId}`);
  expect(screen.getByText("已确认的执行方案")).toBeVisible();
  expect(screen.queryByRole("button", { name: "确认并创建任务" })).toBeNull();
  expect(fixture.request.mock.calls.every(([call]) => ["conversation-read","schedule-availability"].includes(call.route))).toBe(true);
});

it("reuses the confirmation key after a lost response and links to the committed task", async () => {
  const proposalId = "d5d9d1ca-1db3-43af-9649-dcdf3663745b";
  const taskId = "5892474d-1c8a-47e4-9550-45ed946e8197";
  const proposal = { id: proposalId, digest: "a".repeat(64), sourceSequence: 1, goalSummary: "优化标题", productKey: "product",
    targetPlatform: "shein", humanReviewRequired: true, detailsAvailable: true, titleProfileReady: true, executionAuthorized: true };
  let confirmCalls = 0;
  fixture.request.mockImplementation(async ({ route }) => {
    if (route === "conversation-read") return { conversation, messages: [{ ID: operationId, ConversationID: conversationId, Sequence: 1,
      Author: "USER", Content: "优化标题", CreatedAt: "2026-10-03T00:00:00Z" }], proposals: [proposal], before: "" };
    if (route === "confirm") {
      confirmCalls++;
      if (confirmCalls === 1) throw new AIWorkbenchError("OUTCOME_UNKNOWN");
      return { task: { id: taskId } };
    }
    throw new AIWorkbenchError("INVALID_REQUEST");
  });
  const view = render(tree());
  await screen.findByText("待确认的执行方案");
  fireEvent.click(screen.getByRole("button", { name: "确认并创建任务" }));
  await screen.findByText(/结果暂无法确认/);
  const first = fixture.request.mock.calls.find(call => call[0].route === "confirm")?.[0];
  view.unmount();

  render(tree());
  await screen.findByRole("button", { name: "确认并创建任务" });
  fireEvent.click(screen.getByRole("button", { name: "确认并创建任务" }));
  await waitFor(() => expect(fixture.request.mock.calls.filter(call => call[0].route === "confirm")).toHaveLength(2));
  const second = fixture.request.mock.calls.filter(call => call[0].route === "confirm")[1][0];
  expect(second.key).toBe(first.key);
  expect(await screen.findByRole("link", { name: "查看已确认任务" })).toHaveAttribute("href", `/workbench/ai/tasks/${taskId}`);
  expect(fixture.push).toHaveBeenCalledWith(`/workbench/ai/tasks/${taskId}`);
});

it.each([true, false])("does not show a cached conversation after authorization changes, native role changes=%s", async (roleChanged) => {
  fixture.context.roles = ["sumi_role_0123456789abcdef0123456789abcdef_01"];
  let reads = 0;
  fixture.request.mockImplementation(async ({ route }) => {
    if (route !== "conversation-read") throw new AIWorkbenchError("INVALID_REQUEST");
    reads++;
    if (reads === 1) return { conversation, messages: [{ ID: operationId, ConversationID: conversationId, Sequence: 1,
      Author: "USER", Content: "之前有权查看的内容", CreatedAt: "2026-10-03T00:00:00Z" }], proposals: [], before: "" };
    return new Promise(() => {});
  });
  const view = render(tree());
  await screen.findByText("之前有权查看的内容");
  if (roleChanged) fixture.context.roles = ["sumi_role_0123456789abcdef0123456789abcdef_02"];
  fixture.context.permissions = fixture.context.permissions.filter(permission => permission !== "workbench.chat.read");
  view.rerender(tree());
  await screen.findByText("正在读取会话");
  expect(screen.queryByText("之前有权查看的内容")).toBeNull();
  expect(reads).toBe(2);
});

it("discards an A response that arrives after A to B to A context switching", async () => {
  let resolveOld!: (value: unknown) => void;
  const old = new Promise<unknown>(resolve => { resolveOld = resolve; });
  let aReads = 0;
  fixture.request.mockImplementation(async ({ route, scope }) => {
    if (route !== "conversation-read") throw new AIWorkbenchError("INVALID_REQUEST");
    if (scope.organizationId === "org-b") return { conversation: { ...conversation, Scope: { OrganizationID: "org-b", ActorID: "user-a" } },
      messages: [], proposals: [], before: "" };
    aReads++;
    if (aReads === 1) return old;
    return { conversation, messages: [{ ID: operationId, ConversationID: conversationId, Sequence: 1,
      Author: "USER", Content: "企业甲的当前内容", CreatedAt: "2026-10-03T00:00:00Z" }], proposals: [], before: "" };
  });
  const view = render(tree());
  await waitFor(() => expect(aReads).toBe(1));
  const oldSignal: AbortSignal = fixture.request.mock.calls[0][0].signal;
  fixture.context.isSwitching = true;
  view.rerender(tree());
  expect(oldSignal.aborted).toBe(true);
  client.clear();
  fixture.context.isSwitching = false;
  fixture.context.effectiveOrganization = { id: "org-b", capabilities: { "workbench.chat.use": true } };
  view.rerender(tree());
  await screen.findByText("开始讨论");
  fixture.context.isSwitching = true;
  view.rerender(tree());
  client.clear();
  fixture.context.isSwitching = false;
  fixture.context.effectiveOrganization = { id: "org-a", capabilities: { "workbench.chat.use": true } };
  view.rerender(tree());
  await screen.findByText("企业甲的当前内容");
  await act(async () => resolveOld({ conversation, messages: [{ ID: operationId, ConversationID: conversationId, Sequence: 1,
    Author: "USER", Content: "企业甲的过期响应", CreatedAt: "2026-10-03T00:00:00Z" }], proposals: [], before: "" }));
  expect(screen.queryByText("企业甲的过期响应")).toBeNull();
  expect(screen.getByText("企业甲的当前内容")).toBeVisible();
});

it("creates content intent from a saved own product without an acquisition or title route and restores its exact unknown request",async()=>{
 fixture.context.aiWorkbenchExecutionKind="product.content.localize";fixture.context.aiWorkbenchTitleReadiness="UNAVAILABLE";
 const subject={kind:"own_product",id:operationId,catalogVersion:"3",publicationId:"effective-publication"};
 fixture.request.mockImplementation(async({route})=>{
  if(route==="conversation-read")return {conversation,messages:[],proposals:[],before:""};
  if(route==="subject-read")return {subject,productKey:"product"};
  throw new AIWorkbenchError("OUTCOME_UNKNOWN");
 });
 const view=render(tree());await screen.findByText("开始讨论");
 expect(screen.queryByLabelText("采集操作 ID")).toBeNull();
 fireEvent.click(screen.getByRole("button",{name:"选择我的商品"}));await screen.findByText(/版本 3/);
 fireEvent.click(screen.getByRole("checkbox",{name:"卖点"}));
 fireEvent.change(screen.getByLabelText("需求"),{target:{value:"整理描述与卖点"}});
 fireEvent.click(screen.getByRole("button",{name:"发送消息"}));await screen.findByRole("button",{name:"重试同一消息"});
 const original=fixture.request.mock.calls.find(([v])=>v.route==="message")![0];
 expect(original.body).toEqual({content:"整理描述与卖点",taskSelection:{subject,capability:"product.content.localize",parameters:{targetLanguage:"en-US",description:true,sellingPoints:true}}});
 expect(screen.getByLabelText("目标语言")).toBeDisabled();expect(screen.getByRole("button",{name:"选择我的商品"})).toBeDisabled();
 view.unmount();render(tree());await screen.findByRole("button",{name:"重试同一消息"});
 await waitFor(()=>expect(screen.getByRole("button",{name:"重试同一消息"})).toBeEnabled());
 fireEvent.click(screen.getByRole("button",{name:"重试同一消息"}));
 await waitFor(()=>expect(fixture.request.mock.calls.filter(([v])=>v.route==="message")).toHaveLength(2));
 expect(fixture.request.mock.calls.filter(([v])=>v.route==="message")[1][0]).toMatchObject({key:original.key,body:original.body});
});
it("confirms a content proposal without exposing internal execution units",async()=>{
 fixture.context.aiWorkbenchExecutionKind="product.content.localize";fixture.context.aiWorkbenchTitleReadiness="UNAVAILABLE";
 const proposal={id:operationId,digest:"a".repeat(64),sourceSequence:1,kind:"product.content.localize",goalSummary:"整理描述与卖点",taskSelection:{subject:{kind:"own_product",id:operationId,catalogVersion:"3",publicationId:"effective-publication"},capability:"product.content.localize",parameters:{targetLanguage:"en-US",description:true,sellingPoints:true}},detailsAvailable:true,titleProfileReady:false,executionProfileReady:true,executionAuthorized:true,humanReviewRequired:true,maximumTokens:6000,maximumCostMicros:1000,currency:"CNY",maximumSteps:8,maximumModelCalls:3,maximumRuntimeSeconds:60};
 fixture.request.mockImplementation(async({route})=>{if(route==="conversation-read")return {conversation,messages:[{ID:operationId,Sequence:1,Author:"USER",Content:"需求",CreatedAt:conversation.CreatedAt}],proposals:[proposal],before:""};throw new AIWorkbenchError("OUTCOME_UNKNOWN")});
 render(tree());expect(await screen.findByRole("button",{name:"确认并创建任务"})).toBeEnabled();
 expect(screen.getByText("en-US · 描述、卖点")).toBeVisible();
 expect(screen.queryByText(/6000 tokens|1000 μCNY|3 次模型调用/)).not.toBeInTheDocument();
 fixture.context.aiWorkbenchExecutionReadiness="NEEDS_CONFIGURATION";
 cleanup();render(tree());expect(await screen.findByRole("button",{name:"确认并创建任务"})).toBeDisabled();
});
