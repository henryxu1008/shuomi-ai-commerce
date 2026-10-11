import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, expect, it, vi } from "vitest";
import Page from "./page";

const navigate = vi.hoisted(() => ({ redirect: vi.fn() }));
vi.mock("next/navigation", () => ({ redirect: navigate.redirect }));
afterEach(() => {
  cleanup();
  navigate.redirect.mockReset();
  vi.unstubAllEnvs();
});

it("keeps its independent homepage and exposes official tools when mounted", () => {
  vi.stubEnv("LISTINGKIT_TOOL_MARKET_ENABLED", "true");
  render(<Page />);
  expect(navigate.redirect).not.toHaveBeenCalled();
  expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("按业务场景启用工具");
  expect(screen.getByRole("link", { name: /浏览官方工具/ })).toHaveAttribute("href", "/workbench/tools/official");
});

it.each([undefined, "false", "TRUE"])("keeps an unconfigured module unavailable (%s)", (enabled) => {
  vi.stubEnv("LISTINGKIT_TOOL_MARKET_ENABLED", enabled);
  render(<Page />);
  expect(navigate.redirect).not.toHaveBeenCalled();
  expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("按业务场景启用工具");
  expect(screen.getAllByText("暂未启用")).toHaveLength(3);
  expect(screen.queryByRole("link", { name: /浏览官方工具/ })).not.toBeInTheDocument();
});
