import { cleanup, render, screen, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, it, vi } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ecoRequest } from "@/lib/api/ecoservices";
import { ConsoleNavigation } from "./console-navigation";
import { findConsoleRoute } from "@/lib/workbench/console-navigation";

const query=vi.hoisted(()=>({value:""}));
it("keeps the Orders overview on its parent with five business entries without advertising fulfillment from observation availability",()=>{
 render(<ConsoleNavigation pathname="/workbench/store-orders" ariaLabel="主导航" storeOrdersAvailable/>);
 expect(screen.getByRole("button",{name:"收起订单管理"})).toHaveAttribute("aria-expanded","true");
 expect(screen.queryByRole("link",{name:"订单概览"})).not.toBeInTheDocument();
 expect(screen.getByRole("link",{name:"订单管理"})).toHaveAttribute("aria-current","page");
 expect(screen.getByRole("link",{name:"平台订单记录"})).toHaveAttribute("title","平台订单记录");
 for(const name of ["订单处理","采购与收货","装箱与发货","交接与物流"])
  expect(screen.getByRole("link",{name})).toHaveAttribute("title",`${name}：业务暂未启用`);
});
it("selects each Order view and keeps the original source when switching query-only steps",()=>{
 query.value="step=stock&storeId=store-1&sourceId=source-1&kind=platform_supply&ignored=value";
 const view=render(<ConsoleNavigation pathname="/workbench/store-orders/fulfill" ariaLabel="主导航" fulfillmentAvailable/>);
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
 expect(screen.getByRole("link",{name:"平台订单记录"})).toHaveAttribute("title","平台订单记录：业务暂未启用");
 query.value="step=logistics&storeId=store-1&sourceId=source-1&kind=platform_supply";
 view.rerender(<ConsoleNavigation pathname="/workbench/store-orders/fulfill" ariaLabel="主导航" fulfillmentAvailable/>);
 expect(screen.getByRole("link",{name:"交接与物流"})).toHaveAttribute("aria-current","page");
 expect(screen.getByRole("link",{name:"装箱与发货"})).not.toHaveAttribute("aria-current");
 query.value="step=platform";
 view.rerender(<ConsoleNavigation pathname="/workbench/store-orders/fulfill" ariaLabel="主导航" fulfillmentAvailable/>);
 expect(screen.getByRole("link",{name:"装箱与发货"})).toHaveAttribute("aria-current","page");
});
it("does not carry duplicate or malformed Order identity parameters into another step",()=>{
 query.value="step=stock&step=logistics&storeId=one&storeId=two&sourceId=bad%2Fid&kind=platform_supply&kind=consumer_order";
 render(<ConsoleNavigation pathname="/workbench/store-orders/fulfill" ariaLabel="主导航" fulfillmentAvailable/>);
 expect(screen.getByRole("link",{name:"订单处理"})).toHaveAttribute("aria-current","page");
 expect(screen.getByRole("link",{name:"采购与收货"})).toHaveAttribute("href","/workbench/store-orders/fulfill?step=purchase");
});
it("removes closing menu links from keyboard and accessibility access and supports immediate reopening", async () => {
  const user = userEvent.setup();
  render(<ConsoleNavigation pathname="/workbench/stores" ariaLabel="主导航" />);
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
 render(<ConsoleNavigation pathname="/workbench/ai/tasks" ariaLabel="主导航" nativeRunsAvailable cockpitPermissions={["workbench.agent.read","workbench.agent.use"]}/>);
 expect(screen.getByRole("link",{name:"任务中心"})).toBeVisible();
 expect(screen.queryByRole("button",{name:"展开任务中心"})).not.toBeInTheDocument();
 expect(screen.queryByRole("link",{name:"硕米Chat"})).not.toBeInTheDocument();
});
it("shows a single primary listing entry before Store Center and retains the selected context",()=>{
 const id="11111111-1111-4111-8111-111111111111";query.value=`preparation=${id}&store=${id}&operation=${id}&mode=platform`;
 render(<ConsoleNavigation pathname="/workbench/listing/missing" ariaLabel="主导航" supplyChainAvailable supplyCompletionDraftsAvailable/>);
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
it("opens the skill library categories under AI workbench", () => {
  render(<ConsoleNavigation pathname="/workbench/agents/skills/mine" ariaLabel="主导航" />);
  expect(screen.getByRole("link", {name:"技能库"})).toHaveAttribute("href", "/workbench/agents/skills");
  expect(screen.getByRole("link", {name:"官方模板"})).toHaveAttribute("href", "/workbench/agents/skills/official");
  expect(screen.getByRole("link", {name:"我的技能"})).toHaveAttribute("aria-current", "page");
  expect(screen.getByRole("link", {name:"团队技能"})).toHaveAttribute("href", "/workbench/agents/skills/team");
  expect(findConsoleRoute("/workbench/agents/skills/mine")?.trail.map(node => node.label)).toEqual(["AI工作台", "技能库", "我的技能"]);
});
it("uses the installed tool capability for both customization children", () => {
  const view=render(<ConsoleNavigation pathname="/workbench/tools/custom/new" ariaLabel="主导航" toolMarketAvailable />);
  expect(screen.getByRole("link", {name:"提交定制需求"})).toHaveAttribute("aria-current","page");
  expect(screen.getByRole("link", {name:"定制进度"})).toHaveAttribute("title","定制进度");
  view.rerender(<ConsoleNavigation pathname="/workbench/tools/custom/new" ariaLabel="主导航" />);
  expect(screen.getByRole("link", {name:"提交定制需求"})).toHaveAttribute("title","提交定制需求：业务暂未启用");
});
it.each(["运营驾驶舱", "AI工作台", "智能市场", "工具市场", "数据服务", "店铺中心"])("keeps the %s primary homepage available independently of child services", label => {
  render(<ConsoleNavigation pathname="/workbench" ariaLabel="主导航" />);
  expect(screen.getByRole("link", { name: label })).toHaveAttribute("title", label);
});
it.each([
  ["/workbench/supply/catalogs/selection", "货盘选品"],
  ["/workbench/supply/catalogs/apply", "申请对接"],
])("opens the catalog hierarchy at %s only with its installed runtime", (pathname, label) => {
  const view=render(<ConsoleNavigation pathname={pathname} ariaLabel="主导航" supplyMarketAvailable/>);
  expect(screen.getByRole("button", {name:"收起货盘集成"})).toHaveAttribute("aria-expanded", "true");
  expect(screen.getByRole("link", {name:label})).toHaveAttribute("aria-current", "page");
  expect(screen.getByRole("link", {name:label}).closest("li")).toHaveClass("console-nav-level-3");
  for (const child of ["货盘选品", "申请对接"]) {
    expect(screen.getByRole("link", {name:child})).toHaveAttribute("title", child);
    if(child!==label)expect(screen.getByRole("link", {name:child})).not.toHaveAttribute("aria-current");
  }
  view.rerender(<ConsoleNavigation pathname={pathname} ariaLabel="主导航"/>);
  expect(screen.getByRole("link", {name:label})).toHaveAttribute("title", `${label}：业务暂未启用`);
});
it.each(["productAcquisitionAvailable", "supplyMarketAvailable", "supplyChainAvailable"])("keeps the Supply homepage with %s", capability => {
  const view = render(<ConsoleNavigation pathname="/workbench/supply" ariaLabel="主导航" />);
  expect(screen.getByRole("link", {name: "供应市场"})).toHaveAttribute("title", "供应市场");
  view.rerender(<ConsoleNavigation pathname="/workbench/supply" ariaLabel="主导航" {...{[capability]: true}} />);
  expect(screen.getByRole("link", {name: "供应市场"})).toHaveAttribute("title", "供应市场");
});
it.each(["智能市场", "店铺中心"])("connects the %s parent to its existing child", label => {
  render(<ConsoleNavigation pathname="/workbench" ariaLabel="主导航" />);
  expect(screen.getByRole("link", {name: label})).toHaveAttribute("title", label);
});
it.each(["aiWorkbenchAvailable", "projectCenterAvailable", "knowledgeAvailable", "reportCenterAvailable", "productReviewAvailable", "sheinRecordsAvailable"])("marks the AI parent connected when %s is connected", (capability) => {
  const view = render(<ConsoleNavigation pathname="/workbench" ariaLabel="主导航" />);
  expect(screen.getByRole("link", {name: "AI工作台"})).toHaveAttribute("title", "AI工作台");
  view.rerender(<ConsoleNavigation pathname="/workbench" ariaLabel="主导航" {...{[capability]: true}} />);
  expect(screen.getByRole("link", {name: "AI工作台"})).toHaveAttribute("title", "AI工作台");
});
it("opens data market and API only with their runtime and keeps My Data separate", () => {
 const view=render(<ConsoleNavigation pathname="/workbench/data/api" ariaLabel="主导航" productCollectionsAvailable />);
 expect(screen.getByRole("link",{name:"数据市场"})).toHaveAttribute("title","数据市场：业务暂未启用");
 expect(screen.getByRole("link",{name:"API管理"})).toHaveAttribute("title","API管理：业务暂未启用");
 expect(screen.getByRole("link",{name:"我的数据"})).toHaveAttribute("title","我的数据");
 view.rerender(<ConsoleNavigation pathname="/workbench/data/api" ariaLabel="主导航" dataServicesAvailable />);
 expect(screen.getByRole("link",{name:"数据市场"})).toHaveAttribute("title","数据市场");
 expect(screen.getByRole("link",{name:"API管理"})).toHaveAttribute("title","API管理");
 expect(screen.getByRole("link",{name:"我的数据"})).toHaveAttribute("title","我的数据：业务暂未启用");
});
it("opens saved reports without an Agent only when the report runtime is configured", () => {
 const view=render(<ConsoleNavigation pathname="/workbench/ai/reports" ariaLabel="主导航" />);
 expect(screen.getByRole("link",{name:"我的报告"})).toHaveAttribute("title","我的报告：业务暂未启用");
 expect(screen.queryByRole("link",{name:"最近报告"})).not.toBeInTheDocument();
 view.rerender(<ConsoleNavigation pathname="/workbench/ai/reports/recent" ariaLabel="主导航" reportCenterAvailable />);
 expect(screen.getByRole("link",{name:"我的报告"})).toHaveAttribute("title","我的报告");
 expect(screen.getByRole("link",{name:"最近报告"})).toHaveAttribute("aria-current","page");
 expect(screen.getByRole("link",{name:"收藏报告"})).toHaveAttribute("href","/workbench/ai/reports/favorites");
 expect(screen.getByRole("link",{name:"全部报告"})).toHaveAttribute("href","/workbench/ai/reports/all");
 expect(screen.queryByRole("link",{name:"硕米Chat"})).not.toBeInTheDocument();
});

it("opens current market entries only with the serving feature",()=>{
 const view=render(<ConsoleNavigation pathname="/workbench/supply/official" ariaLabel="主导航"/>);
 expect(screen.getByRole("link",{name:"硕米自营"})).toHaveAttribute("title","硕米自营：业务暂未启用");
 view.rerender(<ConsoleNavigation pathname="/workbench/supply/official" ariaLabel="主导航" supplyMarketAvailable/>);
 expect(screen.getByRole("link",{name:"硕米自营"})).toHaveAttribute("title","硕米自营");
 expect(screen.getByRole("link",{name:"优选申请"})).toHaveAttribute("title","优选申请");
});

it("only opens the granted cockpit page and keeps settings under its goal",()=>{
 const view=render(<ConsoleNavigation pathname="/workbench/overview/goals/settings" ariaLabel="主导航" operationsCockpitAvailable cockpitPermissions={["workbench.cockpit.goals.read"]}/>);
 expect(screen.getByRole("link",{name:"目标管理"})).toHaveAttribute("title","目标管理");
 expect(screen.getByRole("link",{name:"店铺矩阵"})).toHaveAttribute("title","店铺矩阵：业务暂未启用");
 expect(findConsoleRoute("/workbench/overview/goals/settings")?.trail.map(n=>n.label)).toEqual(["运营驾驶舱","目标管理","设置经营目标"]);
 view.rerender(<ConsoleNavigation pathname="/workbench/overview/goals" ariaLabel="主导航" operationsCockpitAvailable={false} cockpitPermissions={["workbench.cockpit.goals.read"]}/>);
 expect(screen.getByRole("link",{name:"目标管理"})).toHaveAttribute("title","目标管理：业务暂未启用");
});
it("advertises each Store observation entry only with ready runtime and its read permission", () => {
  const view=render(<ConsoleNavigation pathname="/workbench/store-orders" ariaLabel="主导航" storeProductsAvailable={false} storeOrdersAvailable={false}/>);
  expect(screen.getByRole("link",{name:"订单管理"})).toHaveAttribute("title","订单管理：业务暂未启用");
  view.rerender(<ConsoleNavigation pathname="/workbench/store-orders" ariaLabel="主导航" storeProductsAvailable={false} storeOrdersAvailable/>);
  expect(screen.getByRole("link",{name:"订单管理"})).toHaveAttribute("title","订单管理");
  fireEvent.click(screen.getByRole("button",{name:"展开店铺中心"}));
  expect(screen.getByRole("link",{name:"店铺商品"})).toHaveAttribute("title","店铺商品：业务暂未启用");
  view.rerender(<ConsoleNavigation pathname="/workbench/store-orders" ariaLabel="主导航" storeProductsAvailable storeOrdersAvailable/>);
  expect(screen.getByRole("link",{name:"店铺商品"})).toHaveAttribute("title","店铺商品");
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
  const view = render(<ConsoleNavigation pathname="/workbench/ai/chat" ariaLabel="主导航" aiWorkbenchAvailable={false} />);
  expect(screen.queryByRole("link", { name: "硕米Chat" })).not.toBeInTheDocument();
  expect(screen.queryByRole("link", { name: "任务中心" })).not.toBeInTheDocument();
  view.rerender(<ConsoleNavigation pathname="/workbench/ai/chat" ariaLabel="主导航" aiWorkbenchAvailable />);
  expect(screen.getByRole("link", { name: "硕米Chat" })).toBeVisible();
  expect(screen.getByRole("link", { name: "任务中心" })).toBeVisible();
});

it("keeps independently available Review and history pages discoverable without AI Workbench", async () => {
  render(<ConsoleNavigation pathname="/workbench" ariaLabel="主导航" aiWorkbenchAvailable={false} productReviewAvailable sheinRecordsAvailable />);
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
    const view = render(<ConsoleNavigation pathname="/workbench" ariaLabel="主导航" aiWorkbenchAvailable={false} productReviewAvailable={item.productReviewAvailable} sheinRecordsAvailable={item.sheinRecordsAvailable} />);
    await userEvent.click(screen.getByRole("button", { name: "展开AI工作台" }));
    expect(screen.getByRole("link", { name: "任务中心" })).toHaveAttribute("href", "/workbench/ai/tasks");
    expect(screen.queryByRole("button", { name: "展开任务中心" })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: item.visible })).not.toBeInTheDocument();
    expect(screen.queryByRole("link", { name: item.hidden })).not.toBeInTheDocument();
    view.unmount();
  }
});

