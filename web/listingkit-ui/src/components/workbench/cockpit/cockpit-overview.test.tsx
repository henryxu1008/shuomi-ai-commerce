import { act, cleanup, render, screen, waitFor, within } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { afterEach, expect, it, vi } from "vitest";
import userEvent from "@testing-library/user-event";
import { CockpitEntry } from "../console/cockpit-entry";
import { cockpitRequest } from "@/lib/api/operations-cockpit";
import { readSavedOrderAmount } from "@/lib/api/shein-order-amount";
import { readTikTokOrderAmount } from "@/lib/api/tiktok-order-amount";
import { readWalmartOrderAmount } from "@/lib/api/walmart-order-amount";
import { readCockpitMetrics } from "@/lib/api/cockpit-metrics";
const state = vi.hoisted(() => ({ context: { user: { id: "user-a" }, effectiveOrganization: { id: "org-a" }, permissions: ["workbench.cockpit.stores.read"], operationsCockpitAvailable: true, isLoading: false, isSwitching: false, selectionRequired: false, error: null, blockingError: null } }));
vi.mock("@/components/providers/workbench-context-provider", () => ({ useWorkbenchContext: () => state.context }));
vi.mock("@/lib/api/operations-cockpit", async original => ({ ...await original<typeof import("@/lib/api/operations-cockpit")>(), cockpitRequest: vi.fn() }));
vi.mock("@/lib/api/cockpit-metrics", () => ({ readCockpitMetrics: vi.fn() }));
vi.mock("@/components/ui/chart", () => ({ EChart: ({ ariaLabel }: { ariaLabel: string }) => <div role="img" aria-label={ariaLabel} /> }));
vi.mock("@/lib/api/shein-order-amount", () => ({ readSavedOrderAmount: vi.fn() }));
vi.mock("@/lib/api/tiktok-order-amount", () => ({ readTikTokOrderAmount: vi.fn() }));
vi.mock("@/lib/api/walmart-order-amount", () => ({ readWalmartOrderAmount: vi.fn() }));
const clients: QueryClient[] = [];
const access = { goalsRead: false, goalsCreate: false, goalsManage: false, storesRead: true, factsWrite: false, alertsRead: false, adviceRead: false };
const totals = { revenue: 120000, refunds: 0, procurement: 100000, logistics: 0, platform: 0, advertising: 0, other: 0, netRevenue: 120000, netProfit: 20000, margin: null };
const capabilities = { access, today: "2026-10-11", stores: [] };
const matrix = { rows: [], total: 0, page: 1, summary: totals, summaryComplete: false, states: { loss: 0, break_even: 0, recorded_complete: 0, data_incomplete: 0 }, capturedAt: "2026-10-11T00:00:00Z" };
function mount() { const client = new QueryClient({ defaultOptions: { queries: { retry: false } } }); clients.push(client); const tree = () => <QueryClientProvider client={client}><CockpitEntry /></QueryClientProvider>; const view = render(tree()); return { ...view, update: () => view.rerender(tree()) }; }
afterEach(() => { cleanup(); clients.splice(0).forEach(c => c.clear()); vi.mocked(cockpitRequest).mockReset(); vi.mocked(readCockpitMetrics).mockReset(); vi.mocked(readSavedOrderAmount).mockReset(); vi.mocked(readWalmartOrderAmount).mockReset(); state.context.permissions = ["workbench.cockpit.stores.read"]; state.context.isSwitching = false; state.context.effectiveOrganization = { id: "org-a" }; });
it("updates listing and task metrics together when changing both date and store", async () => {
  const first = "123e4567-e89b-42d3-a456-426614174000";
  const second = "123e4567-e89b-42d3-a456-426614174001";
  const store = { id: first, name: "店铺一", platform: "shein", region: "US", status: "active" };
  const capturedAt = "2026-10-11T00:00:00Z";
  state.context.permissions = ["workbench.cockpit.stores.read", "workbench.supply.read", "workbench.task.read"];
  vi.mocked(cockpitRequest).mockImplementation(async (_scope,path) => path === "capabilities"
    ? { ...capabilities, stores: [store, { ...store, id: second, name: "店铺二" }] } : matrix);
  const taskResult = (errors: number) => ({
    result: { status: "complete", capturedAt, counts: { RUNNING: 1, WAITING_CONFIRMATION: 0, COMPLETED: 0, ERROR: errors, PAUSED: 0, UNKNOWN: 0 } },
  });
  let resolveFilteredTask!: (result: ReturnType<typeof taskResult>) => void;
  vi.mocked(readCockpitMetrics).mockImplementation(async (_scope, owner, query) => {
    const params = new URLSearchParams(query);
    const isYesterday = params.get("startDate") === "2026-10-10";
    const oneStore = params.getAll("storeId").length === 1;
    if (owner === "tasks" && isYesterday && oneStore) return new Promise(resolve => { resolveFilteredTask = resolve; });
    if (owner === "tasks") return taskResult(isYesterday ? 2 : 5);
    if (String(owner) === "shein-publish") return { result: { status: "complete", capturedAt, counts: { confirmed: 0, unknown: 0 } } };
    return { result: { status: "complete", capturedAt, counts: { pending: oneStore ? 1 : isYesterday ? 3 : 9, review: oneStore ? 0 : isYesterday ? 1 : 4 } } };
  });

  const cardValue = (name: string) => screen.getByRole("heading", { name }).parentElement?.parentElement?.querySelector("strong");
  mount();
  await screen.findByRole("img", { name: "执行状态分布，共 6 条任务" });
  expect(cardValue("待处理商品")).toHaveTextContent("9");
  expect(cardValue("待审核商品")).toHaveTextContent("4");
  expect(cardValue("异常任务")).toHaveTextContent("5");

  await userEvent.click(screen.getByRole("button", { name: "昨日" }));
  await screen.findByRole("img", { name: "执行状态分布，共 3 条任务" });
  expect(cardValue("待处理商品")).toHaveTextContent("3");
  expect(cardValue("待审核商品")).toHaveTextContent("1");
  expect(cardValue("异常任务")).toHaveTextContent("2");

  await userEvent.selectOptions(screen.getByRole("combobox", { name: "店铺" }), second);
  await waitFor(() => expect(resolveFilteredTask).toBeDefined());
  for (const name of ["待处理商品", "待审核商品", "异常任务"]) expect(cardValue(name)).toHaveTextContent("—");
  expect(screen.queryByRole("img", { name: /执行状态分布/ })).not.toBeInTheDocument();
  await act(async () => resolveFilteredTask(taskResult(0)));
  await screen.findByRole("img", { name: "执行状态分布，共 1 条任务" });
  expect(cardValue("待处理商品")).toHaveTextContent("1");
  expect(cardValue("待审核商品")).toHaveTextContent("0");
  expect(cardValue("异常任务")).toHaveTextContent("0");
  const ringLegend = screen.getByRole("list", { name: "任务状态数量" });
  expect(within(ringLegend).getByText("异常").parentElement).toHaveTextContent("异常0");
  const calls = vi.mocked(readCockpitMetrics).mock.calls;
  expect(calls.slice(-4).map(([,owner,query]) => [owner,query])).toEqual([
    ["supply-preparations", `startDate=2026-10-10&endDate=2026-10-10&storeId=${second}`],
    ["tasks", `startDate=2026-10-10&endDate=2026-10-10&storeId=${second}`],
    ["shein-publish", `startDate=2026-10-10&endDate=2026-10-10&storeId=${second}`],
    ["submission-overview", `startDate=2026-10-10&endDate=2026-10-10&storeId=${second}`],
  ]);
  expect(screen.getByText("GMV 数据待接入")).toBeVisible();
  expect(screen.getByRole("heading", {name:"待核实操作次数"})).toBeVisible();
});
it("does not advertise or request oversized detail scopes and restores a valid single-store link", async () => {
  const stores=Array.from({length:400},(_,i)=>({id:`${(i+1).toString(16).padStart(8,"0")}-1111-4111-8111-111111111111`,name:`Store ${i}`,platform:"shein",region:"US"}));
  vi.mocked(cockpitRequest).mockImplementation(async (_scope,path)=>path==="capabilities"?{...capabilities,stores}:matrix);
  mount();
  await userEvent.selectOptions(await screen.findByRole("combobox",{name:"平台"}),"shein");
  expect(screen.getAllByText("范围较大，请按店铺筛选").length).toBeGreaterThan(0);
  expect(screen.queryByRole("link",{name:/查看真实经营明细/})).not.toBeInTheDocument();
  expect(vi.mocked(cockpitRequest).mock.calls.filter(call=>call[1].startsWith("stores?")).every(call=>call[1].length<16000)).toBe(true);
  await userEvent.selectOptions(screen.getByRole("combobox",{name:"店铺"}),stores[0].id);
  const href=screen.getByRole("link",{name:/查看真实经营明细/}).getAttribute("href")!;
  expect(new URL(href,"https://localhost").searchParams.getAll("storeId")).toEqual([stores[0].id]);
});
it("does not reinterpret partial manual totals as full profit or GMV", async () => {
  vi.mocked(cockpitRequest).mockImplementation(async (_scope, path) => path === "capabilities" ? capabilities : matrix);
  mount();
  await screen.findByText("待完整数据，未按部分录入推定完整利润");
  expect(screen.queryByText("¥200.00")).not.toBeInTheDocument();
  expect(screen.getByText("GMV 数据待接入")).toBeVisible();
  expect(vi.mocked(cockpitRequest).mock.calls.every(call => call[1] === "capabilities" || call[1].startsWith("stores?"))).toBe(true);
});
it("uses real enabled stores and only queries completed days when changing the period or refreshing", async () => {
  const store = { id: "123e4567-e89b-42d3-a456-426614174000", name: "真实店铺", platform: "shein", region: "US", status: "active" };
  vi.mocked(cockpitRequest).mockImplementation(async (_scope, path) => path === "capabilities" ? {
    ...capabilities, stores: [store, { ...store, id: "123e4567-e89b-42d3-a456-426614174001", status: "disabled" }],
  } : matrix);
  mount();
  await screen.findByRole("button", { name: "刷新" });
  const active = screen.getByRole("heading", { name: "活跃店铺" });
  expect(active.parentElement?.parentElement?.querySelector("strong")).toHaveTextContent("1");
  expect(screen.getByRole("heading", { name: "跨境电商业务链路" })).toBeVisible();
  expect(screen.getByRole("heading", { name: "执行状态分布" })).toBeVisible();
  await userEvent.click(screen.getByRole("button", { name: "昨日" }));
  await waitFor(() => expect(vi.mocked(cockpitRequest).mock.calls.at(-1)?.[1])
    .toContain("startDate=2026-10-10&endDate=2026-10-10"));
  const requests = vi.mocked(cockpitRequest).mock.calls.filter(call => call[1] === "capabilities").length;
  await userEvent.click(screen.getByRole("button", { name: "刷新" }));
  await waitFor(() => expect(vi.mocked(cockpitRequest).mock.calls.filter(call => call[1] === "capabilities")).toHaveLength(requests + 1));
  expect(screen.getByRole("button", { name: "昨日" })).toHaveAttribute("aria-pressed", "true");
});
it("cancels the old scope and never displays its late profit after switching enterprise", async () => {
  let resolve!: (value: unknown) => void, oldSignal: AbortSignal | undefined;
  vi.mocked(cockpitRequest).mockImplementation(async (scope, path, _schema, init) => {
    if (path === "capabilities") return capabilities;
    if (scope.organizationId === "org-a") { oldSignal = init?.signal as AbortSignal; return new Promise(done => { resolve = done; }); }
    return matrix;
  });
  const view = mount();
  await waitFor(() => expect(oldSignal).toBeDefined());
  state.context.isSwitching = true; view.update();
  expect(oldSignal?.aborted).toBe(true);
  expect(screen.queryByText("净利润")).not.toBeInTheDocument();
  state.context.isSwitching = false; state.context.effectiveOrganization = { id: "org-b" }; view.update();
  await screen.findByText("待完整数据，未按部分录入推定完整利润");
  await act(async () => resolve({ ...matrix, summaryComplete: true }));
  expect(screen.queryByText("¥200.00")).not.toBeInTheDocument();
});
it.each([false, true])("preserves unavailable goal evaluation alongside actual rules (%s)", async (hasRules) => {
  const rule = { id: "rule-a", title: "真实经营预警", reason: "已保存成本", advice: "核对成本", level: "attention", actionPath: "/workbench/overview/stores", actionLabel: "查看" };
  vi.mocked(cockpitRequest).mockImplementation(async (_scope, path) => path === "capabilities" ? { ...capabilities, access: { ...access, alertsRead: true, adviceRead: true } } : path.startsWith("stores?") ? matrix : { rules: hasRules ? [rule] : [], goalUnavailable: true });
  mount();
  await waitFor(() => expect(screen.getAllByText("企业当前目标或关联店铺不可访问，目标评判暂不可用。")).toHaveLength(2));
  expect(screen.queryByText("当前范围暂无规则预警")).not.toBeInTheDocument();
  expect(screen.queryByText("当前范围暂无人工处理建议")).not.toBeInTheDocument();
  if (hasRules) expect(screen.getAllByText("真实经营预警")).toHaveLength(2);
});

