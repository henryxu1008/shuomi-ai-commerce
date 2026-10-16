import { beforeEach, expect, it, vi } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ToolPage } from "./tool-page";
const navigation = vi.hoisted(() => ({ push: vi.fn() }));
vi.mock("next/navigation", () => ({ useRouter: () => navigation }));
const switchGuards = vi.hoisted(() => new Set<() => boolean>());
const context = vi.hoisted(() => ({
  user: { id: "actor" },
  effectiveOrganization: { id: "org-a", name: "A企业" } as {
    id: string;
    name: string;
  } | null,
  permissions: [
    "workbench.tools.read",
    "workbench.tools.manage",
    "workbench.tools.customize",
  ],
  isLoading: false,
  isSwitching: false,
  selectionRequired: false,
  error: null,
  blockingError: null,
  retry: vi.fn(),
  registerOrganizationSwitchGuard: (guard: () => boolean) => {
    switchGuards.add(guard);
    return () => {
      switchGuards.delete(guard);
    };
  },
}));
vi.mock("@/components/providers/workbench-context-provider", () => ({
  useWorkbenchContext: () => context,
}));
vi.mock("next/link", () => ({
  default: ({
    href,
    children,
    ...props
  }: React.AnchorHTMLAttributes<HTMLAnchorElement>) => (
    <a href={href} {...props}>
      {children}
    </a>
  ),
}));
const tool = {
  id: "product-acquisition",
  version: "0.1.0",
  name: "商品采集插件",
  category: "采集",
  description: "采集1688商品资料",
  status: "AVAILABLE",
  reason: "",
  activation: null,
  localCapture: true,
  onlineCapture: true,
  download: true,
};
const fullDemand = {
  scenario: "商品管理",
  contactName: "测试联系人",
  contactMethod: "test-wechat",
  consent: true as const,
};
const fullRequest = {
  ...fullDemand,
  createdBy: "actor",
  consentVersion: "tool-customization-contact-v1",
  attachments: [],
};
const summary = (record: Record<string, unknown>) =>
  Object.fromEntries(
    [
      "id",
      "organizationId",
      "kind",
      "title",
      "stage",
      "revision",
      "createdAt",
      "updatedAt",
    ].map((key) => [key, record[key]]),
  );

