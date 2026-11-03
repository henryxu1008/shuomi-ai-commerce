import { researchAnalysisInput } from "@/lib/contracts/research-analysis";

export type ResearchCenter = "research" | "selection";
export const researchWorkflows = [
  {
    center: "research", label: "市场调研", root: "/workbench/data/research",
    phases: [
      { id: "planning", label: "调研准备", steps: [{ id: "projects", label: "调研项目", section: "research-project" }] },
      { id: "sources", label: "资料收集", steps: [{ id: "sources", label: "收集与整理资料", section: "research-sources" }] },
      { id: "research", label: "竞品研究", steps: [
        { id: "evidence", label: "查看商品资料", section: "research-evidence" },
        { id: "summary", label: "比较重点商品", section: "research-summary" },
      ] },
    ],
  },
  {
    center: "selection", label: "智能选品", root: "/workbench/supply/selection",
    phases: [
      { id: "comparison", label: "商品比较", steps: [
        { id: "candidates", label: "重点商品", section: "research-evidence" },
        { id: "comparison", label: "价格与特点比较", section: "research-summary" },
      ] },
      { id: "analysis", label: "分析建议", steps: [{ id: "analysis", label: "AI建议与记录", section: "research-analysis" }] },
      { id: "decisions", label: "选品决定", steps: [{ id: "decisions", label: "评估与决定", section: "research-decisions" }] },
      { id: "transfer", label: "后续准备", steps: [{ id: "transfer", label: "确认商品并转入", section: "research-transfer" }] },
    ],
  },
] as const;

export function researchWorkflow(center: ResearchCenter) {
  return researchWorkflows.find(workflow => workflow.center === center)!;
}

export function researchStep(center: ResearchCenter, value?: string) {
  for (const phase of researchWorkflow(center).phases) {
    for (const step of phase.steps) if (step.id === value) return { ...step, phase: phase.label };
  }
}

export function researchLocation(pathname: string, query: URLSearchParams) {
  const workflow = researchWorkflows.find(({ root }) => pathname === root ||
    pathname.startsWith(root + "/") && /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/.test(pathname.slice(root.length + 1)));
  if (!workflow) return;
  const values = query.getAll("step"), step = values.length === 1 ? researchStep(workflow.center, values[0]) : undefined;
  return { workflow, pathname, query, step, projectId: pathname === workflow.root ? undefined : pathname.slice(workflow.root.length + 1) };
}
export type ResearchLocation = NonNullable<ReturnType<typeof researchLocation>>;

// Only navigation context is carried. The original readers still validate exact
// observations, authorization and revisions before any business operation.
export function researchNavigationHref(href: string, location?: ResearchLocation) {
  if (!location || !href.startsWith(location.workflow.root + "?step=")) return href;
  const query = new URLSearchParams(href.split("?")[1]);
  const mode = location.query.getAll("mode");
  if (mode.length === 1 && (mode[0] === "active" || mode[0] === "archived")) query.set("mode", mode[0]);
  const search = location.query.getAll("search");
  if (search.length === 1 && search[0].length <= 512) query.set("search", search[0]);
  if (location.projectId) {
    const samples = location.query.getAll("sample");
    if (samples.length === 1 && samples[0].length <= 16384) {
      try {
        if (researchAnalysisInput.safeParse({ candidates: JSON.parse(samples[0]) }).success) query.set("sample", samples[0]);
      } catch { /* Invalid selection never becomes navigation context. */ }
    }
    const after = location.query.getAll("candidateAfter");
    if (after.length === 1 && /^[0-9a-f-]{36}$/.test(after[0])) query.set("candidateAfter", after[0]);
  }
  return location.pathname + "?" + query;
}
