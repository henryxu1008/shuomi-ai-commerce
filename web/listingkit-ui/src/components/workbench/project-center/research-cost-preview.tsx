"use client";

import { useEffect, useRef, useState, type FormEvent } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import {ResearchError} from "@/lib/api/market-research";
import {
  blankCostBasis, blankCostDraft, costFields,
  type CostBasis, type CostDraft, type CostField, type CostOutcome, type CostResult, type CostValue,
} from "@/lib/workbench/research-cost-preview";

const terms: Record<CostField, string> = {
  procurement: "采购与包装消耗", logistics: "物流和履约（含该情景退运）",
  platformFixed: "平台和支付固定费用", platformRate: "平台和支付费率",
  taxFixed: "卖家承担固定税费", taxRate: "卖家承担税率",
  advertising: "广告分摊", exchangeCost: "额外换汇成本",
};
export const costFieldLabels=terms;

const fields: Record<string, string> = {
  channel: "计划销售渠道", fulfillment: "履约方式", specification: "商品规格",
  quantity: "本次采购数量", price: "假设售价", refundRate: "全额退款概率",
  exchangeRate: "人工汇率", purchaseCurrency: "采购币种",
  salesBasis: "售价与退款概率", exchangeBasis: "汇率", completed: "正常成交", refunded: "全额退款",
  ...terms,
  source: "出处", asOf: "适用日期", basis: "依据",
};

function fieldLabel(path: string): string {
  return path.split(".").map(part => fields[part] ?? part).join(" · ");
}

function TextField({ label, value, onChange, type = "text", inputMode, maxLength = 120 }: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  type?: "date" | "text";
  inputMode?: "numeric" | "decimal";
  maxLength?: number;
}) {
  return <label className="grid min-w-0 gap-1">
    <span>{label}</span>
    <Input type={type} inputMode={inputMode} maxLength={maxLength} value={value}
      onChange={event => onChange(event.target.value)} />
  </label>;
}

function BasisFields({ label, value, onChange }: {
  label: string;
  value: CostBasis;
  onChange: (value: CostBasis) => void;
}) {
  return <div className="grid min-w-0 gap-3 sm:grid-cols-2">
    <TextField label={label + "出处"} value={value.source} maxLength={240}
      onChange={source => onChange({ ...value, source })} />
    <TextField label={label + "适用日期"} value={value.asOf} type="date"
      onChange={asOf => onChange({ ...value, asOf })} />
  </div>;
}

function OutcomeFields({ label, value, purchaseCurrency, onChange }: {
  label: string;
  value: CostOutcome;
  purchaseCurrency: CostDraft["purchaseCurrency"];
  onChange: (value: CostOutcome) => void;
}) {
  function setCost(field: CostField, input: CostValue) {
    onChange({ ...value, costs: { ...value.costs, [field]: input } });
  }
  return <fieldset className="grid min-w-0 gap-3 rounded-md border border-border p-3">
    <legend className="px-1 font-medium">{label}费用假设</legend>
    <p className="text-sm text-muted-foreground">这一组费用共用以下出处与日期。单项依据不同可单独填写。未知留空，确认没有费用时填写 0。</p>
    <BasisFields label={label + "共用"} value={value.basis}
      onChange={basis => onChange({ ...value, basis })} />
    <div className="grid gap-4 sm:grid-cols-2">
      {costFields.map(field => {
        const input = value.costs[field];
        const percent = field === "platformRate" || field === "taxRate";
        const unit = percent ? "售价的 %" : (field === "procurement" ? purchaseCurrency : "USD") + "/件";
        const name = label + " " + terms[field];
        return <div key={field} className="grid content-start gap-2">
          {input.kind === "value"
            ? <TextField label={name + "（" + unit + "）"} value={input.value}
                inputMode="decimal" maxLength={16}
                onChange={amount => setCost(field, { ...input, value: amount })} />
            : <TextField label={name + "不适用原因"} value={input.reason} maxLength={240}
                onChange={reason => setCost(field, { ...input, reason })} />}
          {field !== "procurement" && <label className="flex items-center gap-2 text-sm">
            <input type="checkbox" checked={input.kind === "not_applicable"}
              onChange={event => setCost(field, event.target.checked
                ? { kind: "not_applicable", reason: "" } : { kind: "value", value: "" })} />
            {name}不适用
          </label>}
          {input.kind === "value" && <details className="text-sm">
            <summary className="cursor-pointer">{name}的单项依据{input.basis ? "（已选）" : ""}</summary>
            <label className="my-2 flex items-center gap-2">
              <input type="checkbox" checked={!!input.basis}
                onChange={event => setCost(field, { ...input, basis: event.target.checked ? blankCostBasis() : undefined })} />
              {name}使用单独出处
            </label>
            {input.basis && <BasisFields label={name} value={input.basis}
              onChange={basis => setCost(field, { ...input, basis })} />}
          </details>}
        </div>;
      })}
    </div>
  </fieldset>;
}

