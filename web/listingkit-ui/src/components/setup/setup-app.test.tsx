import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, test, vi } from "vitest";
import { SetupApp } from "./setup-app";

afterEach(() => vi.unstubAllGlobals());
test("new-install draft survives review/back and a blocked plan cannot dispatch", async () => {
  const calls: { path: string; body?: Record<string, unknown> }[] = [];
  vi.stubGlobal("fetch", vi.fn(async (path: string, options: RequestInit) => {
    const body = options?.body ? JSON.parse(String(options.body)) : undefined;
    calls.push({ path, body });
    const data = path.endsWith("session") ? { csrf: "csrf" } : path.endsWith("environment") ? { dockerReady: true, version: "a".repeat(40), instances: [], profiles: [{ id: "base", label: "基础", note: "空项目" }], blocks: [], pending: [] } : { id: "instance.original", mode: "fresh", status: "prepared", changes: [{ label: "安装名称", before: "不存在", after: body.project }], services: ["登录、数据存储、应用及所选功能"], blocks: ["当前端口被占用"] };
    return { ok: true, json: async () => data };
  }));
  const user = userEvent.setup(); render(<SetupApp />);
  await user.click(await screen.findByRole("button", { name: "安装新系统" }));
  const project = screen.getByRole("textbox", { name: "安装名称" });
  await user.clear(project); await user.type(project, "my-isolated-project");
  await user.click(screen.getByRole("button", { name: "检查并预览" }));
  expect(await screen.findByText("当前端口被占用")).toBeVisible();
  expect(screen.getByRole("button", { name: "确认安装新系统" })).toBeDisabled();
  await user.click(screen.getByRole("button", { name: "返回配置" }));
  expect(screen.getByRole("textbox", { name: "安装名称" })).toHaveValue("my-isolated-project");
  await waitFor(() => expect(calls.some(c => c.path.endsWith("/apply"))).toBe(false));
});

test("a durable running receipt can be reconciled after controller restart", async () => {
  vi.stubGlobal("fetch", vi.fn(async (path: string) => ({ ok: true, json: async () => path.endsWith("session") ? { csrf: "csrf" } : { dockerReady: true, version: "a".repeat(40), instances: [], profiles: [], blocks: [], pending: [{ id: "same.original-key", mode: "fresh", status: "running", changes: [], services: [], blocks: [] }] } })));
  const user = userEvent.setup(); render(<SetupApp />);
  await user.click(await screen.findByRole("button", { name: "查看操作记录" }));
  expect(screen.getByRole("button", { name: "检查操作结果" })).toBeEnabled();
});

test("a blocked retry explains the record and never confirms; changing input invalidates preview", async () => {
  const calls: string[] = [];
  vi.stubGlobal("fetch", vi.fn(async (path: string) => {
    calls.push(path);
    const data = path.endsWith("session") ? { csrf: "csrf" } : path.endsWith("recovery-preview") ? { id: "preview", operationId: "same.original", action: "retry", changes: [], blocks: ["存储中已有内容，保持原样"], expiresAt: "2099-01-01T00:00:00Z" } : { dockerReady: true, version: "a".repeat(40), instances: [], profiles: [], blocks: [], pending: [{ id: "same.original", name: "我的安装", mode: "fresh", status: "unknown", stage: "start", reason: "启动安装服务未完成", createdAt: "2026-11-11T12:00:00Z", changes: [], services: [], blocks: [] }] };
    return { ok: true, json: async () => data };
  }));
  const user = userEvent.setup(); render(<SetupApp />);
  expect(await screen.findByRole("heading", { name: "我的安装" })).toBeVisible();
  expect(screen.getByText("启动安装服务未完成")).toBeVisible();
  await user.click(screen.getByRole("button", { name: "查看操作记录" }));
  await user.click(screen.getByRole("button", { name: "检查能否重新尝试" }));
  expect(await screen.findByText("存储中已有内容，保持原样")).toBeVisible();
  expect(screen.getByRole("button", { name: "确认重新尝试" })).toBeDisabled();
  await user.click(screen.getByRole("button", { name: "浅色" }));
  await user.type(screen.getByRole("textbox", { name: "重试的网络地址范围" }), "10.251.128.0/24");
  expect(screen.queryByRole("button", { name: "确认重新尝试" })).not.toBeInTheDocument();
  expect(calls.some(c => c.endsWith("recovery-confirm") || c.endsWith("/apply"))).toBe(false);
});
