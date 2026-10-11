import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { SupportLauncher } from "./support-launcher";

const config = { origin: "https://support.example.test", websiteToken: "public-website-token", subject: `platform:${"a".repeat(64)}` };
const frame = () => screen.getByTitle("平台客服聊天") as HTMLIFrameElement;
function loaded(origin = config.origin, source: MessageEventSource | null = frame().contentWindow) {
  fireEvent(window, new MessageEvent("message", { origin, source, data: `chatwoot-widget:${JSON.stringify({ event: "loaded", config: { authToken: "visitor-token" } })}` }));
}
async function open() { fireEvent.click(screen.getByRole("button", { name: "联系客服" })); await waitFor(() => expect(frame()).toBeInTheDocument()); }

describe("SupportLauncher", () => {
  beforeEach(() => { vi.stubGlobal("fetch", vi.fn().mockImplementation((_url, init) => Promise.resolve(Response.json(init?.method === "POST" ? { ok: true } : config)))); });
  afterEach(() => { vi.useRealTimers(); vi.restoreAllMocks(); vi.unstubAllGlobals(); });
  it("keeps unconfigured support disabled without extra requests", () => {
    render(<SupportLauncher available={false} />);
    expect(screen.getByRole("button", { name: "联系客服" })).toBeDisabled();
    expect(fetch).not.toHaveBeenCalled();
  });
  it("keeps the fresh native frame hidden until server identification succeeds", async () => {
    render(<SupportLauncher available />); await open();
    const child = frame();
    expect(child).not.toBeVisible();
    expect(child.src).toBe(`${config.origin}/widget?website_token=public-website-token`);
    expect(child).toHaveAttribute("referrerpolicy", "no-referrer");
    const send = vi.spyOn(child.contentWindow!, "postMessage");
    loaded();
    await waitFor(() => expect(child).toBeVisible());
    expect(fetch).toHaveBeenLastCalledWith("/api/support/session", expect.objectContaining({ method: "POST", body: JSON.stringify({ subject: config.subject, widgetAuthToken: "visitor-token" }) }));
    expect(send).toHaveBeenCalledWith(expect.stringContaining('"event":"config-set"'), config.origin);
    expect(send).toHaveBeenCalledWith(expect.stringContaining('"event":"toggle-open"'), config.origin);
    expect(document.cookie).not.toContain("cw_conversation");
    expect(localStorage.getItem("cw_conversation")).toBeNull();
  });
  it("requires both the exact origin and the exact iframe source", async () => {
    render(<SupportLauncher available />); await open();
    loaded("https://evil.test"); loaded(config.origin, window);
    expect(fetch).toHaveBeenCalledTimes(1);
    expect(frame()).not.toBeVisible();
    loaded(); await waitFor(() => expect(frame()).toBeVisible());
  });
  it("destroys the frame when identification fails", async () => {
    vi.mocked(fetch).mockImplementation((_url, init) => Promise.resolve(init?.method === "POST" ? Response.json({ code: "SUPPORT_UNAVAILABLE" }, { status: 502 }) : Response.json(config)));
    render(<SupportLauncher available />); await open(); loaded();
    expect(await screen.findByRole("alert")).toHaveTextContent("暂时无法连接客服");
    expect(screen.queryByTitle("平台客服聊天")).not.toBeInTheDocument();
  });
  it("closing aborts a pending handshake and ignores its late success", async () => {
    let resolve!: (value: Response) => void;
    vi.mocked(fetch).mockImplementation((_url, init) => init?.method === "POST" ? new Promise(r => { resolve = r; }) : Promise.resolve(Response.json(config)));
    render(<SupportLauncher available />); await open(); loaded();
    const signal = vi.mocked(fetch).mock.calls[1][1]?.signal;
    fireEvent.click(screen.getByRole("button", { name: "关闭客服" }));
    expect(signal?.aborted).toBe(true);
    await act(async () => resolve(Response.json({ ok: true })));
    expect(screen.queryByTitle("平台客服聊天")).not.toBeInTheDocument();
  });
  it("unmounting for a changed user/org removes the frame and aborts work", async () => {
    const view = render(<SupportLauncher available />); await open();
    const signal = vi.mocked(fetch).mock.calls[0][1]?.signal;
    view.unmount();
    expect(signal?.aborted).toBe(true);
    expect(screen.queryByTitle("平台客服聊天")).not.toBeInTheDocument();
  });
  it("rejects a different subject when the window regains focus", async () => {
    render(<SupportLauncher available />); await open(); loaded(); await waitFor(() => expect(frame()).toBeVisible());
    vi.mocked(fetch).mockResolvedValue(Response.json({ ...config, subject: `platform:${"b".repeat(64)}` }));
    fireEvent.focus(window);
    expect(await screen.findByRole("alert")).toHaveTextContent("登录状态已变化");
    expect(screen.queryByTitle("平台客服聊天")).not.toBeInTheDocument();
  });
  it("expires a stalled native loaded handshake", async () => {
    vi.useFakeTimers(); render(<SupportLauncher available />);
    await act(async () => fireEvent.click(screen.getByRole("button", { name: "联系客服" })));
    await act(async () => vi.advanceTimersByTimeAsync(20_000));
    expect(screen.getByRole("alert")).toHaveTextContent("连接客服超时");
    expect(screen.queryByTitle("平台客服聊天")).not.toBeInTheDocument();
  });
});