it("shows a separate Feed rate only for an explicitly selected Walmart store and reloads it with the date", async () => {
  const walmart = { id: "123e4567-e89b-42d3-a456-426614174000", name: "Walmart 验收店", platform: "walmart", region: "US", status: "active" };
  const shein = { ...walmart, id: "123e4567-e89b-42d3-a456-426614174001", name: "SHEIN 验收店", platform: "shein" };
  const title = "Walmart Feed 提交受理率";
  const capturedAt = "2026-10-11T00:00:00Z";
  state.context.permissions.push("workbench.supply.read");
  vi.mocked(cockpitRequest).mockImplementation(async (_scope, path) => path === "capabilities"
    ? { ...capabilities, stores: [walmart, shein] } : matrix);
  vi.mocked(readCockpitMetrics).mockImplementation(async (_scope, owner, query) => {
    const yesterday = new URLSearchParams(query).get("startDate") === "2026-10-10";
    const counts = String(owner) === "walmart-feed"
      ? { accepted: yesterday ? 0 : 1, rejected: yesterday ? 0 : 1, unknown: 1 }
      : { pending: 2, review: 1 };
    return { result: { status: "complete", capturedAt, counts } };
  });
  const user = userEvent.setup();
  mount();
  const shop = await screen.findByRole("combobox", { name: "店铺" });
  expect(screen.queryByRole("heading", { name: title })).not.toBeInTheDocument();
  await user.selectOptions(shop, walmart.id);
  await screen.findByRole("heading", { name: title });
  expect(await screen.findByText("50.0%")).toBeVisible();
  await user.click(screen.getByRole("button", { name: /^昨日$/ }));
  expect(await screen.findByText("暂无已确认结果")).toBeVisible();
  expect(screen.queryByText("50.0%")).not.toBeInTheDocument();
  expect(vi.mocked(readCockpitMetrics).mock.calls.some(([, owner, query]) => String(owner) === "walmart-feed"
    && query === "startDate=2026-10-10&endDate=2026-10-10&storeId=" + walmart.id)).toBe(true);
  await user.selectOptions(shop, shein.id);
  expect(screen.queryByRole("heading", { name: title })).not.toBeInTheDocument();
  expect(screen.getByRole("heading", {name:"待核实操作次数"})).toBeVisible();
  expect(screen.getByText("GMV 数据待接入")).toBeVisible();
});
it("shows the TikTok US submission rate only for the selected TikTok shop", async () => {
  const tikTok = { id: "123e4567-e89b-42d3-a456-426614174000", name: "TikTok 验收店", platform: "tiktokshop", region: "US", status: "active" };
  const shein = { ...tikTok, id: "123e4567-e89b-42d3-a456-426614174001", name: "SHEIN 验收店", platform: "shein" };
  const title = "TikTok Shop US 提交审核率";
  const capturedAt = "2026-10-11T00:00:00Z";
  state.context.permissions.push("workbench.supply.read");
  vi.mocked(cockpitRequest).mockImplementation(async (_scope, path) => path === "capabilities"
    ? { ...capabilities, stores: [tikTok, shein] } : matrix);
  vi.mocked(readCockpitMetrics).mockImplementation(async (_scope, owner, query) => {
    const yesterday = new URLSearchParams(query).get("startDate") === "2026-10-10";
    const counts = String(owner) === "tiktok-publish"
      ? { accepted: yesterday ? 0 : 1, rejected: yesterday ? 0 : 2, unknown: 1 }
      : { pending: 2, review: 1 };
    return { result: { status: "complete", capturedAt, counts } };
  });
  const user = userEvent.setup();
  mount();
  const shop = await screen.findByRole("combobox", { name: "店铺" });
  expect(screen.queryByRole("heading", { name: title })).not.toBeInTheDocument();
  await user.selectOptions(shop, tikTok.id);
  await screen.findByRole("heading", { name: title });
  expect(await screen.findByText("33.3%")).toBeVisible();
  await user.click(screen.getByRole("button", { name: /^昨日$/ }));
  expect(await screen.findByText("暂无已确认结果")).toBeVisible();
  expect(screen.queryByText("33.3%")).not.toBeInTheDocument();
  await user.selectOptions(shop, shein.id);
  expect(screen.queryByRole("heading", { name: title })).not.toBeInTheDocument();
  expect(screen.getByRole("heading", {name:"待核实操作次数"})).toBeVisible();
  expect(screen.getByText("GMV 数据待接入")).toBeVisible();
});
it("shows SHEIN US publication counts for the selected shop and dates", async () => {
  const storeA = { id: "123e4567-e89b-42d3-a456-426614174000", name: "SHEIN 验收店 A", platform: "shein", region: "US", status: "active" };
  const storeB = { ...storeA, id: "123e4567-e89b-42d3-a456-426614174001", name: "SHEIN 验收店 B" };
  const title = "SHEIN US 商品发布";
  state.context.permissions.push("workbench.supply.read");
  vi.mocked(cockpitRequest).mockImplementation(async (_scope, path) => path === "capabilities"
    ? { ...capabilities, stores: [storeA, storeB] } : matrix);
  vi.mocked(readCockpitMetrics).mockImplementation(async (_scope, owner, query) => {
    const params = new URLSearchParams(query);
    const yesterday = params.get("startDate") === "2026-10-10";
    const counts = String(owner) === "shein-publish"
      ? { confirmed: params.get("storeId") === storeB.id ? 0 : yesterday ? 1 : 2, unknown: 1 }
      : { pending: 2, review: 1 };
    return { result: { status: "complete", capturedAt: "2026-10-11T00:00:00Z", counts } };
  });
  const user = userEvent.setup();
  mount();
  const shop = await screen.findByRole("combobox", { name: "店铺" });
  expect(screen.queryByRole("heading", { name: title })).not.toBeInTheDocument();
  await user.selectOptions(shop, storeA.id);
  const panel = await screen.findByRole("heading", { name: title });
  const card = panel.parentElement!.parentElement!;
  expect(await within(card).findByText("2")).toBeVisible();
  await user.click(screen.getByRole("button", { name: /^昨日$/ }));
  await waitFor(() => expect(within(card).getAllByText("1")).toHaveLength(2));
  await user.selectOptions(shop, storeB.id);
  expect(await within(card).findByText("0")).toBeVisible();
  expect(within(card).getByText("已确认提交")).toBeVisible();
  expect(within(card).getByText("结果待核实")).toBeVisible();
  expect(within(card).queryByText(/%/)).not.toBeInTheDocument();
});
it("shows Temu creation counts for the selected shop and dates", async () => {
  const storeA = { id: "123e4567-e89b-42d3-a456-426614174000", name: "Temu 验收店 A", platform: "temu", region: "DE", status: "active" };
  const storeB = { ...storeA, id: "123e4567-e89b-42d3-a456-426614174001", name: "Temu 验收店 B" };
  const title = "Temu 商品创建";
  state.context.permissions.push("workbench.supply.read");
  vi.mocked(cockpitRequest).mockImplementation(async (_scope, path) => path === "capabilities"
    ? { ...capabilities, stores: [storeA, storeB] } : matrix);
  vi.mocked(readCockpitMetrics).mockImplementation(async (_scope, owner, query) => {
    const params = new URLSearchParams(query);
    const yesterday = params.get("startDate") === "2026-10-10";
    const counts = String(owner) === "temu-create"
      ? { confirmed: params.get("storeId") === storeB.id ? 0 : yesterday ? 1 : 2, unknown: 1 }
      : { pending: 2, review: 1 };
    return { result: { status: "complete", capturedAt: "2026-10-11T00:00:00Z", counts } };
  });
  const user = userEvent.setup();
  mount();
  const shop = await screen.findByRole("combobox", { name: "店铺" });
  expect(screen.queryByRole("heading", { name: title })).not.toBeInTheDocument();
  await user.selectOptions(shop, storeA.id);
  const panel = await screen.findByRole("heading", { name: title });
  const card = panel.parentElement!.parentElement!;
  expect(await within(card).findByText("2")).toBeVisible();
  await user.click(screen.getByRole("button", { name: /^昨日$/ }));
  await waitFor(() => expect(within(card).getAllByText("1")).toHaveLength(2));
  await user.selectOptions(shop, storeB.id);
  expect(await within(card).findByText("0")).toBeVisible();
  expect(within(card).getByText("已确认创建")).toBeVisible();
  expect(within(card).getByText("结果待核实")).toBeVisible();
  expect(within(card).queryByText(/%/)).not.toBeInTheDocument();
});

