import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { ProductTitleProposal, ProductTitleProposalList } from "@/lib/api/product-title-review";
import { ProductTitleReviewError } from "@/lib/api/product-title-review-client";
import { titleProposalFixture } from "@/test/product-title-review-fixture";
import { contentProposalFixture } from "@/test/product-content-review-fixture";
import { parseProductTitleProposal } from "@/lib/api/product-title-review";
import { PendingTitleReviewPageContent } from "./pending-title-review-page";

const calls = vi.hoisted(() => ({ list: vi.fn(), get: vi.fn(), decide: vi.fn(), apply: vi.fn(), context: {} as Record<string, unknown> }));
vi.mock("@/components/providers/workbench-context-provider", () => ({ useWorkbenchContext: () => calls.context }));
vi.mock("@/lib/api/product-title-review-client", async (original) => ({ ...await original<object>(), fetchProductTitleProposals: calls.list, fetchProductTitleProposal: calls.get, decideProductTitleProposal: calls.decide, applyProductTitleProposal: calls.apply }));
let client: QueryClient;
const proposal = (state: ProductTitleProposal["state"] = "pending", revision = "1"): ProductTitleProposal => ({ ...titleProposalFixture(), state, revision, input: { product_key: "product", base_version: "9007199254740993" } });
const collection = (): ProductTitleProposalList => ({ schema_version: 1, coverage: "product-title-proposals-only", items: [{ proposal_id: proposal().proposal_id, product_key: "product", base_version: "9007199254740993", proposal_revision: "1", state: "pending" }], next_cursor: null });
const tree = (available = true, initialProposalId?: string) => <QueryClientProvider client={client}><PendingTitleReviewPageContent available={available} initialProposalId={initialProposalId} /></QueryClientProvider>;
const failure = (code: string, outcome: "unknown" | "rejected" = "rejected", status = 409) => new ProductTitleReviewError(status, code, { code, message: "private raw details", requestId: "", fieldErrors: [] }, outcome);
function deferred<T>() { let resolve!: (value: T) => void, reject!: (error: unknown) => void; const promise = new Promise<T>((r, j) => { resolve = r; reject = j; }); return { promise, resolve, reject }; }
beforeEach(() => {
  client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  calls.context = { user: { id: "owner" }, effectiveOrganization: { id: "200" }, roles: ["listingkit_admin"], permissions: ["listingkit.admin.read","listingkit.admin.write","product_sourcing.write","local_agent.write","listingkit.image_agent.read","listingkit.image_agent.write","workbench.agent.read","workbench.agent.use","workbench.agent.configure","workbench.chat.read","workbench.chat.use","workbench.task.read","workbench.knowledge.read","workbench.knowledge.manage","workbench.store.read","workbench.store.create","workbench.store.update","workbench.store.lifecycle","workbench.store.delete","workbench.source_account.read","workbench.source_account.manage","workbench.organization_member.read","workbench.organization_member.manage","workbench.commercial.read","workbench.commercial.purchase","workbench.commercial.wallet_topup"], retry: vi.fn() };
  calls.list.mockReset().mockResolvedValue(collection()); calls.get.mockReset().mockResolvedValue(proposal());
  calls.decide.mockReset().mockResolvedValue(proposal("accepted", "2")); calls.apply.mockReset();
  Object.defineProperty(HTMLDialogElement.prototype, "showModal", { configurable: true, value: function (this: HTMLDialogElement) { this.open = true; } });
  Object.defineProperty(HTMLDialogElement.prototype, "close", { configurable: true, value: function (this: HTMLDialogElement) { this.open = false; } });
});
it.each(["read_only",undefined] as const)("keeps title execution suggestions readable without writes for frozen mode %s",async(mode)=>{
 const {ContentSelection:unused,...execution}=contentProposalFixture().execution;void unused;
 const frozen={...titleProposalFixture(),execution:{...execution,binding:{...execution.binding,TargetPlatform:"shein"}},...(mode?{execution_mode:{mode,revision:"1",snapshot:execution.mode}}:{})};
 calls.get.mockResolvedValue(frozen);await open();
 expect(screen.getByText("Original title")).toBeVisible();expect(screen.queryByRole("button",{name:"接受提案"})).not.toBeInTheDocument();
 expect(screen.queryByRole("button",{name:"编辑标题"})).not.toBeInTheDocument();expect(calls.decide).not.toHaveBeenCalled();
 expect(()=>parseProductTitleProposal(frozen)).not.toThrow();
});
it("rereads access after an action denial while preserving an authorized suggestion",async()=>{
 calls.decide.mockRejectedValue(failure("permission_denied","rejected",403));await open();await userEvent.click(screen.getByRole("button",{name:"接受提案"}));
 expect(await screen.findByText("当前身份没有标题审核权限")).toBeVisible();expect(screen.getByText("Original title")).toBeVisible();
});
afterEach(() => { cleanup(); client.clear(); });
async function open() { render(tree()); await userEvent.click(await screen.findByRole("button", { name: /product.*查看详情/ })); await screen.findByText("Original title"); }

