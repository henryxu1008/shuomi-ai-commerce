import type { ReactNode } from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

const steps = ["选择商品", "选择目标店铺", "配置规则与智能体", "确认提交"];

// The enabled and preparation-only routes share the same page composition.
// Typography, controls and theme treatment come from the existing UI primitives.
export function ListingConfiguration({ children }: { children: ReactNode }) {
  return <section aria-label="四步上传配置" className="space-y-3">
    <Card className="p-4"><ol className="flex flex-wrap gap-3" aria-label="配置顺序">{steps.map((step, index) => <li key={step}>{index > 0 ? <span aria-hidden="true">→ </span> : null}{step}</li>)}</ol></Card>
    <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">{children}</div>
  </section>;
}

export function ListingConfigurationStep({ step, description, children }: { step: 1 | 2 | 3 | 4; description: string; children: ReactNode }) {
  return <Card className="flex min-h-[420px] flex-col">
    <CardHeader className="gap-3 p-4">
      <CardTitle><h2>{step}. {steps[step - 1]}</h2></CardTitle>
      <CardDescription>{description}</CardDescription>
    </CardHeader>
    <CardContent className="flex flex-1 flex-col gap-4 p-4 pt-0">{children}</CardContent>
  </Card>;
}

export function ListingContinuationNotice({ availability }: { availability?: string }) {
  return <Card>
    <CardHeader className="gap-3 p-4">
      <CardTitle><h2>补全与审核后，继续同一次上传</h2></CardTitle>
      <CardDescription>缺资料 → 待补全 → 确认补全并继续；有 AI 候选 → 审核并应用 → 确认继续上传。</CardDescription>
      <CardDescription>原任务固定商品、店铺与配置。成功项跳过，结果待核实项仅核实原操作，不重新发送。</CardDescription>
      {availability ? <CardDescription>{availability}</CardDescription> : null}
    </CardHeader>
  </Card>;
}
