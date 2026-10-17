import { afterEach, expect, it, vi } from "vitest";
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { AgentPage } from "./agent-page";
vi.mock("./product-selection-flow",()=>({ProductSelectionFlow:({nonce}:{nonce:number})=><p>业务流程刷新：{nonce}</p>}));
const scope = vi.hoisted(() => ({
  user: { id: "actor" },
  effectiveOrganization: { id: "org-a", name: "企业A" },
  roles: ["listingkit_admin"],
  permissions: [
    "listingkit.admin.read",
    "listingkit.admin.write",
    "product_sourcing.write",
    "local_agent.write",
    "listingkit.image_agent.read",
    "listingkit.image_agent.write",
    "workbench.agent.read",
    "workbench.agent.use",
    "workbench.agent.configure",
    "workbench.chat.read",
    "workbench.chat.use",
    "workbench.task.read",
    "workbench.knowledge.read",
    "workbench.knowledge.manage",
    "workbench.store.read",
    "workbench.store.create",
    "workbench.store.update",
    "workbench.store.lifecycle",
    "workbench.store.delete",
    "workbench.source_account.read",
    "workbench.source_account.manage",
    "workbench.organization_member.read",
    "workbench.organization_member.manage",
    "workbench.commercial.read",
    "workbench.commercial.purchase",
    "workbench.commercial.wallet_topup",
  ],
  isLoading: false,
  isSwitching: false,
  error: null,
  blockingError: null,
  selectionRequired: false,
  registerOrganizationSwitchGuard:()=>()=>{},
}));
vi.mock("@/components/providers/workbench-context-provider", () => ({
  useWorkbenchContext: () => scope,
}));
vi.mock("next/link", () => ({
  default: ({ children, href, prefetch, ...props }: React.ComponentProps<"a"> & {prefetch?: boolean}) => {
    void prefetch;
    return <a href={href} {...props}>{children}</a>;
  },
}));
const item = {
  agent: {
    agentId: "product.title.agent",
    activation: "NOT_ENABLED",
    revision: "",
    activationEpoch: "",
    defaultTemplate: null,
    updatedAt: "0001-01-01T00:00:00Z",
  },
  name: "商品标题优化智能体",
  description: "生成标题建议",
  definitionVersion: "v1.0.0",
  parameterSchema: "title-config-v1",
  canConfigure: true,
  canUse: false,
  canReadRuns: false,
  capabilities: [],
};
const initialPermissions = [...scope.permissions];
afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
  scope.effectiveOrganization = { id: "org-a", name: "企业A" };
  scope.isSwitching = false;
  scope.permissions = [...initialPermissions];
});

function modeFixture(agentId:string, canConfigure=true) {
  return {subject:{kind:"registered_agent",id:agentId},definition:{id:agentId,version:"v1.0.0"},mode:"default",revision:"0",policyVersion:"agent-execution-policy-v1",canConfigure,updatedAt:"2026-10-17T00:00:00Z",updatedBy:""};
}
function installDialogMethods() {
  Object.defineProperty(HTMLDialogElement.prototype, "showModal", {configurable:true,value:function(this:HTMLDialogElement){this.open=true;}});
  Object.defineProperty(HTMLDialogElement.prototype, "close", {configurable:true,value:function(this:HTMLDialogElement){this.open=false;}});
}

it("opens a read-only capability dialog with fresh facts and keeps the original configuration routes", async () => {
  installDialogMethods();
  const entry = {...marketEntries()[1], canConfigure:false, agent:{...marketEntries()[1].agent,activation:"ENABLED",revision:"7",activationEpoch:"1"}};
  const current = {...entry,capabilities:[
    {id:"text.generate",support:"REQUIRED",readiness:"NEEDS_CONFIGURATION",reason:"本企业尚未配置文本路由，原配置保持不变",observedAt:"2026-10-17T00:00:00Z"},
    {id:"image.generate",support:"NOT_SUPPORTED",readiness:"AVAILABLE",reason:"本智能体不提供图片执行",observedAt:"2026-10-17T00:00:00Z"},
    {id:"knowledge.context",support:"OPTIONAL",readiness:"AVAILABLE",reason:"引用企业知识仍需在执行前明确选择",observedAt:"2026-10-17T00:00:00Z"},
    {id:"platform.write",support:"NOT_SUPPORTED",readiness:"UNAVAILABLE",reason:"不提供平台写入",observedAt:"2026-10-17T00:00:00Z"},
  ]};
  const fetch=vi.fn(async(url:unknown,init?:RequestInit)=>{void init;return Response.json(String(url).endsWith("/agents/product.content.agent")?current:{items:String(url).includes("/agents/mine?")?[entry]:[],nextCursor:""});});
  vi.stubGlobal("fetch",fetch);
  render(<AgentPage mode="mine"/>);
  const card=await screen.findByRole("article",{name:"商品内容与本地化"});
  expect(fetch.mock.calls.some(([url])=>String(url).endsWith("/agents/product.content.agent"))).toBe(false);
  fireEvent.click(within(card).getByRole("button",{name:"能力配置"}));
  const dialog=screen.getByRole("dialog",{name:"智能体能力配置"});
  await within(dialog).findByText("本企业尚未配置文本路由，原配置保持不变");
  expect(within(dialog).getByText("需要配置")).toBeInTheDocument();
  const image=within(dialog).getByRole("region",{name:"图片生成 · 不支持"});
  expect(within(image).getByText("不支持")).toBeInTheDocument();
  expect(within(image).queryByText("可用")).toBeNull();
  expect(within(dialog).getByText("引用企业知识仍需在执行前明确选择")).toBeInTheDocument();
  expect(within(dialog).queryByText(/硕米官方模型|品牌与表达规范/)).toBeNull();
  expect(within(dialog).queryByRole("button",{name:/保存|启用|停用/})).toBeNull();
  expect(within(dialog).getByRole("link",{name:"查看权限与模板"})).toHaveAttribute("href","/workbench/agents/mine/product.content.agent");
  expect(within(dialog).getByRole("link",{name:"标题优化配置与历史"})).toHaveAttribute("href","/workbench/agents/mine/product.title.agent");
  fireEvent.click(within(dialog).getByRole("button",{name:"关闭能力配置"}));
  expect(screen.queryByRole("dialog")).toBeNull();
  expect(within(card).getByRole("button",{name:"能力配置"})).toHaveFocus();
  expect(fetch.mock.calls.every(([,init])=>!init?.method||init.method==="GET")).toBe(true);
});