it("discovers the real collection, GETs only the selected proposal and keeps exact versions", async () => {
  await open();
  expect(calls.list).toHaveBeenCalledOnce(); expect(calls.get).toHaveBeenCalledOnce();
  expect(calls.get.mock.calls[0][0]).toMatchObject({ organizationId: "200", proposalId: proposal().proposal_id });
  expect(screen.getAllByText("9007199254740993").length).toBeGreaterThan(0);
  expect(screen.getByText("sha256:sample")).toBeInTheDocument();
  expect(screen.queryByRole("link", { name: /sha256|evidence/ })).not.toBeInTheDocument();
});

it("accept never applies; only separate confirmation sends the newly accepted exact revision", async () => {
  await open(); await userEvent.click(screen.getByRole("button", { name: "接受提案" }));
  await screen.findByRole("button", { name: "应用到标准商品" });
  expect(calls.apply).not.toHaveBeenCalled();
  expect(calls.decide.mock.calls[0][0].input).toEqual({ action: "accept", expected_revision: "1" });
  await userEvent.click(screen.getByRole("button", { name: "应用到标准商品" }));
  expect(calls.apply).not.toHaveBeenCalled();
  const applied = { ...proposal("applied", "2"), apply_receipt: { proposal_id: proposal().proposal_id, revision: "2", product_version: "9007199254740994", publication_id: "real-receipt-reference", actor: "owner", at: "2026-09-07T00:00:00Z" } };
  calls.apply.mockResolvedValue(applied); calls.list.mockResolvedValue({ ...collection(), items: [] });
  await userEvent.click(screen.getByRole("button", { name: "确认应用" }));
  expect(await screen.findByText("9007199254740994")).toBeVisible();
  expect(calls.apply.mock.calls[0][0].input).toEqual({ expected_revision: "2" });
  expect(calls.apply.mock.calls[0][0].idempotencyKey).not.toBe(calls.decide.mock.calls[0][0].idempotencyKey);
  expect(screen.queryByRole("button", { name: "应用到标准商品" })).not.toBeInTheDocument();
});

it("editing accepted data hides Apply and saves a new pending revision without approval", async () => {
  calls.get.mockResolvedValue(proposal("accepted", "8")); calls.decide.mockResolvedValue({ ...proposal("pending", "9"), after: "人工标题" });
  await open(); await userEvent.click(screen.getByRole("button", { name: "编辑标题" }));
  expect(screen.queryByRole("button", { name: "应用到标准商品" })).not.toBeInTheDocument();
  await userEvent.clear(screen.getByRole("textbox", { name: "编辑标题" })); await userEvent.type(screen.getByRole("textbox", { name: "编辑标题" }), "人工标题");
  await userEvent.click(screen.getByRole("button", { name: "保存为待审核提案" }));
  await screen.findByRole("button", { name: "接受提案" });
  expect(calls.decide.mock.calls[0][0].input).toEqual({ action: "edit", expected_revision: "8", title: "人工标题" });
  expect(calls.apply).not.toHaveBeenCalled();
});