it("shows native platform quantities and only a complete unknown-operation total", async () => {
 const store={id:"123e4567-e89b-42d3-a456-426614174000",name:"Walmart 一店",platform:"walmart",region:"US",status:"active"};
 state.context.permissions.push("workbench.supply.read");
 vi.mocked(cockpitRequest).mockImplementation(async (_scope,path)=>path==="capabilities"?{...capabilities,stores:[store]}:matrix);
 vi.mocked(readCockpitMetrics).mockImplementation(async (_scope,owner,query)=>{
  const yesterday=new URLSearchParams(query).get("startDate")==="2026-10-10";
  if (String(owner)==="submission-overview") return {result:{status:yesterday?"incomplete":"complete",capturedAt:"2026-10-11T00:00:00Z",excludedStores:0,
   ...(!yesterday?{unknownTotal:3}:{}),rows:[
    {platform:"walmart",status:yesterday?"limited":"complete",stores:1,...(!yesterday?{counts:{confirmed:2,rejected:1,unknown:3}}:{})},
    ...["tiktokshop","shein","temu"].map(platform=>({platform,status:"empty",stores:0})),
   ]}};
  return {result:{status:"complete",capturedAt:"2026-10-11T00:00:00Z",counts:{pending:0,review:0}}};
 });
 mount();
 await screen.findByRole("heading",{name:"提交与创建概览"});
 const panel=await screen.findByRole("region",{name:"各平台提交与创建数量"});
 await waitFor(()=>expect(panel).toHaveTextContent("已受理2"));
 expect(panel).toHaveTextContent("明确拒绝1");
 expect(panel).toHaveTextContent("结果未知3");
 expect(panel).toHaveTextContent("口径未覆盖");
 const card=screen.getByRole("heading",{name:"待核实操作次数"}).parentElement?.parentElement;
 expect(card?.querySelector("strong")).toHaveTextContent("3");
 expect(screen.queryByText(/%/)).not.toBeInTheDocument();
 await userEvent.click(screen.getByRole("button",{name:"昨日"}));
 await screen.findByText("记录较多，请缩小日期或店铺范围");
 expect(card?.querySelector("strong")).toHaveTextContent("—");
 expect(screen.getByRole("region",{name:"各平台提交与创建数量"})).not.toHaveTextContent("已受理2");
 expect(screen.getByText("GMV 数据待接入")).toBeVisible();
});