it("retries capability reads without changing saved configuration or issuing a command", async () => {
  installDialogMethods();
  const entry={...marketEntries()[4],agent:{...marketEntries()[4].agent,activation:"ENABLED",revision:"3",activationEpoch:"1"}};
  let reads=0;
  const fetch=vi.fn(async(url:unknown,init?:RequestInit)=>{void init;if(String(url).endsWith("/agents/pod.design.agent")){reads++;return reads===1?Response.json({code:"DEPENDENCY_UNAVAILABLE"},{status:503}):Response.json(entry);}return Response.json({items:String(url).includes("/agents/mine?")?[entry]:[],nextCursor:""});});
  vi.stubGlobal("fetch",fetch);
  render(<AgentPage mode="mine"/>);
  fireEvent.click(within(await screen.findByRole("article",{name:"POD设计智能体"})).getByRole("button",{name:"能力配置"}));
  const dialog=screen.getByRole("dialog",{name:"智能体能力配置"});
  await within(dialog).findByRole("alert");
  expect(within(dialog).queryByRole("link",{name:"管理权限与模板"})).toBeNull();
  fireEvent.click(within(dialog).getByRole("button",{name:"重新读取能力"}));
  expect(await within(dialog).findByRole("link",{name:"管理权限与模板"})).toHaveAttribute("href","/workbench/agents/mine/pod.design.agent");
  expect(reads).toBe(2);
  fireEvent(dialog,new Event("cancel",{cancelable:true}));
  expect(screen.queryByRole("dialog")).toBeNull();
  expect(fetch.mock.calls.every(([,init])=>!init?.method||init.method==="GET")).toBe(true);
});

it("closes capability details on organization change and rejects the late previous-scope response", async () => {
  installDialogMethods();
  const entry={...marketEntries()[1],agent:{...marketEntries()[1].agent,activation:"ENABLED",revision:"1",activationEpoch:"1"}};
  let finish!:(response:Response)=>void;
  const pending=new Promise<Response>(resolve=>{finish=resolve;});
  vi.stubGlobal("fetch",vi.fn(async(url:unknown)=>String(url).endsWith("/agents/product.content.agent")?pending:Response.json({items:String(url).includes("/agents/mine?")?[entry]:[],nextCursor:""})));
  const view=render(<AgentPage mode="mine"/>);
  fireEvent.click(within(await screen.findByRole("article",{name:"商品内容与本地化"})).getByRole("button",{name:"能力配置"}));
  expect(screen.getByRole("dialog")).toBeInTheDocument();
  scope.effectiveOrganization={id:"org-b",name:"企业B"};
  view.rerender(<AgentPage mode="mine"/>);
  expect(screen.queryByRole("dialog")).toBeNull();
  await act(async()=>finish(Response.json({...entry,description:"旧企业的能力详情不得回流"})));
  expect(screen.queryByText("旧企业的能力详情不得回流")).toBeNull();
  expect(screen.queryByRole("dialog")).toBeNull();
});

it("clears an open capability dialog after permissions change and keeps read-only access", async () => {
  installDialogMethods();
  const entry={...marketEntries()[4],canConfigure:false,agent:{...marketEntries()[4].agent,activation:"ENABLED",revision:"1",activationEpoch:"1"}};
  vi.stubGlobal("fetch",vi.fn(async(url:unknown)=>Response.json(String(url).endsWith("/agents/pod.design.agent")?entry:{items:String(url).includes("/agents/mine?")?[entry]:[],nextCursor:""})));
  const view=render(<AgentPage mode="mine"/>);
  fireEvent.click(within(await screen.findByRole("article",{name:"POD设计智能体"})).getByRole("button",{name:"能力配置"}));
  await screen.findByRole("link",{name:"查看权限与模板"});
  scope.permissions=scope.permissions.filter(permission=>permission!=="workbench.agent.configure");
  view.rerender(<AgentPage mode="mine"/>);
  expect(screen.queryByRole("dialog")).toBeNull();
  const card=await screen.findByRole("article",{name:"POD设计智能体"});
  expect(within(card).getByRole("button",{name:"能力配置"})).toBeEnabled();
  expect(screen.getByRole("button",{name:"启用智能体"})).toBeDisabled();
});

