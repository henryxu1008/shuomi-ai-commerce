import { beforeEach, afterEach, it, expect, vi } from "vitest";
import {
  render,
  screen,
  fireEvent,
  waitFor,
  cleanup,
  act,
} from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ShopDiagnosisFlow } from "./shop-diagnosis-flow";
import type { CatalogEntry } from "@/lib/contracts/agent-configuration";
const id = "11111111-1111-4111-8111-111111111111",
  other = "22222222-2222-4222-8222-222222222222";
const guard = vi.hoisted(() => vi.fn(() => () => {}));
vi.mock("@/components/providers/workbench-context-provider", () => ({
  useWorkbenchContext: () => ({ registerOrganizationSwitchGuard: guard }),
}));
const scope = { userId: "actor", organizationId: "org" };
const entry: CatalogEntry = {
  agent: {
    agentId: "shop.operations-diagnosis.agent",
    activation: "ENABLED",
    revision: "1",
    activationEpoch: "1",
    defaultTemplate: null,
    updatedAt: "2026-10-14T00:00:00Z",
  },
  name: "店铺经营诊断",
  description: "",
  definitionVersion: "v1.0.0",
  parameterSchema: "shop-operations-diagnosis-config-v1",
  canConfigure: true,
  canUse: true,
  canReadRuns: false,
  capabilities: [
    {
      id: "text.generate",
      support: "REQUIRED",
      readiness: "AVAILABLE",
      reason: "",
      observedAt: "2026-10-14T00:00:00Z",
    },
    {id: "shop.operations-diagnosis.inspect", support: "REQUIRED", readiness: "AVAILABLE", reason: "", observedAt: "2026-10-14T00:00:00Z"},
  ],
};
let writes: Array<{ path: string; key: string; body: unknown }>;
let preparedDigest: string, inspectedDigest: string;
let currentModeRevision = "0";
beforeEach(() => {
  localStorage.clear();
  writes = [];
  currentModeRevision = "0";
  preparedDigest = inspectedDigest = "c".repeat(64);
  vi.stubGlobal(
    "fetch",
    vi.fn(async (url, init) => {
      const path = String(url);
      if (path.endsWith("/operations-cockpit/capabilities"))
        return Response.json({
          access: {
            goalsRead: false,
            goalsCreate: false,
            goalsManage: false,
            storesRead: true,
            factsWrite: false,
            alertsRead: false,
            adviceRead: true,
          },
          today: "2026-10-14",
          stores: [
            {
              id,
              name: "测试店铺",
              platform: "temu",
              region: "US",
              status: "active",
            },
          ],
        });
      if (path.includes("execution-modes"))
        return Response.json({
          subject: {
            kind: "registered_agent",
            id: "shop.operations-diagnosis.agent",
          },
          definition: {
            id: "shop.operations-diagnosis.agent",
            version: "v1.0.0",
          },
          mode: "read_only",
          revision: currentModeRevision,
          policyVersion: "agent-execution-policy-v1",
          canConfigure: true,
          updatedAt: "2026-10-14T00:00:00Z",
          updatedBy: "",
        });
      if (init?.method === "POST") {
        const body = JSON.parse(init.body),
          key = new Headers(init.headers).get("Idempotency-Key")!;
        writes.push({ path, key, body });
        if (path.endsWith("/prepare"))
          return Response.json({
            contextId: body.contextId,
            requestKey: key,
            inspectionRef: {StoreID: body.storeId, StartDate: body.startDate, EndDate: body.endDate, EvidenceDigest: preparedDigest},
            configurationSnapshotRef: {
              Kind: "agent-configuration-v1",
              ID: id,
              Digest: "a".repeat(64),
            },
            modeSnapshotRef: {
              Kind: "agent-execution-mode-v1",
              ID: other,
              Digest: "b".repeat(64),
            },
            modeRevision: currentModeRevision,
            maximumTokens: 100,
            maximumCostMicros: 10000,
            maximumCallCostMicros: 10000,
            currency: "CNY",
          });
        if (path.endsWith("/inspect"))
          return Response.json({
            ref: {
              StoreID: id,
              StartDate: "2026-10-01",
              EndDate: "2026-10-07",
              EvidenceDigest: inspectedDigest,
            },
            facts: [
              {
                id: "fact:" + "a".repeat(64),
                path: "financial.revenue",
                value: "12345",
              },
              {
                id: "fact:" + "b".repeat(64),
                path: "financial.complete",
                value: "false",
              },
              {
                id: "fact:" + "c".repeat(64),
                path: "observations.orders.state",
                value: "offline_unavailable",
              },
            ],
            missing: ["financial.complete_coverage"],
          });
        throw new Error("analysis response lost");
      }
      return Response.json({ code: "NOT_FOUND" }, { status: 404 });
    }),
  );
});
afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});
function show(value = entry) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  const mounted = render(
    <QueryClientProvider client={client}>
      <ShopDiagnosisFlow scope={scope} entry={value} nonce={0} />
    </QueryClientProvider>,
  );
  return {...mounted, refresh: (nonce: number) => mounted.rerender(
    <QueryClientProvider client={client}>
      <ShopDiagnosisFlow scope={scope} entry={value} nonce={nonce} />
    </QueryClientProvider>,
  )};
}
async function choose() {
  await screen.findByRole("option", { name: /测试店铺/ });
  fireEvent.change(screen.getByLabelText("店铺"), { target: { value: id } });
  fireEvent.change(screen.getByLabelText("开始日期"), {
    target: { value: "2026-10-01" },
  });
  fireEvent.change(screen.getByLabelText("结束日期"), {
    target: { value: "2026-10-07" },
  });
}
async function viewFacts() {
  fireEvent.click(screen.getByRole("button", {name: "读取经营事实"}));
  await screen.findByRole("heading", {name: "已保存经营事实"});
}
it.each(["REVISION_MISMATCH", "CONFIGURATION_CHANGED"])("refetches the current diagnosis mode after known %s without automatically retrying analysis", async code => {
  const baseFetch = fetch;
  let reject = true;
  vi.stubGlobal("fetch", vi.fn((url, init) => {
    if (init?.method === "POST" && String(url).endsWith("/prepare") && reject) {
      reject = false;
      currentModeRevision = "1";
      return Promise.resolve(Response.json({code}, {status: 409}));
    }
    return baseFetch(url, init);
  }));
  show(); await choose(); await viewFacts();
  await waitFor(() => expect(screen.getByRole("button", {name: "准备分析"})).toBeEnabled());
  fireEvent.click(screen.getByRole("button", {name: "准备分析"}));
  await screen.findByRole("alert");
  await waitFor(() => expect(vi.mocked(fetch).mock.calls.filter(([url]) => String(url).includes("execution-modes"))).toHaveLength(2));
  expect(writes).toHaveLength(1);
  fireEvent.click(screen.getByRole("button", {name: "准备分析"}));
  await screen.findByRole("heading", {name: "确认本次分析"});
  expect((writes[1].body as {expectedModeRevision:string}).expectedModeRevision).toBe("1");
  expect(writes.every(write => !write.path.endsWith("/runs"))).toBe(true);
});
it("requires an explicit diagnosis goal before preparing a paid analysis",async()=>{show();await choose();await viewFacts();await waitFor(()=>expect(screen.getByRole("button",{name:"准备分析"})).toBeEnabled());fireEvent.change(screen.getByLabelText("本次诊断目标"),{target:{value:""}});expect(screen.getByRole("button",{name:"准备分析"})).toBeDisabled();expect(writes).toHaveLength(1);expect(writes[0].path).toMatch(/inspect$/)});
it("checks the original UTF-8 goal limit and sends a trimmed diagnosis goal", async () => {
  show(); await choose(); await viewFacts();
  await waitFor(()=>expect(screen.getByRole("button",{name:"准备分析"})).toBeEnabled());
  for (const value of ["x".repeat(513),"中".repeat(171)]) {
    fireEvent.change(screen.getByLabelText("本次诊断目标"),{target:{value}});
    expect(screen.getByRole("button",{name:"准备分析"})).toBeDisabled();
  }
  fireEvent.change(screen.getByLabelText("本次诊断目标"),{target:{value:"  核对已录入经营事实  "}});
  fireEvent.click(screen.getByRole("button",{name:"准备分析"}));
  await screen.findByRole("heading",{name:"确认本次分析"});
  expect((writes[1].body as {goalSummary:string}).goalSummary).toBe("核对已录入经营事实");
});
it("does not offer fact inspection without Agent-use authorization", async () => {
  show({...entry,capabilities:entry.capabilities.map(c => c.id === "shop.operations-diagnosis.inspect" ? {...c,readiness:"REQUIRES_AUTHORIZATION"} : c)});
  await choose(); await act(async()=>{await new Promise(resolve=>setTimeout(resolve,50))});
  expect(screen.getByRole("button",{name:"读取经营事实"})).toBeDisabled();
  fireEvent.click(screen.getByRole("button",{name:"读取经营事实"}));
  expect(writes).toHaveLength(0);
});
it("allows factual comparison without model configuration and never starts analysis automatically", async () => {
  show({
    ...entry,
    capabilities: entry.capabilities.map(c => c.id === "text.generate" ? {...c,readiness:"NEEDS_CONFIGURATION"} : c),
  });
  await choose();
  fireEvent.click(screen.getByRole("button", { name: "读取经营事实" }));
  await screen.findByRole("heading", { name: "已保存经营事实" });
  expect(screen.getByText("¥123.45")).toBeInTheDocument();
  expect(screen.getByText(/人工财务数据不足/)).toBeInTheDocument();
  expect(screen.getByText(/暂不支持离线读取/)).toBeInTheDocument();
  expect(writes).toHaveLength(1);
  expect(writes[0].path).toMatch(/inspect$/);
  expect(screen.getByRole("button", { name: "准备分析" })).toBeDisabled();
});
it("prepares an exact original key and retains it through uncertain paid start", async () => {
  show();
  await choose();
  await viewFacts();
  await waitFor(() =>
    expect(screen.getByRole("button", { name: "准备分析" })).toBeEnabled(),
  );
  fireEvent.click(screen.getByRole("button", { name: "准备分析" }));
  await screen.findByRole("heading", { name: "确认本次分析" });
  expect(writes).toHaveLength(2);
  expect((writes[1].body as { requestKey: string }).requestKey).toBe(
    writes[1].key,
  );
  fireEvent.click(screen.getByRole("button", { name: "确认生成建议" }));
  await screen.findByRole("button", { name: "核实原请求" });
  expect(writes).toHaveLength(3);
  expect(writes[2].key).toBe(writes[1].key);
  cleanup();
  show();
  await screen.findByRole("button", {name: "核实原请求"});
  fireEvent.click(screen.getByRole("button", { name: "核实原请求" }));
  await waitFor(() =>
    expect(screen.getByRole("button", { name: "重试原请求" })).toBeEnabled(),
  );
  expect(writes).toHaveLength(3);
  fireEvent.click(screen.getByRole("button", { name: "重试原请求" }));
  await waitFor(() => expect(writes).toHaveLength(4));
  expect(writes[3].key).toBe(writes[1].key);
});

