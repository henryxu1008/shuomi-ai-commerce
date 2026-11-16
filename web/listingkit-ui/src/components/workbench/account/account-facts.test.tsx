import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, cleanup, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { StrictMode } from "react";
import { afterEach, it, expect, vi } from "vitest";
import { IdentityDates, RegionSettings } from "./account-facts";
import type { AccountPreferences } from "@/lib/api/account-facts";
afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});
it.each(["self", "another-user"])("shows only the current subject's official session authentication time: %s", async userId => {
  vi.stubGlobal("fetch", vi.fn((path: string) => Promise.resolve(Response.json(path === "/api/zitadel-auth/session"
    ? { ok: true, identity: { userId }, authenticatedAt: "2026-09-28T01:00:00.000Z" }
    : { schemaVersion: "account-identity-facts-v1", userId: "self", registeredAt: "2026-09-01T00:00:00Z", lastLogin: null, passwordChangedAt: null, source: "zitadel_auth_v1", readAt: "2026-09-28T02:00:00Z" }))));
  const { container } = render(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}><IdentityDates subject="self" /></QueryClientProvider>);
  await screen.findByText(/当前会话认证时间/);
  await waitFor(() => expect(container.querySelectorAll('time[datetime="2026-09-28T01:00:00.000Z"]')).toHaveLength(userId === "self" ? 1 : 0));
  expect(screen.queryByText(/暂无登录记录/)).not.toBeInTheDocument();
  if (userId !== "self") expect(await screen.findByText(/认证时间暂不可用/)).toBeVisible();
});

const version = "gn-0123456789abcdef";
const initial: AccountPreferences = {
  schemaVersion: "account-preferences-v1", userId: "self",
  country: "", province: "", city: "", updatedAt: null,
  readAt: "2026-09-28T00:00:00Z", source: "account_profile",
};
const china = {
  id: "CN", name: "China", cities: [],
  provinces: [{ id: "1", name: "浙江省", cities: [{ id: "3", name: "杭州市" }] },
    { id: "2", name: "广东省", cities: [{ id: "4", name: "深圳市" }] }],
};
const singapore = { id: "SG", name: "Singapore", provinces: [], cities: [{ id: "8", name: "Singapore" }] };

function mountRegion(stored = initial) {
  let saved = stored;
  const calls = vi.fn(async (path: string, init?: RequestInit) => {
    if (path === "/api/account/region-catalog") return Response.json({
      version,
      countries: [...Array.from({ length: 26 }, (_, i) => ({ id: "A" + String.fromCharCode(65 + i), name: "Placeholder " + i })),
        { id: "CN", name: "China" }, { id: "SG", name: "Singapore" }],
    });
    if (path.includes("?country=")) return Response.json({ version, country: path.endsWith("CN") ? china : singapore });
    if (init?.method === "PUT") {
      const { country } = JSON.parse(String(init.body));
      saved = { ...initial, ...(country ? { country: "Singapore", city: "Singapore" } : {}), updatedAt: "2026-09-28T01:00:00Z" };
    }
    return Response.json(saved);
  });
  vi.stubGlobal("fetch", calls);
  render(<StrictMode><QueryClientProvider client={new QueryClient()}><RegionSettings subject="self" /></QueryClientProvider></StrictMode>);
  return calls;
}

it("searches preset countries, clears descendants on changes and saves IDs after StrictMode replay", async () => {
  const calls = mountRegion();
  const user = userEvent.setup();
  const country = await screen.findByRole("combobox", { name: "国家／地区" });
  const save = screen.getByRole("button", { name: "保存地区资料" });
  const search = await screen.findByRole("searchbox", { name: "搜索国家／地区" });
  await user.type(search, "中国");
  expect(screen.queryByRole("option", { name: "新加坡" })).not.toBeInTheDocument();
  await user.selectOptions(country, "CN");
  await screen.findByRole("option", { name: "浙江省" });
  await user.selectOptions(screen.getByRole("combobox", { name: "省／州" }), "1");
  await user.selectOptions(screen.getByRole("combobox", { name: "城市" }), "3");
  expect(save).toBeEnabled();
  await user.selectOptions(screen.getByRole("combobox", { name: "省／州" }), "2");
  expect(screen.getByRole("combobox", { name: "城市" })).toHaveValue("");
  expect(save).toBeDisabled();
  await user.clear(search);
  await user.selectOptions(country, "SG");
  await screen.findByRole("option", { name: "Singapore" });
  expect(screen.getByRole("combobox", { name: "省／州" })).toBeDisabled();
  await user.selectOptions(screen.getByRole("combobox", { name: "城市" }), "8");
  await user.click(save);
  await waitFor(() => {
    const write = calls.mock.calls.find(([, init]) => init?.method === "PUT");
    expect(write && JSON.parse(String(write[1]?.body))).toEqual({ catalogVersion: version, country: "SG", province: "", city: "8" });
  });
  await waitFor(() => expect(screen.getByRole("combobox", { name: "城市" })).toHaveValue("8"));
});

it("preserves unmatched saved text until explicitly cleared", async () => {
  const calls = mountRegion({ ...initial, country: "旧国家名", province: "旧省", city: "旧城市" });
  const user = userEvent.setup();
  expect(await screen.findByText(/旧国家名 \/ 旧省 \/ 旧城市/)).toBeVisible();
  const save = screen.getByRole("button", { name: "保存地区资料" });
  expect(save).toBeDisabled();
  expect(calls.mock.calls.some(([, init]) => init?.method === "PUT")).toBe(false);
  await user.click(screen.getByRole("button", { name: "清空地区" }));
  await user.click(save);
  await waitFor(() => {
    const write = calls.mock.calls.find(([, init]) => init?.method === "PUT");
    expect(write && JSON.parse(String(write[1]?.body))).toEqual({ catalogVersion: version, country: "", province: "", city: "" });
  });
});
