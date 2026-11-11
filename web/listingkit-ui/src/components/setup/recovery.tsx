import { useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { api, type Plan, type RecoveryPreview } from "./client";

const states: Record<string, string> = { prepared: "待确认", running: "正在执行", failed: "安装前未完成", unknown: "结果待核实", succeeded: "已核对完成" };
const steps: Record<string, string> = { check: "检查安装条件", download: "准备安装文件", build: "准备安装组件", start: "启动安装服务", verify: "检查运行结果", handoff: "提供登录资料" };
export function operationStatus(plan?: Plan) { return plan?.archived ? "安装记录已归档" : states[plan?.status || ""] || "正在读取操作记录"; }
function time(value?: string) { return !value || value.startsWith("0001-") ? "当时未记录" : new Date(value).toLocaleString("zh-CN", { hour12: false }); }
export function OperationFacts({ plan }: { plan: Plan }) {
  return <div className="space-y-3 text-sm leading-6">
    <p>{plan.mode === "fresh" ? "首次安装" : "功能设置"} · {time(plan.createdAt || plan.updatedAt)}</p>
    <p>最后步骤：{steps[plan.stage || ""] || "当时未记录"}</p>
    {(plan.status === "unknown" || plan.status === "failed") && <p>{plan.reason || "当时未记录失败步骤和原因，请先检查原操作。"}</p>}
    {plan.archived && <p>相关资源已清理；原执行结果保留，记录已移入历史。</p>}
    {(plan.attempts || []).length > 1 && <ol className="space-y-2 border-l border-border pl-4">{plan.attempts!.map(a => <li key={a.number}>第 {a.number + 1} 次尝试：{states[a.status] || "待核实"} · {steps[a.stage] || "当时未记录步骤"}{a.reason && <p className="text-muted-foreground">{a.reason}</p>}</li>)}</ol>}
    <details className="text-xs text-muted-foreground"><summary className="cursor-pointer">操作编号</summary><p className="mt-2 break-all">{plan.id}</p></details>
  </div>;
}
export function OperationCard({ plan, onOpen }: { plan: Plan; onOpen: () => void }) {
  return <article className="space-y-3 rounded-xl border border-border bg-card p-5"><div className="flex flex-wrap items-start justify-between gap-2"><h3 className="break-all font-semibold">{plan.name || "安装操作"}</h3><span className="text-sm text-muted-foreground">{operationStatus(plan)}</span></div><OperationFacts plan={plan}/><Button variant="outline" onClick={onOpen}>查看操作记录</Button></article>;
}
export function RecoveryActions({ plan, busy, run, onChange }: { plan: Plan; busy: boolean; run: (fn: () => Promise<void>) => Promise<void>; onChange: (p: Plan) => Promise<void> }) {
  const [subnet, setSubnet] = useState("");
  const [preview, setPreview] = useState<RecoveryPreview>();
  const [previewRevision, setPreviewRevision] = useState<number>();
  const visiblePreview = previewRevision === plan.revision ? preview : undefined;
  const heading = useRef<HTMLHeadingElement>(null), origin = useRef<HTMLButtonElement>(null);
  useEffect(() => { if (visiblePreview) heading.current?.focus(); }, [visiblePreview]);
  const prepare = (action: "archive" | "retry", button: HTMLButtonElement) => {
    origin.current = button;
    void run(async () => { const next = await api<RecoveryPreview>(`plans/${plan.id}/recovery-preview`, { action, ...(action === "retry" && subnet ? { networkSubnet: subnet } : {}) }); setPreviewRevision(plan.revision); setPreview(next); });
  };
  const confirm = () => {
    if (!visiblePreview) return;
    const current = visiblePreview;
    void run(async () => {
      try {
        if (current.action === "retry") await onChange({ ...plan, status: "running" });
        const next = await api<Plan>(`plans/${plan.id}/recovery-confirm`, { previewId: current.id });
        setPreview(undefined); await onChange(next);
      } finally { await onChange(await api<Plan>(`plans/${plan.id}`)); }
    });
  };
  return <section className="space-y-4 rounded-xl border border-border bg-card p-5">
    <h2 className="font-semibold">处理这次安装</h2>
    <p className="text-sm leading-6 text-muted-foreground">先检查再确认。检查不会重新安装；确认没有已有数据且原版本核对通过后，才允许重新尝试。已有内容会保留。</p>
    <label className="block space-y-2 text-sm"><span>重试的网络地址范围（可选）</span><Input aria-label="重试的网络地址范围" disabled={busy} value={subnet} placeholder="留空保留原设置" onChange={e => { setSubnet(e.target.value); setPreview(undefined); }}/></label>
    <div className="flex flex-wrap gap-3"><Button variant="outline" disabled={busy} onClick={e => prepare("retry", e.currentTarget)}>检查能否重新尝试</Button><Button variant="ghost" disabled={busy} onClick={e => prepare("archive", e.currentTarget)}>检查归档条件</Button></div>
    {visiblePreview && <div className="space-y-4 border-t border-border pt-5">
      <h2 ref={heading} tabIndex={-1} className="font-semibold outline-none">{visiblePreview.action === "archive" ? "确认归档前检查" : "确认重新尝试前检查"}</h2>
      {visiblePreview.changes.map((change, i) => <p className="break-words text-sm leading-6" key={i}>{change.label}：{change.before} → {change.after}</p>)}
      {visiblePreview.blocks.length > 0 && <ul className="space-y-2 rounded-lg bg-muted p-4 text-sm">{visiblePreview.blocks.map((block, i) => <li key={i}>{block}</li>)}</ul>}
      <p className="text-sm leading-6 text-muted-foreground">{visiblePreview.action === "archive" ? "仅将资源已清理的记录移入历史，保留原执行结果和记录。" : "继续使用原安装版本和全部空存储，保留前一次记录；不会自动再次执行。"}</p>
      <div className="flex flex-wrap gap-3"><Button variant="outline" disabled={busy} onClick={() => { setPreview(undefined); origin.current?.focus(); }}>返回安装记录</Button><Button disabled={busy || visiblePreview.blocks.length > 0} onClick={confirm}>{visiblePreview.action === "archive" ? "确认归档记录" : "确认重新尝试"}</Button></div>
    </div>}
  </section>;
}
