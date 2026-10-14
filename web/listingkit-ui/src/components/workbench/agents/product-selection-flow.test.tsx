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
import { ProductSelectionFlow } from "./product-selection-flow";
import type { CatalogEntry } from "@/lib/contracts/agent-configuration";
const id = "11111111-1111-4111-8111-111111111111",
  other = "22222222-2222-4222-8222-222222222222";
const guard = vi.hoisted(() => vi.fn(() => () => {}));
vi.mock("@/components/providers/workbench-context-provider", () => ({
  useWorkbenchContext: () => ({ registerOrganizationSwitchGuard: guard }),
}));
vi.mock("../pod/collection-picker", () => ({
  CollectionPicker: ({ onSelect }: { onSelect: (i: unknown) => void }) => (
    <>
      <button
        onClick={() =>
          onSelect({
            id,
            revision: 1,
            title: "商品A",
            source: {
              productKey: "a",
              publicationId: "source-a",
              version: "1",
            },
          })
        }
      >
        选择商品A
      </button>
      <button
        onClick={() =>
          onSelect({
            id: other,
            revision: 1,
            title: "商品B",
            source: {
              productKey: "b",
              publicationId: "source-b",
              version: "1",
            },
          })
        }
      >
        选择商品B
      </button>
    </>
  ),
}));
const scope = { userId: "actor", organizationId: "org" };
const entry: CatalogEntry = {
  agent: {
    agentId: "product.selection.agent",
    activation: "ENABLED",
    revision: "1",
    activationEpoch: "1",
    defaultTemplate: null,
    updatedAt: "2026-10-14T00:00:00Z",
  },
  name: "选品对比",
  description: "",
  definitionVersion: "v1.0.0",
  parameterSchema: "product-selection-config-v1",
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
    {id: "product.selection.inspect", support: "REQUIRED", readiness: "AVAILABLE", reason: "", observedAt: "2026-10-14T00:00:00Z"},
  ],
};
let writes: Array<{ path: string; key: string; body: unknown }>;
let currentModeRevision="0";
beforeEach(() => {
  localStorage.clear();
  writes = [];
  currentModeRevision="0";
  vi.stubGlobal(
    "fetch",
    vi.fn(async (url, init) => {
      const path = String(url);
      if (path.includes("execution-modes"))
        return Response.json({
          subject: { kind: "registered_agent", id: "product.selection.agent" },
          definition: { id: "product.selection.agent", version: "v1.0.0" },
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
            products: [
              {
                ref: {
                  ItemID: id,
                  ItemRevision: "1",
                  ProductKey: "a",
                  CatalogVersion: "1",
                  PublicationID: "source-a",
                },
                sourceKind: "own",
                facts: [
                  {
                    id: "fact:" + "a".repeat(64),
                    path: "title",
                    value: "商品A",
                  },
                ],
                missing: ["sales", "landed_cost"],
              },
              {
                ref: {
                  ItemID: other,
                  ItemRevision: "1",
                  ProductKey: "b",
                  CatalogVersion: "1",
                  PublicationID: "source-b",
                },
                sourceKind: "own",
                facts: [
                  {
                    id: "fact:" + "b".repeat(64),
                    path: "title",
                    value: "商品B",
                  },
                ],
                missing: ["sales", "landed_cost"],
              },
            ],
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
  const mounted=render(
    <QueryClientProvider client={client}>
      <ProductSelectionFlow scope={scope} entry={value} nonce={0} />
    </QueryClientProvider>,
  );
  return {...mounted,refresh:(nonce:number)=>mounted.rerender(<QueryClientProvider client={client}><ProductSelectionFlow scope={scope} entry={value} nonce={nonce}/></QueryClientProvider>)};
}
it("does not offer fact inspection when Agent-use authorization is missing", async () => {
  show({...entry, capabilities: entry.capabilities.map(c => c.id === "product.selection.inspect" ? {...c,readiness:"REQUIRES_AUTHORIZATION"} : c)});
  choose(); await act(async()=>{await new Promise(resolve=>setTimeout(resolve,50))});
  expect(screen.getByRole("button",{name:"读取事实对比"})).toBeDisabled();
  fireEvent.click(screen.getByRole("button",{name:"读取事实对比"}));
  expect(writes).toHaveLength(0);
});
function choose() {
  fireEvent.click(screen.getByText("选择商品A"));
  fireEvent.click(screen.getByText("选择商品B"));
}
it.each(["REVISION_MISMATCH", "CONFIGURATION_CHANGED"])("refetches the current mode after known %s without automatically retrying analysis", async code => {
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
  show(); choose(); await viewFacts();
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
async function viewFacts(){fireEvent.click(screen.getByRole("button",{name:"读取事实对比"}));await screen.findByRole("heading",{name:"已保存原始事实"})}
it("requires a current candidate fact read before preparing analysis",async()=>{show();choose();await act(async()=>{await new Promise(resolve=>setTimeout(resolve,50))});expect(screen.getByRole("button",{name:"准备分析"})).toBeDisabled();expect(writes).toHaveLength(0);await viewFacts();expect(screen.getByRole("button",{name:"准备分析"})).toBeEnabled()});
it("requires an explicit comparison goal before preparing a paid analysis",async()=>{show();choose();await viewFacts();await waitFor(()=>expect(screen.getByRole("button",{name:"准备分析"})).toBeEnabled());fireEvent.change(screen.getByLabelText("本次比较目标"),{target:{value:""}});expect(screen.getByRole("button",{name:"准备分析"})).toBeDisabled();expect(writes).toHaveLength(1)});
it("checks the original UTF-8 goal limit and sends a trimmed goal", async () => {
  show(); choose(); await viewFacts();
  await waitFor(()=>expect(screen.getByRole("button",{name:"准备分析"})).toBeEnabled());
  for (const value of ["x".repeat(513), "中".repeat(171)]) {
    fireEvent.change(screen.getByLabelText("本次比较目标"),{target:{value}});
    expect(screen.getByRole("button",{name:"准备分析"})).toBeDisabled();
  }
  fireEvent.change(screen.getByLabelText("本次比较目标"),{target:{value:"  核对真实商品  "}});
  fireEvent.click(screen.getByRole("button",{name:"准备分析"}));
  await screen.findByRole("heading",{name:"确认本次分析"});
  expect((writes[1].body as {goalSummary:string}).goalSummary).toBe("核对真实商品");
});
it("shows an unavailable history as an error instead of an empty result", async () => {
  const baseFetch = fetch;
  vi.stubGlobal("fetch", vi.fn((url, init) => String(url).includes("/recent-runs") ? Promise.resolve(Response.json({code:"DEPENDENCY_UNAVAILABLE"},{status:503})) : baseFetch(url, init)));
  show({...entry,canReadRuns:true});
  await screen.findByRole("alert");
  expect(screen.queryByText("暂无对比历史。")).toBeNull();
  expect(writes).toHaveLength(0);
});
it("allows factual comparison without model configuration and never starts analysis automatically", async () => {
  show({
    ...entry,
    capabilities: entry.capabilities.map(c => c.id === "text.generate" ? {...c, readiness: "NEEDS_CONFIGURATION"} : c),
  });
  choose();
  fireEvent.click(screen.getByRole("button", { name: "读取事实对比" }));
  await screen.findByRole("heading", { name: "已保存原始事实" });
  expect(screen.getAllByText(/完整到岸成本/).length).toBeGreaterThan(0);
  expect(writes).toHaveLength(1);
  expect(writes[0].path).toMatch(/inspect$/);
  expect(screen.getByRole("button", { name: "准备分析" })).toBeDisabled();
});
it("prepares an exact original key and retains it through uncertain paid start", async () => {
  show();
  choose();
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
  fireEvent.click(screen.getByRole("button", { name: "核实原请求" }));
  await waitFor(() =>
    expect(screen.getByRole("button", { name: "重试原请求" })).toBeEnabled(),
  );
  expect(writes).toHaveLength(3);
  fireEvent.click(screen.getByRole("button", { name: "重试原请求" }));
  await waitFor(() => expect(writes).toHaveLength(4));
  expect(writes[3].key).toBe(writes[1].key);
});

it("keeps a changed mode out of the frozen preparation and requires a new intent",async()=>{const mounted=show();choose();await viewFacts();fireEvent.click(screen.getByRole("button",{name:"准备分析"}));await waitFor(()=>expect(screen.getByRole("button",{name:"确认生成建议"})).toBeEnabled());const original=writes[1].key;currentModeRevision="1";mounted.refresh(1);await screen.findByText(/权限模式已变化或待核对/);expect(screen.getByRole("button",{name:"确认生成建议"})).toBeDisabled();expect(writes).toHaveLength(2);expect(localStorage.getItem("product.selection.prepared:"+JSON.stringify([scope.userId,scope.organizationId]))).toContain(original)});

it("rechecks the same prepared candidates after reload without creating a new preparation",async()=>{show();choose();await viewFacts();fireEvent.click(screen.getByRole("button",{name:"准备分析"}));await screen.findByRole("heading",{name:"确认本次分析"});const original=writes[1];cleanup();show();await screen.findByRole("button",{name:"核对确认依据"});expect(screen.getByRole("button",{name:"确认生成建议"})).toBeDisabled();fireEvent.click(screen.getByRole("button",{name:"核对确认依据"}));await waitFor(()=>expect(screen.getByRole("button",{name:"确认生成建议"})).toBeEnabled());expect(writes).toHaveLength(3);expect(writes[2].path).toMatch(/inspect$/);expect(writes[2].body).toMatchObject({contextId:(original.body as {contextId:string}).contextId,candidates:(original.body as {candidates:unknown}).candidates})});

it("never shows standalone facts under a historical run that has no inspection",async()=>{
 const contextId="33333333-3333-4333-8333-333333333333",baseFetch=fetch;
 vi.stubGlobal("fetch",vi.fn((url,init)=>{const path=String(url);if(path.includes("/recent-runs"))return Promise.resolve(Response.json({items:[{runId:id,operationId:contextId,requestKey:other,productKey:"",targetPlatform:"product",phase:"stopped",revision:"1",startedAt:"2026-10-14T00:00:00Z"}],nextCursor:""}));if(path.endsWith(`/contexts/${contextId}/runs/${other}`))return Promise.resolve(Response.json({runId:id,contextId,requestKey:other,phase:"stopped",revision:"1",configurationSnapshotRef:{Kind:"agent-configuration-v1",ID:id,Digest:"a".repeat(64)},modeSnapshotRef:{Kind:"agent-execution-mode-v1",ID:other,Digest:"b".repeat(64)},executionMode:"read_only",modeRevision:"0",goalSummary:"未读取事实的历史目标",validated:false,unresolved:[],tokens:0,costMicros:0,currency:"CNY",usageStatus:"observed"}));return baseFetch(url,init)}));
 show({...entry,canReadRuns:true});choose();await viewFacts();fireEvent.click(await screen.findByRole("button",{name:/stopped/}));await screen.findByText(/未读取事实的历史目标/);expect(screen.queryByRole("heading",{name:"已保存原始事实"})).toBeNull();expect(writes).toHaveLength(1);
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
                targetPlatform: "product",
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
            inspection: {products:[{ref:{ItemID:id,ItemRevision:"1",ProductKey:"a",CatalogVersion:"1",PublicationID:"source-a"},sourceKind:"own",facts:[{id:"fact:"+"c".repeat(64),path:"title",value:"旧事实标题"}],missing:[]},{ref:{ItemID:other,ItemRevision:"1",ProductKey:"b",CatalogVersion:"1",PublicationID:"source-b"},sourceKind:"own",facts:[],missing:[]}]},
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
  fireEvent.click(await screen.findByRole("button", { name: /stopped/ }));
  await screen.findByText(/旧比较目标/);
  choose();
  fireEvent.click(screen.getByRole("button", { name: "读取事实对比" }));
  await waitFor(() => expect(screen.queryAllByText("旧事实标题")).toHaveLength(0));
  await screen.findByRole("heading", {name:"已保存原始事实"});
  fireEvent.click(screen.getByRole("button", { name: "准备分析" }));
  await screen.findByRole("heading", { name: "确认本次分析" });
  await waitFor(() =>
    expect(screen.getByRole("button", { name: "确认生成建议" })).toBeEnabled(),
  );
  expect(screen.queryByText(/旧比较目标/)).toBeNull();
  expect(writes).toHaveLength(2);
  expect(writes[1].path).toMatch(/prepare$/);
  const preparedKey = writes[1].key;
  fireEvent.click(screen.getByRole("button", {name:/stopped/}));
  await screen.findByText(/旧比较目标/);
  fireEvent.click(await screen.findByRole("button", {name:"返回待确认的对比"}));
  await waitFor(()=>expect(screen.getByRole("button", {name:"确认生成建议"})).toBeEnabled());
  expect(writes).toHaveLength(2);
  fireEvent.click(screen.getByRole("button", {name:"确认生成建议"}));
  await waitFor(()=>expect(writes).toHaveLength(3));
  expect(writes[2].key).toBe(preparedKey);
});

it("resumes a reloaded interrupted run with its persisted custom goal", async () => {
  const contextId = "33333333-3333-4333-8333-333333333333";
  const baseFetch = fetch;
  vi.stubGlobal("fetch", vi.fn((url, init) => {
    const path = String(url);
    if (path.includes("/recent-runs")) return Promise.resolve(Response.json({items: [{runId: id, operationId: contextId, requestKey: other, productKey: "", targetPlatform: "product", phase: "interrupted", revision: "3", startedAt: "2026-10-14T00:00:00Z"}], nextCursor: ""}));
    if (path.endsWith(`/contexts/${contextId}/runs/${other}`)) return Promise.resolve(Response.json({runId: id, contextId, requestKey: other, phase: "interrupted", revision: "3", configurationSnapshotRef: {Kind: "agent-configuration-v1", ID: id, Digest: "a".repeat(64)}, modeSnapshotRef: {Kind: "agent-execution-mode-v1", ID: other, Digest: "b".repeat(64)}, executionMode: "read_only", modeRevision: "0", goalSummary: "优先比较棉质候选的规格", validated: false, unresolved: [], tokens: 0, costMicros: 0, currency: "CNY", usageStatus: "observed"}));
    return baseFetch(url, init);
  }));
  show({...entry, canReadRuns: true});
  fireEvent.click(await screen.findByRole("button", {name: /interrupted/}));
  await screen.findByText(/优先比较棉质候选的规格/);
  fireEvent.click(await screen.findByRole("button", {name: "按原请求继续分析"}));
  await waitFor(() => expect(writes).toHaveLength(1));
  expect(writes[0].path).toMatch(/resume$/);
  expect(writes[0].key).toBe(other);
  expect(writes[0].body).toMatchObject({revision: "3", feedback: "优先比较棉质候选的规格"});
});
