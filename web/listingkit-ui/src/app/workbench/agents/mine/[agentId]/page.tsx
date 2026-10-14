import { notFound } from "next/navigation";
import { AgentPage } from "@/components/workbench/agents/agent-page";
import { registeredAgentIdSchema } from "@/lib/contracts/agent-configuration";
export default async function Page({
  params,
}: {
  params: Promise<{ agentId: string }>;
}) {
  const { agentId } = await params;
  if (!registeredAgentIdSchema.safeParse(agentId).success) notFound();
  return <AgentPage mode="mine" id={agentId} />;
}
