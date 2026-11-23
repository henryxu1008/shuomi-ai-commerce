import { cleanup, render, screen, fireEvent, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { afterEach, expect, it, vi } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ecoRequest } from "@/lib/api/ecoservices";
import { ConsoleNavigation as Sidebar } from "./console-navigation";
import { findConsoleRoute } from "@/lib/workbench/console-navigation";

function ConsoleNavigation(props: Omit<ComponentProps<typeof Sidebar>, "permissions"> & { permissions?: readonly string[] }) {
  return <Sidebar permissions={[]} {...props} />;
}

const query=vi.hoisted(()=>({value:""}));
it("keeps the Orders overview on its parent with only the permitted platform records when fulfillment is unavailable",()=>{
 render(<ConsoleNavigation pathname="/workbench/store-orders" ariaLabel="主导航" storeObservationsAvailable permissions={["workbench.store.read", "workbench.store.orders.read"]}/>);
 expect(screen.getByRole("button",{name:"收起订单管理"})).toHaveAttribute("aria-expanded","true");
 expect(screen.queryByRole("link",{name:"订单概览"})).not.toBeInTheDocument();
 expect(screen.getByRole("link",{name:"订单管理"})).toHaveAttribute("aria-current","page");
 expect(screen.getByRole("link",{name:"平台订单记录"})).toHaveAttribute("title","平台订单记录");
 for(const name of ["订单处理","采购与收货","装箱与发货","交接与物流"])
  expect(screen.queryByRole("link",{name})).not.toBeInTheDocument();
});
it("selects each Order view and keeps the original source when switching query-only steps",()=>{
 query.value="step=stock&storeId=store-1&sourceId=source-1&kind=platform_supply&ignored=value";
 const view=render(<ConsoleNavigation pathname="/workbench/store-orders/fulfill" ariaLabel="主导航" fulfillmentAvailable permissions={["workbench.fulfillment.read"]}/>);
 expect(screen.getByRole("link",{name:"装箱与发货"})).toHaveAttribute("aria-current","page");
 expect(screen.queryByRole("link",{name:"订单概览"})).not.toBeInTheDocument();
 expect(screen.getByRole("link",{name:"订单管理"})).not.toHaveAttribute("aria-current");
 for(const [name,step] of [["订单处理","source"],["采购与收货","purchase"],["装箱与发货","stock"],["交接与物流","logistics"]]){
  const link=screen.getByRole("link",{name});
  expect(link).toHaveAttribute("title",name);
  const href=new URL(link.getAttribute("href")!,"https://test.invalid");
  expect(href.pathname).toBe("/workbench/store-orders/fulfill");
  expect(Object.fromEntries(href.searchParams)).toEqual({step,storeId:"store-1",sourceId:"source-1",kind:"platform_supply"});
 }
 expect(screen.queryByRole("link",{name:"平台订单记录"})).not.toBeInTheDocument();
 query.value="step=logistics&storeId=store-1&sourceId=source-1&kind=platform_supply";
 view.rerender(<ConsoleNavigation pathname="/workbench/store-orders/fulfill" ariaLabel="主导航" fulfillmentAvailable permissions={["workbench.fulfillment.read"]}/>);
 expect(screen.getByRole("link",{name:"交接与物流"})).toHaveAttribute("aria-current","page");
 expect(screen.getByRole("link",{name:"装箱与发货"})).not.toHaveAttribute("aria-current");
 query.value="step=platform";
 view.rerender(<ConsoleNavigation pathname="/workbench/store-orders/fulfill" ariaLabel="主导航" fulfillmentAvailable permissions={["workbench.fulfillment.read"]}/>);
 expect(screen.getByRole("link",{name:"装箱与发货"})).toHaveAttribute("aria-current","page");
});
it("does not carry duplicate or malformed Order identity parameters into another step",()=>{
 query.value="step=stock&step=logistics&storeId=one&storeId=two&sourceId=bad%2Fid&kind=platform_supply&kind=consumer_order";
 render(<ConsoleNavigation pathname="/workbench/store-orders/fulfill" ariaLabel="主导航" fulfillmentAvailable permissions={["workbench.fulfillment.read"]}/>);
 expect(screen.getByRole("link",{name:"订单处理"})).toHaveAttribute("aria-current","page");
 expect(screen.getByRole("link",{name:"采购与收货"})).toHaveAttribute("href","/workbench/store-orders/fulfill?step=purchase");
});
it("removes closing menu links from keyboard and accessibility access and supports immediate reopening", async () => {
  const user = userEvent.setup();
  render(<ConsoleNavigation pathname="/workbench/stores" ariaLabel="主导航" storeObservationsAvailable permissions={["workbench.store.read", "workbench.store.products.read"]} />);
  const close = screen.getByRole("button", { name: "收起店铺中心" });
  const group = document.getElementById(close.getAttribute("aria-controls")!)!;
  await user.click(close);
  expect(group).toHaveAttribute("inert");
  expect(group).toHaveAttribute("aria-hidden", "true");
  expect(screen.queryByRole("link", { name: "店铺商品" })).not.toBeInTheDocument();
  expect(close).toHaveFocus();
  await user.click(screen.getByRole("button", { name: "展开店铺中心" }));
  expect(screen.getAllByRole("link", { name: "店铺商品" })).toHaveLength(1);
  expect(screen.getByRole("link", { name: "店铺商品" }).closest("ul")).not.toHaveAttribute("inert");
});