it("double click dispatches once; unknown writes stay unknown after GET without receipt and replay only explicitly", async () => {
  const late = deferred<ProductTitleProposal>(); calls.decide.mockReturnValueOnce(late.promise).mockResolvedValue(proposal("accepted", "2"));
  await open(); await userEvent.dblClick(screen.getByRole("button", { name: "接受提案" }));
  expect(calls.decide).toHaveBeenCalledOnce();
  await act(async () => late.reject(failure("RESULT_UNVERIFIED", "unknown", 502)));
  expect(await screen.findByText("结果待核实")).toBeVisible();
  await userEvent.click(screen.getByRole("button", { name: "重新读取当前状态" }));
  await screen.findByText("Original title");
  expect(screen.getByText("结果待核实")).toBeVisible(); expect(calls.decide).toHaveBeenCalledOnce();
  calls.get.mockResolvedValue(proposal("accepted", "2"));
  await userEvent.click(screen.getByRole("button", { name: "核实本次操作" }));
  await screen.findByRole("button", { name: "应用到标准商品" });
  const [first, second] = calls.decide.mock.calls.map(([input]) => input);
  expect(second).toMatchObject({ idempotencyKey: first.idempotencyKey, input: first.input, proposalId: first.proposalId, organizationId: first.organizationId });
});

it.each(["stale_product_version", "operation_conflict"])("fails closed on %s without replacing the base or replaying", async (code) => {
  calls.decide.mockRejectedValue(failure(code)); await open();
  await userEvent.click(screen.getByRole("button", { name: "接受提案" }));
  expect(await screen.findByText(/资料已变化，请重新读取/)).toBeVisible();
  expect(screen.queryByRole("button", { name: "接受提案" })).not.toBeInTheDocument();
  expect(calls.decide).toHaveBeenCalledOnce(); expect(screen.queryByText("private raw details")).not.toBeInTheDocument();
});

it.each(["organization", "user", "roles", "switching", "revoked", "logout", "unmount"])("clears a pending write and ignores late success on %s", async (kind) => {
  const late = deferred<ProductTitleProposal>(); calls.decide.mockReturnValue(late.promise);
  const view = render(tree()); await userEvent.click(await screen.findByRole("button", { name: /product.*查看详情/ }));
  await userEvent.click(await screen.findByRole("button", { name: "接受提案" }));
  const signal = calls.decide.mock.calls[0][0].signal;
  if (kind === "organization") calls.context.effectiveOrganization = { id: "100" };
  if (kind === "user") calls.context.user = { id: "other" };
  if (kind === "roles") calls.context.roles = ["listingkit_operator"]; calls.context.permissions = ["product_sourcing.write","local_agent.write","listingkit.image_agent.read","listingkit.image_agent.write","workbench.agent.read","workbench.agent.use","workbench.chat.read","workbench.chat.use","workbench.task.read","workbench.knowledge.read","workbench.store.read","workbench.store.create","workbench.store.update","workbench.store.lifecycle","workbench.source_account.read","workbench.source_account.manage","workbench.organization_member.read","workbench.commercial.read"];
  if (kind === "switching") calls.context.isSwitching = true;
  if (kind === "revoked") calls.context.blockingError = { code: "ORGANIZATION_ACCESS_REVOKED" };
  if (kind === "logout") calls.context.user = null;
  if (kind === "unmount") view.unmount(); else view.rerender(tree());
  expect(signal.aborted).toBe(true);
  await act(async () => late.resolve(proposal("accepted", "2")));
  expect(screen.queryByText("Original title")).not.toBeInTheDocument(); expect(calls.apply).not.toHaveBeenCalled();
  expect(screen.queryByRole("button", { name: "应用到标准商品" })).not.toBeInTheDocument();
});

