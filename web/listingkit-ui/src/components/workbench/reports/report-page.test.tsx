import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, expect, it, vi } from "vitest";
import { ReportPage } from "./report-page";
import { fetchSheinRecords } from "@/lib/api/shein-records-client";
import { type Report } from "@/lib/contracts/report-center";
import { readReportIntent, writeReportIntent } from "@/lib/api/report-center";
const fixture = vi.hoisted(() => ({ context: { user: { id: "actor-a" }, effectiveOrganization: { id: "org-a" }, roles: ["listingkit_operator"], permissions: ["workbench.report.read", "workbench.report.manage"], isSwitching: false, isLoading: false, selectionRequired: false, error: null, blockingError: null, aiWorkbenchAvailable: false }, sourceList: vi.fn() }));
vi.mock("@/components/providers/workbench-context-provider", () => ({ useWorkbenchContext: () => fixture.context }));
vi.mock("@/lib/api/product-title-review-client", () => ({ fetchProductTitleProposals: fixture.sourceList }));
vi.mock("@/lib/api/shein-records-client", () => ({ fetchSheinRecords: vi.fn() }));
const source = { ref: { kind: "TITLE_REVIEW" as const, id: "550e8400-e29b-41d4-a716-446655440000", version: "2:accepted" }, title: "标题审核 · 商品A", productKey: "product-a", storeId: "" };
const report: Report = { ...source, id: "550e8400-e29b-41d4-a716-446655440001", capturedAt: "2026-10-10T00:00:00Z", favorite: false, content: { schemaVersion: 1, sections: [{ title: "标题历史", fields: [{ label: "建议标题", value: "已保存的标题内容" }] }] }, digest: "a".repeat(64) };
beforeEach(() => { vi.unstubAllGlobals(); vi.restoreAllMocks(); sessionStorage.clear(); fixture.context.user = { id: "actor-a" }; fixture.context.effectiveOrganization = { id: "org-a" }; fixture.context.permissions = ["workbench.report.read", "workbench.report.manage"]; fixture.sourceList.mockResolvedValue({ items: [{ proposal_id: source.ref.id }], next_cursor: null }); });
const tree = (client: QueryClient, titleReviewAvailable = true) => <QueryClientProvider client={client}><ReportPage view="all" titleReviewAvailable={titleReviewAvailable} /></QueryClientProvider>;
it("does not suggest adding reports to a read-only user", async () => {
  fixture.context.permissions = ["workbench.report.read"];
  vi.stubGlobal("fetch", vi.fn(async (path: string) => path.endsWith("/summary") ? Response.json({ saved: 0, recent: 0, favorites: 0, stores: 0 }) : Response.json({ items: [], nextCursor: "" })));
  const client = new QueryClient(), view = render(tree(client));
  expect(await screen.findByText("暂无可查看的报告。")).toBeVisible();
  expect(screen.queryByRole("button", {name:"添加已有结果"})).not.toBeInTheDocument();
  view.unmount(); client.clear();
});
it("does not suggest favoriting reports to a read-only user", async () => {
  fixture.context.permissions = ["workbench.report.read"];
  vi.stubGlobal("fetch", vi.fn(async (path: string) => path.endsWith("/summary") ? Response.json({ saved: 0, recent: 0, favorites: 0, stores: 0 }) : Response.json({ items: [], nextCursor: "" })));
  const client = new QueryClient(), view = render(<QueryClientProvider client={client}><ReportPage view="favorites" titleReviewAvailable /></QueryClientProvider>);
  expect(await screen.findByText("暂无可查看的收藏报告。")).toBeVisible();
  view.unmount(); client.clear();
});
it("explains an empty search and clears the applied filters", async () => {
  vi.stubGlobal("fetch", vi.fn(async (path: string) => path.endsWith("/summary") ? Response.json({ saved: 0, recent: 0, favorites: 0, stores: 0 }) : Response.json({ items: [], nextCursor: "" })));
  const client = new QueryClient(), view = render(tree(client));
  await screen.findByText("当前范围暂无报告");
  fireEvent.click(within(screen.getByRole("group", {name:"报告用途"})).getByRole("button", {name:"标题审核"}));
  fireEvent.change(screen.getByRole("textbox", {name:"报告标题或商品关键词"}), {target:{value:"不存在的报告"}});
  fireEvent.click(screen.getByRole("button", {name:"搜索"}));
  await screen.findByText("没有找到符合条件的报告");
  fireEvent.click(screen.getByRole("button", {name:"清除筛选"}));
  expect(screen.getByRole("textbox", {name:"报告标题或商品关键词"})).toHaveValue("");
  expect(within(screen.getByRole("group", {name:"报告用途"})).getByRole("button", {name:"全部"})).toHaveAttribute("aria-pressed", "true");
  await screen.findByText("当前范围暂无报告");
  view.unmount(); client.clear();
});
it.each(["overview", "recent", "favorites", "all"] as const)("exposes report views and marks %s current", view => {
  fixture.context.permissions = [];
  const client=new QueryClient();
  const rendered=render(<QueryClientProvider client={client}><ReportPage view={view} /></QueryClientProvider>);
  const nav=screen.getByRole("navigation",{name:"报告视图"});
  for(const [mode,label,path] of [["overview","报告概览",""],["recent","最近报告","/recent"],["favorites","收藏报告","/favorites"],["all","全部报告","/all"]]){
    const link=within(nav).getByRole("link",{name:label});
    expect(link).toHaveAttribute("href",`/workbench/ai/reports${path}`);
    if(mode===view) expect(link).toHaveAttribute("aria-current","page");
    else expect(link).not.toHaveAttribute("aria-current");
  }
  expect(screen.getByText("报告尚不可用")).toBeVisible();
  rendered.unmount();client.clear();
});
it("previews a native exact record without Review or legacy readers and saves only on explicit confirmation", async () => {
  fixture.context.permissions.push("workbench.supply.read");
  const native = { ...source, title: "SHEIN 已保存资料", ref: { kind: "SHEIN_TARGET_RECORD" as const, id: source.ref.id, version: `sha256:${"b".repeat(64)}` } };
  const posts: unknown[] = [];
  const fetch = vi.fn(async (path: string, init?: RequestInit) => {
    if (init?.method === "POST") { posts.push(JSON.parse(String(init.body))); throw new Error("response lost"); }
    if (path.includes("/sources/")) return Response.json(native);
    return path.endsWith("/summary") ? Response.json({ saved: 0, recent: 0, favorites: 0, stores: 0 }) : Response.json({ items: [], nextCursor: "" });
  });
  vi.stubGlobal("fetch", fetch);
  const client = new QueryClient(), view = render(<QueryClientProvider client={client}><ReportPage view="all" sheinTargetSourcesAvailable initialTargetRecord={native.ref.id} /></QueryClientProvider>);
  const button = await screen.findByRole("button", { name: "保存此版本" });
  await waitFor(() => expect(button).toBeEnabled());
  expect(posts).toHaveLength(0); expect(fixture.sourceList).not.toHaveBeenCalled(); expect(fetchSheinRecords).not.toHaveBeenCalled();
  expect(fetch.mock.calls.some(([path]) => path.includes(`/sources/SHEIN_TARGET_RECORD/${native.ref.id}`))).toBe(true);
  fireEvent.click(button);
  await screen.findByText(/操作结果尚未确认/);
  expect(posts).toEqual([{ source: native.ref }]);
  expect(readReportIntent({ userId: "actor-a", organizationId: "org-a" })).toMatchObject({ source: native.ref });
  fixture.context.permissions = ["workbench.report.read"];
  view.rerender(<QueryClientProvider client={client}><ReportPage view="all" sheinTargetSourcesAvailable initialTargetRecord={native.ref.id} /></QueryClientProvider>);
  expect(screen.queryByRole("button", { name: "保存此版本" })).not.toBeInTheDocument();
  view.unmount(); client.clear();
});
it("drops a native preview when source admission closes while title capture stays available", async () => {
  fixture.context.permissions.push("workbench.supply.read");
  const native = { ...source, title: "先前资料来源", ref: { kind: "SHEIN_TARGET_RECORD" as const, id: source.ref.id, version: `sha256:${"b".repeat(64)}` } };
  vi.stubGlobal("fetch", vi.fn(async (path: string) => path.includes("/sources/") ? Response.json(native) : path.endsWith("/summary") ? Response.json({ saved: 0, recent: 0, favorites: 0, stores: 0 }) : Response.json({ items: [], nextCursor: "" })));
  const client = new QueryClient();
  const page = (available: boolean) => <QueryClientProvider client={client}><ReportPage titleReviewAvailable sheinTargetSourcesAvailable={available} initialTargetRecord={native.ref.id} /></QueryClientProvider>;
  const view = render(page(true));
  await screen.findByRole("button", { name: "保存此版本" });
  view.rerender(page(false));
  expect(screen.queryByText(native.title)).not.toBeInTheDocument();
  expect(screen.queryByRole("button", { name: "保存此版本" })).not.toBeInTheDocument();
  view.unmount(); client.clear();
});
it("clears pending component state when valid colon scopes switch", async () => {
  fixture.context.user = { id: "a:b" }; fixture.context.effectiveOrganization = { id: "c" };
  vi.stubGlobal("fetch", vi.fn(async (path: string, init?: RequestInit) => {
    if (init?.method === "POST") throw new Error("lost");
    if (path.includes("/sources/")) return Response.json(source);
    return path.endsWith("/summary") ? Response.json({ saved: 0, recent: 0, favorites: 0, stores: 0 }) : Response.json({ items: [], nextCursor: "" });
  }));
  const client = new QueryClient(), view = render(tree(client));
  await waitFor(() => expect(screen.getByRole("button", { name: "添加已有结果" })).toBeEnabled());
  fireEvent.click(screen.getByRole("button", { name: "添加已有结果" }));
  fireEvent.click(await screen.findByRole("button", { name: "保存此版本" }));
  await screen.findByText(/操作结果尚未确认/);
  fixture.context.user = { id: "a" }; fixture.context.effectiveOrganization = { id: "b:c" };
  view.rerender(tree(client));
  await waitFor(() => expect(screen.getByRole("button", { name: "添加已有结果" })).toBeEnabled());
  expect(screen.queryByRole("button", { name: "重试上次操作" })).not.toBeInTheDocument();
  view.unmount(); client.clear();
});
it("disables unmounted title capture while saved reports stay readable and downloadable", async () => {
  const { content, digest, ...summary } = report; void content; void digest;
  let favorite = false;
  const fetch = vi.fn(async (path: string, init?: RequestInit) => {
    if (init?.method === "POST") { favorite = true; return Response.json({ commandId: new Headers(init.headers).get("Idempotency-Key"), report: { ...report, favorite }, replayed: false }); }
    return path.endsWith("/summary") ? Response.json({ saved: 1, recent: 1, favorites: favorite ? 1 : 0, stores: 0 }) : path.endsWith(report.id) ? Response.json({ ...report, favorite }) : Response.json({ items: [{ ...summary, favorite }], nextCursor: "" });
  });
  vi.stubGlobal("fetch", fetch);
  fixture.sourceList.mockClear();
  const client = new QueryClient(), view = render(tree(client, false));
  fireEvent.click(await screen.findByRole("button", { name: source.title }));
  await screen.findByText("已保存的标题内容");
  expect(screen.getByRole("button", { name: "添加已有结果" })).toBeDisabled();
  expect(screen.getByText(/标题审核报告保存暂未开放/)).toBeInTheDocument();
  fireEvent.click(screen.getByRole("button", { name: "添加已有结果" }));
  expect(screen.queryByRole("button", { name: "已有审核详情" })).not.toBeInTheDocument();
  expect(fixture.sourceList).not.toHaveBeenCalled();
  expect(fetch.mock.calls.some(([path]) => path.includes("/sources/"))).toBe(false);
  fireEvent.click(screen.getByText("来源与技术信息"));
  expect(screen.getByRole("button", { name: "下载原始数据" })).toBeEnabled();
  await waitFor(() => expect(screen.getByRole("button", { name: "收藏报告" })).toBeEnabled());
  fireEvent.click(screen.getByRole("button", { name: "收藏报告" }));
  await screen.findByRole("button", { name: "取消收藏" });
  view.unmount(); client.clear();
});
it("clearly disables unmounted SHEIN capture without fetching it and still reads saved SHEIN reports", async () => {
  const sheinReport = { ...report, ref: { ...report.ref, kind: "SHEIN_RECORD", version: `sha256:${"a".repeat(64)}` } };
  const { content, digest, ...summary } = sheinReport; void content; void digest;
  vi.stubGlobal("fetch", vi.fn(async (path: string) => path.endsWith("/summary") ? Response.json({ saved: 1, recent: 1, favorites: 0, stores: 0 }) : path.endsWith(report.id) ? Response.json(sheinReport) : Response.json({ items: [summary], nextCursor: "" })));
  fixture.sourceList.mockResolvedValue({ items: [], next_cursor: null });
  const client = new QueryClient(), view = render(tree(client));
  fireEvent.click(await screen.findByRole("button", { name: source.title }));
  await screen.findByText("已保存的标题内容");
  fireEvent.click(screen.getByRole("button", { name: "添加已有结果" }));
  expect(within(screen.getByRole("group",{name:"保存报告用途"})).getByRole("button", { name: "商品资料" })).toBeDisabled();
  expect(screen.getByText("已保存的标题内容")).toBeVisible();
  expect(fetchSheinRecords).not.toHaveBeenCalled();
  fireEvent.click(screen.getByText("来源与技术信息"));
  expect(screen.getByRole("button", { name: "下载原始数据" })).toBeEnabled();
  view.unmount(); client.clear();
});
it.each(["applied", "rejected"])("saves a %s Review from its existing detail link after it leaves the actionable collection", async state => {
  fixture.sourceList.mockResolvedValue({ items: [], next_cursor: null });
  const terminal = { ...source, ref: { ...source.ref, version: `2:${state}` } }, terminalReport = { ...report, ...terminal };
  const bodies: unknown[] = [];
  vi.stubGlobal("fetch", vi.fn(async (path: string, init: RequestInit) => {
    if (init.method === "POST") { bodies.push(JSON.parse(String(init.body))); return Response.json({ commandId: new Headers(init.headers).get("Idempotency-Key"), report: terminalReport, replayed: false }); }
    if (path.includes("/sources/")) return path.endsWith(source.ref.id) ? Response.json(terminal) : Response.json({ code: "NOT_FOUND" }, { status: 404 });
    if (path.endsWith("/summary")) return Response.json({ saved: 0, recent: 0, favorites: 0, stores: 0 });
    if (path.endsWith(report.id)) return Response.json(terminalReport);
    return Response.json({ items: [], nextCursor: "" });
  }));
  const client = new QueryClient(), view = render(tree(client));
  await screen.findByText("当前范围暂无报告");
  await waitFor(() => expect(screen.getByRole("button", { name: "添加已有结果" })).toBeEnabled());
  fireEvent.click(await screen.findByRole("button", { name: "添加已有结果" }));
  fireEvent.click(screen.getByRole("button", { name: "已有审核详情" }));
  const input = screen.getByLabelText("标题审核详情链接或 ID");
  fireEvent.change(input, { target: { value: "/workbench/ai/tasks/pending/other?proposal_id=550e8400-e29b-41d4-a716-446655440099" } });
  fireEvent.click(screen.getByRole("button", { name: "读取当前详情" }));
  await screen.findByText("来源读取失败"); expect(screen.queryByRole("button", { name: "保存此版本" })).not.toBeInTheDocument();
  fireEvent.change(input, { target: { value: `/workbench/ai/tasks/pending/other?proposal_id=${source.ref.id}` } });
  fireEvent.click(screen.getByRole("button", { name: "读取当前详情" }));
  await screen.findByText(`版本 2:${state}`); expect(bodies).toHaveLength(0);
  fireEvent.click(screen.getByRole("button", { name: "保存此版本" }));
  await within(await screen.findByRole("region", { name: "报告详情" })).findByText(`2:${state}`);
  expect(bodies).toEqual([{ source: terminal.ref }]); view.unmount(); client.clear();
});
it("manually saves an exact version, reads it, favorites it and downloads the same snapshot", async () => {
  let saved = false, favorite = false; const bodies: { path: string; key: string; body: object }[] = [];
  vi.stubGlobal("fetch", vi.fn(async (path: string, init: RequestInit) => {
    if (init.method === "POST") { const body = JSON.parse(String(init.body)); const key = new Headers(init.headers).get("Idempotency-Key")!; bodies.push({ path, key, body }); if (path.endsWith("/favorite")) favorite = body.favorite; else saved = true; return Response.json({ commandId: key, report: { ...report, favorite }, replayed: false }); }
    if (path.includes("/sources/")) return Response.json(source);
    if (path.endsWith("/summary")) return Response.json({ saved: saved ? 1 : 0, recent: saved ? 1 : 0, favorites: favorite ? 1 : 0, stores: 0 });
    if (path.endsWith(report.id)) return Response.json({ ...report, favorite });
    const { content, digest, ...summary } = { ...report, favorite }; void content; void digest; return Response.json({ items: saved ? [summary] : [], nextCursor: "" });
  }));
  const create = vi.fn<(blob: Blob) => string>(() => "blob:report"); Object.defineProperty(URL, "createObjectURL", { configurable: true, value: create }); Object.defineProperty(URL, "revokeObjectURL", { configurable: true, value: vi.fn() }); vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {});
  const client = new QueryClient(), view = render(tree(client));
  await screen.findByText("当前范围暂无报告"); fireEvent.click(screen.getByRole("button", { name: "添加已有结果" })); fireEvent.click(await screen.findByRole("button", { name: "保存此版本" }));
  const region = await screen.findByRole("region", { name: "报告详情" }); await within(region).findByText("已保存的标题内容"); expect(within(region).getByText("2:accepted")).toBeInTheDocument(); expect(bodies[0].body).toEqual({ source: source.ref });
  fireEvent.click(within(region).getByRole("button", { name: "收藏报告" })); await within(region).findByRole("button", { name: "取消收藏" }); expect(bodies[1].body).toEqual({ favorite: true }); expect(bodies[1].key).not.toBe(bodies[0].key);
  fireEvent.click(within(region).getByText("来源与技术信息")); fireEvent.click(within(region).getByRole("button", { name: "下载原始数据" })); fireEvent.click(within(region).getByRole("button", { name: "下载文本" })); expect(create).toHaveBeenCalledTimes(2); expect((create.mock.calls[0][0] as Blob).type).toBe("application/json;charset=utf-8");
  expect(within(region).getByRole("link", { name: "打开原来源" })).toHaveAttribute("href", `/workbench/ai/tasks/pending/other?proposal_id=${source.ref.id}`); expect(sessionStorage.length).toBe(0); view.unmount(); client.clear();
});
it("hides historical content immediately when enterprise or report permission changes", async () => {
  const { content, digest, ...summary } = report; void content; void digest;
  vi.stubGlobal("fetch", vi.fn(async (path: string, init: RequestInit) => {
    const org = new Headers(init.headers).get("X-Expected-Organization-ID"); if (path.endsWith("/summary")) return Response.json({ saved: 1, recent: 1, favorites: 0, stores: 0 }); if (path.endsWith(report.id)) return Response.json(report); return Response.json({ items: org === "org-a" ? [summary] : [], nextCursor: "" });
  }));
  const client = new QueryClient(), view = render(tree(client)); fireEvent.click(await screen.findByRole("button", { name: source.title })); await screen.findByText("已保存的标题内容");
  fixture.context.effectiveOrganization = { id: "org-b" }; view.rerender(tree(client)); expect(screen.queryByText("已保存的标题内容")).not.toBeInTheDocument(); await screen.findByText("当前范围暂无报告");
  fixture.context.permissions = []; view.rerender(tree(client)); expect(screen.getByText("报告尚不可用")).toBeInTheDocument(); expect(screen.queryByRole("button", { name: "添加已有结果" })).not.toBeInTheDocument(); view.unmount(); client.clear();
});
it("recovers a saved unknown intent with its original key after remount", async () => {
  const key = "550e8400-e29b-41d4-a716-446655440002"; const intent = { operation: "save" as const, key, source: source.ref }; const scope = { userId: "actor-a", organizationId: "org-a" }; writeReportIntent(scope, intent);
  const fetch = vi.fn(async (path: string, init: RequestInit) => { if (init.method === "POST") throw new Error("lost"); if (path.endsWith("/summary")) return Response.json({ saved: 0, recent: 0, favorites: 0, stores: 0 }); return Response.json({ items: [], nextCursor: "" }); }); vi.stubGlobal("fetch", fetch);
  const client = new QueryClient(), view = render(tree(client)); fireEvent.click(await screen.findByRole("button", { name: "重试上次操作" })); await screen.findByText(/操作结果尚未确认/); expect(screen.getByRole("button", { name: "添加已有结果" })).toBeDisabled();
  await waitFor(() => expect(fetch.mock.calls.some(([,init]) => init.method === "POST")).toBe(true)); const posted = fetch.mock.calls.find(([,init]) => init.method === "POST")![1]; expect(new Headers(posted.headers).get("Idempotency-Key")).toBe(key); expect(JSON.parse(String(posted.body))).toEqual({ source: source.ref }); expect(readReportIntent(scope)).toEqual(intent); view.unmount(); client.clear();
});

