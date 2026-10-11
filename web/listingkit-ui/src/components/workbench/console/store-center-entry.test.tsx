import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { StoreCenterEntry } from "./store-center-entry";
const fixture = vi.hoisted(() => ({context: {
  isLoading: false, isSwitching: false, selectionRequired: false,
  effectiveOrganization: {id: "org-a", permissions: [] as string[]},
  storeObservationsAvailable: true, error: null as unknown, blockingError: null as unknown, retry: vi.fn(),
}}));
vi.mock("@/components/providers/workbench-context-provider", () => ({useWorkbenchContext: () => fixture.context}));
beforeEach(() => { fixture.context.effectiveOrganization.permissions = ["workbench.store.read"]; fixture.context.error = fixture.context.blockingError = null; fixture.context.retry.mockClear(); });
afterEach(cleanup);
it("does not advertise observation pages without their current read permission", () => {
  const view = render(<StoreCenterEntry />);
  expect(screen.getByRole("link", {name: /管理我的店铺/})).toBeVisible();
  expect(screen.queryByRole("link", {name: /查看店铺商品/})).not.toBeInTheDocument();
  expect(screen.queryByRole("link", {name: /查看订单履约/})).not.toBeInTheDocument();
  fixture.context.effectiveOrganization.permissions.push("workbench.store.products.read");
  view.rerender(<StoreCenterEntry />);
  expect(screen.getByRole("link", {name: /查看店铺商品/})).toBeVisible();
  expect(screen.queryByRole("link", {name: /查看订单履约/})).not.toBeInTheDocument();
});
it.each(["error", "blockingError"] as const)("surfaces %s and allows context retry", key => {
  fixture.context[key] = new Error("context failed");
  render(<StoreCenterEntry />);
  expect(screen.getByRole("alert")).toBeVisible();
  fireEvent.click(screen.getByRole("button", {name: "重新确认"}));
  expect(fixture.context.retry).toHaveBeenCalledOnce();
  expect(screen.queryByText("正在读取工作台...")).not.toBeInTheDocument();
});