it("clears sensitive data on write denial and never falls back to admin", async () => {
  calls.get.mockResolvedValueOnce(proposal()).mockRejectedValue(failure("PERMISSION_DENIED","rejected",403));
  calls.decide.mockRejectedValue(failure("PERMISSION_DENIED", "rejected", 403)); await open();
  await userEvent.click(screen.getByRole("button", { name: "接受提案" }));
  expect(await screen.findByText("当前身份没有标题审核权限")).toBeVisible();
  expect(screen.queryByText("Original title")).not.toBeInTheDocument();
  expect(screen.queryByRole("button", { name: /接受提案|编辑标题|应用到标准商品/ })).not.toBeInTheDocument();
});

it("operator can edit their own proposal but cannot accept, reject or apply", async () => {
  calls.context.roles = ["listingkit_operator"]; calls.context.permissions = ["product_sourcing.write","local_agent.write","listingkit.image_agent.read","listingkit.image_agent.write","workbench.agent.read","workbench.agent.use","workbench.chat.read","workbench.chat.use","workbench.task.read","workbench.knowledge.read","workbench.store.read","workbench.store.create","workbench.store.update","workbench.store.lifecycle","workbench.source_account.read","workbench.source_account.manage","workbench.organization_member.read","workbench.commercial.read"]; await open();
  expect(screen.getByRole("button", { name: "编辑标题" })).toBeEnabled();
  expect(screen.queryByRole("button", { name: /接受提案|拒绝提案|应用到标准商品/ })).not.toBeInTheDocument();
});

it("unconfigured page never fetches and URL selection is a real authorized GET even outside the list", async () => {
  const view = render(tree(false)); expect(calls.list).not.toHaveBeenCalled();
  view.unmount(); calls.list.mockResolvedValue({ ...collection(), items: [] }); calls.get.mockResolvedValue(proposal("rejected", "3"));
  render(tree(true, proposal().proposal_id)); await screen.findByText("Original title");
  expect(calls.get).toHaveBeenCalledOnce();
  expect(screen.queryByRole("button", { name: "接受提案" })).not.toBeInTheDocument();
});

it("restores an initial URL selection after context loading, but never resurrects it after a scope transition", async () => {
  calls.context.isLoading = true;
  const view = render(tree(true, proposal().proposal_id));
  calls.context.isLoading = false; view.rerender(tree(true, proposal().proposal_id));
  await screen.findByText("Original title");
  calls.context.isSwitching = true; view.rerender(tree(true, proposal().proposal_id));
  calls.context.isSwitching = false; view.rerender(tree(true, proposal().proposal_id));
  await screen.findByText("选择一条任务或待确认事项");
  expect(calls.get).toHaveBeenCalledOnce();
});

it("a detail read denial clears the whole sensitive review projection", async () => {
  await open(); calls.get.mockRejectedValue(failure("PERMISSION_DENIED", "rejected", 403));
  await userEvent.click(screen.getByRole("button", { name: "重新读取当前状态" }));
  await screen.findByText("当前身份没有标题审核权限");
  expect(screen.queryByRole("button", { name: /product.*查看详情/ })).not.toBeInTheDocument();
});

it("authoritative rereads replace an accepted response and close its old revision confirmation", async () => {
  await open(); await userEvent.click(screen.getByRole("button", { name: "接受提案" }));
  await userEvent.click(await screen.findByRole("button", { name: "应用到标准商品" }));
  expect(screen.getByRole("dialog")).toBeVisible();
  calls.get.mockResolvedValue({ ...proposal("pending", "3"), after: "另一审核者修改" });
  await act(async () => client.invalidateQueries({ predicate: (q) => q.queryKey.includes("title-proposal") }));
  await screen.findByText("另一审核者修改");
  expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  expect(screen.queryByRole("button", { name: "应用到标准商品" })).not.toBeInTheDocument();
  expect(calls.apply).not.toHaveBeenCalled();
});

