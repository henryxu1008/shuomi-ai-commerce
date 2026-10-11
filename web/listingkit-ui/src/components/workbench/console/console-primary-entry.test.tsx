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

function expectDestination(page: () => unknown, destination: string) {
  fixture.redirect.mockImplementation(() => { throw new Error("NEXT_REDIRECT"); });
  expect(page).toThrow("NEXT_REDIRECT");
  expect(fixture.redirect).toHaveBeenCalledWith(destination);
}

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

it("opens the existing Agent configuration catalog without executing an Agent", () => {
  expectDestination(AgentsPage, "/workbench/agents/market");
});
it("opens the existing Store list", () => {
  expectDestination(StorePage, "/workbench/stores");
});

it.each(["goals", "stores", "alerts", "advice"])("opens the granted %s Cockpit page", mode => {
  fixture.context.operationsCockpitAvailable = true;
  fixture.context.permissions = [`workbench.cockpit.${mode}.read`];
  expectDestination(() => render(<WorkbenchPage />), `/workbench/overview/${mode}`);
});
it("uses navigation order when several Cockpit pages are granted", () => {
  fixture.context.operationsCockpitAvailable = true;
  fixture.context.permissions = ["workbench.cockpit.stores.read", "workbench.cockpit.goals.read"];
  expectDestination(() => render(<WorkbenchPage />), "/workbench/overview/goals");
});
it("distinguishes missing Cockpit permission from an unmounted module", () => {
  fixture.context.operationsCockpitAvailable = true;
  render(<WorkbenchPage />);
  expect(fixture.redirect).not.toHaveBeenCalled();
  expect(screen.getByText("没有运营驾驶舱查看权限")).toBeVisible();
  expect(screen.queryByText("经营数据暂未接入")).not.toBeInTheDocument();
});
it("keeps the honest overview when Cockpit is unmounted", () => {
  render(<WorkbenchPage />);
  expect(fixture.redirect).not.toHaveBeenCalled();
  expect(screen.getByText("经营数据暂未接入")).toBeVisible();
});
it.each(["isLoading", "isSwitching", "selectionRequired"] as const)("does not navigate before %s clears", flag => {
  fixture.context.operationsCockpitAvailable = true;
  fixture.context.permissions = ["workbench.cockpit.goals.read"];
  fixture.context[flag] = true;
  render(<WorkbenchPage />);
  expect(fixture.redirect).not.toHaveBeenCalled();
});
