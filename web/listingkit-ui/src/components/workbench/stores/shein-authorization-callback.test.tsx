import { act, cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { OfficialAuthorizationCallback,SheinAuthorizationCallback } from "./shein-authorization-callback";
import { StoreConnectionError } from "@/lib/api/store-connection";
const state = vi.hoisted(() => ({
  context: { user: { id: "actor" }, effectiveOrganization: { id: "org-1" } },
  complete: vi.fn(),
  query: vi.fn(),
  clear: vi.fn(),
  pending: vi.fn(),
}));
vi.mock("@/components/providers/workbench-context-provider", () => ({
  useWorkbenchContext: () => state.context,
}));
vi.mock("@/lib/api/store-connection", async (original) => ({
  ...(await original<typeof import("@/lib/api/store-connection")>()),
  readStoreAuthorization: state.pending,
  completeStoreConnection: state.complete,
  queryStoreConnection: state.query,
  clearStoreAuthorization: state.clear,
}));
const pending = {
  expectedUserId: "actor",
  expectedOrganizationId: "org-1",
  storeId: "11111111-1111-4111-8111-11111111111a",
  attemptId: "22222222-2222-4222-8222-22222222222b",
  expiresAt: new Date(Date.now() + 300000).toISOString(),
};
beforeEach(() => {
  state.context = {
    user: { id: "actor" },
    effectiveOrganization: { id: "org-1" },
  };
  state.pending.mockReturnValue(pending);
  state.complete.mockReset();
  state.query.mockReset();
  state.clear.mockReset();
  window.history.replaceState(
    null,
    "",
    "/workbench/stores/shein/callback?appid=app-1&state=synthetic-state&tempToken=synthetic-temp",
  );
});
afterEach(cleanup);
it("captures a TikTok code only in memory, scrubs URL and binds the original local app",async()=>{
 state.pending.mockReturnValue({...pending,appId:"us-app"});
 window.history.replaceState(null,"","/workbench/stores/tiktokshop/callback?code=private-code&state=private-state");
 state.complete.mockResolvedValue({connectionStatus:"connected"});
 render(<OfficialAuthorizationCallback platform="tiktokshop"/>);
 await userEvent.click(await screen.findByRole("button",{name:"完成官方连接"}));
 expect(window.location.search).toBe("");
 expect(state.pending).toHaveBeenCalledWith("tiktokshop");
 expect(state.complete).toHaveBeenCalledWith({...pending,appId:"us-app"},pending.storeId,{attemptId:pending.attemptId,appId:"us-app",state:"private-state",authCode:"private-code"});
 expect(document.body.textContent).not.toContain("private-code");
 expect(sessionStorage.length).toBe(0);
});
it("rejects duplicated TikTok state and a callback that supplies another app",async()=>{
 state.pending.mockReturnValue({...pending,appId:"us-app"});
 window.history.replaceState(null,"","/workbench/stores/tiktokshop/callback?code=code&state=a&state=b&appid=other");
 render(<OfficialAuthorizationCallback platform="tiktokshop"/>);
 await screen.findByText(/本次授权返回无法确认或已过期/);
 expect(state.complete).not.toHaveBeenCalled();
 expect(window.location.search).toBe("");
});
it.each(["switch", "unmount"])(
  "ignores a late callback result after %s",
  async (mode) => {
    let resolve!: (value: { connectionStatus: string }) => void;
    state.complete.mockImplementationOnce(
      () =>
        new Promise((r) => {
          resolve = r;
        }),
    );
    const view = render(<SheinAuthorizationCallback />);
    await userEvent.click(
      await screen.findByRole("button", { name: "完成官方连接" }),
    );
    if (mode === "switch") {
      state.context = {
        user: { id: "actor" },
        effectiveOrganization: { id: "org-2" },
      };
      view.rerender(<SheinAuthorizationCallback />);
    } else view.unmount();
    await act(async () => {
      resolve({ connectionStatus: "connected" });
    });
    expect(state.clear).not.toHaveBeenCalled();
    expect(screen.queryByText(/官方连接已完成/)).not.toBeInTheDocument();
    expect(state.complete).toHaveBeenCalledTimes(1);
  },
);
it("cleans callback secrets, waits for matching context and exchanges only by explicit POST", async () => {
  state.context.user.id = "other";
  const { rerender } = render(<SheinAuthorizationCallback />);
  expect(window.location.search).toBe("");
  expect(state.complete).not.toHaveBeenCalled();
  expect(
    await screen.findByRole("button", { name: "完成官方连接" }),
  ).toBeDisabled();
  state.context.user.id = "actor";
  rerender(<SheinAuthorizationCallback />);
  state.complete.mockRejectedValueOnce(
    new StoreConnectionError(503, "STORE_AUTHORIZATION_OUTCOME_UNKNOWN", true),
  );
  await userEvent.click(screen.getByRole("button", { name: "完成官方连接" }));
  await screen.findByText(/临时凭据交换结果未知/);
  expect(
    screen.queryByRole("button", { name: "完成官方连接" }),
  ).not.toBeInTheDocument();
  state.query.mockResolvedValueOnce({ connectionStatus: "connected" });
  await userEvent.click(screen.getByRole("button", { name: "核验原连接" }));
  await screen.findByText(/官方连接已完成/);
  expect(state.complete).toHaveBeenCalledTimes(1);
  expect(state.query).toHaveBeenCalledWith(
    pending,
    pending.storeId,
    pending.attemptId,
  );
});
it("rejects duplicate provider fields before any exchange", async () => {
  window.history.replaceState(
    null,
    "",
    "/workbench/stores/shein/callback?appid=app-1&state=a&state=b&tempToken=temp",
  );
  render(<SheinAuthorizationCallback />);
  await screen.findByText(/本次授权返回无法确认或已过期/);
  expect(
    screen.queryByRole("button", { name: "完成官方连接" }),
  ).not.toBeInTheDocument();
  expect(state.complete).not.toHaveBeenCalled();
  expect(window.location.search).toBe("");
});

it("scrubs Walmart credentials and requires confirmation of the verified merchant",async()=>{
 state.pending.mockReturnValue({...pending,appId:"walmart-app"});window.history.replaceState(null,"","/workbench/stores/walmart/callback?code=private-code&state=private-state&sellerId=partner-a&type=auth&clientId=walmart-app");
 state.complete.mockResolvedValueOnce({state:"merchant_confirmation_required",connectionStatus:"disconnected",walmart:{partnerId:"partner-a",merchantName:"Verified merchant"}}).mockResolvedValueOnce({connectionStatus:"connected"});
 render(<OfficialAuthorizationCallback platform="walmart"/>);expect(window.location.search).toBe("");expect(state.complete).not.toHaveBeenCalled();
 await userEvent.click(await screen.findByRole("button",{name:"完成官方连接"}));await screen.findByText("Walmart Partner ID：partner-a");expect(state.clear).not.toHaveBeenCalled();expect(document.body.textContent).not.toContain("private-code");
 await userEvent.click(screen.getByRole("button",{name:"确认连接此商家"}));expect(state.complete).toHaveBeenLastCalledWith({...pending,appId:"walmart-app"},pending.storeId,{attemptId:pending.attemptId,appId:"walmart-app",merchantConfirmation:"partner-a"});await screen.findByText(/官方连接已完成/);expect(state.clear).toHaveBeenCalledTimes(1)
});
it("Walmart callback cannot exchange under another organization or duplicated state",async()=>{
 state.pending.mockReturnValue({...pending,appId:"walmart-app"});window.history.replaceState(null,"","/workbench/stores/walmart/callback?code=secret&state=a&state=b&sellerId=partner-a&type=auth");render(<OfficialAuthorizationCallback platform="walmart"/>);await screen.findByText(/本次授权返回无法确认或已过期/);expect(state.complete).not.toHaveBeenCalled();expect(window.location.search).toBe("")
});