it("shows the task center independently of Chat using the existing run read permissions",()=>{
 render(<ConsoleNavigation pathname="/workbench/ai/tasks" ariaLabel="主导航" nativeRunsAvailable permissions={["workbench.agent.read","workbench.agent.use"]}/>);
 expect(screen.getByRole("link",{name:"任务中心"})).toBeVisible();
 expect(screen.queryByRole("button",{name:"展开任务中心"})).not.toBeInTheDocument();
 expect(screen.queryByRole("link",{name:"硕米Chat"})).not.toBeInTheDocument();
});
it("shows a single primary listing entry before Store Center and retains the selected context",()=>{
 const id="11111111-1111-4111-8111-111111111111";query.value=`preparation=${id}&store=${id}&operation=${id}&mode=platform`;
 render(<ConsoleNavigation pathname="/workbench/listing/missing" ariaLabel="主导航" supplyChainAvailable supplyCompletionDraftsAvailable permissions={["workbench.supply.read", "workbench.store.read"]}/>);
 const listing=screen.getByRole("link",{name:"商品刊登"});
 const store=screen.getByRole("link",{name:"店铺中心"});
 expect(listing.closest("li")).toHaveClass("console-nav-level-1");
 expect(listing).toHaveAttribute("aria-current","page");
 expect(listing.compareDocumentPosition(store)&Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
 expect(Object.fromEntries(new URL(listing.getAttribute("href")!,"https://test.invalid").searchParams)).toEqual({preparation:id,store:id,operation:id});
 for(const name of ["待上传","待补全","待审核","已上传","我的供应链"])
  expect(screen.queryByRole("link",{name})).not.toBeInTheDocument();
 query.value="";
});
it.each(["official", "mine", "team"]) ("keeps skill category %s under one second-level entry", category => {
  render(<ConsoleNavigation pathname={`/workbench/agents/skills/${category}`} ariaLabel="主导航" userId="signed-in-user" />);
  expect(screen.getByRole("link", {name:"技能库"})).toHaveAttribute("aria-current", "page");
  expect(screen.getByRole("link", {name:"技能库"})).toHaveAttribute("href", "/workbench/agents/skills/official");
  expect(screen.queryByRole("button", {name:/展开技能库|收起技能库/})).not.toBeInTheDocument();
  for (const name of ["官方模板", "我的技能", "团队技能"]) expect(screen.queryByRole("link", {name})).not.toBeInTheDocument();
  expect(findConsoleRoute(`/workbench/agents/skills/${category}`)).toBeDefined();
});

it.each(["/workbench/tools/custom", "/workbench/tools/custom/new", "/workbench/tools/custom/progress"])("keeps tool customization page %s under one second-level entry", pathname => {
  const view = render(<ConsoleNavigation pathname={pathname} ariaLabel="主导航" toolMarketAvailable permissions={["workbench.tools.read"]} />);
  const entry = screen.getByRole("link", {name:"工具定制"});
  expect(entry).toHaveAttribute("aria-current", "page");
  expect(entry).toHaveAttribute("href", "/workbench/tools/custom");
  expect(entry).toHaveAttribute("title", "工具定制");
  expect(entry.closest("li")).toHaveClass("console-nav-level-2");
  expect(screen.queryByRole("button", {name:/工具定制/})).not.toBeInTheDocument();
  for (const name of ["提交定制需求", "定制进度"]) expect(screen.queryByRole("link", {name})).not.toBeInTheDocument();
  view.rerender(<ConsoleNavigation pathname={pathname} ariaLabel="主导航" />);
  expect(screen.queryByRole("link", {name:"工具定制"})).not.toBeInTheDocument();
});
it.each(["/workbench/supply/catalogs", "/workbench/supply/catalogs/selection", "/workbench/supply/catalogs/apply", "/workbench/supply/catalogs/sds"])("keeps catalog page %s under one second-level entry", pathname => {
  const view=render(<ConsoleNavigation pathname={pathname} ariaLabel="主导航" supplyMarketAvailable permissions={["workbench.supply-market.read"]}/>);
  const catalog=screen.getByRole("link",{name:"货盘集成"});
  expect(catalog).toHaveAttribute("aria-current","page");
  expect(catalog.closest("li")).toHaveClass("console-nav-level-2");
  expect(screen.queryByRole("button",{name:/货盘集成/})).not.toBeInTheDocument();
  expect(screen.queryByRole("link",{name:"货盘选品"})).not.toBeInTheDocument();
  expect(screen.queryByRole("link",{name:"申请对接"})).not.toBeInTheDocument();
  view.rerender(<ConsoleNavigation pathname={pathname} ariaLabel="主导航"/>);
  expect(screen.queryByRole("link",{name:"货盘集成"})).not.toBeInTheDocument();
});
it.each(["/workbench/supply/applications","/workbench/supply/applications/new","/workbench/supply/applications/progress","/workbench/supply/applications/records","/workbench/supply/applications/11111111-1111-4111-8111-111111111111"])("keeps selected application page %s under one second-level entry",pathname=>{
 const view=render(<ConsoleNavigation pathname={pathname} ariaLabel="主导航" supplyMarketAvailable permissions={["workbench.supply-market.read"]}/>);
 const entry=screen.getByRole("link",{name:"优选申请"});expect(entry).toHaveAttribute("aria-current","page");expect(entry.closest("li")).toHaveClass("console-nav-level-2");
 expect(screen.queryByRole("button",{name:/优选申请/})).not.toBeInTheDocument();
 for(const name of ["发起申请","申请进度","申请记录"])expect(screen.queryByRole("link",{name})).not.toBeInTheDocument();
 view.rerender(<ConsoleNavigation pathname={pathname} ariaLabel="主导航"/>);expect(screen.queryByRole("link",{name:"优选申请"})).not.toBeInTheDocument();
});
it("keeps the Supply landing when it links to available listing preparation", () => {
  render(<ConsoleNavigation pathname="/workbench/supply" ariaLabel="主导航" supplyPreparationAvailable permissions={["workbench.supply.read"]} />);
  expect(screen.getByRole("link", { name: "供应市场" })).toHaveAttribute("href", "/workbench/supply");
  expect(screen.queryByRole("button", { name: /展开供应市场|收起供应市场/ })).not.toBeInTheDocument();
});

it("shows only the available data pages with their own grants", () => {
  const pathname = "/workbench/data/api";
  const permissions = ["workbench.data-api.manage", "workbench.collection.read"];
  const view = render(<ConsoleNavigation pathname={pathname} ariaLabel="主导航" productCollectionsAvailable permissions={permissions} />);
  expect(screen.queryByRole("link", { name: "数据市场" })).not.toBeInTheDocument();
  expect(screen.queryByRole("link", { name: "API管理" })).not.toBeInTheDocument();
  expect(screen.getByRole("link", { name: "我的数据" })).toBeVisible();
  view.rerender(<ConsoleNavigation pathname={pathname} ariaLabel="主导航" dataServicesAvailable permissions={permissions} />);
  expect(screen.queryByRole("link", { name: "数据市场" })).not.toBeInTheDocument();
  expect(screen.getByRole("link", { name: "API管理" })).toBeVisible();
  expect(screen.queryByRole("link", { name: "我的数据" })).not.toBeInTheDocument();
});

it("opens saved reports without an Agent only when the report runtime is configured", () => {
 const view=render(<ConsoleNavigation pathname="/workbench/ai/reports" ariaLabel="主导航" />);
 expect(screen.queryByRole("link",{name:"我的报告"})).not.toBeInTheDocument();
 expect(screen.queryByRole("link",{name:"最近报告"})).not.toBeInTheDocument();
 view.rerender(<ConsoleNavigation pathname="/workbench/ai/reports/recent" ariaLabel="主导航" reportCenterAvailable permissions={["workbench.report.read"]} />);
 expect(screen.getByRole("link",{name:"我的报告"})).toHaveAttribute("title","我的报告");
 expect(screen.getByRole("link",{name:"我的报告"})).toHaveAttribute("aria-current","page");
 expect(screen.queryByRole("button",{name:"展开我的报告"})).not.toBeInTheDocument();
 expect(screen.queryByRole("link",{name:"最近报告"})).not.toBeInTheDocument();
 expect(screen.queryByRole("link",{name:"硕米Chat"})).not.toBeInTheDocument();
});

it.each(["recent", "favorites", "all"])("highlights the second-level report entry on %s", (path) => {
 render(<ConsoleNavigation pathname={`/workbench/ai/reports/${path}`} ariaLabel="主导航" reportCenterAvailable permissions={["workbench.report.read"]} />);
 const entry=screen.getByRole("link",{name:"我的报告"});
 expect(entry.closest("li")).toHaveClass("console-nav-level-2");
 expect(entry).toHaveAttribute("aria-current","page");
 expect(screen.queryByRole("button",{name:"展开我的报告"})).not.toBeInTheDocument();
});

it("opens current market entries only with the serving feature",()=>{
 const view=render(<ConsoleNavigation pathname="/workbench/supply/official" ariaLabel="主导航"/>);
 expect(screen.queryByRole("link",{name:"硕米自营"})).not.toBeInTheDocument();
 view.rerender(<ConsoleNavigation pathname="/workbench/supply/official" ariaLabel="主导航" supplyMarketAvailable permissions={["workbench.supply-market.read"]}/>);
 expect(screen.getByRole("link",{name:"硕米自营"})).toHaveAttribute("title","硕米自营");
 expect(screen.getByRole("link",{name:"优选申请"})).toHaveAttribute("title","优选申请");
});

it("only opens the granted cockpit page and keeps settings under its goal",()=>{
 const view=render(<ConsoleNavigation pathname="/workbench/overview/goals/settings" ariaLabel="主导航" operationsCockpitAvailable permissions={["workbench.cockpit.goals.read"]}/>);
 expect(screen.getByRole("link",{name:"目标管理"})).toHaveAttribute("title","目标管理");
 expect(screen.queryByRole("link",{name:"店铺矩阵"})).not.toBeInTheDocument();
 expect(findConsoleRoute("/workbench/overview/goals/settings")?.trail.map(n=>n.label)).toEqual(["运营驾驶舱","目标管理","设置经营目标"]);
 view.rerender(<ConsoleNavigation pathname="/workbench/overview/goals" ariaLabel="主导航" operationsCockpitAvailable={false} permissions={["workbench.cockpit.goals.read"]}/>);
 expect(screen.queryByRole("link",{name:"目标管理"})).not.toBeInTheDocument();
 expect(screen.queryByRole("link",{name:"运营驾驶舱"})).not.toBeInTheDocument();
});
it("advertises each Store observation entry only with ready runtime and its read permission", () => {
  const pathname = "/workbench/store-orders";
  const orders = ["workbench.store.read", "workbench.store.orders.read"];
  const view = render(<ConsoleNavigation pathname={pathname} ariaLabel="主导航" permissions={orders} />);
  expect(screen.queryByRole("link", { name: "订单管理" })).not.toBeInTheDocument();
  view.rerender(<ConsoleNavigation pathname={pathname} ariaLabel="主导航" storeObservationsAvailable permissions={orders} />);
  expect(screen.getByRole("link", { name: "平台订单记录" })).toBeVisible();
  fireEvent.click(screen.getByRole("button", { name: "展开店铺中心" }));
  expect(screen.queryByRole("link", { name: "店铺商品" })).not.toBeInTheDocument();
  view.rerender(<ConsoleNavigation pathname={pathname} ariaLabel="主导航" storeObservationsAvailable permissions={[...orders, "workbench.store.products.read"]} />);
  expect(screen.getByRole("link", { name: "店铺商品" })).toBeVisible();
});
vi.mock("next/navigation",()=>({useSearchParams:()=>new URLSearchParams(query.value)}));
afterEach(()=>{cleanup();query.value="";});

vi.mock("@/lib/api/ecoservices", async (importOriginal) => ({ ...await importOriginal<typeof import("@/lib/api/ecoservices")>(), ecoRequest: vi.fn() }));

it("discovers platform review through the original global backend gate without enterprise grants", async () => {
  vi.mocked(ecoRequest).mockResolvedValue({ applications: [], total: "0" });
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const view = render(<QueryClientProvider client={client}><ConsoleNavigation pathname="/workbench/services/review" ariaLabel="主导航" ecoservicesAvailable userId="platform-user" /></QueryClientProvider>);
  expect(await screen.findByRole("link", { name: "平台审核" })).toHaveAttribute("href", "/workbench/services/review");
  expect(screen.getByRole("link", { name: "平台审核" })).toHaveAttribute("aria-current", "page");
  expect(ecoRequest).toHaveBeenCalledWith({ userId: "platform-user", organizationId: "" }, "applications?page=1&pageSize=20", expect.anything(), expect.anything(), true);
  vi.mocked(ecoRequest).mockRejectedValue(new Error("ECOSERVICES_FORBIDDEN"));
  view.rerender(<QueryClientProvider client={client}><ConsoleNavigation pathname="/workbench/services/review" ariaLabel="主导航" ecoservicesAvailable userId="enterprise-admin-only" /></QueryClientProvider>);
  expect(screen.queryByRole("link", { name: "平台审核" })).not.toBeInTheDocument();
  view.rerender(<QueryClientProvider client={client}><ConsoleNavigation pathname="/workbench/services/review" ariaLabel="主导航" ecoservicesAvailable={false} userId="platform-user" /></QueryClientProvider>);
  expect(screen.queryByRole("link", { name: "平台审核" })).not.toBeInTheDocument();
});


it("only advertises Chat and BusinessTask when the current application mounts AI Workbench", () => {
  const view = render(<ConsoleNavigation pathname="/workbench/ai/chat" ariaLabel="主导航" aiWorkbenchAvailable={false} permissions={["workbench.chat.read", "workbench.task.read"]} />);
  expect(screen.queryByRole("link", { name: "硕米Chat" })).not.toBeInTheDocument();
  expect(screen.queryByRole("link", { name: "任务中心" })).not.toBeInTheDocument();
  view.rerender(<ConsoleNavigation pathname="/workbench/ai/chat" ariaLabel="主导航" aiWorkbenchAvailable permissions={["workbench.chat.read", "workbench.task.read"]} />);
  expect(screen.getByRole("link", { name: "硕米Chat" })).toBeVisible();
  expect(screen.getByRole("link", { name: "任务中心" })).toBeVisible();
});

it("keeps independently available Review and history pages discoverable without AI Workbench", async () => {
  render(<ConsoleNavigation pathname="/workbench" ariaLabel="主导航" aiWorkbenchAvailable={false} productReviewAvailable sheinRecordsAvailable permissions={["local_agent.write", "listingkit.admin.read"]} />);
  await userEvent.click(screen.getByRole("button", { name: "展开AI工作台" }));
  expect(screen.queryByRole("link", { name: "硕米Chat" })).not.toBeInTheDocument();
  expect(screen.getByRole("link", { name: "任务中心" })).toHaveAttribute("href", "/workbench/ai/tasks");
  expect(screen.queryByRole("button", { name: "展开任务中心" })).not.toBeInTheDocument();
  expect(screen.queryByRole("link", { name: "待确认" })).not.toBeInTheDocument();
  expect(screen.queryByRole("link", { name: "已完成" })).not.toBeInTheDocument();
  expect(screen.queryByRole("link", { name: "进行中" })).not.toBeInTheDocument();
  expect(screen.queryByRole("link", { name: "异常任务" })).not.toBeInTheDocument();
  expect(screen.queryByRole("link", { name: "标题审核" })).not.toBeInTheDocument();
  expect(screen.queryByRole("link", { name: "历史工作记录" })).not.toBeInTheDocument();
});

it("keeps the task center entry with either independent capability", async () => {
  for (const item of [
    { productReviewAvailable: true, sheinRecordsAvailable: false, visible: "待确认", hidden: "已完成" },
    { productReviewAvailable: false, sheinRecordsAvailable: true, visible: "已完成", hidden: "待确认" },
  ]) {
    const view = render(<ConsoleNavigation pathname="/workbench" ariaLabel="主导航" aiWorkbenchAvailable={false} productReviewAvailable={item.productReviewAvailable} sheinRecordsAvailable={item.sheinRecordsAvailable} permissions={[item.productReviewAvailable ? "local_agent.write" : "listingkit.admin.read"]} />);
    await userEvent.click(screen.getByRole("button", { name: "展开AI工作台" }));
    expect(screen.getByRole("link", { name: "任务中心" })).toHaveAttribute("href", "/workbench/ai/tasks");
    expect(screen.queryByRole("button", { name: "展开任务中心" })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: item.visible })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: item.hidden })).not.toBeInTheDocument();
    view.unmount();
  }
});