it("requires viewed facts before preparing an analysis", async () => {
  show();
  await choose();
  expect(screen.getByRole("button", {name: "准备分析"})).toBeDisabled();
  expect(writes).toHaveLength(0);
});

it("resumes historical analysis with its stored goal after reload", async () => {
  const baseFetch = fetch;
  vi.stubGlobal("fetch", vi.fn((url, init) => {
    const path = String(url);
    if(path.includes("/recent-runs")) return Promise.resolve(Response.json({items:[{runId:id,operationId:id,requestKey:other,productKey:"",targetPlatform:"business",phase:"interrupted",revision:"1",startedAt:"2026-10-14T00:00:00Z"}],nextCursor:""}));
    if(init?.method === "GET" && path.endsWith(`/contexts/${id}/runs/${other}`)) return Promise.resolve(Response.json({runId:id,contextId:id,requestKey:other,phase:"interrupted",revision:"1",configurationSnapshotRef:{Kind:"agent-configuration-v1",ID:id,Digest:"a".repeat(64)},modeSnapshotRef:{Kind:"agent-execution-mode-v1",ID:other,Digest:"b".repeat(64)},executionMode:"read_only",modeRevision:"0",goalSummary:"保留的自定义诊断目标",validated:false,unresolved:[],tokens:0,costMicros:0,currency:"CNY",usageStatus:"observed"}));
    return baseFetch(url,init);
  }));
  show({...entry,canReadRuns:true});
  fireEvent.click(await screen.findByRole("button",{name:/interrupted/}));
  fireEvent.click(await screen.findByRole("button",{name:"按原请求继续分析"}));
  await waitFor(()=>expect(writes).toHaveLength(1));
  expect((writes[0].body as {feedback:string}).feedback).toBe("保留的自定义诊断目标");
  expect(writes[0].key).toBe(other);
});

