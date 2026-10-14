import { afterEach, expect, it, vi } from "vitest";
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
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
  default: ({ children, href, ...props }: React.ComponentProps<"a">) => (
    <a href={href} {...props}>
      {children}
    </a>
  ),
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
afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
  scope.effectiveOrganization = { id: "org-a", name: "企业A" };
  scope.isSwitching = false;
});

it.each([
  ["market", "product.selection.agent", "product.selection.inspect", "product-selection-config-v1", "查看商品事实 →"],
  ["mine", "product.selection.agent", "product.selection.inspect", "product-selection-config-v1", "查看商品事实 →"],
  ["market", "shop.operations-diagnosis.agent", "shop.operations-diagnosis.inspect", "shop-operations-diagnosis-config-v1", "查看经营事实 →"],
  ["mine", "shop.operations-diagnosis.agent", "shop.operations-diagnosis.inspect", "shop-operations-diagnosis-config-v1", "查看经营事实 →"],
] as const)("exposes authorized facts from %s for %s before activation without enabling analysis", async (mode, agentId, capabilityId, parameterSchema, label) => {
  const fetch = vi.fn(async (_url: unknown, _init?: RequestInit) => { void _url; void _init; return Response.json({items: [{...item, parameterSchema, agent: {...item.agent, agentId}, capabilities: [{id: capabilityId, support: "REQUIRED", readiness: "AVAILABLE", reason: "", observedAt: "2026-10-14T00:00:00Z"}]}], nextCursor: ""}); });
  vi.stubGlobal("fetch", fetch);
  render(<AgentPage mode={mode}/>);
  expect(await screen.findByRole("link", {name: label})).toHaveAttribute("href", `/workbench/agents/mine/${agentId}`);
  expect(screen.queryByRole("link", {name: "进入使用 →"})).toBeNull();
  expect(screen.getByText("尚未启用")).toBeInTheDocument();
  expect(fetch.mock.calls.every(([, init]) => !init?.method || init.method === "GET")).toBe(true);
});

it.each([
  ["product.selection.agent", "product.selection.inspect", "product-selection-config-v1", "查看商品事实 →"],
  ["shop.operations-diagnosis.agent", "shop.operations-diagnosis.inspect", "shop-operations-diagnosis-config-v1", "查看经营事实 →"],
])("does not expose %s inspection when its current capability denies access", async (agentId, capabilityId, parameterSchema, label) => {
  vi.stubGlobal("fetch", vi.fn(async () => Response.json({items: [{...item, parameterSchema, agent: {...item.agent, agentId}, capabilities: [{id: capabilityId, support: "REQUIRED", readiness: "REQUIRES_AUTHORIZATION", reason: "需要授权", observedAt: "2026-10-14T00:00:00Z"}]}], nextCursor: ""})));
  render(<AgentPage mode="market"/>);
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
    return Response.json({
      items: [
        enabled
          ? {
              ...item,
              agent: {
                ...item.agent,
                activation: "ENABLED",
                revision: "1",
                activationEpoch: "1",
              },
            }
          : item,
      ],
      nextCursor: "",
    });
  });
  vi.stubGlobal("fetch", fetch);
  render(<AgentPage mode="market" />);
  fireEvent.click(
    await screen.findByRole("button", { name: "启用到当前企业" }),
  );
  await waitFor(() => expect(screen.getByText("已启用")).toBeInTheDocument());
  const call = fetch.mock.calls.find(([, init]) => init?.method === "POST")!;
  expect(new Headers(call[1]?.headers).get("If-None-Match")).toBe("*");
});

it("presents title and localization as operations of one content responsibility without merging their permissions", async () => {
  const content = { ...item, name: "商品内容完善与本地化智能体", parameterSchema: "content-config-v1",
    agent: { ...item.agent, agentId: "product.content.agent", activation: "ENABLED", revision: "1", activationEpoch: "1" },
    canConfigure: false, canUse: true,
  };
  const fetch = vi.fn(async (_url: unknown, init?: RequestInit) => { void init; return Response.json({ items: [item, content], nextCursor: "" }); });
  vi.stubGlobal("fetch", fetch);
  render(<AgentPage mode="mine" />);
  await screen.findByRole("heading", { name: "商品内容与本地化" });
  expect(screen.getAllByRole("heading", { name: "商品内容与本地化" })).toHaveLength(1);
  expect(screen.queryByRole("heading", { name: "商品标题优化智能体" })).toBeNull();
  expect(screen.getByRole("heading", { name: "标题优化" })).toBeInTheDocument();
  expect(screen.getByRole("heading", { name: "内容完善与本地化" })).toBeInTheDocument();
  expect(screen.getByRole("link", { name: "标题优化配置与历史" })).toHaveAttribute("href", "/workbench/agents/mine/product.title.agent");
  expect(screen.getByRole("link", { name: "内容本地化配置与历史" })).toHaveAttribute("href", "/workbench/agents/mine/product.content.agent");
  expect(fetch.mock.calls.every(call => !call[1]?.method || call[1]?.method === "GET")).toBe(true);
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

it("shows the required acquisition permission when title execution cannot use evidence", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn(async () =>
      Response.json({
        items: [
          {
            ...item,
            agent: {
              ...item.agent,
              activation: "ENABLED",
              revision: "1",
              activationEpoch: "1",
            },
            canConfigure: false,
            canUse: false,
            canReadRuns: true,
            capabilities: [
              {
                id: "text.generate",
                support: "REQUIRED",
                readiness: "AVAILABLE",
                reason: "模型已接入",
                observedAt: "2026-10-08T00:00:00Z",
              },
              {
                id: "product.source-evidence",
                support: "REQUIRED",
                readiness: "REQUIRES_AUTHORIZATION",
                reason: "标题执行需要1688采集模块权限",
                observedAt: "2026-10-08T00:00:00Z",
              },
              {
                id: "knowledge.context",
                support: "OPTIONAL",
                readiness: "UNAVAILABLE",
                reason: "未接入",
                observedAt: "2026-10-08T00:00:00Z",
              },
              {
                id: "image.generate",
                support: "NOT_SUPPORTED",
                readiness: "UNAVAILABLE",
                reason: "未提供",
                observedAt: "2026-10-08T00:00:00Z",
              },
              {
                id: "platform.write",
                support: "NOT_SUPPORTED",
                readiness: "UNAVAILABLE",
                reason: "未提供",
                observedAt: "2026-10-08T00:00:00Z",
              },
            ],
          },
        ],
        nextCursor: "",
      }),
    ),
  );
  render(<AgentPage mode="mine" />);
  expect(await screen.findByText("标题执行需要1688采集模块权限")).toBeTruthy();
  expect(screen.getByText("采集证据读取")).toBeTruthy();
});
