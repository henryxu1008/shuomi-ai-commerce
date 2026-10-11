import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { WorkbenchContextProvider } from "@/components/providers/workbench-context-provider";
import { AcquisitionAPIError, type AcquisitionOperation } from "@/lib/api/product-acquisition";
import { AcquisitionPage } from "./acquisition-page";

const calls = vi.hoisted(() => ({ acquire: vi.fn(), verify: vi.fn(), push: vi.fn() }));
vi.mock("next/navigation", () => ({ useRouter: () => ({ push: calls.push }), useSearchParams: () => new URLSearchParams() }));
vi.mock("@/lib/api/product-acquisition", async original => ({ ...await original<object>(), acquire1688: calls.acquire, verify1688: calls.verify }));
vi.mock("./product-agent-panel", () => ({ ProductAgentPanel: () => null }));

const operationId = "11111111-1111-4111-8111-111111111111";
const storageKey = "listingkit.acquisition.intent";
function context(userId = "actor-A", organizationId = "organization-A") {
  return { user: { id: userId }, homeOrganizationId: organizationId, effectiveOrganizationId: organizationId, selectionRequired: false, organizations: [{ id: organizationId, name: organizationId, roles: [], permissions: [] }] };
}
function mount() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return render(<QueryClientProvider client={client}><WorkbenchContextProvider><AcquisitionPage /></WorkbenchContextProvider></QueryClientProvider>);
}
async function submit() {
  await userEvent.type(await screen.findByLabelText("1688 商品页或 offer ID"), "123");
  await userEvent.click(screen.getByRole("button", { name: "提交采集" }));
}
beforeEach(() => {
  localStorage.clear(); sessionStorage.clear();
  calls.acquire.mockReset(); calls.verify.mockReset(); calls.push.mockReset();
  vi.stubGlobal("fetch", vi.fn<typeof fetch>().mockImplementation(async () => Response.json(context())));
});
afterEach(() => { cleanup(); vi.restoreAllMocks(); vi.unstubAllGlobals(); localStorage.clear(); sessionStorage.clear(); });

it("restores quota intent after the entire shell remounts and resumes the same key", async () => {
  calls.acquire.mockRejectedValueOnce(new AcquisitionAPIError("DATA_QUOTA_INSUFFICIENT", 409));
  calls.acquire.mockResolvedValueOnce({ outcome: "acquiring", operationId });
  calls.acquire.mockResolvedValueOnce({ outcome: "published", operationId, productKey: "crawler:1688:123" });
  const first = mount(); await submit();
  await screen.findByRole("alert");
  const original = calls.acquire.mock.calls[0][0] as AcquisitionOperation;
  first.unmount(); const restored = mount();
  await userEvent.click(await screen.findByRole("button", { name: "补足额度后使用原 key 重试" }));
  await waitFor(() => expect(calls.acquire).toHaveBeenCalledTimes(2));
  expect(calls.acquire.mock.calls[1][0]).toMatchObject(original);
  expect(await screen.findByRole("status")).toHaveTextContent("正在采集");
  restored.unmount(); mount();
  await userEvent.click(await screen.findByRole("button", { name: "补足额度后使用原 key 重试" }));
  await waitFor(() => expect(calls.acquire).toHaveBeenCalledTimes(3));
  expect(calls.acquire.mock.calls[2][0]).toMatchObject(original);
  expect(calls.verify).not.toHaveBeenCalled();
  await waitFor(() => expect(localStorage.getItem(storageKey)).toBeNull());
});

it("restores unknown intent for explicit Verify without dispatching a new acquisition", async () => {
  calls.acquire.mockRejectedValueOnce(new AcquisitionAPIError("OUTCOME_UNKNOWN", 503));
  calls.verify.mockResolvedValueOnce({ outcome: "prepared", operationId });
  const first = mount(); await submit(); await screen.findByRole("alert");
  const original = calls.acquire.mock.calls[0][0] as AcquisitionOperation;
  first.unmount(); mount();
  await userEvent.click(await screen.findByRole("button", { name: "使用原 key 核实" }));
  await waitFor(() => expect(calls.verify).toHaveBeenCalledOnce());
  expect(calls.verify.mock.calls[0][0]).toMatchObject(original);
  expect(calls.acquire).toHaveBeenCalledOnce();
});

it("keeps the collector-capacity original across shell remount and explicitly continues it", async () => {
  calls.acquire.mockRejectedValueOnce(new AcquisitionAPIError("ACQUISITION_CAPACITY", 429));
  calls.acquire.mockResolvedValueOnce({ outcome: "published", operationId, productKey: "crawler:1688:123" });
  const first = mount(); await submit(); await screen.findByRole("alert");
  const original = calls.acquire.mock.calls[0][0] as AcquisitionOperation;
  expect(JSON.parse(localStorage.getItem(storageKey)!)).toMatchObject(original);
  expect(screen.getByRole("button", { name: "提交采集" })).toBeDisabled();
  expect(screen.getByRole("button", { name: "继续原请求" })).toBeEnabled();
  first.unmount(); mount();
  await userEvent.click(await screen.findByRole("button", { name: "继续原请求" }));
  await waitFor(() => expect(calls.acquire).toHaveBeenCalledTimes(2));
  expect(calls.acquire.mock.calls[1][0]).toMatchObject(original);
  expect(calls.verify).not.toHaveBeenCalled();
  await waitFor(() => expect(localStorage.getItem(storageKey)).toBeNull());
});