it("blocks a new paid start when prepared facts changed after viewing", async () => {
  show(); await choose(); await viewFacts();
  preparedDigest = "d".repeat(64);
  fireEvent.click(screen.getByRole("button", {name: "准备分析"}));
  await screen.findByRole("heading", {name: "确认本次分析"});
  expect(screen.getByRole("button", {name: "确认生成建议"})).toBeDisabled();
  fireEvent.click(screen.getByRole("button", {name: "确认生成建议"}));
  expect(writes).toHaveLength(2);
  inspectedDigest = preparedDigest;
  fireEvent.click(screen.getByRole("button", {name: "核对确认依据"}));
  await waitFor(() => expect(screen.getByRole("button", {name: "确认生成建议"})).toBeEnabled());
  expect((writes[2].body as {contextId: string}).contextId).toBe((writes[1].body as {contextId: string}).contextId);
  expect(writes.every(w => !w.path.endsWith("/runs"))).toBe(true);
});

it("requires a new preparation when the current mode no longer matches the frozen one", async () => {
  const mounted = show(); await choose(); await viewFacts();
  fireEvent.click(screen.getByRole("button", {name: "准备分析"}));
  await waitFor(() => expect(screen.getByRole("button", {name: "确认生成建议"})).toBeEnabled());
  const key = writes[1].key;
  currentModeRevision = "1"; mounted.refresh(1);
  await screen.findByText(/权限模式已变化或待核对/);
  expect(screen.getByRole("button", {name: "确认生成建议"})).toBeDisabled();
  expect(writes).toHaveLength(2);
  expect(JSON.stringify(localStorage)).toContain(key);
});