it("does not show a mismatched agent returned for a capability read", async () => {
  installDialogMethods();
  const entry={...marketEntries()[4],agent:{...marketEntries()[4].agent,activation:"ENABLED",revision:"1",activationEpoch:"1"}};
  vi.stubGlobal("fetch",vi.fn(async(url:unknown)=>Response.json(String(url).endsWith("/agents/pod.design.agent")?{...marketEntries()[1],name:"另一个智能体的详情"}:{items:String(url).includes("/agents/mine?")?[entry]:[],nextCursor:""})));
  render(<AgentPage mode="mine"/>);
  fireEvent.click(within(await screen.findByRole("article",{name:"POD设计智能体"})).getByRole("button",{name:"能力配置"}));
  const dialog=screen.getByRole("dialog",{name:"智能体能力配置"});
  await within(dialog).findByRole("alert");
  expect(within(dialog).queryByText("另一个智能体的详情")).toBeNull();
  expect(within(dialog).queryByRole("link",{name:/权限与模板/})).toBeNull();
});

it("labels paginated counts as current-page facts and keeps the existing cursor navigation", async () => {
  const pod = {...marketEntries()[4],agent:{...marketEntries()[4].agent,activation:"ENABLED",revision:"1",activationEpoch:"1"}};
  const materials = {...marketEntries()[2],agent:{...marketEntries()[2].agent,activation:"ENABLED",revision:"1",activationEpoch:"1"}};
  const fetch = vi.fn(async(url:unknown,init?:RequestInit)=>{
    void init;
    const path = String(url);
    return Response.json(!path.includes("/agents/mine?") ? {items:[],nextCursor:""} : path.includes("cursor=second") ? {items:[materials],nextCursor:""} : {items:[pod],nextCursor:"second"});
  });
  vi.stubGlobal("fetch",fetch);
  render(<AgentPage mode="mine"/>);
  await screen.findByRole("article",{name:"POD设计智能体"});
  expect(screen.getByText(/当前页已启用/)).toBeInTheDocument();
  expect(screen.queryByText(/当前企业已启用/)).toBeNull();
  fireEvent.click(screen.getByRole("button",{name:"下一页"}));
  await screen.findByRole("article",{name:"平台上架资料智能体"});
  expect(screen.getByText(/当前页已启用/)).toBeInTheDocument();
  expect(screen.queryByRole("article",{name:"POD设计智能体"})).toBeNull();
  fireEvent.click(screen.getByRole("button",{name:"返回智能体首页"}));
  await screen.findByRole("article",{name:"POD设计智能体"});
  expect(fetch.mock.calls.every(([,init])=>!init?.method||init.method==="GET")).toBe(true);
});

it("organizes mine into responsibility cards with honest activation filters and a separate usage guide", async () => {
  const entries = marketEntries().filter(entry => ["product.title.agent","product.content.agent","listing.materials.agent","pod.design.agent"].includes(entry.agent.agentId)).map(entry => ({
    ...entry,
    agent:{...entry.agent,activation:["product.title.agent","pod.design.agent"].includes(entry.agent.agentId)?"DISABLED":"ENABLED",revision:"2",activationEpoch:"1"},
    canUse:entry.agent.agentId==="product.content.agent",
    capabilities:[{id:"text.generate",support:"REQUIRED",readiness:"NEEDS_CONFIGURATION",reason:"不应在管理卡片展开的模型路由长说明",observedAt:"2026-10-17T00:00:00Z"}],
  }));
  const fetch = vi.fn(async (_url:unknown,init?:RequestInit) => {void init;return Response.json({items:String(_url).includes("private")?[]:entries,nextCursor:""});});
  vi.stubGlobal("fetch", fetch);
  render(<AgentPage mode="mine"/>);
  const content = await screen.findByRole("article",{name:"商品内容与本地化"});
  expect(within(content).getByText("部分已启用")).toBeInTheDocument();
  expect(within(content).getByRole("button",{name:"能力配置"})).toHaveAttribute("aria-haspopup","dialog");
  expect(within(content).getByRole("link",{name:"进入使用"})).toHaveAttribute("href","/workbench/supply/acquisition?agent_kind=content");
  expect(screen.getByRole("complementary",{name:"使用流程"})).toBeInTheDocument();
  expect(screen.queryByText("不应在管理卡片展开的模型路由长说明")).toBeNull();
  const filters = within(screen.getByRole("group",{name:"企业智能体状态"}));
  expect(filters.getByRole("button",{name:"全部 3"})).toHaveAttribute("aria-pressed","true");
  fireEvent.click(filters.getByRole("button",{name:"已停用 1"}));
  expect(screen.queryByRole("article",{name:"商品内容与本地化"})).toBeNull();
  expect(screen.getByRole("article",{name:"POD设计智能体"})).toBeInTheDocument();
  fireEvent.click(filters.getByRole("button",{name:"已启用 2"}));
  expect(screen.getByRole("article",{name:"商品内容与本地化"})).toBeInTheDocument();
  expect(screen.queryByRole("article",{name:"POD设计智能体"})).toBeNull();
  expect(fetch.mock.calls.every(([,init])=>!init?.method||init.method==="GET")).toBe(true);
  expect(fetch.mock.calls.filter(([url])=>String(url).includes("/agents/mine?")).length).toBe(1);
});

