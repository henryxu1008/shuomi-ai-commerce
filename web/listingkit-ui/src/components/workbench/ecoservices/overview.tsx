import Link from "next/link";
import { Card } from "@/components/ui/card";
import { ConsolePage } from "@/components/workbench/console/console-page";
import { NonPaymentNotice } from "./shared";
import "./overview.css";

const entries = [
  {
    title: "服务市场", badge: "01 · 服务市场", tone: "teal", href: "/workbench/services/market",
    description: "按企业或店铺服务类型查看已发布方案，了解服务范围与费用。",
    audience: "适合：查找标准服务方案", action: "浏览服务市场",
  },
  {
    title: "我的服务", badge: "02 · 服务管理", tone: "blue", href: "/workbench/services/mine",
    description: "查看已提交的服务需求，跟进报价、交付与售后状态。",
    audience: "适合：跟进服务与交付进度", action: "管理我的服务",
  },
  {
    title: "申请加入", badge: "03 · 服务商", tone: "purple", href: "/workbench/services/join",
    description: "服务商可提交企业与服务能力资料，审核通过后进入平台服务生态。",
    audience: "适合：专业服务机构入驻", action: "提交入驻申请",
  },
] as const;

export function EcoservicesOverview({ nonPaymentOnly = false }: { nonPaymentOnly?: boolean }) {
  const steps = [
    { title: "查找服务", description: "按企业或店铺服务类型查找合适方案", tone: "teal" },
    { title: "查看服务方案", description: "确认范围、周期、交付和费用", tone: "blue" },
    { title: "提交服务需求", description: nonPaymentOnly ? "服务开放后填写需求，等待服务方确认" : "填写需求并等待服务方确认", tone: "green" },
    { title: "跟进交付", description: "在我的服务查看节点与售后状态", tone: "purple" },
  ];

  return <ConsolePage className="eco-overview-page" breadcrumbs={[{ label: "生态服务" }]}
    title="连接专业服务资源，补齐跨境经营关键环节"
    description="围绕企业设立、品牌保护、店铺开通和日常运营，查找服务方案并跟进已提交需求。">
    <div className="eco-overview-content">
      <div className="eco-overview-entries">
        {entries.map(entry => <Link key={entry.href} href={entry.href} prefetch={false} className="eco-overview-entry" data-tone={entry.tone}>
          <Card className="eco-overview-entry-card">
            <span className="eco-overview-badge">{entry.badge}</span>
            <h2>{entry.title}</h2>
            <p>{entry.description}</p>
            <small>{entry.audience}</small>
            <span className="eco-overview-action">{entry.action} <span aria-hidden="true">→</span></span>
          </Card>
        </Link>)}
      </div>
      <section className="eco-overview-guide" aria-labelledby="eco-overview-guide-title">
        <div className="eco-overview-guide-heading">
          <h2 id="eco-overview-guide-title">如何开始使用</h2>
          <p>从查找服务到完成交付，只需四步</p>
        </div>
        <ol className="eco-overview-steps">
          {steps.map((step, index) => <li key={step.title} data-tone={step.tone}>
            <Card className="eco-overview-step-card">
              <span className="eco-overview-step-number">{String(index + 1).padStart(2, "0")}</span>
              <h3>{step.title}</h3>
              <p>{step.description}</p>
            </Card>
          </li>)}
        </ol>
      </section>
      <div className="eco-overview-notes">
        <Card className="eco-overview-note eco-overview-management">
          <small>服务管理</small>
          <h2>提交后在哪里跟进</h2>
          <p>已提交的服务需求统一进入“我的服务”，查看报价、交付节点和售后状态。</p>
          <strong>服务市场负责发现方案 · 我的服务负责跟进交付</strong>
        </Card>
        <Card className="eco-overview-note eco-overview-explanation">
          <small>服务说明</small>
          <h2>范围、周期和费用均提前说明</h2>
          <p>专业服务由审核通过的服务商提供，服务范围、交付周期和费用在详情中说明。</p>
          <strong>申请加入由专员审核企业资质与服务能力</strong>
        </Card>
      </div>
      {nonPaymentOnly ? <NonPaymentNotice /> : null}
    </div>
  </ConsolePage>;
}