it.each(["new", "recent", "saved", "archived", "550e8400-e29b-41d4-a716-446655440000"])("highlights the second-level chat entry on %s", (path) => {
  render(<ConsoleNavigation pathname={`/workbench/ai/chat/${path}`} ariaLabel="主导航" aiWorkbenchAvailable permissions={["workbench.chat.read"]} />);
  const entry=screen.getByRole("link",{name:"硕米Chat"});
  expect(entry).toHaveAttribute("href","/workbench/ai/chat/new");
  expect(entry.closest("li")).toHaveClass("console-nav-level-2");
  expect(entry).toHaveAttribute("aria-current","page");
  expect(screen.queryByRole("button",{name:"展开硕米Chat"})).not.toBeInTheDocument();
});

it.each(["running", "pending", "completed", "errors", "pending/other", "completed/history"])("highlights the second-level task entry on %s", (path) => {
  render(<ConsoleNavigation pathname={`/workbench/ai/tasks/${path}`} ariaLabel="主导航" aiWorkbenchAvailable permissions={["workbench.task.read"]} />);
  const entry = screen.getByRole("link", { name: "任务中心" });
  expect(entry.closest("li")).toHaveClass("console-nav-level-2");
  expect(entry).toHaveAttribute("aria-current", "page");
  expect(screen.queryByRole("button", { name: "展开任务中心" })).not.toBeInTheDocument();
});