it("blocks organization switching while reference files are read and never dispatches after unmount", async () => {
  const fetch = vi.fn(async () =>
    Response.json({ tools: [tool], canManage: true, canCustomize: true }),
  );
  vi.stubGlobal("fetch", fetch);
  const readers: { result: string | null; onload: (() => void) | null }[] = [];
  vi.stubGlobal(
    "FileReader",
    class {
      result: string | null = null;
      onload: (() => void) | null = null;
      onerror = null;
      readAsDataURL() {
        readers.push(this);
      }
    },
  );
  const view = mount("submit");
  await waitFor(() =>
    expect(screen.getByRole("button", { name: "提交定制需求" })).toBeEnabled(),
  );
  for (const [label, value] of [
    ["工具需求名称 *", "采集需求"],
    ["工具功能与执行规则 *", "采集商品资料"],
    ["使用场景 *", "商品管理"],
    ["联系人 *", "测试联系人"],
    ["手机或微信 *", "test"],
  ])
    fireEvent.change(screen.getByLabelText(label), { target: { value } });
  fireEvent.click(screen.getByRole("checkbox"));
  fireEvent.change(screen.getByLabelText("上传参考资料"), {
    target: { files: [new File(["test"], "flow.txt", { type: "text/plain" })] },
  });
  fireEvent.click(screen.getByRole("button", { name: "提交定制需求" }));
  await waitFor(() => expect(readers).toHaveLength(1));
  expect([...switchGuards].some((guard) => !guard())).toBe(true);
  view.unmount();
  readers[0].result = "data:text/plain;base64,dGVzdA==";
  readers[0].onload?.();
  await Promise.resolve();
  await Promise.resolve();
  expect(
    fetch.mock.calls.filter(
      (call: unknown[]) =>
        (call[1] as RequestInit | undefined)?.method === "POST",
    ),
  ).toHaveLength(0);
});
beforeEach(() => {
  sessionStorage.clear();
  vi.unstubAllGlobals();
  switchGuards.clear();
  navigation.push.mockReset();
  context.effectiveOrganization = { id: "org-a", name: "A企业" };
});
const mount = (
  mode: "official" | "mine" | "custom" | "submit" | "progress" | "admin",
  requestId?: string,
) => {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  const content = (
    mode: Parameters<typeof ToolPage>[0]["mode"],
    requestId?: string,
  ) => (
    <QueryClientProvider client={client}>
      <ToolPage mode={mode} requestId={requestId} />
    </QueryClientProvider>
  );
  const view = render(content(mode, requestId));
  return {
    ...view,
    navigate: (
      mode: Parameters<typeof ToolPage>[0]["mode"],
      requestId?: string,
    ) => view.rerender(content(mode, requestId)),
  };
};
it("enables a real enterprise choice and rereads server state", async () => {
  let enabled = false;
  const fetch = vi.fn(async (_url: string, init?: RequestInit) => {
    if (init?.method === "PUT") {
      enabled = true;
      const h = new Headers(init.headers);
      return Response.json({
        commandId: h.get("Idempotency-Key"),
        operation: "activation",
        id: tool.id,
        revision: "1",
        committedAt: "2026-10-09T00:00:00Z",
      });
    }
    return Response.json({
      tools: [
        {
          ...tool,
          activation: enabled
            ? {
                toolId: tool.id,
                enabled: true,
                revision: "1",
                updatedAt: "2026-10-09T00:00:00Z",
              }
            : null,
        },
      ],
      canManage: true,
      canCustomize: true,
    });
  });
  vi.stubGlobal("fetch", fetch);
  mount("official");
  fireEvent.click(await screen.findByRole("button", { name: "立即启用" }));
  await screen.findByRole("button", { name: "已启用" });
  expect(
    fetch.mock.calls.some(
      (c) =>
        new Headers(c[1]?.headers).get("X-Expected-Organization-ID") ===
        "org-a",
    ),
  ).toBe(true);
});
it("specialists without enterprise membership record versioned progress", async () => {
  context.effectiveOrganization = null;
  const id = "aa1043df-c64c-499d-9c99-57598f7bff18",
    now = "2026-10-09T00:00:00Z";
  let updated = false;
  const writes: RequestInit[] = [];
  vi.stubGlobal(
    "fetch",
    vi.fn(async (url: string, init?: RequestInit) => {
      if (init?.method === "POST") {
        writes.push(init);
        updated = true;
        return Response.json({
          commandId: new Headers(init.headers).get("Idempotency-Key"),
          operation: "progress",
          id,
          revision: "2",
          committedAt: now,
        });
      }
      const record = {
        ...fullRequest,
        id,
        organizationId: "customer-org",
        kind: "DATA",
        title: "采购商品资料",
        description: "保存授权来源商品资料",
        stage: updated ? "EVALUATING" : "SUBMITTED",
        revision: updated ? "2" : "1",
        createdAt: now,
        updatedAt: now,
      };
      if (url.endsWith("/" + id))
        return Response.json({
          request: record,
          nextEventsBefore: "",
          events: [
            {
              revision: record.revision,
              stage: record.stage,
              note: updated ? "正在核实采集范围" : "需求已保存",
              occurredAt: now,
            },
          ],
        });
      return Response.json({
        items: [summary(record)],
        nextCursor: "",
      });
    }),
  );
  mount("admin");
  fireEvent.click(
    await screen.findByRole("button", { name: /采购商品资料.*待评估/ }),
  );
  fireEvent.change(await screen.findByLabelText("处理阶段"), {
    target: { value: "EVALUATING" },
  });
  fireEvent.change(screen.getByLabelText("真实处理进度 / 关闭原因"), {
    target: { value: "正在核实采集范围" },
  });
  fireEvent.click(screen.getByRole("button", { name: "记录进度" }));
  await screen.findByText("正在核实采集范围");
  expect(writes).toHaveLength(1);
  expect(new Headers(writes[0].headers).get("If-Match")).toBe('"1"');
  expect(new Headers(writes[0].headers).get("X-Expected-Organization-ID")).toBe(
    "",
  );
});
it("unknown response retains its original key through remount", async () => {
  const keys: string[] = [];
  vi.stubGlobal(
    "fetch",
    vi.fn(async (_url: string, init?: RequestInit) => {
      if (init?.method === "PUT") {
        keys.push(new Headers(init.headers).get("Idempotency-Key")!);
        throw new Error("lost");
      }
      return Response.json({
        tools: [tool],
        canManage: true,
        canCustomize: true,
      });
    }),
  );
  const first = mount("official");
  fireEvent.click(await screen.findByRole("button", { name: "立即启用" }));
  await screen.findByRole("button", { name: "重试原操作" });
  first.unmount();
  mount("official");
  fireEvent.click(await screen.findByRole("button", { name: "重试原操作" }));
  await waitFor(() => expect(keys).toHaveLength(2));
  expect(keys[0]).toBe(keys[1]);
});

