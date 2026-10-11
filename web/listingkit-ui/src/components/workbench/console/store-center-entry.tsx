"use client";
import { useWorkbenchContext } from "@/components/providers/workbench-context-provider";
import { ConsoleState } from "./console-page";
import { PrimaryIntroduction } from "./primary-introduction";
import { Button } from "@/components/ui/button";
export function StoreCenterEntry() {
  const context = useWorkbenchContext();
  if (context.error || context.blockingError) return <ConsoleState kind="error" title="企业上下文不可用"><Button onClick={() => void context.retry()}>重新确认</Button></ConsoleState>;
  if (context.isLoading || context.isSwitching) return <ConsoleState kind="loading" title="正在读取工作台..." />;
  if (context.selectionRequired || !context.effectiveOrganization) return <ConsoleState kind="unavailable" title="请先选择可访问的企业" />;
  const permissions = context.effectiveOrganization.permissions ?? [];
  const stores = permissions.includes("workbench.store.read");
  return <PrimaryIntroduction kind="stores" available={[stores, stores && context.storeObservationsAvailable && permissions.includes("workbench.store.products.read"), stores && context.storeObservationsAvailable && permissions.includes("workbench.store.orders.read")]} />;
}
