import { BusinessTaskPage } from "../ai-workbench/task-page";

export function PendingTitleReviewPageContent({ available, initialProposalId, nativeRunsAvailable = false }: { available: boolean; initialProposalId?: string; nativeRunsAvailable?:boolean }) {
  return <BusinessTaskPage mode="pending" productReviewAvailable={available} nativeRunsAvailable={nativeRunsAvailable} initialProposalId={initialProposalId} />;
}
