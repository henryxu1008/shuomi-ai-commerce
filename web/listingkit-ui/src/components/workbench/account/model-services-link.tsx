"use client";

import { Suspense } from "react";
import Link from "next/link";
import { usePathname, useSearchParams } from "next/navigation";
import { useWorkbenchContext } from "@/components/providers/workbench-context-provider";
import { Button } from "@/components/ui/button";
import { modelServicesHref, type ModelServiceCategory } from "@/lib/contracts/model-services-navigation";

function BusinessSettingsLink({ category, organizationId }: { category: ModelServiceCategory; organizationId: string }) {
  const pathname = usePathname();
  const search = useSearchParams()?.toString();
  const returnTo = pathname ? pathname + (search ? "?" + search : "") : undefined;
  return <Button asChild variant="outline" size="sm"><Link prefetch={false} href={modelServicesHref(category, organizationId, returnTo)}>查看模型与服务</Link></Button>;
}

export function ModelServicesLink({ category }: { category: ModelServiceCategory }) {
  const context = useWorkbenchContext();
  const organizationId = context.effectiveOrganization?.id;
  if (!organizationId || context.isLoading || context.isSwitching || context.error || context.blockingError || context.selectionRequired) return null;
  const canConfigure = context.permissions.includes("workbench.agent.configure") ||
    context.permissions.includes("workbench.ai.credentials.manage");
  if (!canConfigure) return <p className="text-sm text-muted-foreground">请联系本企业管理员查看模型设置。</p>;
  return <Suspense fallback={null}><BusinessSettingsLink category={category} organizationId={organizationId} /></Suspense>;
}
