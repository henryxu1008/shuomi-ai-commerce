import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { BusinessTaskPage } from "./task-page";
import { ConsoleNavigation } from "../console/console-navigation";
import { titleProposalFixture } from "@/test/product-title-review-fixture";
import { ProductTitleReviewError } from "@/lib/api/product-title-review-client";
import { AIWorkbenchError } from "@/lib/api/ai-workbench";

const calls = vi.hoisted(() => ({ tasks: vi.fn(), list: vi.fn(), get: vi.fn(), decide: vi.fn(), apply: vi.fn(), context: {} as Record<string, unknown> }));
vi.mock("@/components/providers/workbench-context-provider", () => ({ useWorkbenchContext: () => calls.context }));
vi.mock("next/navigation", () => ({ useSearchParams: () => new URLSearchParams() }));
vi.mock("@/lib/api/ai-workbench", async original => ({ ...await original<object>(), requestAIWorkbench: calls.tasks }));
vi.mock("@/lib/api/product-title-review-client", async original => ({ ...await original<object>(), fetchProductTitleProposals: calls.list, fetchProductTitleProposal: calls.get, decideProductTitleProposal: calls.decide, applyProductTitleProposal: calls.apply }));
const proposal = titleProposalFixture();
const task = { id: "550e8400-e29b-41d4-a716-446655440001", conversationId: "550e8400-e29b-41d4-a716-446655440002", proposalId: "550e8400-e29b-41d4-a716-446655440003",
  title: "蓝色水杯标题优化", goalSummary: "为商品生成可审核标题", createdAt: "2026-10-18T00:00:00Z", projectionAvailable: true, state: "WAITING_CONFIRMATION",
  canStart: false, canReconcile: false, canResume: false, canReview: false, productDetailsAvailable: true, productKey: "blue-cup", targetPlatform: "shein", reviewId: proposal.proposal_id };
let client: QueryClient;
const tree = () => <QueryClientProvider client={client}><BusinessTaskPage productReviewAvailable /></QueryClientProvider>;
beforeEach(() => {
  client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  calls.context = { user: { id: "owner" }, effectiveOrganization: { id: "200" }, roles: [], permissions: ["listingkit.admin.read", "listingkit.admin.write"], aiWorkbenchAvailable: false };
  calls.tasks.mockReset().mockImplementation(({route}) => Promise.resolve(route === "task-read" ? {task} : {tasks:[task],next:""}));
  calls.list.mockReset().mockResolvedValue({ schema_version: 1, coverage: "product-title-proposals-only", items: [{ proposal_id: proposal.proposal_id, product_key: "blue-cup", base_version: "1", proposal_revision: "1", state: "pending" }], next_cursor: null });
  calls.get.mockReset().mockResolvedValue(proposal); calls.decide.mockReset(); calls.apply.mockReset();
});
afterEach(() => { cleanup(); client.clear(); sessionStorage.clear(); });
it("reads Review directly in Task Center when Chat is unmounted", async () => {
  render(tree());
  const row = await screen.findByRole("button", {name:/调整标准商品标题.*blue-cup/});
  expect(screen.queryByRole("link",{name:"打开标题审核"})).not.toBeInTheDocument();
  expect(calls.tasks).not.toHaveBeenCalled();
  fireEvent.click(row);
  expect(await screen.findByText("Original title")).toBeVisible();
  expect(screen.getByRole("region",{name:"任务详情"})).toBeVisible();
  expect(calls.decide).not.toHaveBeenCalled(); expect(calls.apply).not.toHaveBeenCalled();
});
it("keeps linked Review once and opens its existing controls inside the selected task", async () => {
  calls.context.aiWorkbenchAvailable = true;
  render(tree());
  fireEvent.click(await screen.findByRole("button",{name:/蓝色水杯标题优化.*查看详情/}));
  expect(await screen.findByText("Original title")).toBeVisible();
  expect(screen.queryByRole("button",{name:/调整标准商品标题.*blue-cup/})).not.toBeInTheDocument();
  expect(within(screen.getByRole("region",{name:"任务详情"})).getByRole("button",{name:"接受提案"})).toBeVisible();
});
it("keeps selection locked after an uncertain Review write", async () => {
  calls.decide.mockRejectedValue(new ProductTitleReviewError(502,"RESULT_UNVERIFIED",{code:"RESULT_UNVERIFIED",message:"",requestId:"",fieldErrors:[]},"unknown"));
  render(tree());
  fireEvent.click(await screen.findByRole("button",{name:/调整标准商品标题.*blue-cup/}));
  fireEvent.click(await screen.findByRole("button",{name:"接受提案"}));
  await screen.findByText("结果待核实");
  expect(screen.getByRole("searchbox",{name:"搜索已加载记录"})).toBeDisabled();
  expect(screen.getByRole("button",{name:"刷新状态"})).toBeDisabled();
  expect(screen.getByRole("button",{name:/调整标准商品标题.*blue-cup/})).toBeDisabled();
  expect(calls.decide).toHaveBeenCalledOnce(); expect(calls.apply).not.toHaveBeenCalled();
});
it("clears the selected authorized result when enterprise changes", async () => {
  const view = render(tree());
  fireEvent.click(await screen.findByRole("button",{name:/调整标准商品标题.*blue-cup/}));
  await screen.findByText("Original title");
  calls.context.effectiveOrganization = {id:"other"};
  calls.list.mockImplementation(() => new Promise(()=>{}));
  view.rerender(tree());
  expect(screen.queryByText("Original title")).not.toBeInTheDocument();
  expect(calls.apply).not.toHaveBeenCalled();
});
it("filters only loaded rows without dispatching an execution", async () => {
  render(tree());
  await screen.findByRole("button",{name:/调整标准商品标题.*blue-cup/});
  fireEvent.change(screen.getByRole("searchbox",{name:"搜索已加载记录"}),{target:{value:"没有匹配商品"}});
  expect(screen.queryByRole("button",{name:/调整标准商品标题.*blue-cup/})).not.toBeInTheDocument();
  expect(calls.decide).not.toHaveBeenCalled(); expect(calls.tasks).not.toHaveBeenCalled();
});
it("puts direct proposal navigation under Task Center pending instead of a parallel module", () => {
  render(<ConsoleNavigation pathname="/workbench/ai/tasks/pending/other" ariaLabel="主导航" productReviewAvailable />);
  expect(screen.queryByRole("link",{name:"标题审核"})).not.toBeInTheDocument();
  expect(screen.getByRole("link",{name:"任务中心"})).toBeVisible();
  expect(screen.getByRole("link",{name:"待确认"})).toHaveAttribute("aria-current","page");
});