it("allows switching after an unknown outcome and retries only the restored original scope", async () => {
  const writes: { key: string; organization: string }[] = [];
  const lost = Promise.withResolvers<Response>();
  vi.stubGlobal(
    "fetch",
    vi.fn(async (_url: string, init?: RequestInit) => {
      if (init?.method === "PUT") {
        const headers = new Headers(init.headers);
        writes.push({
          key: headers.get("Idempotency-Key")!,
          organization: headers.get("X-Expected-Organization-ID")!,
        });
        if (writes.length === 1) return lost.promise;
        return Response.json({
          commandId: headers.get("Idempotency-Key"),
          operation: "activation",
          id: tool.id,
          revision: "1",
          committedAt: "2026-10-09T00:00:00Z",
        });
      }
      return Response.json({
        tools: [tool],
        canManage: true,
        canCustomize: true,
      });
    }),
  );
  const first = mount("official");
  fireEvent.click(await screen.findByRole("button", { name: "立即启用" }));
  await waitFor(() => expect(writes).toHaveLength(1));
  expect(switchGuards.size).toBeGreaterThan(0);
  expect(
    [...switchGuards].every((guard) => guard()),
    "in-flight dispatch must block switching",
  ).toBe(false);
  lost.reject(new Error("response lost"));
  await screen.findByRole("button", { name: "重试原操作" });
  await waitFor(() =>
    expect(
      [...switchGuards].every((guard) => guard()),
      "retained unknown intent must allow switching away",
    ).toBe(true),
  );
  first.unmount();
  context.effectiveOrganization = { id: "org-b", name: "B企业" };
  const second = mount("official");
  await screen.findByRole("button", { name: "立即启用" });
  expect(
    screen.queryByRole("button", { name: "重试原操作" }),
  ).not.toBeInTheDocument();
  expect(
    writes,
    "entering another enterprise must not replay the retained command",
  ).toHaveLength(1);
  second.unmount();
  context.effectiveOrganization = { id: "org-a", name: "A企业" };
  mount("official");
  fireEvent.click(await screen.findByRole("button", { name: "重试原操作" }));
  await waitFor(() => expect(writes).toHaveLength(2));
  expect(writes[1]).toEqual(writes[0]);
});
it("readonly members cannot enable the enterprise tool", async () => {
  vi.stubGlobal(
    "fetch",
    vi
      .fn()
      .mockResolvedValue(
        Response.json({ tools: [tool], canManage: false, canCustomize: false }),
      ),
  );
  mount("official");
  expect(
    await screen.findByRole("button", { name: "立即启用" }),
  ).toBeDisabled();
});
it("online-only enabled tools do not expose an unavailable local receiver", async () => {
  const activation = {
    toolId: tool.id,
    enabled: true,
    revision: "1",
    updatedAt: "2026-10-09T00:00:00Z",
  };
  vi.stubGlobal(
    "fetch",
    vi.fn().mockResolvedValue(
      Response.json({
        tools: [{ ...tool, activation, localCapture: false, download: false }],
        canManage: true,
        canCustomize: true,
      }),
    ),
  );
  mount("mine");
  await screen.findByRole("heading", { name: "商品采集插件" });
  expect(
    screen.queryByRole("link", { name: "打开插件接收页" }),
  ).not.toBeInTheDocument();
  expect(screen.getByText("本地插件采集尚未开放")).toBeInTheDocument();
  expect(screen.getByRole("link", { name: "在线采集" })).toBeInTheDocument();
});
it("customization submission opens its persisted demand and progress", async () => {
  const id = "aa1043df-c64c-499d-9c99-57598f7bff18",
    now = "2026-10-09T00:00:00Z";
  let demand: { kind: string; title: string; description: string } | null =
    null;
  vi.stubGlobal(
    "fetch",
    vi.fn(async (url: string, init?: RequestInit) => {
      if (init?.method === "POST") {
        demand = JSON.parse(String(init.body));
        return Response.json({
          commandId: new Headers(init.headers).get("Idempotency-Key"),
          operation: "create",
          id,
          revision: "1",
          committedAt: now,
        });
      }
      if (url.endsWith("/market"))
        return Response.json({
          tools: [tool],
          canManage: true,
          canCustomize: true,
        });
      const record = {
        ...fullRequest,
        id,
        organizationId: "org-a",
        kind: "DATA",
        title: "采购商品资料",
        description: "保存授权来源的商品资料",
        stage: "SUBMITTED",
        revision: "1",
        createdAt: now,
        updatedAt: now,
      };
      if (url.endsWith("/requests/" + id))
        return Response.json({
          request: record,
          nextEventsBefore: "",
          events: [
            {
              revision: "1",
              stage: "SUBMITTED",
              note: "需求已保存，待硕米专员评估",
              occurredAt: now,
            },
          ],
        });
      return Response.json({ items: [summary(record)], nextCursor: "" });
    }),
  );
  const view = mount("submit");
  await waitFor(() =>
    expect(screen.getByRole("button", { name: "提交定制需求" })).toBeEnabled(),
  );
  fireEvent.change(screen.getByLabelText("工具需求名称 *"), {
    target: { value: "采购商品资料" },
  });
  fireEvent.change(screen.getByLabelText("工具功能与执行规则 *"), {
    target: { value: "保存授权来源的商品资料" },
  });
  fireEvent.change(screen.getByLabelText("使用场景 *"), {
    target: { value: fullDemand.scenario },
  });
  fireEvent.change(screen.getByLabelText("联系人 *"), {
    target: { value: fullDemand.contactName },
  });
  fireEvent.change(screen.getByLabelText("手机或微信 *"), {
    target: { value: fullDemand.contactMethod },
  });
  expect(screen.getByRole("checkbox")).not.toBeChecked();
  fireEvent.submit(
    screen.getByRole("button", { name: "提交定制需求" }).closest("form")!,
  );
  expect(await screen.findByRole("alert")).toHaveTextContent("明确同意");
  expect(demand).toBeNull();
  fireEvent.click(screen.getByRole("checkbox"));
  fireEvent.click(screen.getByRole("button", { name: "提交定制需求" }));
  await waitFor(() =>
    expect(navigation.push).toHaveBeenCalledWith(
      "/workbench/tools/custom/progress?request=" + id,
    ),
  );
  view.navigate("progress", id);
  await screen.findByText("需求已保存，待硕米专员评估");
  expect(demand).toEqual({
    ...fullDemand,
    kind: "DATA",
    title: "采购商品资料",
    description: "保存授权来源的商品资料",
  });
});