it("stopping a dispatched write retains uncertainty even if explicit verification cannot send", async () => {
  calls.decide.mockImplementationOnce(({ signal }: { signal: AbortSignal }) => new Promise((_, reject) => {
    signal.addEventListener("abort", () => reject(failure("RESULT_UNVERIFIED", "unknown", 502)));
  })).mockRejectedValueOnce(new ProductTitleReviewError(400, "INVALID_REQUEST", { code: "INVALID_REQUEST", message: "", requestId: "", fieldErrors: [] }, "not_sent"));
  await open(); await userEvent.click(screen.getByRole("button", { name: "接受提案" }));
  await userEvent.click(screen.getByRole("button", { name: "停止等待" }));
  await screen.findByText("结果待核实");
  await userEvent.click(screen.getByRole("button", { name: "核实本次操作" }));
  expect(await screen.findByText("结果待核实")).toBeVisible();
  expect(screen.queryByRole("button", { name: "接受提案" })).not.toBeInTheDocument();
  expect(calls.decide.mock.calls[1][0].idempotencyKey).toBe(calls.decide.mock.calls[0][0].idempotencyKey);
});

it("a rejected proposal leaves the actionable list but keeps its authorized terminal detail", async () => {
  calls.decide.mockResolvedValue(proposal("rejected", "2")); await open();
  calls.list.mockResolvedValue({ ...collection(), items: [] });
  await userEvent.click(screen.getByRole("button", { name: "拒绝提案" }));
  await screen.findByText("当前筛选无任务");
  expect(screen.getByText("Original title")).toBeVisible();
  expect(screen.getByText("已拒绝")).toBeVisible();
  expect(calls.decide.mock.calls[0][0].input).toEqual({ action: "reject", expected_revision: "1" });
  expect(calls.apply).not.toHaveBeenCalled();
});

it("pagination forwards the opaque cursor and clears details; refresh resets to the first page", async () => {
  calls.list.mockResolvedValue({ ...collection(), next_cursor: "opaque-cursor" }); await open();
  await userEvent.click(screen.getByRole("button", { name: "下一页" }));
  await screen.findByText("第 2 页");
  expect(calls.list.mock.calls.at(-1)![0].cursor).toBe("opaque-cursor");
  expect(screen.queryByText("Original title")).not.toBeInTheDocument();
  await userEvent.click(screen.getByRole("button", { name: /刷新/ }));
  await screen.findByText("第 1 页");
  expect(calls.list.mock.calls.at(-1)![0].cursor).toBeUndefined();
});

it("a late write failure cannot replace a new organization's review page", async () => {
  const late = deferred<ProductTitleProposal>(); calls.decide.mockReturnValue(late.promise);
  const view = render(tree()); await userEvent.click(await screen.findByRole("button", { name: /product.*查看详情/ }));
  await userEvent.click(await screen.findByRole("button", { name: "接受提案" }));
  calls.context.effectiveOrganization = { id: "100" }; view.rerender(tree());
  await act(async () => late.reject(failure("PERMISSION_DENIED", "rejected", 403)));
  expect(screen.queryByText("当前身份没有标题审核权限")).not.toBeInTheDocument();
  expect(screen.queryByText("结果待核实")).not.toBeInTheDocument();
  expect(await screen.findByText("选择一条任务或待确认事项")).toBeVisible();
});

it("a late collection read failure preserves an unknown write and its exact verification intent", async () => {
  const list = deferred<ProductTitleProposalList>(); calls.list.mockReturnValue(list.promise);
  calls.decide.mockRejectedValue(failure("RESULT_UNVERIFIED", "unknown", 502));
  render(tree(true, proposal().proposal_id));
  await userEvent.click(await screen.findByRole("button", { name: "接受提案" }));
  await screen.findByText("结果待核实");
  await act(async () => list.reject(failure("DEPENDENCY_UNAVAILABLE", "rejected", 503)));
  await screen.findByText("标题提案服务暂不可用");
  expect(screen.getByText("结果待核实")).toBeVisible();
  await userEvent.click(screen.getByRole("button", { name: "核实本次操作" }));
  expect(calls.decide.mock.calls[1][0].idempotencyKey).toBe(calls.decide.mock.calls[0][0].idempotencyKey);
});

