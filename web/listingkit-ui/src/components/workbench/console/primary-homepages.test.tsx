import { cleanup, render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { afterEach, expect, it, vi } from "vitest";
import AgentsPage from "@/app/workbench/agents/page";
import ToolsPage from "@/app/workbench/tools/page";
import StorePage from "@/app/workbench/store-center/page";
import WorkbenchPage from "@/app/workbench/page";
import AIPage from "@/app/workbench/ai/page";

const fixture = vi.hoisted(() => ({ redirect: vi.fn(), context: {
  user: { id: "user-a" }, effectiveOrganization: { id: "org-a" }, permissions: [] as string[],
  operationsCockpitAvailable: false, aiWorkbenchAvailable: false, projectCenterAvailable: false,
  isLoading: false, isSwitching: false, selectionRequired: false, error: null, blockingError: null,
} }));
vi.mock("next/navigation", () => ({ redirect: fixture.redirect }));
vi.mock("@/components/providers/workbench-context-provider", () => ({ useWorkbenchContext: () => fixture.context }));
vi.mock("@/lib/server/knowledge-availability", () => ({ isKnowledgeAvailable: () => false }));
vi.mock("@/lib/server/report-center-availability", () => ({ isReportCenterAvailable: () => false }));
vi.mock("@/lib/server/product-title-review-request", () => ({ configuredProductReviewOrigin: () => null }));
vi.mock("@/lib/server/shein-records-availability", () => ({ isSheinRecordsAvailable: () => false }));
afterEach(() => { cleanup(); vi.unstubAllEnvs(); fixture.redirect.mockReset(); fixture.context.isSwitching = false; });

it.each([
  [AgentsPage, "按岗位启用智能体，打造属于自己的AI电商团队"],
  [ToolsPage, "按业务场景启用工具，让智能体获得可靠执行能力"],
  [StorePage, "统一管理店铺、商品与订单，让经营链路更清晰"],
  [WorkbenchPage, "经营全局，一屏掌握"],
  [AIPage, "AI工作台"],
])("retains the independent primary homepage", (Page, title) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(<QueryClientProvider client={client}><Page /></QueryClientProvider>);
  expect(fixture.redirect).not.toHaveBeenCalled();
  expect(screen.getByRole("heading", { level: 1, name: title })).toBeVisible();
});

it("keeps the tools introduction when its owner is not mounted without advertising active child links", () => {
  vi.stubEnv("LISTINGKIT_TOOL_MARKET_ENABLED", "false");
  render(<ToolsPage />);
  expect(fixture.redirect).not.toHaveBeenCalled();
  expect(screen.getByRole("heading", { name: "如何开始使用" })).toBeVisible();
  expect(screen.queryByRole("link", { name: /浏览官方工具/ })).not.toBeInTheDocument();
});

it("retains the AI homepage when a child becomes available", () => {
  fixture.context.projectCenterAvailable = true;
  render(<AIPage />);
  expect(fixture.redirect).not.toHaveBeenCalled();
  expect(screen.getByRole("link", { name: /进入项目中心/ })).toHaveAttribute("href", "/workbench/ai/projects");
  fixture.context.projectCenterAvailable = false;
});