it("can find a linked suggestion by its review title after its task is filtered out", async () => {
  calls.context.aiWorkbenchAvailable = true;
  render(tree());
  await screen.findByRole("button",{name:/蓝色水杯标题优化/});
  fireEvent.change(screen.getByRole("searchbox",{name:"搜索已加载记录"}),{target:{value:"调整标准商品标题"}});
  expect(await screen.findByRole("button",{name:/调整标准商品标题.*blue-cup/})).toBeVisible();
});
it("keeps loaded but unprojectable task counts unknown", async () => {
  calls.context.aiWorkbenchAvailable = true;
  calls.tasks.mockResolvedValue({tasks:[{...task,projectionAvailable:false,state:undefined}],next:""});
  calls.list.mockResolvedValue({items:[],next_cursor:null});
  render(tree());
  await screen.findByRole("button",{name:/蓝色水杯标题优化/});
  expect(screen.getAllByText("—")).toHaveLength(3);
});

it("restores keyboard focus to the selected row when closing detail", async () => {
  render(tree());
  const row = await screen.findByRole("button",{name:/调整标准商品标题.*blue-cup/});
  fireEvent.click(row);
  expect(screen.getByRole("heading",{name:"任务详情"})).toHaveFocus();
  fireEvent.click(screen.getByRole("button",{name:"返回列表"}));
  expect(row).toHaveFocus();
});
it.each(["failed", "redacted"])("retains an uncertain Review session when its linked task read is %s", async outcome => {
  calls.context.aiWorkbenchAvailable = true;
  calls.decide.mockRejectedValue(new ProductTitleReviewError(502,"RESULT_UNVERIFIED",{code:"RESULT_UNVERIFIED",message:"",requestId:"",fieldErrors:[]},"unknown"));
  render(tree());
  fireEvent.click(await screen.findByRole("button",{name:/蓝色水杯标题优化.*查看详情/}));
  fireEvent.click(await screen.findByRole("button",{name:"接受提案"}));
  await screen.findByText("结果待核实");
  calls.tasks.mockImplementation(({route})=>route==="task-read" ? outcome === "failed" ? Promise.reject(new Error("offline")) : Promise.resolve({task:{...task,projectionAvailable:false,reviewId:undefined}}) : Promise.resolve({tasks:[task],next:""}));
  await act(async()=>{await client.invalidateQueries({queryKey:["ai-task"]});});
  if(outcome === "failed") expect(await screen.findByText("任务不可用")).toBeVisible();
  else expect(await screen.findByText("Owner 暂不可用，无法确认当前状态")).toBeVisible();
  expect(screen.getByText("结果待核实")).toBeVisible();
  expect(screen.getByRole("button",{name:"刷新状态"})).toBeDisabled();
  expect(calls.decide).toHaveBeenCalledOnce();
});