it("retains the original slot across another identity and organization then returning", async () => {
  calls.acquire.mockRejectedValueOnce(new AcquisitionAPIError("DATA_QUOTA_INSUFFICIENT", 409));
  calls.acquire.mockResolvedValueOnce({ outcome: "acquiring", operationId });
  const first = mount(); await submit(); await screen.findByRole("alert");
  const original = calls.acquire.mock.calls[0][0] as AcquisitionOperation;
  first.unmount();
  vi.stubGlobal("fetch", vi.fn<typeof fetch>().mockResolvedValue(Response.json(context("actor-B", "organization-B"))));
  const foreign = mount();
  await screen.findByText(/另一个企业或身份下有尚未确认的采集请求/);
  expect(screen.getByRole("button", { name: "提交采集" })).toBeDisabled();
  expect(screen.getByLabelText("1688 商品页或 offer ID")).toHaveValue("");
  expect(calls.acquire).toHaveBeenCalledOnce();
  foreign.unmount();
  vi.stubGlobal("fetch", vi.fn<typeof fetch>().mockResolvedValue(Response.json(context())));
  mount();
  await userEvent.click(await screen.findByRole("button", { name: "补足额度后使用原 key 重试" }));
  await waitFor(() => expect(calls.acquire).toHaveBeenCalledTimes(2));
  expect(calls.acquire.mock.calls[1][0]).toMatchObject(original);
});

it("does not dispatch when the original intent cannot be saved", async () => {
  mount(); await screen.findByLabelText("1688 商品页或 offer ID");
  vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => { throw new Error("storage denied"); });
  await submit();
  expect(calls.acquire).not.toHaveBeenCalled();
  expect(await screen.findByRole("alert")).toHaveTextContent("无法保留原采集请求");
});

it("does not overwrite another submitted key written after hydration", async () => {
  mount(); await screen.findByLabelText("1688 商品页或 offer ID");
  const other = { userId: "actor-A", organizationId: "organization-A", key: operationId, source: "https://detail.1688.com/offer/456.html" };
  localStorage.setItem(storageKey, JSON.stringify(other));
  await submit();
  expect(calls.acquire).not.toHaveBeenCalled();
  expect(JSON.parse(localStorage.getItem(storageKey)!)).toEqual(other);
});

it("keeps the original slot and blocks a new submission when terminal cleanup fails", async () => {
  let removal: ReturnType<typeof vi.spyOn>;
  calls.acquire.mockImplementationOnce(async () => {
    removal = vi.spyOn(Storage.prototype, "removeItem").mockImplementation(() => { throw new Error("storage denied"); });
    return { outcome: "failed", operationId };
  });
  mount(); await submit();
  expect(await screen.findByRole("alert")).toHaveTextContent("无法保留原采集请求");
  const original = calls.acquire.mock.calls[0][0] as AcquisitionOperation;
  expect(JSON.parse(localStorage.getItem(storageKey)!)).toMatchObject(original);
  expect(screen.getByRole("button", { name: "提交采集" })).toBeDisabled();
  expect(calls.acquire).toHaveBeenCalledOnce();
  removal!.mockRestore();
  calls.verify.mockResolvedValueOnce({ outcome: "failed", operationId });
  await userEvent.click(screen.getByRole("button", { name: "使用原 key 核实" }));
  await waitFor(() => expect(localStorage.getItem(storageKey)).toBeNull());
  expect(calls.verify.mock.calls[0][0]).toMatchObject(original);
  expect(screen.getByRole("button", { name: "提交采集" })).toBeEnabled();
  expect(calls.acquire).toHaveBeenCalledOnce();
});

it.each(["not-json", JSON.stringify({ userId: "actor-A", organizationId: "organization-A", key: operationId, source: "https://evil.example/offer/123.html" }), "x".repeat(10000)])("does not discard an invalid stored intent to dispatch a new key (%#)", async raw => {
  localStorage.setItem(storageKey, raw);
  mount(); await submit();
  expect(calls.acquire).not.toHaveBeenCalled();
  expect(localStorage.getItem(storageKey)).toBe(raw);
  expect(await screen.findByRole("alert")).toHaveTextContent("无法保留原采集请求");
});

it("resumes the original request after quota metadata storage fails and the shell reloads", async () => {
  const originalSave = Storage.prototype.setItem;
  let writes = 0;
  const storage = vi.spyOn(Storage.prototype, "setItem").mockImplementation(function (this: Storage, key: string, value: string) {
    if (++writes === 2) throw new Error("quota metadata write failed");
    originalSave.call(this, key, value);
  });
  calls.acquire.mockRejectedValueOnce(new AcquisitionAPIError("DATA_QUOTA_INSUFFICIENT", 409));
  calls.acquire.mockResolvedValueOnce({ outcome: "published", operationId, productKey: "crawler:1688:123" });
  const first = mount(); await submit();
  expect(await screen.findByRole("alert")).toHaveTextContent("无法保留原采集请求");
  const original = calls.acquire.mock.calls[0][0] as AcquisitionOperation;
  expect(JSON.parse(localStorage.getItem(storageKey)!)).toEqual(original);
  storage.mockRestore(); first.unmount(); mount();
  await userEvent.click(await screen.findByRole("button", { name: "继续原请求" }));
  await waitFor(() => expect(calls.acquire).toHaveBeenCalledTimes(2));
  expect(calls.acquire.mock.calls[1][0]).toMatchObject(original);
  expect(calls.verify).not.toHaveBeenCalled();
});
