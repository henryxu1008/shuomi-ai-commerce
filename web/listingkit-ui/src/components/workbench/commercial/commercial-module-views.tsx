"use client";

import { useState, type ReactNode } from "react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import type { CommercialOverview } from "@/lib/api/commercial";
import type { ResourceEventPage, ResourceEventFilters } from "@/lib/api/resource-events";
import { AccountDialog } from "../account/account-dialog";
import { resourceDefinitions } from "./resource-purchase-panel";
import styles from "./commercial.module.css";
import { PlansOverview } from "./plans-overview";

export function CommercialOverviewView({ data }: { data: CommercialOverview }) {
  return <PlansOverview data={data} />;
}

const reasons: Record<string, string> = {
  purchase: "购买到账",
  allocated: "分配",
  reclaimed: "收回",
  reserved: "预留",
  consumed: "使用",
  released: "释放预留",
  grant_commercial_purchase: "购买到账",
  grant_local_ai_trial: "AI 试用赠送",
  allocate_member_resource: "分配给成员",
  reclaim_member_resource: "收回成员额度",
  reserve: "预留",
  commit: "使用",
  release: "释放预留",
  model_point_reserve: "模型调用预留",
  model_point_committed: "模型调用扣除",
  model_point_released: "释放模型调用预留",
};

const balances = [
  { label: "可用", delta: "available_delta", after: "available_after" },
  { label: "已分配", delta: "allocated_delta", after: "allocated_after" },
  { label: "预留", delta: "reserved_delta", after: "reserved_after" },
  { label: "已使用", delta: "consumed_delta", after: "consumed_after" },
] as const;

type UsageEvent = ResourceEventPage["items"][number];

function quantity(value: string, signed = true) {
  const negative = value.startsWith("-");
  const [whole, fraction] = (negative ? value.slice(1) : value).split(".");
  const integer = BigInt(whole);
  if (integer === BigInt(0) && (!fraction || BigInt(fraction) === BigInt(0))) return "0";
  const grouped = integer.toLocaleString("zh-CN");
  return `${negative ? "-" : signed ? "+" : ""}${grouped}${fraction ? `.${fraction}` : ""}`;
}

function resourceFor(event: UsageEvent) {
  return resourceDefinitions.find(({ type }) => type === event.resource_type);
}

function reasonFor(event: UsageEvent) {
  return Object.hasOwn(reasons, event.reason) ? reasons[event.reason] : "其他变动";
}

function UsageEventTime({ timestamp }: { timestamp: string }) {
  return <time dateTime={timestamp}>{timestamp.replace("T", " ").replace("Z", "")}</time>;
}