it.each(["active", "recent", "templates", "archived", "550e8400-e29b-41d4-a716-446655440000"])("highlights the second-level project entry on %s", (path) => {
  render(<ConsoleNavigation pathname={`/workbench/ai/projects/${path}`} ariaLabel="主导航" projectCenterAvailable permissions={["workbench.project.read"]} />);
  const entry = screen.getByRole("link", { name: "项目中心" });
  expect(entry.closest("li")).toHaveClass("console-nav-level-2");
  expect(entry).toHaveAttribute("aria-current", "page");
  expect(screen.queryByRole("button", { name: "展开项目中心" })).not.toBeInTheDocument();
});

it.each([
  [false, false], [true, false], [false, true], [true, true],
])("shows one second-level online acquisition entry by capability (1688: %s, Amazon: %s)", async (has1688, hasAmazon) => {
  render(<ConsoleNavigation pathname="/workbench" ariaLabel="主导航" productAcquisitionAvailable={has1688} dataServicesAvailable={hasAmazon} permissions={["product_sourcing.write", "workbench.data-market.use"]} />);
  if (has1688 || hasAmazon) await userEvent.click(screen.getByRole("button", { name: "展开供应市场" }));
  expect(screen.queryByRole("link", { name: "1688 在线采集" })).not.toBeInTheDocument();
  expect(screen.queryByRole("link", { name: "Amazon 在线采集" })).not.toBeInTheDocument();
  if (!has1688 && !hasAmazon) {
    expect(screen.queryByRole("link", { name: "在线采集" })).not.toBeInTheDocument();
    return;
  }
  const entry=screen.getByRole("link", { name: "在线采集" });
  expect(entry).toHaveAttribute("href", "/workbench/supply/acquisition");
  expect(entry.closest("li")).toHaveClass("console-nav-level-2");
  expect(screen.queryByRole("button", { name: "展开在线采集" })).not.toBeInTheDocument();
});