it("loads earlier progress without losing current detail or its write revision", async () => {
  context.effectiveOrganization = null;
  const id = "aa1043df-c64c-499d-9c99-57598f7bff18";
  const record = {
    ...fullRequest,
    id,
    organizationId: "customer-org",
    kind: "DATA",
    title: "长历史需求",
    description: "保存授权来源商品资料",
    stage: "SUBMITTED",
    revision: "100",
    createdAt: "2026-10-09T00:00:00Z",
    updatedAt: "2026-10-09T00:00:00Z",
  };
  let failEarlier = true;
  let denyEarlier = false;
  const writes: RequestInit[] = [];
  const fetch = vi.fn(async (url: string, init?: RequestInit) => {
    if (init?.method === "POST") {
      writes.push(init);
      return Response.json({
        commandId: new Headers(init.headers).get("Idempotency-Key"),
        operation: "progress",
        id,
        revision: "101",
        committedAt: record.updatedAt,
      });
    }
    if (url.endsWith("/requests"))
      return Response.json({
        items: [summary(record)],
        nextCursor: "",
      });
    const earlier = url.endsWith("?eventsBefore=85");
    if (url.includes("?eventsBefore=") && denyEarlier)
      return Response.json({ code: "FORBIDDEN" }, { status: 403 });
    if (earlier && failEarlier) throw new Error("temporarily unavailable");
    return Response.json({
      request: record,
      nextEventsBefore: earlier ? "69" : "85",
      events: Array.from({ length: 16 }, (_, i) => {
        const revision = (earlier ? 69 : 85) + i;
        return {
          revision: String(revision),
          stage: "SUBMITTED",
          note: "进度记录 " + revision,
          occurredAt: record.updatedAt,
        };
      }),
    });
  });
  vi.stubGlobal("fetch", fetch);
  mount("admin");
  fireEvent.click(await screen.findByRole("button", { name: /长历史需求/ }));
  await screen.findByText("进度记录 100");
  expect(screen.queryByText("进度记录 84")).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole("button", { name: "加载更早进度" }));
  await screen.findByText(/更早进度暂时无法读取/);
  expect(screen.getByText("进度记录 100")).toBeInTheDocument();
  failEarlier = false;
  fireEvent.click(screen.getByRole("button", { name: "加载更早进度" }));
  await screen.findByText("进度记录 84");
  const notes = screen
    .getAllByText(/^进度记录 /)
    .map((element) => element.textContent);
  expect(notes[0]).toBe("进度记录 69");
  expect(notes.at(-1)).toBe("进度记录 100");
  fireEvent.change(screen.getByLabelText("真实处理进度 / 关闭原因"), {
    target: { value: "新的进度说明" },
  });
  fireEvent.click(screen.getByRole("button", { name: "记录进度" }));
  await waitFor(() => expect(writes).toHaveLength(1));
  expect(new Headers(writes[0].headers).get("If-Match")).toBe('"100"');
  expect(
    fetch.mock.calls.some(([url]) => url.endsWith("?eventsBefore=85")),
  ).toBe(true);
  await waitFor(() =>
    expect(screen.getByLabelText("真实处理进度 / 关闭原因")).toHaveValue(""),
  );
  denyEarlier = true;
  fireEvent.click(screen.getByRole("button", { name: "加载更早进度" }));
  await screen.findByText("当前身份没有操作权限。");
  expect(screen.queryByText("进度记录 100")).not.toBeInTheDocument();
});