it("reads saved Walmart amounts by the same date and store and hides the previous value",async()=>{
 const a={id:"123e4567-e89b-42d3-a456-426614174000",name:"Walmart A",platform:"walmart",region:"US",status:"active"};
 const b={...a,id:"123e4567-e89b-42d3-a456-426614174001",name:"Walmart B"};
 state.context.permissions.push("workbench.store.orders.read");
 vi.mocked(cockpitRequest).mockImplementation(async(_scope,path)=>path==="capabilities"?{...capabilities,stores:[a,b]}:matrix);
 const source={syncId:a.id,observedAt:"2026-10-11T00:00:00Z",range:{start:"2026-09-11T16:00:00Z",end:"2026-10-10T16:00:00Z"}};
 vi.mocked(readWalmartOrderAmount).mockImplementation(async(_scope,query)=>{
  const p=new URLSearchParams(query);
  if(p.get("storeId")===b.id)return {result:{status:"coverage_incomplete",capturedAt:"2026-10-11T00:00:00Z",source,reason:"range_not_covered"}};
  if(p.get("startDate")==="2026-10-10")return {result:{status:"missing_fields",capturedAt:"2026-10-11T00:00:00Z",source,matchedOrderCount:1,missing:{amountCount:1,currencyCount:0,unknownTimeCount:0}}};
  return {result:{status:"complete",capturedAt:"2026-10-11T00:00:00Z",source,matchedOrderCount:1,amounts:[{currency:"USD",amount:"19.90",orderCount:1}]}};
 });
 mount();const shop=await screen.findByRole("combobox",{name:"店铺"});
 await userEvent.selectOptions(shop,a.id);await screen.findByText("19.90");
 expect(screen.getByText(/不再乘订单行数量/)).toBeVisible();
 await userEvent.click(screen.getByRole("button",{name:"昨日"}));await screen.findByText(/缺金额 1 笔/);expect(screen.queryByText("19.90")).not.toBeInTheDocument();
 await userEvent.selectOptions(shop,b.id);await screen.findByText("原同步范围未覆盖所选完整日期，暂不显示金额。");
 expect(new URLSearchParams(vi.mocked(readWalmartOrderAmount).mock.calls.at(-1)![1]).get("storeId")).toBe(b.id);
 expect(screen.getByText("GMV 数据待接入")).toBeVisible();expect(vi.mocked(readSavedOrderAmount)).not.toHaveBeenCalled();
});
it("shows saved SHEIN order amounts separately and hides stale amounts for missing fields or incomplete coverage",async()=>{
 const first={id:"123e4567-e89b-42d3-a456-426614174000",name:"SHEIN 金额店 A",platform:"shein",region:"US",status:"active"};
 const second={...first,id:"123e4567-e89b-42d3-a456-426614174001",name:"SHEIN 金额店 B"};
 state.context.permissions.push("workbench.store.orders.read");
 vi.mocked(cockpitRequest).mockImplementation(async(_scope,path)=>path==="capabilities"?{...capabilities,stores:[first,second]}:matrix);
 const source={syncId:first.id,observedAt:"2026-10-11T00:00:00Z",range:{start:"2026-09-11T16:00:00Z",end:"2026-10-10T15:59:59Z"}};
 vi.mocked(readSavedOrderAmount).mockImplementation(async(_scope,query)=>{
  const params=new URLSearchParams(query);
  if(params.get("storeId")===second.id)return {result:{status:"coverage_incomplete",capturedAt:"2026-10-11T00:00:00Z",source,reason:"range_not_covered"}};
  if(params.get("startDate")==="2026-10-10")return {result:{status:"missing_fields",capturedAt:"2026-10-11T00:00:00Z",source,matchedOrderCount:1,missing:{amountCount:1,currencyCount:0,unknownTimeCount:0}}};
  return {result:{status:"complete",capturedAt:"2026-10-11T00:00:00Z",source,matchedOrderCount:2,amounts:[{currency:"USD",amount:"0.30",orderCount:2}]}};
 });
 mount();
 const shop=await screen.findByRole("combobox",{name:"店铺"});
 expect(vi.mocked(readSavedOrderAmount)).not.toHaveBeenCalled();
 await userEvent.selectOptions(shop,first.id);
 const title=await screen.findByRole("heading",{name:"已保存订单商品金额（原币）"});
 await screen.findByText("0.30");
 expect(title).toBeVisible();
 await userEvent.click(screen.getByRole("button",{name:"昨日"}));
 await screen.findByText(/缺金额 1 笔/);
 expect(screen.queryByText("0.30")).not.toBeInTheDocument();
 await userEvent.selectOptions(shop,second.id);
 await screen.findByText("原同步范围未覆盖所选完整日期，暂不显示金额。");
 expect(screen.queryByText("0.30")).not.toBeInTheDocument();
 expect(new URLSearchParams(vi.mocked(readSavedOrderAmount).mock.calls.at(-1)![1]).get("storeId")).toBe(second.id);
 expect(screen.getByText("GMV 数据待接入")).toBeVisible();
 await userEvent.selectOptions(shop,"");
 expect(screen.queryByRole("heading",{name:"已保存订单商品金额（原币）"})).not.toBeInTheDocument();
});

