import { expect, it, vi } from "vitest";
import RootPage from "@/app/workbench/ai/tasks/page";
import PendingPage from "@/app/workbench/ai/tasks/pending/page";
import RunningPage from "@/app/workbench/ai/tasks/running/page";
import CompletedPage from "@/app/workbench/ai/tasks/completed/page";
import ErrorsPage from "@/app/workbench/ai/tasks/errors/page";

vi.mock("next/server", () => ({connection: vi.fn()}));
vi.mock("@/lib/server/product-title-review-request", () => ({configuredProductReviewOrigin: () => "http://review.invalid"}));
vi.mock("@/lib/server/shein-records-availability", () => ({isSheinRecordsAvailable: () => true}));
vi.mock("@/lib/server/agent-configuration-origin",()=>({isAgentConfigurationAvailable:()=>true}));
vi.mock("./task-page", () => ({BusinessTaskPage: () => null}));

it.each([RootPage, PendingPage])("restores the exact proposal selected in a Task Center URL", async Page => {
  const id = "550e8400-e29b-41d4-a716-446655440009";
  const element = await Page({searchParams:Promise.resolve({proposal_id:id})});
  expect(element.props).toMatchObject({initialProposalId:id,productReviewAvailable:true});
});
it.each([RootPage, PendingPage])("does not resolve multiple URL values into a different proposal", async Page => {
  const element = await Page({searchParams:Promise.resolve({proposal_id:["one","two"]})});
  expect(element.props.initialProposalId).toBeUndefined();
});
it.each([RootPage,PendingPage,RunningPage,CompletedPage,ErrorsPage])("keeps native run discovery connected in each task filter",async Page=>{
 const element=await Page({searchParams:Promise.resolve({})});expect(element.props.nativeRunsAvailable).toBe(true);
});