it("keeps the unified entry selected on acquisition results and direct Amazon links", () => {
  const pathname = "/workbench/supply/acquisition/operation/11111111-1111-4111-8111-111111111111";
  const v=render(<ConsoleNavigation pathname={pathname} ariaLabel="主导航" productAcquisitionAvailable permissions={["product_sourcing.write"]} />);
  expect(screen.getByRole("link", { name: "在线采集" })).toHaveAttribute("aria-current", "page");
  expect(findConsoleRoute(pathname)?.trail.map(node => node.label)).toEqual(["供应市场", "在线采集", "采集结果"]);
  v.rerender(<ConsoleNavigation pathname="/workbench/supply/acquisition/amazon" ariaLabel="主导航" dataServicesAvailable permissions={["workbench.data-market.use"]} />);
  expect(screen.getByRole("link", { name: "在线采集" })).toHaveAttribute("aria-current", "page");
});

it("keeps only three account sections and selects the parent when opening a detail", () => {
  const view = render(<ConsoleNavigation pathname="/workbench/stores" ariaLabel="主导航" />);
  for (const [path, parent] of [
    ["profile/settings", "账户资料"],
    ["organization/members", "企业空间"],
    ["organization/resources", "企业空间"],
    ["organization/resources/source-accounts", "企业空间"],
    ["organization/audit", "企业空间"],
    ["referrals/withdrawals", "推广与收益"],
  ]) {
    view.rerender(<ConsoleNavigation pathname={`/workbench/account/${path}`} ariaLabel="主导航" hasOrganization />);
    const account = screen.getByRole("link", { name: "我的账户" }).closest("li")!;
    const [overview, ...sections] = within(account).getAllByRole("link");
    expect(overview).toHaveAttribute("href", "/workbench/account");
    expect(sections.map(link => link.textContent)).toEqual(["账户资料", "企业空间", "推广与收益"]);
    expect(within(account).getAllByRole("button")).toHaveLength(1);
    expect(within(account).getByRole("link", { name: parent })).toHaveAttribute("aria-current", "page");
  }
});

