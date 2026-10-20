import { afterEach, expect, it, vi } from "vitest";
import AgentDetailPage from "./page";
vi.mock("next/navigation", () => ({ notFound: () => { throw new Error("not found"); } }));
vi.mock("@/components/workbench/agents/agent-page", () => ({ AgentPage: () => null }));
vi.mock("next/server", () => ({ connection: async () => {} }));
afterEach(() => vi.unstubAllEnvs());

it("opens the registered content agent through its actual app route", async () => {
  const page = await AgentDetailPage({ params: Promise.resolve({ agentId: "product.content.agent" }) });
  expect(page.props.id).toBe("product.content.agent");
  expect(page.props.mode).toBe("mine");
});
it.each([["false","true",false],["true","false",false],["true","true",true]])("binds POD continuation to the original serving assertions %s %s", async (pod, market, available) => {
  vi.stubEnv("LISTINGKIT_SDS_POD_ENABLED",pod);
  vi.stubEnv("LISTINGKIT_SUPPLY_MARKET_ENABLED",market);
  const page = await AgentDetailPage({params:Promise.resolve({agentId:"pod.design.agent"})});
  expect(page.props.podAvailable).toBe(available);
});
it("keeps arbitrary agent ids out of the registered configuration page", async () => {
  await expect(AgentDetailPage({ params: Promise.resolve({ agentId: "other.agent" }) })).rejects.toThrow("not found");
});