it.each(["running", "pending", "completed", "errors", "pending/other", "completed/history"])("highlights the second-level task entry on %s", (path) => {
  render(<ConsoleNavigation pathname={`/workbench/ai/tasks/${path}`} ariaLabel="主导航" aiWorkbenchAvailable />);
  const entry = screen.getByRole("link", { name: "任务中心" });
  expect(entry.closest("li")).toHaveClass("console-nav-level-2");
  expect(entry).toHaveAttribute("aria-current", "page");
  expect(screen.queryByRole("button", { name: "展开任务中心" })).not.toBeInTheDocument();
});

it.each(["active", "recent", "templates", "archived", "550e8400-e29b-41d4-a716-446655440000"])("highlights the second-level project entry on %s", (path) => {
  render(<ConsoleNavigation pathname={`/workbench/ai/projects/${path}`} ariaLabel="主导航" projectCenterAvailable />);
  const entry = screen.getByRole("link", { name: "项目中心" });
  expect(entry.closest("li")).toHaveClass("console-nav-level-2");
  expect(entry).toHaveAttribute("aria-current", "page");
  expect(screen.queryByRole("button", { name: "展开项目中心" })).not.toBeInTheDocument();
});

it.each([
  [false, false], [true, false], [false, true], [true, true],
])("shows one second-level online acquisition entry by capability (1688: %s, Amazon: %s)", async (has1688, hasAmazon) => {
  render(<ConsoleNavigation pathname="/workbench" ariaLabel="主导航" productAcquisitionAvailable={has1688} dataServicesAvailable={hasAmazon} />);
  await userEvent.click(screen.getByRole("button", { name: "展开供应市场" }));
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
  const v=render(<ConsoleNavigation pathname={pathname} ariaLabel="主导航" productAcquisitionAvailable />);
  expect(screen.getByRole("link", { name: "在线采集" })).toHaveAttribute("aria-current", "page");
  expect(findConsoleRoute(pathname)?.trail.map(node => node.label)).toEqual(["供应市场", "在线采集", "采集结果"]);
  v.rerender(<ConsoleNavigation pathname="/workbench/supply/acquisition/amazon" ariaLabel="主导航" dataServicesAvailable />);
  expect(screen.getByRole("link", { name: "在线采集" })).toHaveAttribute("aria-current", "page");
});