export function PreviewResult({ result }: { result: CostResult }) {
  if (result.status === "UNSUPPORTED_COST_RULE") {
    return <p role="status">暂不支持阶梯、最低收费或其他超出本次范围的费用规则，完整利润和保本价待核实。</p>;
  }
  if (result.status !== "COMPUTABLE") {
    return <div role={result.status === "INVALID_INPUT" ? "alert" : "status"}>
      <p className="font-medium">{result.status === "INVALID_INPUT" ? "有输入需要更正" : "资料仍有缺项"}</p>
      <ul className="list-disc pl-5">{result.fields.map(field => <li key={field}>{fieldLabel(field)}</li>)}</ul>
      <p>完整贡献利润和保本价待核实。</p>
    </div>;
  }
  return <section aria-label="参考试算结果" role="status" className="grid gap-2">
    <h5 className="font-medium">按当前假设估算</h5>
    <dl className="grid gap-2 sm:grid-cols-2">
      <div><dt>期望商品毛利</dt><dd>USD {result.gross} / 件</dd></div>
      <div><dt>期望贡献利润</dt><dd>USD {result.contribution} / 次销售尝试
        {result.contributionSign === "negative" && "（亏损）"}</dd></div>
      <div><dt>参考保本售价</dt><dd>{result.breakEven === null ? "当前假设下无保本价" : "USD " + result.breakEven}</dd></div>
      {result.completedContribution !== null && <div><dt>正常成交贡献</dt><dd>USD {result.completedContribution}</dd></div>}
      {result.refundedContribution !== null && <div><dt>全额退款贡献</dt><dd>USD {result.refundedContribution}</dd></div>}
    </dl>
    <p className="text-sm text-muted-foreground">贡献利润未扣企业固定开支。人工决定和合规核对需单独完成。</p>
  </section>;
}

