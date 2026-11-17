"use client";

import {formatMicropoints,parseDisplayPoints} from "@/lib/contracts/ai-points";
import { useQuery } from "@tanstack/react-query";
import Image from "next/image";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { getCommercialResources, type CommercialResources } from "@/lib/api/commercial-billing";
import { EntitlementTime } from "./entitlement-time";
import entitlementStyles from "./entitlements.module.css";
import styles from "./commercial.module.css";

const definitions = [
  { type: "store_renewal_period", name: "店铺续费期数", color: "blue", unit: "期", note: "续费期数不代表已绑定、服务中或到期的店铺数量。" },
  { type: "ai_point", name: "AI 点数", color: "teal", unit: "点", note: "AI 点数按配置费率扣减；模型 Token 单独记录用量，钱包资金用于购买点数。" },
  { type: "data_row", name: "数据额度", color: "purple", unit: "条", note: "按数据条数计量，不以操作次数或存储字节替代。" },
] as const;

export function EnterpriseResources({ userId, organizationId, scope, sequence, showSummary = false, hideActions = false, memberResourcesAnchor, showObservation = true }: { userId: string; organizationId: string; scope: string; sequence: number; showSummary?: boolean; hideActions?: boolean; memberResourcesAnchor?: string; showObservation?: boolean }) {
  const response = useQuery({ queryKey: ["workbench", "commercial-resources", userId, organizationId, scope, sequence], queryFn: ({ signal }) => getCommercialResources(userId, organizationId, signal), gcTime: 0, staleTime: 0, retry: false, refetchOnWindowFocus: false, refetchOnReconnect: false });
  const loading = response.isPending || response.isFetching;
  const data = !loading && !response.isError && response.data?.organization_id === organizationId ? response.data : undefined;
  const code = response.error && "code" in response.error ? String(response.error.code) : "INVALID_UPSTREAM_RESPONSE";
  const errors: Record<string, string> = { PERMISSION_DENIED: "无资源查看权限", AUTHENTICATION_REQUIRED: "登录已失效", ORGANIZATION_ACCESS_REVOKED: "企业访问已撤销", ORGANIZATION_ACCESS_DENIED: "企业访问被拒绝", ORGANIZATION_SUSPENDED: "企业已暂停访问", ORGANIZATION_CONTEXT_CHANGED: "企业上下文已变化", IDENTITY_CONTEXT_CHANGED: "登录身份已变化", DEPENDENCY_UNAVAILABLE: "资源服务暂不可用", DEADLINE_EXCEEDED: "资源读取超时" };
  const fallback = loading ? "正在读取" : "本次未取得";
  const balance = (kind: CommercialResources["resources"][number]["resource_type"]) => data?.resources.find(v => v.resource_type === kind);
  const values = definitions.map(definition => {
    const resource = balance(definition.type);
    const total = resource?.state === "recorded"
      ? resource.resource_type === "ai_point"
        ? formatMicropoints(parseDisplayPoints(resource.available) + parseDisplayPoints(resource.allocated))
        : (BigInt(resource.available) + BigInt(resource.allocated)).toString()
      : undefined;
    return { ...definition, resource, total, value: total !== undefined ? `${total} ${definition.unit}` : resource?.state === "not_recorded" ? "尚无资源记录" : fallback };
  });
  const manage = <Link href="/workbench/account/organization/resources" prefetch={false}>管理成员额度</Link>;
  const wallet = <Link href="/workbench/plans/top-up" prefetch={false}>查看钱包与充值状态</Link>;
  if (showSummary) {
    return (
      <Card role="region" aria-labelledby="entitlement-balances" className={entitlementStyles.ledger}>
        <div className={entitlementStyles.ledgerHeader}>
          <h2 id="entitlement-balances">企业资源余额</h2>
          {data && showObservation && <p className={entitlementStyles.metadata}>余额读取于 <EntitlementTime value={data.observed_at} /></p>}
        </div>
        {loading ? <p role="status">正在读取企业资源余额</p> : !data ? (
          <p role="alert">{errors[code] ?? "暂时无法查看企业资源"}，请刷新重试。</p>
        ) : (
          <ul className={entitlementStyles.resourceList}>
            {values.map(({ name, type, unit, resource, total, note }) => {
              const recorded = resource?.state === "recorded" ? resource : null;
              const updatedAt = recorded?.updated_at;
              return (
                <li key={type}>
                  <section className={entitlementStyles.resourceRow} aria-label={`${name}余额`}>
                    <div>
                      <h3 className={entitlementStyles.resourceName}>{name}</h3>
                      {total !== undefined ? (
                        <>
                          <strong className={entitlementStyles.resourceBalance}>{amountWithUnit(total, unit)}</strong>
                          <p className={entitlementStyles.metadata}>{type === "ai_point" ? "企业可用" : "企业合计可用"}</p>
                        </>
                      ) : (
                        <p className={entitlementStyles.metadata}>尚无资源记录</p>
                      )}
                    </div>
                    {recorded && (
                      <dl className={entitlementStyles.resourceFacts} data-with-member-balances={type !== "ai_point"}>
                        {type !== "ai_point" && (
                          <>
                            <ResourceFact label="未分配（管理员可用）" value={recorded.available} unit={unit} />
                            <ResourceFact label="成员可用" value={recorded.allocated} unit={unit} />
                          </>
                        )}
                        <ResourceFact label="预留" value={recorded.reserved} unit={unit} />
                        <ResourceFact label="已用" value={recorded.consumed} unit={unit} />
                        {recorded.debt !== "0" && <ResourceFact label="待偿还" value={recorded.debt} unit={unit} />}
                      </dl>
                    )}
                    <div className={entitlementStyles.resourceNote}>
                      <p>{type === "ai_point" ? "AI 点数按费率扣减，模型用量另行记录。可用企业钱包购买点数。" : note}</p>
                      {showObservation && updatedAt && updatedAt !== data.observed_at && (
                        <p>更新于 <EntitlementTime value={updatedAt} /></p>
                      )}
                    </div>
                  </section>
                </li>
              );
            })}
          </ul>
        )}
        {!hideActions && <div className={entitlementStyles.ledgerActions}>{wallet}</div>}
      </Card>
    );
  }
  return <div className={styles.stack}>
    {loading ? <p role="status">正在读取当前企业资源余额</p> : !data ? <p role="alert">{errors[code] ?? "资源响应无效"}：本次未取得余额，请确认企业后刷新。</p> : showObservation ? <p className={styles.observation}>资源观察时间：<time dateTime={data.observed_at}>{data.observed_at.replace("T", " ").replace(/Z$/, " UTC")}</time></p> : null}
    <div className={memberResourcesAnchor ? styles.resourceGrid : styles.threeColumns}>{values.map(card => <Card role="region" aria-label={`${card.name}余额`} className={`${styles.resourceCard} ${styles[card.color]}`} key={card.type}>
      <div className={styles.cardHeading}><h3>{card.name}</h3><span>{card.resource?.state === "recorded" ? "已记录" : card.resource?.state === "not_recorded" ? "尚无记录" : fallback}</span></div>
      <p className={styles.resourceValue}>{card.value}</p>
      {card.resource?.state === "recorded" && card.type !== "ai_point" ? <p className={styles.subtle}>未分配（管理员可用）{card.resource.available} {card.unit} · 成员可用 {card.resource.allocated} {card.unit}</p> : null}
      <ul>
        {card.resource?.state === "recorded" ? <><li><Image src={`/console/commercial/bullet-${card.color}.svg`} alt="" width={5} height={5} unoptimized /><span>预留 {card.resource.reserved} {card.unit} · 已消费 {card.resource.consumed} {card.unit}</span></li>{(card.type==="ai_point"?parseDisplayPoints(card.resource.debt):BigInt(card.resource.debt)) > BigInt(0) ? <li><span>待偿还 {card.resource.debt} {card.unit}</span></li> : null}</> : null}
        <li><Image src={`/console/commercial/bullet-${card.color}.svg`} alt="" width={5} height={5} unoptimized /><span>{card.note}</span></li>
      </ul>
      {card.resource?.state === "recorded" ? <p className={styles.subtle}>更新时间：<time dateTime={card.resource.updated_at}>{card.resource.updated_at.replace("T", " ").replace(/Z$/, " UTC")}</time></p> : null}
    </Card>)}</div>
    {memberResourcesAnchor ? (
      <div className={styles.resourceActions}>
        <Button asChild variant="outline"><a href={`#${memberResourcesAnchor}`}>查看成员额度</a></Button>
        <Button asChild variant="outline"><Link href="/workbench/plans/top-up" prefetch={false}>钱包与充值</Link></Button>
      </div>
    ) : !hideActions && <p className={styles.subtle}>{manage} · {wallet}</p>}
  </div>;
}

function amountWithUnit(value: string, unit: string) {
  const [whole, fraction] = value.split(".");
  const grouped = BigInt(whole).toLocaleString("zh-CN");
  return `${grouped}${fraction ? `.${fraction}` : ""} ${unit}`;
}

function ResourceFact({ label, value, unit }: { label: string; value: string; unit: string }) {
  return <div><dt>{label}</dt><dd>{amountWithUnit(value, unit)}</dd></div>;
}