it("reveals the selected account page after navigation from another module", async () => {
  const view = render(<ConsoleNavigation pathname="/workbench/stores" ariaLabel="主导航" />);
  view.rerender(<ConsoleNavigation pathname="/workbench/account/organization/members" ariaLabel="主导航" />);
  expect(screen.getByRole("link", { name: "成员与权限" })).toHaveAttribute("aria-current", "page");
  await userEvent.click(screen.getByRole("button", { name: "收起企业空间" }));
  expect(screen.queryByRole("link", { name: "成员与权限" })).not.toBeInTheDocument();
  view.rerender(<ConsoleNavigation pathname="/workbench/account/organization/audit" ariaLabel="主导航" />);
  expect(screen.getByRole("link", { name: "操作记录" })).toHaveAttribute("aria-current", "page");
  view.rerender(<ConsoleNavigation pathname="/workbench/account/organization/members" ariaLabel="主导航" />);
  expect(screen.getByRole("link", { name: "成员与权限" })).toHaveAttribute("aria-current", "page");
});

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
  const view = render(<ConsoleNavigation pathname="/workbench/plans/usage" ariaLabel="主导航" />);
  expect(screen.getByRole("button", { name: "收起套餐与权益" })).toBeVisible();
  expect(screen.getByRole("link", { name: "用量明细" })).toHaveAttribute("aria-current", "page");
  expect(screen.getByRole("link", { name: "充值中心" })).toBeVisible();
  view.rerender(<ConsoleNavigation pathname="/workbench/plans/orders" ariaLabel="主导航" />);
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
 expect(screen.getByRole("link",{name:"商品刊登"})).toHaveAttribute("title","商品刊登：业务暂未启用");
 view.rerender(<ConsoleNavigation pathname="/workbench/listing/waiting" ariaLabel="主导航" supplyCompletionDraftsAvailable/>);
 expect(screen.getByRole("link",{name:"商品刊登"})).toHaveAttribute("title","商品刊登");
});
it("preserves valid context and skips invalid or repeated listing parameters",()=>{
 const id="11111111-1111-4111-8111-111111111111";
 query.value="preparation="+id+"&store=bad&operation="+id+"&operation="+id;
 render(<ConsoleNavigation pathname="/workbench/listing/waiting" ariaLabel="主导航" supplyChainAvailable/>);
 expect(screen.getByRole("link",{name:"商品刊登"})).toHaveAttribute("href","/workbench/listing?preparation="+id);
 query.value="";
});
it("opens official reading independently while retaining the enterprise knowledge gate", () => {
  const view = render(<ConsoleNavigation pathname="/workbench/ai/knowledge/mine" ariaLabel="主导航" knowledgeAvailable />);
  expect(screen.getByRole("link", {name:"知识库"})).toHaveAttribute("title", "知识库");
  expect(screen.getByRole("link", {name:"我的知识库"})).toHaveAttribute("aria-current", "page");
  expect(screen.getByRole("link", {name:"我的知识库"})).toHaveAttribute("title", "我的知识库");
  expect(screen.getByRole("link", {name:"官方知识库"})).toHaveAttribute("title", "官方知识库");
  view.rerender(<ConsoleNavigation pathname="/workbench/ai/knowledge/mine" ariaLabel="主导航" />);
  expect(screen.getByRole("link", {name:"知识库"})).toHaveAttribute("title", "知识库");
  expect(screen.getByRole("link", {name:"我的知识库"})).toHaveAttribute("title", "我的知识库：业务暂未启用");
});

