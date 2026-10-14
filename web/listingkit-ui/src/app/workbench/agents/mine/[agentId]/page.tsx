import { connection } from "next/server";
import { notFound } from "next/navigation";
import { AgentPage } from "@/components/workbench/agents/agent-page";
import { registeredAgentIdSchema } from "@/lib/contracts/agent-configuration";
export default async function Page({
  params,
}: {
  params: Promise<{ agentId: string }>;
}) {
  await connection();
  const { agentId } = await params;
  if (!registeredAgentIdSchema.safeParse(agentId).success) notFound();
  return <AgentPage mode="mine" id={agentId} podAvailable={process.env.LISTINGKIT_SDS_POD_ENABLED === "true" && process.env.LISTINGKIT_SUPPLY_MARKET_ENABLED === "true"} />;
}
