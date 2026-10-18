import {configuredAgentConfigurationOrigin} from "@/lib/server/agent-configuration-origin";
import { BusinessTaskPage } from "@/components/workbench/ai-workbench/task-page";
import { configuredProductReviewOrigin } from "@/lib/server/product-title-review-request";
import { isSheinRecordsAvailable } from "@/lib/server/shein-records-availability";
import { connection } from "next/server";

export default async function Page({ searchParams }: { searchParams: Promise<{ proposal_id?: string | string[] }> }) {
  await connection();
  const query = await searchParams;
  return <BusinessTaskPage nativeRunsAvailable={configuredAgentConfigurationOrigin() !== null} productReviewAvailable={configuredProductReviewOrigin() !== null} sheinRecordsAvailable={isSheinRecordsAvailable()}
    initialProposalId={typeof query.proposal_id === "string" ? query.proposal_id : undefined} />;
}
