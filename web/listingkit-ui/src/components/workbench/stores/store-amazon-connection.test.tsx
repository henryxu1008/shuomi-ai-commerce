import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, it, vi } from "vitest";

import type { WorkbenchStore } from "@/lib/api/workbench-stores";
import { StoreOfficialConnection } from "./store-official-connection";

const store = {
  id: "afafb20d-132d-44c2-bff8-03eb949c7410",
  version: 3,
  platform: "amazon",
  region: "US",
  recordStatus: "active",
} as WorkbenchStore;

function renderAmazon(canWrite = true) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <StoreOfficialConnection store={store} canWrite={canWrite} administrator={false}
        scope={{ expectedOrganizationId: "org-a", expectedUserId: "actor-a" }} onChanged={vi.fn()} />
    </QueryClientProvider>,
  );
}

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

it("offers Amazon authorization for the current store and handles rejection", async () => {
  const request = vi.fn(async (_url: string, init?: RequestInit) =>
    init?.method === "POST" ? new Response(null, { status: 403 }) : Response.json({ status: "not_connected" }),
  );
  vi.stubGlobal("fetch", request);
  renderAmazon();
  await screen.findByText("尚未连接");
  await userEvent.click(screen.getByRole("button", { name: "连接 Amazon" }));
  expect(await screen.findByRole("alert")).toHaveTextContent("无法连接 Amazon");
  expect(request).toHaveBeenLastCalledWith(
    "/workbench/stores/amazon/begin",
    expect.objectContaining({
      method: "POST",
      headers: expect.objectContaining({
        "X-Expected-Organization-ID": "org-a",
        "X-Expected-User-ID": "actor-a",
        "X-Store-ID": store.id,
        "If-Match": '"3"',
      }),
    }),
  );
});

it("reads saved authorization without opening a write action for a viewer", async () => {
  vi.stubGlobal("fetch", vi.fn(async () => Response.json({ status: "authorization_saved" })));
  renderAmazon(false);
  expect(await screen.findByText("已保存授权")).toBeVisible();
  expect(screen.queryByRole("button", { name: /连接 Amazon|重新授权/ })).toBeNull();
});

it("preserves unknown status when the authorization service is unavailable", async () => {
  vi.stubGlobal("fetch", vi.fn(async () => new Response(null, { status: 404 })));
  renderAmazon();
  expect(await screen.findByRole("alert")).toHaveTextContent("暂时无法读取 Amazon 授权状态");
  expect(screen.queryByRole("button", { name: "连接 Amazon" })).toBeNull();
});