it("reloads preparation without inventing facts and rereads the original context", async () => {
  show(); await choose(); await viewFacts();
  fireEvent.click(screen.getByRole("button", {name: "准备分析"}));
  await screen.findByRole("heading", {name: "确认本次分析"});
  cleanup(); show();
  await screen.findByRole("heading", {name: "确认本次分析"});
  expect(screen.getByRole("button", {name: "确认生成建议"})).toBeDisabled();
  fireEvent.click(screen.getByRole("button", {name: "核对确认依据"}));
  await waitFor(() => expect(screen.getByRole("button", {name: "确认生成建议"})).toBeEnabled());
  expect(writes).toHaveLength(3);
  expect((writes[2].body as {contextId: string}).contextId).toBe((writes[1].body as {contextId: string}).contextId);
});

it("switches from viewed history to the newly prepared exact comparison", async () => {
  const oldContext = "33333333-3333-4333-8333-333333333333";
  const baseFetch = fetch;
  vi.stubGlobal(
    "fetch",
    vi.fn((url, init) => {
      const path = String(url);
      if (path.includes("/recent-runs"))
        return Promise.resolve(
          Response.json({
            items: [
              {
                runId: id,
                operationId: oldContext,
                requestKey: other,
                productKey: "",
                targetPlatform: "business",
                phase: "stopped",
                revision: "1",
                startedAt: "2026-10-14T00:00:00Z",
              },
            ],
            nextCursor: "",
          }),
        );
      if (path.endsWith("/contexts/" + oldContext + "/runs/" + other))
        return Promise.resolve(
          Response.json({
            runId: id,
            contextId: oldContext,
            requestKey: other,
            phase: "stopped",
            revision: "1",
            configurationSnapshotRef: {
              Kind: "agent-configuration-v1",
              ID: id,
              Digest: "a".repeat(64),
            },
            modeSnapshotRef: {
              Kind: "agent-execution-mode-v1",
              ID: other,
              Digest: "b".repeat(64),
            },
            executionMode: "read_only",
            modeRevision: "0",
            goalSummary: "旧比较目标",
            validated: false,
            unresolved: [],
            tokens: 0,
            costMicros: 0,
            currency: "CNY",
            usageStatus: "observed",
          }),
        );
      return baseFetch(url, init);
    }),
  );
  show({ ...entry, canReadRuns: true });
  await choose(); await viewFacts();
  fireEvent.click(await screen.findByRole("button", { name: /stopped/ }));
  await screen.findByText(/旧比较目标/);
  expect(screen.queryByRole("heading", {name: "已保存经营事实"})).toBeNull();
  await choose();
  await viewFacts();
  fireEvent.click(screen.getByRole("button", { name: "准备分析" }));
  await screen.findByRole("heading", { name: "确认本次分析" });
  await waitFor(() =>
    expect(screen.getByRole("button", { name: "确认生成建议" })).toBeEnabled(),
  );
  expect(screen.queryByText(/旧比较目标/)).toBeNull();
  expect(writes).toHaveLength(3);
  expect(writes[2].path).toMatch(/prepare$/);
  const key = writes[2].key;
  fireEvent.click(screen.getByRole("button", {name:/stopped/}));
  await screen.findByText(/旧比较目标/);
  fireEvent.click(await screen.findByRole("button", {name:"返回待确认的诊断"}));
  await waitFor(()=>expect(screen.queryByText(/旧比较目标/)).toBeNull());
  const reread = screen.queryByRole("button", {name:"核对确认依据"});
  if(reread) fireEvent.click(reread);
  await waitFor(()=>expect(screen.getByRole("button", {name:"确认生成建议"})).toBeEnabled());
  expect(writes.every(w=>!w.path.endsWith("/runs"))).toBe(true);
  fireEvent.click(screen.getByRole("button", {name:"确认生成建议"}));
  await screen.findByRole("button", {name:"核实原请求"});
  expect(writes[writes.length-1].key).toBe(key);
});