it.each([
  ["/workbench/agents/custom/new", "提交定制需求"],
  ["/workbench/agents/custom/progress", "定制进度"],
])("opens the customization hierarchy at %s and marks only its leaf current", (pathname, label) => {
  render(<ConsoleNavigation pathname={pathname} ariaLabel="主导航" />);
  expect(screen.getByRole("button", {name:"收起智能市场"})).toHaveAttribute("aria-expanded", "true");
  expect(screen.getByRole("button", {name:"收起智能体定制"})).toHaveAttribute("aria-expanded", "true");
  const current = screen.getByRole("link", {name:label});
  expect(current).toHaveAttribute("href", pathname);
  expect(current).toHaveAttribute("aria-current", "page");
  expect(current.closest("li")).toHaveClass("console-nav-level-3");
  expect(screen.getByRole("link", {name:"智能体定制"})).not.toHaveAttribute("aria-current");
  for (const child of ["提交定制需求", "定制进度"]) {
    expect(screen.getByRole("link", {name:child})).toHaveAttribute("title", child);
    if (child !== label) expect(screen.getByRole("link", {name:child})).not.toHaveAttribute("aria-current");
  }
});

it("lets members collapse and reopen the customization children from its overview", async () => {
  const user = userEvent.setup();
  const view = render(<ConsoleNavigation pathname="/workbench/agents/custom" ariaLabel="主导航" />);
  expect(screen.getByRole("link", {name:"智能体定制"})).toHaveAttribute("aria-current", "page");
  expect(screen.getByRole("link", {name:"提交定制需求"})).toHaveAttribute("href", "/workbench/agents/custom/new");
  expect(screen.getByRole("link", {name:"定制进度"})).toHaveAttribute("href", "/workbench/agents/custom/progress");
  await user.click(screen.getByRole("button", {name:"收起智能体定制"}));
  expect(screen.queryByRole("link", {name:"提交定制需求"})).not.toBeInTheDocument();
  await user.click(screen.getByRole("button", {name:"展开智能体定制"}));
  expect(screen.getByRole("link", {name:"提交定制需求"})).toBeVisible();
  view.rerender(<ConsoleNavigation pathname="/workbench/agents/custom/progress" ariaLabel="主导航" />);
  expect(screen.getByRole("link", {name:"定制进度"})).toHaveAttribute("aria-current", "page");
});