it("previews the exact saved offline source and keeps an explicit save intent on response loss",async()=>{
 fixture.context.permissions.push("listingkit.admin.read");
 const saved={...source,title:"保存时离线诊断",ref:{kind:"SHEIN_RECORD" as const,id:source.ref.id,version:`sha256:${"c".repeat(64)}`}};
 const posts:unknown[]=[];const fetch=vi.fn(async(path:string,init?:RequestInit)=>{if(init?.method==="POST"){posts.push(JSON.parse(String(init.body)));throw new Error("lost")};if(path.includes("/sources/"))return Response.json(saved);return path.endsWith("/summary")?Response.json({saved:0,recent:0,favorites:0,stores:0}):Response.json({items:[],nextCursor:""})});vi.stubGlobal("fetch",fetch);
 const client=new QueryClient(),view=render(<QueryClientProvider client={client}><ReportPage view="all" sheinOfflineSourcesAvailable initialOfflineRecord={saved.ref.id}/></QueryClientProvider>);
 const button=await screen.findByRole("button",{name:"保存此版本"});await waitFor(()=>expect(button).toBeEnabled());expect(posts).toHaveLength(0);expect(fetchSheinRecords).not.toHaveBeenCalled();
 fireEvent.click(button);await screen.findByText(/操作结果尚未确认/);expect(posts).toEqual([{source:saved.ref}]);expect(readReportIntent({userId:"actor-a",organizationId:"org-a"})).toMatchObject({source:saved.ref});view.unmount();client.clear();
});

