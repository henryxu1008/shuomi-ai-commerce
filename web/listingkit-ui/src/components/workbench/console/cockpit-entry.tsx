"use client";

import { useWorkbenchContext } from "@/components/providers/workbench-context-provider";
import { CockpitOverview } from "../cockpit/cockpit-overview";
import { ConsoleState } from "./console-page";

export function CockpitEntry() {
  const context = useWorkbenchContext();
  if (context.isLoading || context.isSwitching || context.selectionRequired || !context.effectiveOrganization || context.error || context.blockingError) {
    return <ConsoleState kind="loading" title="正在读取工作台..." />;
  }
  const scope = context.user && context.permissions.some(permission => /^workbench\.cockpit\.(goals|stores|alerts|advice)\.read$/.test(permission)) ? { userId: context.user.id, organizationId: context.effectiveOrganization.id } : undefined;
  return <CockpitOverview key={JSON.stringify([scope, context.permissions])} scope={scope} permissions={context.permissions} enabled={context.operationsCockpitAvailable} />;
}