it.each([
  ["new", "发起申请"], ["progress", "申请进度"], ["records", "申请记录"],
])("connects the selected-application %s leaf only with its runtime", (path, label) => {
  const pathname = `/workbench/supply/applications/${path}`;
  const view = render(<ConsoleNavigation pathname={pathname} ariaLabel="主导航" supplyMarketAvailable />);
  expect(screen.getByRole("button", {name:"收起优选申请"})).toHaveAttribute("aria-expanded", "true");
  expect(screen.getByRole("link", {name:label})).toHaveAttribute("href", pathname);
  expect(screen.getByRole("link", {name:label})).toHaveAttribute("aria-current", "page");
  expect(screen.getByRole("link", {name:label}).closest("li")).toHaveClass("console-nav-level-3");
  expect(screen.getByRole("link", {name:"优选申请"})).not.toHaveAttribute("aria-current");
  for (const child of ["发起申请", "申请进度", "申请记录"]) {
    expect(screen.getByRole("link", {name:child})).toHaveAttribute("title", child);
    if(child !== label) expect(screen.getByRole("link", {name:child})).not.toHaveAttribute("aria-current");
  }
  view.rerender(<ConsoleNavigation pathname={pathname} ariaLabel="主导航" />);
  expect(screen.getByRole("link", {name:label})).toHaveAttribute("title", `${label}：业务暂未启用`);
});