it.each(["model-route-pricing", "kernel-releases", "kernel-releases/new", "data-row-pricing"])(
  "keeps Enterprise Space selected when opening %s directly or from the account section",
  (detail) => {
    const organization = "/workbench/account/organization";
    const detailPath = `${organization}/${detail}`;
    const sidebar = (pathname: string) => <ConsoleNavigation pathname={pathname} ariaLabel="主导航" hasOrganization />;
    const view = render(sidebar(detailPath));

    function expectEnterpriseSectionSelected() {
      const account = screen.getByRole("link", { name: "我的账户" }).closest("li")!;
      const menu = within(account);
      expect(menu.getByRole("button", { name: /我的账户/ })).toHaveAttribute("aria-expanded", "true");
      expect(menu.getAllByRole("link").map(link => link.getAttribute("href"))).toEqual([
        "/workbench/account",
        "/workbench/account/profile",
        organization,
        "/workbench/account/referrals",
      ]);
      expect(menu.getByRole("link", { name: "企业空间" })).toHaveAttribute("aria-current", "page");
    }

    expectEnterpriseSectionSelected();
    view.rerender(sidebar(organization));
    expectEnterpriseSectionSelected();
    view.rerender(sidebar(detailPath));
    expectEnterpriseSectionSelected();
  },
);

