"use client";

import { useId } from "react";
import type { CatalogEntry } from "@/lib/contracts/agent-configuration";
import styles from "./agent-capabilities.module.css";

const names: Record<string, string> = {
  "text.generate": "文本生成",
  "image.generate": "图片生成",
  "product.source-evidence": "采集证据读取",
  "listing.materials.inspect": "草稿、店铺规则与已批准资产读取",
  "product.selection.inspect": "候选商品事实读取与对比",
  "shop.operations-diagnosis.inspect": "店铺经营事实读取与诊断",
  "knowledge.context": "企业知识引用",
  "platform.write": "平台写入",
};
const supportLabels = { REQUIRED: "必需", OPTIONAL: "可选", NOT_SUPPORTED: "不支持" };
const readinessLabels = {
  AVAILABLE: "可用",
  NEEDS_CONFIGURATION: "需要配置",
  REQUIRES_AUTHORIZATION: "需要授权",
  UNAVAILABLE: "不可用",
  NOT_SUPPORTED: "不支持",
};
type Capability = CatalogEntry["capabilities"][number];

export function AgentCapabilityPanel({ entry }: { entry: CatalogEntry }) {
  const id = useId();
  const generation = entry.capabilities.filter(cap => cap.id === "text.generate" || cap.id === "image.generate");
  const context = entry.capabilities.filter(cap => cap.id !== "text.generate" && cap.id !== "image.generate");
  function row(cap: Capability) {
    const readiness = cap.support === "NOT_SUPPORTED" ? "NOT_SUPPORTED" : cap.readiness;
    const name = cap.id === "product.source-evidence" && entry.agent.agentId === "product.image.agent"
      ? "商品素材读取" : names[cap.id];
    return <section key={cap.id} aria-labelledby={`${id}-${cap.id}`} className={styles.capability}>
      <div className={styles.copy}>
        <h3 id={`${id}-${cap.id}`}>{name} · {supportLabels[cap.support]}</h3>
        <p>{cap.reason || "执行前仍需确认当前权限与预算。"}</p>
      </div>
      <span className={styles.badge} data-readiness={readiness}>{readinessLabels[readiness]}</span>
    </section>;
  }
  return <section aria-label="当前能力" className={styles.panel}>
    <h2>当前能力</h2>
    {generation.length ? <div className={styles.generation}>{generation.map(row)}</div> : null}
    {context.length ? <div className={styles.context}>{context.map(row)}</div> : null}
    {!entry.capabilities.length ? <p className={styles.empty}>当前没有可展示的能力信息，请在原配置页确认。</p> : null}
    <p className={styles.boundary}>配置状态不代表执行许可，执行时重新确认当前权限与预算。模型配置不会自动开放知识读取或平台写入。</p>
  </section>;
}