it("promotes research and selection to independent primary menus and keeps runtime availability",()=>{
 const v=render(<ConsoleNavigation pathname="/workbench/data/research" ariaLabel="主导航" marketResearchAvailable/>);
 expect(screen.getByRole("link",{name:"市场调研"})).toHaveAttribute("title","市场调研");
 expect(screen.getByRole("link",{name:"市场调研"}).closest("li")).toHaveClass("console-nav-level-1");
 expect(screen.queryByRole("button",{name:"展开资料收集"})).not.toBeInTheDocument();
 expect(screen.getByRole("link",{name:"收集与整理资料"}).closest("li")).toHaveClass("console-nav-level-2");
 expect(screen.queryByRole("link",{name:"资料收集"})).not.toBeInTheDocument();
 v.rerender(<ConsoleNavigation pathname="/workbench/supply/selection" ariaLabel="主导航" marketResearchAvailable/>);
 expect(screen.getByRole("link",{name:"智能选品"})).toHaveAttribute("title","智能选品");
 expect(screen.getByRole("link",{name:"智能选品"}).closest("li")).toHaveClass("console-nav-level-1");
 expect(findConsoleRoute("/workbench/supply/selection/11111111-1111-4111-8111-111111111111")?.trail.map(n=>n.label)).toEqual(["智能选品","项目评审"]);
 v.rerender(<ConsoleNavigation pathname="/workbench/supply/selection" ariaLabel="主导航"/>);
 expect(screen.getByRole("link",{name:"智能选品"})).toHaveAttribute("title","智能选品：业务暂未启用");
});

