import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { SupportLauncher } from "./support-launcher";
const subject = `platform:${"a".repeat(64)}`;
const history = { subject, messages: [{ id: 20, content: "人工客服回复", type: "outgoing", createdAt: 1, attachments: [] }], nextBefore: null };
describe("platform support panel", () => {
  beforeEach(() => { vi.stubGlobal("fetch", vi.fn().mockImplementation((input: string, init?: RequestInit) => Promise.resolve(Response.json(
    input.endsWith("session") ? { subject } : init?.method === "POST" ? { subject, conversation: { id: 11, status: "open", createdAt: 1 }, messageId: 22 } : input.endsWith("conversations") ? { subject, conversations: [{ id: 11, status: "open", createdAt: 1 }] } : history)))); });
  afterEach(() => { vi.unstubAllGlobals(); vi.restoreAllMocks(); });
  it("leaves unconfigured support disabled", () => { render(<SupportLauncher available={false} />); expect(screen.getByRole("button", { name: "联系客服" })).toBeDisabled(); expect(fetch).not.toHaveBeenCalled(); });
  it("opens the platform chat panel with personal history and no vendor iframe", async () => { render(<SupportLauncher available />); fireEvent.click(screen.getByRole("button", { name: "联系客服" })); expect(await screen.findByText("人工客服回复")).toBeVisible(); expect(document.querySelector("iframe")).toBeNull(); expect(screen.getByRole("textbox", { name: "消息" })).toBeVisible(); });
  it("sends a question once and clears the confirmed draft", async () => { render(<SupportLauncher available />); fireEvent.click(screen.getByRole("button", { name: "联系客服" })); const input = await screen.findByRole("textbox", { name: "消息" }); fireEvent.change(input, { target: { value: "如何使用平台？" } }); fireEvent.click(screen.getByRole("button", { name: "发送" })); await waitFor(() => expect(input).toHaveValue("")); expect(vi.mocked(fetch).mock.calls.filter(([, options]) => options?.method === "POST")).toHaveLength(1); });
  it("keeps an unconfirmed draft and does not resend automatically", async () => { const original = vi.mocked(fetch).getMockImplementation()!; vi.mocked(fetch).mockImplementation((input, init) => init?.method === "POST" ? Promise.reject(new Error("offline")) : original(input, init)); render(<SupportLauncher available />); fireEvent.click(screen.getByRole("button", { name: "联系客服" })); const input = await screen.findByRole("textbox", { name: "消息" }); fireEvent.change(input, { target: { value: "我的问题" } }); fireEvent.click(screen.getByRole("button", { name: "发送" })); expect(await screen.findByRole("alert")).toHaveTextContent("未确认"); expect(input).toHaveValue("我的问题"); expect(vi.mocked(fetch).mock.calls.filter(([, options]) => options?.method === "POST")).toHaveLength(1); });
  it("clears the previous transcript when the login changes", async () => { render(<SupportLauncher available />); await act(async () => { fireEvent.click(screen.getByRole("button", { name: "联系客服" })); }); await screen.findByText("人工客服回复"); vi.mocked(fetch).mockResolvedValue(Response.json({ subject: `platform:${"b".repeat(64)}` })); await act(async () => { fireEvent.focus(window); }); await waitFor(() => expect(screen.queryByText("人工客服回复")).not.toBeInTheDocument()); expect(await screen.findByRole("alert")).toHaveTextContent("登录状态"); });

  it.each(["manual", "focus"])("recovers the first unconfirmed send through %s refresh without resending", async (refresh) => {
    let committed = false;
    vi.mocked(fetch).mockImplementation(async (input, init) => {
      const path = String(input);
      if (init?.method === "POST") {
        committed = true;
        throw new Error("response lost after commit");
      }
      if (path.endsWith("session")) return Response.json({ subject });
      if (path.endsWith("conversations")) return Response.json({ subject, conversations: committed ? [{ id: 11, status: "open", createdAt: 1 }] : [] });
      return Response.json({ subject, messages: [{ id: 22, content: "已保存的首次咨询", type: "incoming", createdAt: 1, attachments: [] }], nextBefore: 22 });
    });
    render(<SupportLauncher available />);
    fireEvent.click(screen.getByRole("button", { name: "联系客服" }));
    const input = await screen.findByRole("textbox", { name: "消息" });
    fireEvent.change(input, { target: { value: "首次咨询草稿" } });
    const screenshot = new File(["image"], "error.png", { type: "image/png" });
    fireEvent.change(screen.getByLabelText("添加截图"), { target: { files: [screenshot] } });
    fireEvent.click(screen.getByRole("button", { name: "发送" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("未确认");
    await act(async () => {
      if (refresh === "focus") fireEvent.focus(window);
      else fireEvent.click(screen.getByRole("button", { name: "刷新记录" }));
    });
    expect(await screen.findByText("已保存的首次咨询")).toBeVisible();
    expect(screen.getByRole("textbox", { name: "消息" })).toHaveValue("首次咨询草稿");
    expect(screen.getByText("error.png")).toBeVisible();
    expect(screen.getByRole("button", { name: "重新发送" })).toBeVisible();
    expect(screen.getByRole("alert")).toHaveTextContent("未确认");
    expect(screen.getByRole("button", { name: "查看更早消息" })).toBeVisible();
    expect(vi.mocked(fetch).mock.calls.filter(([url]) => String(url).endsWith("conversations"))).toHaveLength(3);
    expect(vi.mocked(fetch).mock.calls.filter(([, init]) => init?.method === "POST")).toHaveLength(1);
    expect(vi.mocked(fetch).mock.calls.some(([url]) => String(url).endsWith("conversations/11/messages"))).toBe(true);
  });

  it("keeps an empty recovery unknown and never creates a conversation while refreshing", async () => {
    vi.mocked(fetch).mockImplementation(async (input, init) => {
      if (init?.method === "POST") throw new Error("outcome unknown");
      return Response.json(String(input).endsWith("session") ? { subject } : { subject, conversations: [] });
    });
    render(<SupportLauncher available />);
    fireEvent.click(screen.getByRole("button", { name: "联系客服" }));
    const input = await screen.findByRole("textbox", { name: "消息" });
    fireEvent.change(input, { target: { value: "保留未知草稿" } });
    fireEvent.click(screen.getByRole("button", { name: "发送" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("未确认");
    await act(async () => { fireEvent.click(screen.getByRole("button", { name: "刷新记录" })); });
    expect(vi.mocked(fetch).mock.calls.filter(([url, init]) => String(url).endsWith("conversations") && init?.method === "GET")).toHaveLength(2);
    expect(input).toHaveValue("保留未知草稿");
    expect(screen.getByRole("alert")).toHaveTextContent("未确认");
    expect(vi.mocked(fetch).mock.calls.filter(([, init]) => init?.method === "POST")).toHaveLength(1);
    expect(vi.mocked(fetch).mock.calls.some(([url]) => String(url).includes("/messages"))).toBe(false);
  });

  it("preserves loaded older messages and their cursor when refreshing an existing conversation", async () => {
    const original = vi.mocked(fetch).getMockImplementation()!;
    vi.mocked(fetch).mockImplementation(async (input, init) => {
      if (String(input).endsWith("messages?before=20")) return Response.json({ subject, messages: [{ id: 10, content: "更早回复", type: "outgoing", createdAt: 0, attachments: [] }], nextBefore: null });
      if (String(input).endsWith("/messages")) return Response.json({ ...history, nextBefore: 20 });
      return original(input, init);
    });
    render(<SupportLauncher available />);
    fireEvent.click(screen.getByRole("button", { name: "联系客服" }));
    fireEvent.click(await screen.findByRole("button", { name: "查看更早消息" }));
    expect(await screen.findByText("更早回复")).toBeVisible();
    await act(async () => { fireEvent.click(screen.getByRole("button", { name: "刷新记录" })); });
    expect(screen.getByText("更早回复")).toBeVisible();
    expect(screen.getByText("人工客服回复")).toBeVisible();
    expect(screen.queryByRole("button", { name: "查看更早消息" })).not.toBeInTheDocument();
    expect(vi.mocked(fetch).mock.calls.filter(([, init]) => init?.method === "POST")).toHaveLength(0);
  });

  it("ignores a recovered conversation arriving after the panel closes", async () => {
    let lists = 0;
    let resolveList!: (response: Response) => void;
    vi.mocked(fetch).mockImplementation(async (input) => {
      if (String(input).endsWith("session")) return Response.json({ subject });
      if (++lists === 1) return Response.json({ subject, conversations: [] });
      return new Promise<Response>((resolve) => { resolveList = resolve; });
    });
    render(<SupportLauncher available />);
    fireEvent.click(screen.getByRole("button", { name: "联系客服" }));
    await screen.findByRole("textbox", { name: "消息" });
    fireEvent.click(screen.getByRole("button", { name: "刷新记录" }));
    await waitFor(() => expect(lists).toBe(2));
    fireEvent.click(screen.getByRole("button", { name: "关闭客服" }));
    await act(async () => { resolveList(Response.json({ subject, conversations: [{ id: 11, status: "open", createdAt: 1 }] })); });
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(vi.mocked(fetch).mock.calls.some(([url]) => String(url).includes("/messages"))).toBe(false);
  });
});
