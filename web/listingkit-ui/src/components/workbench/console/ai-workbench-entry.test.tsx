import {cleanup, render, screen} from "@testing-library/react";
import {afterEach, beforeEach, expect, it, vi} from "vitest";
import Page from "@/app/workbench/ai/page";

const fixture = vi.hoisted(() => ({
  redirect: vi.fn(), knowledge: false, reports: false, review: false, history: false, native:false,
  context: {aiWorkbenchAvailable: false, projectCenterAvailable: false, isLoading: false,
    isSwitching: false, selectionRequired: false, error: null, blockingError: null,
    effectiveOrganization: {id: "org-a"},permissions:["workbench.agent.read","workbench.agent.use"]},
}));
vi.mock("next/navigation", () => ({redirect: fixture.redirect}));
vi.mock("@/components/providers/workbench-context-provider", () => ({useWorkbenchContext: () => fixture.context}));
vi.mock("@/lib/server/knowledge-availability", () => ({isKnowledgeAvailable: () => fixture.knowledge}));
vi.mock("@/lib/server/report-center-availability", () => ({isReportCenterAvailable: () => fixture.reports}));
vi.mock("@/lib/server/product-title-review-request", () => ({configuredProductReviewOrigin: () => fixture.review ? "https://review.test" : null}));
vi.mock("@/lib/server/shein-records-availability", () => ({isSheinRecordsAvailable: () => fixture.history}));
vi.mock("@/lib/server/agent-configuration-origin",()=>({isAgentConfigurationAvailable:()=>fixture.native}));
beforeEach(() => {
  fixture.redirect.mockReset();
  fixture.knowledge = fixture.reports = fixture.review = fixture.history = fixture.native = false;
  fixture.context.aiWorkbenchAvailable = fixture.context.projectCenterAvailable = fixture.context.isLoading = fixture.context.isSwitching = false;
  fixture.context.selectionRequired = false;
});
afterEach(cleanup);

it('keeps all four introduction cards and gates each business capability', () => {
  fixture.context.projectCenterAvailable = true;
  fixture.reports = true;
  render(<Page />);
  expect(fixture.redirect).not.toHaveBeenCalled();
  expect(screen.getByRole('link', { name: /进入项目中心/ })).toHaveAttribute('href', '/workbench/ai/projects');
  expect(screen.getByRole('link', { name: /进入我的报告/ })).toHaveAttribute('href', '/workbench/ai/reports');
  expect(screen.queryByRole('link', { name: /进入硕米Chat/ })).not.toBeInTheDocument();
});
it.each([['review', '/workbench/ai/tasks'], ['history', '/workbench/ai/tasks'],['native','/workbench/ai/tasks']] as const)('retains independent %s task access', (capability, destination) => {
  fixture[capability] = true;
  render(<Page />);
  expect(fixture.redirect).not.toHaveBeenCalled();
  expect(screen.getByRole('link', { name: /进入任务中心/ })).toHaveAttribute('href', destination);
});
it('keeps the AI introduction without optional services', () => {
  render(<Page />);
  expect(fixture.redirect).not.toHaveBeenCalled();
  expect(screen.getByRole('heading', { name: 'AI工作台', level: 1 })).toBeVisible();
  expect(screen.getByRole('heading', { name: '硕米Chat' })).toBeVisible();
  expect(screen.getByRole('heading', { name: '任务中心' })).toBeVisible();
  expect(screen.getByRole('heading', { name: '项目中心' })).toBeVisible();
  expect(screen.getByRole('heading', { name: '我的报告' })).toBeVisible();
  expect(screen.queryByRole('link')).not.toBeInTheDocument();
});
it.each(['isLoading', 'isSwitching', 'selectionRequired'] as const)('clears the AI homepage during %s', flag => {
  fixture.context[flag] = true;
  render(<Page />);
  expect(fixture.redirect).not.toHaveBeenCalled();
  expect(screen.queryByRole('heading', { level: 1 })).not.toBeInTheDocument();
});