it("keeps direct second-level workflow links in the current project",()=>{
 const id="11111111-1111-4111-8111-111111111111";
 query.value="step=evidence&mode=archived&search=收纳";
 render(<ConsoleNavigation pathname={`/workbench/data/research/${id}`} ariaLabel="主导航" marketResearchAvailable/>);
 expect(screen.queryByRole("button",{name:"收起竞品研究"})).not.toBeInTheDocument();
 const selected=screen.getByRole("link",{name:"查看商品资料"});
 expect(selected).toHaveAttribute("aria-current","page");
 expect(selected.closest("li")).toHaveClass("console-nav-level-2");
 const target=new URL(screen.getByRole("link",{name:"比较重点商品"}).getAttribute("href")!,"https://test.invalid");
 expect(target.pathname).toBe(`/workbench/data/research/${id}`);
 expect(Object.fromEntries(target.searchParams)).toEqual({step:"summary",mode:"archived",search:"收纳"});
 expect(screen.queryByRole("link",{name:"市场调研中心"})).not.toBeInTheDocument();
});

it("keeps selection links at level two with the original sample and narrow-navigation callback",()=>{
 const id="11111111-1111-4111-8111-111111111111",other="22222222-2222-4222-8222-222222222222";
 const sample=JSON.stringify([id,other].map(candidateId=>({candidateId,expectedRevision:"2",linkId:candidateId,evidenceDigest:"a".repeat(64),selection:{itemId:candidateId,expectedRevision:"1",originalPublicationId:candidateId,originalVersion:"1"}})));
 query.value=new URLSearchParams({step:"analysis",sample,candidateAfter:other,mode:"archived",search:"收纳"}).toString();
 const onNavigate=vi.fn();
 const v=render(<ConsoleNavigation pathname={`/workbench/supply/selection/${id}`} ariaLabel="移动导航" marketResearchAvailable onNavigate={onNavigate}/>);
 const current=screen.getByRole("link",{name:"AI建议与记录"});
 expect(current).toHaveAttribute("aria-current","page");
 expect(current.closest("li")).toHaveClass("console-nav-level-2");
 expect(screen.getByRole("link",{name:"智能选品"}).closest("li")!.querySelector(".console-nav-level-3")).toBeNull();
 const next=screen.getByRole("link",{name:"评估与决定"}),target=new URL(next.getAttribute("href")!,"https://test.invalid");
 expect(target.pathname).toBe(`/workbench/supply/selection/${id}`);
 expect(Object.fromEntries(target.searchParams)).toEqual({step:"decisions",sample,candidateAfter:other,mode:"archived",search:"收纳"});
 fireEvent.click(next);expect(onNavigate).toHaveBeenCalledTimes(1);
 v.rerender(<ConsoleNavigation pathname={`/workbench/supply/selection/${id}`} ariaLabel="移动导航"/>);
 expect(screen.getByRole("link",{name:"AI建议与记录"})).toHaveAttribute("title","AI建议与记录：业务暂未启用");
});
