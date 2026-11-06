import { trialUUID } from "@/lib/contracts/shein-us-trial";

export function sheinTrialTarget() {
  const parts = process.env.LISTINGKIT_SHEIN_US_TRIAL_TARGET?.split("/");
  if (parts?.length !== 2) return null;
  const [project, approval] = parts;
  return trialUUID.safeParse(project).success && trialUUID.safeParse(approval).success
    ? { project, approval }
    : null;
}