it("loads the activation picker only when requested and navigates to the existing scoped detail without enabling", async () => {
  installDialogMethods();
  const fetch = vi.fn(async (url:unknown,init?:RequestInit)=>{void init;return Response.json({items:String(url).includes("/agents/market?")?marketEntries():[],nextCursor:""});});
  vi.stubGlobal("fetch",fetch);
  render(<AgentPage mode="mine"/>);
  await screen.findByText("当前企业尚未启用智能体");
  expect(fetch.mock.calls.some(([url])=>String(url).includes("/agents/market?"))).toBe(false);
  fireEvent.click(screen.getByRole("button",{name:"启用智能体"}));
  const picker = screen.getByRole("dialog",{name:"启用当前企业智能体"});
  expect(await within(picker).findByRole("article",{name:"POD设计智能体"})).toBeInTheDocument();
  expect(within(within(picker).getByRole("article",{name:"POD设计智能体"})).getByRole("link",{name:"前往启用"})).toHaveAttribute("href","/workbench/agents/mine/pod.design.agent");
  expect(within(picker).queryByRole("button",{name:"启用到当前企业"})).toBeNull();
  fireEvent.click(within(picker).getByRole("button",{name:"关闭"}));
  expect(screen.queryByRole("dialog")).toBeNull();
  expect(fetch.mock.calls.every(([,init])=>!init?.method||init.method==="GET")).toBe(true);
});

it("closes the activation picker and discards its old organization response when context changes", async () => {
  installDialogMethods();
  let resolveCatalog!:(response:Response)=>void;
  const delayedCatalog = new Promise<Response>(resolve=>{resolveCatalog=resolve;});
  vi.stubGlobal("fetch",vi.fn(async(url:unknown)=>String(url).includes("/agents/market?")?delayedCatalog:Response.json({items:[],nextCursor:""})));
  const view = render(<AgentPage mode="mine"/>);
  await screen.findByText("当前企业尚未启用智能体");
  fireEvent.click(screen.getByRole("button",{name:"启用智能体"}));
  expect(screen.getByRole("dialog")).toBeInTheDocument();
  scope.isSwitching=true;
  view.rerender(<AgentPage mode="mine"/>);
  expect(screen.queryByRole("dialog")).toBeNull();
  scope.effectiveOrganization={id:"org-b",name:"企业B"};
  scope.isSwitching=false;
  view.rerender(<AgentPage mode="mine"/>);
  await act(async()=>resolveCatalog(Response.json({items:[{...marketEntries()[4],name:"旧企业待启用智能体"}],nextCursor:""})));
  expect(screen.queryByText("旧企业待启用智能体")).toBeNull();
  expect(screen.queryByRole("dialog")).toBeNull();
});

it("removes an open activation picker after configuration permission is revoked", async () => {
  installDialogMethods();
  vi.stubGlobal("fetch",vi.fn(async()=>Response.json({items:[],nextCursor:""})));
  const view=render(<AgentPage mode="mine"/>);
  await screen.findByText("当前企业尚未启用智能体");
  fireEvent.click(screen.getByRole("button",{name:"启用智能体"}));
  expect(screen.getByRole("dialog")).toBeInTheDocument();
  scope.permissions=scope.permissions.filter(permission=>permission!=="workbench.agent.configure");
  view.rerender(<AgentPage mode="mine"/>);
  await screen.findByText("当前企业尚未启用智能体");
  expect(screen.queryByRole("dialog")).toBeNull();
  expect(screen.getByRole("button",{name:"启用智能体"})).toBeDisabled();
});

