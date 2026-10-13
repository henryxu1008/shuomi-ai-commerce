"use client";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { editedContentValuesSchema, type ContentValue, type ProductContentProposal } from "@/lib/api/product-title-review";
import { contentSelectionLabel } from "@/lib/contracts/content-selection";
import styles from "./title-review.module.css";

export const contentValueLabels = { title: "标题", description: "描述", selling_points: "卖点", seo_keywords: "SEO关键词" } as const;
function listField(field: ContentValue["field"]) { return field === "selling_points" || field === "seo_keywords"; }
function ValueDisplay({ field, value }: ContentValue) {
  if (listField(field)) {
    try { const values: unknown = JSON.parse(value); if (Array.isArray(values) && values.every(v => typeof v === "string")) return values.length ? <ul>{values.map((v, i) => <li key={i}>{v}</li>)}</ul> : <p>暂无内容</p>; } catch {}
  }
  return <p className="whitespace-pre-wrap break-words">{value || "暂无内容"}</p>;
}
export function ContentComparison({ proposal: p }: { proposal: ProductContentProposal }) {
  const original = p.content.decisions?.[0]?.before ?? p.content.after;
  return <section aria-label="内容修改前后对比"><p>{contentSelectionLabel(p.content.selection)}</p>
    {p.content.after.map((after, i) => <section key={after.field} className={styles.comparison} aria-label={`${contentValueLabels[after.field]}修改`}>
      <h3>{contentValueLabels[after.field]}</h3>
      <div className={styles.before}><h4>修改前</h4><ValueDisplay {...p.content.before[i]} /></div>
      <div className={styles.after}><h4>当前建议</h4><ValueDisplay {...after} /></div>
      {original[i]?.value !== after.value && <details><summary>查看原始内容建议</summary><ValueDisplay {...original[i]} /></details>}
    </section>)}
  </section>;
}
export function ContentEditor({ values, pending, onSave, onCancel }: { values: ContentValue[]; pending: boolean; onSave: (values: ContentValue[]) => void; onCancel: () => void }) {
  const [texts, setTexts] = useState(values.map(v => listField(v.field) ? (JSON.parse(v.value) as string[]).join("\n") : v.value));
  const [failure, setFailure] = useState("");
  return <form className={styles.comparison} onSubmit={event => {
    event.preventDefault(); if (pending) return;
    const candidate = values.map((v, i) => ({ field: v.field, value: listField(v.field) ? JSON.stringify(texts[i].split("\n")) : texts[i] }));
    const parsed = editedContentValuesSchema.safeParse(candidate);
    if (!parsed.success) { setFailure("请保留全部选中字段；文本和每条卖点、关键词不能为空或含首尾空白，标题与列表每条最多4096 UTF-8字节。"); return; }
    onSave(parsed.data);
  }}>
    {values.map((v, i) => <label key={v.field} className="block">编辑{contentValueLabels[v.field]}<Textarea value={texts[i]} disabled={pending} onChange={event => { setTexts(old => old.map((text, index) => index === i ? event.target.value : text)); setFailure(""); }} autoFocus={i === 0} />{listField(v.field) && <span className="text-sm">每行一条，保存完整列表。</span>}</label>)}
    <p>保存完整选中内容后需重新接受，再单独采用。语言与字段范围沿用原运行。</p>
    {failure && <p role="alert">{failure}</p>}
    <div className={styles.actions}><Button type="submit" disabled={pending}>保存为待审核提案</Button><Button type="button" variant="outline" disabled={pending} onClick={onCancel}>取消编辑</Button></div>
  </form>;
}
