import Link from "next/link";
import { Card } from "@/components/ui/card";
import type { ConsolePrimaryCapabilities } from "@/lib/workbench/console-primary-entry";
import styles from "./data-services-overview.module.css";

const entries = [
  { badge: "01 · 数据能力", title: "数据市场", description: "选择实时数据抓取或数据集定制，查看支持的站点、字段和计费说明。", audience: "适合：查找并获取所需数据", action: "浏览数据市场", href: "/workbench/data/market", capability: "dataServicesAvailable", tone: "teal" },
  { badge: "02 · 调用管理", title: "API管理", description: "管理API密钥、调用权限、额度限制、调用日志和费用明细。", audience: "适合：控制权限、用量与费用", action: "管理API", href: "/workbench/data/api", capability: "dataServicesAvailable", tone: "blue" },
  { badge: "03 · 数据资产", title: "我的数据", description: "查看已保存的采集与调用结果，支持筛选、导出和重复使用。", audience: "适合：沉淀和复用业务数据", action: "查看我的数据", href: "/workbench/data/mine", capability: "productCollectionsAvailable", tone: "purple" },
] as const;

const steps = [
  { title: "选择数据服务", description: "在数据市场查找所需的数据能力", tone: "teal" },
  { title: "查看字段与价格", description: "确认支持的字段和计费方式；定制规格与报价由专员线下确认", tone: "blue" },
  { title: "测试并调用", description: "按API文档使用密钥获取数据，或在数据市场发起抓取", tone: "green" },
  { title: "保存到我的数据", description: "成功保存的结果可在我的数据中查看、导出和复用", tone: "purple" },
] as const;

export function DataServicesOverview({ capabilities }: { capabilities: Pick<ConsolePrimaryCapabilities, "dataServicesAvailable" | "productCollectionsAvailable"> }) {
  return <section className={styles.page} aria-labelledby="data-services-title">
    <header className={styles.header}>
      <p className={styles.eyebrow}>数据服务</p>
      <h1 id="data-services-title">让数据成为可调用、可沉淀、可复用的业务资产</h1>
      <p className={styles.subtitle}>采集、数据市场获取及API调用结果统一保存到“我的数据”中，便于集中管理和复用。</p>
    </header>

    <div className={styles.entries}>
      {entries.map(entry => {
        const content = <>
          <span className={styles.badge}>{entry.badge}</span>
          <h2>{entry.title}</h2>
          <p className={styles.description}>{entry.description}</p>
          <p className={styles.audience}>{entry.audience}</p>
          <span className={styles.action}>{capabilities[entry.capability] ? <>{entry.action} <span aria-hidden="true"> →</span></> : "暂未启用"}</span>
        </>;
        return capabilities[entry.capability]
          ? <Link key={entry.href} href={entry.href} prefetch={false} className={styles.entry} data-tone={entry.tone}>{content}</Link>
          : <Card key={entry.href} className={styles.entry} data-tone={entry.tone} data-state="unavailable">{content}</Card>;
      })}
    </div>

    <section className={styles.flow} aria-labelledby="data-services-start">
      <div className={styles.flowHeading}><h2 id="data-services-start">如何开始使用</h2><p>从选择数据到沉淀资产，只需四步</p></div>
      <ol className={styles.steps}>
        {steps.map((step, index) => <li key={step.title}><Card className={styles.step} data-tone={step.tone}>
          <span className={styles.number}>{String(index + 1).padStart(2, "0")}</span>
          <h3>{step.title}</h3><p>{step.description}</p>
        </Card></li>)}
      </ol>
    </section>

    <div className={styles.notes}>
      <Card className={`${styles.note} ${styles.dataFlow}`} data-tone="teal">
        <p className={styles.noteLabel}>数据流转</p><h2>三个模块如何协同</h2>
        <p className={styles.description}>采集与数据市场负责获取数据，API管理负责权限与用量，我的数据负责统一沉淀与复用。</p>
        <p className={styles.noteFooter}>选择数据 → 管理调用 → 沉淀数据资产</p>
      </Card>
      <Card className={styles.note} data-tone="purple">
        <p className={styles.noteLabel}>计费与安全</p><h2>调用前明确价格、额度和权限</h2>
        <p className={styles.description}>实时抓取与API按成功保存的数据条数消耗预付资源，支持密钥权限与额度限制；定制费用线下确认。</p>
        <p className={styles.noteFooter}>调用日志与费用明细可在API管理中查看</p>
      </Card>
    </div>
  </section>;
}
