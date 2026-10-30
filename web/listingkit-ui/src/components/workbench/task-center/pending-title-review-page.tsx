import { BusinessTaskPage } from "../ai-workbench/task-page";
import type {NativeRunDestinations} from "@/lib/agent-run-destination";

export function PendingTitleReviewPageContent({ available, initialProposalId, nativeRunsAvailable = false, nativeRunDestinations }: { available: boolean; initialProposalId?: string; nativeRunsAvailable?:boolean; nativeRunDestinations?:NativeRunDestinations }) {
  return <BusinessTaskPage mode="pending" productReviewAvailable={available} nativeRunsAvailable={nativeRunsAvailable} nativeRunDestinations={nativeRunDestinations} initialProposalId={initialProposalId} />;
}