it.each([
  ["mine", "product.selection.agent", "product.selection.inspect", "product-selection-config-v1", "查看商品事实 →"],
  ["mine", "shop.operations-diagnosis.agent", "shop.operations-diagnosis.inspect", "shop-operations-diagnosis-config-v1", "查看经营事实 →"],
] as const)("exposes authorized facts from %s for %s before activation without enabling analysis", async (mode, agentId, capabilityId, parameterSchema, label) => {
  const fetch = vi.fn(async (_url: unknown, _init?: RequestInit) => { void _url; void _init; return Response.json({items: [{...item, parameterSchema, agent: {...item.agent, agentId}, capabilities: [{id: capabilityId, support: "REQUIRED", readiness: "AVAILABLE", reason: "", observedAt: "2026-10-14T00:00:00Z"}]}], nextCursor: ""}); });
  vi.stubGlobal("fetch", fetch);
  render(<AgentPage mode={mode}/>);
  expect(await screen.findByRole("link", {name: label})).toHaveAttribute("href", `/workbench/agents/mine/${agentId}`);
  expect(screen.getByText(agentId === "product.selection.agent" ? "商品事实对比 · 有引用建议" : "经营事实诊断 · 有引用建议")).toBeInTheDocument();
  expect(screen.queryByText("标题优化 · 人工审核")).toBeNull();
  expect(screen.queryByRole("link", {name: "进入使用 →"})).toBeNull();
  expect(screen.getByText("尚未启用")).toBeInTheDocument();
  expect(fetch.mock.calls.every(([, init]) => !init?.method || init.method === "GET")).toBe(true);
});

it.each([
  ["product.selection.agent", "product.selection.inspect", "product-selection-config-v1", "查看商品事实 →"],
  ["shop.operations-diagnosis.agent", "shop.operations-diagnosis.inspect", "shop-operations-diagnosis-config-v1", "查看经营事实 →"],
])("does not expose %s inspection when its current capability denies access", async (agentId, capabilityId, parameterSchema, label) => {
  vi.stubGlobal("fetch", vi.fn(async () => Response.json({items: [{...item, parameterSchema, agent: {...item.agent, agentId}, capabilities: [{id: capabilityId, support: "REQUIRED", readiness: "REQUIRES_AUTHORIZATION", reason: "需要授权", observedAt: "2026-10-14T00:00:00Z"}]}], nextCursor: ""})));
  render(<AgentPage mode="mine"/>);
  await screen.findByText("需要授权");
  expect(screen.queryByRole("link", {name: label})).toBeNull();
});

it("refreshes the adjacent business flow after the mode sidebar saves its exact receipt",async()=>{
 localStorage.clear();let revision="0";
 Object.defineProperty(HTMLDialogElement.prototype,"showModal",{configurable:true,value:function(this:HTMLDialogElement){this.open=true}});Object.defineProperty(HTMLDialogElement.prototype,"close",{configurable:true,value:function(this:HTMLDialogElement){this.open=false}});
 const subject={kind:"registered_agent",id:"product.selection.agent"},definition={id:subject.id,version:"v1.0.0"};
 const mode=()=>({subject,definition,mode:"read_only",revision,policyVersion:"agent-execution-policy-v1",canConfigure:true,updatedAt:"2026-10-14T00:00:00Z",updatedBy:"actor"});
 vi.stubGlobal("fetch",vi.fn(async(url,init)=>{
  const path=String(url);
  if(path.includes("execution-modes")){
   if(init?.method==="PUT"){const payload=JSON.parse(init.body),key=new Headers(init.headers).get("Idempotency-Key");revision="1";return Response.json({...mode(),receipt:{subject,requestKey:key,commandId:"11111111-1111-4111-8111-111111111111",operation:"set-execution-mode",mode:payload.mode,beforeRevision:payload.expectedRevision,revision,noop:false,committedAt:"2026-10-14T00:00:00Z",updatedBy:"actor",updatedAt:"2026-10-14T00:00:00Z"}})}
   return Response.json(mode());
  }
  if(path.endsWith("/agents/product.selection.agent"))return Response.json({...item,agent:{...item.agent,agentId:subject.id,activation:"ENABLED",revision:"1",activationEpoch:"1"},name:"选品对比",parameterSchema:"product-selection-config-v1",canUse:true});
  return Response.json({items:[],nextCursor:""});
 }));
 render(<AgentPage mode="mine" id="product.selection.agent"/>);await screen.findByText("业务流程刷新：0");fireEvent.click(await screen.findByRole("button",{name:"刷新权限状态"}));await screen.findByText("业务流程刷新：1");fireEvent.click(await screen.findByRole("button",{name:"配置权限"}));fireEvent.click(screen.getByRole("button",{name:"保存配置"}));await screen.findByText("业务流程刷新：2");localStorage.clear();
});
it("waits for a committed enable receipt and refetches the shared fact", async () => {
  let enabled = false;
  const fetch = vi.fn(async (_url: unknown, init?: RequestInit) => {
    if (init?.method === "POST") {
      enabled = true;
      return Response.json({
        commandId: "8fc227bb-b572-4138-8e2a-5f1a0be98617",
        operation: "enable",
        agentId: "product.title.agent",
        revision: "1",
        noop: false,
        committedAt: "2026-09-29T00:00:00Z",
      });
    }
    if (String(_url).includes("execution-modes")) return Response.json({
      subject:{kind:"registered_agent",id:"product.title.agent"},
      definition:{id:"product.title.agent",version:"v1.0.0"},
      mode:"default",revision:"0",policyVersion:"agent-execution-policy-v1",
      canConfigure:true,updatedAt:"2026-10-17T00:00:00Z",updatedBy:"",
    });
    if (String(_url).endsWith("/agents/product.title.agent")) return Response.json(enabled
      ? {...item,agent:{...item.agent,activation:"ENABLED",revision:"1",activationEpoch:"1"}}
      : item);
    return Response.json({items:[],nextCursor:""});
  });
  vi.stubGlobal("fetch", fetch);
  render(<AgentPage mode="mine" id="product.title.agent" />);
  fireEvent.click(
    await screen.findByRole("button", { name: "启用到当前企业" }),
  );
  await waitFor(() => expect(screen.getByText(/^已启用 ·/)).toBeInTheDocument());
  const call = fetch.mock.calls.find(([, init]) => init?.method === "POST")!;
  expect(new Headers(call[1]?.headers).get("If-None-Match")).toBe("*");
});

