import Link from "next/link";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import type { CommercialOverview } from "@/lib/api/commercial";
import { EntitlementTime } from "./entitlement-time";
import entitlementStyles from "./entitlements.module.css";
import styles from "./commercial.module.css";
export function EntitlementsOverview({ data, showManagementLinks = true }: { data: CommercialOverview; showManagementLinks?: boolean }) {
  const stores =
    data.store_services.state === "available"
      ? data.store_services.value
      : null;
  const panelClass = showManagementLinks ? entitlementStyles.overviewPanel : styles.panel;
  const storeCount = (count: number) => showManagementLinks ? count.toLocaleString("zh-CN") : count;
  return (
    <div className={showManagementLinks ? entitlementStyles.overview : styles.stack}>
      <Card role="region" aria-label="基础方案" className={panelClass}>
        <h2>{data.base_plan.name}</h2>
        <p>无需购买或激活订阅；按实际需要购买店铺期数、AI 点数和数据条数。</p>
        <p>企业已购资源不按月清零；成员 AI 消费上限按 UTC 自然月管理。</p>
      </Card>
      <Card role="region" aria-label="店铺服务权益" className={panelClass}>
        <h2>店铺服务权益</h2>
        {stores ? (
          <dl className={showManagementLinks ? entitlementStyles.storeFacts : styles.facts}>
            <div>
              <dt>店铺记录</dt>
              <dd>{storeCount(stores.records)} 家</dd>
            </div>
            <div>
              <dt>生效服务</dt>
              <dd>{storeCount(stores.active)} 家</dd>
            </div>
            <div>
              <dt>已到期</dt>
              <dd>{storeCount(stores.expired)} 家</dd>
            </div>
            <div>
              <dt>7 天内到期</dt>
              <dd>{storeCount(stores.expiring_soon)} 家</dd>
            </div>
          </dl>
        ) : (
          <p role="alert">店铺服务统计暂不可用，请刷新重试。</p>
        )}
        {showManagementLinks ? (
          <div className={entitlementStyles.overviewActions}>
            <p>请先授权平台，再使用已分配的期数开通或续费店铺服务。每期 30 天。</p>
            <Button asChild variant="outline">
              <Link href="/workbench/stores" prefetch={false}>管理店铺服务</Link>
            </Button>
          </div>
        ) : (
          <p>购买期数不会自动连接店铺。请先授权平台，再用分配的期数开通或续费，每期 30 天。</p>
        )}
        {showManagementLinks && stores && <p className={entitlementStyles.metadata}>店铺统计读取于 <EntitlementTime value={data.observed_at} /></p>}
      </Card>
      {showManagementLinks && <Card role="region" aria-label="成员资源管理" className={panelClass}>
        <h2>成员资源管理</h2>
        <div className={entitlementStyles.overviewActions}>
          <p>管理员可分配店铺访问权限、分配或回收续费期数和数据额度，并设置成员 AI 月度消费上限。连接平台仍需完成平台授权。</p>
          <Button asChild variant="outline">
            <Link href="/workbench/account/organization/resources" prefetch={false}>管理资源与额度</Link>
          </Button>
        </div>
      </Card>}
      {!showManagementLinks && <p className={styles.observation}>
        观察时间：{data.observed_at.replace("T", " ").replace("Z", " UTC")}
      </p>}
    </div>
  );
}
