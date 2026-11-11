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