function marketEntries() {
  return [
    ["product.title.agent", "title-config-v1", "商品标题优化智能体"],
    ["product.content.agent", "content-config-v1", "商品内容完善与本地化智能体"],
    ["listing.materials.agent", "listing-materials-config-v1", "平台上架资料智能体"],
    ["product.image.agent", "image-config-v1", "商品图片智能体"],
    ["pod.design.agent", "pod-artwork-config-v1", "POD设计智能体"],
    ["product.selection.agent", "product-selection-config-v1", "选品对比智能体"],
    ["shop.operations-diagnosis.agent", "shop-operations-diagnosis-config-v1", "店铺经营诊断智能体"],
  ].map(([agentId, parameterSchema, name]) => ({...item, name, parameterSchema, agent: {...item.agent, agentId}}));
}

it("browses the original Figma categories without inventing agents in empty categories", async () => {
  const fetch = vi.fn(async () => Response.json({items:marketEntries(),nextCursor:""}));
  vi.stubGlobal("fetch", fetch);
  render(<AgentPage mode="market" />);
  await screen.findByRole("button", {name:"商品"});
  const categories = within(screen.getByRole("group", {name:"智能体业务分类"}));
  expect(categories.getAllByRole("button").map(button => button.textContent)).toEqual(["全部","商品","选品","运营","广告","客服","数据"]);
  expect(screen.getAllByRole("article")).toHaveLength(6);
  expect(screen.getAllByRole("heading",{name:"商品内容与本地化"})).toHaveLength(1);
  fireEvent.click(categories.getByRole("button",{name:"商品"}));
  expect(screen.getAllByRole("article")).toHaveLength(4);
  expect(screen.queryByRole("heading",{name:"选品对比智能体"})).toBeNull();
  fireEvent.click(categories.getByRole("button",{name:"选品"}));
  expect(screen.getAllByRole("article")).toHaveLength(1);
  expect(screen.getByRole("heading",{name:"选品对比智能体"})).toBeInTheDocument();
  fireEvent.click(categories.getByRole("button",{name:"运营"}));
  expect(screen.getByRole("heading",{name:"店铺经营诊断智能体"})).toBeInTheDocument();
  for (const name of ["广告","客服","数据"]) {
    fireEvent.click(categories.getByRole("button",{name}));
    expect(screen.queryByRole("article")).toBeNull();
    expect(screen.getByRole("heading",{name:"该分类暂无智能体"})).toBeInTheDocument();
  }
  fireEvent.click(categories.getByRole("button",{name:"全部"}));
  expect(screen.getAllByRole("article")).toHaveLength(6);
  expect(fetch).toHaveBeenCalledTimes(1);
});

it("keeps market browsing read-only and sends configuration and capability details to mine", async () => {
  const entries = marketEntries().map(entry => ({...entry,capabilities:[{id:"text.generate",support:"REQUIRED",readiness:"NEEDS_CONFIGURATION",reason:"需要独立模型路由与预算配置",observedAt:"2026-10-17T00:00:00Z"}]}));
  const fetch = vi.fn(async (_url: unknown, init?: RequestInit) => { void _url; void init; return Response.json({items:entries,nextCursor:""}); });
  vi.stubGlobal("fetch", fetch);
  render(<AgentPage mode="market" />);
  const card = await screen.findByRole("article",{name:"平台上架资料智能体"});
  expect(within(card).getByRole("link",{name:"前往启用"})).toHaveAttribute("href","/workbench/agents/mine/listing.materials.agent");
  expect(screen.queryByText("需要独立模型路由与预算配置")).toBeNull();
  expect(screen.queryByRole("link",{name:/配置|历史|查看商品事实|查看经营事实/})).toBeNull();
  expect(screen.queryByRole("button",{name:"启用到当前企业"})).toBeNull();
  expect(fetch.mock.calls.every(([, init]) => !init?.method || init.method === "GET")).toBe(true);
});

it("keeps mixed content activation honest and does not expose an enable action to a read-only viewer", async () => {
  const entries = marketEntries().slice(0,2).map((entry,index) => ({...entry,canConfigure:false,agent:{...entry.agent,...(index===1?{activation:"ENABLED",revision:"1",activationEpoch:"1"}:{})}}));
  vi.stubGlobal("fetch",vi.fn(async () => Response.json({items:entries,nextCursor:""})));
  render(<AgentPage mode="market" />);
  const card = await screen.findByRole("article",{name:"商品内容与本地化"});
  expect(within(card).getByText("部分已启用")).toBeInTheDocument();
  expect(within(card).getByRole("link",{name:"前往我的智能体"})).toHaveAttribute("href","/workbench/agents/mine");
  expect(screen.queryByRole("heading",{name:"标题优化"})).toBeNull();
});

