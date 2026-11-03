"use client";
import { useWorkbenchContext } from "@/components/providers/workbench-context-provider";
import { Button } from "@/components/ui/button";
import { researchLocation, researchNavigationHref, researchWorkflow, type ResearchCenter } from "@/lib/workbench/research-navigation";
import { ConsoleState } from "./console-page";
import { PrimaryIntroduction } from "./primary-introduction";

export function ResearchHomepage({ center, mode = "active", initialSearch = "" }: { center: ResearchCenter; mode?: "active" | "archived"; initialSearch?: string }) {
  const context = useWorkbenchContext();
  if (context.error || context.blockingError) return <ConsoleState kind="error" title="企业上下文不可用"><Button onClick={() => void context.retry()}>重新确认</Button></ConsoleState>;
  if (context.isLoading || context.isSwitching) return <ConsoleState kind="loading" title="正在读取工作台..." />;
  if (context.selectionRequired || !context.user || !context.effectiveOrganization) return <ConsoleState kind="unavailable" title="请先选择可访问的企业" />;

  const enabled = !!context.marketResearchAvailable && !!context.projectCenterAvailable;
  const projects = enabled && context.permissions.includes("workbench.project.read");
  const evidence = projects && context.permissions.includes("workbench.collection.read");
  const available = center === "research" ? [projects, evidence, evidence] : [evidence, evidence, evidence];
  const notice = !enabled ? "当前应用尚未启用研究选品，流程说明可查看，业务入口暂未开放。" : !projects ? "当前身份没有项目查看权限，请联系企业管理员核对。" : !evidence ? "当前身份没有商品资料查看权限，资料研究与候选评审入口暂不可用。" : undefined;
  const root = researchWorkflow(center).root;
  const location = researchLocation(root, new URLSearchParams({ mode, search: initialSearch }));
  const steps = center === "research" ? ["projects", "sources", "summary"] : ["candidates", "analysis", "transfer"];
  return <PrimaryIntroduction kind={center} available={available} notice={notice} entryHrefs={steps.map(step => researchNavigationHref(`${root}?step=${step}`, location))} />;
}
