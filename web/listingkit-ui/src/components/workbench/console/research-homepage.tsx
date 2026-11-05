"use client";
import { useWorkbenchContext } from "@/components/providers/workbench-context-provider";
import { Button } from "@/components/ui/button";
import { researchLocation, researchNavigationHref, researchWorkflow, type ResearchCenter } from "@/lib/workbench/research-navigation";
import { ConsoleState } from "./console-page";
import { PrimaryIntroduction } from "./primary-introduction";

export function ResearchHomepage({ center, mode = "active", initialSearch = "" }: { center: ResearchCenter; mode?: "active" | "archived"; initialSearch?: string }) {
  const context = useWorkbenchContext();
  if (context.error || context.blockingError) return <ConsoleState kind="error" title="暂时无法确认当前企业"><Button onClick={() => void context.retry()}>重新确认</Button></ConsoleState>;
  if (context.isLoading || context.isSwitching) return <ConsoleState kind="loading" title="正在加载工作台..." />;
  if (context.selectionRequired || !context.user || !context.effectiveOrganization) return <ConsoleState kind="unavailable" title="请先选择可访问的企业" />;

  const enabled = !!context.marketResearchAvailable && !!context.projectCenterAvailable;
  const projects = enabled && context.permissions.includes("workbench.project.read");
  const evidence = projects && context.permissions.includes("workbench.collection.read");
  const available = center === "research" ? [projects, evidence, evidence] : [evidence, evidence, evidence];
  const notice = !enabled ? "当前暂未开放调研与选品功能，可先了解业务流程。" : !projects ? "当前账号还不能查看调研项目，请联系企业管理员核对权限。" : !evidence ? "当前账号没有商品资料查看权限，暂时不能查看资料或评估商品。" : undefined;
  const root = researchWorkflow(center).root;
  const location = researchLocation(root, new URLSearchParams({ mode, search: initialSearch }));
  const steps = center === "research" ? ["projects", "sources", "summary"] : ["candidates", "analysis", "transfer"];
  return <PrimaryIntroduction kind={center} available={available} notice={notice} entryHrefs={steps.map(step => researchNavigationHref(`${root}?step=${step}`, location))} />;
}
