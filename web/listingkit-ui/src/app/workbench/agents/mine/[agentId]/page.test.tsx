import { expect, it, vi } from "vitest";
import AgentDetailPage from "./page";
vi.mock("next/navigation", () => ({ notFound: () => { throw new Error("not found"); } }));
vi.mock("@/components/workbench/agents/agent-page", () => ({ AgentPage: () => null }));

it("opens the registered content agent through its actual app route", async () => {
  const page = await AgentDetailPage({ params: Promise.resolve({ agentId: "product.content.agent" }) });
  expect(page.props.id).toBe("product.content.agent");
  expect(page.props.mode).toBe("mine");
});
it("keeps arbitrary agent ids out of the registered configuration page", async () => {
  await expect(AgentDetailPage({ params: Promise.resolve({ agentId: "other.agent" }) })).rejects.toThrow("not found");
});