function UsageEventDetails({ event, onClose }: { event: UsageEvent; onClose: () => void }) {
  const resource = resourceFor(event);
  return (
    <AccountDialog title="用量记录详情" onClose={onClose}>
      <div className={styles.usageDetails}>
        <dl className={styles.orderFacts}>
          <div><dt>时间（UTC）</dt><dd><UsageEventTime timestamp={event.occurred_at} /></dd></div>
          <div><dt>资源</dt><dd>{resource?.name}</dd></div>
          <div><dt>变动说明</dt><dd>{reasonFor(event)}</dd></div>
          <div><dt>关联编号</dt><dd>{event.source_identity}</dd></div>
        </dl>
        <table className={styles.usageDetailTable}>
          <caption className="sr-only">数量与变动后余额</caption>
          <thead><tr><th scope="col">项目</th><th scope="col">本次变动</th><th scope="col">变动后</th></tr></thead>
          <tbody>
            {balances.map(({ label, delta, after }) => (
              <tr key={delta}>
                <th scope="row">{label}</th>
                <td>{quantity(event[delta])} {resource?.unit}</td>
                <td>{quantity(event[after], false)} {resource?.unit}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </AccountDialog>
  );
}

function UsageEventsTable({ items }: { items: UsageEvent[] }) {
  const [selected, setSelected] = useState<UsageEvent>();
  return (
    <>
      <div className={styles.usageTableWrap}>
        <table className={styles.usageTable}>
          <caption className="sr-only">企业用量记录</caption>
          <thead>
            <tr>
              <th scope="col" className={styles.usageWide}>时间（UTC）</th>
              <th scope="col">资源及变动说明</th>
              <th scope="col">本次变动</th>
              <th scope="col" className={styles.usageWide}>变动后可用</th>
              <th scope="col">操作</th>
            </tr>
          </thead>
          <tbody>
            {items.map((event) => {
              const resource = resourceFor(event);
              const changes = balances.map(({ label, delta }) => ({ label, value: quantity(event[delta]) }))
                .filter(({ value }) => value !== "0");
              return (
                <tr key={event.event_id}>
                  <td className={styles.usageWide}><UsageEventTime timestamp={event.occurred_at} /></td>
                  <td>
                    <strong>{resource?.name}</strong>
                    <span className={styles.usageReason}>{reasonFor(event)}</span>
                    <span className={styles.usageCompact}>
                      <UsageEventTime timestamp={event.occurred_at} /> UTC
                    </span>
                  </td>
                  <td>
                    {changes.length ? (
                      <ul className={styles.usageChanges}>
                        {changes.map(({ label, value }) => (
                          <li key={label}>{label} <strong>{value} {resource?.unit}</strong></li>
                        ))}
                      </ul>
                    ) : "无数量变动"}
                  </td>
                  <td className={styles.usageWide}>{quantity(event.available_after, false)} {resource?.unit}</td>
                  <td><Button size="sm" variant="outline" onClick={() => setSelected(event)}>详情</Button></td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
      {selected ? <UsageEventDetails event={selected} onClose={() => setSelected(undefined)} /> : null}
    </>
  );
}

function boundary(value: string, next = false) {
  return new Date(`${value}T00:00:00Z`).getTime() + (next ? 86400000 : 0);
}

export function UsageDetailsView({
  page,
  notice,
  filters = {},
  pageNumber = 1,
  onFilter,
  onPrevious,
  onNext,
}: {
  page?: ResourceEventPage;
  notice?: ReactNode;
  filters?: ResourceEventFilters;
  pageNumber?: number;
  onFilter: (filters: ResourceEventFilters) => void;
  onPrevious?: () => void;
  onNext?: () => void;
}) {
  const [from, setFrom] = useState(filters.from?.slice(0, 10) ?? "");
  const [until, setUntil] = useState(
    filters.until ? new Date(Date.parse(filters.until) - 86400000).toISOString().slice(0, 10) : "",
  );
  const [invalid, setInvalid] = useState(false);
  const categories = [{ name: "全部记录", type: undefined }, ...resourceDefinitions];
  return (
    <Card role="region" aria-label="用量记录" className={`${styles.panel} ${styles.usagePanel}`}>
      <h2>用量记录</h2>
      <p className={styles.observation}>按资源和日期查看记录。点击详情可核对各项变动和关联编号。</p>
      <div role="group" aria-label="资源类型" className={styles.usageCategories}>
        {categories.map(({ type, name }) => (
          <Button
            key={type ?? "all"}
            size="sm"
            variant={filters.resourceType === type ? "secondary" : "ghost"}
            aria-pressed={filters.resourceType === type}
            onClick={() => onFilter({ ...filters, resourceType: type })}
          >
            {name}
          </Button>
        ))}
      </div>
      <form
        className={styles.filterBar}
        aria-label="日期筛选"
        onSubmit={(event) => {
          event.preventDefault();
          if (from && until && boundary(from) >= boundary(until, true)) {
            setInvalid(true);
            return;
          }
          setInvalid(false);
          onFilter({
            resourceType: filters.resourceType,
            from: from ? new Date(boundary(from)).toISOString() : undefined,
            until: until ? new Date(boundary(until, true)).toISOString() : undefined,
          });
        }}
      >
        <label>
          开始日期（UTC）
          <input type="date" value={from} onChange={(event) => setFrom(event.target.value)} />
        </label>
        <label>
          结束日期（UTC）
          <input type="date" value={until} onChange={(event) => setUntil(event.target.value)} />
        </label>
        <div className={styles.actions}>
          <Button type="submit">筛选</Button>
          <Button type="button" variant="outline" onClick={() => {
            setFrom("");
            setUntil("");
            setInvalid(false);
            onFilter({});
          }}>
            重置
          </Button>
        </div>
      </form>
      {invalid ? <p role="alert" className={styles.usageEmpty}>结束日期不能早于开始日期。</p> : null}
      {notice}
      {page ? (
        <>
          {page.items.length ? <UsageEventsTable items={page.items} /> : (
            <p className={styles.usageEmpty}>暂无符合条件的用量记录。</p>
          )}
          <div className={styles.pagination}>
            <span aria-live="polite">第 {pageNumber} 页 · {page.items.length} 条记录</span>
            <div className={styles.actions}>
              <Button size="sm" variant="outline" disabled={!onPrevious} onClick={onPrevious}>上一页</Button>
              <Button size="sm" variant="outline" disabled={!onNext} onClick={onNext}>下一页</Button>
            </div>
          </div>
        </>
      ) : null}
    </Card>
  );
}