it("collection permission denial clears an in-flight operation and releases the page lock", async () => {
  const list = deferred<ProductTitleProposalList>(); calls.list.mockReturnValue(list.promise);
  const write = deferred<ProductTitleProposal>(); calls.decide.mockReturnValue(write.promise);
  render(tree(true, proposal().proposal_id));
  await userEvent.click(await screen.findByRole("button", { name: "接受提案" }));
  await act(async () => list.reject(failure("PERMISSION_DENIED", "rejected", 403)));
  await screen.findByText("当前身份没有标题审核权限");
  expect(calls.decide.mock.calls[0][0].signal.aborted).toBe(true);
  expect(screen.queryByText("Original title")).not.toBeInTheDocument();
  expect(screen.getByRole("button", { name: /刷新/ })).toBeEnabled();
  await act(async () => write.resolve(proposal("accepted", "2")));
  expect(screen.queryByRole("button", { name: "应用到标准商品" })).not.toBeInTheDocument();
});

it("shows the handed-off 413 not_sent error without an unknown-result replay", async () => {
  calls.decide.mockRejectedValue(new ProductTitleReviewError(413, "INPUT_TOO_LARGE", { code: "INPUT_TOO_LARGE", message: "private", requestId: "", fieldErrors: [] }, "not_sent"));
  await open(); await userEvent.click(screen.getByRole("button", { name: "编辑标题" }));
  await userEvent.click(screen.getByRole("button", { name: "保存为待审核提案" }));
  expect(await screen.findByText("标题或请求内容过长")).toBeVisible();
  expect(screen.queryByText("结果待核实")).not.toBeInTheDocument();
  expect(screen.getByRole("textbox", { name: "编辑标题" })).toBeEnabled();
});

const ownItemId = "550e8400-e29b-41d4-a716-446655440022";
function ownProposal(state: ProductTitleProposal["state"] = "applied", binding: Record<string, string> = {}, scope: Record<string, string> = {}) {
  const original = contentProposalFixture();
  const selection = { targetLanguage: "zh-CN", title: true };
  const revision = state === "pending" ? "1" : state === "accepted" ? "2" : "3";
  const value = parseProductTitleProposal({ ...original, state, revision, owner: scope.ActorID ?? original.owner,
    content: { ...original.content, selection, before: original.content.before.slice(0, 1), after: original.content.after.slice(0, 1) },
    execution: { ...original.execution, ContentSelection: selection, scope: { OrganizationID: "200", ActorID: "owner", ...scope },
      binding: { ...original.execution.binding, ContextKind: "own_product", ContextID: ownItemId, ...binding } },
    ...(state === "applied" ? { apply_receipt: { proposal_id: original.proposal_id, revision, product_version: "2", publication_id: "own-apply-receipt", actor: "owner", at: "2026-10-15T00:00:00Z" } } : {}),
  });
  if (!value) throw new Error("invalid own review fixture");
  return value;
}

it("returns after a confirmed own Apply and rereads fresh eligibility instead of cached ineligible data", async () => {
  calls.get.mockResolvedValue(ownProposal("pending")); calls.decide.mockResolvedValue(ownProposal("accepted")); calls.apply.mockResolvedValue(ownProposal());
  const keys = ["application-products", "choice", "application-overview"].map(kind => ["supply-market", "owner", "200", kind]);
  const other = ["supply-market", "owner", "100", "application-products"];
  for (const key of [...keys, other]) client.setQueryData(key, { optimized: false });
  render(tree(true, ownProposal().proposal_id));
  await userEvent.click(await screen.findByRole("button", { name: "接受提案" }));
  await userEvent.click(await screen.findByRole("button", { name: "应用到标准商品" }));
  expect(screen.queryByRole("link", { name: "返回优选申请并刷新资格" })).not.toBeInTheDocument();
  await userEvent.click(screen.getByRole("button", { name: "确认应用" }));
  const back = await screen.findByRole("link", { name: "返回优选申请并刷新资格" });
  expect(back).toHaveAttribute("href", "/workbench/supply/applications/new");
  expect(screen.getByRole("link", { name: "查看自有商品" })).toHaveAttribute("href", `/workbench/data/mine/own/${ownItemId}`);
  const fresh = vi.fn().mockResolvedValue({ optimized: true });
  for (const key of keys) expect(await client.fetchQuery({ queryKey: key, queryFn: fresh, staleTime: 60_000 })).toEqual({ optimized: true });
  expect(fresh).toHaveBeenCalledTimes(3);
  expect(await client.fetchQuery({ queryKey: other, queryFn: fresh, staleTime: 60_000 })).toEqual({ optimized: false });
  // Returning must recheck even if the inactive cache was refreshed while reviewing the receipt.
  back.addEventListener("click", event => event.preventDefault());
  await userEvent.click(back);
  await client.fetchQuery({ queryKey: keys[0], queryFn: fresh, staleTime: 60_000 });
  expect(fresh).toHaveBeenCalledTimes(4);
  expect(calls.apply).toHaveBeenCalledOnce(); expect(calls.decide).toHaveBeenCalledOnce();
});