it("uses the configured content operation's exact disable ID and revision", async () => {
  installDialogMethods();
  const content = {...marketEntries()[1],agent:{...marketEntries()[1].agent,activation:"ENABLED",revision:"4",activationEpoch:"1"}};
  const fetch = vi.fn(async (url:unknown,init?:RequestInit) => {
    if (init?.method === "POST") return Response.json({commandId:"8fc227bb-b572-4138-8e2a-5f1a0be98617",operation:"disable",agentId:"product.content.agent",revision:"5",noop:false,committedAt:"2026-10-17T00:00:00Z"});
    if (String(url).includes("execution-modes")) return Response.json(modeFixture("product.content.agent"));
    if (String(url).endsWith("/agents/product.content.agent")) return Response.json(content);
    return Response.json({items:[],nextCursor:""});
  });
  vi.stubGlobal("fetch",fetch);
  render(<AgentPage mode="mine" id="product.content.agent"/>);
  fireEvent.click(await screen.findByRole("button",{name:"停用智能体"}));
  fireEvent.click(screen.getByRole("button",{name:"确认停用"}));
  await waitFor(() => expect(fetch.mock.calls.some(([,init])=>init?.method==="POST")).toBe(true));
  const call=fetch.mock.calls.find(([,init])=>init?.method==="POST")!;
  expect(String(call[0])).toMatch(/\/agents\/product\.content\.agent\/disable$/);
  expect(new Headers(call[1]?.headers).get("If-Match")).toBe('"4"');
});
it("presents one content responsibility and uses only the actually available operation", async () => {
  const content = { ...item, name: "商品内容完善与本地化智能体", parameterSchema: "content-config-v1",
    agent: { ...item.agent, agentId: "product.content.agent", activation: "ENABLED", revision: "1", activationEpoch: "1" },
    canConfigure: false, canUse: true,
  };
  const fetch = vi.fn(async (url: unknown, init?: RequestInit) => { void init; return Response.json({ items: String(url).includes("/agents/mine?") ? [item, content] : [], nextCursor: "" }); });
  vi.stubGlobal("fetch", fetch);
  render(<AgentPage mode="mine" />);
  const card = await screen.findByRole("article", {name:"商品内容与本地化"});
  expect(screen.getAllByRole("heading", { name: "商品内容与本地化" })).toHaveLength(1);
  expect(within(card).getByText("部分已启用")).toBeInTheDocument();
  expect(within(card).getByRole("button", {name:"能力配置"})).toHaveAttribute("aria-haspopup","dialog");
  expect(within(card).getByRole("link", {name:"进入使用"})).toHaveAttribute("href","/workbench/supply/acquisition?agent_kind=content");
  expect(within(card).queryByRole("button", {name:/停用|启用到当前企业/})).toBeNull();
  expect(fetch.mock.calls.every(call => !call[1]?.method || call[1]?.method === "GET")).toBe(true);
});