it("keeps the plans module as a connected five-page tree", () => {
  const pages = [
    ["/workbench/plans", "套餐与权益"],
    ["/workbench/plans/options", "套餐方案"],
    ["/workbench/plans/entitlements", "我的权益"],
    ["/workbench/plans/usage", "用量明细"],
    ["/workbench/plans/top-up", "充值中心"],
    ["/workbench/plans/orders", "账单与订单"],
  ] as const;

  for (const [pathname, label] of pages) {
    const route = findConsoleRoute(pathname);
    expect(route?.node.label).toBe(label);
    expect(route?.node.availability).toBe("connected");
    expect(route?.trail.map((item) => item.label)).toEqual(
      pathname === "/workbench/plans"
        ? ["套餐与权益"]
        : ["套餐与权益", label],
    );
  }
});

it("expands the plans module and selects a child route", () => {
  const view = render(<ConsoleNavigation pathname="/workbench/plans/usage" ariaLabel="主导航" permissions={["workbench.commercial.read"]} />);
  expect(screen.getByRole("button", { name: "收起套餐与权益" })).toBeVisible();
  expect(screen.getByRole("link", { name: "用量明细" })).toHaveAttribute("aria-current", "page");
  expect(screen.getByRole("link", { name: "充值中心" })).toBeVisible();
  view.rerender(<ConsoleNavigation pathname="/workbench/plans/orders" ariaLabel="主导航" permissions={["workbench.commercial.read"]} />);
  expect(screen.getByRole("link", { name: "账单与订单" })).toHaveAttribute("aria-current", "page");
});

it("retains three-level route trails outside the plans module", () => {
  const route = findConsoleRoute("/workbench/account/profile/settings");
  expect(route?.trail.map((item) => item.label)).toEqual([
    "我的账户",
    "账户资料",
    "账户设置",
  ]);
});