it("derives the original own item from an authorized applied proposal without a return URL", async () => {
  calls.get.mockResolvedValue(ownProposal());
  render(tree(true, ownProposal().proposal_id));
  expect(await screen.findByRole("link", { name: "查看自有商品" })).toHaveAttribute("href", `/workbench/data/mine/own/${ownItemId}`);
  expect(calls.apply).not.toHaveBeenCalled(); expect(calls.decide).not.toHaveBeenCalled();
});

it.each([
  ["acquisition", { ContextKind: "acquisition" }, {}],
  ["invalid item", { ContextID: "https://example.invalid/return" }, {}],
  ["other actor", {}, { ActorID: "another-member" }],
  ["other organization", {}, { OrganizationID: "100" }],
] as const)("does not offer an own return for %s provenance", async (_, binding, scope) => {
  calls.get.mockResolvedValue(ownProposal("applied", binding, scope));
  render(tree(true, ownProposal().proposal_id)); await screen.findByRole("region", { name: "应用回执" });
  expect(screen.queryByRole("link", { name: "查看自有商品" })).not.toBeInTheDocument();
  expect(screen.queryByRole("link", { name: "返回优选申请并刷新资格" })).not.toBeInTheDocument();
});

it("keeps an own Apply unknown after GET until the original action is explicitly verified", async () => {
  calls.get.mockResolvedValue(ownProposal("accepted")); calls.apply.mockRejectedValueOnce(failure("RESULT_UNVERIFIED", "unknown", 502));
  render(tree(true, ownProposal().proposal_id));
  await userEvent.click(await screen.findByRole("button", { name: "应用到标准商品" }));
  await userEvent.click(screen.getByRole("button", { name: "确认应用" })); await screen.findByText("结果待核实");
  calls.get.mockResolvedValue(ownProposal());
  await userEvent.click(screen.getByRole("button", { name: "重新读取当前状态" }));
  await screen.findByRole("region", { name: "应用回执" });
  expect(screen.queryByRole("link", { name: "返回优选申请并刷新资格" })).not.toBeInTheDocument();
  calls.apply.mockResolvedValue(ownProposal());
  await userEvent.click(screen.getByRole("button", { name: "核实本次操作" }));
  await screen.findByRole("link", { name: "返回优选申请并刷新资格" });
  expect(calls.apply.mock.calls[1][0].idempotencyKey).toBe(calls.apply.mock.calls[0][0].idempotencyKey);
});

it("removes the product-aware return when the current organization changes", async () => {
  calls.get.mockResolvedValue(ownProposal());
  const view = render(tree(true, ownProposal().proposal_id));
  await screen.findByRole("link", { name: "查看自有商品" });
  calls.context.effectiveOrganization = { id: "100" }; view.rerender(tree(true, ownProposal().proposal_id));
  expect(screen.queryByRole("link", { name: "查看自有商品" })).not.toBeInTheDocument();
  expect(screen.queryByRole("link", { name: "返回优选申请并刷新资格" })).not.toBeInTheDocument();
});