it("retains separate title and localization configuration and history routes in the existing detail", async () => {
  const content = {...marketEntries()[1],canConfigure:false,agent:{...marketEntries()[1].agent,activation:"ENABLED",revision:"1",activationEpoch:"1"}};
  vi.stubGlobal("fetch",vi.fn(async(url:unknown)=>{
    if(String(url).includes("execution-modes"))return Response.json(modeFixture("product.content.agent",false));
    return Response.json(String(url).endsWith("/agents/product.content.agent")?content:{items:[],nextCursor:""});
  }));
  render(<AgentPage mode="mine" id="product.content.agent"/>);
  const navigation = await screen.findByRole("navigation",{name:"商品内容操作配置"});
  expect(within(navigation).getByRole("link",{name:"标题优化配置与历史"})).toHaveAttribute("href","/workbench/agents/mine/product.title.agent");
  expect(within(navigation).getByRole("link",{name:"内容本地化配置与历史"})).toHaveAttribute("href","/workbench/agents/mine/product.content.agent");
  expect(screen.queryByRole("button",{name:"停用智能体"})).toBeNull();
});
it("configures platform materials with SHEIN only and never requests title knowledge",async()=>{
 Object.defineProperty(HTMLDialogElement.prototype,"showModal",{configurable:true,value:function(this:HTMLDialogElement){this.open=true}});
 Object.defineProperty(HTMLDialogElement.prototype,"close",{configurable:true,value:function(this:HTMLDialogElement){this.open=false}});
 const materials={...item,name:"平台上架资料智能体",parameterSchema:"listing-materials-config-v1",agent:{...item.agent,agentId:"listing.materials.agent",activation:"ENABLED",revision:"1",activationEpoch:"1"}};
 const fetch=vi.fn(async(url:unknown,init?:RequestInit)=>{
  const path=String(url);
  if(init?.method==="POST")return Response.json({commandId:"8fc227bb-b572-4138-8e2a-5f1a0be98617",operation:"create-template",agentId:"listing.materials.agent",templateId:"85cf7a89-4979-411a-a58f-90fd61b7415a",revision:"1",version:"1",noop:false,committedAt:"2026-10-13T00:00:00Z"});
  if(path.includes("/execution-modes/"))return Response.json({subject:{kind:"registered_agent",id:"listing.materials.agent"},definition:{id:"listing.materials.agent",version:"v1.0.0"},mode:"default",revision:"0",policyVersion:"agent-execution-policy-v1",canConfigure:true,updatedAt:"2026-10-13T00:00:00Z",updatedBy:""});
  if(path.includes("/templates?"))return Response.json({items:[],nextCursor:""});
  return Response.json(materials);
 });
 vi.stubGlobal("fetch",fetch);
 render(<AgentPage mode="mine" id="listing.materials.agent"/>);
 fireEvent.click(await screen.findByRole("button",{name:"新建模板"}));
 await screen.findByRole("heading",{name:"保存平台资料模板"});
 expect(screen.queryByLabelText("默认企业知识")).toBeNull();
 expect(screen.queryByRole("option",{name:"TEMU"})).toBeNull();
 expect(screen.getByRole("option",{name:"SHEIN"})).toBeInTheDocument();
 fireEvent.change(screen.getByLabelText("模板名称"),{target:{value:"SHEIN 资料"}});
 fireEvent.click(screen.getByRole("button",{name:"保存模板"}));
 await waitFor(()=>expect(fetch.mock.calls.some(([,init])=>init?.method==="POST")).toBe(true));
 const call=fetch.mock.calls.find(([,init])=>init?.method==="POST")!;
 expect(JSON.parse(String(call[1]?.body))).toEqual({name:"SHEIN 资料",targetPlatform:"shein"});
 expect(fetch.mock.calls.some(([url])=>String(url).includes("knowledge-bases"))).toBe(false);
});
it("drops a late protected response after switching A to B to A", async () => {
  let resolve!: (r: Response) => void;
  const old = new Promise<Response>((r) => {
    resolve = r;
  });
  const fetch = vi
    .fn()
    .mockReturnValueOnce(old)
    .mockResolvedValue(Response.json({ items: [], nextCursor: "" }));
  vi.stubGlobal("fetch", fetch);
  const view = render(<AgentPage mode="mine" />);
  scope.isSwitching = true;
  view.rerender(<AgentPage mode="mine" />);
  scope.effectiveOrganization = { id: "org-b", name: "企业B" };
  scope.isSwitching = false;
  view.rerender(<AgentPage mode="mine" />);
  scope.isSwitching = true;
  view.rerender(<AgentPage mode="mine" />);
  scope.effectiveOrganization = { id: "org-a", name: "企业A" };
  scope.isSwitching = false;
  view.rerender(<AgentPage mode="mine" />);
  await act(async () =>
    resolve(
      Response.json({
        items: [{ ...item, name: "旧企业受保护内容" }],
        nextCursor: "",
      }),
    ),
  );
  expect(screen.queryByText("旧企业受保护内容")).toBeNull();
});

it("shows the required acquisition permission in configuration when title execution cannot use evidence", async () => {
  const detail = {...item,agent:{...item.agent,activation:"ENABLED",revision:"1",activationEpoch:"1"},canConfigure:false,canUse:false,canReadRuns:true,capabilities:[
    {id:"text.generate",support:"REQUIRED",readiness:"AVAILABLE",reason:"模型已接入",observedAt:"2026-10-08T00:00:00Z"},
    {id:"product.source-evidence",support:"REQUIRED",readiness:"REQUIRES_AUTHORIZATION",reason:"标题执行需要1688采集模块权限",observedAt:"2026-10-08T00:00:00Z"},
    {id:"knowledge.context",support:"OPTIONAL",readiness:"UNAVAILABLE",reason:"未接入",observedAt:"2026-10-08T00:00:00Z"},
    {id:"image.generate",support:"NOT_SUPPORTED",readiness:"UNAVAILABLE",reason:"未提供",observedAt:"2026-10-08T00:00:00Z"},
    {id:"platform.write",support:"NOT_SUPPORTED",readiness:"UNAVAILABLE",reason:"未提供",observedAt:"2026-10-08T00:00:00Z"},
  ]};
  vi.stubGlobal("fetch",vi.fn(async(url:unknown)=>{
    if(String(url).includes("execution-modes"))return Response.json(modeFixture("product.title.agent",false));
    return Response.json(String(url).endsWith("/agents/product.title.agent")?detail:{items:[],nextCursor:""});
  }));
  render(<AgentPage mode="mine" id="product.title.agent"/>);
  expect(await screen.findByText("标题执行需要1688采集模块权限")).toBeInTheDocument();
  expect(screen.getByRole("heading",{name:"采集证据读取 · 必需"})).toBeInTheDocument();
  expect(screen.queryByRole("link",{name:"选择商品并进入使用 →"})).toBeNull();
});
