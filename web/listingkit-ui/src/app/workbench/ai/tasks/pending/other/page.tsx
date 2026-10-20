import { connection } from "next/server";
import { PendingTitleReviewPageContent } from "@/components/workbench/task-center/pending-title-review-page";
import { configuredProductReviewOrigin } from "@/lib/server/product-title-review-request";
import {isAgentConfigurationAvailable} from "@/lib/server/agent-configuration-origin";
export default async function Page({ searchParams }: { searchParams: Promise<{ proposal_id?: string | string[] }> }) {
  await connection();
  const query = await searchParams;
  return <PendingTitleReviewPageContent available={!!configuredProductReviewOrigin()} nativeRunsAvailable={isAgentConfigurationAvailable()} initialProposalId={typeof query.proposal_id === "string" ? query.proposal_id : undefined} />;
}
