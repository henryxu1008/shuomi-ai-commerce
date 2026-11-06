import { expectedTrialApprovalHeader, sheinTrialError, sheinTrialState, trialUUID } from "@/lib/contracts/shein-us-trial";
import type { ProjectScope } from "./project-center";
import { readBoundedStrictJSON } from "./strict-json-response";

export class SheinTrialError extends Error {
  constructor(readonly code: string) { super(code); }
}

export async function requestSheinTrial(
  scope: ProjectScope, projectId: string, signal: AbortSignal,
  action: "view" | "submit" | "reconcile", expectedApproval?: string,
) {
  const write = action !== "view";
  if (!trialUUID.safeParse(projectId).success || (write && !trialUUID.safeParse(expectedApproval).success))
    throw new SheinTrialError("INVALID_REQUEST");
  const path = `/api/workbench/research/projects/${projectId}/shein-us-trial${write ? `/${action}` : ""}`;
  const headers = new Headers({
    Accept: "application/json", "X-Expected-User-ID": scope.userId, "X-Expected-Organization-ID": scope.organizationId,
  });
  if (write) headers.set(expectedTrialApprovalHeader, expectedApproval!);
  try {
    const response = await fetch(path, {
      method: write ? "POST" : "GET", headers, signal, credentials: "same-origin", cache: "no-store", redirect: "error",
    });
    const payload = await readBoundedStrictJSON(response, 4096, signal);
    const error = sheinTrialError.safeParse(payload);
    if (!response.ok && error.success) throw new SheinTrialError(error.data.code);
    const parsed = sheinTrialState.parse(payload);
    if (response.status !== (write && parsed.state === "UNKNOWN" ? 409 : 200) ||
        (write && (parsed.state === "NOT_STARTED" || parsed.approvalId !== expectedApproval))) throw new Error();
    return parsed;
  } catch (error) {
    if (error instanceof SheinTrialError) throw error;
    throw new SheinTrialError(write ? "OUTCOME_UNKNOWN" : "SHEIN_TRIAL_UNAVAILABLE");
  }
}