export function ResearchCostPreview({ calculate,onEdit,onSave,initialDraft,disabled=false }: {
  calculate: (draft:CostDraft,signal: AbortSignal) => Promise<CostResult>;
  onEdit?:()=>void;
  onSave?:(draft:CostDraft)=>Promise<void>;
  initialDraft?:CostDraft;
  disabled?:boolean;
}) {
  const [draft, setDraft] = useState(initialDraft??blankCostDraft);
  const [result, setResult] = useState<CostResult>();
  const [message, setMessage] = useState("");
  const [checking, setChecking] = useState(false);
  const current = useRef<AbortController | null>(null);
  useEffect(() => () => current.current?.abort(), []);

  function clearPreview() {
    current.current?.abort();
    current.current = null;
    setChecking(false);
    setResult(undefined);
    setMessage("");
  }

  function edit<K extends keyof CostDraft>(field: K, value: CostDraft[K]) {
    onEdit?.();
    clearPreview();
    setDraft(previous => ({ ...previous, [field]: value }));
  }

  function changeCurrency(purchaseCurrency: CostDraft["purchaseCurrency"]) {
    onEdit?.();
    clearPreview();
    const clearProcurement = (outcome: CostOutcome): CostOutcome => ({
      ...outcome, costs: { ...outcome.costs, procurement: { kind: "value", value: "" } },
    });
    setDraft(previous => ({
      ...previous, purchaseCurrency, exchangeRate: "", exchangeBasis: blankCostBasis(),
      completed: clearProcurement(previous.completed), refunded: clearProcurement(previous.refunded),
    }));
  }

  async function preview(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (current.current) return;
    const request = new AbortController();
    current.current = request;
    setResult(undefined);
    setMessage("");
    setChecking(true);
    try {
      const response = await calculate(draft,request.signal);
      if (request.signal.aborted) return;
      setResult(response);
    } catch (error) {
      if (request.signal.aborted)return;
      if(error instanceof ResearchError&&error.code==="INVALID_REQUEST"){
        setResult({status:"INVALID_INPUT",fields:error.fields??[]});
        setMessage("输入格式或数值需要更正，请检查金额、退款概率及日期；已保留输入。");return;
      }
      const transient=error instanceof ResearchError&&(["DEPENDENCY_UNAVAILABLE","DEADLINE_EXCEEDED"].includes(error.code)||error.code==="INVALID_UPSTREAM_RESPONSE"&&!!error.status&&error.status>=500);
      if(error instanceof ResearchError&&!transient||error instanceof Error&&["REVISION_CONFLICT","FORBIDDEN","NOT_FOUND","PROJECT_ARCHIVED"].includes(error.message)){
        setDraft(blankCostDraft());
        setMessage("原商品资料已变化或暂不可读，请刷新商品资料后重新填写。");
      }else setMessage("暂时无法核对原商品资料。已保留输入，请重新查看试算。");
    } finally {
      if (current.current === request) {
        current.current = null;
        setChecking(false);
      }
    }
  }

  const refundIsZero = /^0(?:\.0{1,2})?$/.test(draft.refundRate);
  const refundIsCertain = /^100(?:\.0{1,2})?$/.test(draft.refundRate);

  return <form aria-label="经营假设和参考试算" className="grid min-w-0 gap-4 border-t border-border pt-4"
    onSubmit={preview}>
    <p className="text-sm text-muted-foreground">美国市场 · USD 销售 · 每单一件。请填写自己的经营假设，商品资料的展示价不会带入。未保存的输入只留在当前页面。</p>
    <fieldset disabled={disabled} className="grid min-w-0 gap-4">
    <div className="grid min-w-0 gap-3 sm:grid-cols-2">
      <TextField label="计划销售渠道" value={draft.channel} onChange={value => edit("channel", value)} />
      <TextField label="履约方式" value={draft.fulfillment} onChange={value => edit("fulfillment", value)} />
      <TextField label="商品规格" value={draft.specification} onChange={value => edit("specification", value)} />
      <TextField label="本次采购数量（件）" value={draft.quantity} inputMode="numeric" maxLength={7}
        onChange={value => edit("quantity", value)} />
    </div>
    <label className="grid gap-1">费用规则
      <Select value={draft.rule} onChange={event => edit("rule", event.target.value === "fixed_percent" ? "fixed_percent" : "unsupported")}>
        <option value="fixed_percent">单件固定费用与按售价比例费用</option>
        <option value="unsupported">含阶梯、最低收费或其他规则</option>
      </Select>
    </label>
    {draft.rule === "fixed_percent" && <>
      <fieldset className="grid min-w-0 gap-3 rounded-md border border-border p-3">
        <legend className="px-1 font-medium">售价与退款假设</legend>
        <div className="grid gap-3 sm:grid-cols-2">
          <TextField label="假设售价（USD/件）" value={draft.price} inputMode="decimal" maxLength={16}
            onChange={value => edit("price", value)} />
          <TextField label="全额退款概率（%）" value={draft.refundRate} inputMode="decimal" maxLength={6}
            onChange={value => edit("refundRate", value)} />
        </div>
        <p className="text-sm text-muted-foreground">售价填写优惠后、剔除代收税的商品收入。全额退款时商品收入按 0 计，不再另扣退款金额。</p>
        <BasisFields label="售价与退款概率共用" value={draft.salesBasis}
          onChange={value => edit("salesBasis", value)} />
      </fieldset>
      <label className="grid gap-1">采购币种
        <Select value={draft.purchaseCurrency} onChange={event => changeCurrency(event.target.value === "CNY" ? "CNY" : "USD")}>
          <option value="USD">USD（美元）</option>
          <option value="CNY">CNY（人民币）</option>
        </Select>
      </label>
      {draft.purchaseCurrency === "CNY" && <fieldset className="grid min-w-0 gap-3 rounded-md border border-border p-3">
        <legend className="px-1 font-medium">手填采购换算</legend>
        <TextField label="人工汇率（1 CNY 折合多少 USD）" value={draft.exchangeRate}
          inputMode="decimal" maxLength={16} onChange={value => edit("exchangeRate", value)} />
        <BasisFields label="人工汇率" value={draft.exchangeBasis} onChange={value => edit("exchangeBasis", value)} />
      </fieldset>}
      {!refundIsCertain && <OutcomeFields label="正常成交" value={draft.completed} purchaseCurrency={draft.purchaseCurrency}
        onChange={value => edit("completed", value)} />}
      {draft.refundRate && !refundIsZero && <OutcomeFields label="全额退款" value={draft.refunded} purchaseCurrency={draft.purchaseCurrency}
        onChange={value => edit("refunded", value)} />}
    </>}
    <div className="flex flex-wrap gap-2"><Button type="submit" variant="secondary" disabled={checking}>{checking ? "正在核对商品资料…" : "查看参考试算"}</Button>
    {onSave&&<Button type="button" disabled={checking||!result} onClick={async()=>{setChecking(true);try{await onSave(draft)}finally{setChecking(false)}}}>保存本次评估</Button>}</div>
    </fieldset>
    {message && <p role="alert">{message}</p>}
    {result && <PreviewResult result={result} />}
  </form>;
}