it("filters product reports from both SHEIN sources by purpose and platform",async()=>{
 const target={...report,id:"550e8400-e29b-41d4-a716-446655440002",title:"商品资料 A",ref:{...report.ref,kind:"SHEIN_TARGET_RECORD",version:`sha256:${"b".repeat(64)}`}};
 const offline={...report,id:"550e8400-e29b-41d4-a716-446655440003",title:"商品资料 B",ref:{...report.ref,kind:"SHEIN_RECORD",version:`sha256:${"c".repeat(64)}`}};
 const summary=(r:typeof report)=>{const {content,digest,...rest}=r;void content;void digest;return rest;};
 const fetch=vi.fn(async(path:string)=>{if(path.endsWith("/summary"))return Response.json({saved:3,recent:3,favorites:0,stores:0});const url=new URL(path,"https://example.test");const kind=url.searchParams.get("kind");const items=[report,target,offline].filter(r=>!kind||r.ref.kind===kind).map(r=>summary(r as typeof report));return Response.json({items,nextCursor:""});});
 vi.stubGlobal("fetch",fetch);const client=new QueryClient(),view=render(tree(client));
 await screen.findByRole("button",{name:"标题审核 · 商品A"});fireEvent.click(screen.getByRole("button",{name:"商品资料"}));
 expect(await screen.findByRole("button",{name:"商品资料 A"})).toBeVisible();expect(screen.getByRole("button",{name:"商品资料 B"})).toBeVisible();
 expect(screen.queryByRole("button",{name:"标题审核 · 商品A"})).not.toBeInTheDocument();
 expect(screen.getByRole("button",{name:"SHEIN"})).toHaveAttribute("aria-pressed","true");
 expect(screen.queryByRole("button",{name:/TEMU|TikTok/i})).not.toBeInTheDocument();
 expect(fetch.mock.calls.some(([p])=>p.includes("kind=SHEIN_RECORD"))).toBe(true);
 expect(fetch.mock.calls.some(([p])=>p.includes("kind=SHEIN_TARGET_RECORD"))).toBe(true);view.unmount();client.clear();
});
it("keeps technical details collapsed while preserving the business content",async()=>{
 const {content,digest,...summary}=report;void content;void digest;
 vi.stubGlobal("fetch",vi.fn(async(path:string)=>path.endsWith("/summary")?Response.json({saved:1,recent:1,favorites:0,stores:0}):path.endsWith(report.id)?Response.json(report):Response.json({items:[summary],nextCursor:""})));
 const client=new QueryClient(),view=render(tree(client));fireEvent.click(await screen.findByRole("button",{name:source.title}));await screen.findByText("已保存的标题内容");
 const details=screen.getByText("来源与技术信息").closest("details")!;expect(details).not.toHaveAttribute("open");expect(within(details).getByText(report.digest)).toBeInTheDocument();view.unmount();client.clear();
});