it.each([
  ["start", "启动原任务", {canStart:true}],
  ["start", "核实超期执行结果", {canReconcile:true}],
  ["resume", "继续原任务", {canResume:true,agentRevision:"2"}],
  ["review", "提交标题供人工审核", {canReview:true}],
] as const)("locks browsing throughout %s (%s), including an unknown result", async (action, label, flags) => {
  calls.context.aiWorkbenchAvailable = true;
  const item = {...task,reviewId:undefined,...flags};
  let reject!: (error: Error) => void;
  calls.tasks.mockImplementation(({route}) => route === `task-${action}` ? new Promise((_, rejectWrite) => {reject = rejectWrite;}) : Promise.resolve(route === "task-read" ? {task:item} : {tasks:[item],next:"more"}));
  render(tree());
  fireEvent.click(await screen.findByRole("button",{name:/蓝色水杯标题优化.*查看详情/}));
  const command = await screen.findByRole("button",{name:label});
  if(action === "resume") fireEvent.change(screen.getByRole("textbox",{name:"继续执行的反馈"}),{target:{value:"保持原商品事实"}});
  fireEvent.click(command);
  await waitFor(()=>expect(calls.tasks).toHaveBeenCalledWith(expect.objectContaining({route:`task-${action}`})));
  const expectLocked = () => {
    expect(screen.getByRole("searchbox",{name:"搜索已加载记录"})).toBeDisabled();
    expect(screen.getByRole("button",{name:/蓝色水杯标题优化.*查看详情/})).toBeDisabled();
    expect(screen.getByRole("button",{name:"返回列表"})).toBeDisabled();
    expect(screen.getByRole("button",{name:"刷新状态"})).toBeDisabled();
    expect(screen.getByRole("button",{name:"加载更早任务"})).toBeDisabled();
    expect(screen.getByRole("link",{name:"执行中"})).toHaveAttribute("aria-disabled","true");
  };
  expectLocked();
  await act(async()=>reject(new AIWorkbenchError("TASK_OUTCOME_UNKNOWN")));
  await screen.findByText("操作未完成");
  expectLocked();
  expect(command).toBeDisabled();
  expect(screen.getByRole("button",{name:"核实原操作结果"})).toBeEnabled();
});

it("replays the original Resume key and frozen inputs after the task revision advances", async () => {
  calls.context.aiWorkbenchAvailable = true;
  let item = {...task,reviewId:undefined,canResume:true,agentRevision:"2"};
  let writes = 0;
  calls.tasks.mockImplementation(({route}) => route === "task-resume" ? ++writes === 1 ? Promise.reject(new AIWorkbenchError("OUTCOME_UNKNOWN")) : Promise.resolve({}) : Promise.resolve(route === "task-read" ? {task:item} : {tasks:[item],next:""}));
  render(tree());
  fireEvent.click(await screen.findByRole("button",{name:/蓝色水杯标题优化.*查看详情/}));
  await screen.findByRole("button",{name:"继续原任务"});
  fireEvent.change(screen.getByRole("textbox",{name:"继续执行的反馈"}),{target:{value:"保持原商品事实"}});
  fireEvent.click(screen.getByRole("button",{name:"继续原任务"}));
  await screen.findByText("操作未完成");
  item = {...item,agentRevision:"3",canResume:false};
  await act(async()=>{await client.invalidateQueries({queryKey:["ai-task"]});});
  fireEvent.click(screen.getByRole("button",{name:"核实原操作结果"}));
  await waitFor(()=>expect(screen.getByRole("button",{name:"刷新状态"})).toBeEnabled());
  const sent = calls.tasks.mock.calls.map(([input])=>input).filter(input=>input.route === "task-resume");
  expect(sent).toHaveLength(2);
  expect(sent[1].key).toBe(sent[0].key);
  expect(sent[1].body).toEqual({revision:"2",feedback:"保持原商品事实"});
});

