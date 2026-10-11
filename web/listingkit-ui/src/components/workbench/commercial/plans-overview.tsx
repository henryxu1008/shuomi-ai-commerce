import Image from "next/image";
import Link from "next/link";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import type { CommercialOverview } from "@/lib/api/commercial";
import styles from "../console/primary-overview.module.css";

const entries = [
  ["套餐方案", "计费方案", "查看店铺基础服务与按需计费规则", "/workbench/plans/options", "teal"],
  ["我的权益", "当前权益", "查看店铺服务状态与已开通能力", "/workbench/plans/entitlements", "blue"],
  ["用量明细", "实际消耗", "查看AI点数、数据与服务使用记录", "/workbench/plans/usage", "purple"],
  ["充值中心", "资源充值", "企业钱包充值与资源购买", "/workbench/plans/top-up", "orange"],
  ["账单与订单", "费用记录", "查看充值、购买与续费订单", "/workbench/plans/orders", "green"],
];
export function PlansOverview({ data }: { data: CommercialOverview }) {
  const stores = data.store_services.state === "available" ? data.store_services.value : null;
  const balance = (type: string) => {
    const value = data.resources.state === "available" ? data.resources.value.resources.find(r => r.resource_type === type) : null;
    return value?.state === "recorded" ? BigInt(value.available).toLocaleString("zh-CN") : value?.state === "not_recorded" ? "尚无余额记录" : "暂不可用";
  };
  return <div className={`${styles.stack} ${styles.plans}`}>
    <Card className={`${styles.panel} ${styles.plan}`}><div className={styles.heroHeader}><div><p className={styles.label}>当前计费方式</p><h2>{data.base_plan.name}</h2><p className={styles.description}>按需使用 · 店铺服务按30天周期订购；AI点数与数据资源预付使用。</p></div><div className={styles.actions}><Button asChild variant="outline" size="sm"><Link href="/workbench/plans/entitlements" prefetch={false}>查看我的权益</Link></Button><Button asChild size="sm"><Link href="/workbench/plans/top-up" prefetch={false}>进入钱包</Link></Button></div></div>
      <div className={styles.metrics}><div data-tone="blue"><span>店铺服务</span><strong>{stores ? `${stores.active} 家生效中` : "暂不可用"}</strong><small>{stores ? `${stores.expiring_soon} 家7天内到期` : "未取得店铺服务统计"}</small></div><div data-tone="teal"><span>AI点数余额</span><strong>{balance("ai_point")}</strong><small>企业可用余额 · 不按月清零</small></div><div data-tone="purple"><span>数据余额（条）</span><strong>{balance("data_row")}</strong><small>企业可用余额 · 按成功保存结果扣减</small></div></div>
    </Card>
    <header className={styles.sectionHeading}><h2>管理入口</h2><p>方案、权益、用量、充值和账单分别管理，避免信息混在一起。</p></header>
    <div className={styles.entriesFive}>{entries.map(([title, subtitle, description, href, tone]) => <Card key={href} className={styles.entry} data-tone={tone}><h3>{title}</h3><p className={styles.entrySubtitle}>{subtitle}</p><p className={styles.description}>{description}</p><Button asChild variant="outline" size="sm"><Link href={href} prefetch={false}>{title === "套餐方案" ? "购买资源" : "进入"}</Link></Button></Card>)}</div>
    <div className={styles.columns}>
      <Card className={styles.panel}><header className={styles.heroHeader}><div><h2>近30天用量</h2><p className={styles.description}>周期汇总尚未接入，完整消费记录进入「用量明细」。</p></div><Link className={styles.textLink} href="/workbench/plans/usage" prefetch={false}>查看明细 →</Link></header><div className={styles.usage}>{[["AI点数", balance("ai_point"), "teal"], ["数据服务", balance("data_row"), "purple"]].map(([title, value, tone]) => <div key={title} data-tone={tone}><div><strong>{title}</strong><span>已使用 —</span></div><div className={styles.track} aria-label={`${title}近30天用量未接入`} /><small>当前余额 {value} · 未取得近30天用量汇总</small></div>)}<div data-tone="blue"><div><strong>店铺服务</strong><span>{stores ? `当前${stores.active}家生效中` : "暂不可用"}</span></div><small>{stores ? `${stores.expiring_soon}家7天内到期` : "未取得店铺服务统计"}</small></div></div></Card>
      <Card className={styles.panel}><h2>费用与提醒</h2><p className={styles.description}>需要处理的权益与费用事项</p><div className={styles.notices}><div><Image src="/figma/primary-homepages/plans-imgEllipse.svg" width={8} height={8} alt="" /><div><strong>店铺服务</strong><p>{stores ? `${stores.expiring_soon}家店铺7天内到期` : "到期统计暂不可用"}</p></div><Link href="/workbench/plans/entitlements" prefetch={false}>查看</Link></div><div><Image src="/figma/primary-homepages/plans-imgEllipse1.svg" width={8} height={8} alt="" /><div><strong>资源余额</strong><p>按业务需要购买AI点数与数据额度</p></div><Link href="/workbench/plans/top-up" prefetch={false}>充值</Link></div></div><p className={styles.description}>最近订单请进入账单查看。</p><Link className={styles.textLink} href="/workbench/plans/orders" prefetch={false}>查看账单 →</Link></Card>
    </div>
    <details className={styles.description}><summary>计费规则与观察时间</summary><p>店铺1期为30天；购买后先完成平台连接，再开通或续费。企业已购资源不按月清零；成员AI月度上限按UTC自然月管理。</p><p>观察时间：{data.observed_at.replace("T", " ").replace("Z", " UTC")}</p></details>
  </div>;
}