it("reads saved TikTok amounts by the same date and store and hides the previous value",async()=>{
 const a={id:"123e4567-e89b-42d3-a456-426614174000",name:"TikTok A",platform:"tiktokshop",region:"US",status:"active"};
 const b={...a,id:"123e4567-e89b-42d3-a456-426614174001",name:"TikTok B"};
 state.context.permissions.push("workbench.store.orders.read");
 vi.mocked(cockpitRequest).mockImplementation(async(_scope,path)=>path==="capabilities"?{...capabilities,stores:[a,b]}:matrix);
 const source={syncId:a.id,observedAt:"2026-10-11T00:00:00Z",range:{start:"2026-09-11T16:00:00Z",end:"2026-10-10T16:00:00Z"}};
 vi.mocked(readTikTokOrderAmount).mockImplementation(async(_scope,query)=>{
  const p=new URLSearchParams(query);
  if(p.get("storeId")===b.id)return {result:{status:"coverage_incomplete",capturedAt:"2026-10-11T00:00:00Z",source,reason:"range_not_covered"}};
  if(p.get("startDate")==="2026-10-10")return {result:{status:"missing_fields",capturedAt:"2026-10-11T00:00:00Z",source,matchedOrderCount:1,missing:{amountCount:1,currencyCount:0,unknownTimeCount:0}}};
  return {result:{status:"complete",capturedAt:"2026-10-11T00:00:00Z",source,matchedOrderCount:1,amounts:[{currency:"USD",amount:"19.90",orderCount:1}]}};
 });
 mount();const shop=await screen.findByRole("combobox",{name:"店铺"});
 await userEvent.selectOptions(shop,a.id);await screen.findByText("19.90");
 expect(screen.getByText(/累计已保存的原商品小计/)).toBeVisible();
 await userEvent.click(screen.getByRole("button",{name:"昨日"}));await screen.findByText(/缺金额 1 笔/);expect(screen.queryByText("19.90")).not.toBeInTheDocument();
 await userEvent.selectOptions(shop,b.id);await screen.findByText("原同步范围未覆盖所选完整日期，暂不显示金额。");
 expect(new URLSearchParams(vi.mocked(readTikTokOrderAmount).mock.calls.at(-1)![1]).get("storeId")).toBe(b.id);
 expect(screen.getByText("GMV 数据待接入")).toBeVisible();expect(vi.mocked(readSavedOrderAmount)).not.toHaveBeenCalled();
});
