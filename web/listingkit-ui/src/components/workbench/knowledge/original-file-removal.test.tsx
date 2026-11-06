import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, it, vi } from "vitest";
import { OriginalFileRemoval } from "./original-file-removal";

const scope = { userId: "admin", organizationId: "org-a" };
const sourceId = "4841d296-ef14-4c16-8d25-a7667e534feb";
const revisionId = "d6c33a5b-95dd-4a3c-a420-652ec52b5f08";
const operationId = "f5dd72eb-e639-436e-b057-38c713d12ce9";
const original = { sourceId, sourceVersion: 3, revisionId, revisionNumber: 1, bytes: 35, canDelete: true };
const receipt = { ...original, canDelete: false, operationId };

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

function openPanel() {
  const client = new QueryClient();
  const onAuthorityError = vi.fn();
  Object.defineProperty(HTMLDialogElement.prototype, "showModal", { configurable: true, value: function (this: HTMLDialogElement) { this.open = true; } });
  Object.defineProperty(HTMLDialogElement.prototype, "close", { configurable: true, value: function (this: HTMLDialogElement) { this.open = false; } });
  render(<QueryClientProvider client={client}><OriginalFileRemoval scope={scope} sourceId={sourceId} onAuthorityError={onAuthorityError} /></QueryClientProvider>);
  return onAuthorityError;
}

it.each([
  { result: original, action: "永久删除原文件" },
  { result: { ...receipt, state: "DELETE_DISPATCHED" }, action: "继续核实" },
])("recovers the original-file read and shows $action after a temporary failure", async ({ result, action }) => {
  const fetch = vi.fn<(path: string, init: RequestInit) => Promise<Response>>(async () => Response.json(result))
    .mockResolvedValueOnce(Response.json({ code: "KNOWLEDGE_UNAVAILABLE" }, { status: 503 }));
  vi.stubGlobal("fetch", fetch);
  openPanel();

  const retry = await screen.findByRole("button", { name: "重新读取原文件状态" });
  expect(screen.queryByRole("button", { name: action })).not.toBeInTheDocument();
  await userEvent.click(retry);
  expect(await screen.findByRole("button", { name: action })).toBeEnabled();
  expect(fetch).toHaveBeenCalledTimes(2);
  for (const [path, init] of fetch.mock.calls) {
    expect(path).toBe("/api/workbench/knowledge-sources/" + sourceId + "/original-file");
    expect(init.method ?? "GET").toBe("GET");
    expect(new Headers(init.headers).get("X-Expected-Organization-ID")).toBe(scope.organizationId);
  }
});

it("keeps an original-file 404 hidden without a retry or delete action", async () => {
  vi.stubGlobal("fetch", vi.fn(async () => Response.json({ code: "KNOWLEDGE_NOT_FOUND" }, { status: 404 })));
  const onReadError = openPanel();
  await waitFor(() => expect(onReadError).toHaveBeenCalledWith(expect.objectContaining({ status: 404 })));
  expect(screen.queryByRole("button", { name: "重新读取原文件状态" })).not.toBeInTheDocument();
  expect(screen.queryByRole("button", { name: "永久删除原文件" })).not.toBeInTheDocument();
});

it("confirms the exact original, then resolves both pending states through the same operation", async () => {
  let state = "DELETE_DISPATCHED";
  let posted = false;
  const fetch = vi.fn(async (path: string, init: RequestInit) => {
    const headers = new Headers(init.headers);
    expect(headers.get("X-Expected-Organization-ID")).toBe(scope.organizationId);
    if (!init.method) return Response.json(posted ? { ...receipt, state } : original);
    if (path.endsWith("/delete")) {
      expect(path).toMatch(new RegExp(revisionId + "/delete$"));
      expect(headers.get("If-Match")).toBe('"3"');
      posted = true;
      return Response.json({ code: "OUTCOME_UNKNOWN" }, { status: 502 });
    }
    expect(path).toMatch(new RegExp(operationId + "/continue$"));
    expect(headers.get("Idempotency-Key")).toBe(operationId);
    state = state === "DELETE_DISPATCHED" ? "OBJECT_ABSENCE_PROVEN" : "RELEASED";
    return Response.json({ ...receipt, state });
  });
  vi.stubGlobal("fetch", fetch);
  openPanel();
  const user = userEvent.setup();
  await user.click(await screen.findByRole("button", { name: "永久删除原文件" }));
  const dialog = screen.getByRole("dialog", { name: "永久删除原文件？" });
  expect(within(dialog).getByText(sourceId)).toBeVisible();
  expect(within(dialog).getByText("35 字节")).toBeVisible();
  expect(within(dialog).getByRole("button", { name: "取消" })).toHaveFocus();
  await user.click(within(dialog).getByRole("button", { name: "取消" }));
  expect(fetch).toHaveBeenCalledTimes(1);
  await user.click(screen.getByRole("button", { name: "永久删除原文件" }));
  await user.click(screen.getByRole("button", { name: "确认永久删除" }));
  await screen.findByRole("heading", { name: "删除待核实" });
  await waitFor(() => expect(screen.getByRole("button", { name: "继续核实" })).toBeEnabled());
  await user.click(screen.getByRole("button", { name: "继续核实" }));
  await screen.findByRole("heading", { name: "容量待确认" });
  expect(screen.getByText("操作单号 " + operationId)).toBeVisible();
  await user.click(screen.getByRole("button", { name: "继续核实" }));
  await screen.findByRole("heading", { name: "已释放" });
  expect(screen.getByRole("link", { name: "查看资源与额度" })).toHaveAttribute("href", "/workbench/account/organization/resources");
  expect(fetch.mock.calls.filter(([, init]) => init.method === "POST").map(([path]) => path.split("/").at(-1))).toEqual(["delete", "continue", "continue"]);
});

it("replays the confirmed key when the first response is lost and no operation is visible", async () => {
  const keys: (string | null)[] = [];
  const fetch = vi.fn(async (_path: string, init: RequestInit) => {
    if (!init.method) return Response.json(original);
    keys.push(new Headers(init.headers).get("Idempotency-Key"));
    return keys.length === 1
      ? Response.json({ code: "OUTCOME_UNKNOWN" }, { status: 504 })
      : Response.json({ ...receipt, state: "DELETE_DISPATCHED" }, { status: 202 });
  });
  vi.stubGlobal("fetch", fetch);
  openPanel();
  const user = userEvent.setup();
  await user.click(await screen.findByRole("button", { name: "永久删除原文件" }));
  await user.click(screen.getByRole("button", { name: "确认永久删除" }));
  await waitFor(() => expect(screen.getByRole("button", { name: "继续核实" })).toBeEnabled());
  await user.click(screen.getByRole("button", { name: "继续核实" }));
  await screen.findByText("操作单号 " + operationId);
  expect(keys).toEqual([expect.any(String), keys[0]]);
});
