import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import ResearchPage from "@/app/workbench/data/research/page";
import SelectionPage from "@/app/workbench/supply/selection/page";

const fixture = vi.hoisted(() => ({ project: vi.fn(), context: {
  user: { id: "user-a" }, effectiveOrganization: { id: "org-a" },
  permissions: ["workbench.project.read", "workbench.collection.read"],
  marketResearchAvailable: true, projectCenterAvailable: true,
  isLoading: false, isSwitching: false, selectionRequired: false,
  error: null, blockingError: null, retry: vi.fn(),
} }));
vi.mock("@/components/providers/workbench-context-provider", () => ({ useWorkbenchContext: () => fixture.context }));
vi.mock("@/components/workbench/project-center/project-page", () => ({ ProjectPage: (props: unknown) => {
  fixture.project(props); return <p>原项目操作页面</p>;
} }));

beforeEach(() => {
  fixture.project.mockClear(); Object.assign(fixture.context, {
    permissions: ["workbench.project.read", "workbench.collection.read"],
    marketResearchAvailable: true, projectCenterAvailable: true,
    isLoading: false, isSwitching: false, selectionRequired: false,
  });
});
afterEach(cleanup);

it.each([
  { Page: ResearchPage, title: "从研究目标到样本证据，整理有依据的候选", flow: "市场调研业务流程", action: /管理研究项目/, root: "/workbench/data/research", step: "projects" },
  { Page: SelectionPage, title: "比较候选、记录决定，让选品有据可查", flow: "智能选品业务流程", action: /比较同项目候选/, root: "/workbench/supply/selection", step: "candidates" },
])("shows the primary business homepage before project operations at $root", async ({ Page, title, flow, action, root, step }) => {
  render(await Page({ searchParams: Promise.resolve({ mode: "archived", search: "收纳" }) }));
  expect(screen.getByRole("heading", { name: title, level: 1 })).toBeVisible();
  const process = screen.getByRole("region", { name: flow });
  expect(process.querySelectorAll("ol > li")).toHaveLength(4);
  const target = new URL(screen.getByRole("link", { name: action }).getAttribute("href")!, "https://test.invalid");
  expect(target.pathname).toBe(root);
  expect(Object.fromEntries(target.searchParams)).toEqual({ step, mode: "archived", search: "收纳" });
  expect(fixture.project).not.toHaveBeenCalled();
  expect(screen.queryByText("还没有项目")).not.toBeInTheDocument();
});

it.each([
  { Page: ResearchPage, step: "sources" },
  { Page: SelectionPage, step: "decisions" },
])("keeps the approved $step function on the existing project consumer", async ({ Page, step }) => {
  render(await Page({ searchParams: Promise.resolve({ step, mode: "archived", search: "收纳" }) }));
  expect(screen.getByText("原项目操作页面")).toBeVisible();
  expect(fixture.project.mock.lastCall?.[0]).toMatchObject({ workflowStep: step, mode: "archived", initialSearch: "收纳" });
});

it("does not enter project operations for a foreign or repeated step", async () => {
  const view = render(await ResearchPage({ searchParams: Promise.resolve({ step: "analysis" }) }));
  expect(screen.getByRole("heading", { name: "市场调研业务流程" })).toBeVisible();
  view.rerender(await ResearchPage({ searchParams: Promise.resolve({ step: ["projects", "sources"] }) }));
  expect(fixture.project).not.toHaveBeenCalled();
});

it("keeps the workflow introduction with inactive entries when research is not installed", async () => {
  fixture.context.marketResearchAvailable = false;
  render(await ResearchPage({ searchParams: Promise.resolve({}) }));
  expect(screen.getByRole("heading", { name: "市场调研业务流程" })).toBeVisible();
  expect(screen.queryAllByRole("link")).toHaveLength(0);
  expect(screen.getByRole("status")).toHaveTextContent("当前应用尚未启用研究选品");
  expect(fixture.project).not.toHaveBeenCalled();
});

it("keeps project management separate from evidence access", async () => {
  fixture.context.permissions = ["workbench.project.read"];
  render(await ResearchPage({ searchParams: Promise.resolve({}) }));
  expect(screen.getByRole("link", { name: /管理研究项目/ })).toBeVisible();
  expect(screen.queryByRole("link", { name: /进入资料整理/ })).not.toBeInTheDocument();
  expect(screen.getByRole("status")).toHaveTextContent("商品资料查看权限");
});

it("removes the previous enterprise entry links while organization context is switching", async () => {
  const view = render(await SelectionPage({ searchParams: Promise.resolve({}) }));
  expect(screen.getByRole("link", { name: /比较同项目候选/ })).toBeVisible();
  fixture.context.isSwitching = true;
  view.rerender(await SelectionPage({ searchParams: Promise.resolve({}) }));
  expect(screen.queryAllByRole("link")).toHaveLength(0);
  expect(screen.queryByRole("heading", { name: "智能选品业务流程" })).not.toBeInTheDocument();
});