it("shows saved product issues without exposing source identifiers by default",async()=>{
 const target:Report={...report,title:"测试商品资料",ref:{...report.ref,kind:"SHEIN_TARGET_RECORD",version:`sha256:${"b".repeat(64)}`},content:{schemaVersion:1,sections:[{title:"保存时资料身份",fields:[{label:"资料记录 ID",value:"record-private-id"},{label:"站点",value:"US"}]},{title:"保存时构建结果（不代表当前上传许可或离线诊断）",fields:[{label:"保存时 ReadyForUpload",value:"false"},{label:"brand · required",value:"请填写品牌"}]}]}};
 const {content,digest,...summary}=target;void content;void digest;
 vi.stubGlobal("fetch",vi.fn(async(path:string)=>path.endsWith("/summary")?Response.json({saved:1,recent:1,favorites:0,stores:0}):path.endsWith(target.id)?Response.json(target):Response.json({items:[summary],nextCursor:""})));
 const client=new QueryClient(),view=render(tree(client));fireEvent.click(await screen.findByRole("button",{name:target.title}));
 expect((await screen.findAllByText("请填写品牌"))[0]).toBeVisible();expect(screen.getByText("待处理项")).toBeVisible();
 expect(screen.getByText("record-private-id")).not.toBeVisible();expect(screen.getByText("brand · required")).not.toBeVisible();
 fireEvent.click(screen.getByText("来源与技术信息"));expect(screen.getByText("record-private-id")).toBeVisible();view.unmount();client.clear();
});

it("does not suggest saving product reports when their source is unavailable", async()=>{
 vi.stubGlobal("fetch",vi.fn(async(path:string)=>path.endsWith("/summary")?Response.json({saved:0,recent:0,favorites:0,stores:0}):Response.json({items:[],nextCursor:""})));
 const client=new QueryClient(),view=render(tree(client));fireEvent.click(await screen.findByRole("button",{name:"商品资料"}));
 expect(await screen.findByText("当前无法保存商品资料报告，已有历史报告仍可查看。")).toBeVisible();view.unmount();client.clear();
});
