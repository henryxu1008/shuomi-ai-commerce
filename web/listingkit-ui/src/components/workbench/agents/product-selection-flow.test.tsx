import { beforeEach, afterEach, it, expect, vi } from "vitest";
import {
  render,
  screen,
  fireEvent,
  waitFor,
  cleanup,
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
  ],
};
let writes: Array<{ path: string; key: string; body: unknown }>;
beforeEach(() => {
  localStorage.clear();
  writes = [];
  vi.stubGlobal(
    "fetch",
    vi.fn(async (url, init) => {
      const path = String(url);
      if (path.includes("execution-modes"))
        return Response.json({
          subject: { kind: "registered_agent", id: "product.selection.agent" },
          definition: { id: "product.selection.agent", version: "v1.0.0" },
          mode: "read_only",
          revision: "0",
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
            modeRevision: "0",
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
  return render(
    <QueryClientProvider client={client}>
      <ProductSelectionFlow scope={scope} entry={value} nonce={0} />
    </QueryClientProvider>,
  );
}
function choose() {
  fireEvent.click(screen.getByText("选择商品A"));
  fireEvent.click(screen.getByText("选择商品B"));
}
it("allows factual comparison without model configuration and never starts analysis automatically", async () => {
  show({
    ...entry,
    capabilities: [
      { ...entry.capabilities[0], readiness: "NEEDS_CONFIGURATION" },
    ],
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
  await waitFor(() =>
    expect(screen.getByRole("button", { name: "准备分析" })).toBeEnabled(),
  );
  fireEvent.click(screen.getByRole("button", { name: "准备分析" }));
  await screen.findByRole("heading", { name: "确认本次分析" });
  expect(writes).toHaveLength(1);
  expect((writes[0].body as { requestKey: string }).requestKey).toBe(
    writes[0].key,
  );
  fireEvent.click(screen.getByRole("button", { name: "确认生成建议" }));
  await screen.findByRole("button", { name: "核实原请求" });
  expect(writes).toHaveLength(2);
  expect(writes[1].key).toBe(writes[0].key);
  fireEvent.click(screen.getByRole("button", { name: "核实原请求" }));
  await waitFor(() =>
    expect(screen.getByRole("button", { name: "重试原请求" })).toBeEnabled(),
  );
  expect(writes).toHaveLength(2);
  fireEvent.click(screen.getByRole("button", { name: "重试原请求" }));
  await waitFor(() => expect(writes).toHaveLength(3));
  expect(writes[2].key).toBe(writes[0].key);
});
