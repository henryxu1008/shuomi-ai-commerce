"use client";
import { useWorkbenchContext } from "@/components/providers/workbench-context-provider";
import { ConsoleState } from "./console-page";
import { PrimaryIntroduction } from "./primary-introduction";
export function StoreCenterEntry() {
  const context = useWorkbenchContext();
  if (context.isLoading || context.isSwitching || context.selectionRequired || !context.effectiveOrganization || context.error || context.blockingError) return <ConsoleState kind="loading" title="正在读取工作台..." />;
  return <PrimaryIntroduction kind="stores" available={[true, context.storeObservationsAvailable, context.storeObservationsAvailable]} />;
}