it("recovers a saved Resume intent after reload with a newer task revision", async () => {
  calls.context.aiWorkbenchAvailable = true;
  let item = {...task,reviewId:undefined,canResume:true,agentRevision:"2"};
  const sent: {key:string;body:unknown}[] = [];
  calls.tasks.mockImplementation(({route,key,body}) => {
    if(route === "task-resume") {sent.push({key,body});return sent.length === 1 ? Promise.reject(new AIWorkbenchError("OUTCOME_UNKNOWN")) : Promise.resolve({});}
    return Promise.resolve(route === "task-read" ? {task:item} : {tasks:[item],next:""});
  });
  const first = render(tree());
  fireEvent.click(await screen.findByRole("button",{name:/蓝色水杯标题优化.*查看详情/}));
  await screen.findByRole("button",{name:"继续原任务"});
  fireEvent.change(screen.getByRole("textbox",{name:"继续执行的反馈"}),{target:{value:"原反馈"}});
  fireEvent.click(screen.getByRole("button",{name:"继续原任务"}));
  await screen.findByText("操作未完成"); first.unmount();
  item = {...item,agentRevision:"3"};
  client.clear(); render(tree());
  fireEvent.click(await screen.findByRole("button",{name:/蓝色水杯标题优化.*查看详情/}));
  fireEvent.click(await screen.findByRole("button",{name:"继续原任务"}));
  expect(sent).toHaveLength(1);
  fireEvent.click(await screen.findByRole("button",{name:"核实原操作结果"}));
  await waitFor(()=>expect(sent).toHaveLength(2));
  expect(sent[1]).toEqual(sent[0]);
});

it.each(["start", "resume", "review"] as const)("restores %s verification without current task capabilities or a readable task", async action => {
  calls.context.aiWorkbenchAvailable = true;
  const key = "550e8400-e29b-41d4-a716-446655440099";
  const original = {key,revision:action === "resume" ? "2" : "",feedback:action === "resume" ? "原反馈" : ""};
  sessionStorage.setItem(`ai-workbench-task-action:owner:200:${task.id}:${action}`,JSON.stringify(original));
  let unreadable = false;
  calls.tasks.mockImplementation(({route}) => route === `task-${action}` ? Promise.resolve({}) : route === "task-read" && unreadable ? Promise.reject(new Error("offline")) : Promise.resolve(route === "task-read" ? {task:{...task,reviewId:undefined}} : {tasks:[{...task,reviewId:undefined}],next:""}));
  const first = render(tree());
  fireEvent.click(await screen.findByRole("button",{name:/蓝色水杯标题优化.*查看详情/}));
  expect(await screen.findByRole("button",{name:"核实原操作结果"})).toBeEnabled();
  expect(calls.tasks.mock.calls.some(([input])=>input.method === "POST")).toBe(false);
  first.unmount(); client.clear(); unreadable = true;
  render(tree());
  fireEvent.click(await screen.findByRole("button",{name:/蓝色水杯标题优化.*查看详情/}));
  await screen.findByText("任务不可用");
  expect(screen.getByRole("button",{name:"刷新状态"})).toBeDisabled();
  fireEvent.click(screen.getByRole("button",{name:"核实原操作结果"}));
  await waitFor(()=>expect(screen.getByRole("button",{name:"刷新状态"})).toBeEnabled());
  const sent = calls.tasks.mock.calls.map(([input])=>input).filter(input=>input.method === "POST");
  expect(sent).toHaveLength(1);
  expect(sent[0]).toMatchObject({route:`task-${action}`,key,...(action === "resume" ? {body:{revision:"2",feedback:"原反馈"}} : {})});
  expect(sessionStorage.getItem(`ai-workbench-task-action:owner:200:${task.id}:${action}`)).toBeNull();
});
