import { afterEach, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { KernelReleasesPage } from "./kernel-releases";
import { proxyKernelReleases } from "@/lib/server/kernel-releases-proxy";

const state = vi.hoisted(() => ({ context: { user: { id: "admin" }, permissions: ["listingkit.platform_admin"], isLoading: false, isSwitching: false, error: null, blockingError: null } }));
vi.mock("@/components/providers/workbench-context-provider", () => ({ useWorkbenchContext: () => state.context }));
const id = "12345678-1234-4234-8234-123456789abc";
const payload = { version: "150.0.7871.186", os: "windows", arch: "amd64", minClientVersion: "1.8.0", maxClientVersionExclusive: "1.8.1-0", archiveFormat: "zip", size: 1, sha256: "a".repeat(64), downloadUrl: "https://assets.example.test/k.zip" };
const artifact = { ...payload, releaseId: id, recommended: false, downloadOrigins: ["https://assets.example.test"] };
const clients: QueryClient[] = [];
function mount(create = true) { const client = new QueryClient(); clients.push(client); return render(<QueryClientProvider client={client}><KernelReleasesPage expectedUserId="admin" create={create} /></QueryClientProvider>); }
function fill() {
  const values: [RegExp, string][] = [[/发行编号/, id], [/^内核版本/, payload.version], [/结束客户端版本/, payload.maxClientVersionExclusive], [/制品大小/, "1"], [/^SHA256/, payload.sha256], [/固定 HTTPS 下载地址/, payload.downloadUrl]];
  for (const [name, value] of values) fireEvent.change(screen.getByRole("textbox", { name }), { target: { value } });
}
function save() { fireEvent.click(screen.getByRole("button", { name: "登记并核实" })); }
afterEach(() => { cleanup(); clients.splice(0).forEach(c => c.clear()); sessionStorage.clear(); vi.unstubAllGlobals(); vi.unstubAllEnvs(); state.context.permissions = ["listingkit.platform_admin"]; });

it("waits for matching readback and prevents duplicate publication while in flight", async () => {
  let finish!: (response: Response) => void;
  const fetch = vi.fn().mockResolvedValueOnce(Response.json({ releaseId: id, state: "published" })).mockImplementationOnce(() => new Promise<Response>(resolve => { finish = resolve; }));
  vi.stubGlobal("fetch", fetch); mount(); fill(); save();
  await waitFor(() => expect(fetch).toHaveBeenCalledTimes(2));
  expect(screen.queryByRole("heading", { name: "已登记并读回确认" })).not.toBeInTheDocument();
  expect(screen.getByRole("button", { name: "生成编号" })).toBeDisabled();
  fireEvent.submit(screen.getByRole("button", { name: "正在登记和核实…" }).closest("form")!);
  expect(fetch.mock.calls.filter(([, options]) => options.method === "PUT")).toHaveLength(1);
  finish(Response.json(artifact)); await screen.findByRole("heading", { name: "已登记并读回确认" });
});
it("retains the original intent across remount after readback failure and only queries it", async () => {
  const fetch = vi.fn().mockResolvedValueOnce(Response.json({ releaseId: id, state: "published" })).mockResolvedValueOnce(Response.json({ code: "UNAVAILABLE" }, { status: 503 })).mockResolvedValueOnce(Response.json(artifact));
  vi.stubGlobal("fetch", fetch); const page = mount(); fill(); save();
  await screen.findByText(/服务暂不可用/); expect(screen.queryByRole("heading", { name: "已登记并读回确认" })).not.toBeInTheDocument();
  page.unmount(); mount(); await screen.findByRole("heading", { name: "登记结果待核实" });
  fireEvent.click(screen.getByRole("button", { name: "查询本次登记" }));
  await screen.findByRole("heading", { name: "已登记并读回确认" });
  expect(fetch.mock.calls.filter(([, options]) => options.method === "PUT")).toHaveLength(1);
});
it("keeps unknown writes locked when readback is absent or metadata mismatches", async () => {
  const fetch = vi.fn().mockRejectedValueOnce(new Error("lost response")).mockResolvedValueOnce(Response.json({ code: "KERNEL_RELEASE_NOT_FOUND" }, { status: 404 })).mockResolvedValueOnce(Response.json({ ...artifact, size: 2 }));
  vi.stubGlobal("fetch", fetch); mount(); fill(); save(); await screen.findByText(/登记结果待核实。保留原编号/);
  for (let i = 0; i < 2; i++) { fireEvent.click(screen.getByRole("button", { name: "查询本次登记" })); await waitFor(() => expect(screen.getByRole("button", { name: "查询本次登记" })).toBeEnabled()); }
  expect(screen.getByRole("button", { name: "生成编号" })).toBeDisabled();
  expect(screen.queryByRole("heading", { name: "已登记并读回确认" })).not.toBeInTheDocument();
  expect(fetch.mock.calls.filter(([, options]) => options.method === "PUT")).toHaveLength(1);
});
it("preserves editable input on a definitive permission rejection", async () => {
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(Response.json({ code: "FORBIDDEN" }, { status: 403 })));
  mount(); fill(); save(); await screen.findByText("无平台维护权限。");
  expect(screen.getByRole("textbox", { name: /发行编号/ })).toHaveValue(id);
  expect(screen.getByRole("button", { name: "生成编号" })).toBeEnabled();
});
it("allows an explicit same-ID retry only when the BFF confirms no publication was sent", async () => {
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(Response.json({ code: "NOT_SENT" }, { status: 503 })));
  mount(); fill(); save(); await screen.findByText(/本次登记尚未发送/);
  expect(screen.getByRole("textbox", { name: /发行编号/ })).toHaveValue(id);
  expect(screen.getByRole("button", { name: "登记并核实" })).toBeEnabled();
});
it("does not fetch management data for a nonadministrator", () => {
  const fetch = vi.fn(); vi.stubGlobal("fetch", fetch); state.context.permissions = []; mount();
  expect(screen.getByRole("heading", { name: "无平台维护权限或登录已失效" })).toBeVisible(); expect(fetch).not.toHaveBeenCalled();
});
it("validates the complete form before publishing", () => {
  const fetch = vi.fn(); vi.stubGlobal("fetch", fetch); mount(); fill();
  fireEvent.change(screen.getByRole("textbox", { name: /制品大小/ }), { target: { value: "1e3" } }); save();
  expect(screen.getByText(/请检查标出的字段/)).toBeVisible(); expect(fetch).not.toHaveBeenCalled();
});
it("shows an empty page with a next cursor rather than losing later compatible rows", async () => {
  const fetch = vi.fn().mockResolvedValueOnce(Response.json({ releases: [], nextCursor: id })).mockResolvedValueOnce(Response.json({ releases: [{ releaseId: id, version: payload.version, os: "windows", arch: "amd64", minClientVersion: "1.8.0", maxClientVersionExclusive: "", recommended: false }], nextCursor: "" }));
  vi.stubGlobal("fetch", fetch); mount(false); await screen.findByText("本页没有兼容记录。"); fireEvent.click(screen.getByRole("button", { name: "下一页" }));
  await screen.findByText(payload.version); expect(fetch.mock.calls[1][0]).toContain("cursor=" + id);
});
it("runs the production component, client and BFF together against isolated owner responses", async () => {
  vi.stubEnv("LISTINGKIT_PUBLIC_BASE_URL", "https://console.test"); vi.stubEnv("LISTINGKIT_SERVICE_API_BASE", "http://127.0.0.1:9000/api/v1");
  let stored: Record<string, unknown> | undefined, writes = 0;
  vi.stubGlobal("fetch", vi.fn(async (url: string, init: RequestInit = {}) => {
    if (url.startsWith("/api/account/kernel-releases")) { const headers = new Headers(init.headers); headers.set("Origin", "https://console.test"); return proxyKernelReleases(new Request("https://console.test" + url, { ...init, headers }), "unit-test-token", "admin"); }
    if (url.endsWith("/workbench/context")) return Response.json({ user: { id: "admin" }, homeOrganizationId: "org", effectiveOrganizationId: "org", selectionRequired: false, organizations: [{ id: "org", name: "Isolated", roles: [], permissions: ["listingkit.platform_admin"] }] });
    if (url.includes("/platform/browser/kernel-releases/")) { stored = JSON.parse(String(init.body)); writes++; return Response.json({ releaseId: id, state: "published" }); }
    if (url.includes("/artifact?") && stored) return Response.json({ ...stored, releaseId: id, recommended: false, downloadOrigins: ["https://assets.example.test"] });
    throw new Error("Unexpected fixture transport");
  }));
  mount(); fill(); save(); await screen.findByRole("heading", { name: "已登记并读回确认" });
  expect(writes).toBe(1); expect(stored).toEqual(payload); expect(Object.keys(stored!)).toHaveLength(9);
});
