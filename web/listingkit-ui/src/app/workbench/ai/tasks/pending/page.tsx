import { BusinessTaskPage } from "@/components/workbench/ai-workbench/task-page";
import { configuredProductReviewOrigin } from "@/lib/server/product-title-review-request";
import { connection } from "next/server";

export default async function Page({ searchParams }: { searchParams: Promise<{ proposal_id?: string | string[] }> }) {
  await connection();
  const query = await searchParams;
  return <BusinessTaskPage mode="pending" productReviewAvailable={configuredProductReviewOrigin() !== null}
    initialProposalId={typeof query.proposal_id === "string" ? query.proposal_id : undefined} />;
}