it("keeps an outstanding historical run from starting a new paid diagnosis", async () => {
  const oldContext = "33333333-3333-4333-8333-333333333333";
  const baseFetch = fetch;
  vi.stubGlobal(
    "fetch",
    vi.fn((url, init) => {
      const path = String(url);
      if (path.includes("/recent-runs"))
        return Promise.resolve(
          Response.json({
            items: [
              {
                runId: id,
                operationId: oldContext,
                requestKey: other,
                productKey: "",
                targetPlatform: "business",
                phase: "running",
                revision: "1",
                startedAt: "2026-10-14T00:00:00Z",
              },
            ],
            nextCursor: "",
          }),
        );
      if (path.endsWith("/contexts/" + oldContext + "/runs/" + other))
        return Promise.resolve(
          Response.json({
            runId: id,
            contextId: oldContext,
            requestKey: other,
            phase: "running",
            revision: "1",
            configurationSnapshotRef: {
              Kind: "agent-configuration-v1",
              ID: id,
              Digest: "a".repeat(64),
            },
            modeSnapshotRef: {
              Kind: "agent-execution-mode-v1",
              ID: other,
              Digest: "b".repeat(64),
            },
            executionMode: "read_only",
            modeRevision: "0",
            goalSummary: "旧比较目标",
            validated: false,
            unresolved: [],
            tokens: 0,
            costMicros: 0,
            currency: "CNY",
            usageStatus: "unknown_reserved",
          }),
        );
      return baseFetch(url, init);
    }),
  );
  show({ ...entry, canReadRuns: true });
  fireEvent.click(await screen.findByRole("button", { name: /running/ }));
  await screen.findByText(/旧比较目标/);
  await choose();
  expect(screen.getByRole("button", {name:"准备分析"})).toBeDisabled();
  expect(writes).toHaveLength(0);
});
