import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { OfficialAnnouncementsEntry, OfficialAnnouncementsPage } from "./official-announcements";
import type { PlatformAnnouncementScope } from "@/lib/api/notifications";

vi.mock("@/components/providers/workbench-context-provider", () => ({
  useWorkbenchContext: () => ({ user: { id: "publisher" }, effectiveOrganization: null }),
}));

const scope: PlatformAnnouncementScope = { channel: "platform/official", userId: "publisher", realm: "https://identity.example" };
const id = "4f5e1184-83c7-45b0-9dd9-c61242c9fb25";
const now = "2026-11-07T00:00:00Z";
const content = { category: "SYSTEM", title: "计划维护", summary: "维护摘要", paragraphs: ["公告正文内容"], target: { kind: "none", id: "" } };
const row = { id, category: content.category, title: content.title, revision: 1, state: "PUBLISHED", publishedAt: now, withdrawnAt: null };

function mount(child = <OfficialAnnouncementsPage scope={scope} />) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={client}>{child}</QueryClientProvider>);
}

beforeEach(() => {
  Object.defineProperty(HTMLDialogElement.prototype, "showModal", {
    configurable: true, value(this: HTMLDialogElement) { this.open = true; },
  });
  Object.defineProperty(HTMLDialogElement.prototype, "close", {
    configurable: true, value(this: HTMLDialogElement) { this.open = false; },
  });
});

afterEach(() => {
  cleanup();
  sessionStorage.clear();
  vi.unstubAllGlobals();
});

it("requires explicit confirmation and recovers an uncertain publication with the original request", async () => {
  let posts = 0;
  const fetch = vi.fn(async (_url: string, init: RequestInit) => {
    if (init.method !== "POST") return Response.json({ items: posts >= 2 ? [row] : [], next: "" });
    posts++;
    if (posts === 1) return Response.json({ code: "OUTCOME_UNKNOWN" }, { status: 503 });
    return Response.json({ key: new Headers(init.headers).get("Idempotency-Key"), operation: "publish", committedAt: now, resultId: id });
  });
  vi.stubGlobal("fetch", fetch);
  const page = mount();
  fireEvent.click(await screen.findByRole("button", { name: "新建公告" }));
  for (const [label, value] of [["标题", content.title], ["摘要", content.summary], ["正文", content.paragraphs[0]]]) {
    fireEvent.change(screen.getByLabelText(label), { target: { value } });
  }
  fireEvent.click(screen.getByRole("button", { name: "核对发送范围" }));
  const confirm = await screen.findByRole("dialog", { name: "确认发布官方公告" });
  expect(within(confirm).getByText("当前登录域内的所有已登录用户")).toBeInTheDocument();
  expect(within(confirm).getByText(content.paragraphs[0])).toBeInTheDocument();
  expect(posts).toBe(0);
  fireEvent.click(within(confirm).getByRole("button", { name: "取消" }));
  expect(screen.getByLabelText("标题")).toHaveValue(content.title);
  expect(posts).toBe(0);
  fireEvent.click(screen.getByRole("button", { name: "核对发送范围" }));
  fireEvent.click(screen.getByRole("button", { name: "确认发布" }));
  await screen.findByRole("button", { name: "核实原操作" });
  page.unmount();

  mount();
  fireEvent.click(await screen.findByRole("button", { name: "核实原操作" }));
  await screen.findByText("官方公告已发布。");
  await screen.findByRole("heading", { name: content.title });
  const writes = fetch.mock.calls.filter(([, init]) => init.method === "POST");
  expect(writes).toHaveLength(2);
  expect(new Headers(writes[0][1].headers).get("Idempotency-Key")).toBeTruthy();
  expect(new Headers(writes[0][1].headers).get("Idempotency-Key")).toBe(new Headers(writes[1][1].headers).get("Idempotency-Key"));
  expect(writes[0][1].body).toBe(writes[1][1].body);
  expect(JSON.parse(String(writes[1][1].body))).toEqual(content);
});

it("views stored content and withdraws with the expected version after confirmation", async () => {
  let withdrawn = false;
  const fetch = vi.fn(async (url: string, init: RequestInit) => {
    if (url.endsWith(id)) return Response.json(content);
    if (url.endsWith("/withdraw")) {
      withdrawn = true;
      return Response.json({ key: new Headers(init.headers).get("Idempotency-Key"), operation: "withdraw", committedAt: now, resultId: id });
    }
    return Response.json({ items: [withdrawn ? { ...row, revision: 2, state: "WITHDRAWN", withdrawnAt: now } : row], next: "" });
  });
  vi.stubGlobal("fetch", fetch);
  mount();
  fireEvent.click(await screen.findByRole("button", { name: "查看公告 计划维护" }));
  expect(await screen.findByText(content.paragraphs[0])).toBeInTheDocument();
  fireEvent.click(screen.getByRole("button", { name: "关闭公告正文" }));
  fireEvent.click(screen.getByRole("button", { name: "撤回公告 计划维护" }));
  expect(withdrawn).toBe(false);
  fireEvent.click(screen.getByRole("button", { name: "确认撤回" }));
  await screen.findByText("官方公告已撤回。");
  await screen.findByText(/系统公告 · 已撤回/);
  expect(screen.queryByRole("button", { name: "撤回公告 计划维护" })).toBeNull();
  expect(fetch.mock.calls.find(([url]) => url.endsWith("/withdraw"))?.[1].body).toBe('{"revision":1}');
});

it("only displays management after the platform backend authorizes the current identity", async () => {
  const fetch = vi.fn(async () => Response.json({ code: "FORBIDDEN" }, { status: 403 }));
  vi.stubGlobal("fetch", fetch);
  const page = mount(<><OfficialAnnouncementsEntry scope={scope} /><OfficialAnnouncementsPage scope={scope} /></>);
  await screen.findByText("只有平台管理员可以管理官方公告。");
  expect(screen.queryByRole("link", { name: "官方公告管理" })).toBeNull();
  expect(screen.queryByRole("button", { name: "新建公告" })).toBeNull();
  page.unmount();

  fetch.mockImplementation(async () => Response.json({ items: [], next: "" }));
  mount(<OfficialAnnouncementsEntry scope={scope} />);
  await waitFor(() => expect(screen.getByRole("link", { name: "官方公告管理" })).toHaveAttribute("href", "/workbench/notifications/announcements"));
});
