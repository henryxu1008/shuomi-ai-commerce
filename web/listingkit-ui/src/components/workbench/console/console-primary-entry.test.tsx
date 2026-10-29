import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import WorkbenchPage from "@/app/workbench/page";
import DataPage from "@/app/workbench/data/page";
import AgentsPage from "@/app/workbench/agents/page";
import StorePage from "@/app/workbench/store-center/page";

const fixture = vi.hoisted(() => ({
  redirect: vi.fn(),
  context: { operationsCockpitAvailable: false, permissions: [] as string[],
    isLoading: false, isSwitching: false, selectionRequired: false,
    effectiveOrganization: { id: "org-a" } as { id: string } | null, error: null, blockingError: null },
}));
it("removes the duplicate skill library entry from the intelligent market introduction", () => {
  render(<AgentsPage/>);
  expect(screen.queryByRole("link", {name: /进入技能库/})).not.toBeInTheDocument();
  expect(screen.queryByRole("heading", {name: "技能库"})).not.toBeInTheDocument();
  expect(screen.getByRole("link", {name: /提交定制需求/})).toBeVisible();
});
vi.mock("next/navigation", () => ({ redirect: fixture.redirect }));
vi.mock("@/components/providers/workbench-context-provider", () => ({ useWorkbenchContext: () => fixture.context }));
beforeEach(() => {
  fixture.redirect.mockReset();
  fixture.context.operationsCockpitAvailable = false;
  fixture.context.permissions = [];
  fixture.context.isLoading = fixture.context.isSwitching = fixture.context.selectionRequired = false;
  fixture.context.effectiveOrganization = { id: "org-a" };
});
afterEach(() => { cleanup(); vi.unstubAllEnvs(); });


it.each([
  ["true", "true"],
  ["true", "false"],
  ["false", "true"],
  [undefined, "false"],
])("keeps the data introduction and links only mounted capabilities (%s/%s)", (data, collections) => {
  vi.stubEnv("LISTINGKIT_DATA_SERVICES_ENABLED", data);
  vi.stubEnv("LISTINGKIT_PRODUCT_COLLECTIONS_ENABLED", collections);
  fixture.redirect.mockImplementation(() => { throw new Error("NEXT_REDIRECT"); });
  render(<DataPage />);
  expect(fixture.redirect).not.toHaveBeenCalled();
  expect(screen.getByRole("heading", { level: 1, name: "让数据成为可调用、可沉淀、可复用的业务资产" })).toBeVisible();
  const destinations = [
    { name: /浏览数据市场/, href: "/workbench/data/market", available: data === "true" },
    { name: /管理API/, href: "/workbench/data/api", available: data === "true" },
    { name: /查看我的数据/, href: "/workbench/data/mine", available: collections === "true" },
  ];
  for (const { name, href, available } of destinations) {
    const link = screen.queryByRole("link", { name });
    if (available) expect(link).toHaveAttribute("href", href);
    else expect(link).not.toBeInTheDocument();
  }
  expect(screen.queryAllByText("暂未启用")).toHaveLength(destinations.filter(item => !item.available).length);
  expect(screen.getByRole("heading", { name: "如何开始使用" })).toBeVisible();
});

it.each([[AgentsPage, '按岗位启用智能体，打造属于自己的AI电商团队'], [StorePage, '统一管理店铺、商品与订单，让经营链路更清晰']])('keeps a standalone introduction', (Page, title) => {
  render(<Page />);
  expect(fixture.redirect).not.toHaveBeenCalled();
  expect(screen.getByRole('heading', { name: title, level: 1 })).toBeVisible();
});
it.each(['goals', 'stores', 'alerts', 'advice'])('keeps the cockpit homepage with %s permission', mode => {
  fixture.context.operationsCockpitAvailable = true;
  fixture.context.permissions = [`workbench.cockpit.${mode}.read`];
  render(<WorkbenchPage />);
  expect(fixture.redirect).not.toHaveBeenCalled();
  expect(screen.getByRole('heading', { level: 1, name: '经营全局，一屏掌握' })).toBeVisible();
});
it('distinguishes missing permission from an unmounted module', () => {
  fixture.context.operationsCockpitAvailable = true;
  render(<WorkbenchPage />);
  expect(screen.getAllByText('没有运营驾驶舱查看权限').length).toBeGreaterThan(0);
  expect(screen.queryByText('经营数据暂未接入')).not.toBeInTheDocument();
});
it('keeps an honest overview when unmounted', () => {
  render(<WorkbenchPage />);
  expect(fixture.redirect).not.toHaveBeenCalled();
  expect(screen.getAllByText('经营数据暂未接入').length).toBeGreaterThan(0);
});
it.each(['isLoading', 'isSwitching', 'selectionRequired'] as const)('clears the cockpit during %s', flag => {
  fixture.context[flag] = true;
  render(<WorkbenchPage />);
  expect(fixture.redirect).not.toHaveBeenCalled();
  expect(screen.queryByRole('heading', { level: 1 })).not.toBeInTheDocument();
});
