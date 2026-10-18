import { BusinessTaskPage } from "../ai-workbench/task-page";

export function PendingTitleReviewPageContent({ available, initialProposalId }: { available: boolean; initialProposalId?: string }) {
  return <BusinessTaskPage mode="pending" productReviewAvailable={available} initialProposalId={initialProposalId} />;
}