it("reflects installed capability on the primary listing entry",()=>{
 const view=render(<ConsoleNavigation pathname="/workbench/listing/waiting" ariaLabel="主导航"/>);
 expect(screen.queryByRole("link",{name:"商品刊登"})).not.toBeInTheDocument();
 view.rerender(<ConsoleNavigation pathname="/workbench/listing/waiting" ariaLabel="主导航" supplyCompletionDraftsAvailable permissions={["workbench.supply.read"]}/>);
 expect(screen.getByRole("link",{name:"商品刊登"})).toHaveAttribute("title","商品刊登");
});
it("preserves valid context and skips invalid or repeated listing parameters",()=>{
 const id="11111111-1111-4111-8111-111111111111";
 query.value="preparation="+id+"&store=bad&operation="+id+"&operation="+id;
 render(<ConsoleNavigation pathname="/workbench/listing/waiting" ariaLabel="主导航" supplyChainAvailable permissions={["workbench.supply.read"]}/>);
 expect(screen.getByRole("link",{name:"商品刊登"})).toHaveAttribute("href","/workbench/listing?preparation="+id);
 query.value="";
});
it.each(["official", "mine"]) ("keeps knowledge category %s under one second-level entry", category => {
  const view = render(<ConsoleNavigation pathname={`/workbench/ai/knowledge/${category}`} ariaLabel="主导航" permissions={["workbench.knowledge.read"]} />);
  expect(screen.getByRole("link", {name:"知识库"})).toHaveAttribute("aria-current", "page");
  expect(screen.queryByRole("button", {name:/展开知识库|收起知识库/})).not.toBeInTheDocument();
  expect(screen.queryByRole("link", {name:"官方知识库"})).not.toBeInTheDocument();
  expect(screen.queryByRole("link", {name:"我的知识库"})).not.toBeInTheDocument();
  view.rerender(<ConsoleNavigation pathname={`/workbench/ai/knowledge/${category}`} ariaLabel="主导航" />);
  expect(screen.queryByRole("link", {name:"知识库"})).not.toBeInTheDocument();
});

it.each(["/workbench/agents/custom", "/workbench/agents/custom/new", "/workbench/agents/custom/progress"])("keeps customization page %s under one second-level entry", pathname => {
  render(<ConsoleNavigation pathname={pathname} ariaLabel="主导航" permissions={["workbench.agent.read"]} />);
  expect(screen.getByRole("button", {name:"收起智能市场"})).toHaveAttribute("aria-expanded", "true");
  const entry = screen.getByRole("link", {name:"智能体定制"});
  expect(entry).toHaveAttribute("aria-current", "page");
  expect(entry).toHaveAttribute("href", "/workbench/agents/custom");
  expect(entry).toHaveAttribute("title", "智能体定制");
  expect(entry.closest("li")).toHaveClass("console-nav-level-2");
  expect(screen.queryByRole("button", {name:/智能体定制/})).not.toBeInTheDocument();
  for (const name of ["提交定制需求", "定制进度"]) expect(screen.queryByRole("link", {name})).not.toBeInTheDocument();
});

it.each(["/workbench/data/research", "/workbench/supply/selection"]) ("keeps %s under the market instead of a primary sidebar entry", (pathname) => {
 const id="11111111-1111-4111-8111-111111111111";
 render(<ConsoleNavigation pathname={`${pathname}/${id}`} ariaLabel="主导航" permissions={["workbench.agent.read"]}/>);
 expect(screen.queryByRole("link",{name:"市场调研"})).not.toBeInTheDocument();
 expect(screen.queryByRole("link",{name:"智能选品"})).not.toBeInTheDocument();
 expect(screen.getByRole("link",{name:"智能体市场"})).toHaveAttribute("aria-current","page");
 expect(findConsoleRoute(`${pathname}/${id}`)).toBeDefined();
});

it("keeps the market as the entry to enabled research for a project reader", () => {
  render(<ConsoleNavigation pathname="/workbench/data/research" ariaLabel="主导航"
    marketResearchAvailable projectCenterAvailable permissions={["workbench.project.read"]} />);
  expect(screen.getByRole("link", { name: "智能体市场" })).toHaveAttribute("aria-current", "page");
  expect(screen.queryByRole("link", { name: "我的智能体" })).not.toBeInTheDocument();
  expect(screen.queryByRole("link", { name: "智能体定制" })).not.toBeInTheDocument();
});
