import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import ServicesPage from "@/app/workbench/services/page";
import { firstConnectedConsoleEntry } from "@/lib/workbench/console-primary-entry";

const redirect = vi.hoisted(() => vi.fn());
vi.mock("next/navigation", () => ({ redirect }));

beforeEach(() => {
  redirect.mockReset();
  vi.stubEnv("LISTINGKIT_ECOSERVICES_ENABLED", "true");
  vi.stubEnv("LISTINGKIT_ECOSERVICES_NON_PAYMENT_ONLY", "false");
});
afterEach(() => { cleanup(); vi.unstubAllEnvs(); });

it("keeps the primary ecosystem introduction separate from its service catalog", () => {
  render(<ServicesPage />);
  expect(redirect).not.toHaveBeenCalled();
  expect(screen.getByRole("heading", { level: 1, name: "连接专业服务资源，补齐跨境经营关键环节" })).toBeVisible();
  for (const [name, href] of [
    ["浏览服务市场", "/workbench/services/market"],
    ["管理我的服务", "/workbench/services/mine"],
    ["提交入驻申请", "/workbench/services/join"],
  ]) expect(screen.getByRole("link", { name: new RegExp(name) })).toHaveAttribute("href", href);
  expect(screen.queryByRole("button", { name: /全部服务/ })).not.toBeInTheDocument();
  expect(firstConnectedConsoleEntry("/workbench/services", { ecoservicesAvailable: true })).toBe("/workbench/services");
});

it("retains the unavailable state when the deployment does not mount ecosystems", () => {
  vi.stubEnv("LISTINGKIT_ECOSERVICES_ENABLED", "false");
  render(<ServicesPage />);
  expect(redirect).not.toHaveBeenCalled();
  expect(screen.getByRole("heading", { name: "生态服务尚未开放" })).toBeVisible();
  expect(screen.queryByRole("link")).not.toBeInTheDocument();
  expect(firstConnectedConsoleEntry("/workbench/services", { ecoservicesAvailable: false })).toBeNull();
});

it("explains the nonpayment trial without advertising an open purchase path", () => {
  vi.stubEnv("LISTINGKIT_ECOSERVICES_NON_PAYMENT_ONLY", "true");
  render(<ServicesPage />);
  expect(screen.getByRole("status")).toHaveTextContent("服务草稿与发布及支付交易暂未开放");
  expect(screen.getByText("服务开放后填写需求，等待服务方确认")).toBeVisible();
  expect(screen.queryByRole("button")).not.toBeInTheDocument();
});
